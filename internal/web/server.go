package web

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aerodomigue/Seed_Ghost/internal/client"
	"github.com/aerodomigue/Seed_Ghost/internal/config"
	"github.com/aerodomigue/Seed_Ghost/internal/database"
	"github.com/aerodomigue/Seed_Ghost/internal/engine"
	"github.com/aerodomigue/Seed_Ghost/internal/prowlarr"
)

// Server is the HTTP server for the SeedGhost web UI and API.
type Server struct {
	db       *database.DB
	manager  *engine.Manager
	profiles *client.ProfileStore
	config   *config.Service
	frontend fs.FS // embedded frontend files
}

// NewServer creates a new web server.
func NewServer(db *database.DB, manager *engine.Manager, profiles *client.ProfileStore, cfgService *config.Service, frontend fs.FS) *Server {
	return &Server{
		db:       db,
		manager:  manager,
		profiles: profiles,
		config:   cfgService,
		frontend: frontend,
	}
}

// Handler returns the HTTP handler with all routes configured.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/torrents", s.handleTorrents)
	mux.HandleFunc("/api/v1/torrents/", s.handleTorrentByID)
	mux.HandleFunc("/api/v1/stats/overview", s.handleStatsOverview)
	mux.HandleFunc("/api/v1/stats/history", s.handleStatsHistory)
	mux.HandleFunc("/api/v1/settings", s.handleSettings)
	mux.HandleFunc("/api/v1/logs", s.handleLogs)
	mux.HandleFunc("/api/v1/ratio-targets", s.handleRatioTargets)
	mux.HandleFunc("/api/v1/clients/profiles", s.handleClientProfiles)
	mux.HandleFunc("/api/v1/clients/refresh", s.handleRefreshProfiles)
	mux.HandleFunc("/api/v1/prowlarr/config", s.handleProwlarrConfig)
	mux.HandleFunc("/api/v1/prowlarr/indexers", s.handleProwlarrIndexers)
	mux.HandleFunc("/api/v1/prowlarr/fetch", s.handleProwlarrFetch)

	// Serve frontend — wrap mux to handle SPA fallback for non-API routes
	handler := http.Handler(mux)
	if s.frontend != nil {
		fileServer := http.FileServer(http.FS(s.frontend))
		// Pre-read index.html for SPA fallback (avoids FileServer redirect loops)
		indexHTML, _ := fs.ReadFile(s.frontend, "index.html")

		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Let API routes go through the mux
			if strings.HasPrefix(r.URL.Path, "/api/") {
				mux.ServeHTTP(w, r)
				return
			}
			// Try to serve static file (JS, CSS, images, etc.)
			if r.URL.Path != "/" {
				if _, err := fs.Stat(s.frontend, strings.TrimPrefix(r.URL.Path, "/")); err == nil {
					fileServer.ServeHTTP(w, r)
					return
				}
			}
			// SPA fallback: serve index.html directly
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
		})
	}

	return corsMiddleware(handler)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Torrents ---

