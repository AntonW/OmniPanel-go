//go:build !linux && (!windows || !cgo)

package devices

import "fmt"

// KeyNameToCode is empty on unsupported/stub builds.
// It exists so callers can compile in environments where the real
// platform keyboard implementation is not available (e.g., CGO disabled).
var KeyNameToCode = map[string]int{}

// stubJoystick is a no-op joystick implementation for unsupported platforms.
type stubJoystick struct{}

// newJoystick returns an error on unsupported platforms.
func newJoystick(index int) (*stubJoystick, error) {
	return nil, fmt.Errorf("virtual joystick not supported on this platform (linux and windows only)")
}

func (j *stubJoystick) SendButton(id int, state uint8) {}
func (j *stubJoystick) SendAxis(id int, value uint8)   {}
func (j *stubJoystick) Close()                         {}

// stubMousepad is a no-op mousepad implementation for unsupported platforms.
type stubMousepad struct{}

// newMousepad returns an error on unsupported platforms.
func newMousepad(index int) (*stubMousepad, error) {
	return nil, fmt.Errorf("virtual mouse not supported on this platform (linux and windows only)")
}

func (m *stubMousepad) SendMove(dx, dy int32)           {}
func (m *stubMousepad) SendWheel(delta int32)           {}
func (m *stubMousepad) SendButton(btn int, state uint8) {}
func (m *stubMousepad) Close()                          {}

// stubKeyboard is a no-op keyboard implementation for unsupported platforms.
type stubKeyboard struct{}

// newKeyboard returns an error on unsupported platforms.
func newKeyboard(index int) (*stubKeyboard, error) {
	return nil, fmt.Errorf("virtual keyboard not supported on this platform (linux and windows only)")
}

func (k *stubKeyboard) SendKey(code int, state uint8)      {}
func (k *stubKeyboard) SendCombo(codes []int, state uint8) {}
func (k *stubKeyboard) Close()                             {}
