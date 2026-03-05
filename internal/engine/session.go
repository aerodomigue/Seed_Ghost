package engine

import (
	"context"
	"fmt"
	"log"
	mrand "math/rand"
	"net"
	"sync"
	"time"

	"github.com/anthony/seed_ghost/internal/announce"
	"github.com/anthony/seed_ghost/internal/client"
	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/torrent"
)

const blockSize64 = int64(blockSize) // 16KB

// Session manages the announce loop for a single torrent.
type Session struct {
	mu sync.Mutex

	TorrentID int64
	Torrent   *torrent.Torrent
	Profile   *client.Profile

	// State
	PeerID     string
	Key        string
	Port       int
	Uploaded   int64
	Downloaded int64

	// Tracking
	lastInterval   int
	minInterval    int
	lastLeechers   int
	lastSeeders    int
	allocatedSpeed float64 // speed assigned by bandwidth dispatcher (bytes/s)
	currentSpeed   float64 // actual speed with jitter (bytes/s)
	accumulator    float64 // fractional bytes not yet added to Uploaded

	// Download simulation
	downloadSpeed        float64 // assigned download speed (bytes/s)
	currentDownloadSpeed float64 // actual download speed with jitter (bytes/s)
	downloadAccumulator  float64 // fractional bytes not yet added to Downloaded
	downloadComplete     bool    // true when Downloaded >= TotalSize
	downloadedAtLastAnnounce int64

	// Seed time countdown (nil = manual torrent, skip decrement)
	seedTimeRemainingMs *int64

	// For announce logging
	uploadedAtLastAnnounce int64
	announced              bool // true after first successful announce

	// TCP listener for connectable check
	listener net.Listener

	// Signals announceLoop to send "completed" immediately
	downloadDone chan struct{}

	cancel context.CancelFunc
	wg     sync.WaitGroup
	db     *database.DB
}

// NewSession creates a new session for a torrent.
func NewSession(torrentID int64, tor *torrent.Torrent, profile *client.Profile, db *database.DB) *Session {
	return &Session{
		TorrentID:    torrentID,
		Torrent:      tor,
		Profile:      profile,
		lastInterval: 1800, // Default 30 min
		db:           db,
		downloadDone: make(chan struct{}, 1),
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
	s.uploadedAtLastAnnounce = state.Uploaded
	s.Downloaded = state.Downloaded
	s.downloadedAtLastAnnounce = state.Downloaded
	if s.Downloaded >= s.Torrent.TotalSize {
		s.downloadComplete = true
	}
	s.lastInterval = state.LastInterval
	s.lastLeechers = state.LastLeechers
	s.lastSeeders = state.LastSeeders
	s.announced = true
}

// Start begins the announce loop and upload simulation in goroutines.
func (s *Session) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	// Initialize state if new session
	s.mu.Lock()
	if s.PeerID == "" {
		s.PeerID = s.Profile.GeneratePeerID()
		s.Key = s.Profile.GenerateKey()
		s.Port = s.Profile.RandomPort()
	}
	s.uploadedAtLastAnnounce = s.Uploaded
	s.downloadedAtLastAnnounce = s.Downloaded
	s.mu.Unlock()

	// Start TCP listener for connectable checks
	s.startListener()

	s.wg.Add(2)
	go s.simulateTraffic(ctx)
	go s.announceLoop(ctx)
}

// Stop cancels the announce loop, waits for cleanup, and closes the listener.
func (s *Session) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
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
		Downloaded:   s.Downloaded,
		LastAnnounce: &now,
		LastInterval: s.lastInterval,
		LastLeechers: s.lastLeechers,
		LastSeeders:  s.lastSeeders,
		LastDelta:    0,
	}
}

// GetSpeed returns the current simulated upload speed in bytes/s.
func (s *Session) GetSpeed() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentSpeed
}

// GetDownloadSpeed returns the current simulated download speed in bytes/s.
func (s *Session) GetDownloadSpeed() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentDownloadSpeed
}

// SetDownloadSpeed sets the assigned download speed in bytes/s.
func (s *Session) SetDownloadSpeed(bytesPerSec float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.downloadSpeed = bytesPerSec
}

// IsDownloadComplete returns whether the download simulation has finished.
func (s *Session) IsDownloadComplete() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.downloadComplete
}

