# Chapter 6: WebSocket Real-time Communication

## What This Package Does

WebSocket provides a persistent, bidirectional connection between the browser and server. Unlike HTTP (request → response), WebSocket allows the server to push data to the client at any time. OmniPanel-go uses WebSocket for:

- Broadcasting client connect/disconnect log events (with IP address)
- Broadcasting speech config on connect (`speech-config-init`)
- Broadcasting system metrics every 500ms
- Receiving simulated input events (buttons, sliders, joysticks, mousepad, mouse wheel, mouse buttons, keyboard)
- Executing commands and returning results
- Pushing custom data from clients to the DataBus
- Managing speech recording lifecycle (host and client modes)

## WebSocket Upgrade (router.go)

```go
// internal/routes/router.go
app.Get("/ws", ws.New(func(c *ws.Conn) {
    handleWS(c, s)
}))
```

> **Concept: WebSocket upgrade**
> A WebSocket connection starts as a regular HTTP GET request. The `ws.New()` middleware checks for the `Upgrade: websocket` header and transforms the connection. After the upgrade, HTTP is gone — only raw message frames remain.

> **Note: Authentication in serve/connect modes**
> In distributed deployment (serve/connect modes), all HTTP routes including the WebSocket upgrade are protected by token-based authentication middleware (`internal/auth/middleware.go`). The token is accepted via query parameter (`?token=xxx`) or Authorization header (`Bearer xxx`). Host WebSocket connections additionally validate the token from the query parameter (`?type=host&token=xxx`). See [Chapter 20](20-distributed-deployment.md) for details.

## The Connection Handler

```go
func handleWS(c *ws.Conn, s *state.AppState) {
    ch := make(chan []byte, 256)
    s.RegisterClient(ch)

    clientIP := c.IP()
    s.BroadcastJSON(map[string]any{
        "type":      "log-event",
        "timestamp": time.Now().Format("2006-01-02 15:04:05"),
        "data":      "Client connected: " + clientIP,
    })

    if s.Config != nil {
        s.BroadcastJSON(map[string]any{
            "type": "speech-config-init",
            "data": map[string]any{
                "enabled":           s.Config.Speech.Enabled,
                "recordingLocation": s.Config.Speech.RecordingLoc,
                "triggerMode":       s.Config.Speech.TriggerMode,
                "wakeWord":          s.Config.Speech.WakeWord,
                "wakeWordListenSec": s.Config.Speech.WakeWordListenSec,
                "ttsEnabled":        s.Config.Speech.TTSEnabled,
            },
        })
    }

    defer func() {
        s.UnregisterClient(ch)
        s.BroadcastJSON(map[string]any{
            "type":      "log-event",
            "timestamp": time.Now().Format("2006-01-02 15:04:05"),
            "data":      "Client disconnected: " + clientIP,
        })
    }()
```

**Step 2:** Create a buffered channel (capacity 256) and register it with `AppState`. Then broadcast a "client connected" log event to all connected clients. Immediately after, broadcast `speech-config-init` with the current speech settings (enabled, recording location, trigger mode, wake word, TTS). This lets all clients initialize their speech UI correctly. The client's IP address is captured via `c.IP()`.

```go
    go func() {
        for msg := range ch {
            if err := c.WriteMessage(ws.TextMessage, msg); err != nil {
                return
            }
        }
    }()
```

**Step 3:** Launch a goroutine that reads from the channel and writes to the WebSocket. This is necessary because `WriteMessage` is blocking — if we called it directly in the broadcast loop, a slow network write would block all other clients.

> **Key Pattern: Separate read and write goroutines**
> WebSocket connections need one goroutine for reading and one for writing. The broadcast goroutine (in `AppState`) sends to channels; this per-connection goroutine reads from its channel and writes to the socket.

```go
    // Read loop: reads from WebSocket, dispatches to message handler
    for {
        msgType, msg, err := c.ReadMessage()
        if err != nil {
            break
        }
        if msgType == ws.BinaryMessage {
            websocket.HandleAudioChunk(s, msg)
        } else {
            websocket.HandleMessage(s, clientID, string(msg))
        }
    }

    slog.Info("WebSocket disconnected")
}
```

**Step 4:** The read loop. `ReadMessage()` blocks until a message arrives and returns the message type. Binary frames are routed to `HandleAudioChunk`, text frames to `HandleMessage` (with the `clientID` for per-client tracking). When the client disconnects, it returns an error, breaking the loop. The `defer` function then runs: it unregisters the client from the RSS manager, unregisters the broadcast channel, and broadcasts a "client disconnected" log event.

