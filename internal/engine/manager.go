package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthony/seed_ghost/internal/client"
	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/torrent"
)

// SeedTimeRemainingMs returns the live seed time remaining for a torrent (from session or DB).
func (m *Manager) SeedTimeRemainingMs(id int64) *int64 {
	m.mu.Lock()
	session, ok := m.sessions[id]
	m.mu.Unlock()
	if ok {
		return session.GetSeedTimeRemainingMs()
	}
	row, err := m.db.GetTorrent(id)
	if err != nil {
		return nil
	}
	return row.SeedTimeRemainingMs
}

// Manager manages all active seeding sessions and dispatches bandwidth.
type Manager struct {
	mu             sync.Mutex
	sessions       map[int64]*Session
	db             *database.DB
	profiles       *client.ProfileStore
	defaultProfile string
	config         RatioConfig
	globalSpeed    float64 // current global speed in bytes/s
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewManager creates a new session manager.
func NewManager(db *database.DB, profiles *client.ProfileStore, cfg RatioConfig, defaultProfile string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		sessions:       make(map[int64]*Session),
		db:             db,
		profiles:       profiles,
		defaultProfile: defaultProfile,
		config:         cfg,
		ctx:            ctx,
		cancel:         cancel,
	}
	// Pick initial global speed and start dispatcher
	m.refreshGlobalSpeed()
	go m.bandwidthDispatcher(ctx)
	return m
}

// RestoreActiveSessions loads all active torrents from the database and starts their sessions.
func (m *Manager) RestoreActiveSessions() error {
	torrents, err := m.db.GetActiveTorrents()
	if err != nil {
		return fmt.Errorf("get active torrents: %w", err)
	}

	for _, row := range torrents {
		if err := m.startSessionFromRow(row); err != nil {
			log.Printf("[manager] failed to restore session for torrent %d: %v", row.ID, err)
		}
	}

	// Redistribute bandwidth with restored sessions
	m.distributeBandwidth()

	log.Printf("[manager] restored %d active sessions", len(m.sessions))
	return nil
}

// StartTorrent starts seeding a torrent by ID.
func (m *Manager) StartTorrent(id int64) error {
	m.mu.Lock()
	if _, exists := m.sessions[id]; exists {
		m.mu.Unlock()
		return fmt.Errorf("torrent %d already active", id)
	}
	m.mu.Unlock()

	row, err := m.db.GetTorrent(id)
	if err != nil {
		return fmt.Errorf("get torrent: %w", err)
	}

	if err := m.db.SetTorrentActive(id, true); err != nil {
		return fmt.Errorf("set active: %w", err)
	}

	if err := m.startSessionFromRow(row); err != nil {
		return err
	}

	// Redistribute bandwidth with new session
	m.distributeBandwidth()
	return nil
}

// StopTorrent stops seeding a torrent by ID.
func (m *Manager) StopTorrent(id int64) error {
	m.mu.Lock()
	session, exists := m.sessions[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("torrent %d not active", id)
	}
	delete(m.sessions, id)
	m.mu.Unlock()

	session.Stop()

	if err := m.db.SetTorrentActive(id, false); err != nil {
		return fmt.Errorf("set inactive: %w", err)
	}

	// Redistribute bandwidth among remaining sessions
	m.distributeBandwidth()
	return nil
}

// AddTorrent adds a new torrent from raw .torrent data and optionally starts it.
// indexerID is non-nil when added via Prowlarr, nil for manual adds.
func (m *Manager) AddTorrent(torrentData []byte, profileName string, autoStart bool, indexerID *int64, seedTimeMs *int64) (int64, error) {
	tor, err := torrent.Parse(torrentData)
	if err != nil {
		return 0, fmt.Errorf("parse torrent: %w", err)
	}

	trackerURL := ""
	if len(tor.Trackers) > 0 {
		trackerURL = tor.Trackers[0]
	}

	source := "manual"
	if indexerID != nil {
		source = "prowlarr"
	}

	row := &database.TorrentRow{
		InfoHash:      tor.InfoHashHex(),
		Name:          tor.Name,
		TotalSize:     tor.TotalSize,
		TrackerURL:    trackerURL,
		TorrentData:   torrentData,
		ClientProfile: profileName,
		Active:        autoStart,
		Source:        source,
		IndexerID:     indexerID,
	}
	if indexerID != nil {
		if seedTimeMs != nil {
			row.SeedTimeRemainingMs = seedTimeMs
		} else {
			seedTime := int64(259200000) // 72h in ms default
			row.SeedTimeRemainingMs = &seedTime
		}
	} else {
		seedTime := int64(0)
		row.SeedTimeRemainingMs = &seedTime
	}

	id, err := m.db.InsertTorrent(row)
	if err != nil {
		return 0, fmt.Errorf("insert torrent: %w", err)
	}

	if autoStart {
		row.ID = id
		if err := m.startSessionFromRow(row); err != nil {
			return id, fmt.Errorf("start session: %w", err)
		}
		m.distributeBandwidth()
	}

	return id, nil
}

