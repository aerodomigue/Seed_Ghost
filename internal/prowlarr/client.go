package prowlarr

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
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
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Indexer represents a Prowlarr indexer (IndexerResource).
type Indexer struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Protocol           string `json:"protocol"`
	Enable             bool   `json:"enable"`
	ImplementationName string `json:"implementationName"`
	InfoLink           string `json:"infoLink"`
}

// SearchResult represents a torrent search result (ReleaseResource).
type SearchResult struct {
	GUID        string     `json:"guid"`
	Title       string     `json:"title"`
	IndexerID   int        `json:"indexerId"`
	Indexer     string     `json:"indexer"`
	DownloadURL string     `json:"downloadUrl"`
	MagnetURL   string     `json:"magnetUrl"`
	InfoURL     string     `json:"infoUrl"`
	Size        int64      `json:"size"`
	Seeders     int        `json:"seeders"`
	Leechers    int        `json:"leechers"`
	Protocol    string     `json:"protocol"`
	PublishDate string     `json:"publishDate"`
	Categories  []Category `json:"categories"`
}

// Category represents a Prowlarr category.
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
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

// Search performs a search across specified indexers.
// Returns results filtered to torrent protocol and sorted by leechers descending.
func (c *Client) Search(indexerIDs []int, query string, categories []int) ([]SearchResult, error) {
	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	}
	params.Set("type", "search")
	for _, id := range indexerIDs {
		params.Add("indexerIds", fmt.Sprintf("%d", id))
	}
	for _, cat := range categories {
		params.Add("categories", fmt.Sprintf("%d", cat))
	}

	searchPath := "/api/v1/search?" + params.Encode()
	log.Printf("[prowlarr] search URL: %s%s", c.baseURL, searchPath)
	body, err := c.get(searchPath)
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

	// Sort by leechers descending
	sort.Slice(torrents, func(i, j int) bool {
		return torrents[i].Leechers > torrents[j].Leechers
	})

	return torrents, nil
}

// DownloadTorrent downloads a .torrent file from a search result's downloadUrl.
// Rewrites the URL to use the /api/v1/indexer/{id}/download endpoint so that
// requests go through the Prowlarr API path (bypasses reverse proxy auth).
func (c *Client) DownloadTorrent(downloadURL string) ([]byte, error) {
	downloadURL = c.rewriteDownloadURL(downloadURL)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("download: empty response")
	}

	// Validate bencode: .torrent files always start with 'd' (dict)
	if data[0] != 'd' {
		preview := string(data)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("download: not a .torrent file (got: %q...)", preview)
	}

	return data, nil
}

// TestConnection tests the Prowlarr API connection.
func (c *Client) TestConnection() error {
	_, err := c.get("/api/v1/system/status")
	return err
}

func (c *Client) get(path string) ([]byte, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prowlarr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("prowlarr read response: %w", err)
	}

	// Prowlarr returns HTML when the URL hits the SPA frontend instead of the API
	// This happens when the base URL is wrong (e.g. missing /prowlarr prefix)
	if len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("prowlarr returned HTML instead of JSON — the URL %q does not point to the Prowlarr API. Make sure the URL is correct (e.g. http://localhost:9696 or http://host/prowlarr)", c.baseURL)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("prowlarr: invalid API key (HTTP %d)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("prowlarr: %s (HTTP %d)", apiErr.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("prowlarr: HTTP %d", resp.StatusCode)
	}

	return body, nil
}

// downloadPathRe matches Prowlarr download paths like /{indexerId}/download
var downloadPathRe = regexp.MustCompile(`/(\d+)/download`)

// rewriteDownloadURL rewrites a Prowlarr download URL to use the API endpoint.
// Prowlarr returns: https://host/{id}/download?apikey=...&link=...
// We rewrite to:    {baseURL}/api/v1/indexer/{id}/download?link=...
// This ensures all requests go through the /api/ path (bypasses reverse proxy auth).
func (c *Client) rewriteDownloadURL(dlURL string) string {
	parsed, err := url.Parse(dlURL)
	if err != nil {
		return dlURL
	}

	m := downloadPathRe.FindStringSubmatch(parsed.Path)
	if m == nil {
		// Not a Prowlarr download URL, resolve relative to base
		if strings.HasPrefix(dlURL, "/") {
			return c.baseURL + dlURL
		}
		return dlURL
	}

	idxID := m[1]
	q := parsed.Query()
	q.Del("apikey") // we use the X-Api-Key header instead

	apiURL := fmt.Sprintf("%s/api/v1/indexer/%s/download?%s", c.baseURL, idxID, q.Encode())
	return apiURL
}
