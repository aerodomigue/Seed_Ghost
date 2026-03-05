package prowlarr

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aerodomigue/Seed_Ghost/internal/database"
	"github.com/aerodomigue/Seed_Ghost/internal/engine"
)

// Fetcher periodically searches Prowlarr for popular torrents and adds them.
type Fetcher struct {
	client          *Client
	db              *database.DB
	manager         *engine.Manager
	defaultInterval time.Duration
	defaultProfile  string // default client profile name for new torrents
	maxSlots        int    // max concurrent active torrents
	cancel          context.CancelFunc
}

// NewFetcher creates a new auto-fetcher.
func NewFetcher(client *Client, db *database.DB, manager *engine.Manager, intervalMinutes int, defaultProfile string, maxSlots int) *Fetcher {
	if intervalMinutes <= 0 {
		intervalMinutes = 1440 // 24h
	}
	if maxSlots <= 0 {
		maxSlots = 5
	}
	return &Fetcher{
		client:          client,
		db:              db,
		manager:         manager,
		defaultInterval: time.Duration(intervalMinutes) * time.Minute,
		defaultProfile:  defaultProfile,
		maxSlots:        maxSlots,
	}
}

// Start begins the periodic fetch loop.
// Checks every minute which indexers need fetching based on their interval.
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

// FetchNow performs a single fetch for all enabled indexers immediately.
func (f *Fetcher) FetchNow() {
	indexers, err := f.db.GetProwlarrIndexers()
	if err != nil {
		log.Printf("[prowlarr] get indexers: %v", err)
		return
	}
	var enabledIDs []int
	for _, idx := range indexers {
		if idx.Enabled {
			enabledIDs = append(enabledIDs, int(idx.ID))
		}
	}
	if len(enabledIDs) == 0 {
		log.Println("[prowlarr] no indexers selected, skipping fetch")
		return
	}
	f.doFetch(enabledIDs)
}

func (f *Fetcher) loop(ctx context.Context) {
	// Check every minute which indexers are due for a fetch
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Initial fetch on startup
	f.checkAndFetch()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.checkAndFetch()
		}
	}
}

// checkAndFetch looks at each enabled indexer's last_fetch time and interval,
// then fetches for indexers that are due.
func (f *Fetcher) checkAndFetch() {
	if f.client == nil {
		return
	}

	indexers, err := f.db.GetProwlarrIndexers()
	if err != nil {
		log.Printf("[prowlarr] get indexers: %v", err)
		return
	}

	now := time.Now()
	var dueIDs []int

	for _, idx := range indexers {
		if !idx.Enabled {
			continue
		}

		interval := f.defaultInterval
		if idx.FetchIntervalMinutes != nil && *idx.FetchIntervalMinutes > 0 {
			interval = time.Duration(*idx.FetchIntervalMinutes) * time.Minute
		}

		lastFetch := f.db.GetIndexerLastFetch(idx.ID)
		if lastFetch.IsZero() || now.Sub(lastFetch) >= interval {
			dueIDs = append(dueIDs, int(idx.ID))
		}
	}

	if len(dueIDs) == 0 {
		return
	}

	log.Printf("[prowlarr] %d indexers due for fetch", len(dueIDs))
	f.doFetch(dueIDs)

	// Update last_fetch for all fetched indexers
	for _, id := range dueIDs {
		f.db.UpdateIndexerLastFetch(int64(id), now)
	}
}


// activeSlot holds info about a currently active torrent session.
type activeSlot struct {
	torrentID int64
	name      string
	leechers  int
	indexerID *int64
}