// RemoveTorrent stops and removes a torrent.
func (m *Manager) RemoveTorrent(id int64) error {
	m.mu.Lock()
	if session, exists := m.sessions[id]; exists {
		session.Stop()
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	m.distributeBandwidth()
	return m.db.DeleteTorrent(id)
}

// GetSessions returns info about all active sessions.
func (m *Manager) GetSessions() map[int64]*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[int64]*Session, len(m.sessions))
	for k, v := range m.sessions {
		result[k] = v
	}
	return result
}

// UpdateConfig updates the ratio config and redistributes bandwidth.
func (m *Manager) UpdateConfig(cfg RatioConfig) {
	m.mu.Lock()
	m.config = cfg
	m.mu.Unlock()
	m.refreshGlobalSpeed()
	m.distributeBandwidth()
}

// Shutdown stops all sessions gracefully, waiting for each to save state.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	sessions := make(map[int64]*Session, len(m.sessions))
	for id, s := range m.sessions {
		sessions[id] = s
	}
	m.mu.Unlock()

	for id, session := range sessions {
		session.Stop()
		log.Printf("[manager] session %d stopped and state saved", id)
	}

	m.mu.Lock()
	m.sessions = make(map[int64]*Session)
	m.mu.Unlock()
	m.cancel()
}

// refreshGlobalSpeed picks a new random global speed.
func (m *Manager) refreshGlobalSpeed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.globalSpeed = RandomGlobalSpeed(m.config)
	log.Printf("[bandwidth] new global speed: %.0f KB/s", m.globalSpeed/1024)
}

// distributeBandwidth distributes the global speed across sessions by weight.
func (m *Manager) distributeBandwidth() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) == 0 {
		return
	}

	// Calculate weights for each session
	type sessionWeight struct {
		session *Session
		weight  float64
	}

	var entries []sessionWeight
	var totalWeight float64

	for _, s := range m.sessions {
		leechers := s.GetLeechers()
		seeders := s.GetSeeders()
		w := CalculateWeight(leechers, seeders)
		entries = append(entries, sessionWeight{session: s, weight: w})
		totalWeight += w
	}

	// Distribute speed proportionally to weight
	for _, e := range entries {
		var speed float64
		if totalWeight > 0 && e.weight > 0 {
			speed = m.globalSpeed * (e.weight / totalWeight)
		}
		e.session.SetAllocatedSpeed(speed)
	}
}

// bandwidthDispatcher refreshes global speed every 20 minutes and redistributes.
func (m *Manager) bandwidthDispatcher(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()

	// Also redistribute every 30 seconds to react to leecher count changes
	redistTicker := time.NewTicker(30 * time.Second)
	defer redistTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.refreshGlobalSpeed()
			m.distributeBandwidth()
		case <-redistTicker.C:
			m.distributeBandwidth()
		}
	}
}

func (m *Manager) startSessionFromRow(row *database.TorrentRow) error {
	tor, err := torrent.Parse(row.TorrentData)
	if err != nil {
		return fmt.Errorf("parse torrent data: %w", err)
	}

	profileName := row.ClientProfile
	if profileName == "" {
		profileName = m.defaultProfile
	}
	profile, ok := m.profiles.Get(profileName)
	if !ok {
		return fmt.Errorf("client profile %q not found", profileName)
	}

	session := NewSession(row.ID, tor, profile, m.db)
	session.SetSeedTimeRemaining(row.SeedTimeRemainingMs)

	// Try to restore state from database
	state, err := m.db.GetAnnounceState(row.ID)
	if err == nil {
		session.RestoreState(state)
	}

	m.mu.Lock()
	m.sessions[row.ID] = session
	m.mu.Unlock()

	session.Start(m.ctx)
	return nil
}
