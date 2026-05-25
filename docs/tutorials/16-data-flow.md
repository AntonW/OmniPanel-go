# Chapter 16: End-to-End Data Flow

## Tracing a Button Press

Let's follow a single button tap from the user's finger to the kernel's input subsystem.

```
User taps button on tablet
         │
         ▼
┌─────────────────────────────────────────────┐
│  client.js: enableInputs()                  │
│  button.addEventListener('pointerdown')     │
│  socket.send({                              │
│    type: 'simulate-button',                 │
│    data: { js: 0, id: 2, state: 1 }         │
│  })                                         │
└──────────────────┬──────────────────────────┘
                   │ WebSocket message
                   ▼
┌─────────────────────────────────────────────┐
│  router.go: handleWS()                      │
│  _, msg, _ := c.ReadMessage()               │
│  websocket.HandleMessage(s, string(msg))    │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: HandleMessage()                │
│  switch msgType {                           │
│  case "simulate-button":                    │
│    handleButton(s, data)                    │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: handleButton()                 │
│  s.JoystickManager.Send(devices.Command{   │
│    Type: ButtonType, Js: 0, Id: 2,          │
│    Value: 1,                                │
│  })                                         │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  devices.go: JoystickManager.Send()         │
│  m.joysticks[0].SendButton(2, 1)            │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  linux.go: linuxJoystick.SendButton()       │
│  events := []inputEvent{                    │
│    {EV_KEY, buttons[2], 1},                 │
│    {EV_SYN, SYN_REPORT, 0},                 │
│  }                                          │
│  unix.Write(fd, events)                     │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  Linux kernel: /dev/uinput                  │
│  Virtual joystick #1, button #3 pressed     │
│  Any app listening to /dev/input/js0        │
│  receives the event                         │
└─────────────────────────────────────────────┘
```

**Key insight:** The entire chain is under 10ms. The browser sends a WebSocket message, Go routes it, and writes to `/dev/uinput`. The kernel delivers it to any application reading the virtual joystick device.

## Tracing a Metric Update

How does CPU usage appear on a client's screen?

```
┌─────────────────────────────────────────────┐
│  databus.go: StartMetrics(500ms ticker)     │
│  Every 500ms:                               │
│    collectCPU() → reads /proc/stat          │
│    Calculates delta-based usage %           │
│    db.data["cpu_usage"] = {75.3, "%"}       │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  state.go: StartDataBroadcast()             │
│  Every 500ms (separate ticker):             │
│    snapshot := s.DataBus.Snapshot()         │
│    s.BroadcastJSON({                        │
│      type: "data-update",                   │
│      data: snapshot,                        │
│    })                                       │
└──────────────────┬──────────────────────────┘
                   │ Broadcast to all clients
                   ▼
┌─────────────────────────────────────────────┐
│  AppState.Broadcast()                       │
│  for ch := range s.clients {                │
│    select {                                 │
│    case ch <- msg:  // non-blocking send    │
│    default:         // drop if full         │
│    }                                        │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  router.go: handleWS() write goroutine      │
│  for msg := range ch {                      │
│    c.WriteMessage(ws.TextMessage, msg)      │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │ WebSocket message
                   ▼
┌─────────────────────────────────────────────┐
│  client.js: socket.onmessage                │
│  if (msg.type === 'data-update') {          │
│    handleDataUpdate(msg.data)               │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  client.js: handleDataUpdate()              │
│  document.querySelectorAll('[data-key]')    │
│  For each block with data-key="cpu_usage":  │
│    valueEl.textContent = "75.3"             │
│    valueEl.classList.add('data-flash')      │
│    setTimeout(remove flash, 300ms)          │
└─────────────────────────────────────────────┘
```

**Key insight:** Two independent 500ms tickers work together. The DataBus ticker collects fresh metrics. The AppState ticker snapshots and broadcasts them. They don't need to be synchronized because the DataBus is thread-safe.

## Tracing a Command Execution