func (f *Fetcher) doFetch(indexerIDs []int) {
	if f.client == nil || len(indexerIDs) == 0 {
		return
	}

	log.Printf("[prowlarr] searching %d indexers", len(indexerIDs))

	// Search selected indexers — results come back sorted by leechers desc
	results, err := f.client.Search(indexerIDs, "", nil)
	if err != nil {
		log.Printf("[prowlarr] search error: %v", err)
		return
	}

	if len(results) == 0 {
		log.Println("[prowlarr] no results found")
		return
	}

	// Load per-indexer overrides
	savedIndexers, _ := f.db.GetProwlarrIndexers()
	indexerMaxSlots := make(map[int64]int)    // indexerID -> max slots
	indexerSeedTimeH := make(map[int64]int)   // indexerID -> seed time hours
	for _, idx := range savedIndexers {
		if idx.MaxSlots != nil {
			indexerMaxSlots[idx.ID] = *idx.MaxSlots
		}
		if idx.SeedTimeHours != nil {
			indexerSeedTimeH[idx.ID] = *idx.SeedTimeHours
		}
	}

	// Build list of active sessions with their current leechers
	sessions := f.manager.GetSessions()
	existing, _ := f.db.ListTorrents()

	existingHashes := make(map[string]bool, len(existing))
	var activeSlots []activeSlot
	for _, t := range existing {
		existingHashes[t.InfoHash] = true
		leechers := 0
		if sess, ok := sessions[t.ID]; ok {
			state := sess.GetState()
			leechers = state.LastLeechers
		}
		activeSlots = append(activeSlots, activeSlot{
			torrentID: t.ID,
			name:      t.Name,
			leechers:  leechers,
			indexerID: t.IndexerID,
		})
	}

	// Sort active slots by leechers ascending (weakest first)
	sort.Slice(activeSlots, func(i, j int) bool {
		return activeSlots[i].leechers < activeSlots[j].leechers
	})

	// Count active slots per indexer
	slotsPerIndexer := make(map[int64]int)
	for _, slot := range activeSlots {
		if slot.indexerID != nil {
			slotsPerIndexer[*slot.indexerID]++
		}
	}

	// getMaxSlots returns the max slots for an indexer (per-indexer override or global default)
	getMaxSlots := func(indexerID int64) int {
		if limit, ok := indexerMaxSlots[indexerID]; ok {
			return limit
		}
		return f.maxSlots
	}

	// getSeedTimeMs returns the seed time in ms for an indexer (per-indexer override or 72h default)
	getSeedTimeMs := func(indexerID int64) int64 {
		if h, ok := indexerSeedTimeH[indexerID]; ok {
			return int64(h) * 3600000
		}
		return 259200000 // 72h default
	}

	log.Printf("[prowlarr] %d active torrents", len(activeSlots))

	added := 0
	replaced := 0
	for _, result := range results {
		if result.Leechers <= 0 {
			break
		}
		if result.DownloadURL == "" {
			continue
		}

		indexerID := int64(result.IndexerID)
		limit := getMaxSlots(indexerID)
		used := slotsPerIndexer[indexerID]

		// Check if we have room for this indexer or can replace a weaker torrent
		canAdd := used < limit
		var replaceTarget *activeSlot
		var replaceIdx int
		if !canAdd {
			// Find weakest torrent from the same indexer
			for i := range activeSlots {
				slot := &activeSlots[i]
				if slot.indexerID != nil && *slot.indexerID == indexerID && result.Leechers > slot.leechers {
					replaceTarget = slot
					replaceIdx = i
					break // slots are sorted weakest first, so first match is weakest
				}
			}
		}

		if !canAdd && replaceTarget == nil {
			continue
		}

		// Check seed time guard before replacing
		if replaceTarget != nil {
			remaining := f.manager.SeedTimeRemainingMs(replaceTarget.torrentID)
			if remaining != nil && *remaining > 0 {
				remainingH := *remaining / 3600000
				configuredH := getSeedTimeMs(indexerID) / 3600000
				log.Printf("[prowlarr] skipping replacement of #%d %q: %dh remaining of %dh seed time",
					replaceTarget.torrentID, replaceTarget.name, remainingH, configuredH)
				continue
			}
		}

		// Download the .torrent
		torrentData, err := f.client.DownloadTorrent(result.DownloadURL)
		if err != nil {
			log.Printf("[prowlarr] download error for %q: %v", result.Title, err)
			continue
		}

		// If replacing, remove the weakest torrent first
		if replaceTarget != nil {
			log.Printf("[prowlarr] replacing #%d %q (leechers=%d) with %q (leechers=%d)",
				replaceTarget.torrentID, replaceTarget.name, replaceTarget.leechers,
				result.Title, result.Leechers)
			if err := f.manager.RemoveTorrent(replaceTarget.torrentID); err != nil {
				log.Printf("[prowlarr] remove error for #%d: %v", replaceTarget.torrentID, err)
				continue
			}
			activeSlots = append(activeSlots[:replaceIdx], activeSlots[replaceIdx+1:]...)
			slotsPerIndexer[indexerID]--
			replaced++
		}

		seedTimeMs := getSeedTimeMs(indexerID)
		id, err := f.manager.AddTorrent(torrentData, f.defaultProfile, true, &indexerID, &seedTimeMs)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint") {
				log.Printf("[prowlarr] skipping %q: already exists", result.Title)
			} else {
				log.Printf("[prowlarr] add error for %q: %v", result.Title, err)
			}
			continue
		}

		// Add new torrent to active slots
		activeSlots = append(activeSlots, activeSlot{
			torrentID: id,
			name:      result.Title,
			leechers:  result.Leechers,
			indexerID: &indexerID,
		})
		sort.Slice(activeSlots, func(i, j int) bool {
			return activeSlots[i].leechers < activeSlots[j].leechers
		})
		slotsPerIndexer[indexerID]++

		log.Printf("[prowlarr] added #%d: %s (leechers=%d, seeders=%d, indexer=%d)", id, result.Title, result.Leechers, result.Seeders, indexerID)
		added++
	}

	log.Printf("[prowlarr] fetch complete: %d results, %d added, %d replaced", len(results), added, replaced)
}
