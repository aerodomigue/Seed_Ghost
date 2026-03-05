package client

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	mrand "math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Profile represents a BitTorrent client profile for emulation.
type Profile struct {
	Name               string            `json:"name"`
	PeerIDPrefix       string            `json:"peerIdPrefix"`
	PeerIDSuffixType   string            `json:"peerIdSuffixCharset"` // "random_bytes", "alphanumeric", "alphanumeric_extended"
	PeerIDUrlEncode    *bool             `json:"peerIdUrlEncode"`     // whether to URL-encode peer_id (default true)
	UserAgent          string            `json:"userAgent"`
	KeyLength          int               `json:"keyLength"`
	KeyCharset         string            `json:"keyCharset"`      // "hex_lower", "hex_upper", "hex_upper_no_leading_zero", "alphanumeric"
	UrlEncodingHexCase string            `json:"urlEncodingHexCase"` // "upper" (default) or "lower"
	QueryParamOrder    []string          `json:"queryParamOrder"`
	PortRange          PortRange         `json:"portRange"`
	SupportsCompact    bool              `json:"supportsCompact"`
	NumwantDefault     int               `json:"numwantDefault"`
	NumwantOnStop      int               `json:"numwantOnStop"` // numwant to send on "stopped" event (default: same as numwantDefault)
	ExtraHeaders       map[string]string `json:"extraHeaders"`
	ExtraQueryParams   map[string]string `json:"extraQueryParams"` // e.g. {"corrupt":"0","no_peer_id":"1","supportcrypto":"1"}
}

// ShouldUrlEncodePeerID returns whether peer_id should be URL-encoded.
func (p *Profile) ShouldUrlEncodePeerID() bool {
	if p.PeerIDUrlEncode == nil {
		return true
	}
	return *p.PeerIDUrlEncode
}

// PortRange defines the port range for the emulated client.
type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// GeneratePeerID generates a 20-byte peer_id using the profile's prefix and suffix type.
func (p *Profile) GeneratePeerID() string {
	prefix := p.PeerIDPrefix
	suffixLen := 20 - len(prefix)
	if suffixLen <= 0 {
		return prefix[:20]
	}

	var suffix string
	switch p.PeerIDSuffixType {
	case "alphanumeric":
		const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		b := make([]byte, suffixLen)
		for i := range b {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			b[i] = charset[n.Int64()]
		}
		suffix = string(b)
	case "alphanumeric_extended":
		// Matches JOAL pattern: [A-Za-z0-9_~()\!\.\*-]
		const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_~()!.*-"
		b := make([]byte, suffixLen)
		for i := range b {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			b[i] = charset[n.Int64()]
		}
		suffix = string(b)
	default: // random_bytes
		b := make([]byte, suffixLen)
		rand.Read(b)
		suffix = string(b)
	}

	return prefix + suffix
}

// GenerateKey generates a session key based on the profile's charset.
func (p *Profile) GenerateKey() string {
	length := p.KeyLength
	if length <= 0 {
		length = 8
	}

	switch p.KeyCharset {
	case "hex_upper":
		b := make([]byte, (length+1)/2)
		rand.Read(b)
		return strings.ToUpper(hex.EncodeToString(b))[:length]
	case "hex_upper_no_leading_zero":
		// JOAL HASH_NO_LEADING_ZERO: hex uppercase, first char is never '0'
		for {
			b := make([]byte, (length+1)/2)
			rand.Read(b)
			key := strings.ToUpper(hex.EncodeToString(b))[:length]
			if key[0] != '0' {
				return key
			}
		}
	case "alphanumeric":
		const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		result := make([]byte, length)
		for i := range result {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			result[i] = charset[n.Int64()]
		}
		return string(result)
	default: // hex_lower
		b := make([]byte, (length+1)/2)
		rand.Read(b)
		return hex.EncodeToString(b)[:length]
	}
}

// RandomPort returns a random port within the profile's configured range.
func (p *Profile) RandomPort() int {
	min, max := p.PortRange.Min, p.PortRange.Max
	if min <= 0 {
		min = 10000
	}
	if max <= min {
		max = 65535
	}
	return min + mrand.Intn(max-min+1)
}

// BuildAnnounceURL constructs the announce URL with query params in the profile's defined order.
func (p *Profile) BuildAnnounceURL(baseURL string, params map[string]string) string {
	// Build query string preserving parameter order from profile
	var parts []string
	used := make(map[string]bool)

	for _, key := range p.QueryParamOrder {
		if val, ok := params[key]; ok {
			// info_hash and peer_id are already URL-encoded
			if key == "info_hash" || key == "peer_id" {
				parts = append(parts, key+"="+val)
			} else {
				parts = append(parts, key+"="+url.QueryEscape(val))
			}
			used[key] = true
		}
	}

	// Add any remaining params not in the order list
	for key, val := range params {
		if !used[key] {
			parts = append(parts, key+"="+url.QueryEscape(val))
		}
	}

	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return baseURL + separator + strings.Join(parts, "&")
}

// ProfileStore manages loading and caching of client profiles.
type ProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	dir      string
}

// NewProfileStore creates a new ProfileStore that loads profiles from the given directory.
func NewProfileStore(dir string) *ProfileStore {
	return &ProfileStore{
		profiles: make(map[string]*Profile),
		dir:      dir,
	}
}

// Load reads all JSON profile files from the configured directory.
func (ps *ProfileStore) Load() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.profiles = make(map[string]*Profile)

	entries, err := os.ReadDir(ps.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read profiles dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(ps.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read profile %s: %w", entry.Name(), err)
		}
		var profile Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			return fmt.Errorf("parse profile %s: %w", entry.Name(), err)
		}
		ps.profiles[profile.Name] = &profile
	}

	return nil
}

// Get returns a profile by name.
func (ps *ProfileStore) Get(name string) (*Profile, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	p, ok := ps.profiles[name]
	return p, ok
}

// List returns all available profile names.
func (ps *ProfileStore) List() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	names := make([]string, 0, len(ps.profiles))
	for name := range ps.profiles {
		names = append(names, name)
	}
	return names
}

// All returns all profiles.
func (ps *ProfileStore) All() map[string]*Profile {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	result := make(map[string]*Profile, len(ps.profiles))
	for k, v := range ps.profiles {
		result[k] = v
	}
	return result
}
