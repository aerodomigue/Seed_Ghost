package bootstrap

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aerodomigue/Seed_Ghost/internal/client/profiles"
)

// Run executes the first-start initialization phase.
// It checks for a .initialized marker file in dataDir and skips if already done,
// unless SEEDGHOST_FORCE_INIT=true is set.
func Run(dataDir, profilesDir string) error {
	markerPath := filepath.Join(dataDir, ".initialized")
	forceInit := strings.EqualFold(os.Getenv("SEEDGHOST_FORCE_INIT"), "true")

	if !forceInit {
		if _, err := os.Stat(markerPath); err == nil {
			log.Println("Already initialized, skipping")
			return nil
		}
	} else {
		log.Println("SEEDGHOST_FORCE_INIT=true, forcing re-initialization")
	}

	log.Println("Running first-start initialization...")

	if err := bootstrapProfiles(profilesDir); err != nil {
		return fmt.Errorf("bootstrap profiles: %w", err)
	}

	// Mark initialization as done
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(markerPath, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0644); err != nil {
		return fmt.Errorf("write init marker: %w", err)
	}

	log.Println("Initialization complete")
	return nil
}

// bootstrapProfiles copies bundled JSON profiles to profilesDir if no .json files exist there.
func bootstrapProfiles(profilesDir string) error {
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return fmt.Errorf("create profiles dir: %w", err)
	}

	// Check if any .json files already exist
	existing, _ := filepath.Glob(filepath.Join(profilesDir, "*.json"))
	if len(existing) > 0 {
		log.Printf("Profiles directory already contains %d profile(s), skipping copy", len(existing))
		return nil
	}

	// Copy bundled profiles
	entries, err := fs.ReadDir(profiles.BundledFS, ".")
	if err != nil {
		return fmt.Errorf("read bundled profiles: %w", err)
	}

	copied := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := fs.ReadFile(profiles.BundledFS, entry.Name())
		if err != nil {
			return fmt.Errorf("read bundled %s: %w", entry.Name(), err)
		}
		dst := filepath.Join(profilesDir, entry.Name())
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", entry.Name(), err)
		}
		copied++
	}

	log.Printf("Copied %d bundled profile(s) to %s", copied, profilesDir)
	return nil
}
