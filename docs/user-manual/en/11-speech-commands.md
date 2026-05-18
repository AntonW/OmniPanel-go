# Chapter 11: Speech Commands

Control your panel with your voice. This chapter covers how to set up and use speech recognition to trigger commands, press buttons, and adjust sliders just by speaking.

## What Are Speech Commands?

Speech Commands let you control your panel by speaking instead of tapping. You can:

- Run shell commands by saying a phrase
- Send HTTP requests with your voice
- Press buttons on your panel by speaking
- Adjust sliders to specific values by saying a number

## Two Recording Locations

OmniPanel-go can record audio from two different places. This is controlled by the `recording_location` setting in `config.json`.

### Client Recording (Default)

Your tablet or phone's microphone records the audio. The audio is sent to your PC over the network for processing.

**How it works:**
1. You hold the microphone button on your panel
2. Your tablet's browser asks for microphone permission (first time only)
3. Your tablet records your voice
4. The audio is sent to your PC via the network
5. Your PC processes it and executes the command

**Best for:**
- Push-to-talk mode
- When your tablet is closer to you than your PC
- When your PC doesn't have a microphone

**Requirements:**
- Browser microphone permission (you'll be asked on first use)
- Network connection between tablet and PC

### Host Recording (Server-Side)

Your gaming PC's microphone records the audio directly. Your tablet just sends start/stop signals — no audio leaves your tablet.

**How it works:**
1. You hold the microphone button on your panel
2. Your tablet sends a "start recording" signal to your PC
3. Your PC starts recording from its own microphone
4. You speak your command
5. You release the button — your tablet sends a "stop recording" signal
6. Your PC processes the audio it just recorded and executes the command

**Best for:**
- Wake word mode (PC is always listening)
- When your PC has a good microphone (headset, webcam, etc.)
- When you don't want to grant microphone permission to your browser
- When your tablet is far from you but your PC microphone is close

**Requirements:**
- A working microphone connected to your PC
- On Linux: PulseAudio or ALSA (usually pre-installed)
- On Windows: Any microphone that works with Windows

> **Tip:** Host recording means your tablet never needs microphone permission. The browser just sends tiny control signals. All audio processing happens on your PC.

---

## Two Activation Modes

### Push-to-Talk

Hold a microphone button on your panel, speak your command, then release the button. This is the default mode and works on all browsers.

**How it works:**
1. A floating microphone button appears on your panel
2. Hold the button down
3. Speak your command
4. Release the button
5. OmniPanel-go processes what you said and executes the matching command

### Wake Word

OmniPanel-go continuously listens for a specific word (the "wake word"). When you say it, the system starts listening for your command.

**How it works:**
1. OmniPanel-go is always listening (but not recording)
2. Say the wake word (default: "omnipanel-go")
3. After the wake word, speak your command
4. OmniPanel-go processes and executes the matching command

> **Note:** Wake word mode only works in Chrome and Edge browsers. Safari and Firefox don't support it.

---

## Two Speech Recognition Engines

### Vosk (Offline)

Vosk runs entirely on your PC — no internet connection required. It uses **grammar-constrained recognition**, which means it only tries to recognize the phrases you've defined as speech commands. This dramatically improves accuracy compared to trying to recognize any English word.

**How grammar-constrained recognition works:**
- OmniPanel-go collects all your speech command phrases and aliases
- It builds a "grammar" — a list of allowed phrases — and gives it to Vosk
- Vosk only tries to match your defined phrases, ignoring everything else
- When you add new speech triggers via the editor, the grammar updates automatically

**Pros:**
- Works without internet
- Fast response time
- Private (audio never leaves your PC)
- **High accuracy** for defined commands (grammar-constrained mode)

**Cons:**
- Requires downloading a model (~50MB)
- Only recognizes phrases you've defined (won't understand free-form speech)
- Limited language support

**Setup:**
1. In `config.json`, set `"stt_engine": "vosk"`
2. Leave `vosk_model_path` empty to auto-download the model
3. Or set `vosk_model_path` to a model you've already downloaded

### llama-cpp-server (AI Server)