## Message Routing (websocket/handler.go)

The handler accepts an `AppStateInterface` instead of a concrete `*state.AppState`. This allows it to work with both `AppState` (default mode) and `Agent` (connect mode):

```go
// internal/websocket/handler.go
func HandleMessage(s state.AppStateInterface, clientID uint64, raw string) {
    var parsed map[string]json.RawMessage
    if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
        slog.Warn("Failed to parse WS message", "error", err)
        return
    }

    msgType := ""
    if t, ok := parsed["type"]; ok {
        json.Unmarshal(t, &msgType)
    }

    data := parsed["data"]

    switch msgType {
    case "host-register":
        slog.Info("Host agent registered")
    case "heartbeat":
        s.BroadcastJSON(map[string]any{"type": "heartbeat-ack"})
    case "heartbeat-ack":
        // Acknowledgment of our heartbeat, no action needed
    case "simulate-button":
        handleButton(s, data)
    case "simulate-slider":
        handleSlider(s, data)
    case "simulate-joystick":
        handleJoystick(s, data)
    // ... more cases
    }
}
```

> **Concept: `AppStateInterface`**
> The WebSocket handler uses an interface instead of a concrete type. In Go, interfaces are satisfied implicitly — any type that implements the required methods automatically satisfies the interface. This lets the same handler code work with `AppState` (default mode, with HTTP server) and `Agent` (connect mode, WebSocket client only).

> **Concept: `json.RawMessage`**
> `json.RawMessage` is a `[]byte` that defers JSON parsing. We parse the top-level object to get `"type"` and `"data"`, but leave `"data"` as raw bytes. Each handler then parses only the fields it needs. This is more efficient than unmarshaling everything into a full struct.

> **Concept: `clientID` parameter**
> The `clientID` is passed to `HandleMessage` so handlers that need per-client tracking (like `handleRSSConfigure`) can register the client directly. This avoids a separate lookup step and ensures the client is registered before any updates are sent.

## Speech Config

```go
func handleSpeechConfig(s *state.AppState, data json.RawMessage) {
    var obj map[string]json.RawMessage
    json.Unmarshal(data, &obj)

    var enabled bool
    if raw, ok := obj["enabled"]; ok {
        json.Unmarshal(raw, &enabled)
    }

    slog.Info("Speech config updated", "enabled", enabled)

    s.BroadcastJSON(map[string]any{
        "type": "speech-config-updated",
        "data": map[string]any{
            "enabled": enabled,
        },
    })
}
```

Clients send this to toggle speech recognition on/off. The server broadcasts `speech-config-updated` so all clients sync their UI state.

## Register Speech Trigger

```go
func handleRegisterSpeechTrigger(s *state.AppState, data json.RawMessage) {
    var obj map[string]json.RawMessage
    json.Unmarshal(data, &obj)

    var blockID, phrase, triggerType string
    var aliases []string
    var joystickIndex, buttonID, axisID int

    if raw, ok := obj["block_id"]; ok {
        json.Unmarshal(raw, &blockID)
    }
    if raw, ok := obj["phrase"]; ok {
        json.Unmarshal(raw, &phrase)
    }
    if raw, ok := obj["type"]; ok {
        json.Unmarshal(raw, &triggerType)
    }
    if raw, ok := obj["aliases"]; ok {
        json.Unmarshal(raw, &aliases)
    }
    if raw, ok := obj["joystick_index"]; ok {
        json.Unmarshal(raw, &joystickIndex)
    }
    if raw, ok := obj["button_id"]; ok {
        json.Unmarshal(raw, &buttonID)
    }
    if raw, ok := obj["axis_id"]; ok {
        json.Unmarshal(raw, &axisID)
    }

    if phrase == "" || blockID == "" {
        slog.Warn("register-speech-trigger missing phrase or block_id")
        return
    }

    if triggerType == "" {
        triggerType = "button"
    }

    if s.SpeechManager != nil {
        s.SpeechManager.RegisterBlockTrigger(blockID, phrase, aliases, triggerType, joystickIndex, buttonID, axisID)
        slog.Info("Registered speech trigger", "block_id", blockID, "phrase", phrase, "type", triggerType, "joystick_index", joystickIndex)
    }
}
```

