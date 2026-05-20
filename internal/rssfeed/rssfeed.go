// Package rssfeed manages RSS/Atom feed polling and pushes updates to WebSocket clients.
//
// Each block on a panel can configure one or more feed URLs with a refresh interval
// and maximum entry count. The server fetches feeds using gofeed, merges entries from
// all configured sources, deduplicates by GUID, sorts by publication date, and pushes
// updates to connected clients via targeted WebSocket messages.
//
// Per-client seen-entry tracking determines which entries are "new". The newest entry
// (most recent by publication date) always remains marked as new until an even newer
// entry arrives. Older entries are marked as seen after their first delivery, so the
// "new" highlight persists across polling cycles until superseded. This allows each
// client to independently track which entries it has acknowledged.
//
// Feed URLs support an optional display label using the format "URL|Label". If no label
// is provided, the feed's own title from the RSS metadata is used. The label is included
// in each entry's feed_label field and displayed on the client alongside the date.
//
// Clicking an entry on the client opens the URL either on the host (via an
// "open-url" WebSocket message, handled by xdg-open on Linux or start on Windows)
// or directly in the client browser via window.open, controlled by the block's
// open_url_location setting ("host" or "client", default "host").
package rssfeed

import (
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

// FeedEntry represents a single RSS/Atom feed entry pushed to clients.
type FeedEntry struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Published   string `json:"published"`
	Description string `json:"description"`
	FeedLabel   string `json:"feed_label"`
}

// FeedSource holds a feed URL and its optional display label.
type FeedSource struct {
	URL   string
	Label string
}

// FeedConfig holds the polling configuration for a single RSS block.
type FeedConfig struct {
	BlockID         string
	FeedSources     []FeedSource
	RefreshInterval int // seconds between polls
	MaxEntries      int // maximum entries to keep per block
}

// EntryWithNew wraps a FeedEntry with a per-client is_new flag for JSON serialization.
type EntryWithNew struct {
	FeedEntry
	IsNew bool `json:"is_new"`
}

// Manager handles RSS feed polling and targeted WebSocket broadcasting.
//
// It maintains a map of block configurations, fetched entries, and per-client
// seen-entry tracking. When feeds are polled, entries are merged from all sources,
// deduplicated by GUID, sorted by publication date (newest first), and limited
// to MaxEntries. The newest entry always remains marked as new (is_new=true) until
// a newer entry arrives. Older entries are marked as seen after first delivery.
// Each client receives entries with an is_new flag based on its own seen-entry history.
type Manager struct {
	mu            sync.RWMutex
	configs       map[string]FeedConfig
	entries       map[string][]FeedEntry
	seenPerClient map[string]map[uint64]map[string]bool // blockID → client ID → set of seen GUIDs
	stopCh        chan struct{}
	nextClientID  uint64
	broadcast     func(clientID uint64, blockID string, entries []EntryWithNew)
}

// New creates and initializes the RSS feed manager.
//
// The broadcast function is called when new entries are available for a specific
// client. It receives the client ID, block ID, and the list of entries with
// per-client is_new flags.
func New(broadcast func(clientID uint64, blockID string, entries []EntryWithNew)) *Manager {
	return &Manager{
		configs:       make(map[string]FeedConfig),
		entries:       make(map[string][]FeedEntry),
		seenPerClient: make(map[string]map[uint64]map[string]bool),
		stopCh:        make(chan struct{}),
		broadcast:     broadcast,
	}
}

// parseFeedSources parses lines of "URL|Label" or "URL" into FeedSource slices.
// Lines that are empty or contain only whitespace are skipped. The label portion
// is optional; if omitted, the feed's own title will be used at fetch time.
func parseFeedSources(lines []string) []FeedSource {
	var sources []FeedSource
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		url := strings.TrimSpace(parts[0])
		if url == "" {
			continue
		}
		label := ""
		if len(parts) > 1 {
			label = strings.TrimSpace(parts[1])
		}
		sources = append(sources, FeedSource{URL: url, Label: label})
	}
	return sources
}

// Configure sets up feed polling for a block and registers the requesting client
// for per-client seen-entry tracking.
//
// It parses the feed URLs (supporting "URL|Label" format), validates the
// configuration, registers the client in seenPerClient so it receives updates,
// and starts an immediate fetch. Subsequent fetches are scheduled at the
// configured refresh interval. The minimum refresh interval is 10 seconds;
// values below this are clamped.
func (m *Manager) Configure(blockID string, clientID uint64, feedURLs []string, refreshIntervalSec int, maxEntries int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sources := parseFeedSources(feedURLs)

	if len(sources) == 0 {
		slog.Warn("RSS configure: no valid URLs", "block_id", blockID)
		return
	}

	if refreshIntervalSec < 10 {
		refreshIntervalSec = 10
	}
	if maxEntries < 1 {
		maxEntries = 20
	}

	m.configs[blockID] = FeedConfig{
		BlockID:         blockID,
		FeedSources:     sources,
		RefreshInterval: refreshIntervalSec,
		MaxEntries:      maxEntries,
	}

	// Initialize seen tracking for this block and register the requesting client
	if m.seenPerClient[blockID] == nil {
		m.seenPerClient[blockID] = make(map[uint64]map[string]bool)
	}
	m.seenPerClient[blockID][clientID] = make(map[string]bool)

	// Ensure all other existing clients also have an entry for this block
	for existingBlockID, clients := range m.seenPerClient {
		if existingBlockID == blockID {
			continue
		}
		for cid := range clients {
			if _, exists := m.seenPerClient[blockID][cid]; !exists {
				m.seenPerClient[blockID][cid] = make(map[string]bool)
			}
		}
	}

	slog.Info("RSS feed configured", "block_id", blockID, "sources", len(sources), "interval", refreshIntervalSec, "max_entries", maxEntries)

	// Do an immediate fetch
	go m.fetchFeed(blockID)
}

