# Chapter 8: Virtual Input Devices

## What This Package Does

This is the most technically complex part of OmniPanel-go. It creates **virtual** joystick and mouse devices using platform-specific APIs:

- **Linux:** Uses the `uinput` kernel subsystem (`/dev/uinput`)
- **Windows:** Uses [vJoy](https://github.com/BrunnerInnovation/vJoy) for joysticks and the `SendInput` API for mouse
- **Other platforms:** No-op stub (app runs but virtual input is unavailable)

When the web panel sends a "button press" or "joystick move", the Go code translates it into OS-level input events that the system treats as coming from a real hardware device.

## Platform-Specific Implementations

The package uses Go build tags to select the correct implementation at compile time:

| File | Build Tag | Purpose |
|------|-----------|---------|
| `linux.go` | `//go:build linux` | Linux joystick via uinput |
| `mousepad_linux.go` | `//go:build linux` | Linux mouse via uinput |
| `keyboard_linux.go` | `//go:build linux` | Linux keyboard via uinput |
| `windows.go` | `//go:build windows` | Windows joystick via vJoy |
| `mousepad_windows.go` | `//go:build windows` | Windows mouse via SendInput |
| `keyboard_windows.go` | `//go:build windows` | Windows keyboard via SendInput |
| `stub.go` | `//go:build !linux && !windows` | No-op for other platforms |

> **Concept: Build tags**
> `//go:build linux` tells the Go compiler to only include this file when building for Linux. This is how OmniPanel-go compiles on any OS — platform-specific code is simply excluded.

## Command Types

```go
// internal/devices/virtual_input.go
type CommandType string

const (
    AxisType   CommandType = "axis"
    ButtonType CommandType = "button"
    MouseType  CommandType = "mouse"
    WheelType  CommandType = "wheel"
)

type Command struct {
    Type  CommandType
    Js    int
    Id    int
    Value uint8
}
```

Commands are the abstract representation of input. The `JoystickManager` receives these and dispatches them to the correct virtual device.

## Platform-Agnostic Interfaces

```go
type Joystick interface {
    SendButton(id int, state uint8)
    SendAxis(id int, value uint8)
    Close()
}

type Mousepad interface {
    SendMove(dx, dy int32)
    SendWheel(delta int32)
    SendButton(btn int, state uint8)
    Close()
}

type Keyboard interface {
    SendKey(code int, state uint8)
    SendCombo(codes []int, state uint8)
    Close()
}
```

> **Concept: Interfaces in Go**
> An interface defines a set of methods. Any type that implements all these methods satisfies the interface implicitly — no `implements` keyword needed. This allows swapping implementations (e.g., Linux vs. Windows) without changing the calling code.

### Mouse Button Constants

To avoid platform-specific button codes leaking into the WebSocket handler, the package defines platform-agnostic constants:

```go
const (
    MouseBtnLeft   = 0
    MouseBtnRight  = 1
    MouseBtnMiddle = 2
)
```

Each platform implementation maps these to its native codes (e.g., `0x110` for Linux `BTN_LEFT`, or `MOUSEEVENTF_LEFTDOWN` for Windows).

## JoystickManager

```go
type JoystickManager struct {
    mu        sync.Mutex
    joysticks []Joystick
}

func (m *JoystickManager) Send(cmd Command) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if cmd.Js >= len(m.joysticks) || m.joysticks[cmd.Js] == nil {
        return
    }
    switch cmd.Type {
    case ButtonType:
        m.joysticks[cmd.Js].SendButton(cmd.Id, cmd.Value)
    case AxisType:
        m.joysticks[cmd.Js].SendAxis(cmd.Id, cmd.Value)
    }
}
```

The manager holds a slice of `Joystick` interfaces (one per virtual joystick). `Send` routes commands to the right device based on the `Js` index.

```go
func (m *JoystickManager) Reload(count uint8) {
    m.mu.Lock()
    defer m.mu.Unlock()

    for _, js := range m.joysticks {
        if js != nil {
            js.Close()
        }
    }
    m.joysticks = make([]Joystick, count)
    for i := range count {
        js, err := newJoystick(int(i))
        if err != nil {
            slog.Warn("Failed to create joystick", "index", i, "error", err)
            continue
        }
        m.joysticks[i] = js
    }
}
```

`Reload` destroys all existing devices and creates new ones. This is called when the user changes the joystick count at runtime.

## Linux Implementation (uinput)

### Creating a Virtual Joystick (linux.go)

```go
//go:build linux

package devices
```

### Constants

```go
const (
    uinputPath = "/dev/uinput"

    EV_ABS = 0x03    // Absolute axis events (joystick axes)
    EV_KEY = 0x01    // Key/button events
    EV_SYN = 0x00    // Synchronization event

    SYN_REPORT = 0   // "Send all pending events now"

    ABS_X        = 0x00
    ABS_Y        = 0x01
    ABS_Z        = 0x02
    ABS_RX       = 0x03
    ABS_RY       = 0x04
    ABS_RZ       = 0x05
    ABS_THROTTLE = 0x06
    ABS_RUDDER   = 0x07

    BTN_JOYSTICK_BASE = 0x100

    UI_DEV_CREATE  = 0x5501
    UI_DEV_DESTROY = 0x5502
    // ... ioctl command codes
)
```

These are Linux kernel constants. `EV_ABS` means "absolute axis" (like a joystick that reports 0–255). `EV_KEY` means "key or button" (pressed or released). `EV_SYN` is a synchronization marker.

### Creating the Device

```go
func newJoystick(index int) (*linuxJoystick, error) {
    fd, err := unix.Open(uinputPath, unix.O_WRONLY|unix.O_NONBLOCK, 0)
    if err != nil {
        return nil, fmt.Errorf("open %s: %w", uinputPath, err)
    }
```

Open `/dev/uinput` for writing. `O_NONBLOCK` means writes won't block if the kernel buffer is full.

```go
    if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_ABS); err != nil {
        unix.Close(fd)
        return nil, fmt.Errorf("UI_SET_EVBIT EV_ABS: %w", err)
    }
    if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_KEY); err != nil {
        unix.Close(fd)
        return nil, fmt.Errorf("UI_SET_EVBIT EV_KEY: %w", err)
    }
```

> **Concept: ioctl**
> `ioctl` (I/O control) is a system call for device-specific operations. `UI_SET_EVBIT` tells uinput "this device will generate absolute events." `UI_SET_EVBIT` with `EV_KEY` says "and button events too."

```go
    for _, axis := range axes {
        if err := unix.IoctlSetInt(fd, UI_SET_ABSBIT, int(axis)); err != nil {
            // ...
        }
        absSetup := uinputAbsSetup{
            Code: axis,
            AbsInfo: inputAbsInfo{
                Value:   128,
                Minimum: 0,
                Maximum: 255,
            },
        }
        if err := ioctlSetStruct(fd, UI_ABS_SETUP, unsafe.Pointer(&absSetup)); err != nil {
            // ...
        }
    }
```

Register each axis (X, Y, Z, RX, RY, RZ, Throttle, Rudder) with a range of 0–255. The initial value of 128 is the center position.

```go
    for _, btn := range buttons {
        if err := unix.IoctlSetInt(fd, UI_SET_KEYBIT, int(btn)); err != nil {
            // ...
        }
    }
```

Register 16 buttons (BTN_JOYSTICK_BASE through BTN_JOYSTICK_BASE+15).

```go
    name := fmt.Sprintf("OmniPanel-go-Virtual-Controller-%d", index+1)
    var nameBuf [80]byte
    copy(nameBuf[:], name)

    setup := uinputSetup{
        Name:      nameBuf,
        FfEffects: 0,
    }
    if err := ioctlSetStruct(fd, UI_DEV_SETUP, unsafe.Pointer(&setup)); err != nil {
        // ...
    }

    if err := unix.IoctlSetInt(fd, UI_DEV_CREATE, 0); err != nil {
        // ...
    }
```

Set the device name and tell the kernel to create it. After `UI_DEV_CREATE`, the virtual device appears in `/dev/input/` and applications can use it.

```go
    js := &linuxJoystick{fd: fd}
    js.centerAxes()
    return js, nil
}
```

Center all axes to 128 (neutral position) so games don't see the joystick as being pushed.

### Sending Events

```go
func (j *linuxJoystick) SendButton(id int, state uint8) {
    if id >= len(buttons) {
        return
    }
    events := []inputEvent{
        {
            Type:  EV_KEY,
            Code:  buttons[id],
            Value: int32(state),
        },
        {
            Type:  EV_SYN,
            Code:  SYN_REPORT,
            Value: 0,
        },
    }
    _, _ = j.writeEvents(events)
}
```

Every event batch ends with `EV_SYN / SYN_REPORT`. This tells the kernel "here's a complete set of changes, process them now." Without it, events might be buffered.

### Writing Events to the Kernel

```go
func (j *linuxJoystick) writeEvents(events []inputEvent) (int, error) {
    if len(events) == 0 {
        return 0, nil
    }
    buf := unsafe.Slice((*byte)(unsafe.Pointer(&events[0])), len(events)*24)
    return unix.Write(j.fd, buf)
}
```

> **Concept: `unsafe.Slice`**
> `unsafe.Slice` creates a byte slice that views the memory of the `inputEvent` structs directly. Each `inputEvent` is 24 bytes (8+8+2+2+4). This avoids copying — the kernel reads the struct data straight from Go memory.

> **Warning: `unsafe`**
> The `unsafe` package bypasses Go's type safety. It's needed here because we're talking directly to the kernel, which expects a specific binary layout. Misuse can cause crashes or memory corruption.

### Linux Virtual Mouse (mousepad_linux.go)

The mousepad follows the same pattern but uses **relative** events instead of absolute:

```go
const (
    EV_REL = 0x02

    REL_X = 0x00
    REL_Y = 0x01
    REL_WHEEL = 0x08

    BTN_LEFT   = 0x110
    BTN_RIGHT  = 0x111
    BTN_MIDDLE = 0x112
)
```

- `EV_REL` = relative movement (the mouse reports "I moved 5 pixels right", not "I'm at position 500")
- `REL_X`, `REL_Y` = movement axes
- `REL_WHEEL` = scroll wheel
- `INPUT_PROP_POINTER` = tells the kernel this is a pointer device (not a touchscreen)

```go
func (m *linuxMousepad) SendMove(dx, dy int32) {
    events := []inputEvent{
        {Type: EV_REL, Code: REL_X, Value: dx},
        {Type: EV_REL, Code: REL_Y, Value: dy},
        {Type: EV_SYN, Code: SYN_REPORT, Value: 0},
    }
    _, _ = m.writeEvents(events)
}
```

## Windows Implementation

### Virtual Joystick via vJoy (windows.go)

Windows has no equivalent to Linux's `/dev/uinput`. OmniPanel-go uses the [vJoy driver](https://github.com/BrunnerInnovation/vJoy) (BrunnerInnovation fork, v2.2.2.0) for virtual joystick simulation.

vJoy is a signed kernel driver that creates up to 16 virtual joystick devices. Each device supports up to 16 axes, 128 buttons, and 4 POV hats.

#### CGO Integration and DLL Loading

The Windows joystick implementation uses CGO to call `vJoyInterface.dll`. To improve reliability across different Windows installations and build environments, the code uses a **multi-path DLL loading strategy** and **caches the loaded DLL handle for the process lifetime**:

```go
//go:build windows && cgo

package devices

/*
#include <windows.h>

// load_vJoyInterface attempts to load vJoyInterface.dll from multiple paths:
// 1. Current directory or PATH (most reliable for bundled/installed builds)
// 2. Standard vJoy x64 install paths
// 3. Standard vJoy bin paths
// Returns the loaded module handle, or NULL if all paths fail.
// The handle is cached to avoid repeated load/unload reinitialization.
HMODULE load_vJoyInterface(void) {
    static HMODULE cached = NULL;
    if (cached) return cached;

    HMODULE h = NULL;
    const char* paths[] = {
        "vJoyInterface.dll",              // Current dir / PATH
        "C:\\Program Files\\vJoy\\x64\\vJoyInterface.dll",
        "C:\\Program Files (x86)\\vJoy\\x64\\vJoyInterface.dll",
        "C:\\Program Files\\vJoy\\bin\\vJoyInterface.dll",
        "C:\\Program Files (x86)\\vJoy\\bin\\vJoyInterface.dll",
        NULL
    };
    for (int i = 0; paths[i] != NULL; i++) {
        h = LoadLibraryA(paths[i]);
        if (h) {
            cached = h;
            return cached;
        }
    }
    return NULL;
}

int wrap_vJoyEnabled(void) {
    typedef BOOL (__cdecl *vJoyEnabled_t)(void);
    HMODULE h = load_vJoyInterface();
    if (!h) return 0;
    vJoyEnabled_t fn = (vJoyEnabled_t)GetProcAddress(h, "vJoyEnabled");
    if (!fn) return 0;
    BOOL result = fn();
    return result ? 1 : 0;
}
*/
import "C"
```

> **Why wrapper functions?**
> Go CGO can't directly call `__cdecl` functions from a dynamically loaded DLL. The C wrapper functions use `LoadLibraryA`/`GetProcAddress` to dynamically load the DLL at runtime, then call the vJoy functions. This avoids requiring the DLL at link time.

> **Concept (Go/C interop): process-lifetime resource caching**
> In `internal/devices/windows.go`, `load_vJoyInterface` stores the first successful `HMODULE` in a static variable and reuses it. This avoids repeated DLL initialization/cleanup cycles, which can trigger startup dialog errors in some `vJoyInterface.dll` builds.

> **Why multi-path loading?**
> vJoy installs to different paths depending on Windows version, user permissions, and upgrade history. By checking multiple paths in order (PATH first for flexibility, then standard install directories), the code gracefully handles various configurations without requiring users to manually add vJoy to PATH or move DLLs around.

> **Key Pattern: Graceful fallback chains**
> The path array is a fallback chain — try the easiest option first, then progressively try more specific locations. This pattern appears throughout systems engineering: DNS resolution, environment variable lookup, and configuration file discovery all use similar strategies to maximize compatibility.

> **Concept (JavaScript): stable transport contract**
> In `static/client/client.js` (`initJoystick`), the browser only sends normalized `simulate-joystick` messages (`0-255` axis range). It does not know about DLL paths or OS APIs. This separation keeps frontend behavior stable while backend internals evolve.

> **Key Pattern (JavaScript): protocol-first design**
> Keep wire messages small and consistent (`type` + `data` payload), and isolate platform-specific details in backend adapters. The same UI code works for Linux `uinput` and Windows vJoy because both consume the same protocol.

#### Axis Mapping

OmniPanel-go's 8 axes (0-255 range) are mapped to vJoy HID usage codes:

| OmniPanel-go Axis | vJoy Usage Code | Description |
|----------------|-----------------|-------------|
| 0 | 0x30 | X |
| 1 | 0x31 | Y |
| 2 | 0x32 | Z |
| 3 | 0x33 | RX |
| 4 | 0x34 | RY |
| 5 | 0x35 | RZ |
| 6 | 0x36 | Slider 0 |
| 7 | 0x37 | Slider 1 |

vJoy axes use a 0-32767 range (`VJOY_AXIS_MAX = 0x7FFF`). OmniPanel-go values are scaled:
```go
scaled := value * 32767 / 255
```

#### Device Acquisition

vJoy uses 1-based Report IDs. OmniPanel-go's 0-based index is converted:
```go
rID := uint32(index + 1)
C.wrap_AcquireVJD(C.uint(rID))
```

If acquisition fails, the device is already in use by another application or not configured in the vJoy settings.

### Virtual Mouse via SendInput (mousepad_windows.go)

Windows mouse simulation uses the built-in `SendInput` API from `user32.dll`. **No driver installation is required.**

```go
//go:build windows

package devices

/*
#include <windows.h>

unsigned int wrap_SendInputMouse(MouseInputGo* mi) {
    INPUT input = {0};
    input.type = INPUT_MOUSE;
    input.mi.dx = mi->dx;
    input.mi.dy = mi->dy;
    input.mi.mouseData = mi->mouseData;
    input.mi.dwFlags = mi->dwFlags;
    input.mi.time = mi->time;
    input.mi.dwExtraInfo = (ULONG_PTR)mi->dwExtraInfo;
    return SendInput(1, &input, sizeof(INPUT));
}
*/
import "C"
```

#### Event Flags

| Action | Windows Flag |
|--------|-------------|
| Move | `MOUSEEVENTF_MOVE` |
| Scroll Up/Down | `MOUSEEVENTF_WHEEL` |
| Left Press | `MOUSEEVENTF_LEFTDOWN` |
| Left Release | `MOUSEEVENTF_LEFTUP` |
| Right Press | `MOUSEEVENTF_RIGHTDOWN` |
| Right Release | `MOUSEEVENTF_RIGHTUP` |
| Middle Press | `MOUSEEVENTF_MIDDLEDOWN` |
| Middle Release | `MOUSEEVENTF_MIDDLEUP` |

#### Mouse Button Mapping

The platform-agnostic button constants are mapped to Windows flags:

```go
func (m *windowsMousepad) SendButton(btn int, state uint8) {
    var flags C.ulong
    switch btn {
    case MouseBtnLeft:
        if state == 1 {
            flags = C.MOUSEEVENTF_LEFTDOWN
        } else {
            flags = C.MOUSEEVENTF_LEFTUP
        }
    // ... right, middle
    }
    mi := C.MouseInputGo{dwFlags: flags}
    C.wrap_SendInputMouse(&mi)
}
```

## Virtual Keyboard

The keyboard implementation creates a virtual keyboard device that injects key presses into the system input stream.

### Key Name Mapping

Both platforms use a `KeyNameToCode` map to translate human-readable key names to platform-specific codes:

```go
// internal/devices/keyboard_linux.go
var KeyNameToCode = map[string]int{
    "a": KEY_A, "b": KEY_B, // ... all letters
    "0": KEY_0, "1": KEY_1, // ... all digits
    "ctrl": KEY_LEFTCTRL, "shift": KEY_LEFTSHIFT,
    "alt": KEY_LEFTALT, "meta": KEY_LEFTMETA,
    "escape": KEY_ESC, "esc": KEY_ESC, "enter": KEY_ENTER, "space": KEY_SPACE,
    // ... function keys, punctuation, etc.
}
```

This map is shared between the WebSocket handler and the platform implementations, enabling a consistent key naming scheme across the application.

### Linux Virtual Keyboard (keyboard_linux.go)

The Linux keyboard uses the same `uinput` subsystem as the joystick and mousepad, but registers only key events (`EV_KEY`):

```go
//go:build linux

package devices

// allKeys contains all key codes to register with uinput.
var allKeys = []uint16{
    KEY_ESC, KEY_1, KEY_2, /* ... all standard keys ... */
    KEY_LEFTMETA, KEY_RIGHTMETA,
}

func newKeyboard(index int) (*linuxKeyboard, error) {
    fd, err := unix.Open(uinputPath, unix.O_WRONLY|unix.O_NONBLOCK, 0)
    // ... enable EV_KEY, register all keys via UI_SET_KEYBIT ...
    // ... set device name "OmniPanel-go-Virtual-Keyboard-N" ...
    // ... create device via UI_DEV_CREATE ...
}
```

> **Key Pattern: Shared uinput infrastructure**
> The keyboard reuses the same `inputEvent` struct, `writeEvents` method, and ioctl helpers as the joystick and mousepad. Each virtual input type follows the same pattern: open `/dev/uinput`, register capabilities, set metadata, create device, write events.

### Windows Virtual Keyboard (keyboard_windows.go)

The Windows keyboard uses `SendInput` with `INPUT_KEYBOARD` — the same API as the mousepad but for keys:

```go
//go:build windows && cgo

/*
unsigned int wrap_SendInputKey(KeyInputGo* ki) {
    INPUT input = {0};
    input.type = INPUT_KEYBOARD;
    input.ki.wVk = ki->wVk;
    input.ki.dwFlags = ki->dwFlags; // 0 = press, KEYEVENTF_KEYUP = release
    return SendInput(1, &input, sizeof(INPUT));
}

unsigned int wrap_SendInputKeys(KeyInputGo* ki, int count) {
    // Sends multiple keys in a single SendInput call for combos
}
*/
```

> **Key Pattern: Batch SendInput for combos**
> `SendCombo` constructs an array of `INPUT` structures and passes them to `SendInput` in a single call. This ensures all keys in a combination (e.g., Ctrl+A) are processed atomically by the OS.

### KeyboardManager

```go
// internal/devices/virtual_input.go
type KeyboardManager struct {
    mu         sync.Mutex
    keyboards  []Keyboard
}

func (m *KeyboardManager) SendKey(index int, code int, state uint8) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if index >= len(m.keyboards) || m.keyboards[index] == nil {
        return
    }
    m.keyboards[index].SendKey(code, state)
}

func (m *KeyboardManager) SendCombo(index int, codes []int, state uint8) {
    // Same pattern, delegates to SendCombo on the keyboard instance
}
```

### WebSocket Keyboard Handler

The WebSocket handler parses key strings like `"ctrl+a"` and looks up each part in `KeyNameToCode`:

```go
// internal/websocket/handler.go
func handleKeyboard(s *state.AppState, data json.RawMessage) {
    kbIndex := int(parseUintField(data, "keyboard_index"))
    key := parseStringField(data, "key") // e.g., "ctrl+a"
    keyState := uint8(parseUintField(data, "state"))

    parts := strings.Split(strings.ToLower(key), "+")
    codes := make([]int, 0, len(parts))
    for _, part := range parts {
        code, ok := devices.KeyNameToCode[part]
        // ...
        codes = append(codes, code)
    }
    if len(codes) == 1 {
        s.KeyboardManager.SendKey(kbIndex, codes[0], keyState)
    } else {
        s.KeyboardManager.SendCombo(kbIndex, codes, keyState)
    }
}
```

> **Concept: String splitting for combos**
> The `+` separator lets users specify combinations in a natural way. `"ctrl+shift+a"` splits into `["ctrl", "shift", "a"]`, each looked up in the key map. Single keys use `SendKey`, multi-key combos use `SendCombo`.

### Sequence Button: CTRL+WASD Pattern

The `sequence_button` block (defined in `user/blocks/sequence_button.html`) sends a sequence of WASD key presses while holding Ctrl. This is useful for games that use Ctrl+directional inputs instead of arrow keys.

The client-side JavaScript (`static/client/client.js`, function `executeSequence`) implements this pattern:

```javascript
// static/client/client.js
function executeSequence(btn) {
    const keys = btn.dataset.sequenceCode.split(','); // e.g., ["d","d","w","s","a"]
    const keyboardIndex = parseInt(btn.dataset.keyboardIndex) || 0;

    // 1. Press and hold Ctrl
    socket.send({ type: 'simulate-keyboard', data: { keyboard_index: keyboardIndex, key: 'ctrl', state: 1 } });

    // 2. Tap each key in sequence (press → 50ms → release → delay → next)
    for (const key of keys) {
        socket.send({ type: 'simulate-keyboard', data: { keyboard_index: keyboardIndex, key: key, state: 1 } });
        setTimeout(() => {
            socket.send({ type: 'simulate-keyboard', data: { keyboard_index: keyboardIndex, key: key, state: 0 } });
        }, 50);
    }

    // 3. Release Ctrl after all keys are done
    socket.send({ type: 'simulate-keyboard', data: { keyboard_index: keyboardIndex, key: 'ctrl', state: 0 } });
}
```

> **Key Pattern: Hold-modifier-tap-sequence**
> Unlike `SendCombo` which presses all keys simultaneously, the sequence button holds a modifier (Ctrl) and taps individual keys one at a time. The server receives separate `simulate-keyboard` events for each key, but because Ctrl's `state=1` comes first and `state=0` comes last, the OS sees Ctrl as held throughout the sequence.

The sequence code uses lowercase key names (`w`, `a`, `s`, `d`, `ctrl`) that match the `KeyNameToCode` map in `internal/devices/keyboard_linux.go`. The WebSocket handler (`internal/websocket/handler.go`, function `handleKeyboard`) parses each key name and dispatches it to the `KeyboardManager`.

## Stub Implementation (stub.go)

For platforms without virtual input support (macOS, BSD, etc.), a stub implementation allows the app to compile and run:

```go
//go:build !linux && !windows

package devices

type stubJoystick struct{}

func newJoystick(index int) (*stubJoystick, error) {
    return nil, fmt.Errorf("virtual joystick not supported on this platform")
}

func (j *stubJoystick) SendButton(id int, state uint8) {}
func (j *stubJoystick) SendAxis(id int, value uint8)  {}
func (j *stubJoystick) Close()                        {}
```

The `JoystickManager` and `MousepadManager` handle nil devices gracefully — commands to non-existent devices are silently ignored.

## MousepadManager

Similar to `JoystickManager`, but manages `Mousepad` interface instances:

```go
func (m *MousepadManager) SendMove(index int, dx, dy int32) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if index >= len(m.mousepads) || m.mousepads[index] == nil {
        return
    }
    m.mousepads[index].SendMove(dx, dy)
}
```

## Key Takeaways

- **Build tags** select platform-specific implementations at compile time
- **Interfaces** (`Joystick`, `Mousepad`, `Keyboard`) enable swapping implementations without changing calling code
- **Platform-agnostic constants** (`MouseBtnLeft`, etc.) and key name maps (`KeyNameToCode`) prevent native codes from leaking into higher layers
- **Linux:** `uinput` kernel subsystem, `ioctl` for configuration, `unsafe.Slice` for zero-copy event writing — used for joystick, mouse, and keyboard
- **Windows:** vJoy driver (CGO) for joysticks, SendInput API (CGO) for mouse and keyboard — no driver needed for mouse/keyboard
- Every event batch ends with `EV_SYN / SYN_REPORT` (Linux) or a single `SendInput` call (Windows)
- Virtual devices must be explicitly destroyed on cleanup (Linux) or released (Windows vJoy)
- Stub implementations allow graceful degradation on unsupported platforms
- Keyboard combos use `+`-separated strings parsed at runtime (e.g., `"ctrl+shift+a"`)

[← Back: Chapter 7](07-commands.md) · [Next: Chapter 9 →](09-speech-overview.md)
