//go:build windows && cgo

package devices

/*
#include <windows.h>
#include <stdint.h>

// vJoy axis HID usage codes (from public.h)
#define VJOY_AXIS_X     0x30
#define VJOY_AXIS_Y     0x31
#define VJOY_AXIS_Z     0x32
#define VJOY_AXIS_RX    0x33
#define VJOY_AXIS_RY    0x34
#define VJOY_AXIS_RZ    0x35
#define VJOY_AXIS_SL0   0x36
#define VJOY_AXIS_SL1   0x37

// vJoy axis max value
#define VJOY_AXIS_MAX   0x7FFF

// CGO wrapper functions for vJoyInterface.dll calls
// These are needed because Go CGO can't directly call __cdecl DLL functions
// Attempts to load vJoyInterface.dll from multiple paths: current directory, vJoy install dir, and PATH.

HMODULE load_vJoyInterface(void) {
	HMODULE h = NULL;
	// Try: current directory, then vJoy standard install paths
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
		if (h) return h;
	}
	return NULL;
}

int wrap_vJoyEnabled(void) {
	typedef BOOL (__cdecl *vJoyEnabled_t)(void);
	HMODULE h = load_vJoyInterface();
	if (!h) return 0;
	vJoyEnabled_t fn = (vJoyEnabled_t)GetProcAddress(h, "vJoyEnabled");
	if (!fn) { FreeLibrary(h); return 0; }
	BOOL result = fn();
	FreeLibrary(h);
	return result ? 1 : 0;
}

int wrap_AcquireVJD(unsigned int rID) {
	typedef BOOL (__cdecl *AcquireVJD_t)(unsigned int);
	HMODULE h = load_vJoyInterface();
	if (!h) return 0;
	AcquireVJD_t fn = (AcquireVJD_t)GetProcAddress(h, "AcquireVJD");
	if (!fn) { FreeLibrary(h); return 0; }
	BOOL result = fn(rID);
	FreeLibrary(h);
	return result ? 1 : 0;
}

void wrap_RelinquishVJD(unsigned int rID) {
	typedef void (__cdecl *RelinquishVJD_t)(unsigned int);
	HMODULE h = load_vJoyInterface();
	if (!h) return;
	RelinquishVJD_t fn = (RelinquishVJD_t)GetProcAddress(h, "RelinquishVJD");
	if (!fn) { FreeLibrary(h); return; }
	fn(rID);
	FreeLibrary(h);
}

int wrap_SetAxis(long Value, unsigned int rID, unsigned int Axis) {
	typedef BOOL (__cdecl *SetAxis_t)(long, unsigned int, unsigned int);
	HMODULE h = load_vJoyInterface();
	if (!h) return 0;
	SetAxis_t fn = (SetAxis_t)GetProcAddress(h, "SetAxis");
	if (!fn) { FreeLibrary(h); return 0; }
	BOOL result = fn(Value, rID, Axis);
	FreeLibrary(h);
	return result ? 1 : 0;
}

int wrap_SetBtn(int Value, unsigned int rID, unsigned char nBtn) {
	typedef BOOL (__cdecl *SetBtn_t)(int, unsigned int, unsigned char);
	HMODULE h = load_vJoyInterface();
	if (!h) return 0;
	SetBtn_t fn = (SetBtn_t)GetProcAddress(h, "SetBtn");
	if (!fn) { FreeLibrary(h); return 0; }
	BOOL result = fn(Value, rID, nBtn);
	FreeLibrary(h);
	return result ? 1 : 0;
}
*/
import "C"
import (
	"fmt"
	"sync"
)

var vjoyInitOnce sync.Once
var vjoyAvailable bool

// windowsJoystick represents a virtual joystick device created via vJoy.
type windowsJoystick struct {
	rID uint32 // Report ID (1-based)
}

// initVJoy checks if the vJoy driver is installed and available.
// Tries multiple load strategies: PATH, binary directory, and standard vJoy install paths.
func initVJoy() bool {
	vjoyInitOnce.Do(func() {
		vjoyAvailable = C.wrap_vJoyEnabled() == 1
	})
	return vjoyAvailable
}

// newJoystick creates a virtual joystick device via vJoy.
// The index is 0-based internally, but vJoy uses 1-based Report IDs.
func newJoystick(index int) (*windowsJoystick, error) {
	if !initVJoy() {
		return nil, fmt.Errorf("vJoy driver not installed or not available")
	}

	rID := uint32(index + 1) // vJoy uses 1-based IDs

	if C.wrap_AcquireVJD(C.uint(rID)) != 1 {
		return nil, fmt.Errorf("failed to acquire vJoy device %d (already in use or not configured)", rID)
	}

	js := &windowsJoystick{rID: rID}
	js.centerAxes()
	return js, nil
}

// centerAxes sets all axes to the neutral position (center of 0-32767 range).
func (j *windowsJoystick) centerAxes() {
	center := C.long(C.VJOY_AXIS_MAX / 2)
	axes := []C.uint{
		C.VJOY_AXIS_X, C.VJOY_AXIS_Y, C.VJOY_AXIS_Z,
		C.VJOY_AXIS_RX, C.VJOY_AXIS_RY, C.VJOY_AXIS_RZ,
		C.VJOY_AXIS_SL0, C.VJOY_AXIS_SL1,
	}
	for _, axis := range axes {
		C.wrap_SetAxis(center, C.uint(j.rID), axis)
	}
}

// SendButton sends a button press (state=1) or release (state=0).
// vJoy buttons are 1-based, so we add 1 to the ID.
func (j *windowsJoystick) SendButton(id int, state uint8) {
	C.wrap_SetBtn(C.int(state), C.uint(j.rID), C.uchar(id+1))
}

// SendAxis sets an axis to the given value (0-255), scaled to vJoy's 0-32767 range.
func (j *windowsJoystick) SendAxis(id int, value uint8) {
	if id >= len(vjoyAxes) {
		return
	}
	// Scale 0-255 to 0-32767
	scaled := C.long(uint32(value) * C.VJOY_AXIS_MAX / 255)
	C.wrap_SetAxis(scaled, C.uint(j.rID), vjoyAxes[id])
}

// Close releases the vJoy device.
func (j *windowsJoystick) Close() {
	C.wrap_RelinquishVJD(C.uint(j.rID))
}

// vjoyAxes maps OmniPanel-go axis IDs to vJoy HID usage codes.
var vjoyAxes = []C.uint{
	C.VJOY_AXIS_X,   // 0: X
	C.VJOY_AXIS_Y,   // 1: Y
	C.VJOY_AXIS_Z,   // 2: Z
	C.VJOY_AXIS_RX,  // 3: RX
	C.VJOY_AXIS_RY,  // 4: RY
	C.VJOY_AXIS_RZ,  // 5: RZ
	C.VJOY_AXIS_SL0, // 6: Slider 0
	C.VJOY_AXIS_SL1, // 7: Slider 1
}
