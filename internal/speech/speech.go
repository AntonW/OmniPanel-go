// Package speech adds voice-controlled commands to OmniPanel-go. It handles:
//   - Speech-to-text (STT) via pluggable backends (Vosk offline, llama-cpp HTTP)
//   - Grammar-constrained recognition for Vosk, improving accuracy by restricting
//     output to user-defined phrases, aliases, and wake words
//   - Phrase matching with parameter extraction and fuzzy matching
//   - Command execution (shell, HTTP, button press, slider change)
//   - Security allowlisting to restrict which phrases can execute
//   - Audio recording from the host microphone (malgo, cross-platform)
//
// The STTEngine interface enables swapping between backends without changing
// the SpeechManager code. The Broadcaster interface breaks import cycles
// between the speech and state packages.
//
// Audio recording can happen on the client (browser MediaRecorder, sent as
// binary WebSocket frames) or on the host (malgo library capturing from the
// PC's microphone), controlled by the recording_location config setting.
//
// Button and slider commands execute directly via the JoystickManager on the
// server, so they work even when no browser panel is open. The server also
// broadcasts speech-button-trigger and speech-slider-trigger messages to
// connected clients for visual feedback (button flash animation).
//
// See docs/tutorials/09-speech-overview.md through 12-audio-recording.md
// for a detailed walkthrough.
package speech

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"omnipanel-go/internal/commands"
	"omnipanel-go/internal/config"
	"omnipanel-go/internal/devices"
)

// STTEngine defines the interface for speech-to-text backends.
// Implementations must support audio recognition and optional grammar
// constraints for improved accuracy.
type STTEngine interface {
	Recognize(pcm []byte) (string, error)
	SetGrammar(grammar string)
	Close()
}

// Broadcaster defines the interface for broadcasting messages to clients.
type Broadcaster interface {
	BroadcastJSON(msg map[string]any)
}

// SpeechCommand represents a voice-triggered action.
// For button and slider types, JoystickIndex, ButtonID, and AxisID are
// populated either from speech_commands.json or extracted from panel JSON
// files at startup. This allows speech commands to execute directly via the
// JoystickManager even when no browser panel is open.
type SpeechCommand struct {
	Phrase        string          `json:"phrase"`
	Aliases       []string        `json:"aliases"`
	Type          string          `json:"type"`
	Command       string          `json:"command"`
	HTTPMethod    string          `json:"http_method"`
	HTTPURL       string          `json:"http_url"`
	HTTPBody      string          `json:"http_body"`
	BlockID       string          `json:"block_id"`
	JoystickIndex int             `json:"joystick_index"`
	ButtonID      int             `json:"button_id"`
	AxisID        int             `json:"axis_id"`
	ValuePattern  string          `json:"value_pattern"`
	Params        json.RawMessage `json:"params"`
}

// buildGrammar creates a JSON array of unique phrases from all speech commands
// and the wake word, used for grammar-constrained recognition in Vosk.
// Duplicate phrases are removed to keep the grammar compact.
func buildGrammar(commands []SpeechCommand, wakeWord string) string {
	seen := make(map[string]bool)
	phrases := []string{}

	for _, cmd := range commands {
		if !seen[cmd.Phrase] {
			seen[cmd.Phrase] = true
			phrases = append(phrases, cmd.Phrase)
		}
		for _, alias := range cmd.Aliases {
			if !seen[alias] {
				seen[alias] = true
				phrases = append(phrases, alias)
			}
		}
	}

	if wakeWord != "" && !seen[wakeWord] {
		phrases = append(phrases, wakeWord)
	}

	data, _ := json.Marshal(phrases)
	return string(data)
}

// SpeechManager orchestrates speech recognition, matching, and command execution.
// It holds the STT engine, loaded commands, and a blockIDMap for fast lookup
// of button/slider commands by their block ID. The joystickMgr field enables
// direct button/slider execution without requiring a browser panel to be open.
// The blockTriggers map indexes commands by phrase/alias for quick matching.
type SpeechManager struct {
	engine        STTEngine
	commands      []SpeechCommand
	blockTriggers map[string]SpeechCommand
	blockIDMap    map[string]*SpeechCommand
	config        *config.SpeechConfig
	broadcaster   Broadcaster
	joystickMgr   *devices.JoystickManager
	userPath      string
	recorder      *HostRecorder

	wakeWordDone chan struct{}
	wakeWordWg   sync.WaitGroup
}

