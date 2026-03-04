package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePeerID(t *testing.T) {
	p := &Profile{
		PeerIDPrefix:     "-qB4620-",
		PeerIDSuffixType: "random_bytes",
	}
	peerID := p.GeneratePeerID()
	if len(peerID) != 20 {
		t.Errorf("peer_id length = %d, want 20", len(peerID))
	}
	if !strings.HasPrefix(peerID, "-qB4620-") {
		t.Errorf("peer_id should start with -qB4620-, got %q", peerID[:8])
	}

	// Two calls should produce different IDs
	peerID2 := p.GeneratePeerID()
	if peerID == peerID2 {
		t.Error("two generated peer IDs should differ")
	}
}

func TestGeneratePeerIDAlphanumeric(t *testing.T) {
	p := &Profile{
		PeerIDPrefix:     "-TR4050-",
		PeerIDSuffixType: "alphanumeric",
	}
	peerID := p.GeneratePeerID()
	if len(peerID) != 20 {
		t.Errorf("peer_id length = %d, want 20", len(peerID))
	}
}

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		charset string
		length  int
	}{
		{"hex_lower", 8},
		{"hex_upper", 8},
		{"alphanumeric", 10},
	}
	for _, tt := range tests {
		p := &Profile{KeyLength: tt.length, KeyCharset: tt.charset}
		key := p.GenerateKey()
		if len(key) != tt.length {
			t.Errorf("key length = %d, want %d (charset=%s)", len(key), tt.length, tt.charset)
		}
	}
}

func TestRandomPort(t *testing.T) {
	p := &Profile{PortRange: PortRange{Min: 10000, Max: 10010}}
	for i := 0; i < 100; i++ {
		port := p.RandomPort()
		if port < 10000 || port > 10010 {
			t.Errorf("port %d out of range [10000, 10010]", port)
		}
	}
}

func TestBuildAnnounceURL(t *testing.T) {
	p := &Profile{
		QueryParamOrder: []string{"info_hash", "peer_id", "port", "uploaded", "downloaded", "left"},
	}
	params := map[string]string{
		"info_hash":  "%01%02%03",
		"peer_id":    "%2DqB4620%2D",
		"port":       "12345",
		"uploaded":   "0",
		"downloaded": "0",
		"left":       "0",
	}
	result := p.BuildAnnounceURL("http://tracker.example.com/announce", params)
	if !strings.HasPrefix(result, "http://tracker.example.com/announce?") {
		t.Errorf("unexpected URL prefix: %s", result)
	}
	// Check that params are in correct order
	query := result[strings.Index(result, "?")+1:]
	parts := strings.Split(query, "&")
	expectedOrder := []string{"info_hash", "peer_id", "port", "uploaded", "downloaded", "left"}
	for i, part := range parts {
		key := strings.Split(part, "=")[0]
		if key != expectedOrder[i] {
			t.Errorf("param %d: got %s, want %s", i, key, expectedOrder[i])
		}
	}
}

func TestProfileStore(t *testing.T) {
	dir := t.TempDir()
	profileJSON := `{
		"name": "Test Client 1.0",
		"peerIdPrefix": "-TC1000-",
		"peerIdSuffixCharset": "random_bytes",
		"userAgent": "TestClient/1.0",
		"keyLength": 8,
		"keyCharset": "hex_lower",
		"queryParamOrder": ["info_hash","peer_id","port","uploaded","downloaded","left"],
		"portRange": {"min": 10000, "max": 65535},
		"supportsCompact": true,
		"numwantDefault": 200,
		"extraHeaders": {}
	}`
	err := os.WriteFile(filepath.Join(dir, "test-client-1.0.json"), []byte(profileJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	store := NewProfileStore(dir)
	if err := store.Load(); err != nil {
		t.Fatalf("load error: %v", err)
	}

	names := store.List()
	if len(names) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(names))
	}

	p, ok := store.Get("Test Client 1.0")
	if !ok {
		t.Fatal("profile not found")
	}
	if p.UserAgent != "TestClient/1.0" {
		t.Errorf("userAgent = %q, want %q", p.UserAgent, "TestClient/1.0")
	}
}
