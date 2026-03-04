package prowlarr

import (
	"context"
	"log"
	"time"

	"github.com/anthony/seed_ghost/internal/database"
	"github.com/anthony/seed_ghost/internal/engine"
)

// Fetcher periodically searches Prowlarr for popular torrents and adds them.
type Fetcher struct {
	client   *Client
	db       *database.DB
	manager  *engine.Manager
	interval time.Duration
	cancel   context.CancelFunc
}

// NewFetcher creates a new auto-fetcher.
func NewFetcher(client *Client, db *database.DB, manager *engine.Manager, intervalMinutes int) *Fetcher {
	if intervalMinutes <= 0 {
		intervalMinutes = 30
	}
	return &Fetcher{
		client:   client,
		db:       db,
		manager:  manager,
		interval: time.Duration(intervalMinutes) * time.Minute,
	}
}

// Start begins the periodic fetch loop.
func (f *Fetcher) Start(ctx context.Context) {
	ctx, f.cancel = context.WithCancel(ctx)
	go f.loop(ctx)
}

// Stop stops the fetcher.
func (f *Fetcher) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
}

// FetchNow performs a single fetch immediately.
func (f *Fetcher) FetchNow() {
	f.doFetch()
}

func (f *Fetcher) loop(ctx context.Context) {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	// Do an initial fetch
	f.doFetch()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.doFetch()
		}
	}
}

func (f *Fetcher) doFetch() {
	if f.client == nil {
		return
	}

	log.Println("[prowlarr] starting auto-fetch")

	// Get enabled indexers from DB
	// For now, search all indexers with empty query (browse)
	results, err := f.client.Search(nil, "")
	if err != nil {
		log.Printf("[prowlarr] search error: %v", err)
		return
	}

	// Take top results by leechers
	maxToAdd := 5
	added := 0
	for _, result := range results {
		if added >= maxToAdd {
			break
		}
		if result.Leechers <= 0 || result.DownloadURL == "" {
			continue
		}

		// Check if we already have this torrent
		// (basic duplicate check by title — proper check would use info_hash)
		existing, _ := f.db.ListTorrents()
		duplicate := false
		for _, t := range existing {
			if t.Name == result.Title {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		// Download .torrent file
		torrentData, err := f.client.DownloadTorrent(result.DownloadURL)
		if err != nil {
			log.Printf("[prowlarr] download error for %q: %v", result.Title, err)
			continue
		}

		// Add to manager
		id, err := f.manager.AddTorrent(torrentData, "", true)
		if err != nil {
			log.Printf("[prowlarr] add error for %q: %v", result.Title, err)
			continue
		}

		log.Printf("[prowlarr] added torrent #%d: %s (leechers=%d)", id, result.Title, result.Leechers)
		added++
	}

	log.Printf("[prowlarr] auto-fetch complete: added %d torrents", added)
}
