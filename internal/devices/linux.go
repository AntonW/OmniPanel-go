//go:build linux

package devices

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux uinput subsystem constants.
// These mirror the kernel's input.h and uinput.h definitions.
const (
	uinputPath = "/dev/uinput"

	// Event types
	EV_ABS = 0x03 // Absolute axis events (joystick axes: 0-255 range)
	EV_KEY = 0x01 // Key/button events (pressed=1, released=0)
	EV_SYN = 0x00 // Synchronization event (marks end of a batch)

	SYN_REPORT = 0 // "Send all pending events now"

	// Absolute axis codes (8 axes for a virtual joystick)
	ABS_X        = 0x00
	ABS_Y        = 0x01
	ABS_Z        = 0x02
	ABS_RX       = 0x03
	ABS_RY       = 0x04
	ABS_RZ       = 0x05
	ABS_THROTTLE = 0x06
	ABS_RUDDER   = 0x07

	BTN_JOYSTICK_BASE = 0x100 // First joystick button code

	// uinput ioctl commands
	UI_DEV_CREATE  = 0x5501 // Create the virtual device
	UI_DEV_DESTROY = 0x5502 // Destroy the virtual device

	UI_SET_EVBIT  = 0x40045564 // Enable an event type
	UI_SET_KEYBIT = 0x40045565 // Enable a specific key/button
	UI_SET_ABSBIT = 0x40045567 // Enable a specific axis

	UI_DEV_SETUP = 0x405C5503 // Set device metadata (name, etc.)
	UI_ABS_SETUP = 0x401C5504 // Configure axis range (min, max, etc.)
)

// axes defines the 8 absolute axes our virtual joystick supports.
var axes = []uint16{
	ABS_X, ABS_Y, ABS_Z,
	ABS_RX, ABS_RY, ABS_RZ,
	ABS_THROTTLE, ABS_RUDDER,
}

// buttons defines 16 buttons (BTN_JOYSTICK_BASE through BTN_JOYSTICK_BASE+15).
var buttons = func() []uint16 {
	b := make([]uint16, 16)
	for i := range b {
		b[i] = BTN_JOYSTICK_BASE + uint16(i)
	}
	return b
}()

// inputEvent mirrors the kernel's struct input_event.
// Each event has a timestamp, type, code, and value.
type inputEvent struct {
	Sec   int64
	Usec  int64
	Type  uint16
	Code  uint16
	Value int32
}

// inputAbsInfo mirrors the kernel's struct input_absinfo.
// Describes the range and characteristics of an absolute axis.
type inputAbsInfo struct {
	Value      int32
	Minimum    int32
	Maximum    int32
	Fuzz       int32
	Flat       int32
	Resolution int32
}

// uinputAbsSetup is used with UI_ABS_SETUP ioctl to configure axis ranges.
type uinputAbsSetup struct {
	Code    uint16
	_       uint16
	AbsInfo inputAbsInfo
}

