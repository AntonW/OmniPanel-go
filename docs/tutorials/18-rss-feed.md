# Chapter 18: RSS Feed Integration

This chapter covers the RSS feed block — how the server polls feeds, pushes updates to clients via WebSocket, and opens URLs on the host system when a user clicks an entry.

## Overview

The RSS feed block allows users to display live entries from one or more RSS/Atom feeds on their panel. Unlike the DataBus (which broadcasts to all clients), RSS updates are pushed individually to each client with per-client "new entry" tracking — meaning each client independently determines which entries are new and highlights them accordingly.

### Key Features

- **Server-side polling**: The server fetches feeds, avoiding CORS issues and allowing access to internal networks
- **Per-client seen tracking**: Each client maintains its own set of seen entry GUIDs
- **Feed labels**: Each feed URL can have an optional display label (`URL|Label` format)
- **Configurable URL opening**: Clicking an entry opens the URL on the host or client device based on the `open_url_location` setting

## Architecture

```
┌─────────────┐     rss-configure     ┌──────────────┐
│  Browser     │ ────────────────────▶ │  RSS Manager │
│  (Client)    │                       │  (Server)    │
│              │ ◀──────────────────── │              │
│              │    rss-update         │              │
│              │   (per-client)        │              │
└─────────────┘                       └──────┬───────┘
                                             │
                                    ┌────────▼────────┐
                                    │  gofeed Parser  │
                                    │  (RSS/Atom)     │
                                    └────────┬────────┘
                                             │
                                    ┌────────▼────────┐
                                    │  Feed Sources   │
                                    │  (HTTP URLs)    │
                                    └─────────────────┘
```

## The RSS Manager

The `rssfeed` package lives in `internal/rssfeed/rssfeed.go`. It manages feed polling, entry merging, and per-client update delivery.

### Manager Struct

```go
// internal/rssfeed/rssfeed.go:48-56
type Manager struct {
    mu            sync.RWMutex
    configs       map[string]FeedConfig
    entries       map[string][]FeedEntry
    seenPerClient map[string]map[uint64]map[string]bool // blockID → client ID → set of seen GUIDs
    stopCh        chan struct{}
    nextClientID  uint64
    broadcast     func(clientID uint64, blockID string, entries []EntryWithNew)
}
```

> **Concept: Callback Injection**
> The `broadcast` field is a function passed to `New()` during construction. This is a common Go pattern for dependency injection — instead of the Manager importing the `state` package (which would create a circular import), it receives a function that knows how to send messages to specific clients. This keeps packages decoupled.

### Feed Configuration

Each block configures its feeds via the `Configure` method. The `clientID` parameter is passed so the manager can register the requesting client for per-client seen-entry tracking:

```go
// internal/rssfeed/rssfeed.go:117-170
func (m *Manager) Configure(blockID string, clientID uint64, feedURLs []string, refreshIntervalSec int, maxEntries int) {
    m.mu.Lock()
    defer m.mu.Unlock()

    sources := parseFeedSources(feedURLs)

    if len(sources) == 0 {
        slog.Warn("RSS configure: no valid URLs", "block_id", blockID)
        return
    }
    // ... validation and setup ...

    m.configs[blockID] = FeedConfig{
        BlockID:         blockID,
        FeedSources:     sources,
        RefreshInterval: refreshIntervalSec,
        MaxEntries:      maxEntries,
    }

    // Register the requesting client in seen tracking
    if m.seenPerClient[blockID] == nil {
        m.seenPerClient[blockID] = make(map[uint64]map[string]bool)
    }
    m.seenPerClient[blockID][clientID] = make(map[string]bool)

    // Ensure all other existing clients also have an entry for this block
    // ...

    // Do an immediate fetch
    go m.fetchFeed(blockID)
}
```

> **Key Pattern: Client Registration on Configure**
> The `clientID` is passed directly to `Configure` so the requesting client is immediately registered in `seenPerClient`. Without this, when a client sends `rss-configure` before any blocks are configured, `seenPerClient[blockID]` would be empty and the client would never receive updates — a classic race condition between WebSocket connection and RSS configuration.

### Feed URL Parsing

