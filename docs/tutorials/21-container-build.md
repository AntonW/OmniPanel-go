# Chapter 21: Container Build with ko (serve mode)

## What This Chapter Covers

OmniPanel-go's `serve` mode can be built as a Docker container using [ko](https://github.com/ko-build/ko). This enables deploying the relay server as a containerized service with minimal image size and no Dockerfile needed.

The container build:
- Uses `CGO_ENABLED=0` for a fully static Go binary
- Embeds `static/` (WebUI files) and `starter/` (default user content) via ko's kodata mechanism
- Runs on `gcr.io/distroless/static-debian12` (~2MB base image)
- Automatically populates an empty mounted `user/` volume with starter files on first run

> **Note:** Speech features (Vosk STT, host microphone recording) require CGO and are **not available** in the container image. Use llama-cpp-server (HTTP API) for speech recognition with the container, or run a native build with CGO enabled.

## How ko Works

ko builds Go binaries into container images without a Dockerfile. It:
1. Compiles the Go binary for the target platform
2. Creates a minimal container image with the binary
3. Embeds files from `kodata/` into the image at `/var/run/ko/`
4. Sets the entrypoint to the compiled binary

```
┌─────────────────────────────────────────────────┐
│  ko build process                               │
│                                                 │
│  kodata/static/  ──────────────────► /var/run/ko/static/
│  kodata/config.json ───────────────► /var/run/ko/config.json
│  kodata/starter/ ──────────────────► /var/run/ko/starter/
│                                                 │
│  Go binary ───────────────────────► /ko-app/omnipanel-go
│                                                 │
│  Base: gcr.io/distroless/static-debian12        │
└─────────────────────────────────────────────────┘
```

> **Concept: ko's kodata mechanism**
> ko automatically copies everything in `kodata/` into the container image at `/var/run/ko/`. This is ko's equivalent of Docker's `COPY` instruction but driven by convention. The environment variable `KO_DATA_PATH` is set to `/var/run/ko/` at runtime, so the application can locate embedded files.

## Build Configuration

### `.ko.yaml`

```yaml
defaultBaseImage: gcr.io/distroless/static-debian12
builds:
  - id: omnipanel-go
    main: .
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -s -w
```

| Field | Purpose |
|-------|---------|
| `defaultBaseImage` | Minimal distroless image (~2MB, no shell) |
| `CGO_ENABLED=0` | Fully static binary, no C dependencies |
| `-trimpath` | Removes local file paths from the binary |
| `-s -w` | Strips debug info and symbol table (smaller binary) |

> **Concept: Distroless images**
> Distroless images contain only the application and its runtime dependencies — no package manager, no shell, no extra tools. This reduces the attack surface and image size. The tradeoff: you cannot `docker exec` into the container for debugging. Use `docker logs` for diagnostics instead.

### `Makefile`

The Makefile provides two targets:

| Target | Purpose |
|--------|---------|
| `make build-dev` | Build and load into local Docker daemon |
| `make build` | Build and push to a container registry |

```makefile
build-dev:
	rm -rf kodata/ starter/ && \
	mkdir -p starter kodata && \
	cp -rav user/blocks starter/ && \
	cp -rav user/panels starter/ && \
	cp -rav user/themes starter/ && \
	cp -rav user/assets starter/ && \
	cp user/speech_commands.json starter/ && \
	mkdir -p starter/speech-models && touch starter/speech-models/.gitkeep && \
	cp -rav static/ kodata/static && \
	cp config.json kodata/config.json && \
	cp -rav starter/ kodata/starter && \
	ko build -B --local
```

The build process:
1. Creates `starter/` from `user/` at build time (no duplication in git)
2. Copies `static/` and `config.json` into `kodata/`
3. Copies `starter/` into `kodata/`
4. Runs `ko build` to create the container image

> **Key Pattern: Build-time file preparation**
> Instead of committing a duplicate `starter/` directory to git, the Makefile creates it from `user/` during the build. This keeps the repository clean while ensuring the container always ships with the latest user content. The `starter/` and `kodata/` directories are listed in `.gitignore`.

