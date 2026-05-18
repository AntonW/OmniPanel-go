// Package devices creates virtual joystick, mouse, and keyboard devices using
// platform-specific APIs. When the web panel sends input events, this package
// translates them into OS-level input that the system treats as real hardware.
//
// Platform implementations (selected at compile time via build tags):
//   - Linux: uinput kernel subsystem (/dev/uinput)
//   - Windows: vJoy driver (joystick) + SendInput API (mouse, keyboard)
//   - Other: no-op stub (app runs but virtual input is unavailable)
//
// The Joystick, Mousepad, and Keyboard interfaces enable swapping implementations
// without changing the calling code. Platform-agnostic constants
// (MouseBtnLeft, etc.) prevent native codes from leaking into higher layers.
//
// See docs/tutorials/08-virtual-input.md for a detailed walkthrough.
package devices

import (
	"log/slog"
	"sync"
)

// CommandType identifies the kind of input event.
type CommandType string

const (
	AxisType   CommandType = "axis"
	ButtonType CommandType = "button"
	MouseType  CommandType = "mouse"
	WheelType  CommandType = "wheel"
)

// Platform-agnostic mouse button constants.
// Each platform implementation maps these to native codes.
const (
	MouseBtnLeft   = 0
	MouseBtnRight  = 1
	MouseBtnMiddle = 2
)

// Command represents an abstract input event sent from the WebSocket handler
// to the joystick manager.
type Command struct {
	Type  CommandType
	Js    int
	Id    int
	Value uint8
}

// MouseCommand represents a relative mouse movement.
type MouseCommand struct {
	Dx int32
	Dy int32
}

// WheelCommand represents a scroll wheel event.
type WheelCommand struct {
	Delta int32
}

// Joystick is the interface for virtual joystick implementations.
type Joystick interface {
	SendButton(id int, state uint8)
	SendAxis(id int, value uint8)
	Close()
}

// Mousepad is the interface for virtual mouse implementations.
type Mousepad interface {
	SendMove(dx, dy int32)
	SendWheel(delta int32)
	SendButton(btn int, state uint8)
	Close()
}

// Keyboard is the interface for virtual keyboard implementations.
// SendKey presses or releases a single key by its platform-specific code.
// SendCombo presses or releases multiple keys simultaneously (e.g., Ctrl+A).
type Keyboard interface {
	SendKey(code int, state uint8)
	SendCombo(codes []int, state uint8)
	Close()
}

// JoystickManager manages multiple virtual joystick devices.
// It routes Command messages to the correct joystick instance.
type JoystickManager struct {
	mu        sync.Mutex
	joysticks []Joystick
}

// MousepadManager manages multiple virtual mouse devices.
type MousepadManager struct {
	mu        sync.Mutex
	mousepads []Mousepad
}

// KeyboardManager manages multiple virtual keyboard devices.
// It routes key and combo events to the correct keyboard instance.
type KeyboardManager struct {
	mu        sync.Mutex
	keyboards []Keyboard
}

// New creates a JoystickManager and initializes the requested number of joysticks.
func New(count uint8) *JoystickManager {
	js := &JoystickManager{}
	js.Reload(count)
	return js
}

// Send dispatches a command to the appropriate joystick.
// Silently ignores commands for non-existent joysticks.
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

// Reload destroys all existing joysticks and creates new ones.
// Called when the user changes the joystick count at runtime.
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

// Close destroys all virtual joystick devices.
func (m *JoystickManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, js := range m.joysticks {
		if js != nil {
			js.Close()
		}
	}
	m.joysticks = nil
}

// NewMousepad creates a MousepadManager and initializes the requested number of mousepads.
func NewMousepad(count uint8) *MousepadManager {
	mp := &MousepadManager{}
	mp.Reload(count)
	return mp
}

// SendMove sends a relative mouse movement to the specified mousepad.
func (m *MousepadManager) SendMove(index int, dx, dy int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= len(m.mousepads) || m.mousepads[index] == nil {
		return
	}
	m.mousepads[index].SendMove(dx, dy)
}

// SendWheel sends a scroll wheel event to the specified mousepad.
func (m *MousepadManager) SendWheel(index int, delta int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= len(m.mousepads) || m.mousepads[index] == nil {
		return
	}
	m.mousepads[index].SendWheel(delta)
}

// SendButton sends a mouse button press/release to the specified mousepad.
func (m *MousepadManager) SendButton(index int, btn int, state uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= len(m.mousepads) || m.mousepads[index] == nil {
		return
	}
	m.mousepads[index].SendButton(btn, state)
}

// Reload destroys all existing mousepads and creates new ones.
func (m *MousepadManager) Reload(count uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mp := range m.mousepads {
		if mp != nil {
			mp.Close()
		}
	}
	m.mousepads = make([]Mousepad, count)
	for i := range count {
		mp, err := newMousepad(int(i))
		if err != nil {
			slog.Warn("Failed to create mousepad", "index", i, "error", err)
			continue
		}
		m.mousepads[i] = mp
	}
}

// Close destroys all virtual mouse devices.
func (m *MousepadManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mp := range m.mousepads {
		if mp != nil {
			mp.Close()
		}
	}
	m.mousepads = nil
}

// NewKeyboard creates a KeyboardManager and initializes the requested number of keyboards.
// Each keyboard is a separate virtual keyboard device visible to the OS.
func NewKeyboard(count uint8) *KeyboardManager {
	kb := &KeyboardManager{}
	kb.Reload(count)
	return kb
}

// SendKey sends a single key press/release to the specified keyboard.
func (m *KeyboardManager) SendKey(index int, code int, state uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= len(m.keyboards) || m.keyboards[index] == nil {
		return
	}
	m.keyboards[index].SendKey(code, state)
}

// SendCombo sends a key combination press/release to the specified keyboard.
func (m *KeyboardManager) SendCombo(index int, codes []int, state uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= len(m.keyboards) || m.keyboards[index] == nil {
		return
	}
	m.keyboards[index].SendCombo(codes, state)
}

// Reload destroys all existing keyboards and creates new ones.
func (m *KeyboardManager) Reload(count uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, kb := range m.keyboards {
		if kb != nil {
			kb.Close()
		}
	}
	m.keyboards = make([]Keyboard, count)
	for i := range count {
		k, err := newKeyboard(int(i))
		if err != nil {
			slog.Warn("Failed to create keyboard", "index", i, "error", err)
			continue
		}
		m.keyboards[i] = k
	}
}

// Close destroys all virtual keyboard devices.
func (m *KeyboardManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, kb := range m.keyboards {
		if kb != nil {
			kb.Close()
		}
	}
	m.keyboards = nil
}
