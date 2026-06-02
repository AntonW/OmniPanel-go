# Chapter 17: Media Player Integration

## What This Package Does

The `mpris` package provides the media watcher backend used by the Media Player block. On Linux it uses MPRIS over D-Bus; on Windows it uses SMTC (System Media Transport Controls). Both implementations publish the same `mpris_*` compatibility keys to the `DataBus` so the frontend logic stays the same.

> **Platform Note:** Linux uses `github.com/godbus/dbus/v5` (`internal/mpris/mpris.go`), while Windows uses WinRT/COM SMTC APIs (`internal/mpris/watcher_windows.go`). The REST API is platform-neutral (`/api/media/*`). DataBus keys use the `mpris_*` prefix to keep frontend code consistent with the media watcher terminology.

## Key Concepts

### What Is MPRIS? (Linux)

MPRIS (Media Player Remote Interfacing Specification) is a D-Bus interface standard used by Linux media players. It allows external programs to:
- Discover what media players are running
- Read metadata (title, artist, album, cover art URL)
- Read playback state (playing, paused, stopped, position, volume)
- Send control commands (play, pause, next, previous, stop, volume)

> **Concept (D-Bus):** D-Bus is an inter-process communication system on Linux. Think of it like a message bus where applications can publish services and other applications can connect to them. The "session bus" is per-user (your desktop session), while the "system bus" is system-wide. MPRIS players register themselves on the session bus with names like `org.mpris.MediaPlayer2.spotify`.

### What Is SMTC? (Windows)

SMTC (System Media Transport Controls) is the Windows equivalent of MPRIS. It's a WinRT API that provides access to media player metadata and controls on Windows 10/11. Any SMTC-registered player (Spotify, Firefox, VLC, Edge, etc.) can be monitored and controlled.

### How the Watcher Works

The media watcher operates in two phases:

1. **Initial discovery** — When the server starts, it discovers already-running media sessions
2. **Polling loop** — Every N milliseconds (configurable), it queries each discovered session for state updates

On **Linux**, it additionally uses **signal monitoring** to detect when new players appear or existing ones disappear via D-Bus `NameOwnerChanged` signals.

## The Data Structures

```go
// internal/mpris/types.go
type PlayerState struct {
    Identity       string  // Human-readable name (e.g., "Spotify", "VLC")
    PlayerName     string  // D-Bus service suffix (Linux) or AppUserModelId (Windows)
    PlaybackStatus string  // "Playing", "Paused", or "Stopped"
    Title          string  // Track title
    Artist         string  // Track artist (joined with ", " if multiple)
    Album          string  // Album name
    ArtURL         string  // Cover art URL (may be file:// on Linux)
    Length         int64   // Track duration in microseconds
    Position       int64   // Current playback position in microseconds
    Volume         float64 // Volume 0.0 (muted) to 1.0 (max)
    CanPlay        bool    // Whether Play method is supported
    CanPause       bool    // Whether Pause method is supported
    CanGoNext      bool    // Whether Next method is supported
    CanGoPrevious  bool    // Whether Previous method is supported
    CanControl     bool    // Whether the player accepts control commands
}
```

```go
type Watcher struct {
    mu             sync.RWMutex
    config         *config.MediaPlayerConfig
    databus        *databus.DataBus
    logger         *slog.Logger
    conn           *dbus.Conn           // Linux only
    players        map[string]*PlayerState
    selectedPlayer string // Name suffix of the active player
    stopCh         chan struct{}
    running        bool
}
```

The `Watcher` holds references to discovered players, a `selectedPlayer` field that tracks which player's state is published to the DataBus, and a stop channel for clean shutdown. The `sync.RWMutex` protects concurrent access to the players map.

## Initialization

```go
// internal/mpris/mpris.go (Linux) and watcher_windows.go (Windows)
func New(cfg *config.MediaPlayerConfig, db *databus.DataBus, logger *slog.Logger) *Watcher {
    if cfg == nil || !cfg.Enabled {
        return nil
    }
    // ... initialize Watcher
    return w
}
```

`New` returns `nil` if media integration is disabled in config. The caller (`state.New`) checks for `nil` before calling `Start()`.

