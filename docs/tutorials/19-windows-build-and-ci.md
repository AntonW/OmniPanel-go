# Chapter 19: Windows Build Script and CI Alignment

## What This Chapter Covers

This chapter explains how the maintained Windows build script (`scripts/build-with-vosk.ps1`) relates to the CI pipeline and why both exist.

- Local/native Windows builds: use PowerShell script in `scripts/`
- CI Linux-to-Windows CGO cross-build is currently not a supported/working path
- Runtime behavior: app can auto-download missing Vosk runtime DLLs on Windows at startup

## Recommended Windows Build Path

Use the script:

```powershell
.\scripts\build-with-vosk.ps1
```

Optional authenticated mode (helps avoid GitHub API rate limits):

```powershell
.\scripts\build-with-vosk.ps1 -GitHubToken "ghp_xxx"
```

> **Concept (Go + tooling): deterministic build wrapper**
> Instead of asking each developer to remember CGO flags, paths, and DLL copy steps, a script codifies the workflow once. This reduces machine-to-machine drift and keeps local behavior close to CI behavior.

## What the Script Does

From `scripts/build-with-vosk.ps1`:

1. Verifies toolchain (`gcc`, `go`)
2. Reuses existing `third_party/vosk` files if present
3. Otherwise downloads Vosk release asset and extracts headers/libs
4. Generates MinGW import library (`libvosk.dll.a`) when missing
5. Builds with CGO and writes logs to:
   - `bin/build.stdout.log`
   - `bin/build.stderr.log`
6. Copies runtime DLLs next to the built EXE, including `vJoyInterface.dll` if found in vJoy install locations

> **Key Pattern (PowerShell + CGO): fallback chain**
> The script uses a fallback strategy for fragile external dependencies: authenticated GitHub API if available, direct release URL fallback, then local reuse on later runs. Similar to Go fallback code paths, this improves reliability under rate limits and network variation.

### vJoyInterface.dll Provisioning

The script automatically detects and copies `vJoyInterface.dll` from standard vJoy install paths:

```powershell
$vjoyDll = "C:\Program Files\vJoy\x64\vJoyInterface.dll"
if (Test-Path $vjoyDll) {
    $runtimeDlls += $vjoyDll
}
```

This ensures the built binary has all required runtime dependencies. At runtime, the Go code uses a multi-path loading strategy to find `vJoyInterface.dll` from:
1. Current directory (where the script placed it)
2. Standard vJoy installation directories
3. System PATH

This two-pronged approach — build-time bundling + runtime multi-path search — maximizes compatibility across different Windows configurations.

## CI vs Local Script

Linux-to-Windows CGO cross-compilation in CI is currently unreliable for this project.
Use the script (`scripts/build-with-vosk.ps1`) as the source of truth for working Windows builds.

Recommended environments:

- CI: Linux build/test path
- Windows binary: native Windows PowerShell + local MSYS2/MinGW via `scripts/build-with-vosk.ps1`

## Runtime DLL Auto-Provisioning (Windows App Startup)

Even after build, runtime DLLs may be missing on user machines. The app now handles this at startup for Vosk mode:

- Code path: `internal/speech/vosk_runtime_windows.go`
- Trigger: speech engine `vosk`
- Downloads/extracts when needed:
  - `libvosk.dll`
  - `libstdc++-6.dll`
- Uses config override `speech.vosk_runtime_url` with default fallback URL if empty

> **Concept (platform isolation): build tags**
> Windows-only runtime bootstrap logic is in a Windows build-tagged file, with a no-op stub for non-Windows. This keeps platform-specific behavior isolated while preserving one shared call site in `speech.go`.

## Troubleshooting Quick Checks

- Build fails early: verify `gcc --version` and `go version`
- API rate limit: set `GITHUB_TOKEN` or pass `-GitHubToken`
- Linker errors around `-lvosk`: clean and rerun script to regenerate import library
- Runtime missing DLL errors: ensure app can write to user config dir and has network access (or set `speech.vosk_runtime_url` to an internal mirror)

## Key Takeaways

- Use `scripts/build-with-vosk.ps1` for reliable native Windows builds
- Native Windows script is the supported path for working CGO+Vosk Windows binaries
- Windows runtime DLL bootstrap in app startup reduces end-user setup friction
- `speech.vosk_runtime_url` enables enterprise/mirrored deployment flows

[← Back: Chapter 18](18-rss-feed.md) · [Next: Chapter 20 →](20-distributed-deployment.md)
