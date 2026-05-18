//go:build windows && cgo

// Windows virtual keyboard implementation using the SendInput API from user32.dll.
// No driver installation is required — SendInput injects keyboard events directly
// into the Windows input stream. Supports single keys and key combinations
// (e.g., Ctrl+A, Ctrl+Shift+Esc).
package devices

/*
#include <windows.h>

// CGO wrapper for SendInput keyboard events
// Go CGO can't easily construct the INPUT union struct directly

typedef struct {
    WORD wVk;
    WORD wScan;
    DWORD dwFlags;
    DWORD time;
    unsigned long long dwExtraInfo;
} KeyInputGo;

unsigned int wrap_SendInputKey(KeyInputGo* ki) {
    INPUT input = {0};
    input.type = INPUT_KEYBOARD;
    input.ki.wVk = ki->wVk;
    input.ki.wScan = ki->wScan;
    input.ki.dwFlags = ki->dwFlags;
    input.ki.time = ki->time;
    input.ki.dwExtraInfo = (ULONG_PTR)ki->dwExtraInfo;
    return SendInput(1, &input, sizeof(INPUT));
}

unsigned int wrap_SendInputKeys(KeyInputGo* ki, int count) {
    INPUT* inputs = (INPUT*)malloc(sizeof(INPUT) * count);
    for (int i = 0; i < count; i++) {
        inputs[i].type = INPUT_KEYBOARD;
        inputs[i].ki.wVk = ki[i].wVk;
        inputs[i].ki.wScan = ki[i].wScan;
        inputs[i].ki.dwFlags = ki[i].dwFlags;
        inputs[i].ki.time = ki[i].time;
        inputs[i].ki.dwExtraInfo = (ULONG_PTR)ki[i].dwExtraInfo;
    }
    unsigned int result = SendInput(count, inputs, sizeof(INPUT));
    free(inputs);
    return result;
}
*/
import "C"

// Windows virtual key codes.
const (
	VK_LCONTROL            = 0xA2
	VK_RCONTROL            = 0xA3
	VK_LSHIFT              = 0xA0
	VK_RSHIFT              = 0xA1
	VK_LMENU               = 0xA4 // Left Alt
	VK_RMENU               = 0xA5 // Right Alt
	VK_LWIN                = 0x5B
	VK_RWIN                = 0x5C
	VK_BACK                = 0x08
	VK_TAB                 = 0x09
	VK_RETURN              = 0x0D
	VK_ESCAPE              = 0x1B
	VK_SPACE               = 0x20
	VK_CAPITAL             = 0x14
	VK_NUMLOCK             = 0x90
	VK_SCROLL              = 0x91
	VK_F1                  = 0x70
	VK_F2                  = 0x71
	VK_F3                  = 0x72
	VK_F4                  = 0x73
	VK_F5                  = 0x74
	VK_F6                  = 0x75
	VK_F7                  = 0x76
	VK_F8                  = 0x77
	VK_F9                  = 0x78
	VK_F10                 = 0x79
	VK_F11                 = 0x7A
	VK_F12                 = 0x7B
	VK_OEM_1               = 0xBA // ;:
	VK_OEM_PLUS            = 0xBB // =+
	VK_OEM_COMMA           = 0xBC // ,<
	VK_OEM_MINUS           = 0xBD // -_
	VK_OEM_PERIOD          = 0xBE // .>
	VK_OEM_2               = 0xBF // /?
	VK_OEM_3               = 0xC0 // `~
	VK_OEM_4               = 0xDB // [{
	VK_OEM_5               = 0xDC // \|
	VK_OEM_6               = 0xDD // ]}
	VK_OEM_7               = 0xDE // '"
	VK_VOLUME_UP           = 0xAF
	VK_VOLUME_DOWN         = 0xAE
	VK_VOLUME_MUTE         = 0xAD
	VK_MEDIA_PLAY_PAUSE    = 0xB3
	VK_MEDIA_STOP          = 0xB2
	VK_MEDIA_PREV_TRACK    = 0xB1
	VK_MEDIA_NEXT_TRACK    = 0xB0
	VK_SNAPSHOT            = 0x2C
	VK_LAUNCH_APP1         = 0xB6
	VK_LAUNCH_APP2         = 0xB7
	VK_LAUNCH_MEDIA_SELECT = 0xB5
)

