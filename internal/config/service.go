package config

import (
	"strconv"
	"sync"
)

// Store is the interface for persisting settings (implemented by database.DB).
type Store interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}

// Service is the single source of truth for configuration.
// It loads defaults, overrides from DB, and persists changes back.
type Service struct {
	mu    sync.RWMutex
	cfg   Config
	store Store
}

// NewService creates a config service. It starts with the given defaults,
// then overrides with any values saved in the store.
func NewService(defaults *Config, store Store) *Service {
	s := &Service{
		cfg:   *defaults, // copy
		store: store,
	}
	s.loadFromStore()
	return s
}

// Get returns a snapshot of the current config.
func (s *Service) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// GetDefaultClient returns the default client profile name.
func (s *Service) GetDefaultClient() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.DefaultClient
}

// GetAutoStart returns whether torrents should auto-start when added.
func (s *Service) GetAutoStart() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.AutoStart
}

// GetMinUploadSpeedKBs returns the minimum upload speed in KB/s.
func (s *Service) GetMinUploadSpeedKBs() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.MinUploadSpeedKBs
}

// GetMaxUploadSpeedKBs returns the maximum upload speed in KB/s.
func (s *Service) GetMaxUploadSpeedKBs() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.MaxUploadSpeedKBs
}

// GetFetchInterval returns the Prowlarr fetch interval in minutes.
func (s *Service) GetFetchInterval() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.FetchInterval
}

// GetProwlarrURL returns the Prowlarr URL.
func (s *Service) GetProwlarrURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.ProwlarrURL
}

// GetProwlarrAPIKey returns the Prowlarr API key.
func (s *Service) GetProwlarrAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.ProwlarrAPIKey
}

// Update applies changes to the config and persists them to the store.
func (s *Service) Update(fn func(cfg *Config)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.cfg)
	s.saveToStore()
}

// DB settings key mapping
var settingsKeys = []struct {
	key   string
	load  func(val string, cfg *Config)
	save  func(cfg *Config) string
}{
	{
		key:  "default_client",
		load: func(v string, c *Config) { if v != "" { c.DefaultClient = v } },
		save: func(c *Config) string { return c.DefaultClient },
	},
	{
		key:  "auto_start",
		load: func(v string, c *Config) { c.AutoStart = v == "true" },
		save: func(c *Config) string { return strconv.FormatBool(c.AutoStart) },
	},
	{
		key: "min_upload_speed_kbs",
		load: func(v string, c *Config) {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				c.MinUploadSpeedKBs = f
			}
		},
		save: func(c *Config) string { return strconv.FormatFloat(c.MinUploadSpeedKBs, 'f', -1, 64) },
	},
	{
		key: "max_upload_speed_kbs",
		load: func(v string, c *Config) {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				c.MaxUploadSpeedKBs = f
			}
		},
		save: func(c *Config) string { return strconv.FormatFloat(c.MaxUploadSpeedKBs, 'f', -1, 64) },
	},
	{
		key: "log_retention_days",
		load: func(v string, c *Config) {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				c.LogRetentionDays = n
			}
		},
		save: func(c *Config) string { return strconv.Itoa(c.LogRetentionDays) },
	},
	{
		key:  "prowlarr_url",
		load: func(v string, c *Config) { c.ProwlarrURL = v },
		save: func(c *Config) string { return c.ProwlarrURL },
	},
	{
		key:  "prowlarr_api_key",
		load: func(v string, c *Config) { c.ProwlarrAPIKey = v },
		save: func(c *Config) string { return c.ProwlarrAPIKey },
	},
	{
		key: "prowlarr_fetch_interval",
		load: func(v string, c *Config) {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				c.FetchInterval = n
			}
		},
		save: func(c *Config) string { return strconv.Itoa(c.FetchInterval) },
	},
	{
		key: "prowlarr_max_slots",
		load: func(v string, c *Config) {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				c.ProwlarrMaxSlots = n
			}
		},
		save: func(c *Config) string { return strconv.Itoa(c.ProwlarrMaxSlots) },
	},
}

func (s *Service) loadFromStore() {
	for _, sk := range settingsKeys {
		if v, err := s.store.GetSetting(sk.key); err == nil {
			sk.load(v, &s.cfg)
		}
	}
}

func (s *Service) saveToStore() {
	for _, sk := range settingsKeys {
		s.store.SetSetting(sk.key, sk.save(&s.cfg))
	}
}