> **Key Pattern: Hardware IDs for direct execution**
> The `joystick_index`, `button_id`, and `axis_id` fields enable the SpeechManager to execute actions directly on the server side. When speech matches a trigger, the server sends the command to the virtual device without requiring the client to send a follow-up `simulate-button` or `simulate-slider` message. This reduces latency and works even if the client UI is not focused.

## Parsing Helper Functions

```go
func parseUintField(data json.RawMessage, field string) uint64 {
    var obj map[string]json.RawMessage
    if err := json.Unmarshal(data, &obj); err != nil {
        return 0
    }
    raw, ok := obj[field]
    if !ok {
        return 0
    }

    var num float64
    if err := json.Unmarshal(raw, &num); err == nil {
        return uint64(num)
    }

    var str string
    if err := json.Unmarshal(raw, &str); err == nil {
        parsed, _ := strconv.ParseUint(str, 10, 64)
        return parsed
    }
    return 0
}
```

JavaScript sends numbers as JSON numbers, but sometimes as strings. This helper tries both interpretations. It's lenient — if parsing fails, it returns 0 instead of an error.

```go
func parseStringField(data json.RawMessage, field string) string {
    var obj map[string]json.RawMessage
    if err := json.Unmarshal(data, &obj); err != nil {
        return ""
    }
    var val string
    if raw, ok := obj[field]; ok {
        json.Unmarshal(raw, &val)
    }
    return val
}
```

Extracts a string field from a JSON object. Used by keyboard and mouse button handlers.

## Input Simulation Handlers

### Button

```go
func handleButton(s *state.AppState, data json.RawMessage) {
    jsRaw := parseStringField(data, "js")
    idRaw := parseStringField(data, "id")
    state := uint8(parseUintField(data, "state"))

    if jsRaw == "" || idRaw == "" {
        slog.Debug("Button event ignored (no joystick/button configured)")
        return
    }

    js := int(parseUintField(data, "js"))
    id := int(parseUintField(data, "id"))

    slog.Info("Button event", "js", js, "id", id, "state", state)
    s.JoystickManager.Send(devices.Command{
        Type:  devices.ButtonType,
        Js:    js,
        Id:    id,
        Value: state,
    })
}
```

Receives `{js: 0, id: 2, state: 1}` and sends a button-press command to virtual joystick #0, button #2. If `js` or `id` are empty strings (meaning the button has no joystick/button configured), the event is silently ignored to prevent sending spurious input.

### Joystick (2-axis)

```go
func handleJoystick(s *state.AppState, data json.RawMessage) {
    jsRaw := parseStringField(data, "js")
    idRaw := parseStringField(data, "id")

    if jsRaw == "" || idRaw == "" {
        slog.Debug("Joystick event ignored (no joystick/axis configured)")
        return
    }

    js := int(parseUintField(data, "js"))
    id := int(parseUintField(data, "id"))

    var valueObj map[string]json.RawMessage
    json.Unmarshal(data, &valueObj)

    var x, y uint64
    if raw, ok := valueObj["value"]; ok {
        var v map[string]uint64
        json.Unmarshal(raw, &v)
        x = v["x"]
        y = v["y"]
    }

    s.JoystickManager.Send(devices.Command{
        Type:  devices.AxisType,
        Js:    js,
        Id:    id,
        Value: uint8(x),
    })
    s.JoystickManager.Send(devices.Command{
        Type:  devices.AxisType,
        Js:    js,
        Id:    id + 1,
        Value: uint8(y),
    })
}
```

A joystick has two axes (X and Y). The client sends them as `{value: {x: 128, y: 128}}`. The handler splits this into two separate axis commands, using `id` for X and `id+1` for Y.

### Slider

```go
func handleSlider(s *state.AppState, data json.RawMessage) {
    jsRaw := parseStringField(data, "js")
    idRaw := parseStringField(data, "id")
    value := uint8(parseUintField(data, "value"))

    if jsRaw == "" || idRaw == "" {
        slog.Debug("Slider event ignored (no joystick/axis configured)")
        return
    }

    js := int(parseUintField(data, "js"))
    id := int(parseUintField(data, "id"))

    s.JoystickManager.Send(devices.Command{
        Type:  devices.AxisType,
        Js:    js,
        Id:    id,
        Value: value,
    })
}
```

Sliders are mapped to single-axis joystick events. The client sends `{js: 0, id: 0, value: 128}` and the server routes it as an axis command.

