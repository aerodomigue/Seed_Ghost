package announce

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthony/seed_ghost/internal/bencode"
	"github.com/anthony/seed_ghost/internal/client"
)

func TestParseResponseSuccess(t *testing.T) {
	respData := map[string]interface{}{
		"interval":   int64(1800),
		"complete":   int64(10),
		"incomplete": int64(5),
		"peers":      "",
	}
	data, _ := bencode.Encode(respData)

	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp.Interval != 1800 {
		t.Errorf("interval = %d, want 1800", resp.Interval)
	}
	if resp.Seeders != 10 {
		t.Errorf("seeders = %d, want 10", resp.Seeders)
	}
	if resp.Leechers != 5 {
		t.Errorf("leechers = %d, want 5", resp.Leechers)
	}
}

func TestParseResponseFailure(t *testing.T) {
	respData := map[string]interface{}{
		"failure reason": "torrent not registered",
	}
	data, _ := bencode.Encode(respData)

	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp.FailureMsg != "torrent not registered" {
		t.Errorf("failure msg = %q", resp.FailureMsg)
	}
}

func TestParseResponseWithWarning(t *testing.T) {
	respData := map[string]interface{}{
		"interval":        int64(900),
		"complete":        int64(5),
		"incomplete":      int64(2),
		"warning message": "please update your client",
		"peers":           "",
	}
	data, _ := bencode.Encode(respData)

	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp.WarningMsg != "please update your client" {
		t.Errorf("warning = %q", resp.WarningMsg)
	}
}

func TestParseResponseMinInterval(t *testing.T) {
	respData := map[string]interface{}{
		"interval":     int64(1800),
		"min interval": int64(900),
		"complete":     int64(1),
		"incomplete":   int64(0),
		"peers":        "",
	}
	data, _ := bencode.Encode(respData)

	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp.MinInterval != 900 {
		t.Errorf("min interval = %d, want 900", resp.MinInterval)
	}
}

func TestDoWithMockServer(t *testing.T) {
	respData := map[string]interface{}{
		"interval":   int64(1800),
		"complete":   int64(10),
		"incomplete": int64(3),
		"peers":      "",
	}
	data, _ := bencode.Encode(respData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user agent
		if r.Header.Get("User-Agent") != "qBittorrent/4.6.2" {
			t.Errorf("user-agent = %q", r.Header.Get("User-Agent"))
		}
		// Verify query params exist
		q := r.URL.Query()
		if q.Get("port") == "" {
			t.Error("missing port param")
		}
		w.Write(data)
	}))
	defer server.Close()

	profile := &client.Profile{
		UserAgent:       "qBittorrent/4.6.2",
		QueryParamOrder: []string{"info_hash", "peer_id", "port", "uploaded", "downloaded", "left", "key", "compact", "numwant"},
		SupportsCompact: true,
		NumwantDefault:  200,
		ExtraHeaders:    map[string]string{"Accept-Encoding": "gzip"},
	}

	params := &Params{
		TrackerURL: server.URL,
		InfoHash:   [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		PeerID:     "-qB4620-abcdefghijkl",
		Port:       12345,
		Uploaded:   1024,
		Downloaded: 0,
		Left:       0,
		Key:        "a1b2c3d4",
	}

	resp, err := Do(params, profile)
	if err != nil {
		t.Fatalf("announce error: %v", err)
	}
	if resp.Interval != 1800 {
		t.Errorf("interval = %d, want 1800", resp.Interval)
	}
	if resp.Leechers != 3 {
		t.Errorf("leechers = %d, want 3", resp.Leechers)
	}
}

func TestParseResponseInvalid(t *testing.T) {
	_, err := ParseResponse([]byte("not bencode"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}
