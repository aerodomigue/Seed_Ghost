package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthony/seed_ghost/internal/bencode"
	"github.com/anthony/seed_ghost/internal/client"
	"github.com/anthony/seed_ghost/internal/config"
	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/engine"
)

func testServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	profileDir := t.TempDir()
	profiles := client.NewProfileStore(profileDir)

	cfg := config.DefaultConfig()
	cfg.AutoStart = false
	cfgService := config.NewService(cfg, db)
	ratioCfg := engine.DefaultRatioConfig()
	manager := engine.NewManager(db, profiles, ratioCfg, cfg.DefaultClient)
	t.Cleanup(func() { manager.Shutdown() })

	s := NewServer(db, manager, profiles, cfgService, nil)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	return s, ts
}

func TestListTorrentsEmpty(t *testing.T) {
	_, ts := testServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/torrents")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var result []interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

func makeTorrentData(t *testing.T) []byte {
	t.Helper()
	data := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
		"info": map[string]interface{}{
			"name":         "test.txt",
			"piece length": int64(262144),
			"length":       int64(1024),
			"pieces":       string(make([]byte, 20)),
		},
	}
	encoded, err := bencode.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestAddAndListTorrent(t *testing.T) {
	_, ts := testServer(t)

	torrentData := makeTorrentData(t)

	// Add torrent via multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("torrent", "test.torrent")
	part.Write(torrentData)
	writer.Close()

	resp, err := http.Post(ts.URL+"/api/v1/torrents", writer.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("add status = %d, want 200", resp.StatusCode)
	}

	var addResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&addResult)
	if addResult["id"] == nil {
		t.Error("expected id in response")
	}

	// List torrents
	resp2, _ := http.Get(ts.URL + "/api/v1/torrents")
	defer resp2.Body.Close()

	var list []map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("expected 1 torrent, got %d", len(list))
	}
	if list[0]["name"] != "test.txt" {
		t.Errorf("name = %v, want test.txt", list[0]["name"])
	}
}

func TestDeleteTorrent(t *testing.T) {
	_, ts := testServer(t)

	torrentData := makeTorrentData(t)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("torrent", "test.torrent")
	part.Write(torrentData)
	writer.Close()

	resp, _ := http.Post(ts.URL+"/api/v1/torrents", writer.FormDataContentType(), &buf)
	var addResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&addResult)
	resp.Body.Close()
	id := int(addResult["id"].(float64))

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/torrents/%d", ts.URL, id), nil)
	resp2, _ := http.DefaultClient.Do(req)
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Errorf("delete status = %d, want 200", resp2.StatusCode)
	}
}

func TestStatsOverview(t *testing.T) {
	_, ts := testServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/stats/overview")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats["totalTorrents"].(float64) != 0 {
		t.Errorf("totalTorrents = %v, want 0", stats["totalTorrents"])
	}
}

func TestSettings(t *testing.T) {
	_, ts := testServer(t)

	// GET settings
	resp, _ := http.Get(ts.URL + "/api/v1/settings")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET settings status = %d", resp.StatusCode)
	}

	// PUT settings
	body := `{"defaultClient":"Transmission 4.0.5","minUploadSpeedKBs":100,"maxUploadSpeedKBs":10000}`
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req)
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("PUT settings status = %d", resp2.StatusCode)
	}
}

func TestLogsEndpoint(t *testing.T) {
	_, ts := testServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/logs")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRatioTargets(t *testing.T) {
	_, ts := testServer(t)

	// GET
	resp, _ := http.Get(ts.URL + "/api/v1/ratio-targets")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET status = %d", resp.StatusCode)
	}

	// PUT
	body := `{"tracker.example.com": 2.5}`
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/ratio-targets", strings.NewReader(body))
	resp2, _ := http.DefaultClient.Do(req)
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("PUT status = %d", resp2.StatusCode)
	}
}

func TestClientProfiles(t *testing.T) {
	_, ts := testServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/clients/profiles")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestCORS(t *testing.T) {
	_, ts := testServer(t)

	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/v1/torrents", nil)
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}
