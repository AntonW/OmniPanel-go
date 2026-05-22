# Chapter 10: STT Engines — Vosk & llama-cpp

## What This Chapter Covers

OmniPanel-go supports two speech-to-text backends through the `STTEngine` interface:
1. **Vosk** — fully offline, downloads a model automatically, uses grammar-constrained recognition
2. **llama-cpp-server** — HTTP API, supports both Whisper-compatible and chat completions modes

Both implement the same interface, so the rest of the code doesn't care which one is active.

## Vosk Engine (Offline)

### Setup

Vosk requires the shared library (`libvosk.so` on Linux, `vosk.dll` on Windows) and a model directory. The model is auto-downloaded on first startup:

```go
// internal/speech/download.go
func EnsureVoskModel(modelPath string, userPath string) (string, error) {
    if modelPath != "" {
        if isValidVoskModel(modelPath) {
            return modelPath, nil
        }
    }

    modelsDir := filepath.Join(userPath, "speech-models")
    targetDir := filepath.Join(modelsDir, "vosk-model-small-en-us-0.15")

    if isValidVoskModel(targetDir) {
        return targetDir, nil
    }

    // Download ~50MB zip, extract, validate
    // ...
}
```

> **Concept: Fallback with validation**
> `isValidVoskModel` checks for required subdirectories (`am`, `conf`, `ivector`, `graph`). This prevents partially downloaded or corrupted models from being used. The download-extract-validate pattern is common for auto-provisioned resources.

On Windows, the app also auto-provisions the native Vosk DLL runtime if missing:

```go
// internal/speech/vosk_runtime_windows.go
func ensureVoskWindowsRuntime(userPath string, runtimeURL string) error {
    if strings.TrimSpace(runtimeURL) == "" {
        runtimeURL = defaultVoskRuntimeURL
    }

    if hasVoskWindowsRuntime(userPath) {
        return ensureVoskRuntimeOnPath(userPath)
    }

    runtimeDir := filepath.Join(userPath, "speech-runtime", "vosk")
    zipPath := filepath.Join(runtimeDir, "vosk-win64.zip")
    if err := downloadFile(runtimeURL, zipPath); err != nil {
        return err
    }

    if err := extractSelectedVoskWindowsFiles(zipPath, runtimeDir); err != nil {
        return err
    }

    return ensureVoskRuntimeOnPath(userPath)
}
```

> **Key Pattern: Config override with default fallback (Go)**
> `runtimeURL` is read from `speech.vosk_runtime_url`. If it is empty, the code falls back to a safe built-in URL (`defaultVoskRuntimeURL`). This keeps first-run setup beginner-friendly while still allowing advanced deployments to host their own ZIP.

> **Concept: Runtime PATH bootstrapping (Go)**
> `ensureVoskRuntimeOnPath` prepends the extracted runtime directory to `PATH` at startup. This lets CGO-loaded libraries resolve `libvosk.dll` and `libstdc++-6.dll` without requiring users to manually copy files next to the EXE.

### Grammar-Constrained Recognition

Vosk supports **grammar-constrained recognition**, which restricts the recognizer to only output phrases from a predefined list. This dramatically improves accuracy for command-based use cases.

```go
// internal/speech/vosk.go (+build cgo)
func newVoskEngine(cfg *config.SpeechConfig, grammar string) (*voskEngine, error) {
    modelPath := cfg.VoskModelPath
    if modelPath == "" {
        return nil, fmt.Errorf("vosk model path not configured")
    }

    model, err := vosk.NewModel(modelPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load Vosk model from %s: %w", modelPath, err)
    }

    recognizer, err := vosk.NewRecognizerGrm(model, 16000.0, grammar)
    if err != nil {
        model.Free()
        return nil, fmt.Errorf("failed to create Vosk recognizer: %w", err)
    }

    slog.Info("Vosk engine initialized", "model", modelPath, "grammar_phrases", strings.Count(grammar, ",")+1)

    return &voskEngine{
        model:      model,
        recognizer: recognizer,
        sampleRate: 16000,
        grammar:    grammar,
    }, nil
}
```

