package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/anthony/seed_ghost/internal/client"
	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/torrent"
)

// Manager manages all active seeding sessions.
type Manager struct {
	mu       sync.Mutex
	sessions map[int64]*Session
	db       *database.DB
	profiles *client.ProfileStore
	config   RatioConfig
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewManager creates a new session manager.
func NewManager(db *database.DB, profiles *client.ProfileStore, cfg RatioConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		sessions: make(map[int64]*Session),
		db:       db,
		profiles: profiles,
		config:   cfg,
		ctx:      ctx,
		cancel:   cancel,
	}
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

	return m.startSessionFromRow(row)
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

	return nil
}

// AddTorrent adds a new torrent from raw .torrent data and optionally starts it.
func (m *Manager) AddTorrent(torrentData []byte, profileName string, autoStart bool) (int64, error) {
	tor, err := torrent.Parse(torrentData)
	if err != nil {
		return 0, fmt.Errorf("parse torrent: %w", err)
	}

	trackerURL := ""
	if len(tor.Trackers) > 0 {
		trackerURL = tor.Trackers[0]
	}

	row := &database.TorrentRow{
		InfoHash:      tor.InfoHashHex(),
		Name:          tor.Name,
		TotalSize:     tor.TotalSize,
		TrackerURL:    trackerURL,
		TorrentData:   torrentData,
		ClientProfile: profileName,
		Active:        autoStart,
		Source:        "manual",
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

// UpdateConfig updates the ratio config for all sessions.
func (m *Manager) UpdateConfig(cfg RatioConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg
	for _, s := range m.sessions {
		s.mu.Lock()
		s.Config = cfg
		s.mu.Unlock()
	}
}

// Shutdown stops all sessions gracefully.
func (m *Manager) Shutdown() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, session := range m.sessions {
		session.Stop()
		delete(m.sessions, id)
	}
}

func (m *Manager) startSessionFromRow(row *database.TorrentRow) error {
	tor, err := torrent.Parse(row.TorrentData)
	if err != nil {
		return fmt.Errorf("parse torrent data: %w", err)
	}

	profileName := row.ClientProfile
	if profileName == "" {
		names := m.profiles.List()
		if len(names) > 0 {
			profileName = names[0]
		}
	}

	profile, ok := m.profiles.Get(profileName)
	if !ok {
		// Fallback: use first available profile
		all := m.profiles.All()
		for _, p := range all {
			profile = p
			break
		}
		if profile == nil {
			return fmt.Errorf("no client profiles available")
		}
	}

	session := NewSession(row.ID, tor, profile, m.config, m.db)

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
