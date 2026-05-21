// Package websocket handles WebSocket message routing and dispatching.
// It receives JSON messages from clients, parses the "type" field, and
// routes to the appropriate handler function.
//
// Message types handled:
//   - simulate-button/slider/joystick/mousepad/mousebtn/wheel/keyboard: input simulation
//   - execute-command: shell or HTTP command execution
//   - push-data: custom metric injection into DataBus
//   - start/stop-recording: speech recording lifecycle
//   - speech-config/register-speech-trigger: speech settings
//   - enter/exit-fullscreen: fullscreen control broadcast
//   - rss-configure: set up RSS feed polling for a block
//   - open-url: open a URL in the host's default browser (when block's open_url_location is "host")
//   - host-register: host agent registration (distributed deployment)
//   - heartbeat/heartbeat-ack: keepalive for distributed deployment
//
// Binary WebSocket frames (audio data) are routed to HandleAudioChunk,
// while text frames (JSON) are routed to HandleMessage.
//
// The handler accepts an AppStateInterface, allowing it to work with both
// AppState (default mode) and Agent (connect mode) implementations.
//
// See docs/tutorials/06-websocket.md for a detailed walkthrough.
package websocket

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"omnipanel-go/internal/commands"
	"omnipanel-go/internal/devices"
	"omnipanel-go/internal/rssfeed"
	"omnipanel-go/internal/speech"
	"omnipanel-go/internal/state"
)

// HandleMessage routes incoming WebSocket messages to the appropriate handler.
// The clientID is passed for handlers that need per-client tracking (e.g., RSS configuration).
// Uses json.RawMessage to defer parsing of the "data" field until the specific handler needs it.
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

	slog.Info("WS event received", "type", msgType, "data", string(data))

	switch msgType {
	case "host-register":
		slog.Info("Host agent registered")
	case "heartbeat":
		s.BroadcastJSON(map[string]any{"type": "heartbeat-ack"})
	case "simulate-button":
		handleButton(s, data)
	case "simulate-slider":
		handleSlider(s, data)
	case "simulate-joystick":
		handleJoystick(s, data)
	case "simulate-mousepad":
		handleMousepad(s, data)
	case "simulate-mousewheel":
		handleMousewheel(s, data)
	case "simulate-mousebtn":
		handleMousebtn(s, data)
	case "simulate-keyboard":
		handleKeyboard(s, data)
	case "save-joystick-count":
		handleJoystickCount(s, data)
	case "execute-command":
		handleCommand(s, parsed)
	case "push-data":
		handlePushData(s, data)
	case "enter-fullscreen":
		s.BroadcastJSON(map[string]any{"type": "enter-fullscreen"})
	case "exit-fullscreen":
		s.BroadcastJSON(map[string]any{"type": "exit-fullscreen"})
	case "start-recording":
		handleStartRecording(s, data)
	case "stop-recording":
		handleStopRecording(s)
	case "speech-config":
		handleSpeechConfig(s, data)
	case "register-speech-trigger":
		handleRegisterSpeechTrigger(s, data)
	case "rss-configure":
		handleRSSConfigure(s, clientID, data)
	case "open-url":
		handleOpenURL(s, data)
	default:
		slog.Info("Unhandled WS message type", "type", msgType)
	}
}

// parseUintField extracts a uint64 field from a JSON object.
// Tries parsing as number first, then as string (lenient for JavaScript clients).
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

// parseStringField extracts a string field from a JSON object.
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

// handleButton routes a button press/release to the joystick manager.
// If js or id are empty strings (unconfigured button), the event is ignored
// to prevent sending spurious input on joystick 0, button 0.
func handleButton(s state.AppStateInterface, data json.RawMessage) {
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
	s.GetJoystickManager().Send(devices.Command{
		Type:  devices.ButtonType,
		Js:    js,
		Id:    id,
		Value: state,
	})
}

// handleSlider routes a slider value change to the joystick manager as an axis event.
// If js or id are empty strings (unconfigured slider), the event is ignored
// to prevent sending spurious input on joystick 0, axis 0.
func handleSlider(s state.AppStateInterface, data json.RawMessage) {
	jsRaw := parseStringField(data, "js")
	idRaw := parseStringField(data, "id")
	value := uint8(parseUintField(data, "value"))

	if jsRaw == "" || idRaw == "" {
		slog.Debug("Slider event ignored (no joystick/axis configured)")
		return
	}

	js := int(parseUintField(data, "js"))
	id := int(parseUintField(data, "id"))

	s.GetJoystickManager().Send(devices.Command{
		Type:  devices.AxisType,
		Js:    js,
		Id:    id,
		Value: value,
	})
}