## Starter File Initialization

When the container starts with an empty mounted `user/` volume, the `starter` package copies default files into it:

```go
// internal/starter/starter.go
func Init(userPath string) {
    starterDir := findStarterDir()
    if starterDir == "" {
        return
    }

    empty, err := isDirEmpty(userPath)
    if err != nil {
        // Directory doesn't exist — create it
        os.MkdirAll(userPath, 0755)
        empty = true
    }

    if !empty {
        return  // Existing data — don't overwrite
    }

    copyDir(starterDir, userPath)
}
```

The starter directory is searched in order:
1. `$KO_DATA_PATH/starter` (ko container runtime)
2. `/var/run/ko/starter` (ko default mount point)
3. `starter` (local development)
4. `./starter` (local development, relative)

> **Key Pattern: First-run seeding**
> The `isDirEmpty` check ensures starter files are only copied when the volume is truly empty. This preserves user data across container restarts and upgrades. The pattern is: check → seed if empty → proceed.

## Running the Container

### Basic usage

```bash
docker run --rm -p 3000:3000 \
  -v $(pwd)/user:/var/run/ko/user \
  -w /var/run/ko \
  ko.local/omnipanel-go:latest serve
```

| Flag | Purpose |
|------|---------|
| `-p 3000:3000` | Expose the relay server port |
| `-v $(pwd)/user:/var/run/ko/user` | Mount persistent user data volume |
| `-w /var/run/ko` | Set working directory so `os.Getwd()` resolves correctly |
| `serve` | Start in relay server mode |

### With custom port

```bash
docker run --rm -p 8080:8080 \
  -v $(pwd)/user:/var/run/ko/user \
  -w /var/run/ko \
  -e OMNIPANEL_PORT=8080 \
  ko.local/omnipanel-go:latest serve
```

### With custom config

```bash
docker run --rm -p 3000:3000 \
  -v $(pwd)/user:/var/run/ko/user \
  -v $(pwd)/config.json:/var/run/ko/config.json:ro \
  -w /var/run/ko \
  ko.local/omnipanel-go:latest serve
```

## CGO Build Tags and Speech

The speech package uses Go build tags to provide stub implementations when CGO is disabled:

| File | Build Tag | Purpose |
|------|-----------|---------|
| `internal/speech/recorder.go` | `//go:build cgo` | Host microphone recording (malgo) |
| `internal/speech/recorder_stub.go` | `//go:build !cgo` | Stub — returns "requires CGO" error |
| `internal/speech/vosk.go` | `//go:build cgo` | Vosk STT engine |
| `internal/speech/vosk_stub.go` | `//go:build !cgo` | Stub — returns "requires CGO" error |

> **Concept: Go build tags**
> Build tags are special comments at the top of Go files that control which files are included in a build. `//go:build cgo` includes the file only when CGO is enabled. `//go:build !cgo` includes the file only when CGO is disabled. This pattern lets you provide platform-specific or feature-specific implementations while keeping the same public interface.

When the container runs with speech enabled but CGO is unavailable, the SpeechManager will log errors when attempting to use Vosk or host recording. Client-side recording and llama-cpp-server (HTTP API) continue to work.

## Key Takeaways

- ko builds Go binaries into minimal container images without a Dockerfile
- `kodata/` files are embedded at `/var/run/ko/` in the container
- `CGO_ENABLED=0` produces a fully static binary — speech features are unavailable
- The `starter` package copies default user content into an empty volume on first run
- `starter/` is created at build time from `user/` — no duplication in git
- The container runs on distroless/static-debian12 (~2MB base image)
- User data persists via a mounted volume at `/var/run/ko/user`
- Build tags (`//go:build cgo` / `//go:build !cgo`) provide stub implementations for non-CGO builds

[← Back: Chapter 20](20-distributed-deployment.md) · [Next: Chapter 22 →](22-kubernetes-deployment.md)
