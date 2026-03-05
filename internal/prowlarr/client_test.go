package prowlarr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetIndexers(t *testing.T) {
	indexers := []Indexer{
		{ID: 1, Name: "Test Indexer", Protocol: "torrent", Enable: true},
		{ID: 2, Name: "Another Indexer", Protocol: "torrent", Enable: false},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/indexer" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("missing API key")
		}
		json.NewEncoder(w).Encode(indexers)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.GetIndexers()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 indexers, got %d", len(result))
	}
	if result[0].Name != "Test Indexer" {
		t.Errorf("name = %q", result[0].Name)
	}
}

func TestSearch(t *testing.T) {
	results := []SearchResult{
		{Title: "Popular Torrent", Leechers: 50, Seeders: 100, Protocol: "torrent", DownloadURL: "http://dl.example.com/1.torrent"},
		{Title: "Less Popular", Leechers: 5, Seeders: 10, Protocol: "torrent", DownloadURL: "http://dl.example.com/2.torrent"},
		{Title: "NZB Result", Leechers: 100, Seeders: 200, Protocol: "usenet"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(results)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	torrents, err := client.Search([]int{1}, "test", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// Should only have torrent protocol results, sorted by leechers
	if len(torrents) != 2 {
		t.Fatalf("expected 2 torrents, got %d", len(torrents))
	}
	if torrents[0].Leechers != 50 {
		t.Errorf("first result leechers = %d, want 50", torrents[0].Leechers)
	}
	if torrents[1].Leechers != 5 {
		t.Errorf("second result leechers = %d, want 5", torrents[1].Leechers)
	}
}

func TestSearchSortsByLeechers(t *testing.T) {
	results := []SearchResult{
		{Title: "A", Leechers: 10, Protocol: "torrent"},
		{Title: "B", Leechers: 100, Protocol: "torrent"},
		{Title: "C", Leechers: 50, Protocol: "torrent"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(results)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	torrents, err := client.Search(nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	if torrents[0].Title != "B" || torrents[1].Title != "C" || torrents[2].Title != "A" {
		t.Errorf("wrong sort order: %v", torrents)
	}
}

func TestGetIndexersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	_, err := client.GetIndexers()
	if err == nil {
		t.Error("expected error for 401")
	}
}