```
User taps command button on panel
         │
         ▼
┌─────────────────────────────────────────────┐
│  client.js: executeCommand()                │
│  socket.send({                              │
│    type: 'execute-command',                 │
│    data: {                                  │
│      block_id: "block_123",                 │
│      command_type: "shell",                 │
│      command: "echo Hello {name}",          │
│      params: { name: "World" }              │
│    }                                        │
│  })                                         │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: handleCommand()                │
│  substituted = SubstituteParams(cmd, params)│
│  → "echo Hello World"                       │
│  success, output = ExecuteShell(substituted)│
│  → true, "Hello World"                      │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: handleCommand() (cont.)        │
│  s.BroadcastJSON({                          │
│    type: "command-result",                  │
│    data: {                                  │
│      block_id: "block_123",                 │
│      success: true,                         │
│      output: "Hello World"                  │
│    }                                        │
│  })                                         │
└──────────────────┬──────────────────────────┘
                   │ Broadcast to ALL clients
                   ▼
┌─────────────────────────────────────────────┐
│  client.js: handleCommandResult()           │
│  btn = document.getElementById("block_123") │
│       .querySelector('.command-button')     │
│  btn.classList.add('success')               │
│  showCommandToast("Hello World", true)      │
│  setTimeout(remove classes, 1500ms)         │
└─────────────────────────────────────────────┘
```

**Key insight:** Command results are broadcast to **all** clients, not just the one that triggered the command. This means if multiple tablets show the same panel, they all see the result feedback.

## Panel Loading (New Flow)

How does a panel get displayed on a client's screen?

```
User opens /panel?name=MyPanel
         │
         ▼
┌─────────────────────────────────────────────┐
│  client.js: DOMContentLoaded                │
│  const panelName = urlParams.get('name')    │
│  fetch(`/api/panel/content?name=MyPanel`)   │
└──────────────────┬──────────────────────────┘
                   │ HTTP GET
                   ▼
┌─────────────────────────────────────────────┐
│  editor.go: getPanelContent()               │
│  path = user/panels/MyPanel.json            │
│  serveFile(c, path, "application/json")     │
└──────────────────┬──────────────────────────┘
                   │ JSON response
                   ▼
┌─────────────────────────────────────────────┐
│  client.js: loadPanel(data)                 │
│  Clears body, renders each block            │
│  enableInputs() attaches event listeners    │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  WebSocket connects to /ws                  │
│  Real-time: button presses, data updates    │
│  No load-panel sent on connect              │
└─────────────────────────────────────────────┘
```

**Key insight:** Each panel tab is independent. The URL query parameter determines which panel to load, making every panel URL bookmarkable. Multiple tabs can display different panels simultaneously.

## Tracing a Speech Command (Host Recording, No Panel Open)

```
User says "ship" then "gear" to PC microphone
           │
           ▼
┌─────────────────────────────────────────────┐
│  speech.go: listenForWakeWord()             │
│  Continuously records audio chunks          │
│  Recognizes "ship" in the audio             │
│  Waits wake_word_listen_sec seconds         │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  speech.go: listenForWakeWord() (cont.)     │
│  Stops recorder, collects follow-up audio   │
│  text, matched, speak, err :=               │
│    sm.Process(postPCM)                      │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  speech.go: SpeechManager.Process()         │
│  text = engine.Recognize(pcm)               │
│  // Vosk returns "gear"                     │
│  matched, speakText = matchAndExecute(text) │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  matcher.go: matchPhrase()                  │
│  phrase = "gear"                            │
│  text = "gear"                              │
│  Match! → executeCommand(cmd, nil)          │
│  → case "button": sm.simulateButton(id)     │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  speech.go: simulateButton()                │
│  cmd = sm.blockIDMap[blockID]               │
│  // button_id=15, joystick_index=3          │
│  // (extracted from panel block settings)   │
│  sm.joystickMgr.Send(ButtonType, js=3,      │
│    id=15, value=1)                          │
│  time.Sleep(200ms)                          │
│  sm.joystickMgr.Send(ButtonType, js=3,      │
│    id=15, value=0)                          │
│  sm.broadcaster.BroadcastJSON(              │
│    speech-button-trigger)                   │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  devices.go: JoystickManager.Send()         │
│  m.joysticks[0].SendButton(9, 1)            │
│  → linux.go: unix.Write(fd, inputEvent)     │
│  → Linux kernel: /dev/uinput                │
│  → Game receives button 9 press             │
└─────────────────────────────────────────────┘
```

**Key insight:** Speech button/slider commands execute entirely on the server
via the JoystickManager. No browser panel needs to be open. The
`speech-button-trigger` broadcast is only for visual feedback when a panel
happens to be connected. Button/axis IDs and joystick indices are extracted
from panel JSON files at startup by parsing each block's settings JSON for
`button`, `axis`, and `joystick` fields, so commands defined in
`speech_commands.json` work immediately with the correct hardware targets.

