# Chapter 20: Distributed Deployment (Server + Host Agent)

## What This Feature Does

OmniPanel-go can be split into two processes running on different machines:

1. **Relay Server** (`serve` mode) — serves the WebUI and acts as a WebSocket hub
2. **Host Agent** (`connect` mode) — runs all subsystems and connects to the server

This enables deploying the WebUI on a central server (always-on machine, cloud instance) while the host agent runs on a machine behind a firewall. The host initiates an **outbound** WebSocket connection to the server, so no inbound ports need to be opened on the host.

## Why Distributed Deployment?

In the traditional mode, the WebUI and subsystems run on the same machine. This works great for local networks but has limitations:

- The machine running OmniPanel-go must be reachable from your tablet/phone
- If the machine is behind a NAT or firewall, you need port forwarding
- You can't easily share panels across multiple hosts

Distributed deployment solves this by separating concerns:

- **Central server**: always-on, accessible from your network, serves the WebUI
- **Host agent**: runs on any machine (even behind a firewall), connects outbound to the server

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Central Server (serve mode)                    │
│  - Fiber HTTP (static files, API)               │
│  - WebSocket relay hub                          │
│    - Multiple browser connections               │
│    - Single host connection (1:1)               │
└──────────────┬──────────────────────────────────┘
               │ WebSocket (ws://server/ws)
               │ ← simulate-*, execute-command
               │ → data-update, speech-result
               ▼
┌─────────────────────────────────────────────────┐
│  Host Agent (connect mode)                      │
│  - WebSocket client (no HTTP server)            │
│  - Joystick/Mousepad/Keyboard managers          │
│  - DataBus (metrics)                            │
│  - SpeechManager (Vosk/llama-cpp)               │
│  - MPRISWatcher                                 │
│  - RSSManager                                   │
│  - Auto-reconnect with backoff                  │
└─────────────────────────────────────────────────┘
```

> **Concept: Outbound vs inbound connections**
> An outbound connection is initiated by the client (host agent) to the server. This is important because firewalls typically allow outbound connections but block inbound ones. By having the host agent connect to the server, you don't need to open any ports on the host machine.

## Three Deployment Modes

| Mode | Command | HTTP Server | Subsystems | WebSocket |
|------|---------|:-----------:|:----------:|-----------|
| Default | `./omnipanel-go` | Yes (local) | Yes | Server |
| Serve | `./omnipanel-go serve` | Yes (central) | No | Relay hub |
| Connect | `./omnipanel-go connect <addr>` | No | Yes | Client |

The default mode is unchanged — everything runs on one machine as before.

## Using Serve Mode

```bash
# Start the relay server on the default port (3000)
./omnipanel-go serve

# Override the port
./omnipanel-go serve --port 8080
```

The relay server:
- Serves all static files (WebUI, editor, client)
- Handles all API endpoints (panel CRUD, config, MPRIS, etc.)
- Accepts browser WebSocket connections at `/ws` (multiple allowed)
- Accepts exactly one host WebSocket connection at `/ws?type=host`
- Relays messages bidirectionally between the host and all browsers

> **Key Pattern: 1:1 host connection**
> The server accepts exactly one host connection. If a second host tries to connect, it is rejected with a WebSocket close message. This simplifies the architecture — all browser commands go to the single host, and all host responses go to all browsers.

## Using Connect Mode

```bash
# Connect to a relay server
./omnipanel-go connect 10.0.0.1:3000

# Or use the address from config.json
./omnipanel-go connect
```

The host agent:
- Creates all subsystems (joystick, mousepad, keyboard, databus, speech, MPRIS, RSS)
- Does **not** start an HTTP server
- Connects to the relay server via WebSocket (`ws://server/ws?type=host`)
- Sends `{"type": "host-register"}` on connect
- Receives forwarded browser commands and executes them locally
- Sends results back to the server for broadcast to browsers
- Auto-reconnects on disconnect with exponential backoff (1s → 2s → 4s → max 30s)
- Sends heartbeat every 15s to keep the connection alive

## Configuration

### Server Address

The host agent needs to know where to find the relay server. Three ways to configure it:

1. **Command-line argument** (highest priority):
   ```bash
   ./omnipanel-go connect 10.0.0.1:3000
   ```

2. **Config file** (`config.json`):
   ```json
   {
     "port": 3000,
     "server_address": "10.0.0.1:3000",
     "numJoysticks": 5
   }
   ```

3. **Environment variable**:
   ```bash
   export OMNIPANEL_SERVER_ADDRESS=10.0.0.1:3000
   ./omnipanel-go connect
   ```

## Message Flow

### Browser → Server → Host

When a user taps a button on their tablet:

1. Browser sends `{"type": "simulate-button", "data": {...}}` to server
2. Server forwards the message to the host agent
3. Host agent routes to `handleButton()` → `JoystickManager.Send()`
4. Virtual joystick button is pressed on the host machine

### Host → Server → Browsers

When the host collects system metrics:

1. Host agent's DataBus collects CPU, memory, disk, network metrics
2. Host agent sends `{"type": "data-update", "data": {...}}` to server
3. Server broadcasts the message to all connected browsers
4. All browsers update their data display blocks

### Control Messages

| Message | Direction | Purpose |
|---------|-----------|---------|
| `host-register` | Host → Server | Host identifies itself after connecting |
| `heartbeat` | Host → Server | Keepalive (every 15s) |
| `heartbeat-ack` | Server → Host | Acknowledge heartbeat |

## The Relay Server Code

### `internal/relay/server.go`

The `RelayServer` struct holds the Fiber HTTP app and connection tracking:

```go
type RelayServer struct {
    app       *fiber.App
    hostConn  *ws.Conn
    hostMu    sync.Mutex
    browsers  map[*ws.Conn]struct{}
    browserMu sync.Mutex
    staticDir string
    userPath  string
    config    *config.Config
}
```

Two separate mutexes protect the host connection and browser connections independently. This allows concurrent browser message handling without blocking on host operations.

> **Concept: Separate mutexes for separate concerns**
> Using two mutexes (`hostMu` and `browserMu`) instead of one reduces contention. Browser connections can be added/removed concurrently with host message handling, since they protect different data.

### `internal/relay/handler.go`

The handler distinguishes connections by query parameter:

```go
func (s *RelayServer) handleWS(c *ws.Conn) {
    connType := c.Query("type", "")
    if connType == "host" {
        s.handleHostConn(c)
    } else {
        s.handleBrowserConn(c)
    }
}
```

Host connection handling enforces the 1:1 constraint:

```go
func (s *RelayServer) handleHostConn(c *ws.Conn) {
    s.hostMu.Lock()
    if s.hostConn != nil {
        s.hostMu.Unlock()
        c.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.ClosePolicyViolation, "host already connected"))
        c.Close()
        return
    }
    s.hostConn = c
    s.hostMu.Unlock()
    // ... read loop
}
```

> **Key Pattern: Reject second connection**
> The mutex-protected check-and-set ensures only one host can be registered at a time. If a second host connects, it receives a WebSocket close message with a descriptive reason.

## The Host Agent Code

### `internal/agent/agent.go`

The `Agent` struct mirrors `AppState` but uses a WebSocket client instead of an HTTP server:

```go
type Agent struct {
    conn   *websocket.Conn
    connMu sync.RWMutex
    // ... all subsystems (joystick, mousepad, keyboard, databus, speech, etc.)
}
```

The `Run()` method implements the auto-reconnect loop:

```go
func (a *Agent) Run(serverAddr string) {
    backoff := time.Second
    maxBackoff := 30 * time.Second

    for {
        url := fmt.Sprintf("ws://%s/ws?type=host", serverAddr)
        conn, _, err := websocket.DefaultDialer.Dial(url, nil)
        if err != nil {
            time.Sleep(backoff)
            backoff = min(backoff*2, maxBackoff)
            continue
        }

        backoff = time.Second
        a.sendJSON(map[string]any{"type": "host-register"})
        a.runConnection(conn)

        time.Sleep(backoff)
        backoff = min(backoff*2, maxBackoff)
    }
}
```

> **Key Pattern: Exponential backoff**
> The backoff doubles on each failed connection attempt (1s → 2s → 4s → 8s → ... → max 30s). On successful connection, it resets to 1s. This prevents hammering the server during outages while recovering quickly when it comes back online.

### AppStateInterface

Both `AppState` (default mode) and `Agent` (connect mode) implement the `AppStateInterface`:

```go
type AppStateInterface interface {
    RegisterClient(ch chan []byte) uint64
    UnregisterClient(ch chan []byte)
    Broadcast(msg []byte)
    BroadcastJSON(msg map[string]any)
    UpdateJoystickCount(count uint8)
    GetConfig() *config.Config
    GetJoystickManager() *devices.JoystickManager
    GetMousepadManager() *devices.MousepadManager
    GetKeyboardManager() *devices.KeyboardManager
    GetDataBus() *databus.DataBus
    GetSpeechManager() *speech.SpeechManager
    GetMPRISWatcher() *mpris.Watcher
    GetRSSManager() *rssfeed.Manager
}
```

This allows the same WebSocket message handler code (`internal/websocket/handler.go`) to work in both modes without modification.

> **Concept: Go interfaces for polymorphism**
> Go interfaces are satisfied implicitly — any type that implements the required methods automatically satisfies the interface. This lets you write code that works with multiple implementations without inheritance or generics.

## Edge Cases

### Host Disconnects

When the host agent disconnects, the server broadcasts a log event to all browsers:

```go
s.broadcastToBrowsers(map[string]any{
    "type":      "log-event",
    "timestamp": time.Now().Format("2006-01-02 15:04:05"),
    "data":      "Host disconnected",
})
```

The browser's connection log shows this message, so users know the host is unavailable.

### No Host Connected

When a browser sends a command but no host is connected, the server logs a warning and drops the message:

```go
if hostConn == nil {
    slog.Warn("No host connected, dropping message", "type", msgType)
    return
}
```

### Server Unreachable

The host agent retries indefinitely with exponential backoff. The backoff caps at 30 seconds to avoid excessive wait times.

### Host Reconnects

On reconnection, the host agent sends `host-register` again. The server resets its host connection and resumes relaying messages. Subsystem state (speech config, RSS config) is maintained on the host side.

## Key Takeaways

- Distributed deployment splits OmniPanel-go into a relay server and host agent
- The host agent initiates an outbound WebSocket connection (no inbound ports needed)
- Serve mode: Fiber HTTP server + WebSocket relay hub (no subsystems)
- Connect mode: WebSocket client + all subsystems (no HTTP server)
- 1:1 host connection — second host is rejected with a close message
- Auto-reconnect with exponential backoff (1s → 2s → 4s → max 30s)
- Heartbeat every 15s keeps the connection alive
- `AppStateInterface` allows the same WebSocket handler to work in both modes
- Server address configurable via CLI arg, config file, or environment variable
- Panel files, themes, and blocks live on the central server

[← Back: Chapter 19](19-windows-build-and-ci.md) · [Next: Chapter 7 →](07-commands.md)