// New creates a SpeechManager, loads commands from speech_commands.json,
// extracts button/axis IDs and joystick indices from all panel JSON files,
// initializes the STT engine, and starts the host wake word loop if configured.
// The joystickMgr parameter enables direct button/slider execution without a
// browser panel.
func New(cfg *config.SpeechConfig, broadcaster Broadcaster, joystickMgr *devices.JoystickManager, userPath string) *SpeechManager {
	sm := &SpeechManager{
		config:        cfg,
		broadcaster:   broadcaster,
		joystickMgr:   joystickMgr,
		userPath:      userPath,
		blockTriggers: make(map[string]SpeechCommand),
		blockIDMap:    make(map[string]*SpeechCommand),
	}

	if !cfg.Enabled {
		slog.Info("Speech recognition disabled")
		return sm
	}

	if err := sm.loadCommands(); err != nil {
		slog.Warn("Failed to load speech commands", "error", err)
	}

	if err := sm.initEngine(); err != nil {
		slog.Error("Failed to initialize STT engine", "error", err)
	}

	if cfg.RecordingLoc == "host" {
		var err error
		sm.recorder, err = NewHostRecorder()
		if err != nil {
			slog.Warn("Failed to initialize host recorder", "error", err)
		} else {
			slog.Info("Host recorder initialized")
		}
	}

	if cfg.RecordingLoc == "host" && cfg.TriggerMode == "wake-word" && sm.recorder != nil && sm.engine != nil {
		sm.wakeWordDone = make(chan struct{})
		sm.wakeWordWg.Add(1)
		go sm.runHostWakeWordLoop()
	}

	slog.Info("Speech manager initialized",
		"engine", cfg.STTEngine,
		"commands", len(sm.commands),
		"block_triggers", len(sm.blockTriggers),
	)

	return sm
}

// loadCommands reads speech_commands.json from the user directory.
func (sm *SpeechManager) loadCommands() error {
	path := filepath.Join(sm.userPath, "speech_commands.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("No speech_commands.json found, creating default")
			return sm.createDefaultCommands(path)
		}
		return fmt.Errorf("read speech commands: %w", err)
	}

	if err := json.Unmarshal(data, &sm.commands); err != nil {
		return fmt.Errorf("parse speech commands: %w", err)
	}

	for i := range sm.commands {
		if sm.commands[i].BlockID != "" {
			sm.blockTriggers[sm.commands[i].Phrase] = sm.commands[i]
			for _, alias := range sm.commands[i].Aliases {
				sm.blockTriggers[alias] = sm.commands[i]
			}
			sm.blockIDMap[sm.commands[i].BlockID] = &sm.commands[i]
		}
	}

	sm.loadPanelButtonIDs()

	slog.Info("Loaded speech commands", "count", len(sm.commands))
	return nil
}