## Tracing a Speech Command (Client Recording)

```
User holds mic button and says "gear up"
          │
          ▼
┌─────────────────────────────────────────────┐
│  client.js: startRecording()                │
│  navigator.mediaDevices.getUserMedia()      │
│  MediaRecorder starts capturing             │
│  socket.send(JSON.stringify({               │
│    type: 'start-recording',                 │
│    data: { mode: 'push-to-talk' }           │
│  }))                                        │
└──────────────────┬──────────────────────────┘
                   │ WebSocket message
                   ▼
┌─────────────────────────────────────────────┐
│  router.go: handleWS()                      │
│  msgType, msg, _ := c.ReadMessage()         │
│  if msgType == ws.BinaryMessage {           │
│    HandleAudioChunk(s, msg)                 │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: HandleAudioChunk()             │
│  if !speech.ValidatePCM(audioData) {        │
│    pcm = speech.DecodeAudio(audioData)      │
│    // webm/opus → PCM 16kHz mono 16-bit     │
│  }                                          │
│  text, matched, speak, err :=               │
│    s.SpeechManager.Process(pcm)             │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  speech.go: SpeechManager.Process()         │
│  text = engine.Recognize(pcm)               │
│  // Vosk or llama-cpp returns "gear up"     │
│  matched, speakText = matchAndExecute(text) │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  matcher.go: matchPhrase()                  │
│  phrase = "gear up"                         │
│  text = "gear up"                           │
│  Match! → executeCommand(cmd, nil)          │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  speech.go: executeCommand()                │
│  case "shell":                              │
│    success, output = ExecuteShell(cmd)      │
│    → true, "gear_up"                        │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: HandleAudioChunk() (cont.)     │
│  s.BroadcastJSON({                          │
│    type: "speech-result",                   │
│    data: {                                  │
│      text: "gear up",                       │
│      matched: true,                         │
│      speak: "gear up confirmed"             │
│    }                                        │
│  })                                         │
└──────────────────┬──────────────────────────┘
                   │ Broadcast to ALL clients
                   ▼
┌─────────────────────────────────────────────┐
│  client.js: handleSpeechResult()            │
│  showSpeechToast('"gear up" - Matched!')    │
│  if (speakText && ttsEnabled) {             │
│    speechSynthesis.speak(speakText)         │
│  }                                          │
│  updateMicButtonState('idle')               │
└─────────────────────────────────────────────┘
```

**Key insight:** Speech commands reuse the existing command execution infrastructure. The only new pieces are audio capture (browser), audio decoding (pion/opus), speech recognition (Vosk/llama-cpp), and phrase matching. Once a command is matched, it flows through the same `ExecuteShell` or `ExecuteHTTP` functions as regular command blocks.

## Tracing a Speech Command (Host Recording)

```
User holds mic button and says "gear up"
          │
          ▼
┌─────────────────────────────────────────────┐
│  client.js: startRecording()                │
│  socket.send(JSON.stringify({               │
│    type: 'start-recording',                 │
│    data: { mode: 'push-to-talk' }           │
│  }))                                        │
│  // Skip MediaRecorder — host will record   │
└──────────────────┬──────────────────────────┘
                   │ WebSocket message
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: handleStartRecording()         │
│  if recordingLoc == "host" {                │
│    s.SpeechManager.StartHostRecording()     │
│    // Opens PC microphone via malgo         │
│  }                                          │
│  BroadcastJSON({                            │
│    type: "recording-status",                │
│    data: { location: "host", state: "..." } │
│  })                                         │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  recorder.go: HostRecorder.Start()          │
│  deviceConfig = malgo.DefaultDeviceConfig() │
│  deviceConfig.Capture.Format = FormatS16    │
│  deviceConfig.SampleRate = 16000            │
│  deviceConfig.Channels = 1                  │
│  device.Start() → onRecv callback active    │
│  // Audio chunks buffered in hr.chunks      │
└─────────────────────────────────────────────┘

User releases mic button
          │
          ▼
┌─────────────────────────────────────────────┐
│  client.js: stopRecording()                 │
│  socket.send(JSON.stringify({               │
│    type: 'stop-recording'                   │
│  }))                                        │
└──────────────────┬──────────────────────────┘
                   │ WebSocket message
                   ▼
┌─────────────────────────────────────────────┐
│  handler.go: handleStopRecording()          │
│  if recordingLoc == "host" {                │
│    go func() {                              │
│      text, matched, speak, err :=           │
│        s.SpeechManager.StopHostRecording()  │
│      // Stops device, concatenates PCM      │
│      // Processes through STT engine        │
│      s.BroadcastJSON({                      │
│        type: "speech-result",               │
│        data: { text, matched, speak }       │
│      })                                     │
│    }()                                      │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │ Broadcast to ALL clients
                   ▼
┌─────────────────────────────────────────────┐
│  client.js: handleSpeechResult()            │
│  showSpeechToast('"gear up" - Matched!')    │
│  if (speakText && ttsEnabled) {             │
│    speechSynthesis.speak(speakText)         │
│  }                                          │
└─────────────────────────────────────────────┘
```