func (s *Server) handleTorrents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listTorrents(w, r)
	case "POST":
		s.addTorrent(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listTorrents(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.ListTorrents()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type torrentResponse struct {
		ID                  int64   `json:"id"`
		InfoHash            string  `json:"infoHash"`
		Name                string  `json:"name"`
		TotalSize           int64   `json:"totalSize"`
		TrackerURL          string  `json:"trackerUrl"`
		ClientProfile       string  `json:"clientProfile"`
		Active              bool    `json:"active"`
		Status              string  `json:"status"` // "stopped", "pending", "downloading", "seeding"
		AddedAt             string  `json:"addedAt"`
		Source              string  `json:"source"`
		Uploaded            int64   `json:"uploaded"`
		UploadSpeed         float64 `json:"uploadSpeed"` // bytes/s
		Downloaded          int64   `json:"downloaded"`
		DownloadSpeed       float64 `json:"downloadSpeed"` // bytes/s
		DownloadComplete    bool    `json:"downloadComplete"`
		Leechers            int     `json:"leechers"`
		Seeders             int     `json:"seeders"`
		IndexerID           *int64  `json:"indexerId"`
		SeedTimeRemainingMs *int64  `json:"seedTimeRemainingMs"`
	}

	var result []torrentResponse
	sessions := s.manager.GetSessions()

	for _, row := range rows {
		tr := torrentResponse{
			ID:                  row.ID,
			InfoHash:            row.InfoHash,
			Name:                row.Name,
			TotalSize:           row.TotalSize,
			TrackerURL:          row.TrackerURL,
			ClientProfile:       row.ClientProfile,
			Active:              row.Active,
			Status:              "stopped",
			AddedAt:             row.AddedAt.Format("2006-01-02T15:04:05Z"),
			Source:              row.Source,
			IndexerID:           row.IndexerID,
			SeedTimeRemainingMs: row.SeedTimeRemainingMs,
		}
		if session, ok := sessions[row.ID]; ok {
			state := session.GetState()
			tr.Uploaded = state.Uploaded
			tr.Downloaded = state.Downloaded
			tr.Leechers = state.LastLeechers
			tr.Seeders = state.LastSeeders
			tr.UploadSpeed = session.GetSpeed()
			tr.DownloadSpeed = session.GetDownloadSpeed()
			tr.DownloadComplete = session.IsDownloadComplete()
			tr.SeedTimeRemainingMs = session.GetSeedTimeRemainingMs()
			if session.HasAnnounced() {
				if !tr.DownloadComplete {
					tr.Status = "downloading"
				} else {
					tr.Status = "seeding"
				}
			} else {
				tr.Status = "pending"
			}
		} else if state, err := s.db.GetAnnounceState(row.ID); err == nil {
			tr.Uploaded = state.Uploaded
			tr.Downloaded = state.Downloaded
			tr.DownloadComplete = state.Downloaded >= row.TotalSize
			tr.Leechers = state.LastLeechers
			tr.Seeders = state.LastSeeders
		}
		result = append(result, tr)
	}

	if result == nil {
		result = []torrentResponse{}
	}
	jsonResponse(w, result)
}

func (s *Server) addTorrent(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form for file upload
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "invalid form data", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("torrent")
	if err != nil {
		jsonError(w, "missing torrent file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "read file error", http.StatusInternalServerError)
		return
	}

	cfg := s.config.Get()
	id, err := s.manager.AddTorrent(data, cfg.DefaultClient, cfg.AutoStart, nil, nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, map[string]interface{}{"id": id})
}

func (s *Server) handleTorrentByID(w http.ResponseWriter, r *http.Request) {
	// Parse ID from path: /api/v1/torrents/{id}[/action]
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/torrents/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonError(w, "invalid torrent ID", http.StatusBadRequest)
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "start":
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := s.manager.StartTorrent(id); err != nil {
				jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			jsonResponse(w, map[string]string{"status": "started"})
		case "stop":
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := s.manager.StopTorrent(id); err != nil {
				jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			jsonResponse(w, map[string]string{"status": "stopped"})
		default:
			http.NotFound(w, r)
		}
		return
	}

	switch r.Method {
	case "DELETE":
		if err := s.manager.RemoveTorrent(id); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Stats ---

func (s *Server) handleStatsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := s.db.ListTorrents()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var totalUploaded int64
	activeTorrents := 0
	totalTorrents := len(rows)
	sessions := s.manager.GetSessions()

	for _, row := range rows {
		if row.Active {
			activeTorrents++
		}
		if session, ok := sessions[row.ID]; ok {
			state := session.GetState()
			totalUploaded += state.Uploaded
		} else if state, err := s.db.GetAnnounceState(row.ID); err == nil {
			totalUploaded += state.Uploaded
		}
	}

	jsonResponse(w, map[string]interface{}{
		"totalTorrents":  totalTorrents,
		"activeTorrents": activeTorrents,
		"totalUploaded":  totalUploaded,
	})
}

func (s *Server) handleStatsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 {
			hours = v
		}
	}

	points, err := s.db.GetStatsHistory(hours)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if points == nil {
		points = []database.StatsHistoryPoint{}
	}
	jsonResponse(w, points)
}

// --- Settings ---

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResponse(w, s.config.Get())
	case "PUT":
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		s.config.Update(func(cfg *config.Config) {
			cfg.DefaultClient = incoming.DefaultClient
			cfg.AutoStart = incoming.AutoStart
			cfg.MinUploadSpeedKBs = incoming.MinUploadSpeedKBs
			cfg.MaxUploadSpeedKBs = incoming.MaxUploadSpeedKBs
			cfg.MinDownloadSpeedKBs = incoming.MinDownloadSpeedKBs
			cfg.MaxDownloadSpeedKBs = incoming.MaxDownloadSpeedKBs
			cfg.LogRetentionDays = incoming.LogRetentionDays
			if incoming.FetchInterval > 0 {
				cfg.FetchInterval = incoming.FetchInterval
			}
			if incoming.ProwlarrMaxSlots > 0 {
				cfg.ProwlarrMaxSlots = incoming.ProwlarrMaxSlots
			}
		})

		// Update ratio config in running sessions
		cfg := s.config.Get()
		s.manager.UpdateConfig(engine.RatioConfig{
			MinSpeedKBs:         cfg.MinUploadSpeedKBs,
			MaxSpeedKBs:         cfg.MaxUploadSpeedKBs,
			MinDownloadSpeedKBs: cfg.MinDownloadSpeedKBs,
			MaxDownloadSpeedKBs: cfg.MaxDownloadSpeedKBs,
		})

		jsonResponse(w, cfg)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Logs ---

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	torrentID, _ := strconv.ParseInt(q.Get("torrentId"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	logs, err := s.db.ListAnnounceLogs(torrentID, limit, offset)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []database.AnnounceLogRow{}
	}
	jsonResponse(w, logs)
}

// --- Ratio Targets ---

func (s *Server) handleRatioTargets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		targets, err := s.db.GetRatioTargets()
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, targets)
	case "PUT":
		var targets map[string]float64
		if err := json.NewDecoder(r.Body).Decode(&targets); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		for host, ratio := range targets {
			if err := s.db.UpsertRatioTarget(host, ratio); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		jsonResponse(w, targets)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Client Profiles ---

func (s *Server) handleClientProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profiles := s.profiles.All()
	jsonResponse(w, profiles)
}

