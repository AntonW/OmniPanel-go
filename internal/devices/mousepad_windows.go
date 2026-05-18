//go:build windows && cgo

package devices

/*
#include <windows.h>

// CGO wrapper for SendInput mouse events
// Go CGO can't easily construct the INPUT union struct directly

typedef struct {
    long dx;
    long dy;
    unsigned long mouseData;
    unsigned long dwFlags;
    unsigned long time;
    unsigned long long dwExtraInfo;
} MouseInputGo;

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

// windowsMousepad represents a virtual mouse device using Windows SendInput API.
type windowsMousepad struct{}

// newMousepad creates a virtual mouse device.
// No driver needed - uses built-in Windows SendInput API.
func newMousepad(index int) (*windowsMousepad, error) {
	return &windowsMousepad{}, nil
}

// SendMove sends a relative mouse movement event.
// Positive dx = right, positive dy = down.
func (m *windowsMousepad) SendMove(dx, dy int32) {
	mi := C.MouseInputGo{
		dx:      C.long(dx),
		dy:      C.long(dy),
		dwFlags: C.MOUSEEVENTF_MOVE,
	}
	C.wrap_SendInputMouse(&mi)
}

// SendWheel sends a scroll wheel event.
// Positive delta = scroll up, negative = scroll down.
func (m *windowsMousepad) SendWheel(delta int32) {
	mi := C.MouseInputGo{
		mouseData: C.ulong(delta),
		dwFlags:   C.MOUSEEVENTF_WHEEL,
	}
	C.wrap_SendInputMouse(&mi)
}

// SendButton sends a mouse button press (state=1) or release (state=0).
func (m *windowsMousepad) SendButton(btn int, state uint8) {
	var flags C.ulong
	switch btn {
	case MouseBtnLeft:
		if state == 1 {
			flags = C.MOUSEEVENTF_LEFTDOWN
		} else {
			flags = C.MOUSEEVENTF_LEFTUP
		}
	case MouseBtnRight:
		if state == 1 {
			flags = C.MOUSEEVENTF_RIGHTDOWN
		} else {
			flags = C.MOUSEEVENTF_RIGHTUP
		}
	case MouseBtnMiddle:
		if state == 1 {
			flags = C.MOUSEEVENTF_MIDDLEDOWN
		} else {
			flags = C.MOUSEEVENTF_MIDDLEUP
		}
	default:
		return
	}
	mi := C.MouseInputGo{
		dwFlags: flags,
	}
	C.wrap_SendInputMouse(&mi)
}

// Close is a no-op for SendInput-based mouse (no device to destroy).
func (m *windowsMousepad) Close() {}
