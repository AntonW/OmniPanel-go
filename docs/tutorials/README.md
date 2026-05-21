# OmniPanel-go Tutorials

A guided tour through the OmniPanel-go codebase, written for developers who are new to Go (and web development). Each chapter explains the concepts first, then walks through the actual code.

## Part A: Go Backend Fundamentals

| Chapter | Topic | File |
|---------|-------|------|
| [1](01-project-overview.md) | Project Overview & Entry Point | `main.go` |
| [2](02-configuration.md) | Configuration System | `internal/config/config.go` |
| [3](03-state-and-concurrency.md) | Application State & Concurrency | `internal/state/state.go` |
| [4](04-databus.md) | DataBus & System Metrics | `internal/databus/databus.go`, `collect_*.go` |

## Part B: HTTP, WebSocket & Commands

| Chapter | Topic | File |
|---------|-------|------|
| [5](05-http-routing.md) | HTTP Routing & REST API | `internal/routes/` |
| [6](06-websocket.md) | WebSocket Real-time Communication | `internal/websocket/`, `internal/routes/router.go` |
| [7](07-commands.md) | Command Execution | `internal/commands/commands.go` |

## Part C: Platform-Specific Code

| Chapter | Topic | File |
|---------|-------|------|
| [8](08-virtual-input.md) | Virtual Input Devices (Linux/Windows) | `internal/devices/` |

## Part D: Speech Recognition

| Chapter | Topic | File |
|---------|-------|------|
| [9](09-speech-overview.md) | Speech Recognition Overview & Grammar Constraints | `internal/speech/speech.go` |
| [10](10-speech-engines.md) | STT Engines: Vosk (grammar-constrained) & llama-cpp | `internal/speech/vosk.go`, `llama.go` |
| [11](11-speech-matching.md) | Phrase Matching & Security | `internal/speech/matcher.go` |
| [12](12-audio-recording.md) | Audio Recording: Client & Host | `internal/speech/recorder.go`, `static/client/client.js` |

## Part E: Web Frontend

| Chapter | Topic | File |
|---------|-------|------|
| [13](13-panel-ui.md) | The Panel UI (client) | `static/client/` |
| [14](14-editor-ui.md) | The Editor UI | `static/editor/` |
| [15](15-host-ui.md) | The Start Page (panel list, host controls, log) | `static/index.html` |

## Part F: Architecture & Data Flow

| Chapter | Topic |
|---------|-------|
| [16](16-data-flow.md) | End-to-End Data Flow |
| [20](20-distributed-deployment.md) | Distributed Deployment (Server + Host Agent) |

## Part G: Platform Integrations

| Chapter | Topic | File |
|---------|-------|------|
| [17](17-mpris.md) | MPRIS Media Player Integration (Linux D-Bus) | `internal/mpris/mpris.go`, `internal/routes/mpris.go` |
| [18](18-rss-feed.md) | RSS Feed Integration (Polling, WebSocket Push, Host URL Opening) | `internal/rssfeed/`, `static/client/client.js` |
| [19](19-windows-build-and-ci.md) | Windows Build Script and CI Alignment | `scripts/build-with-vosk.ps1`, `.forgejo/workflows/ci.yml` |

## How to Use These Tutorials

- Read chapters in order — each builds on concepts from the previous ones
- Code excerpts are taken directly from the project; line numbers reference the original files
- "Concept" boxes explain Go (or JavaScript) fundamentals for novices
- "Key Pattern" callouts highlight idiomatic techniques worth remembering