func (s *Server) handleRefreshProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.profiles.Load(); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]interface{}{"profiles": s.profiles.List()})
}

// --- Prowlarr ---

func (s *Server) handleProwlarrConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		cfg := s.config.Get()
		jsonResponse(w, map[string]interface{}{
			"url":                  cfg.ProwlarrURL,
			"apiKey":               cfg.ProwlarrAPIKey,
			"fetchIntervalMinutes": cfg.FetchInterval,
		})
	case "PUT":
		var body struct {
			URL           string `json:"url"`
			APIKey        string `json:"apiKey"`
			FetchInterval int    `json:"fetchIntervalMinutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		// Test connection before saving
		if body.URL != "" && body.APIKey != "" {
			pc := prowlarr.NewClient(body.URL, body.APIKey)
			if err := pc.TestConnection(); err != nil {
				jsonError(w, err.Error(), http.StatusBadGateway)
				return
			}
		}

		s.config.Update(func(cfg *config.Config) {
			cfg.ProwlarrURL = body.URL
			cfg.ProwlarrAPIKey = body.APIKey
			if body.FetchInterval > 0 {
				cfg.FetchInterval = body.FetchInterval
			}
		})

		jsonResponse(w, map[string]string{"status": "updated"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type indexerWithSelection struct {
	ID                   int      `json:"id"`
	Name                 string   `json:"name"`
	Protocol             string   `json:"protocol"`
	Enable               bool     `json:"enable"`
	ImplementationName   string   `json:"implementationName"`
	Selected             bool     `json:"selected"`
	MaxUploadSpeedKBs    *float64 `json:"maxUploadSpeedKbs"`
	FetchIntervalMinutes *int     `json:"fetchIntervalMinutes"`
	MaxSlots             *int     `json:"maxSlots"`
	SeedTimeHours        *int     `json:"seedTimeHours"`
}

func (s *Server) handleProwlarrIndexers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		saved, err := s.db.GetProwlarrIndexers()
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, saved)
	case "POST":
		cfg := s.config.Get()
		if cfg.ProwlarrURL == "" || cfg.ProwlarrAPIKey == "" {
			jsonError(w, "Prowlarr URL and API key must be configured first", http.StatusBadRequest)
			return
		}
		pc := prowlarr.NewClient(cfg.ProwlarrURL, cfg.ProwlarrAPIKey)
		indexers, err := pc.GetIndexers()
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadGateway)
			return
		}

		saved, _ := s.db.GetProwlarrIndexers()
		savedMap := make(map[int64]database.ProwlarrIndexerRow)
		for _, row := range saved {
			savedMap[row.ID] = row
		}

		var result []indexerWithSelection
		for _, idx := range indexers {
			item := indexerWithSelection{
				ID:                 idx.ID,
				Name:               idx.Name,
				Protocol:           idx.Protocol,
				Enable:             idx.Enable,
				ImplementationName: idx.ImplementationName,
			}
			if sel, ok := savedMap[int64(idx.ID)]; ok {
				item.Selected = sel.Enabled
				item.MaxUploadSpeedKBs = sel.MaxUploadSpeedKBs
				item.FetchIntervalMinutes = sel.FetchIntervalMinutes
				item.MaxSlots = sel.MaxSlots
				item.SeedTimeHours = sel.SeedTimeHours
			}
			result = append(result, item)
		}
		if result == nil {
			result = []indexerWithSelection{}
		}
		jsonResponse(w, result)
	case "PUT":
		var selections []struct {
			ID                   int64    `json:"id"`
			Name                 string   `json:"name"`
			Selected             bool     `json:"selected"`
			MaxUploadSpeedKBs    *float64 `json:"maxUploadSpeedKbs"`
			FetchIntervalMinutes *int     `json:"fetchIntervalMinutes"`
			MaxSlots             *int     `json:"maxSlots"`
			SeedTimeHours        *int     `json:"seedTimeHours"`
		}
		if err := json.NewDecoder(r.Body).Decode(&selections); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		for _, sel := range selections {
			if err := s.db.UpsertProwlarrIndexer(sel.ID, sel.Name, sel.Selected, sel.MaxUploadSpeedKBs, sel.FetchIntervalMinutes, sel.MaxSlots, sel.SeedTimeHours); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		jsonResponse(w, map[string]string{"status": "updated"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProwlarrFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.config.Get()
	if cfg.ProwlarrURL == "" || cfg.ProwlarrAPIKey == "" {
		jsonError(w, "Prowlarr URL and API key must be configured first", http.StatusBadRequest)
		return
	}

	prowlarrClient := prowlarr.NewClient(cfg.ProwlarrURL, cfg.ProwlarrAPIKey)
	if _, err := prowlarrClient.GetIndexers(); err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	fetcher := prowlarr.NewFetcher(prowlarrClient, s.db, s.manager, cfg.FetchInterval, cfg.DefaultClient, cfg.ProwlarrMaxSlots)
	go fetcher.FetchNow()

	jsonResponse(w, map[string]string{"status": "fetch started"})
}

// --- Helpers ---

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
