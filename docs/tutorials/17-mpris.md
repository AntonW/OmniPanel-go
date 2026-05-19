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

Four endpoints in `internal/routes/mpris.go`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/mpris/players` | GET | List connected players with their current state |
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

    // Security: only allow files under /tmp
    cleanPath := filepath.Clean(filePath)
    if !strings.HasPrefix(cleanPath, "/tmp/") {
        return c.Status(fiber.StatusForbidden).JSON(...)
    }

    data, err := os.ReadFile(cleanPath)
    // ... serve with correct Content-Type
}
```

> **Key Pattern (Security):** The path is cleaned with `filepath.Clean()` and checked against `/tmp/` prefix before reading. This prevents directory traversal attacks (`../../../etc/passwd`).

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

Media control buttons route to either the MPRIS API or keyboard simulation:

```javascript
if (controlMode === 'mpris') {
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