### Mousepad

```go
func handleMousepad(s *state.AppState, data json.RawMessage) {
    js := int(parseUintField(data, "js"))

    var valueObj map[string]json.RawMessage
    json.Unmarshal(data, &valueObj)

    var dx, dy int32
    if raw, ok := valueObj["value"]; ok {
        var v map[string]int32
        json.Unmarshal(raw, &v)
        dx = v["x"]
        dy = v["y"]
    }

    slog.Info("Mousepad move", "js", js, "dx", dx, "dy", dy)
    s.MousepadManager.SendMove(js, dx, dy)
}
```

Mouse movement is **relative** (delta X, delta Y), unlike joystick axes which are **absolute** (0–255).

### Mouse Wheel

```go
func handleMousewheel(s *state.AppState, data json.RawMessage) {
    js := int(parseUintField(data, "js"))
    delta := int32(parseUintField(data, "delta"))

    s.MousepadManager.SendWheel(js, delta)
}
```

Scroll wheel events use a signed delta value. Positive = scroll up, negative = scroll down.

### Mouse Button

```go
func handleMousebtn(s *state.AppState, data json.RawMessage) {
    js := int(parseUintField(data, "js"))
    btnName := parseStringField(data, "btn")
    state := uint8(parseUintField(data, "state"))

    var btn int
    switch btnName {
    case "left":
        btn = devices.MouseBtnLeft
    case "right":
        btn = devices.MouseBtnRight
    case "middle":
        btn = devices.MouseBtnMiddle
    default:
        btn = devices.MouseBtnLeft
    }

    s.MousepadManager.SendButton(js, btn, state)
}
```

Maps human-readable button names (`"left"`, `"right"`, `"middle"`) to platform-agnostic constants. Defaults to left button if unknown.

### Keyboard

```go
func handleKeyboard(s *state.AppState, data json.RawMessage) {
    kbIndex := int(parseUintField(data, "keyboard_index"))
    key := parseStringField(data, "key")
    keyState := uint8(parseUintField(data, "state"))

    if key == "" {
        slog.Warn("Keyboard event missing key")
        return
    }

    // Parse key combination (e.g., "ctrl+shift+a")
    parts := strings.Split(strings.ToLower(key), "+")
    codes := make([]int, 0, len(parts))

    for _, part := range parts {
        part = strings.TrimSpace(part)
        code, ok := devices.KeyNameToCode[part]
        if !ok {
            slog.Warn("Unknown keyboard key", "key", part)
            return
        }
        codes = append(codes, code)
    }

    if len(codes) == 1 {
        slog.Info("Keyboard key event", "keyboard_index", kbIndex, "key", key, "state", keyState)
        s.KeyboardManager.SendKey(kbIndex, codes[0], keyState)
    } else {
        slog.Info("Keyboard combo event", "keyboard_index", kbIndex, "keys", key, "state", keyState)
        s.KeyboardManager.SendCombo(kbIndex, codes, keyState)
    }
}
```

Supports single keys (`"a"`) and combinations (`"ctrl+a"`, `"ctrl+shift+a"`). The key string is split on `+`, each part is looked up in `devices.KeyNameToCode`, then sent as either a single key or a combo.

### Joystick Count

```go
func handleJoystickCount(s *state.AppState, data json.RawMessage) {
    var count uint64
    json.Unmarshal(data, &count)
    s.UpdateJoystickCount(uint8(count))
}
```

Allows clients to change the number of virtual joysticks at runtime. The data field is a plain number, not an object.

## Command Execution

```go
func handleCommand(s *state.AppState, parsed map[string]json.RawMessage) {
    data := parsed["data"]
    var cmdObj map[string]json.RawMessage
    json.Unmarshal(data, &cmdObj)

    blockID := ""
    if raw, ok := cmdObj["block_id"]; ok {
        json.Unmarshal(raw, &blockID)
    }

    cmdType := "shell"
    if raw, ok := cmdObj["command_type"]; ok {
        json.Unmarshal(raw, &cmdType)
    }

    command := ""
    if raw, ok := cmdObj["command"]; ok {
        json.Unmarshal(raw, &command)
    }

    params := cmdObj["params"]

    var success bool
    var output string

    switch cmdType {
    case "shell":
        substituted := commands.SubstituteParams(command, params)
        success, output = commands.ExecuteShell(substituted)
    case "http":
        httpMethod := ""
        if raw, ok := cmdObj["http_method"]; ok {
            json.Unmarshal(raw, &httpMethod)
        }
        // ... extract URL and body
        substitutedURL := commands.SubstituteParams(httpURL, params)
        substitutedBody := commands.SubstituteParams(httpBody, params)
        success, output = commands.ExecuteHTTP(httpMethod, substitutedURL, substitutedBody)
    }

    result := map[string]any{
        "type": "command-result",
        "data": map[string]any{
            "block_id": blockID,
            "success":  success,
            "output":   output,
        },
    }
    s.BroadcastJSON(result)
}
```

