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

## Project Structure

```
OmniPanel-go/
├── main.go                      # Application entry point
├── config.json                  # Runtime configuration
├── internal/
│   ├── config/config.go         # Config loading/saving
│   ├── logger/logger.go         # Structured logging (color, text, JSON)
│   ├── state/state.go           # Central application state
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

Let's walk through the entire `main.go` file (61 lines).

```go
package main

import (
    "flag"
    "fmt"
    "log/slog"
    "os"
    "os/signal"

    "omnipanel-go/internal/config"
    "omnipanel-go/internal/logger"
    "omnipanel-go/internal/routes"
    "omnipanel-go/internal/state"
)
```

Every Go program starts with a `package` declaration. The `main` package is special — it produces an executable. The `import` block lists external and internal dependencies.

> **Concept: Go imports**
> Go imports are paths, not names. `"omnipanel-go/internal/config"` maps to the directory `internal/config/`. The package name used in code (`config`) comes from the `package config` declaration inside that directory's `.go` files.

### Step 1: Logging and Config Discovery

```go
func main() {
    logFormat := flag.String("log-format", "", "log format: color (default), text, json")
    flag.Parse()

    format := *logFormat
    if format == "" {
        format = os.Getenv("LOG_FORMAT")
    }
    logger.Init(format)

    slog.Info("Starting OmniPanel-go server", "log_format", format)

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

### Step 2: Loading Configuration

```go
    cfg, err := config.Load(configPath)
    if err != nil {
        slog.Warn("Failed to load config, using defaults", "error", err)
        cfg = &config.Config{
            Panel:        "default",
            Port:         3000,
            NumJoysticks: 4,
        }
    }
```

> **Concept: Error handling in Go**
> Go doesn't have exceptions. Functions return errors as values. The idiomatic pattern is `value, err := doSomething()` followed by `if err != nil { ... }`. This makes error flow explicit and easy to follow.

If config loading fails, we fall back to sensible defaults. `&config.Config{...}` creates a pointer to a new `Config` struct.

### Step 3: Creating Application State

```go
    appState := state.New(cfg, configPath, userPath, baseDir)
    defer appState.Close()
```

`state.New()` creates the central `AppState` object that holds everything: config, joystick managers, the databus, and WebSocket clients.

> **Concept: `defer`**
> `defer` schedules a function call to run when the surrounding function returns. It's commonly used for cleanup (closing files, releasing locks, etc.). Multiple `defer` calls execute in LIFO order.

### Step 4: Building the Router and Starting the Server

```go
    app := routes.NewRouter(appState)

    addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
    slog.Info("Server starting", "addr", addr)

    go func() {
        if err := app.Listen(addr); err != nil {
            slog.Error("Server failed", "error", err)
            os.Exit(1)
        }
    }()
```

`routes.NewRouter()` builds the HTTP server with all routes registered.

> **Concept: Goroutines**
> `go func() { ... }()` launches an anonymous function as a **goroutine** — a lightweight concurrent thread managed by the Go runtime. Without `go`, `app.Listen()` would block forever and the rest of `main()` would never run.

The server listens on `0.0.0.0` (all network interfaces) so devices on the same network can connect.

### Step 5: Graceful Shutdown

```go
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit

    slog.Info("Shutting down...")
}
```

> **Concept: Channels**
> A channel is a communication pipe between goroutines. `make(chan os.Signal, 1)` creates a buffered channel (capacity 1). `signal.Notify` tells the OS to send interrupt signals into this channel. `<-quit` **receives** from the channel — it blocks until a signal arrives.

When you press Ctrl+C (or the system sends an interrupt signal), the program unblocks, logs "Shutting down...", and exits. The `defer appState.Close()` runs automatically, cleaning up virtual devices.

> **Cross-Platform Note:** Earlier versions used `syscall.SIGINT` and `syscall.SIGTERM`, which don't exist on Windows. Using `os.Interrupt` works on all platforms — it maps to SIGINT/SIGTERM on Unix and Ctrl+C/Close events on Windows.

## Key Takeaways

- `main.go` is short because responsibilities are delegated to packages
- Goroutines + channels form Go's concurrency model ("share memory by communicating")
- `defer` ensures cleanup happens even if the program exits early
- Structured logging with `slog` supports color, text, and JSON output via the `logger` package
- Command-line flags (`flag` package) and environment variables provide flexible configuration

[Next: Chapter 2 — Configuration System →](02-configuration.md)
