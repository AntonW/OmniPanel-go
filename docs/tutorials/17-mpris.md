# Chapter 17: MPRIS Media Player Integration

## What This Package Does

The `mpris` package monitors media players on Linux desktop environments via the D-Bus session bus. It automatically discovers MPRIS-compliant players (Spotify, VLC, Firefox, etc.), polls their state (title, artist, cover art, progress, playback status), and publishes that data to the `DataBus` for frontend consumption.

> **Platform Note:** MPRIS is Linux-only. It uses the `github.com/godbus/dbus/v5` library to connect to the D-Bus session bus. Windows and macOS do not have D-Bus, so this package is effectively a no-op on those platforms (the watcher returns `nil` when MPRIS is disabled or unavailable).

## Key Concepts

### What Is MPRIS?

MPRIS (Media Player Remote Interfacing Specification) is a D-Bus interface standard used by Linux media players. It allows external programs to:
- Discover what media players are running
- Read metadata (title, artist, album, cover art URL)
- Read playback state (playing, paused, stopped, position, volume)
- Send control commands (play, pause, next, previous, stop, volume)

> **Concept (D-Bus):** D-Bus is an inter-process communication system on Linux. Think of it like a message bus where applications can publish services and other applications can connect to them. The "session bus" is per-user (your desktop session), while the "system bus" is system-wide. MPRIS players register themselves on the session bus with names like `org.mpris.MediaPlayer2.spotify`.

### How the Watcher Works

The MPRIS watcher operates in two phases:

1. **Initial discovery** — When the server starts, it scans the D-Bus session bus for any already-running MPRIS players
2. **Signal monitoring** — It subscribes to D-Bus `NameOwnerChanged` signals to detect when new players appear or existing ones disappear
3. **Polling loop** — Every N milliseconds (configurable), it queries each discovered player for state updates

## The Data Structures

```go
// internal/mpris/mpris.go
type PlayerState struct {
    Identity       string  // Human-readable name (e.g., "Spotify", "VLC")
    PlayerName     string  // D-Bus service suffix (e.g., "spotify", "vlc")
    PlaybackStatus string  // "Playing", "Paused", or "Stopped"
    Title          string  // Track title from xesam:title
    Artist         string  // Track artist from xesam:artist (joined with ", ")
    Album          string  // Album name from xesam:album
    ArtURL         string  // Cover art URL from mpris:artUrl (may be file://)
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
    config         *config.MPRISConfig
    databus        *databus.DataBus
    logger         *slog.Logger
    conn           *dbus.Conn
    players        map[string]*PlayerState
    selectedPlayer string // D-Bus name suffix of the active player
    stopCh         chan struct{}
    running        bool
}
```

The `Watcher` holds a D-Bus connection, a map of discovered players, a `selectedPlayer` field that tracks which player's state is published to the DataBus, and a stop channel for clean shutdown. The `sync.RWMutex` protects concurrent access to the players map.

## Initialization

```go
// internal/mpris/mpris.go
func New(cfg *config.MPRISConfig, db *databus.DataBus, logger *slog.Logger) *Watcher {
    if cfg == nil || !cfg.Enabled {
        return nil
    }
    // ... initialize Watcher
    return w
}
```

`New` returns `nil` if MPRIS is disabled in config. The caller (`state.New`) checks for `nil` before calling `Start()`.

```go
// internal/state/state.go
mprisWatcher := mpris.New(&cfg.MPRIS, db, slog.Default())
if mprisWatcher != nil {
    if err := mprisWatcher.Start(); err != nil {
        slog.Warn("MPRIS watcher failed to start", "error", err)
    }
}
app.MPRISWatcher = mprisWatcher
```

## Starting the Watcher

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

> **Key Pattern (Mutex):** The mutex is explicitly unlocked before calling `discoverExistingPlayers()` and launching goroutines. This is critical because `discoverExistingPlayers()` calls `discoverPlayer()` which also acquires the mutex. Using `defer w.mu.Unlock()` here would cause a deadlock since `sync.Mutex` is not reentrant.

## Player Discovery

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

