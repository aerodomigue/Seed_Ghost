package torrent

import (
	"testing"

	"github.com/anthony/seed_ghost/internal/bencode"
)

func makeTorrentBytes(t *testing.T, announce string, name string, length int64) []byte {
	t.Helper()
	torrent := map[string]interface{}{
		"announce": announce,
		"info": map[string]interface{}{
			"name":         name,
			"piece length": int64(262144),
			"length":       length,
			"pieces":       string(make([]byte, 20)), // 1 fake piece hash
		},
	}
	data, err := bencode.Encode(torrent)
	if err != nil {
		t.Fatalf("encode torrent: %v", err)
	}
	return data
}

func TestParseSingleFile(t *testing.T) {
	data := makeTorrentBytes(t, "http://tracker.example.com/announce", "test.txt", 1024)
	tor, err := Parse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if tor.Name != "test.txt" {
		t.Errorf("name = %q, want %q", tor.Name, "test.txt")
	}
	if tor.TotalSize != 1024 {
		t.Errorf("totalSize = %d, want 1024", tor.TotalSize)
	}
	if tor.PieceLength != 262144 {
		t.Errorf("pieceLength = %d, want 262144", tor.PieceLength)
	}
	if len(tor.Trackers) != 1 || tor.Trackers[0] != "http://tracker.example.com/announce" {
		t.Errorf("trackers = %v, want [http://tracker.example.com/announce]", tor.Trackers)
	}
	if len(tor.InfoHash) != 20 {
		t.Error("info hash should be 20 bytes")
	}
	if tor.InfoHashHex() == "" {
		t.Error("hex info hash should not be empty")
	}
}

func TestParseMultiFile(t *testing.T) {
	torrent := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
		"info": map[string]interface{}{
			"name":         "mydir",
			"piece length": int64(262144),
			"files": []interface{}{
				map[string]interface{}{
					"length": int64(100),
					"path":   []interface{}{"file1.txt"},
				},
				map[string]interface{}{
					"length": int64(200),
					"path":   []interface{}{"file2.txt"},
				},
			},
			"pieces": string(make([]byte, 20)),
		},
	}
	data, err := bencode.Encode(torrent)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	tor, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tor.TotalSize != 300 {
		t.Errorf("totalSize = %d, want 300", tor.TotalSize)
	}
	if tor.Name != "mydir" {
		t.Errorf("name = %q, want %q", tor.Name, "mydir")
	}
}

func TestParseAnnounceList(t *testing.T) {
	torrent := map[string]interface{}{
		"announce": "http://tracker1.example.com/announce",
		"announce-list": []interface{}{
			[]interface{}{"http://tracker1.example.com/announce", "http://tracker2.example.com/announce"},
			[]interface{}{"http://tracker3.example.com/announce"},
		},
		"info": map[string]interface{}{
			"name":         "test",
			"piece length": int64(262144),
			"length":       int64(100),
			"pieces":       string(make([]byte, 20)),
		},
	}
	data, err := bencode.Encode(torrent)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	tor, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tor.Trackers) != 3 {
		t.Errorf("trackers count = %d, want 3", len(tor.Trackers))
	}
}

func TestParseInvalidData(t *testing.T) {
	_, err := Parse([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestInfoHashConsistency(t *testing.T) {
	data := makeTorrentBytes(t, "http://tracker.example.com/announce", "test.txt", 1024)
	tor1, _ := Parse(data)
	tor2, _ := Parse(data)
	if tor1.InfoHash != tor2.InfoHash {
		t.Error("same torrent data should produce same info hash")
	}
}