> **Key Pattern: `NewRecognizerGrm` vs `NewRecognizer`**
> `vosk.NewRecognizerGrm(model, sampleRate, grammar)` creates a recognizer constrained to a JSON array of phrases. Instead of `vosk.NewRecognizer(model, sampleRate)` which tries to recognize any English word, the grammar-constrained version builds a finite state transducer that only matches your defined phrases. This reduces false positives and improves accuracy significantly.

The grammar is built from all speech command phrases, aliases, and the wake word:

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

> **Concept: Deduplication with maps**
> The `seen` map ensures each phrase appears only once in the grammar. If "hello" is both a phrase and an alias, it's added only once. This keeps the grammar compact and avoids redundant processing.

### Dynamic Grammar Updates

When new speech triggers are registered at runtime (e.g., via the editor), the grammar is updated:

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

> **Key Pattern: Runtime updates**
> `SetGrm()` updates the recognizer's grammar in place without recreating it. This allows new phrases to be recognized immediately when blocks are added via the editor. The `strings.Count(grammar, ",")+1` trick counts phrases by counting commas (a JSON array `["a","b","c"]` has 2 commas for 3 items).

### Recognition

```go
// internal/speech/vosk.go
func (e *voskEngine) Recognize(pcm []byte) (string, error) {
    e.recognizer.Reset()

    accepted := e.recognizer.AcceptWaveform(pcm)

    var result string
    if accepted != 0 {
        result = e.recognizer.Result()
    } else {
        result = e.recognizer.FinalResult()
    }

    var parsed struct {
        Text string `json:"text"`
    }
    if err := json.Unmarshal([]byte(result), &parsed); err != nil {
        return result, nil
    }

    return parsed.Text, nil
}
```

> **Concept: CGO and C libraries**
> Vosk is a C library. Go calls it through CGO — the `import "C"` block and `C.vosk_recognizer_accept_waveform()` calls. The Vosk Go bindings (`github.com/alphacep/vosk-api/go`) wrap the raw C API into Go-friendly methods. CGO requires a C compiler (GCC/Clang) at build time, which is why the build process needs MinGW on Windows. The file `vosk.go` has `//go:build cgo` at the top, and a companion `vosk_stub.go` with `//go:build !cgo` provides a stub for non-CGO builds (e.g., Docker containers).

The `AcceptWaveform` method returns non-zero when a complete utterance is detected (usually after silence). `Result()` returns the final transcription, while `FinalResult()` returns text without waiting for silence. The recognizer is reset before each recognition to clear any previous state.

## llama-cpp Engine (HTTP API)

### Two API Modes

The llama-cpp engine supports two modes, controlled by `llama_cpp_api_mode`:

**1. Transcriptions mode** (Whisper-compatible):

```go
func (e *llamaEngine) recognizeTranscriptions(pcm []byte) (string, error) {
    wavData, _ := pcmToWav(pcm, 16000)

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, _ := writer.CreateFormFile("file", "audio.wav")
    part.Write(wavData)
    writer.WriteField("model", e.model)
    writer.WriteField("response_format", "json")
    writer.Close()

    req, _ := http.NewRequest("POST", e.url+"/v1/audio/transcriptions", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    // ... send request, parse response
}
```

This matches the OpenAI Whisper API format. llama-cpp-server with a Whisper model responds identically to OpenAI's `/v1/audio/transcriptions` endpoint.

**2. Chat mode** (multimodal models like gemma-3):

```go
func (e *llamaEngine) recognizeChat(pcm []byte) (string, error) {
    wavData, _ := pcmToWav(pcm, 16000)
    audioBase64 := base64.StdEncoding.EncodeToString(wavData)

    requestBody := map[string]any{
        "model": e.model,
        "messages": []map[string]any{
            {"role": "system", "content": e.prompt},
            {"role": "user", "content": []map[string]any{
                {"type": "audio_url", "audio_url": map[string]string{
                    "url": "data:audio/wav;base64," + audioBase64,
                }},
            }},
        },
        "max_tokens":  512,
        "temperature": 0.0,
    }

    // POST to /v1/chat/completions
}
```

