//go:build linux

// Linux virtual keyboard implementation using the uinput kernel subsystem.
// Creates a virtual USB keyboard device that appears as real hardware to the OS.
// Each key event is sent as an EV_KEY event followed by EV_SYN/SYN_REPORT.
//
// Key names (e.g., "w", "ctrl", "enter") are mapped to Linux input codes via
// the KeyNameToCode map. This map is used by the WebSocket handler to translate
// client-side key strings into platform-specific codes. The sequence_button block
// relies on this mapping to send WASD keys while holding Ctrl (e.g., ctrl held,
// then d, d, w, s, a tapped, then ctrl released).
package devices

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux input constants for keyboard events.
const (
	KEY_RESERVED       = 0
	KEY_ESC            = 1
	KEY_1              = 2
	KEY_2              = 3
	KEY_3              = 4
	KEY_4              = 5
	KEY_5              = 6
	KEY_6              = 7
	KEY_7              = 8
	KEY_8              = 9
	KEY_9              = 10
	KEY_0              = 11
	KEY_MINUS          = 12
	KEY_EQUAL          = 13
	KEY_BACKSPACE      = 14
	KEY_TAB            = 15
	KEY_Q              = 16
	KEY_W              = 17
	KEY_E              = 18
	KEY_R              = 19
	KEY_T              = 20
	KEY_Y              = 21
	KEY_U              = 22
	KEY_I              = 23
	KEY_O              = 24
	KEY_P              = 25
	KEY_LEFTBRACE      = 26
	KEY_RIGHTBRACE     = 27
	KEY_ENTER          = 28
	KEY_LEFTCTRL       = 29
	KEY_A              = 30
	KEY_S              = 31
	KEY_D              = 32
	KEY_F              = 33
	KEY_G              = 34
	KEY_H              = 35
	KEY_J              = 36
	KEY_K              = 37
	KEY_L              = 38
	KEY_SEMICOLON      = 39
	KEY_APOSTROPHE     = 40
	KEY_GRAVE          = 41
	KEY_LEFTSHIFT      = 42
	KEY_BACKSLASH      = 43
	KEY_Z              = 44
	KEY_X              = 45
	KEY_C              = 46
	KEY_V              = 47
	KEY_B              = 48
	KEY_N              = 49
	KEY_M              = 50
	KEY_COMMA          = 51
	KEY_DOT            = 52
	KEY_SLASH          = 53
	KEY_RIGHTSHIFT     = 54
	KEY_KPASTERISK     = 55
	KEY_LEFTALT        = 56
	KEY_SPACE          = 57
	KEY_CAPSLOCK       = 58
	KEY_F1             = 59
	KEY_F2             = 60
	KEY_F3             = 61
	KEY_F4             = 62
	KEY_F5             = 63
	KEY_F6             = 64
	KEY_F7             = 65
	KEY_F8             = 66
	KEY_F9             = 67
	KEY_F10            = 68
	KEY_NUMLOCK        = 69
	KEY_SCROLLLOCK     = 70
	KEY_F11            = 87
	KEY_F12            = 88
	KEY_RIGHTCTRL      = 97
	KEY_RIGHTALT       = 100
	KEY_LEFTMETA       = 125
	KEY_RIGHTMETA      = 126
	KEY_MUTE           = 113
	KEY_VOLUMEDOWN     = 114
	KEY_VOLUMEUP       = 115
	KEY_PLAYPAUSE      = 164
	KEY_STOPCD         = 166
	KEY_PREVIOUSSONG   = 165
	KEY_NEXTSONG       = 163
	KEY_BRIGHTNESSDOWN = 224
	KEY_BRIGHTNESSUP   = 225
	KEY_PRINT          = 99
	KEY_SYSRQ          = 183

	// Navigation cluster
	KEY_INSERT   = 110
	KEY_DELETE   = 111
	KEY_HOME     = 102
	KEY_END      = 107
	KEY_PAGEUP   = 104
	KEY_PAGEDOWN = 109

	// Arrow keys
	KEY_UP    = 103
	KEY_DOWN  = 108
	KEY_LEFT  = 105
	KEY_RIGHT = 106

	// Numpad
	KEY_KP0      = 82
	KEY_KP1      = 79
	KEY_KP2      = 80
	KEY_KP3      = 81
	KEY_KP4      = 75
	KEY_KP5      = 76
	KEY_KP6      = 77
	KEY_KP7      = 71
	KEY_KP8      = 72
	KEY_KP9      = 73
	KEY_KPDOT    = 83
	KEY_KPENTER  = 96
	KEY_KPPLUS   = 78
	KEY_KPMINUS  = 74
	KEY_KPSLASH  = 98
	// KEY_KPASTERISK = 55 (already defined above)
)