// inputID identifies the bus type, vendor, product, and version of the device.
type inputID struct {
	Bus     uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

// uinputSetup is used with UI_DEV_SETUP ioctl to set device metadata.
type uinputSetup struct {
	ID        inputID
	Name      [80]byte
	FfEffects uint32
}

// Compile-time size assertions: ensure our structs match the kernel's layout.
// If sizes don't match, these lines produce a compilation error.
var _ = uint(unsafe.Sizeof(inputEvent{})) - 24
var _ = uint(unsafe.Sizeof(inputAbsInfo{})) - 24
var _ = uint(unsafe.Sizeof(uinputAbsSetup{})) - 28
var _ = uint(unsafe.Sizeof(uinputSetup{})) - 92

// linuxJoystick represents a virtual joystick device created via uinput.
type linuxJoystick struct {
	fd int // File descriptor for /dev/uinput
}

// newJoystick creates a virtual joystick device.
// Steps: open uinput → configure capabilities → set name → create device → center axes.
func newJoystick(index int) (*linuxJoystick, error) {
	fd, err := unix.Open(uinputPath, unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", uinputPath, err)
	}

	// Enable absolute axis and button event types
	if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_ABS); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_EVBIT EV_ABS: %w", err)
	}
	if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_KEY); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_EVBIT EV_KEY: %w", err)
	}

	// Register each axis with range 0-255, center value 128
	for _, axis := range axes {
		if err := unix.IoctlSetInt(fd, UI_SET_ABSBIT, int(axis)); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("UI_SET_ABSBIT %d: %w", axis, err)
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
			unix.Close(fd)
			return nil, fmt.Errorf("UI_ABS_SETUP %d: %w", axis, err)
		}
	}

	// Register all 16 buttons
	for _, btn := range buttons {
		if err := unix.IoctlSetInt(fd, UI_SET_KEYBIT, int(btn)); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("UI_SET_KEYBIT %d: %w", btn, err)
		}
	}

	// Set device name (appears in /dev/input/ and lsinput)
	name := fmt.Sprintf("OmniPanel-go-Virtual-Controller-%d", index+1)
	var nameBuf [80]byte
	copy(nameBuf[:], name)

	setup := uinputSetup{
		Name:      nameBuf,
		FfEffects: 0,
	}
	if err := ioctlSetStruct(fd, UI_DEV_SETUP, unsafe.Pointer(&setup)); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_DEV_SETUP: %w", err)
	}

	// Tell the kernel to create the device
	if err := unix.IoctlSetInt(fd, UI_DEV_CREATE, 0); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_DEV_CREATE: %w", err)
	}

	js := &linuxJoystick{fd: fd}
	js.centerAxes()
	return js, nil
}

// centerAxes sets all axes to the neutral position (128).
// Prevents games from seeing the joystick as being pushed on startup.
func (j *linuxJoystick) centerAxes() {
	events := make([]inputEvent, 0, 9)
	for _, axis := range axes {
		events = append(events, inputEvent{
			Type:  EV_ABS,
			Code:  axis,
			Value: 128,
		})
	}
	events = append(events, inputEvent{
		Type:  EV_SYN,
		Code:  SYN_REPORT,
		Value: 0,
	})
	_, _ = j.writeEvents(events)
}

// SendButton sends a button press (state=1) or release (state=0).
// Every event batch ends with EV_SYN/SYN_REPORT to flush to the kernel.
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

// SendAxis sets an axis to the given value (0-255).
func (j *linuxJoystick) SendAxis(id int, value uint8) {
	if id >= len(axes) {
		return
	}
	events := []inputEvent{
		{
			Type:  EV_ABS,
			Code:  axes[id],
			Value: int32(value),
		},
		{
			Type:  EV_SYN,
			Code:  SYN_REPORT,
			Value: 0,
		},
	}
	_, _ = j.writeEvents(events)
}

// Close destroys the virtual device and closes the file descriptor.
func (j *linuxJoystick) Close() {
	if j.fd >= 0 {
		_ = unix.IoctlSetInt(j.fd, UI_DEV_DESTROY, 0)
		_ = unix.Close(j.fd)
		j.fd = -1
	}
}

// writeEvents serializes inputEvent structs to bytes and writes them to /dev/uinput.
// Uses unsafe.Slice for zero-copy conversion from struct slice to byte slice.
func (j *linuxJoystick) writeEvents(events []inputEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&events[0])), len(events)*24)
	return unix.Write(j.fd, buf)
}

// ioctlSetStruct performs an ioctl call with a struct pointer argument.
func ioctlSetStruct(fd int, req uintptr, ptr unsafe.Pointer) error {
	_, _, errno := unix.Syscall6(unix.SYS_IOCTL, uintptr(fd), req, uintptr(ptr), 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("ioctl(%#x): %w", req, errno)
	}
	return nil
}