Uses an existing llama-cpp-server instance for speech recognition. This supports Whisper-compatible models or AI chat modes.

**Pros:**
- More accurate than Vosk for free-form speech
- Supports many languages
- Can understand natural language commands

**Cons:**
- Requires running a separate server
- Needs more powerful hardware
- Slightly slower response time

**Setup:**
1. In `config.json`, set `"stt_engine": "llama-cpp"`
2. Set `llama_cpp_url` to your server address (default: `http://localhost:8080`)
3. Set `llama_cpp_api_mode` to `"transcriptions"` (Whisper-style) or `"chat"` (AI chat)
4. If using chat mode, set `llama_cpp_model` and `llama_cpp_prompt`

---

## Configuring Speech

### Step 1: Enable Speech in config.json

Open `config.json` and find the `speech` section:

```json
{
  "speech": {
    "enabled": true,
    "recording_location": "client",
    "trigger_mode": "push-to-talk",
    "wake_word": "omnipanel-go",
    "wake_word_listen_sec": 8,
    "stt_engine": "vosk",
    "tts_enabled": true,
    "speech_allowlist": []
  }
}
```

| Setting | Options | Description |
|---------|---------|-------------|
| `enabled` | `true` or `false` | Turn speech recognition on or off |
| `recording_location` | `"client"` or `"host"` | Where to record audio: `"client"` uses your tablet's microphone, `"host"` uses your PC's microphone |
| `trigger_mode` | `"push-to-talk"` or `"wake-word"` | How to activate speech recognition |
| `wake_word` | Any word | The word that activates wake word mode (default: "omnipanel-go") |
| `wake_word_listen_sec` | Number | Seconds to listen for speech after wake word detection (host mode only, default: 8) |
| `stt_engine` | `"vosk"` or `"llama-cpp"` | Which speech recognition engine to use |
| `vosk_model_path` | File path | Path to Vosk model (leave empty to auto-download) |
| `llama_cpp_url` | URL | Address of your llama-cpp-server |
| `llama_cpp_api_key` | Text | API key for llama-cpp-server (if required) |
| `llama_cpp_api_mode` | `"transcriptions"` or `"chat"` | API mode for llama-cpp-server |
| `llama_cpp_model` | Text | Model name for chat mode (e.g., "gemma-3-4b") |
| `llama_cpp_prompt` | Text | Instructions for the AI in chat mode |
| `tts_enabled` | `true` or `false` | Enable voice confirmations (text-to-speech) |
| `speech_allowlist` | List of patterns | Allowed voice commands (empty = all allowed) |

### Step 2: Define Speech Commands

You can define speech commands in two ways:

**Method 1: In the Editor (per block)**

1. Open the Editor
2. Click the gear icon on any block
3. Find the **Speech Trigger** settings
4. Set:
   - **Speech Trigger** — the phrase that activates this block (e.g., "gear up")
   - **Aliases** — alternative phrases (e.g., "landing gear up", "gear up please")
   - **Trigger Type** — what the command does (button press, slider change, shell command, HTTP request)

**Method 2: In speech_commands.json**

Edit the `user/speech_commands.json` file:

```json
[
  {
    "phrase": "gear up",
    "aliases": ["landing gear up", "gear up please"],
    "type": "button",
    "joystick": 0,
    "button": 3
  },
  {
    "phrase": "set throttle to *",
    "aliases": ["throttle to *"],
    "type": "slider",
    "joystick": 0,
    "slider": 0
  },
  {
    "phrase": "launch notepad",
    "aliases": ["open notepad"],
    "type": "shell",
    "command": "notepad"
  }
]
```

---

## Command Types

### Button Press

Triggers a button block on your panel. Button-type speech commands execute
directly on the server — you don't need a browser panel open. The button ID
is automatically extracted from your panel files at startup.

**Settings:**
- `type`: `"button"`
- `joystick`: Which virtual controller (0-9)
- `button`: Which button (0-15)

**Example:**
```json
{
  "phrase": "gear up",
  "type": "button",
  "joystick": 0,
  "button": 3
}
```

### Slider Change

Sets a slider to a spoken value. Slider commands also execute directly on the
server — no browser panel needed.