// KeyNameToCode maps human-readable key names to Linux input codes.
// Names are lowercase and used by the WebSocket handler to parse key strings
// from the client. Modifier aliases (ctrl, shift, alt, meta) map to their
// left-hand variants. This map covers all keys registered in allKeys plus
// media keys (volume, play/pause, etc.) and screenshot keys.
var KeyNameToCode = map[string]int{
	"a": KEY_A, "b": KEY_B, "c": KEY_C, "d": KEY_D, "e": KEY_E,
	"f": KEY_F, "g": KEY_G, "h": KEY_H, "i": KEY_I, "j": KEY_J,
	"k": KEY_K, "l": KEY_L, "m": KEY_M, "n": KEY_N, "o": KEY_O,
	"p": KEY_P, "q": KEY_Q, "r": KEY_R, "s": KEY_S, "t": KEY_T,
	"u": KEY_U, "v": KEY_V, "w": KEY_W, "x": KEY_X, "y": KEY_Y,
	"z": KEY_Z,
	"0": KEY_0, "1": KEY_1, "2": KEY_2, "3": KEY_3, "4": KEY_4,
	"5": KEY_5, "6": KEY_6, "7": KEY_7, "8": KEY_8, "9": KEY_9,
	"minus": KEY_MINUS, "equal": KEY_EQUAL, "backspace": KEY_BACKSPACE,
	"tab": KEY_TAB, "enter": KEY_ENTER, "space": KEY_SPACE,
	"leftbracket": KEY_LEFTBRACE, "rightbracket": KEY_RIGHTBRACE,
	"semicolon": KEY_SEMICOLON, "apostrophe": KEY_APOSTROPHE,
	"grave": KEY_GRAVE, "backslash": KEY_BACKSLASH,
	"comma": KEY_COMMA, "dot": KEY_DOT, "slash": KEY_SLASH,
	"escape": KEY_ESC, "esc": KEY_ESC, "capslock": KEY_CAPSLOCK,
	"f1": KEY_F1, "f2": KEY_F2, "f3": KEY_F3, "f4": KEY_F4,
	"f5": KEY_F5, "f6": KEY_F6, "f7": KEY_F7, "f8": KEY_F8,
	"f9": KEY_F9, "f10": KEY_F10, "f11": KEY_F11, "f12": KEY_F12,
	"leftctrl": KEY_LEFTCTRL, "rightctrl": KEY_RIGHTCTRL,
	"leftshift": KEY_LEFTSHIFT, "rightshift": KEY_RIGHTSHIFT,
	"leftalt": KEY_LEFTALT, "rightalt": KEY_RIGHTALT,
	"leftmeta": KEY_LEFTMETA, "rightmeta": KEY_RIGHTMETA,
	"ctrl": KEY_LEFTCTRL, "shift": KEY_LEFTSHIFT,
	"alt": KEY_LEFTALT, "meta": KEY_LEFTMETA,
	"numlock": KEY_NUMLOCK, "scrolllock": KEY_SCROLLLOCK,
	"volumeup": KEY_VOLUMEUP, "volumedown": KEY_VOLUMEDOWN,
	"mute": KEY_MUTE, "playpause": KEY_PLAYPAUSE,
	"stop": KEY_STOPCD, "previoussong": KEY_PREVIOUSSONG,
	"nextsong":     KEY_NEXTSONG,
	"brightnessup": KEY_BRIGHTNESSUP, "brightnessdown": KEY_BRIGHTNESSDOWN,
	"printscreen": KEY_PRINT, "sysrq": KEY_SYSRQ,
	"screenshot": KEY_PRINT,
	// Navigation cluster
	"insert": KEY_INSERT, "ins": KEY_INSERT,
	"delete": KEY_DELETE, "del": KEY_DELETE,
	"home": KEY_HOME, "end": KEY_END,
	"pageup": KEY_PAGEUP, "pgup": KEY_PAGEUP,
	"pagedown": KEY_PAGEDOWN, "pgdn": KEY_PAGEDOWN, "pgdown": KEY_PAGEDOWN,
	// Arrow keys
	"up": KEY_UP, "down": KEY_DOWN, "left": KEY_LEFT, "right": KEY_RIGHT,
	// Numpad
	"numpad0": KEY_KP0, "numpad1": KEY_KP1, "numpad2": KEY_KP2,
	"numpad3": KEY_KP3, "numpad4": KEY_KP4, "numpad5": KEY_KP5,
	"numpad6": KEY_KP6, "numpad7": KEY_KP7, "numpad8": KEY_KP8,
	"numpad9": KEY_KP9,
	"numpaddot": KEY_KPDOT, "numpadenter": KEY_KPENTER,
	"numpadplus": KEY_KPPLUS, "numpadminus": KEY_KPMINUS,
	"numpadmultiply": KEY_KPASTERISK, "numpaddivide": KEY_KPSLASH,
	// Common aliases
	"return": KEY_ENTER,
	"win": KEY_LEFTMETA, "super": KEY_LEFTMETA,
	"windows": KEY_LEFTMETA,
}

