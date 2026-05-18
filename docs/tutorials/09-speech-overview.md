# Chapter 9: Speech Recognition Overview

## What This Package Does

The `speech` package adds voice-controlled commands to OmniPanel-go. It handles:
- Speech-to-text (STT) via pluggable backends (Vosk offline, llama-cpp HTTP)
- Grammar-constrained recognition for improved accuracy (Vosk)
- Phrase matching with parameter extraction
- Command execution (shell, HTTP, button press, slider change)
- Security allowlisting
- Audio recording from the host microphone

It's designed as a self-contained package with a clean interface that the rest of the application interacts with through `SpeechManager`.

## The STTEngine Interface

```go
// internal/speech/speech.go
type STTEngine interface {
    Recognize(pcm []byte) (string, error)
    SetGrammar(grammar string)
    Close()
}
```

> **Concept: Interfaces in Go**
> An interface defines a set of methods that a type must implement. Any type that has these methods automatically satisfies the interface — no explicit declaration needed. This is called **structural typing**. The `STTEngine` interface says: "I need something that can recognize audio, optionally accept a grammar constraint, and clean up after itself." Both `voskEngine` and `llamaEngine` satisfy this.

## Grammar-Constrained Recognition (Vosk)

Vosk supports **grammar-constrained recognition**, which dramatically improves accuracy by restricting the recognizer to only output phrases from a predefined list. This is especially useful when you have a fixed set of commands.

```go
// internal/speech/speech.go
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
```

> **Concept: Grammar constraints**
> Vosk's `NewRecognizerGrm` creates a recognizer with a JSON array of allowed phrases. Instead of trying to recognize any English word, it builds a finite state transducer that only matches your defined phrases. This reduces false positives and improves accuracy significantly. The grammar is built from all speech command phrases, aliases, and the wake word.

The grammar can be updated at runtime via `SetGrammar()`, which is called whenever new speech triggers are registered:

```go
// internal/speech/vosk.go
func (e *voskEngine) SetGrammar(grammar string) {
    if e.recognizer != nil {
        e.recognizer.SetGrm(grammar)
        e.grammar = grammar
        slog.Info("Vosk grammar updated", "phrases_count", strings.Count(grammar, ",")+1)
    } else {
        slog.Warn("Cannot set grammar: recognizer not initialized")
    }
}
```

> **Key Pattern: Dynamic grammar updates**
> When blocks register speech triggers via `RegisterBlockTrigger()`, the grammar is rebuilt and pushed to the recognizer. This allows new phrases to be recognized without restarting the application. The `SetGrm()` method updates the recognizer's internal grammar in place.

## The SpeechManager

```go
type SpeechManager struct {
    engine         STTEngine
    commands       []SpeechCommand
    blockTriggers  map[string]SpeechCommand
    blockIDMap     map[string]*SpeechCommand
    config         *config.SpeechConfig
    broadcaster    Broadcaster
    joystickMgr    *devices.JoystickManager
    userPath       string
}
```

The `SpeechManager` is the central coordinator. It:
- Holds the STT engine (Vosk or llama-cpp)
- Loads speech commands from `user/speech_commands.json`
- Extracts button/axis IDs and joystick indices from all panel JSON files at startup
- Builds the grammar for constrained recognition
- Matches transcribed text against commands
- Executes matched commands (shell, HTTP, or direct joystick input)
- Broadcasts results to clients for visual feedback

The `joystickMgr` field enables button and slider commands to execute directly
via the virtual joystick subsystem, so they work even when no browser panel is
open. The `blockIDMap` provides O(1) lookup of commands by block ID for fast
execution.

## The Broadcaster Interface

```go
type Broadcaster interface {
    BroadcastJSON(msg map[string]any)
}
```

> **Key Pattern: Interface to break import cycles**
> The `speech` package needs to broadcast results to WebSocket clients, but importing `state` would create a circular dependency (`state` imports `speech`, `speech` would import `state`). Instead, we define a minimal `Broadcaster` interface. `AppState` implements it (it already has `BroadcastJSON`), and we pass it to `SpeechManager` at construction time. This is Go's standard approach to dependency injection.

## Initialization Flow

```go
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

    return sm
}
```

Step by step:
1. Create the manager with config, broadcaster, and joystick manager
2. If speech is disabled, return early (no engine, no commands)
3. Load commands from JSON file (creates default if missing)
4. Extract button/axis IDs and joystick indices from all panel JSON files in `user/panels/`
5. Initialize the STT engine (Vosk downloads model if needed, builds grammar)
6. Start host wake word loop if configured (host recording + wake-word mode)

## The SpeechCommand Struct

```go
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
```

This struct mirrors the JSON format in `speech_commands.json`. The `Type` field determines execution mode:
- `"shell"` → run shell command
- `"http"` → send HTTP request
- `"button"` → simulate button press via the JoystickManager
- `"slider"` → set slider value via the JoystickManager

For button and slider types, `JoystickIndex`, `ButtonID`, and `AxisID` are
populated from the panel's block settings at startup. The `joystick` field in
the block's settings JSON determines which virtual joystick device the command
targets, while `button` and `axis` identify the specific control on that device.
This allows speech commands defined in `speech_commands.json` (which only specify
`block_id`) to execute correctly even when no browser panel is open. When a block
registers a speech trigger via the editor, these fields are sent directly in the
`register-speech-trigger` WebSocket message.

## The Process Method

```go
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
```

This is the main entry point. It:
1. Validates the engine is ready
2. Sends audio to the STT engine for transcription
3. Matches the text against commands
4. Executes the matched command
5. Returns the transcription, match status, and TTS text

> **Concept: Multiple return values**
> Go functions can return multiple values. `Process` returns four: the transcribed text, whether a command matched, the TTS confirmation text, and any error. This is idiomatic Go — instead of creating a result struct, we return values directly. The caller unpacks them: `text, matched, speak, err := sm.Process(audio)`.

## Key Takeaways

- The `STTEngine` interface enables pluggable speech backends
- Vosk uses grammar-constrained recognition for improved accuracy
- Grammar is built from all phrases, aliases, and the wake word
- Grammar can be updated at runtime via `SetGrammar()`
- `Broadcaster` interface breaks import cycles between packages
- `SpeechManager` coordinates loading, recognition, matching, and execution
- Button/slider commands execute directly via the JoystickManager, working even without a browser panel open
- Button/axis IDs and joystick indices are extracted from panel JSON files at startup
- The `Process` method is the single entry point for speech recognition
- Multiple return values are idiomatic Go for returning related data

[← Back: Chapter 8](08-virtual-input.md) · [Next: Chapter 10 →](10-speech-engines.md)