**Key insight:** Host recording keeps all audio on the server. The client never accesses the microphone — it only sends control signals. This means no browser permission prompts and the PC's microphone is used directly. The `go func()` goroutine processes the audio asynchronously so the WebSocket handler isn't blocked during STT processing.

## The Complete Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        Linux Host Machine                     │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    Go Server (main.go)                   │ │
│  │                                                          │ │
│  │  ┌────────────┐  ┌──────────────┐  ┌─────────────────┐  │ │
│  │  │   Fiber    │  │  WebSocket   │  │    DataBus      │  │ │
│  │  │   Router   │  │   Handler    │  │   (metrics)     │  │ │
│  │  └─────┬──────┘  └──────┬───────┘  └────────┬────────┘  │ │
│  │        │                 │                   │           │ │
│  │  ┌─────┴─────────────────┴───────────────────┴────────┐  │ │
│  │  │                  AppState                           │  │ │
│  │  │  ┌──────────┐  ┌───────────┐  ┌─────────────────┐  │  │ │
│  │  │  │ Config   │  │ Joystick  │  │  Speech         │  │  │ │
│  │  │  │ Manager  │  │ Manager   │  │  Manager        │  │  │ │
│  │  │  └──────────┘  └─────┬─────┘  └────────┬────────┘  │  │ │
│  │  └──────────────────────┼─────────────────┼───────────┘  │ │
│  └─────────────────────────┼─────────────────┼──────────────┘ │
│                            │                 │                │
│                            ▼                 ▼                │
│                    ┌──────────────┐  ┌──────────────┐        │
│                    │ /dev/uinput  │  │ Vosk/llama   │        │
│                    │ (virtual     │  │ (STT engine) │        │
│                    │  joystick)   │  │              │        │
│                    └──────────────┘  └──────────────┘        │
│                            │                 │                │
│                            ▼                 ▼                │
│                    ┌──────────────────────────────┐           │
│                    │   Linux Input Subsystem      │           │
│                    │   /dev/input/js0, js1, ...   │           │
│                    │   /dev/input/mouse0, ...     │           │
│                    └──────────────────────────────┘           │
└──────────────────────────────────────────────────────────────┘
           ▲
           │ HTTP + WebSocket (text + binary)
           │
┌─────────┴─────────┐  ┌──────────────┐  ┌──────────────┐
│   Start Page      │  │   Panel UI   │  │   Editor UI  │
│   /               │  │  /panel?name │  │ /editor      │
│  (host controls,  │  │  (client.js) │  │ (renderer.js)│
│   panel list, log)│  └──────────────┘  └──────────────┘
└───────────────────┘
```

## Key Takeaways

- Every user action follows a clear path: UI → WebSocket/HTTP → Go handler → subsystem → kernel
- Broadcast messages reach all connected clients simultaneously
- The DataBus uses two independent tickers (collect + broadcast) for decoupled operation
- Panel loading uses URL query parameters + REST API instead of WebSocket load-panel
- Each panel tab is independent — multiple panels can be open simultaneously
- Command results are broadcast so all clients see feedback
- Speech commands extend the pipeline: audio capture → decode → recognize → match → execute
- Binary WebSocket frames carry audio data efficiently without base64 encoding
- The entire system is stateless from the client's perspective — the server holds all state

[← Back: Chapter 15](15-host-ui.md) · [Next: Chapter 17 →](17-mpris.md)
