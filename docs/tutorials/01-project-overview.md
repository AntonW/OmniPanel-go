# Chapter 1: Project Overview & Entry Point

## What is OmniPanel-go?

OmniPanel-go is a web-based control panel system that runs on Linux or Windows. It lets you:

- Design custom control panels (dashboards) with draggable blocks
- Display them on any device with a web browser (tablet, phone, PC)
- Simulate virtual joystick and mouse input on the host machine
- Execute shell commands and HTTP requests from panel buttons
- Monitor system metrics (CPU, memory, disk, network) in real time
- Control your panel with voice commands using speech recognition

## Platform Support

| Feature | Linux | Windows | macOS/BSD |
|---------|-------|---------|-----------|
| Web Server | Yes | Yes | Yes |
| Virtual Joystick | Yes (uinput) | Yes (vJoy driver required) | No |
| Virtual Mouse | Yes (uinput) | Yes (SendInput API) | No |
| Shell Commands | Yes | Yes (cmd.exe) | Yes |
| HTTP Commands | Yes | Yes | Yes |
| DataBus Metrics | Yes | Yes | Yes |
| Speech Recognition (Vosk) | Yes (CGO) | Yes (CGO, DLL required) | Yes (CGO, dylib required) |
| Speech Recognition (llama-cpp) | Yes | Yes | Yes |
| Client Audio Recording | Yes | Yes | Yes |
| Host Audio Recording | Yes (CGO) | Yes (CGO) | Yes (CGO) |

> **CGO note:** Speech features (Vosk STT and host microphone recording) require CGO. The core server, virtual input devices (Linux), and client recording work without CGO. Container builds use `CGO_ENABLED=0` for fully static binaries — speech is unavailable in container images.