```go
// internal/state/state.go
mediaWatcher := mpris.New(&cfg.MediaPlayer, db, slog.Default())
if mediaWatcher != nil {
    if err := mediaWatcher.Start(); err != nil {
        slog.Warn("Media watcher failed to start", "error", err)
    }
}
app.MPRISWatcher = mediaWatcher
```

## Starting the Watcher

### Linux (MPRIS)

```go
// internal/mpris/mpris.go
func (w *Watcher) Start() error {
    w.mu.Lock()

    if w.running {
        w.mu.Unlock()
        return nil
    }

    conn, err := dbus.ConnectSessionBus()
    if err != nil {
        w.mu.Unlock()
        return fmt.Errorf("mpris: connect session bus: %w", err)
    }

    w.conn = conn
    w.running = true
    w.mu.Unlock()

    w.discoverExistingPlayers()  // Find players already running

    go w.runPollLoop()           // Periodic state polling
    go w.watchPlayerListChanges() // D-Bus signal monitoring

    return nil
}
```

> **Key Pattern (Mutex):** The mutex is explicitly unlocked before calling `discoverExistingPlayers()` and launching goroutines. This is critical because these operations may acquire the same mutex. Using `defer w.mu.Unlock()` here would cause a deadlock since `sync.Mutex` is not reentrant.

### Windows (SMTC)

```go
// internal/mpris/watcher_windows.go
func (w *Watcher) Start() error {
    w.mu.Lock()
    if w.running {
        w.mu.Unlock()
        return nil
    }
    w.running = true
    w.mu.Unlock()

    if err := ole.RoInitialize(1); err != nil {
        // WinRT may already be initialized; some errors are benign
    }

    if err := os.MkdirAll(w.tempDir, 0o755); err != nil {
        w.logger.Warn("SMTC: cannot create temp dir for cover art", "path", w.tempDir, "error", err)
    }

    go w.runPollLoop()
    return nil
}
```

## Player Discovery

### Linux (MPRIS)

```go
func (w *Watcher) discoverExistingPlayers() {
    var names []string
    err := w.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
    if err != nil {
        return
    }

    for _, name := range names {
        if strings.HasPrefix(name, mprisPrefix) {  // "org.mpris.MediaPlayer2."
            playerName := strings.TrimPrefix(name, mprisPrefix)
            w.discoverPlayer(playerName)
        }
    }
}
```

When a player is discovered, it is automatically selected if no player has been selected yet. After updating the player's state, `publishPlayersList()` is called to broadcast the full player list to the DataBus.

### Windows (SMTC)

```go
// internal/mpris/watcher_windows.go
func (w *Watcher) pollSessions() {
    asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
    if err != nil {
        return
    }

    manager, err := awaitAsync(asyncOp)
    if err != nil {
        return
    }
    defer manager.Release()

    sessionsVec, err := manager.GetSessions()
    // ... iterate through sessions and build PlayerState for each
}
```

Windows actively polls the SMTC session manager during each poll cycle. There's no real-time signal monitoring like on Linux.

## Signal Monitoring (Linux Only)

```go
func (w *Watcher) watchPlayerListChanges() {
    ch := make(chan *dbus.Signal, 10)
    w.conn.Signal(ch)

    w.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
        "type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'")

    for {
        select {
        case <-w.stopCh:
            w.conn.RemoveSignal(ch)
            return
        case signal := <-ch:
            // Parse signal.Body[0] (name) and Body[2] (new owner)
            // If newOwner == "" → player disconnected → removePlayer()
            // If newOwner != "" → player appeared → discoverPlayer()
        }
    }
}
```

> **Concept (D-Bus Signals):** D-Bus signals are asynchronous notifications. `NameOwnerChanged` fires whenever a D-Bus service appears or disappears. By subscribing to this signal, the watcher detects new media players in real-time without polling.

When a player disconnects, `removePlayer()` removes it from the map. If the disconnected player was the selected one, the watcher automatically picks another available player.

## Polling Player State

```go
func (w *Watcher) runPollLoop() {
    interval := time.Duration(w.config.PollInterval) * time.Millisecond
    if interval < 500*time.Millisecond {
        interval = 1 * time.Second  // Enforce minimum
    }

    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-w.stopCh:
            return
        case <-ticker.C:
            w.pollPlayers()  // or w.pollSessions() on Windows
        }
    }
}
```

