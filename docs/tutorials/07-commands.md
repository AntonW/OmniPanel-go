# Chapter 7: Command Execution

## What This Package Does

The `commands` package provides two execution backends:
1. **Shell commands** — run arbitrary shell scripts and capture output
2. **HTTP requests** — make HTTP calls and capture responses

Both support parameter substitution, allowing panel buttons to be templated.

## Parameter Substitution

```go
// internal/commands/commands.go
func SubstituteParams(template string, params json.RawMessage) string {
    result := template
    var paramsObj map[string]any
    if err := json.Unmarshal(params, &paramsObj); err != nil {
        return result
    }
    for key, value := range paramsObj {
        placeholder := "{" + key + "}"
        valueStr := ""
        if s, ok := value.(string); ok {
            valueStr = s
        }
        result = strings.ReplaceAll(result, placeholder, valueStr)
    }
    return result
}
```

Example:
- Template: `echo "Volume is {volume}%"`
- Params: `{"volume": "75"}`
- Result: `echo "Volume is 75%"`

> **Concept: Type assertion with comma-ok**
> `value.(string)` attempts to cast `value` to `string`. The two-value form `s, ok := value.(string)` returns `ok = false` if the cast fails, instead of panicking. Only string values are substituted; numbers and booleans are ignored.

> **Concept: `strings.ReplaceAll`**
> Replaces **all** occurrences of the placeholder in the template. If `{volume}` appears twice, both get replaced.

## Parameter Types (Frontend)

Parameters are defined on the frontend with three input types, stored in `block.settingsMeta`:

| Type | HTML Input | Stored Value | Use Case |
|------|-----------|--------------|----------|
| `text` | `<input type="text">` | Any string | Names, messages, file paths |
| `number` | `<input type="number">` | Numeric string | Volume, brightness, counts |
| `toggle` | `<input type="checkbox">` | `"true"` or `"false"` | Enable/disable flags |

Parameters are added dynamically in the editor's Properties panel via the **"+ Add Parameter"** button. Each parameter is stored as `param_<name>` in `block.settings` with its type metadata in `block.settingsMeta`. The `settingsMeta` is serialized to JSON so parameter types persist across panel reloads.

> **Key Pattern: Dynamic Parameter Discovery**
> The client (`client.js:renderCommandParams`) scans `block.settings` for keys starting with `param_` and renders an input for each one. The input type is determined by `block.settingsMeta[key].type`. This means any block can have any number of parameters without template changes — just add `param_<name>` to settings and the UI adapts automatically.

## Shell Command Execution

```go
func ExecuteShell(command string) (bool, string) {
    slog.Info("Executing shell command", "command", command)

    output, err := exec.Command("sh", "-c", command).CombinedOutput()
    if err != nil {
        slog.Error("Shell execution failed", "error", err)
        return false, fmt.Sprintf("Failed to execute command: %s", err)
    }

    result := strings.TrimSpace(string(output))
    if result == "" {
        result = "(no output)"
    }

    return true, result
}
```

> **Concept: `exec.Command`**
> `exec.Command("sh", "-c", command)` creates a command that runs `sh -c "your command here"`. The `-c` flag tells the shell to execute the string as a command. This allows pipes, redirects, and other shell features.

> **Concept: `CombinedOutput`**
> Runs the command and captures both stdout and stderr merged together. Returns the output as `[]byte` and an error if the command exits with a non-zero status.

The function returns `(success, output)`:
- `true, "command output"` on success
- `false, "error message"` on failure

## HTTP Request Execution

```go
func ExecuteHTTP(method, url, body string) (bool, string) {
    slog.Info("Executing HTTP request", "method", method, "url", url)

    var req *http.Request
    var err error

    if body != "" {
        req, err = http.NewRequest(method, url, strings.NewReader(body))
    } else {
        req, err = http.NewRequest(method, url, nil)
    }
    if err != nil {
        return false, fmt.Sprintf("Failed to create request: %s", err)
    }
```