// handleJoystick routes a 2-axis joystick movement.
// Splits the {x, y} value into two separate axis commands (id for X, id+1 for Y).
// If js or id are empty strings (unconfigured joystick), the event is ignored
// to prevent sending spurious input on joystick 0.
func handleJoystick(s state.AppStateInterface, data json.RawMessage) {
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

	s.GetJoystickManager().Send(devices.Command{
		Type:  devices.AxisType,
		Js:    js,
		Id:    id,
		Value: uint8(x),
	})
	s.GetJoystickManager().Send(devices.Command{
		Type:  devices.AxisType,
		Js:    js,
		Id:    id + 1,
		Value: uint8(y),
	})
}

// handleJoystickCount updates the number of virtual joysticks at runtime.
func handleJoystickCount(s state.AppStateInterface, data json.RawMessage) {
	var count uint64
	json.Unmarshal(data, &count)
	s.UpdateJoystickCount(uint8(count))
}

// handleMousepad routes relative mouse movement to the mousepad manager.
func handleMousepad(s state.AppStateInterface, data json.RawMessage) {
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
	s.GetMousepadManager().SendMove(js, dx, dy)
}

// handleMousewheel routes scroll wheel events to the mousepad manager.
func handleMousewheel(s state.AppStateInterface, data json.RawMessage) {
	js := int(parseUintField(data, "js"))
	delta := int32(parseUintField(data, "delta"))

	s.GetMousepadManager().SendWheel(js, delta)
}

// handleMousebtn routes mouse button press/release to the mousepad manager.
// Maps button names ("left", "right", "middle") to platform-agnostic constants.
func handleMousebtn(s state.AppStateInterface, data json.RawMessage) {
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

	s.GetMousepadManager().SendButton(js, btn, state)
}

// handleKeyboard routes keyboard key/combo press/release to the keyboard manager.
// The data field uses "keyboard_index" (not "js") to select the virtual keyboard device.
// Supports single keys ("a", "w", "ctrl") and combinations ("ctrl+a", "ctrl+shift+a").
// Key names are matched case-insensitively against devices.KeyNameToCode.
//
// Common use case: the sequence_button block sends individual keys while
// holding "ctrl" (e.g., ctrl, then d, d, w, s, a, then ctrl release). The
// handler sends each as a separate event — state=1 for ctrl, then state=1/0
// for each WASD key, then state=0 for ctrl — allowing the game to receive
// the full combination as if the user held Ctrl and tapped WASD keys.
func handleKeyboard(s state.AppStateInterface, data json.RawMessage) {
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

	if len(codes) == 0 {
		slog.Warn("No valid keys in keyboard event")
		return
	}

	if len(codes) == 1 {
		slog.Info("Keyboard key event", "keyboard_index", kbIndex, "key", key, "state", keyState)
		s.GetKeyboardManager().SendKey(kbIndex, codes[0], keyState)
	} else {
		slog.Info("Keyboard combo event", "keyboard_index", kbIndex, "keys", key, "state", keyState)
		s.GetKeyboardManager().SendCombo(kbIndex, codes, keyState)
	}
}

// handleCommand executes a shell or HTTP command and broadcasts the result.
// Supports parameter substitution ({key} placeholders).
func handleCommand(s state.AppStateInterface, parsed map[string]json.RawMessage) {
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
		httpURL := ""
		if raw, ok := cmdObj["http_url"]; ok {
			json.Unmarshal(raw, &httpURL)
		}
		httpBody := ""
		if raw, ok := cmdObj["http_body"]; ok {
			json.Unmarshal(raw, &httpBody)
		}

		substitutedURL := commands.SubstituteParams(httpURL, params)
		substitutedBody := commands.SubstituteParams(httpBody, params)
		success, output = commands.ExecuteHTTP(httpMethod, substitutedURL, substitutedBody)
	default:
		success = false
		output = "Unknown command type: " + cmdType
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

// handlePushData adds a custom metric to the DataBus from a client message.
func handlePushData(s state.AppStateInterface, data json.RawMessage) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		slog.Warn("Failed to parse push-data", "error", err)
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
	if raw, ok := obj["unit"]; ok {
		json.Unmarshal(raw, &unit)
	}
	if raw, ok := obj["source"]; ok {
		json.Unmarshal(raw, &source)
	}

	if key == "" {
		slog.Warn("push-data missing key")
		return
	}

	if source != "" {
		s.GetDataBus().SetSource(key, value, unit, source)
	} else {
		s.GetDataBus().Set(key, value, unit)
	}

	slog.Info("Data pushed via WebSocket", "key", key, "value", value, "unit", unit, "source", source)
}

