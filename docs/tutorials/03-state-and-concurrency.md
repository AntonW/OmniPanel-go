# Chapter 3: Application State & Concurrency

## What This Package Does

`AppState` is the central hub of the application. It holds references to every subsystem (config, joystick managers, databus) and manages WebSocket client connections. It's the glue that ties everything together.

## The AppState Struct

```go
// internal/state/state.go
type AppState struct {
    mu              sync.RWMutex
    Config          *config.Config
    ConfigPath      string
    UserPath        string
    StaticDir       string
    JoystickManager *devices.JoystickManager
    MousepadManager *devices.MousepadManager
    KeyboardManager *devices.KeyboardManager
    DataBus         *databus.DataBus
    SpeechManager   *speech.SpeechManager
    MPRISWatcher    *mpris.Watcher
    RSSManager      *rssfeed.Manager

    broadcastMu  sync.RWMutex
    clients      map[chan []byte]struct{}
    nextClientID uint64
    clientIDs    map[chan []byte]uint64
}
```

Two separate mutexes protect two different concerns:
- `mu` protects the `Config` (read/written by HTTP handlers)
- `broadcastMu` protects the `clients` map and client ID tracking (read/written by WebSocket handlers)

> **Concept: `sync.RWMutex`**
> A read-write mutex allows **multiple concurrent readers** OR **one exclusive writer**. Use `RLock()`/`RUnlock()` for reads and `Lock()`/`Unlock()` for writes. This is more efficient than a regular `sync.Mutex` when reads are frequent (which they are here — every WebSocket tick reads all clients).

> **Concept: `map[chan []byte]struct{}`**
> This is a **set** implemented as a map. The keys are channels (one per WebSocket client). The values are `struct{}` — an empty struct that takes zero bytes of memory. We don't care about the values; we only care about the keys. This is the idiomatic Go set pattern.

## Creating AppState

```go
func New(cfg *config.Config, configPath, userPath, baseDir string) *AppState {
    staticDir := filepath.Join(baseDir, "static")
    jsMgr := devices.New(cfg.NumJoysticks)
    mpMgr := devices.NewMousepad(cfg.NumJoysticks)
    kbMgr := devices.NewKeyboard(cfg.NumJoysticks)
    db := databus.New()

    app := &AppState{
        Config:          cfg,
        ConfigPath:      configPath,
        UserPath:        userPath,
        StaticDir:       staticDir,
        JoystickManager: jsMgr,
        MousepadManager: mpMgr,
        KeyboardManager: kbMgr,
        DataBus:         db,
        clients:         make(map[chan []byte]struct{}),
        clientIDs:       make(map[chan []byte]uint64),
    }

    sm := speech.New(&cfg.Speech, app, jsMgr, userPath)
    app.SpeechManager = sm

    mprisWatcher := mpris.New(&cfg.MPRIS, db, slog.Default())
    if mprisWatcher != nil {
        if err := mprisWatcher.Start(); err != nil {
            slog.Warn("MPRIS watcher failed to start", "error", err)
        }
    }
    app.MPRISWatcher = mprisWatcher

    app.RSSManager = rssfeed.New(func(clientID uint64, blockID string, entries []rssfeed.EntryWithNew) {
        app.broadcastToClient(clientID, map[string]any{
            "type": "rss-update",
            "data": map[string]any{
                "block_id": blockID,
                "entries":  entries,
            },
        })
    })
    app.RSSManager.Start()

    app.StartDataBroadcast()

    return app
}
```

All subsystems are created here: joystick/mousepad/keyboard managers, databus, speech manager, MPRIS watcher, and RSS feed manager. `StartDataBroadcast()` kicks off the background goroutine that pushes metrics to all connected clients.

## Client Registration

```go
func (s *AppState) RegisterClient(ch chan []byte) uint64 {
    s.broadcastMu.Lock()
    defer s.broadcastMu.Unlock()
    s.nextClientID++
    clientID := s.nextClientID
    s.clients[ch] = struct{}{}
    s.clientIDs[ch] = clientID
    return clientID
}

func (s *AppState) UnregisterClient(ch chan []byte) {
    s.broadcastMu.Lock()
    defer s.broadcastMu.Unlock()
    delete(s.clients, ch)
    delete(s.clientIDs, ch)
    close(ch)
}
```

When a WebSocket connects, it creates a channel and registers it. A unique `uint64` client ID is assigned and returned — used for targeted message delivery (e.g., RSS updates sent to specific clients). When it disconnects, the channel is removed from both maps and closed (so the write goroutine knows to stop).

> **Key Pattern: defer with mutex**
> `defer s.broadcastMu.Unlock()` right after `Lock()` ensures the mutex is always released, even if the function panics or returns early. This is the standard Go pattern for mutex usage.

## Broadcasting to All Clients