**Settings:**
- `type`: `"slider"`
- `joystick`: Which virtual controller (0-9)
- `slider`: Which axis (0-7)

**Example:**
```json
{
  "phrase": "set throttle to *",
  "type": "slider",
  "joystick": 0,
  "slider": 0
}
```

Say "set throttle to 75" and the slider moves to 75 (out of 255).

### Shell Command

Runs a command on your PC.

**Settings:**
- `type`: `"shell"`
- `command`: The command to run

**Example:**
```json
{
  "phrase": "launch notepad",
  "type": "shell",
  "command": "notepad"
}
```

### HTTP Request

Sends a web request.

**Settings:**
- `type`: `"http"`
- `method`: GET, POST, PUT, DELETE, or PATCH
- `url`: The URL to send the request to
- `body`: Optional request body

**Example:**
```json
{
  "phrase": "turn on living room light",
  "type": "http",
  "method": "POST",
  "url": "http://192.168.1.100/api/lights/1",
  "body": "{\"on\": true}"
}
```

---

## Parameter Extraction

Speech Commands can extract values from your spoken phrases using wildcards (`*`).

### How It Works

Define a phrase with `*` where the value should be:

```json
{
  "phrase": "set throttle to *",
  "type": "slider",
  "joystick": 0,
  "slider": 0
}
```

When you say "set throttle to 75", OmniPanel-go extracts `75` and sets the slider to that value.

### Multiple Parameters

You can use multiple wildcards:

```json
{
  "phrase": "set * to *",
  "type": "slider",
  "joystick": 0,
  "slider": 0
}
```

This matches phrases like "set throttle to 75" or "set volume to 50".

---

## Aliases

Aliases let you trigger the same command with different phrases.

```json
{
  "phrase": "gear up",
  "aliases": ["landing gear up", "gear up please", "deploy gear"],
  "type": "button",
  "joystick": 0,
  "button": 3
}
```

All of these phrases will trigger the same button press:
- "gear up"
- "landing gear up"
- "gear up please"
- "deploy gear"

---

## Fuzzy Matching

OmniPanel-go uses fuzzy matching to understand you even if speech recognition isn't perfect. It compares what it heard to your defined commands with an **80% similarity threshold**.

**Example:**
- You defined: "gear up"
- You said: "gear upp" (slightly misrecognized)
- Result: Still matches because it's close enough

---

## Security Allowlist

The speech allowlist restricts which phrases are allowed to execute commands. This prevents accidental or malicious voice commands.

### How It Works

In `config.json`, set `speech_allowlist` to a list of allowed patterns:

```json
{
  "speech_allowlist": [
    "gear up",
    "gear down",
    "set * to *",
    "launch *"
  ]
}
```

- **Exact match:** `"gear up"` — only "gear up" is allowed
- **Wildcard:** `"set * to *"` — any phrase matching this pattern is allowed
- **Empty list:** All commands are allowed (no restrictions)

If a spoken command doesn't match any pattern in the allowlist, it's ignored.

---

## Text-to-Speech Confirmations

When `tts_enabled` is `true`, OmniPanel-go speaks a confirmation after executing a command.

**Example:**
- You say: "gear up"
- OmniPanel-go executes the command
- OmniPanel-go says: "Gear up confirmed"

This gives you audio feedback that your command was understood and executed.

---

## Visual Feedback

### Microphone Button

The floating microphone button shows different states:

- **Normal** — ready to record
- **Pulsing red** — recording your voice
- **Pulsing yellow** — processing what you said

### Toast Notification

A message appears showing what OmniPanel-go heard:

- Shows the recognized text
- Shows which command matched
- Auto-dismisses after a few seconds

### Matched Block Glow

When a speech command matches a block on your panel, that block **glows cyan briefly** so you can see which control was activated.

---

## Practical Examples

### Example 1: Voice-Controlled Flight Panel

