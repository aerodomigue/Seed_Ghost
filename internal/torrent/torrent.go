package torrent

import (
	"crypto/sha1"
	"fmt"
	"net/url"
	"os"

	"github.com/aerodomigue/Seed_Ghost/internal/bencode"
)

// Torrent represents a parsed .torrent file.
type Torrent struct {
	InfoHash    [20]byte
	Name        string
	TotalSize   int64
	PieceLength int64
	Trackers    []string // All announce URLs (announce + announce-list)
	RawBytes    []byte   // Original .torrent file bytes
}

// ParseFile reads and parses a .torrent file from disk.
func ParseFile(path string) (*Torrent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read torrent file: %w", err)
	}
	return Parse(data)
}

// Parse parses a .torrent file from raw bytes.
func Parse(data []byte) (*Torrent, error) {
	decoded, err := bencode.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode torrent: %w", err)
	}

	dict, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("torrent is not a dict")
	}

	// Extract info hash from raw info dict
	rawInfo, err := bencode.ExtractRawValue(data, "info")
	if err != nil {
		return nil, fmt.Errorf("extract info dict: %w", err)
	}
	infoHash := sha1.Sum(rawInfo)

	// Parse info dict
	infoVal, ok := dict["info"]
	if !ok {
		return nil, fmt.Errorf("missing info dict")
	}
	info, ok := infoVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info is not a dict")
	}

	t := &Torrent{
		InfoHash: infoHash,
		RawBytes: data,
	}

	// Name
	if name, ok := info["name"].(string); ok {
		t.Name = name
	}

	// Piece length
	if pl, ok := info["piece length"].(int64); ok {
		t.PieceLength = pl
	}

	// Total size - single file or multi-file
	if length, ok := info["length"].(int64); ok {
		t.TotalSize = length
	} else if files, ok := info["files"].([]interface{}); ok {
		for _, f := range files {
			if fd, ok := f.(map[string]interface{}); ok {
				if length, ok := fd["length"].(int64); ok {
					t.TotalSize += length
				}
			}
		}
	}

	// Trackers
	t.Trackers = extractTrackers(dict)

	return t, nil
}

// InfoHashHex returns the info hash as a hex string.
func (t *Torrent) InfoHashHex() string {
	return fmt.Sprintf("%x", t.InfoHash)
}

// InfoHashURLEncoded returns the info hash URL-encoded for tracker announces.
func (t *Torrent) InfoHashURLEncoded() string {
	var result []byte
	for _, b := range t.InfoHash {
		result = append(result, []byte(url.QueryEscape(string([]byte{b})))...)
	}
	return string(result)
}

func extractTrackers(dict map[string]interface{}) []string {
	seen := make(map[string]bool)
	var trackers []string

	addTracker := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			trackers = append(trackers, u)
		}
	}

	// Main announce
	if announce, ok := dict["announce"].(string); ok {
		addTracker(announce)
	}

	// Announce list (list of lists)
	if al, ok := dict["announce-list"].([]interface{}); ok {
		for _, tier := range al {
			if tierList, ok := tier.([]interface{}); ok {
				for _, u := range tierList {
					if us, ok := u.(string); ok {
						addTracker(us)
					}
				}
			}
		}
	}

	return trackers
}