Each poll queries the properties for every known player/session.

> **Key Pattern (D-Bus Variants):** D-Bus properties are returned as `dbus.Variant` types, which wrap any Go value. You must call `.Value()` and then type-assert to the expected type. The type assertion returns `(value, ok)` — if the property has a different type than expected, `ok` is `false` and the default zero value is used.

## Publishing to DataBus

```go
func (w *Watcher) publishToDataBus(state *PlayerState) {
    // Only publish if this state belongs to the selected player
    w.mu.RLock()
    if w.selectedPlayer != state.PlayerName {
        w.mu.RUnlock()
        return
    }
    w.mu.RUnlock()

    keyPrefix := "mpris_"

    w.databus.SetSource(keyPrefix+"player_name", state.PlayerName, "", "mpris")
    w.databus.SetSource(keyPrefix+"identity", state.Identity, "", "mpris")
    // ... title, artist, album, cover_url, progress, volume, capabilities
}
```

> **Key Pattern (Selection Filtering):** `publishToDataBus()` checks `selectedPlayer` before writing to the DataBus. This ensures the frontend only sees data from the player the user has selected, even though the watcher polls all connected players in the background.

### Player List Publishing

In addition to per-player state, the watcher publishes the full list of available players:

```go
func (w *Watcher) publishPlayersList() {
    type PlayerInfo struct {
        Name     string `json:"name"`
        Identity string `json:"identity"`
    }
    players := make([]PlayerInfo, 0, len(w.players))
    for name, state := range w.players {
        players = append(players, PlayerInfo{Name: name, Identity: state.Identity})
    }
    data, _ := json.Marshal(players)
    w.databus.SetSource("mpris_available_players", string(data), "", "mpris")
}
```

This is called whenever a player appears or disappears. The frontend parses this JSON array and renders source selection tabs when more than one player is available.

### Player Selection

The `SetSelectedPlayer()` method changes which player's state is published:

```go
func (w *Watcher) SetSelectedPlayer(playerName string) error {
    w.mu.Lock()
    if _, exists := w.players[playerName]; !exists {
        w.mu.Unlock()
        return fmt.Errorf("media: player %q not found", playerName)
    }
    oldSelected := w.selectedPlayer
    w.selectedPlayer = playerName
    w.mu.Unlock()

    if w.selectedPlayer != oldSelected {
        w.updatePlayer(w.selectedPlayer)  // Immediately fetch new player's state
    }
    return nil
}
```

> **Key Pattern (Lock Release Before I/O):** The mutex is released before calling `updatePlayer()`, which acquires its own lock. Holding the write lock during `updatePlayer()` would cause a deadlock since `sync.RWMutex` is not reentrant.

## HTTP API Endpoints

Four endpoints in `internal/routes/media.go` (default mode) or `internal/relay/server.go` (serve mode):

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/media/players` | GET | List connected players with their current state (identity, playback status, metadata, volume, capabilities) |
| `/api/media/control` | POST | Send playback command (play, pause, next, etc.) |
| `/api/media/select` | POST | Set the active player. Body: `{ "player": "spotify" }` |
| `/api/media/cover` | GET | Proxy local cover art files for browser access |

### Player Selection

The `/api/media/select` endpoint changes which player's state is published to the DataBus. Control commands (`/api/media/control`) default to the selected player when no `player` field is provided in the request body:

```go
if req.Player == "" {
    req.Player = s.MPRISWatcher.GetSelectedPlayer()
}
```

### Cover Art Proxy

Browsers cannot load `file://` URLs directly for security reasons. Media players often store cover art as local temp files. The cover endpoint reads the local file and serves it over HTTP:

```go
func serveMediaCoverArt(c *fiber.Ctx) error {
    filePath := c.Query("url")
    filePath = strings.TrimPrefix(filePath, "file://")

    // Security: only allow files under trusted temp/cache directories
    cleanPath := filepath.Clean(filePath)
    allowedPrefixes := []string{
        "/tmp/", "/var/tmp/", os.Getenv("HOME") + "/.cache/",
        os.TempDir() + string(filepath.Separator),
    }
    allowed := false
    for _, prefix := range allowedPrefixes {
        if strings.HasPrefix(cleanPath, prefix) {
            allowed = true
            break
        }
    }
    if !allowed {
        return c.Status(fiber.StatusForbidden).JSON(...)
    }

    data, err := os.ReadFile(cleanPath)
    // ... serve with correct Content-Type
}
```

