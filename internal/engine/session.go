package engine

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/anthony/seed_ghost/internal/announce"
	"github.com/anthony/seed_ghost/internal/client"
	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/torrent"
)

// Session manages the announce loop for a single torrent.
type Session struct {
	mu sync.Mutex

	TorrentID int64
	Torrent   *torrent.Torrent
	Profile   *client.Profile
	Config    RatioConfig

	// State
	PeerID   string
	Key      string
	Port     int
	Uploaded int64

	// Tracking
	lastDelta    int64
	lastInterval int
	lastLeechers int
	lastSeeders  int

	// TCP listener for connectable check
	listener net.Listener

	cancel context.CancelFunc
	db     *database.DB
}

// NewSession creates a new session for a torrent.
func NewSession(torrentID int64, tor *torrent.Torrent, profile *client.Profile, cfg RatioConfig, db *database.DB) *Session {
	return &Session{
		TorrentID:    torrentID,
		Torrent:      tor,
		Profile:      profile,
		Config:       cfg,
		lastInterval: 1800, // Default 30 min
		db:           db,
	}
}

// RestoreState restores session state from the database.
func (s *Session) RestoreState(state *database.AnnounceStateRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PeerID = state.PeerID
	s.Key = state.Key
	s.Port = state.Port
	s.Uploaded = state.Uploaded
	s.lastDelta = state.LastDelta
	s.lastInterval = state.LastInterval
	s.lastLeechers = state.LastLeechers
	s.lastSeeders = state.LastSeeders
}

// Start begins the announce loop in a goroutine.
func (s *Session) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	// Initialize state if new session
	s.mu.Lock()
	if s.PeerID == "" {
		s.PeerID = s.Profile.GeneratePeerID()
		s.Key = s.Profile.GenerateKey()
		s.Port = s.Profile.RandomPort()
	}
	s.mu.Unlock()

	// Start TCP listener for connectable checks
	s.startListener()

	go s.announceLoop(ctx)
}

// Stop cancels the announce loop and sends a stopped event.
func (s *Session) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Close()
	}
}

// GetState returns the current session state.
func (s *Session) GetState() *database.AnnounceStateRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	return &database.AnnounceStateRow{
		TorrentID:    s.TorrentID,
		PeerID:       s.PeerID,
		Key:          s.Key,
		Port:         s.Port,
		Uploaded:     s.Uploaded,
		LastAnnounce: &now,
		LastInterval: s.lastInterval,
		LastLeechers: s.lastLeechers,
		LastSeeders:  s.lastSeeders,
		LastDelta:    s.lastDelta,
	}
}

func (s *Session) startListener() {
	s.mu.Lock()
	port := s.Port
	infoHash := s.Torrent.InfoHash
	s.mu.Unlock()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("[session %d] failed to listen on port %d: %v", s.TorrentID, port, err)
		return
	}
	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go handleBTHandshake(conn, infoHash)
		}
	}()
}

// handleBTHandshake responds to the initial BitTorrent handshake then closes.
func handleBTHandshake(conn net.Conn, infoHash [20]byte) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read handshake: 1 byte pstrlen + pstr + 8 reserved + 20 info_hash + 20 peer_id
	buf := make([]byte, 68)
	n, err := conn.Read(buf)
	if err != nil || n < 68 {
		return
	}

	// Respond with our handshake
	response := make([]byte, 68)
	response[0] = 19 // pstrlen
	copy(response[1:20], []byte("BitTorrent protocol"))
	// 8 bytes reserved (zeros)
	copy(response[28:48], infoHash[:])
	// 20 bytes fake peer_id
	copy(response[48:68], []byte("-SG0001-000000000000"))
	conn.Write(response)
}

func (s *Session) announceLoop(ctx context.Context) {
	// Send started event on first announce
	s.doAnnounce("started")
	s.saveState()

	for {
		s.mu.Lock()
		interval := s.lastInterval
		s.mu.Unlock()

		// Apply jitter to interval
		waitSecs := ApplyJitter(interval)
		timer := time.NewTimer(time.Duration(waitSecs) * time.Second)

		select {
		case <-ctx.Done():
			timer.Stop()
			// Send stopped event
			s.doAnnounce("stopped")
			s.saveState()
			return
		case <-timer.C:
			s.doAnnounce("")
			s.saveState()
		}
	}
}

func (s *Session) doAnnounce(event string) {
	if len(s.Torrent.Trackers) == 0 {
		return
	}

	s.mu.Lock()
	trackerURL := s.Torrent.Trackers[0] // Use primary tracker
	leechers := s.lastLeechers

	// Calculate upload delta
	var delta int64
	if event != "started" && event != "stopped" {
		delta = CalculateUploadDelta(s.Config, leechers, s.lastInterval, s.lastDelta)
		s.Uploaded += delta
	}

	params := &announce.Params{
		TrackerURL: trackerURL,
		InfoHash:   s.Torrent.InfoHash,
		PeerID:     s.PeerID,
		Port:       s.Port,
		Uploaded:   s.Uploaded,
		Downloaded: 0,
		Left:       0, // Always seeding (left=0)
		Event:      event,
		Key:        s.Key,
		Compact:    s.Profile.SupportsCompact,
		Numwant:    s.Profile.NumwantDefault,
	}
	s.mu.Unlock()

	resp, err := announce.Do(params, s.Profile)

	s.mu.Lock()
	defer s.mu.Unlock()

	status := "success"
	errMsg := ""

	if err != nil {
		log.Printf("[session %d] announce error: %v", s.TorrentID, err)
		status = "error"
		errMsg = err.Error()
	} else if resp.FailureMsg != "" {
		log.Printf("[session %d] tracker failure: %s", s.TorrentID, resp.FailureMsg)
		status = "error"
		errMsg = resp.FailureMsg
	} else {
		if resp.Interval > 0 {
			s.lastInterval = resp.Interval
		}
		s.lastLeechers = resp.Leechers
		s.lastSeeders = resp.Seeders
		s.lastDelta = delta

		log.Printf("[session %d] announce ok: leechers=%d seeders=%d interval=%d uploaded=%d delta=%d",
			s.TorrentID, resp.Leechers, resp.Seeders, resp.Interval, s.Uploaded, delta)
	}

	// Log to database
	if s.db != nil {
		s.db.InsertAnnounceLog(s.TorrentID, trackerURL, event, delta,
			s.lastLeechers, s.lastSeeders, s.lastInterval, status, errMsg)

		if status == "success" {
			s.db.InsertStatsLog(s.TorrentID, s.Uploaded, s.lastLeechers, s.lastSeeders)
		}
	}
}

func (s *Session) saveState() {
	if s.db == nil {
		return
	}
	state := s.GetState()
	if err := s.db.UpsertAnnounceState(state); err != nil {
		log.Printf("[session %d] save state error: %v", s.TorrentID, err)
	}
}
