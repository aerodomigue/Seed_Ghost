package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	// Server
	ListenAddr string `json:"listenAddr"`

	// Database
	DatabasePath string `json:"databasePath"`

	// Profiles
	ProfilesDir string `json:"profilesDir"`

	// Default client profile name
	DefaultClient string `json:"defaultClient"`

	// Auto-start torrents when added
	AutoStart bool `json:"autoStart"`

	// Upload speed limits (KB/s)
	MinUploadSpeedKBs float64 `json:"minUploadSpeedKBs"`
	MaxUploadSpeedKBs float64 `json:"maxUploadSpeedKBs"`

	// Download speed limits (KB/s) — for fake download simulation
	MinDownloadSpeedKBs float64 `json:"minDownloadSpeedKBs"`
	MaxDownloadSpeedKBs float64 `json:"maxDownloadSpeedKBs"`

	// Prowlarr
	ProwlarrURL    string `json:"prowlarrUrl"`
	ProwlarrAPIKey string `json:"prowlarrApiKey"`
	FetchInterval    int `json:"fetchIntervalMinutes"`    // minutes between auto-fetch
	ProwlarrMaxSlots int `json:"prowlarrMaxSlots"`        // max concurrent active torrents

	// Logging
	LogRetentionDays int `json:"logRetentionDays"`

	// Data directory (for storing .torrent files, etc.)
	DataDir string `json:"dataDir"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:        ":8333",
		DatabasePath:      "data/seedghost.db",
		ProfilesDir:       "profiles",
		DefaultClient:     "qBittorrent 5.1.4",
		AutoStart:         true,
		MinUploadSpeedKBs:   50,
		MaxUploadSpeedKBs:   5000,
		MinDownloadSpeedKBs: 100,
		MaxDownloadSpeedKBs: 10000,
		FetchInterval:     1440,
		ProwlarrMaxSlots:  5,
		LogRetentionDays:  7,
		DataDir:           "data",
	}
}

// Load reads configuration from a JSON file, with environment variable overrides.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return cfg, nil
			}
			return nil, err
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Environment variable overrides
	if v := os.Getenv("SEEDGHOST_LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("SEEDGHOST_DB_PATH"); v != "" {
		cfg.DatabasePath = v
	}
	if v := os.Getenv("SEEDGHOST_PROFILES_DIR"); v != "" {
		cfg.ProfilesDir = v
	}
	if v := os.Getenv("SEEDGHOST_PROWLARR_URL"); v != "" {
		cfg.ProwlarrURL = v
	}
	if v := os.Getenv("SEEDGHOST_PROWLARR_API_KEY"); v != "" {
		cfg.ProwlarrAPIKey = v
	}
	if v := os.Getenv("SEEDGHOST_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}

	return cfg, nil
}

// Save writes the configuration to a JSON file.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