> **Key Pattern (Security):** The path is cleaned with `filepath.Clean()` and checked against allowed directory prefixes before reading. Linux allows `/tmp/`, `/var/tmp/`, and `~/.cache/`; Windows also allows `os.TempDir()`. This blocks directory traversal (`../../../etc/passwd`) while still allowing cover art cache files.

> **Concept (Go Build Tags):** OmniPanel-go uses build tags to compile platform-specific files. `internal/mpris/mpris.go` has `//go:build !windows` (Linux D-Bus), while `internal/mpris/watcher_windows.go` has `//go:build windows` (SMTC). Both expose the same `Watcher` API to the rest of the app.

## Media Integration in Distributed Deployments (Serve + Connect Mode)

In distributed deployments, the relay server (serve mode) has no media watcher — the watcher runs on the host agent (connect mode). The relay server proxies media HTTP API requests to the host agent via WebSocket using a **request-response pattern**.

### How It Works

1. Browser makes HTTP request to `/api/media/*` on the relay server
2. Relay server generates a unique request ID and registers a response channel
3. Relay server sends an `mpris-request` message to the host agent via WebSocket
4. Host agent processes the request locally against the platform watcher (Linux MPRIS or Windows SMTC)
5. Host agent sends an `mpris-response` message back with the matching request ID
6. Relay server delivers the response to the waiting HTTP handler
7. HTTP handler returns the response to the browser

### Request-Response Message Format

```json
// mpris-request (server → host)
{
  "type": "mpris-request",
  "data": {
    "request_id": "mpris-1717200000000000000",
    "endpoint": "/players",
    "method": "GET",
    "body": { "player": "spotify", "action": "play" },
    "query": { "url": "/tmp/cover.jpg" }
  }
}

// mpris-response (host → server)
{
  "type": "mpris-response",
  "data": {
    "request_id": "mpris-1717200000000000000",
    "payload": { "enabled": true, "players": [...] }
  }
}
```

### Pending Request Tracking

The relay server uses a map of request IDs to buffered channels to track in-flight requests:

```go
// internal/relay/server.go
type RelayServer struct {
    // ...
    pendingRequests   map[string]chan map[string]any
    pendingRequestsMu sync.Mutex
}

func (s *RelayServer) sendMPRISRequest(endpoint, method string, body map[string]any, query map[string]string) (map[string]any, error) {
    requestID := generateRequestID()
    respCh := s.registerPendingRequest(requestID)

    // Send mpris-request to host via WebSocket
    s.hostConn.WriteMessage(ws.TextMessage, msgData)

    // Wait for response or timeout (5 seconds)
    select {
    case resp := <-respCh:
        return resp, nil
    case <-time.After(5 * time.Second):
        s.cleanupPendingRequest(requestID)
        return nil, fmt.Errorf("timeout waiting for media response")
    }
}
```

> **Key Pattern (Synchronous WebSocket RPC):** The HTTP handler blocks on a channel read while waiting for the WebSocket response. This turns the asynchronous WebSocket connection into a synchronous request-response mechanism. The 5-second timeout prevents indefinite blocking if the host agent becomes unresponsive.

### Cover Art Over WebSocket

Cover art cannot be served directly as binary over HTTP in distributed mode because the relay server doesn't have access to the host's filesystem. Instead, the host agent reads the file, base64-encodes it, and sends it in the `mpris-response` payload:

```go
// internal/agent/agent.go
func (a *Agent) handleMPRISCover(query map[string]string) map[string]any {
    // ... read file, check security ...
    encoded := base64.StdEncoding.EncodeToString(data)
    return map[string]any{
        "content_type": contentType,
        "data":         encoded,
    }
}
```

The relay server decodes the base64 data and serves it with the correct `Content-Type` header. Base64 encoding increases size by ~33%, but cover art images are typically under 1MB, making this acceptable.

