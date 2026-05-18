//go:build linux

package devices

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux input constants for relative (mouse) events.
// Unlike absolute axes (0-255), relative events report deltas.
const (
	EV_REL = 0x02 // Relative movement events

	REL_X     = 0x00 // X-axis movement
	REL_Y     = 0x01 // Y-axis movement
	REL_WHEEL = 0x08 // Scroll wheel

	BTN_LEFT   = 0x110
	BTN_RIGHT  = 0x111
	BTN_MIDDLE = 0x112

	UI_SET_RELBIT      = 0x40045566 // Enable a relative axis
	UI_SET_PROPBIT     = 0x40045569 // Set device property
	INPUT_PROP_POINTER = 0x00       // Mark device as a pointer (not touchscreen)
)

// linuxMousepad represents a virtual mouse device created via uinput.
type linuxMousepad struct {
	fd int
}

// newMousepad creates a virtual mouse device.
// Configured for relative movement (X, Y), scroll wheel, and 3 buttons.
func newMousepad(index int) (*linuxMousepad, error) {
	fd, err := unix.Open(uinputPath, unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", uinputPath, err)
	}

	// Enable relative movement and button events
	if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_REL); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_EVBIT EV_REL: %w", err)
	}
	if err := unix.IoctlSetInt(fd, UI_SET_EVBIT, EV_KEY); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_EVBIT EV_KEY: %w", err)
	}

	// Mark as a pointer device (not a touchscreen)
	if err := unix.IoctlSetInt(fd, UI_SET_PROPBIT, INPUT_PROP_POINTER); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_PROPBIT INPUT_PROP_POINTER: %w", err)
	}

	// Enable X, Y, and wheel axes
	if err := unix.IoctlSetInt(fd, UI_SET_RELBIT, REL_X); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_RELBIT REL_X: %w", err)
	}
	if err := unix.IoctlSetInt(fd, UI_SET_RELBIT, REL_Y); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_RELBIT REL_Y: %w", err)
	}
	if err := unix.IoctlSetInt(fd, UI_SET_RELBIT, REL_WHEEL); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("UI_SET_RELBIT REL_WHEEL: %w", err)
	}

	// Register left, right, and middle mouse buttons
	mouseButtons := []uint16{BTN_LEFT, BTN_RIGHT, BTN_MIDDLE}
	for _, btn := range mouseButtons {
		if err := unix.IoctlSetInt(fd, UI_SET_KEYBIT, int(btn)); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("UI_SET_KEYBIT %d: %w", btn, err)
		}
	}

	// Set device metadata (USB bus type for compatibility)
	name := fmt.Sprintf("OmniPanel-go-Virtual-Mouse-%d", index+1)
	var nameBuf [80]byte
	copy(nameBuf[:], name)

	setup := uinputSetup{
		ID: inputID{
			Bus:     0x03, // BUS_USB
			Vendor:  0x1234,
			Product: 0x5678,
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

	return &linuxMousepad{fd: fd}, nil
}

// SendMove sends a relative mouse movement event.
// Positive dx = right, positive dy = down.
func (m *linuxMousepad) SendMove(dx, dy int32) {
	events := []inputEvent{
		{Type: EV_REL, Code: REL_X, Value: dx},
		{Type: EV_REL, Code: REL_Y, Value: dy},
		{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
	}
	_, _ = m.writeEvents(events)
}

// SendWheel sends a scroll wheel event.
// Positive delta = scroll up, negative = scroll down.
func (m *linuxMousepad) SendWheel(delta int32) {
	events := []inputEvent{
		{Type: EV_REL, Code: REL_WHEEL, Value: delta},
		{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
	}
	_, _ = m.writeEvents(events)
}

// SendButton sends a mouse button press (state=1) or release (state=0).
func (m *linuxMousepad) SendButton(btn int, state uint8) {
	var code uint16
	switch btn {
	case MouseBtnLeft:
		code = BTN_LEFT
	case MouseBtnRight:
		code = BTN_RIGHT
	case MouseBtnMiddle:
		code = BTN_MIDDLE
	default:
		code = BTN_LEFT
	}
	events := []inputEvent{
		{Type: EV_KEY, Code: code, Value: int32(state)},
		{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
	}
	_, _ = m.writeEvents(events)
}

// Close destroys the virtual mouse device.
func (m *linuxMousepad) Close() {
	if m.fd >= 0 {
		_ = unix.IoctlSetInt(m.fd, UI_DEV_DESTROY, 0)
		_ = unix.Close(m.fd)
		m.fd = -1
	}
}

// writeEvents serializes inputEvent structs to bytes and writes to /dev/uinput.
func (m *linuxMousepad) writeEvents(events []inputEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&events[0])), len(events)*24)
	return unix.Write(m.fd, buf)
}