> **Windows Requirements:** Virtual joystick simulation requires the [vJoy driver](https://github.com/BrunnerInnovation/vJoy/releases) (v2.2.2.0 or later) to be installed. Virtual mouse uses the built-in Windows SendInput API and requires no additional drivers.

## Architecture at a Glance

### Default Mode (traditional)

```
┌─────────────────────────────────────────────────┐
│                   Go Server                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Fiber   │  │WebSocket │  │  DataBus      │  │
│  │  Router  │  │ Handler  │  │  (metrics)    │  │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       │              │                │          │
│  ┌────┴──────────────┴────────────────┴───────┐  │
│  │              AppState (hub)                 │  │
│  └────┬──────────────┬────────────────┬───────┘  │
│       │              │                │          │
│  ┌────┴─────┐  ┌─────┴──────┐  ┌─────┴──────┐   │
│  │ Joystick │  │  Mousepad  │  │  Config    │   │
│  │ Manager  │  │  Manager   │  │  Manager   │   │
│  └──────────┘  └────────────┘  └────────────┘   │
│       │                                          │
│  ┌────┴──────┐                                   │
│  │  Speech   │                                   │
│  │  Manager  │                                   │
│  └──────────┘                                   │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │  System Tray (default mode, with display)  │  │
│  │  - Open Panel (localhost:port)             │  │
│  │  - Toggle Fullscreen (broadcast)           │  │
│  │  - Exit Application (SIGTERM)              │  │
│  └────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
           │                │
           ▼                ▼
┌─────────────────┐ ┌─────────────────┐
│ Linux: uinput   │ │ Linux: uinput   │
│ Windows: vJoy   │ │ Windows:SendIn. │
│ (virtual joy)   │ │ (virtual mouse) │
└─────────────────┘ └─────────────────┘

Web Clients (browser):
┌──────────────┐  ┌──────────┐  ┌──────────┐
│  Start Page  │  │  Panel   │  │  Editor  │
│  / (host     │  │ (client) │  │          │
│  controls)   │  └──────────┘  └──────────┘
└──────────────┘
```

### Distributed Mode (serve + connect)

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
│      Open Panel (remote URL), Toggle Fullscreen, Exit Application    │
└─────────────────────────────────────────────────┘
```

The distributed mode enables deploying the WebUI on a central server (always-on machine, cloud instance) while the host agent runs on a machine behind a firewall. The host initiates an outbound WebSocket connection to the server, so no inbound ports need to be opened on the host.

## Project Structure

```
OmniPanel-go/
├── main.go                      # Application entry point (3 modes: default, serve, connect)
├── config.json                  # Runtime configuration
├── Makefile                     # Container build targets (ko)
├── .ko.yaml                     # ko build configuration
├── internal/
│   ├── config/config.go         # Config loading/saving
│   ├── logger/logger.go         # Structured logging (color, text, JSON)
│   ├── state/state.go           # Central application state + AppStateInterface
│   ├── starter/starter.go       # Starter file initialization for Docker containers
│   ├── agent/agent.go           # Host agent for distributed deployment (connect mode)
│   ├── relay/                   # Central relay server for distributed deployment (serve mode)
│   │   ├── server.go            # Fiber HTTP server + WebSocket hub
│   │   └── handler.go           # WebSocket message routing (browser ↔ host)
│   ├── databus/databus.go       # System metrics collection
│   ├── routes/                  # HTTP & WebSocket routes
│   │   ├── router.go
│   │   ├── editor.go
│   │   └── panel.go
│   ├── websocket/handler.go     # WebSocket message handling
│   ├── commands/commands.go     # Shell & HTTP command execution
│   ├── speech/                  # Speech recognition system
│   │   ├── speech.go            # SpeechManager, STTEngine interface
│   │   ├── vosk.go              # Offline Vosk STT backend (+build cgo)
│   │   ├── vosk_stub.go         # Vosk stub for non-CGO builds (+build !cgo)
│   │   ├── llama.go             # llama-cpp-server HTTP client
│   │   ├── matcher.go           # Phrase matching + allowlist
│   │   ├── recorder.go          # Host microphone recording (+build cgo)
│   │   ├── recorder_stub.go     # Recorder stub for non-CGO builds (+build !cgo)
│   │   ├── decoder.go           # Audio format conversion
│   │   └── download.go          # Vosk model auto-download
│   └── devices/                 # Virtual input device creation
│       ├── virtual_input.go     # Interfaces + managers (platform-agnostic)
│       ├── linux.go             # Linux joystick (uinput)
│       ├── mousepad_linux.go    # Linux mouse (uinput)
│       ├── windows.go           # Windows joystick (vJoy)
│       ├── mousepad_windows.go  # Windows mouse (SendInput)
│       └── stub.go              # Stub for unsupported platforms
├── static/                      # Web frontend files
│   ├── client/                  # Panel display UI
│   └── editor/                  # Panel editor UI
├── user/                        # User-created content (runtime)
│   ├── panels/                  # Saved panel JSON definitions
│   ├── blocks/                  # Reusable HTML block components
│   ├── assets/                  # Images and other assets
│   ├── themes/                  # Theme CSS files
│   └── speech_commands.json     # Voice command definitions
├── starter/                     # Starter files for Docker containers (build-time copy of user/)
│   ├── panels/                  # Default panels for first-run setup
│   ├── blocks/                  # Default block templates
│   ├── assets/                  # Default assets
│   ├── themes/                  # Default themes
│   └── speech_commands.json     # Default speech commands
└── internal/
    └── systray/                 # System tray icon (fyne.io/systray)
        ├── systray.go           # Tray package with Open Panel, fullscreen toggle, and exit
        └── icon.png             # Embedded tray icon (64x64 PNG from SVG logo)
```

## The Entry Point: `main.go`

OmniPanel-go supports three deployment modes, selected via subcommand:

| Mode | Command | Description |
|------|---------|-------------|
| Default | `./omnipanel-go` | HTTP server + subsystems (traditional, unchanged) |
| Serve | `./omnipanel-go serve` | Central relay server (WebUI + WebSocket hub) |
| Connect | `./omnipanel-go connect <addr>` | Host agent (WebSocket client + subsystems). `<addr>` accepts `host:port` (ws://) or full URL (`wss://`) |

Let's walk through the entry point structure.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "omnipanel-go/internal/agent"
    "omnipanel-go/internal/config"
    "omnipanel-go/internal/logger"
    "omnipanel-go/internal/relay"
    "omnipanel-go/internal/routes"
    "omnipanel-go/internal/starter"
    "omnipanel-go/internal/state"
)
```

Every Go program starts with a `package` declaration. The `main` package is special — it produces an executable. The `import` block lists external and internal dependencies. The new `agent` and `relay` packages support distributed deployment, and `starter` handles first-run file initialization for Docker containers.

> **Concept: Go imports**
> Go imports are paths, not names. `"omnipanel-go/internal/config"` maps to the directory `internal/config/`. The package name used in code (`config`) comes from the `package config` declaration inside that directory's `.go` files.

### Step 1: Subcommand Parsing

```go
func main() {
    subcommand := ""
    serverAddrArg := ""
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "serve":
            subcommand = "serve"
        case "connect":
            subcommand = "connect"
            if len(os.Args) > 2 {
                serverAddrArg = os.Args[2]
            }
        }
    }