// allKeys contains all key codes to register with uinput.
var allKeys = []uint16{
	KEY_ESC, KEY_1, KEY_2, KEY_3, KEY_4, KEY_5, KEY_6, KEY_7, KEY_8, KEY_9, KEY_0,
	KEY_MINUS, KEY_EQUAL, KEY_BACKSPACE, KEY_TAB,
	KEY_Q, KEY_W, KEY_E, KEY_R, KEY_T, KEY_Y, KEY_U, KEY_I, KEY_O, KEY_P,
	KEY_LEFTBRACE, KEY_RIGHTBRACE, KEY_ENTER,
	KEY_LEFTCTRL, KEY_A, KEY_S, KEY_D, KEY_F, KEY_G, KEY_H, KEY_J, KEY_K, KEY_L,
	KEY_SEMICOLON, KEY_APOSTROPHE, KEY_GRAVE,
	KEY_LEFTSHIFT, KEY_BACKSLASH,
	KEY_Z, KEY_X, KEY_C, KEY_V, KEY_B, KEY_N, KEY_M,
	KEY_COMMA, KEY_DOT, KEY_SLASH, KEY_RIGHTSHIFT,
	KEY_LEFTALT, KEY_SPACE, KEY_CAPSLOCK,
	KEY_F1, KEY_F2, KEY_F3, KEY_F4, KEY_F5, KEY_F6, KEY_F7, KEY_F8, KEY_F9, KEY_F10,
	KEY_NUMLOCK, KEY_SCROLLLOCK,
	KEY_F11, KEY_F12,
	KEY_RIGHTCTRL, KEY_RIGHTALT,
	KEY_LEFTMETA, KEY_RIGHTMETA,
	KEY_MUTE, KEY_VOLUMEDOWN, KEY_VOLUMEUP,
	KEY_PLAYPAUSE, KEY_STOPCD, KEY_PREVIOUSSONG, KEY_NEXTSONG,
	KEY_BRIGHTNESSUP, KEY_BRIGHTNESSDOWN,
	KEY_PRINT, KEY_SYSRQ,
	// Navigation cluster
	KEY_INSERT, KEY_DELETE, KEY_HOME, KEY_END, KEY_PAGEUP, KEY_PAGEDOWN,
	// Arrow keys
	KEY_UP, KEY_DOWN, KEY_LEFT, KEY_RIGHT,
	// Numpad
	KEY_KP0, KEY_KP1, KEY_KP2, KEY_KP3, KEY_KP4, KEY_KP5,
	KEY_KP6, KEY_KP7, KEY_KP8, KEY_KP9,
	KEY_KPDOT, KEY_KPENTER, KEY_KPPLUS, KEY_KPMINUS, KEY_KPASTERISK, KEY_KPSLASH,
}