The `parseFeedSources` function handles the `URL|Label` format:

```go
// internal/rssfeed/rssfeed.go:71-90
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
```

> **Key Pattern: SplitN with limit 2**
> `strings.SplitN(line, "|", 2)` splits the string into at most 2 parts. This allows labels to contain `|` characters — only the first `|` is treated as the separator. For example, `"https://example.com/rss|Tech | News"` correctly parses as URL=`https://example.com/rss` and Label=`Tech | News`.

### Feed Fetching

The `fetchFeed` method does the heavy lifting:

```go
// internal/rssfeed/rssfeed.go:193-296
func (m *Manager) fetchFeed(blockID string) {
    // ... get config ...

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
            // ... GUID, published, description ...
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
    // ...

    // Limit to max entries
    // ...

    // Broadcast to each client with per-client is_new flags
    for clientID, clientSeen := range seenMap {
        var entriesWithNew []EntryWithNew
        for _, e := range deduped {
            isNew := !clientSeen[e.GUID]
            entriesWithNew = append(entriesWithNew, EntryWithNew{FeedEntry: e, IsNew: isNew})
            clientSeen[e.GUID] = true
        }
        if m.broadcast != nil {
            m.broadcast(clientID, blockID, entriesWithNew)
        }
    }

    // Schedule next fetch
    go m.scheduleNextFetch(blockID)
}
```

> **Key Pattern: Per-Client State Without Client Reference**
> The Manager doesn't hold references to WebSocket connections. Instead, it tracks clients by their `uint64` ID and calls a `broadcast` callback. The `state` package owns the actual client channels and handles the delivery. This keeps the RSS package testable and decoupled from WebSocket internals.

## WebSocket Integration

### Client Registration

When a WebSocket connection is established, the client is registered with both the broadcast system and the RSS manager:

```go
// internal/routes/router.go:84-91
func handleWS(c *ws.Conn, s *state.AppState) {
    ch := make(chan []byte, 256)
    clientID := s.RegisterClient(ch)

    if s.RSSManager != nil {
        s.RSSManager.RegisterClient(clientID)
    }
    // ...
}
```

### Message Handlers

Two new WebSocket message types are handled:

**rss-configure** — sent by the client when an RSS block is rendered:

```go
// internal/websocket/handler.go:624-657
func handleRSSConfigure(s *state.AppState, clientID uint64, data json.RawMessage) {
    var blockID string
    var feedURLs []string
    refreshInterval := int(parseUintField(data, "refresh_interval"))
    maxEntries := int(parseUintField(data, "max_entries"))

    if s.RSSManager != nil {
        s.RSSManager.Configure(blockID, clientID, feedURLs, refreshInterval, maxEntries)
    }
}
```

> **Key Pattern: Passing ClientID Through the Message Pipeline**
> The `clientID` is assigned when the WebSocket connection is established (in `handleWS`) and passed through `HandleMessage` → `handleRSSConfigure` → `RSSManager.Configure`. This avoids the need for the RSS manager to look up the client from the connection, keeping the design clean and testable.

**open-url** — sent when a user clicks an RSS entry:

```go
// internal/websocket/handler.go:657-669
func handleOpenURL(s *state.AppState, data json.RawMessage) {
    url := parseStringField(data, "url")
    if url == "" {
        slog.Warn("open-url missing url")
        return
    }

    slog.Info("Opening URL on host", "url", url)
    if err := rssfeed.OpenURL(url); err != nil {
        slog.Warn("Failed to open URL", "url", url, "error", err)
    }
}
```

### Platform-Specific URL Opening

The `OpenURL` function uses build tags for platform-specific implementations:

```go
// internal/rssfeed/openurl_linux.go
//go:build linux

package rssfeed

import "os/exec"

func OpenURL(url string) error {
    return exec.Command("xdg-open", url).Start()
}
```

```go
// internal/rssfeed/openurl_windows.go
//go:build windows

package rssfeed

import "os/exec"

func OpenURL(url string) error {
    return exec.Command("cmd", "/c", "start", url).Start()
}
```