// GetDownloaded returns the current simulated downloaded bytes.
func (s *Session) GetDownloaded() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Downloaded
}

// HasAnnounced returns true if at least one successful announce has been made.
func (s *Session) HasAnnounced() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.announced
}

// SetAllocatedSpeed sets the speed assigned by the bandwidth dispatcher.
func (s *Session) SetAllocatedSpeed(bytesPerSec float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allocatedSpeed = bytesPerSec
}

// GetLeechers returns the current leecher count for this session.
func (s *Session) GetLeechers() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastLeechers
}

// SetSeedTimeRemaining sets the seed time countdown (nil = manual torrent).
func (s *Session) SetSeedTimeRemaining(ms *int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedTimeRemainingMs = ms
}

// GetSeedTimeRemainingMs returns the current seed time remaining in ms (nil = manual).
func (s *Session) GetSeedTimeRemainingMs() *int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seedTimeRemainingMs == nil {
		return nil
	}
	v := *s.seedTimeRemainingMs
	return &v
}

// GetSeeders returns the current seeder count for this session.
func (s *Session) GetSeeders() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSeeders
}

// simulateTraffic increments Uploaded and Downloaded every second based on allocated speeds.
func (s *Session) simulateTraffic(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			// Decrement seed time countdown
			if s.seedTimeRemainingMs != nil {
				*s.seedTimeRemainingMs -= 1000
			}

			// Upload simulation
			if s.allocatedSpeed > 0 {
				jitter := 1.0 + (mrand.Float64()*0.1 - 0.05)
				speed := s.allocatedSpeed * jitter
				s.currentSpeed = speed

				s.accumulator += speed
				blocks := int64(s.accumulator) / blockSize64
				if blocks > 0 {
					s.Uploaded += blocks * blockSize64
					s.accumulator -= float64(blocks * blockSize64)
				}
			} else {
				s.currentSpeed = 0
				s.accumulator = 0
			}

			// Download simulation
			if !s.downloadComplete && s.downloadSpeed > 0 {
				jitter := 1.0 + (mrand.Float64()*0.1 - 0.05)
				dlSpeed := s.downloadSpeed * jitter
				s.currentDownloadSpeed = dlSpeed

				s.downloadAccumulator += dlSpeed
				blocks := int64(s.downloadAccumulator) / blockSize64
				if blocks > 0 {
					s.Downloaded += blocks * blockSize64
					s.downloadAccumulator -= float64(blocks * blockSize64)
				}

				if s.Downloaded >= s.Torrent.TotalSize {
					s.Downloaded = s.Torrent.TotalSize
					s.downloadComplete = true
					s.currentDownloadSpeed = 0
					// Signal announceLoop to send "completed" immediately
					select {
					case s.downloadDone <- struct{}{}:
					default:
					}
				}
			} else if s.downloadComplete {
				s.currentDownloadSpeed = 0
			}

			s.mu.Unlock()
		}
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
	defer s.wg.Done()
	// Always send "started" on first announce — even restored sessions need to
	// re-register with the tracker (shutdown sends "stopped" which removes the peer).
	s.doAnnounce("started")
	s.saveState()

	sentCompleted := false

	for {
		s.mu.Lock()
		interval := s.lastInterval
		dlComplete := s.downloadComplete
		s.mu.Unlock()

		// Apply jitter to interval, respect min_interval
		waitSecs := ApplyJitter(interval)
		if s.minInterval > 0 && waitSecs < s.minInterval {
			waitSecs = s.minInterval
		}
		timer := time.NewTimer(time.Duration(waitSecs) * time.Second)

		select {
		case <-ctx.Done():
			timer.Stop()
			// Save state BEFORE stopped announce (preserves leechers/seeders)
			s.saveState()
			// Send stopped event
			s.doAnnounce("stopped")
			return
		case <-s.downloadDone:
			timer.Stop()
			s.doAnnounce("completed")
			sentCompleted = true
			s.saveState()
		case <-timer.C:
			// Send "completed" event once when download finishes (fallback if signal missed)
			if dlComplete && !sentCompleted {
				s.doAnnounce("completed")
				sentCompleted = true
			} else {
				s.doAnnounce("")
			}
			s.saveState()
		}
	}
}