> **Concept: `http.NewRequest`**
> Creates an HTTP request object without sending it. The third argument is the request body as an `io.Reader`. `strings.NewReader(body)` wraps a string as a reader. For GET requests (no body), we pass `nil`.

```go
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        slog.Error("HTTP request failed", "error", err)
        return false, fmt.Sprintf("HTTP request failed: %s", err)
    }
    defer resp.Body.Close()
```

> **Concept: `http.Client`**
> The default `&http.Client{}` uses sensible defaults: 10-second timeout, automatic redirect following, connection pooling. For most use cases, you don't need to configure it further.

> **Key Pattern: `defer resp.Body.Close()`**
> HTTP response bodies must be closed to release the underlying connection. `defer` ensures this happens even if we return early due to an error later.

```go
    respBody, _ := io.ReadAll(resp.Body)
    respStr := strings.TrimSpace(string(respBody))
    if respStr == "" {
        respStr = "(no response body)"
    }

    result := fmt.Sprintf("Status: %s\n\n%s", resp.Status, respStr)
    return resp.StatusCode >= 200 && resp.StatusCode < 300, result
}
```

> **Concept: `io.ReadAll`**
> Reads the entire response body into a byte slice. The error is ignored (`_`) because a read error at this point is rare and the status code is already known.

Success is determined by HTTP status code: 2xx (200–299) means success.

## How Commands Flow Through the System

1. User taps a button on the panel UI
2. `client.js` collects parameter values from `blockWrapper.paramValues` (populated by `renderCommandParams()`)
3. `client.js` sends `execute-command` via WebSocket with `command_type`, `command`, and `params`
4. `websocket/handler.go` routes to `handleCommand()`
5. `SubstituteParams` replaces `{placeholders}` with actual string values
6. `ExecuteShell` or `ExecuteHTTP` runs the command
7. Result is broadcast back as `command-result`
8. `client.js` shows button feedback (green/red flash) and a toast notification

## Speech Commands Flow

Speech commands follow a similar but extended path:

1. User holds the microphone button on the panel
2. `client.js` starts recording via `MediaRecorder` API
3. Audio chunks are sent as **binary WebSocket frames** (not JSON)
4. `router.go` detects `BinaryMessage` and routes to `HandleAudioChunk()`
5. If audio is webm/opus, `decoder.go` converts it to PCM 16kHz mono 16-bit
6. `SpeechManager.Process()` sends PCM to the STT engine (Vosk or llama-cpp)
7. Transcribed text is matched against `speech_commands.json` using `matcher.go`
8. If matched, the command is executed (shell, HTTP, button, or slider)
9. Result is broadcast as `speech-result` with optional TTS text
10. `client.js` shows toast notification, block flash, and speaks confirmation

> **Key Pattern: Interface-based design**
> The `STTEngine` interface (`Recognize(pcm []byte) (string, error)`) allows swapping between Vosk and llama-cpp without changing the SpeechManager code. This is Go's approach to polymorphism — define what you need, not what you have.

## Key Takeaways

- `exec.Command("sh", "-c", ...)` enables full shell capabilities
- `CombinedOutput` captures stdout + stderr together
- `http.NewRequest` + `client.Do` is the standard Go HTTP pattern
- Always `defer resp.Body.Close()` after a successful HTTP request
- Parameter substitution uses `{key}` placeholders replaced via `strings.ReplaceAll`
- Only string-typed params are substituted; numbers/booleans are skipped by the Go backend
- Frontend parameters support three types: text, number, toggle — stored in `settingsMeta` and serialized to JSON
- Parameters are discovered dynamically by `param_` prefix — no template changes needed to add new ones
- Speech commands extend the command system with voice input via binary WebSocket frames
- The `STTEngine` interface enables pluggable speech backends

[← Back: Chapter 6](06-websocket.md) · [Next: Chapter 8 →](08-virtual-input.md)