```

> **Concept: Subcommand parsing without a library**
> For simple subcommand parsing, checking `os.Args[1]` directly is sufficient. The `connect` subcommand accepts an optional second argument (the server address). If no address is provided, it falls back to `config.ServerAddress`.

> **Key Pattern: CLI arg → Config → Error fallback**
> The server address resolution follows a common Go pattern: command-line argument takes priority, then config file value, then an error if neither is set. This gives users flexibility in how they configure the connection.

### Step 2: Logging and Config Discovery

```go
    logFormat := flag.String("log-format", "", "log format: color (default), text, json")
    flag.Parse()

    format := *logFormat
    if format == "" {
        format = os.Getenv("LOG_FORMAT")
    }
    logger.Init(format)

    configPath := config.FindConfigPath()
    userPath := config.FindUserPath()
    baseDir, _ := os.Getwd()
```

`main()` is the function that runs when you execute the binary. We use `flag` (Go's standard command-line flag parser) to read `--log-format`, falling back to the `LOG_FORMAT` environment variable if no flag is provided. `logger.Init()` configures the global `slog` logger with the chosen format.

> **Concept: The `flag` package**
> `flag.String("name", default, "usage")` registers a string flag and returns a pointer to its value. `flag.Parse()` processes `os.Args[1:]` and populates all registered flags. This is Go's simplest way to accept command-line arguments.

> **Key Pattern: Flag → Env var → Default fallback**
> Reading a flag first, then an environment variable, then falling back to a default is a common Go pattern. It gives users flexibility: they can set `LOG_FORMAT=json` in their shell for all invocations, or override it per-run with `--log-format=color`.

OmniPanel-go supports three log formats:

| Format | Output | Best for |
|--------|--------|----------|
| `color` (default) | Colored text with timestamps and source info | Development, interactive terminal use |
| `text` | Plain key=value text with source info | Piped output, simple log files |
| `json` | Structured JSON lines | Log aggregation systems (Loki, ELK, etc.) |

The colored output uses the [`tint`](https://github.com/lmittmann/tint) package, which wraps slog's handler interface with ANSI color codes.

### Step 3: Loading Configuration

```go
    cfg, err := config.Load(configPath)
    if err != nil {
        slog.Warn("Failed to load config, using defaults", "error", err)
        cfg = &config.Config{
            Port:         3000,
            NumJoysticks: 4,
        }
    }
