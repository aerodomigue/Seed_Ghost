package web

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/anthony/seed_ghost/internal/client"
	"github.com/anthony/seed_ghost/internal/config"
	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/engine"
)

// Server is the HTTP server for the SeedGhost web UI and API.
type Server struct {
	db       *database.DB
	manager  *engine.Manager
	profiles *client.ProfileStore
	config   *config.Config
	frontend fs.FS // embedded frontend files
}

// NewServer creates a new web server.
func NewServer(db *database.DB, manager *engine.Manager, profiles *client.ProfileStore, cfg *config.Config, frontend fs.FS) *Server {
	return &Server{
		db:       db,
		manager:  manager,
		profiles: profiles,
		config:   cfg,
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
	mux.HandleFunc("/api/v1/settings", s.handleSettings)
	mux.HandleFunc("/api/v1/logs", s.handleLogs)
	mux.HandleFunc("/api/v1/ratio-targets", s.handleRatioTargets)
	mux.HandleFunc("/api/v1/clients/profiles", s.handleClientProfiles)
	mux.HandleFunc("/api/v1/clients/refresh", s.handleRefreshProfiles)
	mux.HandleFunc("/api/v1/prowlarr/config", s.handleProwlarrConfig)
	mux.HandleFunc("/api/v1/prowlarr/indexers", s.handleProwlarrIndexers)
	mux.HandleFunc("/api/v1/prowlarr/fetch", s.handleProwlarrFetch)

	// Serve frontend
	if s.frontend != nil {
		fileServer := http.FileServer(http.FS(s.frontend))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// API routes already handled above
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			// Try to serve static file, fall back to index.html for SPA routing
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}
			if _, err := fs.Stat(s.frontend, strings.TrimPrefix(path, "/")); err != nil {
				// Serve index.html for SPA routes
				r.URL.Path = "/index.html"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	return corsMiddleware(mux)
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
		ID            int64  `json:"id"`
		InfoHash      string `json:"infoHash"`
		Name          string `json:"name"`
		TotalSize     int64  `json:"totalSize"`
		TrackerURL    string `json:"trackerUrl"`
		ClientProfile string `json:"clientProfile"`
		Active        bool   `json:"active"`
		AddedAt       string `json:"addedAt"`
		Source        string `json:"source"`
		Uploaded      int64  `json:"uploaded"`
		Leechers      int    `json:"leechers"`
		Seeders       int    `json:"seeders"`
	}

	var result []torrentResponse
	sessions := s.manager.GetSessions()

	for _, row := range rows {
		tr := torrentResponse{
			ID:            row.ID,
			InfoHash:      row.InfoHash,
			Name:          row.Name,
			TotalSize:     row.TotalSize,
			TrackerURL:    row.TrackerURL,
			ClientProfile: row.ClientProfile,
			Active:        row.Active,
			AddedAt:       row.AddedAt.Format("2006-01-02T15:04:05Z"),
			Source:        row.Source,
		}
		if session, ok := sessions[row.ID]; ok {
			state := session.GetState()
			tr.Uploaded = state.Uploaded
			tr.Leechers = state.LastLeechers
			tr.Seeders = state.LastSeeders
		} else if state, err := s.db.GetAnnounceState(row.ID); err == nil {
			tr.Uploaded = state.Uploaded
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

	profileName := r.FormValue("clientProfile")
	if profileName == "" {
		profileName = s.config.DefaultClient
	}
	autoStart := r.FormValue("autoStart") == "true"

	id, err := s.manager.AddTorrent(data, profileName, autoStart)
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

// --- Settings ---

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResponse(w, s.config)
	case "PUT":
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			jsonError(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		// Apply relevant settings
		s.config.DefaultClient = newCfg.DefaultClient
		s.config.MinUploadSpeedKBs = newCfg.MinUploadSpeedKBs
		s.config.MaxUploadSpeedKBs = newCfg.MaxUploadSpeedKBs
		s.config.LogRetentionDays = newCfg.LogRetentionDays
		s.config.ProwlarrURL = newCfg.ProwlarrURL
		s.config.ProwlarrAPIKey = newCfg.ProwlarrAPIKey
		s.config.FetchInterval = newCfg.FetchInterval

		// Update ratio config in manager
		s.manager.UpdateConfig(engine.RatioConfig{
			MinSpeedKBs: s.config.MinUploadSpeedKBs,
			MaxSpeedKBs: s.config.MaxUploadSpeedKBs,
		})

		jsonResponse(w, s.config)
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
		jsonResponse(w, map[string]interface{}{
			"url":                  s.config.ProwlarrURL,
			"apiKey":               s.config.ProwlarrAPIKey,
			"fetchIntervalMinutes": s.config.FetchInterval,
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
		s.config.ProwlarrURL = body.URL
		s.config.ProwlarrAPIKey = body.APIKey
		if body.FetchInterval > 0 {
			s.config.FetchInterval = body.FetchInterval
		}
		jsonResponse(w, map[string]string{"status": "updated"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProwlarrIndexers(w http.ResponseWriter, r *http.Request) {
	// Placeholder — implemented in Phase 4
	switch r.Method {
	case "GET":
		jsonResponse(w, []interface{}{})
	case "PUT":
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
	// Placeholder — implemented in Phase 4
	jsonResponse(w, map[string]string{"status": "fetch triggered"})
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