```go
func (s *AppState) Broadcast(msg []byte) {
    s.broadcastMu.RLock()
    defer s.broadcastMu.RUnlock()
    for ch := range s.clients {
        select {
        case ch <- msg:
        default:
        }
    }
}
```

> **Key Pattern: Non-blocking channel send**
> The `select` with `default` makes the send **non-blocking**. If a client's channel buffer is full, the `default` case runs (which does nothing), and the message is silently dropped for that client. This prevents a slow client from blocking the entire broadcast.

```go
func (s *AppState) BroadcastJSON(msg map[string]any) {
    data, _ := json.Marshal(msg)
    s.Broadcast(data)
}
```

A convenience wrapper that marshals a map to JSON before broadcasting. The error from `Marshal` is ignored because marshaling a `map[string]any` with JSON-compatible values never fails.

## Updating Configuration at Runtime

```go
func (s *AppState) UpdateJoystickCount(count uint8) {
    s.mu.Lock()
    s.Config.NumJoysticks = count
    s.mu.Unlock()

    s.JoystickManager.Reload(count)
    slog.Info("Joystick count updated", "count", count)

    _ = s.Config.Save(s.ConfigPath)
}
```

When the user changes the joystick count via the web UI:
1. Update the in-memory config (under mutex)
2. Apply the change to the joystick manager
3. Persist to disk

## The Data Broadcast Loop

```go
func (s *AppState) StartDataBroadcast() {
    s.DataBus.StartMetrics(500 * time.Millisecond)

    ticker := time.NewTicker(500 * time.Millisecond)
    go func() {
        for range ticker.C {
            snapshot := s.DataBus.Snapshot()
            s.BroadcastJSON(map[string]any{
                "type": "data-update",
                "data": snapshot,
            })
        }
    }()
}
```

Two things happen every 500ms:
1. The DataBus collects fresh system metrics (CPU, memory, disk, network)
2. A snapshot is broadcast to all WebSocket clients

> **Concept: `time.Ticker`**
> A `Ticker` sends the current time on its channel `C` at regular intervals. `for range ticker.C` loops forever, receiving each tick. Unlike `time.Sleep`, a ticker is designed for repeated periodic work.

## Thread-Safe Config Reading

```go
func (s *AppState) GetConfig() *config.Config {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.Config
}
```

HTTP handlers call this to read the current config. `RLock` allows multiple handlers to read simultaneously.

## Loading Panel JSON

```go
func (s *AppState) LoadPanelJSON(panelName string) ([]byte, error) {
    path := filepath.Join(s.UserPath, "panels", panelName+".json")
    return os.ReadFile(path)
}
```

Reads a panel definition from `user/panels/<name>.json`. Used when a WebSocket client connects to send them the current panel.

## Targeted Client Broadcasting

```go
func (s *AppState) broadcastToClient(clientID uint64, msg map[string]any) {
    data, err := json.Marshal(msg)
    if err != nil {
        slog.Warn("Failed to marshal RSS update", "error", err)
        return
    }

    s.broadcastMu.RLock()
    defer s.broadcastMu.RUnlock()
    for ch, id := range s.clientIDs {
        if id == clientID {
            select {
            case ch <- data:
            default:
            }
            return
        }
    }
}
```

Unlike `Broadcast` which sends to all clients, `broadcastToClient` finds a specific client by ID and sends only to them. Uses the same non-blocking `select` + `default` pattern. This is used by the RSS manager to deliver per-client updates (e.g., new feed entries that only this client hasn't seen yet).

## Cleanup

```go
func (s *AppState) Close() {
    slog.Info("Closing joystick manager...")
    s.JoystickManager.Close()
    slog.Info("Closing mousepad manager...")
    s.MousepadManager.Close()
    slog.Info("Closing keyboard manager...")
    s.KeyboardManager.Close()
    slog.Info("Closing speech manager...")
    s.SpeechManager.Close()
    slog.Info("Closing MPRIS watcher...")
    if s.MPRISWatcher != nil {
        s.MPRISWatcher.Close()
    }
    slog.Info("Closing RSS manager...")
    s.RSSManager.Close()
    slog.Info("All subsystems closed")
}
```

Called via `defer` in `main.go`. Each subsystem is closed with a log message before and after, so if shutdown hangs you can see exactly which subsystem is blocking. Virtual input devices are destroyed so they don't linger after the program exits.

## Key Takeaways

- `AppState` is the central hub — all subsystems flow through it
- Separate mutexes for separate concerns reduces contention
- `map[T]struct{}` is the idiomatic Go set
- Non-blocking channel sends (`select` + `default`) prevent slow consumers from blocking producers
- `time.Ticker` is the right tool for periodic background work
- `defer` with `Unlock()` ensures mutexes are always released
- Each subsystem logs during `Close()` so shutdown hangs are easy to diagnose
- Client IDs enable targeted per-client message delivery (used by RSS manager)

[← Back: Chapter 2](02-configuration.md) · [Next: Chapter 4 →](04-databus.md)