Commands can be shell commands or HTTP requests. Parameters are substituted (e.g., `{volume}` → `50`) before execution. The result is broadcast back to all clients so the triggering button can show success/failure feedback.

## Push Data

```go
func handlePushData(s *state.AppState, data json.RawMessage) {
    var obj map[string]json.RawMessage
    if err := json.Unmarshal(data, &obj); err != nil {
        return
    }

    var key, unit, source string
    var value any

    if raw, ok := obj["key"]; ok {
        json.Unmarshal(raw, &key)
    }
    if raw, ok := obj["value"]; ok {
        json.Unmarshal(raw, &value)
    }
    // ... extract unit and source

    if source != "" {
        s.DataBus.SetSource(key, value, unit, source)
    } else {
        s.DataBus.Set(key, value, unit)
    }
}
```

Allows clients to push custom metrics into the DataBus. These metrics are then included in the periodic `data-update` broadcasts.

## Fullscreen Control

```go
    case "enter-fullscreen":
        s.BroadcastJSON(map[string]any{"type": "enter-fullscreen"})
    case "exit-fullscreen":
        s.BroadcastJSON(map[string]any{"type": "exit-fullscreen"})
```

The Start Page sends these to request fullscreen on all connected client devices. The clients show a prompt (browsers require user interaction to enter fullscreen).

## Binary Message Handling (Audio)

Starting with the speech commands feature, the WebSocket connection now handles both text and binary messages. The router distinguishes between them:

```go
// internal/routes/router.go
for {
    msgType, msg, err := c.ReadMessage()
    if err != nil {
        break
    }
    if msgType == ws.BinaryMessage {
        websocket.HandleAudioChunk(s, msg)
    } else {
        websocket.HandleMessage(s, clientID, string(msg))
    }
}
```

> **Concept: WebSocket message types**
> WebSocket supports two frame types: `TextMessage` (UTF-8 strings) and `BinaryMessage` (raw bytes). OmniPanel-go uses text for JSON commands and binary for audio data. This avoids the overhead of base64 encoding audio inside JSON.

The audio handler decodes the incoming audio and runs speech recognition:

```go
// internal/websocket/handler.go
func HandleAudioChunk(s *state.AppState, audioData []byte) {
    if s.SpeechManager == nil {
        return
    }

    cfg := s.SpeechManager.GetConfig()
    if cfg != nil && cfg.RecordingLoc == "host" {
        return
    }

    pcm := audioData
    if !speech.ValidatePCM(audioData) {
        var err error
        pcm, err = speech.DecodeAudio(audioData)
        if err != nil {
            slog.Warn("Failed to decode audio", "error", err)
            s.BroadcastJSON(map[string]any{
                "type": "speech-error",
                "data": map[string]any{
                    "error": "Failed to decode audio: " + err.Error(),
                },
            })
            return
        }
    }

    text, matched, speakText, err := s.SpeechManager.Process(pcm)
    if err != nil {
        slog.Error("Speech processing failed", "error", err)
        s.BroadcastJSON(map[string]any{
            "type": "speech-error",
            "data": map[string]any{
                "error": err.Error(),
            },
        })
        return
    }

    result := map[string]any{
        "text":    text,
        "matched": matched,
    }
    if speakText != "" {
        result["speak"] = speakText + " confirmed"
    }

    s.BroadcastJSON(map[string]any{
        "type": "speech-result",
        "data": result,
    })
}
```

> **Key Pattern: Early return for host recording**
> When `recordingLocation` is `"host"`, the server records directly from the host microphone. Binary audio frames from the client are ignored (early return). This prevents duplicate audio processing.

> **Key Pattern: Type-switch on message type**
> By checking `msgType` before routing, we can handle completely different data formats on the same connection. This is cleaner than opening a second WebSocket just for audio.

