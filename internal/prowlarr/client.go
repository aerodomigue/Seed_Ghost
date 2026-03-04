package prowlarr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// Client is a Prowlarr API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Prowlarr API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Indexer represents a Prowlarr indexer.
type Indexer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Enable   bool   `json:"enable"`
}

// SearchResult represents a torrent search result from Prowlarr.
type SearchResult struct {
	Title       string `json:"title"`
	GUID        string `json:"guid"`
	IndexerID   int    `json:"indexerId"`
	DownloadURL string `json:"downloadUrl"`
	Size        int64  `json:"size"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	Protocol    string `json:"protocol"`
}

// GetIndexers returns all configured indexers.
func (c *Client) GetIndexers() ([]Indexer, error) {
	body, err := c.get("/api/v1/indexer")
	if err != nil {
		return nil, err
	}
	var indexers []Indexer
	if err := json.Unmarshal(body, &indexers); err != nil {
		return nil, fmt.Errorf("decode indexers: %w", err)
	}
	return indexers, nil
}

// Search performs a search on a specific indexer.
func (c *Client) Search(indexerIDs []int, query string) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", "search")
	for _, id := range indexerIDs {
		params.Add("indexerIds", fmt.Sprintf("%d", id))
	}

	body, err := c.get("/api/v1/search?" + params.Encode())
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("decode search results: %w", err)
	}

	// Filter to torrent protocol only
	var torrents []SearchResult
	for _, r := range results {
		if r.Protocol == "torrent" {
			torrents = append(torrents, r)
		}
	}

	// Sort by leechers (most first)
	sort.Slice(torrents, func(i, j int) bool {
		return torrents[i].Leechers > torrents[j].Leechers
	})

	return torrents, nil
}

// DownloadTorrent downloads the .torrent file from a search result.
func (c *Client) DownloadTorrent(downloadURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download torrent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download torrent: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) get(path string) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(path) > 0 && path[0] != '/' {
		reqURL = c.baseURL + "/" + path
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prowlarr request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