```go
func (w *Watcher) discoverPlayer(playerName string) {
    state := &PlayerState{PlayerName: playerName}

    w.mu.Lock()
    w.players[playerName] = state
    if w.selectedPlayer == "" {
        w.selectedPlayer = playerName  // Auto-select first player
    }
    w.mu.Unlock()

    w.updatePlayer(playerName)
    w.publishPlayersList()  // Notify frontend of available players
}
```

When a player is discovered, it is automatically selected if no player has been selected yet. After updating the player's state, `publishPlayersList()` is called to broadcast the full player list to the DataBus.

## Signal Monitoring

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

When a player disconnects, `removePlayer()` removes it from the map. If the disconnected player was the selected one, the watcher automatically picks another available player. The player list is re-published and the new selected player's state is immediately fetched.

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
            w.pollPlayers()
        }
    }
}
```

Each poll queries the D-Bus properties for every known player:

```go
func (w *Watcher) getPlayerProperties(obj dbus.BusObject) (map[string]any, error) {
    result := make(map[string]any)

    // PlaybackStatus: "Playing", "Paused", "Stopped"
    status, _ := obj.GetProperty(mprisPlayerIface + ".PlaybackStatus")
    result["PlaybackStatus"], _ = status.Value().(string)

    // Metadata: title, artist, album, artUrl, length
    metadata, _ := obj.GetProperty(mprisPlayerIface + ".Metadata")
    meta := parseMetadata(metadata)
    result["Title"] = meta["Title"]
    result["Artist"] = meta["Artist"]
    // ...

    // Position (microseconds)
    position, _ := obj.GetProperty(mprisPlayerIface + ".Position")
    result["Position"], _ = position.Value().(int64)

    // Volume (0.0 - 1.0)
    volume, _ := obj.GetProperty(mprisPlayerIface + ".Volume")
    result["Volume"], _ = volume.Value().(float64)

    return result, nil
}
```

> **Key Pattern (D-Bus Variants):** D-Bus properties are returned as `dbus.Variant` types, which wrap any Go value. You must call `.Value()` and then type-assert to the expected type. The type assertion returns `(value, ok)` — if the property has a different type than expected, `ok` is `false` and the default zero value is used. This is why the code uses blank identifiers (`_`) for errors: missing properties are silently skipped.

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
        return fmt.Errorf("mpris: player %q not found", playerName)
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

> **Key Pattern (Lock Release Before I/O):** The mutex is released before calling `updatePlayer()`, which acquires its own read lock. Holding the write lock during `updatePlayer()` would cause a deadlock since `sync.RWMutex` is not reentrant.

## HTTP API Endpoints

Four endpoints in `internal/routes/mpris.go` (default mode) or `internal/relay/server.go` (serve mode):

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/mpris/players` | GET | List connected players with their current state (identity, playback status, metadata, volume, capabilities) |
| `/api/mpris/control` | POST | Send playback command (play, pause, next, etc.) |
| `/api/mpris/select` | POST | Set the active player. Body: `{ "player": "spotify" }` |
| `/api/mpris/cover` | GET | Proxy local cover art files for browser access |

### Player Selection

The `/api/mpris/select` endpoint changes which player's state is published to the DataBus. Control commands (`/api/mpris/control`) default to the selected player when no `player` field is provided in the request body:

```go
if req.Player == "" {
    req.Player = s.MPRISWatcher.GetSelectedPlayer()
}
```

### Cover Art Proxy

Browsers cannot load `file://` URLs directly for security reasons. MPRIS players often store cover art as local temp files (e.g., `file:///tmp/plasma-browser-integration_artwork_*.jpg`). The cover endpoint reads the local file and serves it over HTTP:

