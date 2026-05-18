# Chapter 12: Audio Recording — Client & Host

## What This Chapter Covers

OmniPanel-go can record audio from two locations, controlled by `recording_location` in config:
- **`"client"`** (default): The browser records via the Web Audio API, sends raw PCM as binary WebSocket frames
- **`"host"`**: The server records from the PC's microphone using the `malgo` library

Both paths converge at the same STT engine — the only difference is who captures the audio.

## The Microphone Button (Client UI)

A floating mic button is created when the panel loads:

```javascript
function createMicButton() {
    const micBtn = document.createElement('button');
    micBtn.id = 'speech-mic-btn';
    micBtn.className = 'mic-btn';
    micBtn.innerHTML = '&#x1F3A4;';
    micBtn.title = 'Push-to-talk';

    micBtn.addEventListener('pointerdown', (e) => {
        e.preventDefault();
        startRecording();
    });

    micBtn.addEventListener('pointerup', (e) => {
        e.preventDefault();
        stopRecording();
    });

    document.body.appendChild(micBtn);
}
```

> **Concept: `pointerdown`/`pointerup` vs `mousedown`/`mouseup`**
> Pointer events work for both mouse and touch input. Using `pointerdown` instead of `mousedown` means the mic button works on tablets and phones without separate touch event handlers.

The button has CSS animations for different states:

```css
.mic-btn.recording {
    background: rgba(245, 66, 66, 0.8);
    animation: pulse-recording 1s infinite;
}

@keyframes pulse-recording {
    0%, 100% { box-shadow: 0 0 0 0 rgba(245, 66, 66, 0.5); }
    50% { box-shadow: 0 0 0 15px rgba(245, 66, 66, 0); }
}
```

## Client-Side Recording (Web Audio API)

When `recording_location` is `"client"`, the browser captures audio using the Web Audio API's `ScriptProcessorNode`:

```javascript
function startPushToTalkRecording() {
    socket.send(JSON.stringify({
        type: 'start-recording',
        data: { mode: 'push-to-talk' }
    }));

    navigator.mediaDevices.getUserMedia({
        audio: {
            sampleRate: 16000,
            channelCount: 1,
            echoCancellation: true,
            noiseSuppression: true
        }
    }).then(stream => {
        audioStream = stream;
        pcmBuffer = [];

        audioContext = new AudioContext({ sampleRate: 16000 });
        const source = audioContext.createMediaStreamSource(stream);

        scriptProcessor = audioContext.createScriptProcessor(4096, 1, 1);

        scriptProcessor.onaudioprocess = (e) => {
            const input = e.inputBuffer.getChannelData(0);
            const int16 = new Int16Array(input.length);
            for (let i = 0; i < input.length; i++) {
                const s = Math.max(-1, Math.min(1, input[i]));
                int16[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
            }
            pcmBuffer.push(new Uint8Array(int16.buffer));
        };

        source.connect(scriptProcessor);
        scriptProcessor.connect(audioContext.destination);
    });
}
```

> **Concept: Web Audio API vs MediaRecorder**
> The Web Audio API gives us direct access to raw PCM audio samples via `ScriptProcessorNode`. Unlike `MediaRecorder` (which encodes to WebM/Opus), this produces 16-bit PCM directly — the exact format the STT engine expects. No server-side decoding is needed.

When the user releases the button:

```javascript
function stopRecording() {
    if (scriptProcessor) {
        scriptProcessor.disconnect();
        scriptProcessor = null;
    }

    if (audioStream) {
        audioStream.getTracks().forEach(track => track.stop());
        audioStream = null;
    }

    if (audioContext) {
        audioContext.close();
        audioContext = null;
    }

    if (pcmBuffer.length > 0) {
        const totalLen = pcmBuffer.reduce((sum, buf) => sum + buf.length, 0);
        const pcm = new Uint8Array(totalLen);
        let offset = 0;
        for (const buf of pcmBuffer) {
            pcm.set(buf, offset);
            offset += buf.length;
        }
        socket.send(pcm);
    }

    socket.send(JSON.stringify({ type: 'stop-recording' }));
}
```

The accumulated PCM buffer is sent as a **binary WebSocket frame**:

> **Key Pattern: Binary WebSocket frames**
> `socket.send(pcm)` sends raw bytes, not a text message. The server's `ReadMessage()` returns `msgType == ws.BinaryMessage`, which routes to `HandleAudioChunk()` instead of `HandleMessage()`. This avoids the 33% overhead of base64 encoding.

## Host-Side Recording (malgo)