## Speech Message Handlers

Three new message types manage the speech recording lifecycle:

```go
func handleStartRecording(s *state.AppState, data json.RawMessage) {
    if s.SpeechManager == nil {
        slog.Warn("Speech manager not available")
        return
    }

    var obj map[string]json.RawMessage
    json.Unmarshal(data, &obj)

    mode := "push-to-talk"
    if raw, ok := obj["mode"]; ok {
        json.Unmarshal(raw, &mode)
    }

    cfg := s.SpeechManager.GetConfig()
    recordingLoc := "client"
    if cfg != nil {
        recordingLoc = cfg.RecordingLoc
    }

    if recordingLoc == "host" {
        if s.SpeechManager.IsHostRecording() {
            slog.Info("Host already recording (wake word mode active)")
            return
        }
        if err := s.SpeechManager.StartHostRecording(); err != nil {
            slog.Error("Failed to start host recording", "error", err)
            s.BroadcastJSON(map[string]any{
                "type": "speech-error",
                "data": map[string]any{
                    "error": "Failed to start host recording: " + err.Error(),
                },
            })
            return
        }
        slog.Info("Host recording started", "mode", mode)
    } else {
        slog.Info("Client recording started", "mode", mode)
    }

    s.BroadcastJSON(map[string]any{
        "type": "recording-status",
        "data": map[string]any{
            "state":    "listening",
            "mode":     mode,
            "location": recordingLoc,
        },
    })
}

func handleStopRecording(s *state.AppState) {
    if s.SpeechManager == nil {
        slog.Warn("Speech manager not available")
        return
    }

    s.BroadcastJSON(map[string]any{
        "type": "recording-status",
        "data": map[string]any{
            "state": "processing",
        },
    })

    cfg := s.SpeechManager.GetConfig()
    if cfg != nil && cfg.RecordingLoc == "host" {
        go func() {
            text, matched, speakText, err := s.SpeechManager.StopHostRecording()
            if err != nil {
                slog.Error("Host recording processing failed", "error", err)
                s.BroadcastJSON(map[string]any{
                    "type": "speech-error",
                    "data": map[string]any{
                        "error": err.Error(),
                    },
                })
                return
            }

            result := map[string]any{
                "text":    text,
                "matched": matched,
            }
            if speakText != "" {
                result["speak"] = speakText + " confirmed"
            }

            s.BroadcastJSON(map[string]any{
                "type": "speech-result",
                "data": result,
            })
        }()
    } else {
        slog.Info("Client recording stopped, waiting for audio data")
    }
}
```

> **Concept: Host vs Client recording**
> OmniPanel-go supports two recording locations. With `recordingLocation: "client"`, the browser records audio and streams binary frames to the server. With `recordingLocation: "host"`, the server records directly from the host microphone — no binary frames are needed. The `start-recording` handler checks `cfg.RecordingLoc` and either starts host recording or tells the client to begin streaming. The `stop-recording` handler processes host audio in a goroutine (to avoid blocking the read loop) or waits for client audio frames.

The response includes a `"location"` field so clients know whether to stream audio or not.

## Full Message Summary

**Client → Server:**
| Type | Data | Description |
|------|------|-------------|
| `simulate-button` | `{ "js": 0, "id": 0, "state": 1 }` | Press/release button |
| `simulate-slider` | `{ "js": 0, "id": 0, "value": 128 }` | Set slider value (single axis) |
| `simulate-joystick` | `{ "js": 0, "id": 0, "value": { "x": 127, "y": 127 } }` | Set joystick X/Y |
| `simulate-mousepad` | `{ "js": 0, "value": { "x": 10, "y": -5 } }` | Relative mouse movement |
| `simulate-mousewheel` | `{ "js": 0, "delta": 120 }` | Scroll wheel (signed delta) |
| `simulate-mousebtn` | `{ "js": 0, "btn": "left", "state": 1 }` | Mouse button press/release |
| `simulate-keyboard` | `{ "keyboard_index": 0, "key": "ctrl+a", "state": 1 }` | Key or combo press/release |
| `save-joystick-count` | `4` | Change virtual joystick count (plain number) |
| `execute-command` | `{ "block_id": "...", "command_type": "shell", ... }` | Execute command |
| `push-data` | `{ "key": "...", "value": ... }` | Push data to DataBus |
| `start-recording` | `{ "mode": "push-to-talk" }` | Begin audio recording |
| `stop-recording` | — | End audio recording |
| `speech-config` | `{ "enabled": true }` | Toggle speech on client |
| `register-speech-trigger` | `{ "block_id": "...", "phrase": "...", "aliases": [...], "type": "button", "joystick_index": 0, "button_id": 3, "axis_id": 0 }` | Register block speech trigger with hardware IDs |
| `rss-configure` | `{ "block_id": "...", "feed_urls": ["..."], "refresh_interval": 60, "max_entries": 20 }` | Set up RSS feed polling for a block |
| `open-url` | `{ "url": "https://..." }` | Open URL in host's default browser |
| `enter-fullscreen` | — | Request fullscreen on all clients |
| `exit-fullscreen` | — | Exit fullscreen on all clients |
| `host-register` | — | Host agent registration (distributed deployment) |
| `heartbeat` | — | Host agent keepalive (distributed deployment, every 15s) |

