# Chapter 3: Application State & Concurrency

## What This Package Does

`AppState` is the central hub of the application. It holds references to every subsystem (config, joystick managers, databus) and manages WebSocket client connections. It's the glue that ties everything together.

## The AppState Struct

```go
// internal/state/state.go
type AppState struct {
    mu             sync.RWMutex
    Config         *config.Config
    ConfigPath     string
    UserPath       string
    StaticDir      string
    JoystickManager *devices.JoystickManager
    MousepadManager *devices.MousepadManager
    DataBus        *databus.DataBus

    broadcastMu sync.RWMutex
    clients     map[chan []byte]struct{}
}
```

Two separate mutexes protect two different concerns:
- `mu` protects the `Config` (read/written by HTTP handlers)
- `broadcastMu` protects the `clients` map (read/written by WebSocket handlers)

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
    db := databus.New()

    app := &AppState{
        Config:        cfg,
        ConfigPath:    configPath,
        UserPath:      userPath,
        StaticDir:     staticDir,
        JoystickManager: jsMgr,
        MousepadManager: mpMgr,
        DataBus:       db,
        clients:       make(map[chan []byte]struct{}),
    }

    app.StartDataBroadcast()
    return app
}
```

All subsystems are created here. `StartDataBroadcast()` kicks off the background goroutine that pushes metrics to all connected clients.

## Client Registration

```go
func (s *AppState) RegisterClient(ch chan []byte) {
    s.broadcastMu.Lock()
    defer s.broadcastMu.Unlock()
    s.clients[ch] = struct{}{}
}

func (s *AppState) UnregisterClient(ch chan []byte) {
    s.broadcastMu.Lock()
    defer s.broadcastMu.Unlock()
    delete(s.clients, ch)
    close(ch)
}
```

When a WebSocket connects, it creates a channel and registers it. When it disconnects, the channel is removed from the map and closed (so the write goroutine knows to stop).

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

func (s *AppState) UpdatePanel(panel string) {
    s.mu.Lock()
    s.Config.Panel = panel
    s.mu.Unlock()

    _ = s.Config.Save(s.ConfigPath)
    s.BroadcastJSON(map[string]any{"type": "force-reload"})
}
```

When the user changes settings via the web UI:
1. Update the in-memory config (under mutex)
2. Apply the change to the relevant subsystem
3. Persist to disk
4. For panel changes: broadcast `force-reload` so all clients refresh

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

## Cleanup

```go
func (s *AppState) Close() {
    s.JoystickManager.Close()
    s.MousepadManager.Close()
}
```

Called via `defer` in `main.go`. Destroys all virtual input devices so they don't linger after the program exits.

## Key Takeaways

- `AppState` is the central hub — all subsystems flow through it
- Separate mutexes for separate concerns reduces contention
- `map[T]struct{}` is the idiomatic Go set
- Non-blocking channel sends (`select` + `default`) prevent slow consumers from blocking producers
- `time.Ticker` is the right tool for periodic background work
- `defer` with `Unlock()` ensures mutexes are always released

[← Back: Chapter 2](02-configuration.md) · [Next: Chapter 4 →](04-databus.md)