// createDefaultCommands writes a template speech_commands.json.
func (sm *SpeechManager) createDefaultCommands(path string) error {
	defaults := []SpeechCommand{
		{
			Phrase:  "hello",
			Aliases: []string{"hi", "hey"},
			Type:    "shell",
			Command: "echo Hello from OmniPanel-go",
		},
	}

	data, err := json.MarshalIndent(defaults, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// initEngine creates the configured STT backend.
func (sm *SpeechManager) initEngine() error {
	switch sm.config.STTEngine {
	case "vosk":
		return sm.initVosk()
	case "llama-cpp":
		sm.engine = newLlamaEngine(sm.config)
		return nil
	default:
		return fmt.Errorf("unknown STT engine: %s", sm.config.STTEngine)
	}
}

// initVosk initializes the Vosk STT engine with grammar-constrained recognition.
// On Windows it also ensures required Vosk runtime DLLs are available,
// downloading and extracting them if needed. It then downloads the model if
// needed, builds a grammar from loaded commands and the wake word, and creates
// the recognizer.
func (sm *SpeechManager) initVosk() error {
	if err := ensureVoskWindowsRuntime(sm.userPath, sm.config.VoskRuntimeURL); err != nil {
		return fmt.Errorf("ensure vosk runtime: %w", err)
	}

	modelPath, err := EnsureVoskModel(sm.config.VoskModelPath, sm.userPath)
	if err != nil {
		return fmt.Errorf("ensure vosk model: %w", err)
	}

	sm.config.VoskModelPath = modelPath
	grammar := buildGrammar(sm.commands, sm.config.WakeWord)
	engine, err := newVoskEngine(sm.config, grammar)
	if err != nil {
		return err
	}

	sm.engine = engine
	return nil
}

// Process transcribes audio and executes the matched command.
// Returns the transcription text, whether a command matched, and any error.
func (sm *SpeechManager) Process(audio []byte) (string, bool, string, error) {
	if sm.engine == nil {
		return "", false, "", fmt.Errorf("STT engine not initialized")
	}

	text, err := sm.engine.Recognize(audio)
	if err != nil {
		return text, false, "", fmt.Errorf("recognition failed: %w", err)
	}

	slog.Info("Speech recognized", "text", text)

	matched, speakText := sm.matchAndExecute(text)
	return text, matched, speakText, nil
}

// matchAndExecute finds a matching command and executes it.
// Returns whether a match was found and the TTS confirmation text.
func (sm *SpeechManager) matchAndExecute(text string) (bool, string) {
	if !sm.isAllowed(text) {
		slog.Warn("Speech command blocked by allowlist", "text", text)
		return false, ""
	}

	for _, cmd := range sm.commands {
		if matched, params := matchPhrase(cmd.Phrase, text); matched {
			return sm.executeCommand(cmd, params)
		}
		for _, alias := range cmd.Aliases {
			if matched, params := matchPhrase(alias, text); matched {
				return sm.executeCommand(cmd, params)
			}
		}
	}

	if fuzzyMatch, cmd := fuzzyMatchCommand(text, sm.commands); fuzzyMatch {
		return sm.executeCommand(cmd, nil)
	}

	return false, ""
}

// isAllowed checks if the text passes the allowlist (if configured).
func (sm *SpeechManager) isAllowed(text string) bool {
	if len(sm.config.SpeechAllowlist) == 0 {
		return true
	}

	for _, pattern := range sm.config.SpeechAllowlist {
		if matchAllowlist(pattern, text) {
			return true
		}
	}
	return false
}

// executeCommand routes a matched speech command to the appropriate executor.
func (sm *SpeechManager) executeCommand(cmd SpeechCommand, params map[string]string) (bool, string) {
	speakText := cmd.Phrase

	switch cmd.Type {
	case "shell":
		substituted := cmd.Command
		if params != nil {
			paramsJSON, _ := json.Marshal(params)
			substituted = commands.SubstituteParams(cmd.Command, paramsJSON)
		}
		success, output := commands.ExecuteShell(substituted)
		slog.Info("Speech shell command", "success", success, "output", output)
		return success, speakText

	case "http":
		substitutedURL := cmd.HTTPURL
		substitutedBody := cmd.HTTPBody
		if params != nil {
			paramsJSON, _ := json.Marshal(params)
			substitutedURL = commands.SubstituteParams(cmd.HTTPURL, paramsJSON)
			substitutedBody = commands.SubstituteParams(cmd.HTTPBody, paramsJSON)
		}
		success, output := commands.ExecuteHTTP(cmd.HTTPMethod, substitutedURL, substitutedBody)
		slog.Info("Speech HTTP command", "success", success, "output", output)
		return success, speakText

	case "button":
		if cmd.BlockID != "" {
			sm.simulateButton(cmd.BlockID)
			return true, speakText
		}

	case "slider":
		if cmd.BlockID != "" && params != nil {
			if valStr, ok := params["value"]; ok {
				sm.simulateSlider(cmd.BlockID, valStr)
				return true, speakText
			}
		}
	}

	return false, ""
}

// simulateButton executes a button press via the JoystickManager and broadcasts
// a speech-button-trigger message to connected clients for visual feedback.
// The server-side execution ensures the button press works even when no
// browser panel is open.
func (sm *SpeechManager) simulateButton(blockID string) {
	slog.Info("Speech simulating button", "block_id", blockID)

	if sm.joystickMgr == nil {
		slog.Warn("Speech button skipped: joystick manager is nil")
	} else if cmd, ok := sm.blockIDMap[blockID]; !ok {
		slog.Warn("Speech button execution failed: block not found in map", "block_id", blockID)
	} else if cmd.Type != "button" {
		slog.Warn("Speech button execution failed: wrong type", "block_id", blockID, "type", cmd.Type)
	} else {
		slog.Info("Speech button executing", "block_id", blockID, "joystick", cmd.JoystickIndex, "button", cmd.ButtonID)
		sm.joystickMgr.Send(devices.Command{
			Type:  devices.ButtonType,
			Js:    cmd.JoystickIndex,
			Id:    cmd.ButtonID,
			Value: 1,
		})
		time.Sleep(200 * time.Millisecond)
		sm.joystickMgr.Send(devices.Command{
			Type:  devices.ButtonType,
			Js:    cmd.JoystickIndex,
			Id:    cmd.ButtonID,
			Value: 0,
		})
	}

	if sm.broadcaster != nil {
		sm.broadcaster.BroadcastJSON(map[string]any{
			"type": "speech-button-trigger",
			"data": map[string]any{
				"block_id": blockID,
			},
		})
	}
}

// simulateSlider executes a slider change via the JoystickManager and broadcasts
// a speech-slider-trigger message to connected clients for visual feedback.
// The server-side execution ensures the slider change works even when no
// browser panel is open.
func (sm *SpeechManager) simulateSlider(blockID, valueStr string) {
	slog.Info("Speech simulating slider", "block_id", blockID, "value", valueStr)

	value := 128
	if v, err := fmt.Sscanf(valueStr, "%d", &value); v == 0 || err != nil {
		value = 128
	}
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}

	if sm.joystickMgr != nil {
		if cmd, ok := sm.blockIDMap[blockID]; ok && cmd.Type == "slider" {
			sm.joystickMgr.Send(devices.Command{
				Type:  devices.AxisType,
				Js:    cmd.JoystickIndex,
				Id:    cmd.AxisID,
				Value: uint8(value),
			})
		}
	}

	if sm.broadcaster != nil {
		sm.broadcaster.BroadcastJSON(map[string]any{
			"type": "speech-slider-trigger",
			"data": map[string]any{
				"block_id": blockID,
				"value":    valueStr,
			},
		})
	}
}

// blockInfo holds hardware IDs extracted from a panel block's settings.
// It captures the button or axis ID, the joystick index it belongs to,
// and flags indicating whether the block is a button or slider.
type blockInfo struct {
	buttonID    int
	axisID      int
	joystickIdx int
	isButton    bool
	isSlider    bool
}

// UpdateButtonIDsFromPanelJSON parses a v2 panel's blocks array to extract
// button IDs, axis IDs, and joystick indices, then updates any matching speech
// commands. This is called at startup for all panel files and again when a panel
// is loaded via the HTTP API. It ensures speech button/slider commands have the
// correct hardware IDs even when defined only by block_id in speech_commands.json.
// The panel JSON must use v2 format with a top-level "blocks" field.
func (sm *SpeechManager) UpdateButtonIDsFromPanelJSON(panelJSON []byte) {
	var panelV2 struct {
		Blocks []json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal(panelJSON, &panelV2); err != nil {
		slog.Warn("Failed to parse panel JSON for speech button IDs", "error", err)
		return
	}
	panel := panelV2.Blocks

	blocks := make(map[string]blockInfo)

	type blockEntry struct {
		ID       string            `json:"id"`
		Settings json.RawMessage   `json:"settings"`
		Blocks   []blockEntry      `json:"blocks"`
		Children []json.RawMessage `json:"children"`
	}

	var extract func(items []json.RawMessage)
	extract = func(items []json.RawMessage) {
		for _, raw := range items {
			var entry blockEntry
			if err := json.Unmarshal(raw, &entry); err != nil {
				continue
			}
			if entry.ID != "" && len(entry.Settings) > 0 {
				if info := parseBlockSettings(entry.Settings); info.isButton || info.isSlider {
					blocks[entry.ID] = info
				}
			}
			for _, block := range entry.Blocks {
				if block.ID != "" && len(block.Settings) > 0 {
					if info := parseBlockSettings(block.Settings); info.isButton || info.isSlider {
						blocks[block.ID] = info
					}
				}
			}
			if len(entry.Children) > 0 {
				extract(entry.Children)
			}
		}
	}
	extract(panel)

	slog.Info("Panel block extraction complete", "blocks_found", len(blocks), "commands_to_check", len(sm.commands))

	updated := 0
	for i := range sm.commands {
		if info, ok := blocks[sm.commands[i].BlockID]; ok {
			if info.isButton && sm.commands[i].Type == "button" {
				slog.Info("Updated button ID from panel", "phrase", sm.commands[i].Phrase, "block_id", sm.commands[i].BlockID, "button_id", info.buttonID, "joystick", info.joystickIdx)
				sm.commands[i].ButtonID = info.buttonID
				sm.commands[i].JoystickIndex = info.joystickIdx
				updated++
			}
			if info.isSlider && sm.commands[i].Type == "slider" {
				sm.commands[i].AxisID = info.axisID
				sm.commands[i].JoystickIndex = info.joystickIdx
				updated++
			}
		}
	}

	for blockID, cmd := range sm.blockIDMap {
		if info, ok := blocks[blockID]; ok {
			if info.isButton && cmd.Type == "button" {
				cmd.ButtonID = info.buttonID
				cmd.JoystickIndex = info.joystickIdx
				updated++
			}
			if info.isSlider && cmd.Type == "slider" {
				cmd.AxisID = info.axisID
				cmd.JoystickIndex = info.joystickIdx
				updated++
			}
		}
	}

	slog.Info("Updated speech button/slider IDs from panel", "updated", updated)
}

// parseBlockSettings extracts button ID, axis ID, and joystick index from a
// block's settings JSON. It returns a blockInfo with isButton or isSlider set
// based on which hardware control the block represents.
func parseBlockSettings(settingsJSON json.RawMessage) blockInfo {
	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return blockInfo{}
	}

	info := blockInfo{}
	if btn, ok := settings["button"]; ok {
		switch v := btn.(type) {
		case float64:
			info.buttonID = int(v)
			info.isButton = true
		case string:
			if n, err := fmt.Sscanf(v, "%d", &info.buttonID); n > 0 && err == nil {
				info.isButton = true
			}
		}
	}
	if axis, ok := settings["axis"]; ok {
		switch v := axis.(type) {
		case float64:
			info.axisID = int(v)
			info.isSlider = true
		case string:
			if n, err := fmt.Sscanf(v, "%d", &info.axisID); n > 0 && err == nil {
				info.isSlider = true
			}
		}
	}
	if js, ok := settings["joystick"]; ok {
		switch v := js.(type) {
		case float64:
			info.joystickIdx = int(v)
		case string:
			fmt.Sscanf(v, "%d", &info.joystickIdx)
		}
	}
	return info
}

// loadPanelButtonIDs scans all panel JSON files and updates speech commands with button/axis IDs and joystick indices.
func (sm *SpeechManager) loadPanelButtonIDs() {
	panelsDir := filepath.Join(sm.userPath, "panels")
	entries, err := os.ReadDir(panelsDir)
	if err != nil {
		slog.Warn("No panels directory found, skipping button ID extraction", "dir", panelsDir, "error", err)
		return
	}

	slog.Info("Scanning panels for speech button IDs", "dir", panelsDir, "files", len(entries))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		panelPath := filepath.Join(panelsDir, entry.Name())
		panelJSON, err := os.ReadFile(panelPath)
		if err != nil {
			slog.Warn("Failed to read panel for speech button IDs", "file", entry.Name(), "error", err)
			continue
		}
		slog.Info("Parsing panel for button IDs", "file", entry.Name(), "size", len(panelJSON))
		sm.UpdateButtonIDsFromPanelJSON(panelJSON)
	}
}