### Frontend Cover Art Auth Token

In serve/connect mode with authentication enabled, the cover art endpoint (`/api/media/cover`) is behind auth middleware. The frontend must include the auth token when setting the cover art `<img>` src:

```javascript
// static/client/client.js
if (newCover && newCover.startsWith('file://')) {
    newCover = '/api/media/cover?url=' + encodeURIComponent(newCover.substring(7));
    newCover = addTokenToUrl(newCover);  // Append ?token=xxx for auth
}
coverImg.src = newCover;
```

The `addTokenToUrl()` function appends the token as a query parameter, which the auth middleware reads to validate the request.

## Frontend Integration

The frontend `handleDataUpdate()` function in `static/client/client.js` processes media data:

```javascript
// static/client/client.js
if (controlMode === 'mpris') {
    // Rewrite file:// URLs to HTTP proxy
    if (newCover && newCover.startsWith('file://')) {
        newCover = '/api/media/cover?url=' + encodeURIComponent(newCover.substring(7));
    }
    coverImg.src = newCover;

    // Toggle play/pause icons based on status
    if (status === 'Playing') {
        playIcon.style.display = 'none';
        pauseIcon.style.display = 'block';
    } else {
        playIcon.style.display = 'block';
        pauseIcon.style.display = 'none';
    }
}
```

### Source Selection Tabs

When `mpris_available_players` contains more than one entry, the client renders pill-style tabs as an overlay on the cover art:

```javascript
function renderMediaSourceTabs() {
    const mediaBlocks = document.querySelectorAll('.media-player[data-control-mode="mpris"]');
    mediaBlocks.forEach(block => {
        const overlay = block.querySelector('.media-source-tabs-overlay');
        if (mprisAvailablePlayers.length <= 1) {
            overlay.style.display = 'none';
            return;
        }
        // Render tabs for each player...
        tab.addEventListener('click', async () => {
            await fetch('/api/media/select', {
                method: 'POST',
                body: JSON.stringify({ player: player.name })
            });
        });
    });
}
```

> **Concept (JSON in DataBus):** The `mpris_available_players` key stores a JSON string (not a parsed object) because the DataBus values are primitive types. The frontend parses it with `JSON.parse()` each time the player list changes.

Media control buttons route to either the media API or keyboard simulation. Volume buttons require special handling: they fetch current system volume via `/api/media/players`, calculate a ±5% delta, and send a `volume` action to `/api/media/control`:

```javascript
// static/client/client.js
if (controlMode === 'mpris') {
    if (action === 'volumedown' || action === 'volumeup') {
        const volumeData = await (await fetch('/api/media/players')).json();
        const currentVolume = volumeData.systemVolume ?? 0.5;
        const delta = action === 'volumedown' ? -0.05 : 0.05;
        const newVolume = Math.max(0, Math.min(1, currentVolume + delta));
        await fetch('/api/media/control', {
            method: 'POST',
            body: JSON.stringify({ action: 'volume', volume: newVolume })
        });
        return;
    }
    await fetch('/api/media/control', {
        method: 'POST',
        body: JSON.stringify({ action: 'playpause' })
    });
} else {
    socket.send(JSON.stringify({
        type: 'simulate-keyboard',
        data: { keyboard_index: 0, key: 'MediaPlayPause', state: 1 }
    }));
}
```

> **Key Pattern (JavaScript Fallback Values):** The volume button handler uses `volumeData.systemVolume ?? 0.5` so a missing field does not break controls. This is a common JS pattern: provide a safe default when API data may be unavailable.

> **Key Pattern (Go Compatibility Contract):** Backend HTTP routes use the generic `/api/media/*` naming, but DataBus keys use the `mpris_*` prefix to clearly indicate media player state. This allows the frontend and blocks to reference media data by its technical origin.

## Configuration

```json
{
  "media_player": {
    "enabled": true,
    "poll_interval": 1000
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable media player monitoring (MPRIS on Linux, SMTC on Windows) |
| `poll_interval` | int | 1000 | Polling frequency in milliseconds (min 500) |

[← Back: Chapter 16](16-data-flow.md) · [Next: Chapter 18 →](18-rss-feed.md)