**Commands:**
```json
[
  {"phrase": "gear up", "aliases": ["landing gear up"], "type": "button", "joystick": 0, "button": 0},
  {"phrase": "gear down", "aliases": ["landing gear down"], "type": "button", "joystick": 0, "button": 1},
  {"phrase": "set throttle to *", "aliases": ["throttle to *"], "type": "slider", "joystick": 0, "slider": 0},
  {"phrase": "flaps up", "type": "button", "joystick": 0, "button": 2},
  {"phrase": "flaps down", "type": "button", "joystick": 0, "button": 3}
]
```

### Example 2: Voice-Controlled Smart Home

**Commands:**
```json
[
  {"phrase": "turn on living room light", "aliases": ["lights on"], "type": "http", "method": "POST", "url": "http://192.168.1.100/api/lights/1", "body": "{\"on\": true}"},
  {"phrase": "turn off living room light", "aliases": ["lights off"], "type": "http", "method": "POST", "url": "http://192.168.1.100/api/lights/1", "body": "{\"on\": false}"},
  {"phrase": "set temperature to *", "type": "http", "method": "POST", "url": "http://192.168.1.100/api/thermostat", "body": "{\"temp\": *}"},
  {"phrase": "launch spotify", "type": "shell", "command": "spotify"}
]
```

---

## Tips for Speech Commands

### Server-Side Execution

Button and slider speech commands execute directly on your PC via the virtual
joystick system. This means:
- **No browser panel needed** — commands work even if no panel is open
- **No double-triggering** — the server handles execution, the browser only shows a visual flash
- **Automatic ID mapping** — button/slider IDs are read from your panel files at startup

### Speaking Clearly

- Speak at a normal pace — don't rush
- Enunciate clearly, especially for important commands
- Reduce background noise for better recognition

## Trigger Modes

These modes are mutually exclusive — set one in `config.json`.

### Push-to-Talk (`"push-to-talk"`)

Hold the floating microphone button on the panel (or a dedicated Push-to-Talk block), speak your command, and release to process.

- Visual feedback: red pulse while recording, yellow while processing
- Works with both client and host recording

### Wake Word (`"wake-word"`)

Continuous hands-free listening. The behavior depends on `recording_location`:

**Host mode** (`"recording_location": "host"`):
- The server continuously listens for the wake word using your PC's microphone
- When the wake word is detected, the server records `wake_word_listen_sec` seconds of follow-up speech and processes it
- Works on all browsers — no browser-specific requirements
- The floating mic button is hidden since the server handles everything

**Client mode** (`"recording_location": "client"`):
- The browser listens for the wake word using the Web Speech API (Chrome/Edge only)
- When detected, the tablet microphone records follow-up speech
- Visual feedback: cyan pulse while listening, faster pulse when wake word detected

### Choosing a Wake Word

- Pick a word that's easy to say and unlikely to come up in conversation
- Avoid common words like "hey" or "ok"
- Two-syllable words work best (e.g., "omnipanel-go," "computer")

### Testing Speech Recognition

1. Start with simple, unique phrases
2. Test each command individually
3. Check the toast notification to see what OmniPanel-go heard
4. Adjust your phrases or aliases if recognition is poor

### Browser Permissions

**Client recording** requires microphone permission from your browser. When you first use speech recognition, your browser will ask for microphone permission. Click **Allow**.

If you denied permission by accident:
- **Chrome/Edge:** Click the lock icon in the address bar, then enable microphone
- **Firefox:** Click the camera/microphone icon in the address bar, then enable

> **Note:** Host recording does NOT require browser microphone permission. Your PC's microphone is used directly, so the browser never needs access to any microphone.

### Recording Location

- **Client recording** (`"recording_location": "client"`) — uses your tablet's microphone. The browser records audio and sends it to your PC over the network. Recommended for push-to-talk when your tablet is close to you.
- **Host recording** (`"recording_location": "host"`) — uses your PC's microphone. Your PC records audio directly from its own microphone. Your tablet only sends start/stop signals. Recommended for wake word mode or when your PC has a better microphone.

**How to switch:**
1. Open `config.json` on your PC
2. Change `"recording_location"` to `"client"` or `"host"`
3. Restart OmniPanel-go

## What's Next?

Understand how games see your panel inputs in [Virtual Joysticks](12-virtual-joysticks.md).