func (s *Session) doAnnounce(event string) {
	if len(s.Torrent.Trackers) == 0 {
		return
	}

	s.mu.Lock()
	// Delta is what accumulated since last announce (from simulateTraffic)
	delta := s.Uploaded - s.uploadedAtLastAnnounce
	s.uploadedAtLastAnnounce = s.Uploaded
	s.downloadedAtLastAnnounce = s.Downloaded

	left := s.Torrent.TotalSize - s.Downloaded
	if left < 0 {
		left = 0
	}
	downloaded := s.Downloaded
	if s.downloadComplete {
		downloaded = s.Torrent.TotalSize
		left = 0
	}

	baseParams := announce.Params{
		InfoHash:   s.Torrent.InfoHash,
		PeerID:     s.PeerID,
		Port:       s.Port,
		Uploaded:   s.Uploaded,
		Downloaded: downloaded,
		Left:       left,
		Event:      event,
		Key:        s.Key,
		Compact:    s.Profile.SupportsCompact,
		Numwant:    s.Profile.NumwantDefault,
	}
	trackers := s.Torrent.Trackers
	profile := s.Profile
	s.mu.Unlock()

	type trackerResult struct {
		trackerURL string
		resp       *announce.Response
		err        error
	}

	// Announce to all trackers in parallel
	results := make(chan trackerResult, len(trackers))
	var announceWg sync.WaitGroup
	for _, trackerURL := range trackers {
		announceWg.Add(1)
		go func(url string) {
			defer announceWg.Done()
			params := baseParams
			params.TrackerURL = url
			resp, err := announce.Do(&params, profile)
			results <- trackerResult{trackerURL: url, resp: resp, err: err}
		}(trackerURL)
	}

	// Close results channel when all goroutines finish
	go func() {
		announceWg.Wait()
		close(results)
	}()

	// Process results — use best successful response for leechers/seeders
	bestLeechers := -1
	for res := range results {
		s.mu.Lock()
		status := "success"
		errMsg := ""

		if res.err != nil {
			log.Printf("[session %d] announce error (%s): %v", s.TorrentID, res.trackerURL, res.err)
			status = "error"
			errMsg = res.err.Error()
		} else if res.resp.FailureMsg != "" {
			log.Printf("[session %d] tracker failure (%s): %s", s.TorrentID, res.trackerURL, res.resp.FailureMsg)
			status = "error"
			errMsg = res.resp.FailureMsg
		} else {
			if res.resp.Interval > 0 {
				s.lastInterval = res.resp.Interval
			}
			if res.resp.MinInterval > 0 {
				s.minInterval = res.resp.MinInterval
			}
			// Keep the highest leecher count across all trackers
			if res.resp.HasLeechers && res.resp.Leechers > bestLeechers {
				if event != "started" || res.resp.Leechers > 0 {
					s.lastLeechers = res.resp.Leechers
					bestLeechers = res.resp.Leechers
				}
			}
			if event != "started" || res.resp.Seeders > 0 {
				if res.resp.HasSeeders {
					s.lastSeeders = res.resp.Seeders
				}
			}

			s.announced = true
			log.Printf("[session %d] announce ok (%s): leechers=%d seeders=%d interval=%d uploaded=%d delta=%d speed=%.0f B/s",
				s.TorrentID, res.trackerURL, s.lastLeechers, s.lastSeeders, res.resp.Interval, s.Uploaded, delta, s.currentSpeed)
		}

		// Log to database
		if s.db != nil {
			s.db.InsertAnnounceLog(s.TorrentID, res.trackerURL, event, delta,
				s.lastLeechers, s.lastSeeders, s.lastInterval, status, errMsg)

			if status == "success" {
				s.db.InsertStatsLog(s.TorrentID, s.Uploaded, s.lastLeechers, s.lastSeeders)
			}
		}
		s.mu.Unlock()
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
	// Persist seed time countdown
	s.mu.Lock()
	remaining := s.seedTimeRemainingMs
	s.mu.Unlock()
	if remaining != nil {
		if err := s.db.UpdateSeedTimeRemaining(s.TorrentID, *remaining); err != nil {
			log.Printf("[session %d] save seed time error: %v", s.TorrentID, err)
		}
	}
}
