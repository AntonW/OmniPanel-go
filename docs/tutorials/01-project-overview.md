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

The distributed mode enables deploying the WebUI on a central server (always-on machine, cloud instance) while the host agent runs on a machine behind a firewall. The host initiates an outbound WebSocket connection to the server, so no inbound ports need to be opened on the host.

## Project Structure

```
OmniPanel-go/
├── main.go                      # Application entry point (3 modes: default, serve, connect)
├── config.json                  # Runtime configuration
├── internal/
│   ├── config/config.go         # Config loading/saving
│   ├── logger/logger.go         # Structured logging (color, text, JSON)
│   ├── state/state.go           # Central application state + AppStateInterface
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
│   │   ├── vosk.go              # Offline Vosk STT backend
│   │   ├── llama.go             # llama-cpp-server HTTP client
│   │   ├── matcher.go           # Phrase matching + allowlist
│   │   ├── recorder.go          # Host microphone recording
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
└── user/                        # User-created content
    ├── panels/                  # Saved panel JSON definitions
    ├── blocks/                  # Reusable HTML block components
    ├── assets/                  # Images and other assets
    └── speech_commands.json     # Voice command definitions
```

## The Entry Point: `main.go`

OmniPanel-go supports three deployment modes, selected via subcommand:

| Mode | Command | Description |
|------|---------|-------------|
| Default | `./omnipanel-go` | HTTP server + subsystems (traditional, unchanged) |
| Serve | `./omnipanel-go serve` | Central relay server (WebUI + WebSocket hub) |
| Connect | `./omnipanel-go connect <addr>` | Host agent (WebSocket client + subsystems) |

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
    "omnipanel-go/internal/state"
)
```

Every Go program starts with a `package` declaration. The `main` package is special — it produces an executable. The `import` block lists external and internal dependencies. The new `agent` and `relay` packages support distributed deployment.

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

    addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)

    go func() {
        if err := app.Listen(addr); err != nil {
            slog.Error("Server failed", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    app.ShutdownWithContext(ctx)
}
```

This is the traditional mode: HTTP server + all subsystems on the same machine.

### Serve Mode: `runServe()`

```go
func runServe(cfg *config.Config, userPath, baseDir string) {
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

    go func() {
        agt.Run(serverAddr)
    }()

    agt.WaitSignal()
    agt.Stop()
}
```

The host agent creates all subsystems (joystick, speech, MPRIS, RSS, etc.) but no HTTP server. It connects to the relay server via WebSocket and auto-reconnects on disconnect with exponential backoff.

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
- `AppStateInterface` allows the same WebSocket handler code to work in both default and connect modes

[Next: Chapter 2 — Configuration System →](02-configuration.md)