// linuxKeyboard represents a virtual keyboard device created via uinput.
type linuxKeyboard struct {
	fd int
}

// newKeyboard creates a virtual keyboard device.
func newKeyboard(index int) (*linuxKeyboard, error) {
	fd, err := unix.Open(uinputPath, unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", uinputPath, err)
	}

	// Enable key events
	if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_KEY); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_EVBIT EV_KEY: %w", err)
	}

	// Register all keys
	for _, key := range allKeys {
		if err := unix.IoctlSetInt(fd, UI_SET_KEYBIT, int(key)); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("UI_SET_KEYBIT %d: %w", key, err)
		}
	}

	// Set device metadata
	name := fmt.Sprintf("OmniPanel-go-Virtual-Keyboard-%d", index+1)
	var nameBuf [80]byte
	copy(nameBuf[:], name)

	setup := uinputSetup{
		ID: inputID{
			Bus:     0x03, // BUS_USB
			Vendor:  0x1234,
			Product: 0x5679,
			Version: 1,
		},
		Name:      nameBuf,
		FfEffects: 0,
	}
	if err := ioctlSetStruct(fd, UI_DEV_SETUP, unsafe.Pointer(&setup)); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_DEV_SETUP: %w", err)
	}

	if err := unix.IoctlSetInt(fd, UI_DEV_CREATE, 0); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_DEV_CREATE: %w", err)
	}

	return &linuxKeyboard{fd: fd}, nil
}

// SendKey sends a single key press (state=1) or release (state=0).
// Each call writes an EV_KEY event followed by EV_SYN/SYN_REPORT to /dev/uinput.
func (k *linuxKeyboard) SendKey(code int, state uint8) {
	events := []inputEvent{
		{Type: EV_KEY, Code: uint16(code), Value: int32(state)},
		{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
	}
	_, _ = k.writeEvents(events)
}

// SendCombo sends a key combination. All keys are pressed or released in a
// single event batch (one SYN_REPORT), ensuring the OS processes them atomically.
// For state=1, all keys are pressed together. For state=0, all keys are released
// together. This is used for combos like Ctrl+A where both keys must be held
// simultaneously from the OS perspective.
func (k *linuxKeyboard) SendCombo(codes []int, state uint8) {
	if len(codes) == 0 {
		return
	}

	events := make([]inputEvent, 0, len(codes)+1)
	for _, code := range codes {
		events = append(events, inputEvent{
			Type:  EV_KEY,
			Code:  uint16(code),
			Value: int32(state),
		})
	}
	events = append(events, inputEvent{Type: EV_SYN, Code: SYN_REPORT, Value: 0})
	_, _ = k.writeEvents(events)
}

// Close destroys the virtual keyboard device.
func (k *linuxKeyboard) Close() {
	if k.fd >= 0 {
		_ = unix.IoctlSetInt(k.fd, UI_DEV_DESTROY, 0)
		_ = unix.Close(k.fd)
		k.fd = -1
	}
}

// writeEvents serializes inputEvent structs to bytes and writes to /dev/uinput.
func (k *linuxKeyboard) writeEvents(events []inputEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&events[0])), len(events)*24)
	return unix.Write(k.fd, buf)
}