// handleStartRecording starts audio recording based on configured location.
func handleStartRecording(s state.AppStateInterface, data json.RawMessage) {
	if s.GetSpeechManager() == nil {
		slog.Warn("Speech manager not available")
		return
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		slog.Warn("Failed to parse start-recording", "error", err)
		return
	}

	mode := "push-to-talk"
	if raw, ok := obj["mode"]; ok {
		json.Unmarshal(raw, &mode)
	}

	cfg := s.GetSpeechManager().GetConfig()
	recordingLoc := "client"
	if cfg != nil {
		recordingLoc = cfg.RecordingLoc
	}

	if recordingLoc == "host" {
		if s.GetSpeechManager().IsHostRecording() {
			slog.Info("Host already recording (wake word mode active)")
			return
		}
		if err := s.GetSpeechManager().StartHostRecording(); err != nil {
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

// handleStopRecording stops recording and processes the audio.
func handleStopRecording(s state.AppStateInterface) {
	if s.GetSpeechManager() == nil {
		slog.Warn("Speech manager not available")
		return
	}

	s.BroadcastJSON(map[string]any{
		"type": "recording-status",
		"data": map[string]any{
			"state": "processing",
		},
	})

	cfg := s.GetSpeechManager().GetConfig()
	if cfg != nil && cfg.RecordingLoc == "host" {
		go func() {
			text, matched, speakText, err := s.GetSpeechManager().StopHostRecording()
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

// handleSpeechConfig updates speech settings from the client.
func handleSpeechConfig(s state.AppStateInterface, data json.RawMessage) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		slog.Warn("Failed to parse speech-config", "error", err)
		return
	}

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

// HandleAudioChunk processes incoming binary audio data from a client.
func HandleAudioChunk(s state.AppStateInterface, audioData []byte) {
	if s.GetSpeechManager() == nil {
		return
	}

	cfg := s.GetSpeechManager().GetConfig()
	if cfg != nil && cfg.RecordingLoc == "host" {
		return
	}

	slog.Info("Received audio chunk", "bytes", len(audioData))

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

	text, matched, speakText, err := s.GetSpeechManager().Process(pcm)
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

// handleRegisterSpeechTrigger registers a speech trigger for an existing block.
// It extracts the block_id, phrase, aliases, trigger type, joystick index,
// button ID, and axis ID from the WebSocket message and passes them to
// SpeechManager.RegisterBlockTrigger. The joystick index, button ID, and
// axis ID enable direct server-side execution without requiring the client
// to send a follow-up simulate-button/slider message.
func handleRegisterSpeechTrigger(s state.AppStateInterface, data json.RawMessage) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		slog.Warn("Failed to parse register-speech-trigger", "error", err)
		return
	}

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

	if s.GetSpeechManager() != nil {
		s.GetSpeechManager().RegisterBlockTrigger(blockID, phrase, aliases, triggerType, joystickIndex, buttonID, axisID)
		slog.Info("Registered speech trigger", "block_id", blockID, "phrase", phrase, "type", triggerType, "joystick_index", joystickIndex)
	}
}

// handleRSSConfigure sets up RSS feed polling for a block.
// It parses the block_id, feed_urls (supporting "URL|Label" format), refresh_interval,
// and max_entries from the WebSocket message and passes them to the RSS manager
// along with the clientID for per-client seen-entry tracking.
// The manager starts an immediate fetch and schedules periodic polling.
func handleRSSConfigure(s state.AppStateInterface, clientID uint64, data json.RawMessage) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		slog.Warn("Failed to parse rss-configure", "error", err)
		return
	}

	var blockID string
	if raw, ok := obj["block_id"]; ok {
		json.Unmarshal(raw, &blockID)
	}

	var feedURLs []string
	if raw, ok := obj["feed_urls"]; ok {
		json.Unmarshal(raw, &feedURLs)
	}

	refreshInterval := int(parseUintField(data, "refresh_interval"))
	maxEntries := int(parseUintField(data, "max_entries"))

	if blockID == "" || len(feedURLs) == 0 {
		slog.Warn("rss-configure missing block_id or feed_urls")
		return
	}

	if s.GetRSSManager() != nil {
		s.GetRSSManager().Configure(blockID, clientID, feedURLs, refreshInterval, maxEntries)
	}
}

// handleOpenURL opens a URL in the host's default browser.
// It uses xdg-open on Linux and the start command on Windows.
// Called when a user clicks an RSS feed entry and the block's
// open_url_location setting is "host". When set to "client",
// the URL is opened directly in the panel browser via window.open.
func handleOpenURL(s state.AppStateInterface, data json.RawMessage) {
	url := parseStringField(data, "url")
	if url == "" {
		slog.Warn("open-url missing url")
		return
	}

	slog.Info("Opening URL on host", "url", url)
	if err := rssfeed.OpenURL(url); err != nil {
		slog.Warn("Failed to open URL", "url", url, "error", err)
	}
}