```

> **Concept: Error handling in Go**
> Go doesn't have exceptions. Functions return errors as values. The idiomatic pattern is `value, err := doSomething()` followed by `if err != nil { ... }`. This makes error flow explicit and easy to follow.

If config loading fails, we fall back to sensible defaults. `&config.Config{...}` creates a pointer to a new `Config` struct.

### Step 4: Running the Selected Mode

```go
    switch subcommand {
    case "serve":
        runServe(cfg, userPath, baseDir)
    case "connect":
        runConnect(cfg, configPath, userPath, baseDir, serverAddrArg)
    default:
        runDefault(cfg, configPath, userPath, baseDir)
    }
```

The three mode functions are:

- `runDefault()` — creates `AppState`, builds HTTP router, starts server (unchanged behavior)
- `runServe()` — creates `RelayServer`, starts HTTP + WebSocket relay hub
- `runConnect()` — creates `Agent`, connects to relay server with auto-reconnect

### Default Mode: `runDefault()`

```go
func runDefault(cfg *config.Config, configPath, userPath, baseDir string) {
    appState := state.New(cfg, configPath, userPath, baseDir)
    defer appState.Close()

    app := routes.NewRouter(appState)

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    if !systray.IsHeadless() {
        fullscreenToggle := false
        tray := systray.New(
            func() {
                fullscreenToggle = !fullscreenToggle
                if fullscreenToggle {
                    appState.BroadcastJSON(map[string]any{"type": "enter-fullscreen"})
                } else {
                    appState.BroadcastJSON(map[string]any{"type": "exit-fullscreen"})
                }
            },
            func() {
                rssfeed.OpenURL(fmt.Sprintf("http://localhost:%d", cfg.Port))
            },
            func() {
                quit <- syscall.SIGTERM
            },
        )
        go tray.Run()
        defer tray.Quit()
    }

    addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)

    go func() {
        if err := app.Listen(addr); err != nil {
            slog.Error("Server failed", "error", err)
            os.Exit(1)
        }
    }()

    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    app.ShutdownWithContext(ctx)
}
```

This is the traditional mode: HTTP server + all subsystems on the same machine. A system tray icon is created when a desktop session is available. On Linux/Unix, this means a display server (X11 or Wayland) is detected. On Windows and macOS, tray support is assumed by default. The tray provides Open Panel (opens http://localhost:port in the default browser), fullscreen toggle, and exit controls. The `quit` channel is shared between signal handling and the tray's exit callback, so either Ctrl+C or "Exit Application" from the tray triggers the same graceful shutdown path.

> **Concept (Go): `systray.IsHeadless()`**
> In `internal/systray/systray.go`, `IsHeadless()` is platform-aware. On Linux/Unix, it checks `DISPLAY` and `WAYLAND_DISPLAY`; if both are empty (for example on a headless server or SSH session without X forwarding), the tray is skipped. On Windows and macOS, it returns `false` so tray initialization is attempted by default.
>
> **Concept (JavaScript): Fullscreen is handled by the panel client**
> In `static/client/client.js`, the panel listens for WebSocket messages `enter-fullscreen` and `exit-fullscreen`. Those messages can be triggered by either start-page buttons or tray actions, but the browser client only needs to react to message type.
>
> **Key Pattern (JavaScript): Transport event over UI source**
> `socket.onmessage` in `static/client/client.js` routes behavior by `msg.type` instead of by where the action originated. This keeps fullscreen behavior consistent across backend triggers (UI buttons and tray menu).

### Serve Mode: `runServe()`

```go
func runServe(cfg *config.Config, userPath, baseDir string) {
    slog.Info("Starting OmniPanel-go relay server", "port", cfg.Port)

    starter.Init(userPath)

    srv := relay.New(cfg, userPath, baseDir)

    addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)

    go func() {
        if err := srv.Listen(addr); err != nil {
            slog.Error("Server failed", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

The `starter.Init(userPath)` call copies default user content (blocks, panels, themes, assets) into the mounted volume if it is empty. This enables a zero-config first run for Docker containers. On subsequent runs, when the volume already contains user files, `Init` does nothing.

> **Key Pattern: First-run initialization**
> `starter.Init` checks if the target directory exists and is empty. If it doesn't exist, it creates it. If it's empty, it copies all starter files. If it already has content, it returns immediately. This pattern is common for containerized applications that need to seed a persistent volume with default data without overwriting user changes.

The relay server serves the WebUI and manages WebSocket connections between browsers and a single host agent. No subsystems are created — the server only relays messages.

### Connect Mode: `runConnect()`

```go
func runConnect(cfg *config.Config, configPath, userPath, baseDir, serverAddrArg string) {
    serverAddr := serverAddrArg
    if serverAddr == "" {
        serverAddr = cfg.ServerAddress
    }
    if serverAddr == "" {
        slog.Error("No server address provided...")
        os.Exit(1)
    }

    agt := agent.New(cfg, configPath, userPath, baseDir)
    defer agt.Close()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    if !systray.IsHeadless() {
        fullscreenToggle := false
        tray := systray.New(
            func() {
                fullscreenToggle = !fullscreenToggle
                if fullscreenToggle {
                    agt.BroadcastJSON(map[string]any{"type": "enter-fullscreen"})
                } else {
                    agt.BroadcastJSON(map[string]any{"type": "exit-fullscreen"})
                }
            },
            func() {
                rssfeed.OpenURL(serverAddrToHTTP(serverAddr))
            },
            func() {
                quit <- syscall.SIGTERM
            },
        )
        go tray.Run()
        defer tray.Quit()
    }

    go func() {
        agt.Run(serverAddr)
    }()

    <-quit
    agt.Stop()
}
```

The host agent creates all subsystems (joystick, speech, MPRIS, RSS, etc.) but no HTTP server. It connects to the relay server via WebSocket and auto-reconnects on disconnect with exponential backoff. Like default mode, a system tray icon is created when a desktop session is available (Linux/Unix requires `DISPLAY` or `WAYLAND_DISPLAY`; Windows and macOS try tray startup by default). The Open Panel menu item converts the WebSocket server address to an HTTP URL (ws:// → http://, wss:// → https://) and opens it in the default browser. A shared `quit` channel receives both OS signals and the tray's exit callback, so either Ctrl+C or "Exit Application" triggers the same shutdown path calling `agt.Stop()`.

## Key Takeaways

- `main.go` supports three deployment modes: default (traditional), serve (relay server), connect (host agent)
- Subcommand parsing uses simple `os.Args` checks — no external library needed
- Goroutines + channels form Go's concurrency model ("share memory by communicating")
- `defer` ensures cleanup happens even if the program exits early
- Structured logging with `slog` supports color, text, and JSON output via the `logger` package
- Command-line flags (`flag` package) and environment variables provide flexible configuration
- Graceful shutdown uses a 5-second context timeout and diagnostic logging to identify hangs
- The `agent` package runs all subsystems without an HTTP server, connecting via WebSocket
- The `relay` package serves the WebUI and relays messages between browsers and a single host
- The `starter` package copies default user content into an empty volume on first run (Docker containers)
- `AppStateInterface` allows the same WebSocket handler code to work in both default and connect modes
- Speech features (Vosk STT, host recording) require CGO and are excluded from container builds
- The `systray` package provides a cross-platform system tray icon (using `fyne.io/systray`) with Open Panel, fullscreen toggle, and exit controls in default and connect modes
- System tray is skipped when no desktop session is available (Linux/Unix: no `DISPLAY` and no `WAYLAND_DISPLAY`; Windows/macOS: tray is attempted by default)
- In default mode, Open Panel opens `http://localhost:<port>`; in connect mode, it converts the WebSocket URL to HTTP (ws:// → http://, wss:// → https://) and opens the remote server
- A shared `quit` channel handles both OS signals and tray exit callbacks in default and connect modes

[Next: Chapter 2 — Configuration System →](02-configuration.md)