// Remove stops feed polling and clears all data for a block.
func (m *Manager) Remove(blockID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.configs, blockID)
	delete(m.entries, blockID)
	delete(m.seenPerClient, blockID)

	slog.Info("RSS feed removed", "block_id", blockID)
}

// GetEntries returns the current entries for a block.
func (m *Manager) GetEntries(blockID string) []FeedEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.entries[blockID]
}

// RegisterClient adds a client for per-client seen-entry tracking.
//
// Called when a WebSocket connection is established. The client will receive
// entries with is_new=true for all entries it hasn't seen before. The newest
// entry in each block always remains marked as new until superseded.
func (m *Manager) RegisterClient(clientID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for blockID := range m.configs {
		if m.seenPerClient[blockID] == nil {
			m.seenPerClient[blockID] = make(map[uint64]map[string]bool)
		}
		m.seenPerClient[blockID][clientID] = make(map[string]bool)
	}
}

// UnregisterClient removes a client from seen-entry tracking.
//
// Called when a WebSocket connection is closed. All per-client seen-entry
// data for this client is removed across all blocks.
func (m *Manager) UnregisterClient(clientID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for blockID := range m.seenPerClient {
		delete(m.seenPerClient[blockID], clientID)
	}
}

// Start begins periodic feed polling for all configured blocks.
func (m *Manager) Start() {
	slog.Info("RSS feed manager started")
}

// Close stops all feed polling goroutines.
func (m *Manager) Close() {
	close(m.stopCh)
	slog.Info("RSS feed manager stopped")
}

// fetchFeed fetches all configured feeds for a block, merges and deduplicates
// entries, and broadcasts updates to all connected clients.
//
// For each feed source, entries are fetched using gofeed. The feed label is
// set to the source's configured label, or falls back to the feed's own title.
// Entries are sorted by publication date (newest first), deduplicated by GUID,
// and limited to MaxEntries. The newest entry (index 0) always remains marked
// as new (is_new=true) until a newer entry arrives. Older entries are marked
// as seen after their first delivery, so the "new" highlight persists across
// polling cycles until superseded. Each client receives entries with an is_new
// flag based on its own seen-entry history.
func (m *Manager) fetchFeed(blockID string) {
	m.mu.RLock()
	config, exists := m.configs[blockID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	var allEntries []FeedEntry

	for _, source := range config.FeedSources {
		feed, err := fp.ParseURL(source.URL)
		if err != nil {
			slog.Warn("RSS feed fetch failed", "url", source.URL, "block_id", blockID, "error", err)
			continue
		}

		feedLabel := source.Label
		if feedLabel == "" {
			feedLabel = feed.Title
		}

		for _, item := range feed.Items {
			entry := FeedEntry{
				Title:     item.Title,
				Link:      item.Link,
				FeedLabel: feedLabel,
			}

			if item.GUID != "" {
				entry.GUID = item.GUID
			} else if item.Link != "" {
				entry.GUID = item.Link
			} else {
				entry.GUID = item.Title + item.Published
			}

			if item.PublishedParsed != nil {
				entry.Published = item.PublishedParsed.Format(time.RFC3339)
			}

			if item.Description != "" {
				entry.Description = item.Description
			} else if item.Content != "" {
				entry.Description = item.Content
			}

			allEntries = append(allEntries, entry)
		}
	}

	// Sort by published date (newest first)
	sort.Slice(allEntries, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, allEntries[i].Published)
		tj, _ := time.Parse(time.RFC3339, allEntries[j].Published)
		return ti.After(tj)
	})

	// Deduplicate by GUID
	seen := make(map[string]bool)
	var deduped []FeedEntry
	for _, e := range allEntries {
		if !seen[e.GUID] {
			seen[e.GUID] = true
			deduped = append(deduped, e)
		}
	}

	// Limit to max entries
	if len(deduped) > config.MaxEntries {
		deduped = deduped[:config.MaxEntries]
	}

	// Store entries
	m.mu.Lock()
	m.entries[blockID] = deduped
	m.mu.Unlock()

	// Build update payload with per-client is_new flags
	m.mu.RLock()
	seenMap := m.seenPerClient[blockID]
	m.mu.RUnlock()

	for clientID, clientSeen := range seenMap {
		// Build entries with is_new flag
		var entriesWithNew []EntryWithNew
		for i, e := range deduped {
			// Only mark as seen (not new) if this is NOT the newest entry
			// The newest entry (index 0) stays "new" until a newer one arrives
			isNew := !clientSeen[e.GUID]
			entriesWithNew = append(entriesWithNew, EntryWithNew{FeedEntry: e, IsNew: isNew})
			// Only mark as seen if this is not the newest entry
			if i > 0 {
				clientSeen[e.GUID] = true
			}
		}

		// Broadcast to this specific client
		if m.broadcast != nil {
			m.broadcast(clientID, blockID, entriesWithNew)
		}
	}

	// Schedule next fetch
	go m.scheduleNextFetch(blockID)
}

// scheduleNextFetch waits for the refresh interval then fetches again.
// It exits early if the manager is closed.
func (m *Manager) scheduleNextFetch(blockID string) {
	m.mu.RLock()
	config, exists := m.configs[blockID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	ticker := time.NewTicker(time.Duration(config.RefreshInterval) * time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		m.fetchFeed(blockID)
	case <-m.stopCh:
		return
	}
}

// fp is the shared gofeed parser instance.
var fp = gofeed.NewParser()