**Server → Client:**
| Type | Data | Description |
|------|------|-------------|
| `load-panel` | `{ ...panel JSON... }` | Panel definition sent on connect |
| `log-event` | `{ "timestamp": "...", "data": "Client connected: ..." }` | Connect/disconnect log (also host connect/disconnect) |
| `speech-config-init` | `{ "enabled": true, "recordingLocation": "client", "triggerMode": "...", "wakeWord": "...", "wakeWordListenSec": 3, "ttsEnabled": true }` | Speech settings sent on connect |
| `data-update` | `{ "key": { "value": ..., "unit": "..." } }` | DataBus snapshot |
| `command-result` | `{ "block_id": "...", "success": true }` | Command execution result |
| `speech-result` | `{ "text": "...", "matched": true, "speak": "..." }` | Speech transcription |
| `speech-error` | `{ "error": "..." }` | Speech processing error |
| `recording-status` | `{ "state": "listening", "mode": "...", "location": "client" }` | Recording state change |
| `speech-button-trigger` | `{ "block_id": "..." }` | Speech-triggered button |
| `speech-slider-trigger` | `{ "block_id": "...", "value": "..." }` | Speech-triggered slider |
| `speech-config-updated` | `{ "enabled": true }` | Broadcast after speech-config change |
| `rss-update` | `{ "block_id": "...", "entries": [{ "guid": "...", "title": "...", "is_new": true, ... }] }` | RSS feed entries with per-client new flags |
| `enter-fullscreen` | — | Broadcast fullscreen request |
| `exit-fullscreen` | — | Broadcast exit fullscreen |
| `heartbeat-ack` | — | Acknowledge host agent heartbeat (distributed deployment) |

## Key Takeaways

- WebSocket upgrade transforms HTTP into a persistent bidirectional connection
- Separate read and write goroutines prevent blocking
- Buffered channels (capacity 256) absorb burst messages
- `json.RawMessage` defers parsing for efficiency
- Non-blocking broadcast (`select` + `default`) protects against slow clients
- `defer` ensures cleanup (RSS manager, broadcast channel) and disconnect log broadcast on disconnect
- Client IP is captured via `c.IP()` and included in connect/disconnect log events
- Timestamps use 24-hour format with date: `2006-01-02 15:04:05`
- Command results are broadcast so all clients see the outcome
- Binary WebSocket frames carry audio data without base64 overhead
- Speech recording lifecycle: `start-recording` → binary frames (client mode) or host mic (host mode) → `stop-recording`
- `speech-config-init` is broadcast on connect so clients initialize speech UI
- Host vs client recording: host mode records from server mic, ignores binary frames; client mode streams audio from browser
- Hardware IDs in `register-speech-trigger` enable direct server-side execution without client follow-up
- `parseStringField` helper extracts string fields (used by keyboard/mouse handlers)
- Keyboard handler supports combos via `+` separator (`"ctrl+shift+a"`)
- `clientID` is passed through `HandleMessage` to handlers that need per-client tracking (e.g., RSS configuration)
- RSS `rss-configure` and `open-url` messages extend the WebSocket protocol for feed integration
- `AppStateInterface` allows the same handler code to work in both default and distributed deployment modes
- `host-register` and `heartbeat`/`heartbeat-ack` messages support distributed deployment
- Log events now include host connect/disconnect messages in addition to client connect/disconnect

[← Back: Chapter 5](05-http-routing.md) · [Next: Chapter 7 →](07-commands.md)