When `recording_location` is `"host"`, the server captures audio from the PC's microphone. The client skips local recording and only sends control signals.

### The HostRecorder

```go
// internal/speech/recorder.go
type HostRecorder struct {
    device    *malgo.Device
    context   *malgo.AllocatedContext
    mu        sync.Mutex
    chunks    [][]byte
    recording bool
}
```

The `HostRecorder` uses [malgo](https://github.com/gen2brain/malgo), a Go binding for miniaudio — a cross-platform audio library that works on Linux (PulseAudio/ALSA), Windows (WASAPI), and macOS (CoreAudio).

> **Concept: Cross-platform audio**
> Audio APIs differ across operating systems. Instead of writing platform-specific code with build tags, malgo abstracts all of this behind a single Go API. This is why `recorder.go` has no `//go:build` directive — it works everywhere.

### Starting Recording

```go
func NewHostRecorder() (*HostRecorder, error) {
    ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
    if err != nil {
        return nil, fmt.Errorf("init malgo context: %w", err)
    }

    hr := &HostRecorder{
        context: ctx,
        chunks:  make([][]byte, 0),
    }

    return hr, nil
}

func (hr *HostRecorder) Start() error {
    hr.mu.Lock()
    defer hr.mu.Unlock()

    if hr.recording {
        return fmt.Errorf("already recording")
    }

    hr.chunks = make([][]byte, 0)

    deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
    deviceConfig.Capture.Format = format
    deviceConfig.Capture.Channels = numChannels
    deviceConfig.SampleRate = sampleRate
    deviceConfig.PeriodSizeInMilliseconds = 100

    sizeInBytes := uint32(malgo.SampleSizeInBytes(format))

    onRecv := func(output, input []byte, framecount uint32) {
        sampleCount := framecount * sizeInBytes
        if len(input) < int(sampleCount) {
            return
        }

        hr.mu.Lock()
        defer hr.mu.Unlock()
        if hr.recording {
            chunk := make([]byte, sampleCount)
            copy(chunk, input[:sampleCount])
            hr.chunks = append(hr.chunks, chunk)
        }
    }

    deviceCallbacks := malgo.DeviceCallbacks{
        Data: onRecv,
    }

    device, err := malgo.InitDevice(hr.context.Context, deviceConfig, deviceCallbacks)
    if err != nil {
        return fmt.Errorf("init capture device: %w", err)
    }

    if err := device.Start(); err != nil {
        device.Uninit()
        return fmt.Errorf("start capture: %w", err)
    }

    hr.device = device
    hr.recording = true

    return nil
}
```

> **Concept: Callback functions**
> `onRecv` is a callback — a function that malgo calls automatically whenever new audio data is available. We don't call it ourselves; malgo calls it on its own internal thread. The callback receives `input` bytes which we copy into our `chunks` slice.

> **Key Pattern: Thread-safe audio buffering**
> The callback runs on malgo's thread, but `Stop()` runs on the WebSocket handler's thread. Both access `hr.chunks`, so we protect it with a `sync.Mutex`. Without the mutex, concurrent reads/writes could corrupt the slice. The mutex is held only for the minimum time needed — `Stop()` and `Close()` release it before calling `device.Stop()` to prevent deadlock with the callback.

### Stopping Recording

```go
func (hr *HostRecorder) Stop() ([]byte, error) {
    hr.mu.Lock()
    if !hr.recording {
        hr.mu.Unlock()
        return nil, fmt.Errorf("not recording")
    }
    hr.recording = false

    device := hr.device
    hr.device = nil

    var pcm []byte
    for _, chunk := range hr.chunks {
        pcm = append(pcm, chunk...)
    }
    hr.chunks = nil
    hr.mu.Unlock()

    if device != nil {
        device.Stop()
        device.Uninit()
    }

    return pcm, nil
}
```

Stopping does three things:
1. Stops and uninitializes the audio device
2. Concatenates all chunks into a single PCM byte slice
3. Returns the PCM data for speech recognition

The returned PCM is 16kHz mono 16-bit — exactly what the STT engines expect.

> **Key Pattern: Avoiding mutex deadlock with audio callbacks**
> The `onRecv` callback runs on malgo's internal thread and acquires `hr.mu.Lock()` to append audio chunks. If `Stop()` held the mutex while calling `device.Stop()`, malgo would wait for the callback to finish — but the callback would be blocked waiting for the mutex. **Deadlock.** The fix: copy the device reference and PCM data while holding the lock, release the lock, *then* call `device.Stop()`. This same pattern applies to `Close()` which frees the malgo context.

## The Routing Decision

The WebSocket handler checks `recording_location` to decide who records:

```go
// internal/websocket/handler.go
func handleStartRecording(s *state.AppState, data json.RawMessage) {
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
            // broadcast error
            return
        }
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
```

The `location` field in the response tells the client whether to record locally or just show the UI state.

## Client-Side Adaptation

The client checks `location` from the server's response:

```javascript
if (msg.type === 'recording-status') {
    if (msg.data.location) {
        recordingLocation = msg.data.location;
    }
    updateMicButtonState(msg.data.state);
}
```

When recording starts:
- **Client mode:** Browser requests microphone permission, starts Web Audio API recording
- **Host mode:** Browser skips local recording, just shows the UI state

When recording stops:
- **Client mode:** Web Audio API stops, raw PCM sent as binary WebSocket frame
- **Host mode:** Browser sends `stop-recording` signal, server processes its own recording

## The Complete Flow

```
Client: hold mic button
    │
    ▼
Client: send start-recording → Server
    │
    ▼
Server: check recording_location
    │
    ├── "client" → broadcast {location: "client"}
    │                 Client: start Web Audio API recording
    │
    └── "host"  → StartHostRecording()
                    Server: open PC microphone
                    broadcast {location: "host"}
                    Client: skip local recording

Client: release mic button
    │
    ▼
Client: send stop-recording → Server
    │
    ├── "client" → wait for binary PCM frame
    │                STT → match → execute
    │
    └── "host"  → StopHostRecording()
                    Server: get PCM from recorder
                    STT → match → execute
                    broadcast speech-result
```

## Wake Word Detection

### Host-Side Wake Word (All Browsers)

When `recording_location` is `"host"` and `trigger_mode` is `"wake-word"`, the server runs a continuous listening loop:

```go
// internal/speech/speech.go
func (sm *SpeechManager) runHostWakeWordLoop() {
    slog.Info("Host wake word detection started", "wake_word", sm.config.WakeWord)

    for {
        select {
        case <-sm.wakeWordDone:
            return
        default:
        }

        if err := sm.listenForWakeWord(); err != nil {
            slog.Error("Host wake word listening error", "error", err)
            time.Sleep(2 * time.Second)
        }
    }
}

func (sm *SpeechManager) listenForWakeWord() error {
    if err := sm.recorder.Start(); err != nil {
        return fmt.Errorf("start recorder: %w", err)
    }

    wakeWord := strings.ToLower(sm.config.WakeWord)
    const maxListen = 30 * time.Second
    listenStart := time.Now()

    for time.Since(listenStart) < maxListen {
        time.Sleep(500 * time.Millisecond)

        // Collect and process audio chunks
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

        if strings.Contains(strings.ToLower(text), wakeWord) {
            // Wake word detected - record follow-up speech
            listenSec := sm.config.WakeWordListenSec
            if listenSec <= 0 {
                listenSec = 8
            }
            time.Sleep(time.Duration(listenSec) * time.Second)

            // Process follow-up speech
            sm.recorder.Stop()
            // ... process and broadcast result
            return nil
        }
    }

    sm.recorder.Stop()
    return nil
}
```

> **Key Pattern: Continuous listening loop**
> The host wake word loop runs in a separate goroutine, continuously recording short audio chunks, feeding them to the STT engine, and checking for the wake word. When detected, it records a configurable number of seconds (`wake_word_listen_sec`) of follow-up speech and processes it as a command.

### Client-Side Wake Word (Chrome/Edge Only)

When `recording_location` is `"client"` and `trigger_mode` is `"wake-word"`, the browser uses the Web Speech API:

```javascript
function startWakeWordMode() {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    speechRecognition = new SpeechRecognition();
    speechRecognition.continuous = true;
    speechRecognition.interimResults = true;
    speechRecognition.lang = 'en-US';

    speechRecognition.onresult = (event) => {
        let transcript = '';
        for (let i = event.resultIndex; i < event.results.length; i++) {
            transcript += event.results[i][0].transcript;
        }

        const wakeWord = speechConfig.wakeWord.toLowerCase();
        if (transcript.toLowerCase().includes(wakeWord)) {
            speechRecognition.stop();
            startPushToTalkRecording();
            setTimeout(() => {
                stopRecording();
                startWakeWordMode();
            }, 3000);
        }
    };

    speechRecognition.start();
}
```

> **Concept: Web Speech API**
> `SpeechRecognition` is a browser API that does real-time speech recognition locally (Chrome) or via Google's servers. It returns interim results as you speak and final results when you pause. We use it only for wake word detection — the actual command recognition still goes through OmniPanel-go's STT engine for consistency and accuracy.

## Handling Speech Results

When the server finishes processing, it broadcasts a `speech-result` message:

```javascript
function handleSpeechResult(data) {
    const text = data.text || '';
    const matched = data.matched || false;
    const speakText = data.speak || '';

    showSpeechToast(`"${text}"${matched ? ' - Matched!' : ''}`, matched);

    if (speakText && speechConfig.ttsEnabled) {
        speakText(speakText);
    }

    updateMicButtonState('idle');
}
```

## Text-to-Speech Feedback

```javascript
function speakText(text) {
    if (!('speechSynthesis' in window)) return;

    const utterance = new SpeechSynthesisUtterance(text);
    utterance.rate = 1.0;
    utterance.pitch = 1.0;
    utterance.volume = 0.8;
    window.speechSynthesis.speak(utterance);
}
```

> **Concept: Speech Synthesis API**
> `window.speechSynthesis` is the browser's built-in text-to-speech engine. It's free, works offline in most browsers, and requires no setup. The `SpeechSynthesisUtterance` object controls voice, rate, pitch, and volume.

## Block Flash Animation

When a speech command triggers a button or slider, the server executes the
action directly via the JoystickManager and broadcasts a message to connected
clients for visual feedback. The client only adds CSS classes for animation —
it does not send any WebSocket messages back to the server:

```javascript
function simulateButtonFromSpeech(blockId) {
    const block = document.getElementById(blockId);
    if (!block) return;

    const btn = block.querySelector('.command-button, [emulate-button]');
    if (btn) {
        btn.classList.add('active');
        setTimeout(() => btn.classList.remove('active'), 200);
    }

    block.classList.add('speech-flash');
    setTimeout(() => block.classList.remove('speech-flash'), 500);
}
```

> **Key Pattern: Server-side execution, client-side visual feedback**
> Button and slider speech commands execute directly on the server via the
> JoystickManager. The `speech-button-trigger` and `speech-slider-trigger`
> WebSocket messages are only for visual feedback (button flash, cyan glow).
> This means speech commands work even when no browser panel is open, and
> there's no risk of double-execution from the client echoing the command back.

## Speech Command Block

The `speech_command.html` block displays recognition status:

```javascript
window.addEventListener('speech-result-' + blockId, (e) => {
    const data = e.detail;
    block.classList.remove('listening', 'matched', 'error');

    if (data.matched) {
        block.classList.add('matched');
        statusEl.textContent = `"${data.text}" - Executed`;
    } else {
        statusEl.textContent = `"${data.text}" - No match`;
    }

    if (displayMode === 'history') {
        history.unshift(data.text);
        historyEl.innerHTML = history.map(t =>
            `<div class="speech-history-item">${t}</div>`
        ).join('');
    }
});
```

> **Concept: Custom Events**
> `new CustomEvent('speech-result-' + blockId, { detail: data })` creates a DOM event that only the specific block listens to. This avoids global state and lets each block update independently.

## Platform Support (Host Recording)

| Platform | Audio Backend | Notes |
|----------|--------------|-------|
| Linux | PulseAudio / ALSA | Usually pre-installed |
| Windows | WASAPI | Built into Windows Vista+ |
| macOS | CoreAudio | Built into macOS |

No additional drivers or libraries are needed — malgo bundles everything.

## Key Takeaways

- `recording_location` config determines who captures audio: client browser or host PC
- Web Audio API `ScriptProcessorNode` captures raw PCM directly in the browser — no encoding/decoding needed
- `malgo` provides cross-platform audio capture with a single Go API
- Binary WebSocket frames avoid base64 encoding overhead for client audio
- Audio is captured as 16kHz mono 16-bit PCM — the format STT engines expect
- The `onRecv` callback runs on malgo's thread, requiring mutex protection
- `Stop()` and `Close()` release the mutex before calling `device.Stop()` to avoid deadlock with the callback
- Server-side recording means the browser never needs microphone permission
- Host wake word detection runs continuously on the server, works on all browsers
- Client wake word detection uses Web Speech API (Chrome/Edge only)
- `wake_word_listen_sec` config controls how long to listen after wake word detection (host mode)
- Speech Synthesis API provides free text-to-speech confirmations
- Custom DOM events let blocks update independently

[← Back: Chapter 11](11-speech-matching.md) · [Next: Chapter 13 →](13-panel-ui.md)