// RegisterBlockTrigger registers a speech trigger for an existing block,
// storing the joystick index, button ID, and axis ID for direct execution.
// If a command with the same block_id already exists (from speech_commands.json),
// it updates that command in place instead of creating a duplicate. The Vosk
// grammar is rebuilt to include the new phrase and aliases.
func (sm *SpeechManager) RegisterBlockTrigger(blockID, phrase string, aliases []string, triggerType string, joystickIndex, buttonID, axisID int) {
	if existing, ok := sm.blockIDMap[blockID]; ok {
		slog.Info("Updating existing speech trigger", "block_id", blockID, "phrase", phrase, "button_id", buttonID, "axis_id", axisID, "old_button_id", existing.ButtonID)
		existing.JoystickIndex = joystickIndex
		existing.ButtonID = buttonID
		existing.AxisID = axisID
		sm.blockTriggers[phrase] = *existing
		for _, alias := range aliases {
			sm.blockTriggers[alias] = *existing
		}
		return
	}

	cmd := SpeechCommand{
		Phrase:        phrase,
		Aliases:       aliases,
		Type:          triggerType,
		BlockID:       blockID,
		JoystickIndex: joystickIndex,
		ButtonID:      buttonID,
		AxisID:        axisID,
	}
	sm.commands = append(sm.commands, cmd)
	sm.blockIDMap[blockID] = &sm.commands[len(sm.commands)-1]
	sm.blockTriggers[phrase] = cmd
	for _, alias := range aliases {
		sm.blockTriggers[alias] = cmd
	}
	slog.Info("Registered new speech trigger", "block_id", blockID, "phrase", phrase, "button_id", buttonID, "axis_id", axisID)

	if sm.engine != nil {
		sm.engine.SetGrammar(buildGrammar(sm.commands, sm.config.WakeWord))
	}
}

