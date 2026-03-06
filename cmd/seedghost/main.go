package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/aerodomigue/Seed_Ghost/internal/bootstrap"
	"github.com/aerodomigue/Seed_Ghost/internal/client"
	"github.com/aerodomigue/Seed_Ghost/internal/config"
	"github.com/aerodomigue/Seed_Ghost/internal/database"
	"github.com/aerodomigue/Seed_Ghost/internal/engine"
	"github.com/aerodomigue/Seed_Ghost/internal/web"
	webembed "github.com/aerodomigue/Seed_Ghost/web"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("SeedGhost starting...")

	// Load config from file + env vars (defaults)
	fileCfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(fileCfg.DataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// Run first-start initialization (bootstrap profiles, etc.)
	if err := bootstrap.Run(fileCfg.DataDir, fileCfg.ProfilesDir); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	// Open database
	db, err := database.Open(fileCfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	// Create config service (single source of truth: defaults + DB overrides)
	cfgService := config.NewService(fileCfg, db)

	// Load client profiles
	cfg := cfgService.Get()
	profilesDir := cfg.ProfilesDir
	if !filepath.IsAbs(profilesDir) {
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		candidates := []string{
			filepath.Join(exeDir, profilesDir),
			profilesDir,
			filepath.Join(exeDir, "internal", "client", "profiles"),
			filepath.Join("internal", "client", "profiles"),
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				profilesDir = p
				break
			}
		}
	}

	profiles := client.NewProfileStore(profilesDir)
	if err := profiles.Load(); err != nil {
		log.Printf("warning: load profiles: %v", err)
	}
	log.Printf("Loaded %d client profiles from %s", len(profiles.List()), profilesDir)

	// Create engine manager
	ratioCfg := engine.RatioConfig{
		MinSpeedKBs:         cfg.MinUploadSpeedKBs,
		MaxSpeedKBs:         cfg.MaxUploadSpeedKBs,
		MinDownloadSpeedKBs: cfg.MinDownloadSpeedKBs,
		MaxDownloadSpeedKBs: cfg.MaxDownloadSpeedKBs,
	}
	manager := engine.NewManager(db, profiles, ratioCfg, cfg.DefaultClient)

	// Restore active sessions
	if err := manager.RestoreActiveSessions(); err != nil {
		log.Printf("warning: restore sessions: %v", err)
	}

	// Try to load embedded frontend
	var frontendFS fs.FS
	frontendFS, err = getFrontendFS()
	if err != nil {
		log.Printf("warning: embedded frontend not available: %v", err)
	}

	// Create web server
	srv := web.NewServer(db, manager, profiles, cfgService, frontendFS)

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: srv.Handler(),
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		manager.Shutdown()
		httpServer.Close()
	}()

	log.Printf("SeedGhost listening on %s", cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getFrontendFS() (fs.FS, error) {
	// Try embedded frontend first (production binary)
	if fsys, err := webembed.FrontendFS(); err == nil {
		return fsys, nil
	}
	// Fall back to filesystem (dev mode)
	if info, err := os.Stat("web/dist"); err == nil && info.IsDir() {
		return os.DirFS("web/dist"), nil
	}
	return nil, fmt.Errorf("no frontend available (run 'cd frontend && npm run build' first)")
}