```go
func serveMPRISCoverArt(c *fiber.Ctx) error {
    filePath := c.Query("url")
    filePath = strings.TrimPrefix(filePath, "file://")

    // Security: only allow files under /tmp, /var/tmp, or ~/.cache/
    cleanPath := filepath.Clean(filePath)
    allowedPrefixes := []string{"/tmp/", "/var/tmp/", os.Getenv("HOME") + "/.cache/"}
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

> **Key Pattern (Security):** The path is cleaned with `filepath.Clean()` and checked against allowed directory prefixes (`/tmp/`, `/var/tmp/`, `~/.cache/`) before reading. This prevents directory traversal attacks (`../../../etc/passwd`) while supporting cover art stored in common cache locations used by KDE Connect and other media players.

## MPRIS in Distributed Deployments (Serve + Connect Mode)

In distributed deployments, the relay server (serve mode) has no MPRIS watcher — the watcher runs on the host agent (connect mode). The relay server proxies MPRIS HTTP API requests to the host agent via WebSocket using a **request-response pattern**.

### How It Works

1. Browser makes HTTP request to `/api/mpris/*` on the relay server
2. Relay server generates a unique request ID and registers a response channel
3. Relay server sends an `mpris-request` message to the host agent via WebSocket
4. Host agent processes the request locally against the D-Bus session
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
        return nil, fmt.Errorf("timeout waiting for MPRIS response")
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

In serve/connect mode with authentication enabled, the cover art endpoint (`/api/mpris/cover`) is behind auth middleware. The frontend must include the auth token when setting the cover art `<img>` src:

```javascript
// static/client/client.js
if (newCover && newCover.startsWith('file://')) {
    newCover = '/api/mpris/cover?url=' + encodeURIComponent(newCover.substring(7));
    newCover = addTokenToUrl(newCover);  // Append ?token=xxx for auth
}
coverImg.src = newCover;
```

The `addTokenToUrl()` function appends the token as a query parameter, which the auth middleware reads to validate the request.

## Frontend Integration

The frontend `handleDataUpdate()` function in `static/client/client.js` processes MPRIS data:

```javascript
// static/client/client.js
if (controlMode === 'mpris') {
    // Rewrite file:// URLs to HTTP proxy
    if (newCover && newCover.startsWith('file://')) {
        newCover = '/api/mpris/cover?url=' + encodeURIComponent(newCover.substring(7));
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
            await fetch('/api/mpris/select', {
                method: 'POST',
                body: JSON.stringify({ player: player.name })
            });
        });
    });
}
```

> **Concept (JSON in DataBus):** The `mpris_available_players` key stores a JSON string (not a parsed object) because the DataBus values are primitive types. The frontend parses it with `JSON.parse()` each time the player list changes.

Media control buttons route to either the MPRIS API or keyboard simulation. Volume buttons require special handling: they fetch the current volume from the selected player via `/api/mpris/players`, calculate a ±5% delta, and send a `volume` action to `/api/mpris/control`:

```javascript
// static/client/client.js
if (controlMode === 'mpris') {
    if (action === 'volumedown' || action === 'volumeup') {
        const volumeData = await (await fetch('/api/mpris/players')).json();
        // Find the selected player; fall back to first available.
        let currentPlayer = volumeData.players.find(p => p.name === mprisSelectedPlayer);
        if (!currentPlayer) currentPlayer = volumeData.players[0];
        const currentVolume = currentPlayer.volume || 0.5;
        const delta = action === 'volumedown' ? -0.05 : 0.05;
        const newVolume = Math.max(0, Math.min(1, currentVolume + delta));
        await fetch('/api/mpris/control', {
            method: 'POST',
            body: JSON.stringify({ action: 'volume', volume: newVolume })
        });
        return;
    }
    await fetch('/api/mpris/control', {
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

> **Key Pattern (Player Selection in Frontend):** The volume button handler uses `mprisSelectedPlayer` to find the correct player in the `/api/mpris/players` response. If no player is selected yet, it falls back to `players[0]`. This ensures volume adjustments are relative to the actual current volume of the active player, not an arbitrary default.

## Configuration

```json
{
  "mpris": {
    "enabled": true,
    "poll_interval": 1000
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable MPRIS D-Bus monitoring |
| `poll_interval` | int | 1000 | Polling frequency in milliseconds (min 500) |

[← Back: Chapter 16](16-data-flow.md) · [Next: Chapter 18 →](18-rss-feed.md)