> **Concept: Build Tags**
> Go's `//go:build` directive controls which files are compiled for which platforms. `//go:build linux` means the file is only included when building for Linux. `//go:build !linux && !windows` means "not Linux and not Windows" — a fallback stub for unsupported platforms. This is the same pattern used for virtual input devices (`linux.go`, `windows.go`, `stub.go`).

## Client-Side JavaScript

### Block Initialization

When an RSS block is rendered, `initRSSFeed` sends the configuration to the server. If the WebSocket is not yet open, it retries every 100ms until connected — this prevents the configuration message from being silently dropped on first panel load:

```javascript
// static/client/client.js:2002-2041
function initRSSFeed(blockWrapper) {
    const feedUrlsSetting = blockWrapper.settings['feed_urls'];
    if (!feedUrlsSetting) return;

    const feedUrls = feedUrlsSetting.split('\n').map(u => u.trim()).filter(u => u);
    if (feedUrls.length === 0) return;

    const refreshInterval = parseInt(blockWrapper.settings['refresh_interval']) || 60;
    const maxEntries = parseInt(blockWrapper.settings['max_entries']) || 20;

    const config = {
        type: 'rss-configure',
        data: {
            block_id: blockWrapper.id,
            feed_urls: feedUrls,
            refresh_interval: refreshInterval,
            max_entries: maxEntries
        }
    };

    function trySend() {
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify(config));
        } else {
            setTimeout(trySend, 100);
        }
    }

    trySend();
}
```

> **Key Pattern: Retry Until Connected**
> Panel rendering and WebSocket connection happen in parallel. If `initRSSFeed()` runs before the socket is open, a simple `if` check would silently drop the message. The `trySend` inner function uses `setTimeout` to retry every 100ms until `socket.readyState === WebSocket.OPEN`. This is a lightweight alternative to queuing messages or blocking panel rendering.

### Handling Updates

When the server pushes an `rss-update` message, `handleRSSUpdate` renders the entries:

```javascript
// static/client/client.js:1885-1949
function handleRSSUpdate(data) {
    const blockId = data.block_id;
    const entries = data.entries || [];
    const block = document.getElementById(blockId);
    if (!block) return;

    const entriesList = block.querySelector('.rss-entries-list');
    if (!entriesList) return;

    const settings = block.settings || {};
    const showDate = settings['show_date'] !== 'false';
    const showFeedLabel = settings['show_feed_label'] !== 'false';
    const showDescription = settings['show_description'] !== 'false';
    const descMaxLength = parseInt(settings['description_max_length']) || 150;

    if (!rssSeenEntries[blockId]) {
        rssSeenEntries[blockId] = new Set();
    }

    const currentGUIDs = new Set();
    let html = '';

    if (entries.length === 0) {
        html = '<div class="rss-empty-state">No entries found</div>';
    } else {
        entries.forEach(entry => {
            const isNew = entry.is_new && !rssSeenEntries[blockId].has(entry.guid);
            currentGUIDs.add(entry.guid);

            let metaHtml = '';
            const metaParts = [];
            if (showFeedLabel && entry.feed_label) {
                metaParts.push(`<span class="rss-entry-feed-label">${escapeHtml(entry.feed_label)}</span>`);
            }
            if (showDate && entry.published) {
                const date = new Date(entry.published);
                metaParts.push(`<span class="rss-entry-date">${date.toLocaleDateString()} ${date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>`);
            }
            if (metaParts.length > 0) {
                metaHtml = `<div class="rss-entry-meta">${metaParts.join('')}</div>`;
            }
            // ... description rendering ...

            html += `
                <div class="rss-entry ${isNew ? 'new-entry' : ''}" data-link="${escapeHtml(entry.link)}">
                    <div class="rss-entry-title">${escapeHtml(entry.title || 'Untitled')}</div>
                    ${metaHtml}
                    ${descHtml}
                </div>
            `;
        });
    }

    entriesList.innerHTML = html;
    rssSeenEntries[blockId] = currentGUIDs;
}
```

> **Key Pattern: Event Delegation for Dynamic Content**
> RSS entries are created dynamically via `innerHTML`, so individual event listeners can't be attached during `enableInputs()`. Instead, a single `document.addEventListener('click', ...)` at the document level uses `e.target.closest('.rss-entry')` to catch clicks on any entry, even ones created after the listener was registered. This is more efficient and works regardless of when entries are rendered.

### Click Handler

Clicking an RSS entry checks the block's `open_url_location` setting:

```javascript
// static/client/client.js:1679-1720
document.querySelectorAll('.rss-entry').forEach(entry => {
    entry.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        const url = entry.getAttribute('data-link');
        if (!url || !socket || socket.readyState !== WebSocket.OPEN) return;
        const block = entry.closest('.loaded-block');
        const openLocation = block?.settings?.['open_url_location'] || 'host';
        if (openLocation === 'client') {
            window.open(url, '_blank');
        } else {
            socket.send(JSON.stringify({
                type: 'open-url',
                data: { url: url }
            }));
        }
    });
});
```

> **Concept: Optional Chaining (`?.`)**
> `block?.settings?.['open_url_location']` safely navigates nested properties without throwing if `block` is null or `settings` is undefined. If any part of the chain is nullish, the expression short-circuits to `undefined`. Combined with `|| 'host'`, this provides a safe default — if the setting hasn't been configured (e.g., on an older panel), URLs still open on the host.

When `open_url_location` is `"host"` (default), an `open-url` WebSocket message is sent to the server, which opens the URL via `xdg-open` (Linux) or `start` (Windows). When set to `"client"`, the URL is opened directly in the panel browser via `window.open()`.

> **Key Pattern: Dual Click Handlers**
> Two handlers are registered: one via `querySelectorAll('.rss-entry')` for entries present when `enableInputs()` runs, and a delegated `document.addEventListener('click', ...)` for entries created later via `innerHTML`. This ensures clicks work regardless of when entries are rendered — a common pattern when DOM content is generated dynamically after page load.

## Editor Integration

The properties panel renders an "RSS Feeds" section with an "+ Add Feed" button:

```javascript
// static/editor/js/properties-panel.js:729-763
addFeed(block) {
    const url = prompt('Enter RSS/Atom feed URL:');
    if (!url) return;

    const trimmedUrl = url.trim();
    if (!trimmedUrl) return;

    const existingFeeds = this.parseFeedEntries(block.settings['feed_urls'] || '');
    if (existingFeeds.some(f => f.url === trimmedUrl)) {
        alert('This feed URL already exists.');
        return;
    }

    const label = prompt('Enter a display label for this feed (optional, leave empty to use feed title):');

    const feedUrlsRaw = block.settings['feed_urls'] || '';
    const lines = feedUrlsRaw.split('\n').filter(l => l.trim());
    const newLine = label && label.trim() ? `${trimmedUrl}|${label.trim()}` : trimmedUrl;
    lines.push(newLine);
    block.settings['feed_urls'] = lines.join('\n');

    this.state.pushState({
        type: 'setting-changed',
        blockId: block.id,
        key: 'feed_urls',
        oldValue: feedUrlsRaw,
        newValue: block.settings['feed_urls']
    });

    this.showProperties(block);
    if (window.editor && window.editor.blockRenderer) {
        window.editor.blockRenderer.renderBlock(block);
    }
}
```

## Data Flow Summary

1. **Block rendered** → `initRSSFeed()` sends `rss-configure` via WebSocket (retries every 100ms if socket not yet open)
2. **Server receives** → `handleRSSConfigure(clientID, ...)` → `RSSManager.Configure(blockID, clientID, ...)`
3. **Client registered** → requesting client added to `seenPerClient[blockID]` for per-client tracking
4. **Immediate fetch** → `fetchFeed()` parses all feed URLs with gofeed
5. **Entries merged** → sorted by date, deduplicated by GUID, limited to max
6. **Per-client push** → `broadcast(clientID, blockID, entriesWithNew)` → `broadcastToClient()` → client channel
7. **Client receives** → `handleRSSUpdate()` renders entries, marks new ones
8. **User clicks entry** → checks `open_url_location`: `"host"` sends `open-url` WebSocket → `handleOpenURL()` → `xdg-open` / `start`; `"client"` → `window.open()` in panel browser
