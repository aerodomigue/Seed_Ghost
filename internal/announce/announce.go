package announce

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/anthony/seed_ghost/internal/bencode"
	"github.com/anthony/seed_ghost/internal/client"
)

// Response represents a parsed tracker announce response.
type Response struct {
	Interval    int
	MinInterval int
	Leechers    int // incomplete
	Seeders     int // complete
	Peers       int
	FailureMsg  string
	WarningMsg  string
}

// Params holds the parameters for an announce request.
type Params struct {
	TrackerURL string
	InfoHash   [20]byte
	PeerID     string
	Port       int
	Uploaded   int64
	Downloaded int64
	Left       int64
	Event      string // "started", "stopped", "completed", or ""
	Key        string
	Compact    bool
	Numwant    int
}

// Do performs an HTTP announce request to a tracker.
func Do(params *Params, profile *client.Profile) (*Response, error) {
	return DoWithClient(params, profile, &http.Client{Timeout: 30 * time.Second})
}

// DoWithClient performs an announce request using the provided HTTP client.
func DoWithClient(params *Params, profile *client.Profile, httpClient *http.Client) (*Response, error) {
	queryParams := buildQueryParams(params, profile)
	announceURL := profile.BuildAnnounceURL(params.TrackerURL, queryParams)

	req, err := http.NewRequest("GET", announceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers from profile
	req.Header.Set("User-Agent", profile.UserAgent)
	for k, v := range profile.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("announce request: %w", err)
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		reader = gr
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return ParseResponse(body)
}

// ParseResponse parses a bencoded tracker response.
func ParseResponse(data []byte) (*Response, error) {
	decoded, err := bencode.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	dict, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response is not a dict")
	}

	resp := &Response{}

	if msg, ok := dict["failure reason"].(string); ok {
		resp.FailureMsg = msg
		return resp, nil
	}
	if msg, ok := dict["warning message"].(string); ok {
		resp.WarningMsg = msg
	}
	if v, ok := dict["interval"].(int64); ok {
		resp.Interval = int(v)
	}
	if v, ok := dict["min interval"].(int64); ok {
		resp.MinInterval = int(v)
	}
	if v, ok := dict["complete"].(int64); ok {
		resp.Seeders = int(v)
	}
	if v, ok := dict["incomplete"].(int64); ok {
		resp.Leechers = int(v)
	}

	// Count peers
	if peers, ok := dict["peers"].(string); ok {
		// Compact format: 6 bytes per peer
		resp.Peers = len(peers) / 6
	} else if peers, ok := dict["peers"].([]interface{}); ok {
		resp.Peers = len(peers)
	}

	return resp, nil
}

func buildQueryParams(params *Params, profile *client.Profile) map[string]string {
	qp := map[string]string{
		"info_hash":  urlEncodeInfoHash(params.InfoHash),
		"peer_id":    urlEncodePeerID(params.PeerID),
		"port":       strconv.Itoa(params.Port),
		"uploaded":   strconv.FormatInt(params.Uploaded, 10),
		"downloaded": strconv.FormatInt(params.Downloaded, 10),
		"left":       strconv.FormatInt(params.Left, 10),
	}

	if params.Key != "" {
		qp["key"] = params.Key
	}
	if params.Event != "" {
		qp["event"] = params.Event
	}
	if profile.SupportsCompact {
		qp["compact"] = "1"
	}
	if profile.NumwantDefault > 0 {
		qp["numwant"] = strconv.Itoa(profile.NumwantDefault)
	}

	return qp
}

func urlEncodeInfoHash(hash [20]byte) string {
	var result []byte
	for _, b := range hash {
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') ||
			b == '-' || b == '_' || b == '.' || b == '~' {
			result = append(result, b)
		} else {
			result = append(result, []byte(fmt.Sprintf("%%%02X", b))...)
		}
	}
	return string(result)
}

func urlEncodePeerID(peerID string) string {
	var result []byte
	for _, b := range []byte(peerID) {
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') ||
			b == '-' || b == '_' || b == '.' || b == '~' {
			result = append(result, b)
		} else {
			result = append(result, []byte(fmt.Sprintf("%%%02X", b))...)
		}
	}
	return string(result)
}