// KeyNameToCode maps human-readable key names to Windows virtual key codes.
var KeyNameToCode = map[string]int{
	"a": 0x41, "b": 0x42, "c": 0x43, "d": 0x44, "e": 0x45,
	"f": 0x46, "g": 0x47, "h": 0x48, "i": 0x49, "j": 0x4A,
	"k": 0x4B, "l": 0x4C, "m": 0x4D, "n": 0x4E, "o": 0x4F,
	"p": 0x50, "q": 0x51, "r": 0x52, "s": 0x53, "t": 0x54,
	"u": 0x55, "v": 0x56, "w": 0x57, "x": 0x58, "y": 0x59,
	"z": 0x5A,
	"0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34,
	"5": 0x35, "6": 0x36, "7": 0x37, "8": 0x38, "9": 0x39,
	"minus": VK_OEM_MINUS, "equal": VK_OEM_PLUS, "backspace": VK_BACK,
	"tab": VK_TAB, "enter": VK_RETURN, "space": VK_SPACE,
	"leftbracket": VK_OEM_4, "rightbracket": VK_OEM_6,
	"semicolon": VK_OEM_1, "apostrophe": VK_OEM_7,
	"grave": VK_OEM_3, "backslash": VK_OEM_5,
	"comma": VK_OEM_COMMA, "dot": VK_OEM_PERIOD, "slash": VK_OEM_2,
	"escape": VK_ESCAPE, "capslock": VK_CAPITAL,
	"f1": VK_F1, "f2": VK_F2, "f3": VK_F3, "f4": VK_F4,
	"f5": VK_F5, "f6": VK_F6, "f7": VK_F7, "f8": VK_F8,
	"f9": VK_F9, "f10": VK_F10, "f11": VK_F11, "f12": VK_F12,
	"leftctrl": VK_LCONTROL, "rightctrl": VK_RCONTROL,
	"leftshift": VK_LSHIFT, "rightshift": VK_RSHIFT,
	"leftalt": VK_LMENU, "rightalt": VK_RMENU,
	"leftmeta": VK_LWIN, "rightmeta": VK_RWIN,
	"ctrl": VK_LCONTROL, "shift": VK_LSHIFT,
	"alt": VK_LMENU, "meta": VK_LWIN,
	"numlock": VK_NUMLOCK, "scrolllock": VK_SCROLL,
	"volumeup": VK_VOLUME_UP, "volumedown": VK_VOLUME_DOWN,
	"mute": VK_VOLUME_MUTE, "playpause": VK_MEDIA_PLAY_PAUSE,
	"stop": VK_MEDIA_STOP, "previoussong": VK_MEDIA_PREV_TRACK,
	"nextsong":    VK_MEDIA_NEXT_TRACK,
	"printscreen": VK_SNAPSHOT, "screenshot": VK_SNAPSHOT,
	"launchapp1": VK_LAUNCH_APP1, "launchapp2": VK_LAUNCH_APP2,
	"launchmedia": VK_LAUNCH_MEDIA_SELECT,
}

// windowsKeyboard represents a virtual keyboard using Windows SendInput API.
type windowsKeyboard struct{}

// newKeyboard creates a virtual keyboard device.
// No driver needed - uses built-in Windows SendInput API.
func newKeyboard(index int) (*windowsKeyboard, error) {
	return &windowsKeyboard{}, nil
}

// SendKey sends a single key press (state=1) or release (state=0).
func (k *windowsKeyboard) SendKey(code int, state uint8) {
	var flags C.DWORD
	if state == 0 {
		flags = C.KEYEVENTF_KEYUP
	}

	ki := C.KeyInputGo{
		wVk:     C.WORD(code),
		dwFlags: flags,
	}
	C.wrap_SendInputKey(&ki)
}

// SendCombo sends a key combination.
// When state=1, all keys are pressed simultaneously.
// When state=0, all keys are released simultaneously.
func (k *windowsKeyboard) SendCombo(codes []int, state uint8) {
	if len(codes) == 0 {
		return
	}

	var flags C.DWORD
	if state == 0 {
		flags = C.KEYEVENTF_KEYUP
	}

	ki := make([]C.KeyInputGo, len(codes))
	for i, code := range codes {
		ki[i] = C.KeyInputGo{
			wVk:     C.WORD(code),
			dwFlags: flags,
		}
	}

	C.wrap_SendInputKeys(&ki[0], C.int(len(codes)))
}

// Close is a no-op for SendInput-based keyboard (no device to destroy).
func (k *windowsKeyboard) Close() {}
