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
               │ WebSocket (ws:// or wss://)
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
│  - System Tray (with display):                  │
│      Enter/Exit Fullscreen, Exit Application    │
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

The serve mode can also run as a Docker container built with [ko](21-container-build.md). The container embeds the WebUI and starter files, and populates an empty mounted `user/` volume on first run.

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
# Connect to a relay server (ws://, default)
./omnipanel-go connect 10.0.0.1:3000

# Connect with WebSocket Secure (wss://)
./omnipanel-go connect wss://10.0.0.1:3000

# Or use the address from config.json
./omnipanel-go connect
```

The host agent:
- Creates all subsystems (joystick, mousepad, keyboard, databus, speech, MPRIS, RSS)
- Does **not** start an HTTP server
- Connects to the relay server via WebSocket (`ws://server/ws?type=host` or `wss://server/ws?type=host`)
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
   ./omnipanel-go connect 10.0.0.1:3000          # ws:// (default)
   ./omnipanel-go connect wss://10.0.0.1:3000    # wss:// (secure)
   ```

2. **Config file** (`config.json`):
   ```json
   {
     "port": 3000,
     "server_address": "10.0.0.1:3000",
     "numJoysticks": 5
   }
   ```
   Or with WebSocket Secure:
   ```json
   {
     "server_address": "wss://10.0.0.1:3000"
   }
   ```

3. **Environment variable**:
   ```bash
   export OMNIPANEL_SERVER_ADDRESS=10.0.0.1:3000
   ./omnipanel-go connect
   ```

The server address accepts either a plain `host:port` (defaults to `ws://`) or a full WebSocket URL (`ws://` or `wss://`). Use `wss://` when the relay server is behind a TLS-terminating reverse proxy.

### Authentication

All HTTP routes and WebSocket connections in serve/connect modes can be protected with a shared token. When `auth_token` is configured, clients must provide the token to access the UI or connect as a host agent. If `auth_token` is empty (default), authentication is disabled — this keeps backward compatibility and the default mode (single machine) open.

Configure the token via:

1. **Config file** (`config.json`):
    ```json
    {
      "auth_token": "your-secret-token"
    }
    ```

2. **Environment variable**:
    ```bash
    export OMNIPANEL_AUTH_TOKEN=your-secret-token
    ```

**Accessing the UI:** When authentication is enabled and no token is present in the URL or browser storage, the frontend automatically redirects to the login page (`/login`). The login form validates the token against `/api/config` and stores it in `localStorage` (persistent) or `sessionStorage` (tab-only). After login, the token is appended to all API requests as an `Authorization: Bearer` header and to WebSocket URLs as a query parameter.

```
http://server:3000/login          # Login form (always accessible)
http://server:3000/?token=xxx     # Direct access with token in URL
```

> **Concept: Login mask**
> Instead of requiring users to manually append `?token=xxx` to URLs, the frontend detects missing authentication by probing `/api/config`. If the server returns 401, the browser redirects to `/login` where users enter their token. The token is then stored and automatically included in all subsequent requests. This provides a familiar login experience without changing the underlying token-based auth model.

> **Key Pattern: Auth detection via probe**
> The frontend calls an unprotected endpoint (`/api/config`) without a token. A 200 response means auth is disabled; a 401 means auth is required. This avoids hardcoding auth state and works correctly when the server config changes.
>
> ```javascript
> async function checkAuthRequired() {
>     try {
>         const res = await fetch('/api/config');
>         if (res.ok) return false;       // no auth needed
>         if (res.status === 401) return true; // auth required
>     } catch { }
>     return false;
> }
> ```

**Host agent connection:** The agent builds the WebSocket URL from the server address, supporting both plain `host:port` and full URLs with `ws://` or `wss://` scheme:
```go
func buildWebSocketURL(serverAddr string, authToken string) string {
    url := serverAddr
    if serverAddr != "" && !strings.HasPrefix(serverAddr, "ws://") && !strings.HasPrefix(serverAddr, "wss://") {
        url = "ws://" + serverAddr
    }
    if !strings.HasSuffix(url, "/") {
        url += "/"
    }
    url += "ws?type=host"
    if authToken != "" {
        url += fmt.Sprintf("&token=%s", authToken)
    }
    return url
}
```

> **Concept: Token-based authentication**
> A shared secret token is simpler than username/password or OAuth. There's no session management, no database, no user accounts. The client presents the token with every request, and the server validates it. This is well-suited for trusted LAN deployments where you just need a basic access barrier. For production use over untrusted networks, combine with TLS (HTTPS/WSS) to encrypt the token in transit.

> **Key Pattern: Optional middleware**
> The auth middleware checks if the token is empty before installing any validation logic. If empty, it returns a no-op handler that just calls `c.Next()`. This avoids branching throughout the codebase — every route goes through the same middleware, but it's a pass-through when auth is disabled.
>
> ```go
> func Middleware(token string) fiber.Handler {
>     if token == "" {
>         return func(c *fiber.Ctx) error {
>             return c.Next()
>         }
>     }
>     // ... validation logic
> }
> ```

**HTTP middleware** (`internal/relay/server.go`):
```go
app.Get("/health", func(c *fiber.Ctx) error {
    return c.SendString("OK")
})

app.Get("/login", s.serveLoginPage)  // exempt from auth

// Cache-busting for JS/CSS
app.Use(noCacheStaticMiddleware)

// Static files (CSS, JS, images, fonts) served without auth
app.Static("/", staticDir, fiber.Static{
    Next: func(c *fiber.Ctx) bool {
        return c.Path() == "/" || c.Path() == "/panel" || c.Path() == "/editor"
    },
})

// HTML page handlers (exempt from auth — templates only, no sensitive data)
app.Get("/", s.serveStartPage)
app.Get("/panel", s.servePanel)
app.Get("/editor", s.serveEditorUI)

app.Use(auth.Middleware(cfg.AuthToken))
```

**Host WebSocket validation** (`internal/relay/handler.go`):
```go
func (s *RelayServer) handleHostConn(c *ws.Conn) {
    token := c.Query("token", "")
    if !auth.ValidateToken(s.config.AuthToken, token) {
        c.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.ClosePolicyViolation, "unauthorized"))
        c.Close()
        return
    }
    // ... rest of host connection handling
}
```

> **Key Pattern: Dual token acceptance**
> The middleware accepts the token from two sources: query parameter (`?token=xxx`) or Authorization header (`Bearer xxx`). Query params are convenient for browser URLs and WebSocket connections. Bearer headers are cleaner for programmatic API access. The middleware checks the query param first, then falls back to the header.

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

Host connection handling enforces the 1:1 constraint and broadcasts the host's IP address to all browsers:

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

    hostIP := c.IP()
    slog.Info("Host agent connected", "ip", hostIP)
    s.broadcastToBrowsers(map[string]any{
        "type":      "log-event",
        "timestamp": time.Now().Format("2006-01-02 15:04:05"),
        "data":      "Host connected: " + hostIP,
    })
    // ... read loop with defer for disconnect broadcast
}
```

> **Key Pattern: Reject second connection**
> The mutex-protected check-and-set ensures only one host can be registered at a time. If a second host connects, it receives a WebSocket close message with a descriptive reason.

> **Key Pattern: IP address broadcast**
> The host's IP address is captured via `c.IP()` on both connect and disconnect, then broadcast as a `log-event` message. The start page (`static/index.html`) displays this in the connection log, and the panel client (`static/client/client.js`) shows a floating indicator with the host IP in the top-left corner.

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

The `Run()` method implements the auto-reconnect loop with WebSocket URL construction that supports both `ws://` and `wss://`:

```go
func (a *Agent) Run(serverAddr string) {
    backoff := time.Second
    maxBackoff := 30 * time.Second

    for {
        url := buildWebSocketURL(serverAddr, a.Config.AuthToken)
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

When the host agent disconnects, the server broadcasts a log event to all browsers that includes the host's IP address:

```go
defer func() {
    s.hostMu.Lock()
    s.hostConn = nil
    hostIP := c.IP()
    s.hostMu.Unlock()
    s.broadcastToBrowsers(map[string]any{
        "type":      "log-event",
        "timestamp": time.Now().Format("2006-01-02 15:04:05"),
        "data":      "Host disconnected: " + hostIP,
    })
}()
```

The browser's connection log shows this message, so users know the host is unavailable. The IP is also displayed as a floating indicator on the panel client UI (`static/client/client.js`), appearing briefly when the host connects and automatically fading out after 5 seconds.

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
- Default and connect modes include a system tray icon (when a display server is detected) with fullscreen toggle and exit controls
- System tray is skipped on headless systems (no `DISPLAY` or `WAYLAND_DISPLAY` env vars on Linux)
- In connect mode, the tray exit callback sends to a shared `quit` channel to trigger graceful shutdown
- WebSocket URL supports plain `host:port` (defaults to `ws://`) or full URLs (`ws://` or `wss://`)
- Use `wss://` when the relay server is behind a TLS-terminating reverse proxy
- 1:1 host connection — second host is rejected with a close message
- Host IP address is captured via `c.IP()` and broadcast on connect/disconnect as `log-event` messages
- The panel client UI shows a floating host IP indicator in the top-left corner that auto-dismisses after 5 seconds
- Auto-reconnect with exponential backoff (1s → 2s → 4s → max 30s)
- Heartbeat every 15s keeps the connection alive
- `AppStateInterface` allows the same WebSocket handler to work in both modes
- Server address configurable via CLI arg, config file, or environment variable
- Panel files, themes, and blocks live on the central server
- Token-based authentication (`auth_token` in config.json or `OMNIPANEL_AUTH_TOKEN` env var) protects serve/connect modes
- Auth middleware accepts token via query parameter (`?token=xxx`) or Authorization header (`Bearer xxx`)
- Host WebSocket connections validate token from query parameter (`?type=host&token=xxx`)
- Empty `auth_token` disables authentication (backward compatible, default mode unaffected)
- The `/login` page is exempt from auth middleware, providing a login form for token entry
- `/favicon.ico` returns 204 to prevent browser 401 errors
- CSS, JS, images, fonts, block templates (/blocks/), theme CSS (/themes/), and user assets (/assets/) are served without authentication so pages can load and execute JavaScript
- HTML pages (/, /panel, /editor) are served without auth — they are templates only; sensitive data is protected at the API level
- Frontend detects auth requirement by probing `/api/config` (401 = auth needed)
- Token is stored in localStorage (persistent) or sessionStorage (tab-only) based on user choice
- All subsequent API requests include the token via Authorization headers and WebSocket URL params
- The relay server can run as a Docker container — see [Chapter 21](21-container-build.md) for the ko-based container build

[← Back: Chapter 19](19-windows-build-and-ci.md) · [Next: Chapter 21 →](21-container-build.md)