> **Concept: Data URIs**
> `"data:audio/wav;base64,..."` embeds the audio directly in the JSON request. This is a standard web format for inline data. It avoids multipart encoding but increases payload size by ~33% (base64 overhead). For multimodal models that accept audio in chat messages, this is the standard format.

### PCM to WAV Conversion

Both modes need WAV-wrapped audio. The `pcmToWav` function builds a WAV file from raw PCM bytes:

```go
func pcmToWav(pcm []byte, sampleRate int) ([]byte, error) {
    buf := &bytes.Buffer{}

    // WAV header: RIFF chunk, fmt subchunk, data subchunk
    writeString(buf, "RIFF")
    writeUint32(buf, uint32(36+len(pcm)))
    writeString(buf, "WAVE")
    // ... fmt and data subchunks
    buf.Write(pcm)

    return buf.Bytes(), nil
}
```

> **Concept: Binary protocols**
> WAV is a simple container format. The header is 44 bytes of structured binary data. We write it byte-by-byte using `bytes.Buffer`. This is a common pattern in Go for building binary protocols — no external library needed.

## Engine Selection

```go
// internal/speech/speech.go
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
```

The engine is chosen at startup based on config. Once initialized, the rest of the code only interacts with the `STTEngine` interface — it doesn't know or care which backend is active.

## Configuration Examples

**Vosk (offline):**
```json
{
  "stt_engine": "vosk",
  "vosk_model_path": "",
  "vosk_runtime_url": ""
}
```
Model auto-downloads to `user/speech-models/vosk-model-small-en-us-0.15/`. On Windows, runtime DLLs auto-download to `user/speech-runtime/vosk/` when missing. Grammar is built automatically from speech commands.

You can override the Windows runtime source URL:

```json
{
  "stt_engine": "vosk",
  "vosk_runtime_url": "https://example.com/vosk-win64-0.3.45.zip"
}
```

> **Key Pattern: Cross-platform guard (Go + JS mindset)**
> Runtime download logic lives in `internal/speech/vosk_runtime_windows.go` (Windows build tag) with a no-op stub in `internal/speech/vosk_runtime_stub.go` for non-Windows builds. This is similar to frontend feature detection in JavaScript: platform-specific logic is isolated so the rest of the app can call one common function.

**llama-cpp with Whisper:**
```json
{
  "stt_engine": "llama-cpp",
  "llama_cpp_url": "http://localhost:8080",
  "llama_cpp_api_mode": "transcriptions",
  "llama_cpp_model": "whisper"
}
```

**llama-cpp with gemma-3 (multimodal):**
```json
{
  "stt_engine": "llama-cpp",
  "llama_cpp_url": "http://localhost:8080",
  "llama_cpp_api_mode": "chat",
  "llama_cpp_model": "gemma-3-4b",
  "llama_cpp_prompt": "Transcribe the audio to text. Only output the transcription."
}
```

## Key Takeaways

- The `STTEngine` interface enables hot-swappable backends
- Vosk uses grammar-constrained recognition (`NewRecognizerGrm`) for improved accuracy
- Grammar is built from all phrases, aliases, and wake word, with deduplication
- Grammar can be updated at runtime via `SetGrammar()` / `SetGrm()`
- Vosk is fully offline but requires CGO and a ~50MB model
- llama-cpp supports both Whisper-compatible and chat completions APIs
- PCM audio is wrapped in WAV format for API compatibility
- Data URIs embed audio in JSON for multimodal chat models
- Engine selection happens at startup; the rest of the code is backend-agnostic

[← Back: Chapter 9](09-speech-overview.md) · [Next: Chapter 11 →](11-speech-matching.md)