// runHostWakeWordLoop continuously listens for the wake word on the host microphone.
func (sm *SpeechManager) runHostWakeWordLoop() {
	defer sm.wakeWordWg.Done()
	slog.Info("Host wake word detection started", "wake_word", sm.config.WakeWord)

	for {
		select {
		case <-sm.wakeWordDone:
			slog.Info("Host wake word loop stopped")
			return
		default:
		}

		if err := sm.listenForWakeWord(); err != nil {
			slog.Error("Host wake word listening error", "error", err)
			select {
			case <-sm.wakeWordDone:
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}

// listenForWakeWord starts a single wake word detection session.
func (sm *SpeechManager) listenForWakeWord() error {
	if err := sm.recorder.Start(); err != nil {
		return fmt.Errorf("start recorder: %w", err)
	}

	wakeWord := strings.ToLower(sm.config.WakeWord)
	const maxListen = 30 * time.Second
	listenStart := time.Now()

	for time.Since(listenStart) < maxListen {
		select {
		case <-sm.wakeWordDone:
			sm.recorder.Stop()
			return nil
		case <-time.After(500 * time.Millisecond):
		}

		sm.recorder.mu.Lock()
		chunks := make([][]byte, len(sm.recorder.chunks))
		copy(chunks, sm.recorder.chunks)
		sm.recorder.chunks = nil
		sm.recorder.mu.Unlock()

		if len(chunks) == 0 {
			continue
		}

		var pcm []byte
		for _, chunk := range chunks {
			pcm = append(pcm, chunk...)
		}

		if len(pcm) < 3200 {
			continue
		}

		text, err := sm.engine.Recognize(pcm)
		if err != nil || text == "" {
			continue
		}

		slog.Info("Wake word listening", "text", text)

		if strings.Contains(strings.ToLower(text), wakeWord) {
			slog.Info("Wake word detected on host", "text", text)
			sm.recorder.mu.Lock()
			sm.recorder.chunks = nil
			sm.recorder.mu.Unlock()

			if sm.broadcaster != nil {
				sm.broadcaster.BroadcastJSON(map[string]any{
					"type": "recording-status",
					"data": map[string]any{
						"state":    "listening",
						"mode":     "wake-word",
						"location": "host",
					},
				})
			}

			listenSec := sm.config.WakeWordListenSec
			if listenSec <= 0 {
				listenSec = 8
			}
			slog.Info("Listening for follow-up speech", "seconds", listenSec)
			select {
			case <-sm.wakeWordDone:
				sm.recorder.Stop()
				return nil
			case <-time.After(time.Duration(listenSec) * time.Second):
			}

			sm.recorder.mu.Lock()
			postChunks := make([][]byte, len(sm.recorder.chunks))
			copy(postChunks, sm.recorder.chunks)
			postCount := len(postChunks)
			sm.recorder.chunks = nil
			sm.recorder.mu.Unlock()

			sm.recorder.Stop()

			var postPCM []byte
			for _, chunk := range postChunks {
				postPCM = append(postPCM, chunk...)
			}

			slog.Info("Processing follow-up speech", "chunks", postCount, "bytes", len(postPCM))

			if len(postPCM) > 0 {
				text, matched, speakText, err := sm.Process(postPCM)
				slog.Info("Follow-up recognition result", "text", text, "matched", matched)
				if err != nil {
					slog.Error("Host wake word processing failed", "error", err)
					if sm.broadcaster != nil {
						sm.broadcaster.BroadcastJSON(map[string]any{
							"type": "speech-error",
							"data": map[string]any{"error": err.Error()},
						})
					}
					return nil
				}

				result := map[string]any{
					"text":    text,
					"matched": matched,
				}
				if speakText != "" {
					result["speak"] = speakText + " confirmed"
				}

				if sm.broadcaster != nil {
					sm.broadcaster.BroadcastJSON(map[string]any{
						"type": "speech-result",
						"data": result,
					})
				}
			}

			if sm.broadcaster != nil {
				sm.broadcaster.BroadcastJSON(map[string]any{
					"type": "recording-status",
					"data": map[string]any{
						"state": "idle",
					},
				})
			}

			return nil
		}
	}

	sm.recorder.Stop()
	return nil
}

// Close releases the STT engine resources.
func (sm *SpeechManager) Close() {
	if sm.wakeWordDone != nil {
		close(sm.wakeWordDone)
		sm.wakeWordWg.Wait()
	}
	if sm.engine != nil {
		sm.engine.Close()
	}
	if sm.recorder != nil {
		sm.recorder.Close()
	}
}

// StartHostRecording begins recording from the host microphone.
func (sm *SpeechManager) StartHostRecording() error {
	if sm.recorder == nil {
		return fmt.Errorf("host recorder not initialized")
	}
	return sm.recorder.Start()
}

// StopHostRecording stops host recording, processes the audio, and returns the result.
func (sm *SpeechManager) StopHostRecording() (string, bool, string, error) {
	if sm.recorder == nil {
		return "", false, "", fmt.Errorf("host recorder not initialized")
	}

	pcm, err := sm.recorder.Stop()
	if err != nil {
		return "", false, "", fmt.Errorf("stop recording: %w", err)
	}

	if len(pcm) == 0 {
		return "", false, "", fmt.Errorf("no audio recorded")
	}

	return sm.Process(pcm)
}

// IsHostRecording returns whether host recording is in progress.
func (sm *SpeechManager) IsHostRecording() bool {
	if sm.recorder == nil {
		return false
	}
	return sm.recorder.IsRecording()
}

// GetConfig returns the speech configuration.
func (sm *SpeechManager) GetConfig() *config.SpeechConfig {
	return sm.config
}
