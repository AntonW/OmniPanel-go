package devices

import "testing"

type fakeJoystick struct {
	buttonCalls int
	axisCalls   int
	closed      bool
}

func (f *fakeJoystick) SendButton(id int, state uint8) { f.buttonCalls++ }
func (f *fakeJoystick) SendAxis(id int, value uint8)   { f.axisCalls++ }
func (f *fakeJoystick) Close()                         { f.closed = true }

type fakeMousepad struct {
	moveCalls   int
	wheelCalls  int
	buttonCalls int
	closed      bool
}

func (f *fakeMousepad) SendMove(dx, dy int32)           { f.moveCalls++ }
func (f *fakeMousepad) SendWheel(delta int32)           { f.wheelCalls++ }
func (f *fakeMousepad) SendButton(btn int, state uint8) { f.buttonCalls++ }
func (f *fakeMousepad) Close()                          { f.closed = true }

type fakeKeyboard struct {
	keyCalls   int
	comboCalls int
	closed     bool
}

func (f *fakeKeyboard) SendKey(code int, state uint8)      { f.keyCalls++ }
func (f *fakeKeyboard) SendCombo(codes []int, state uint8) { f.comboCalls++ }
func (f *fakeKeyboard) Close()                              { f.closed = true }

func TestJoystickManagerMethods(t *testing.T) {
	f := &fakeJoystick{}
	m := &JoystickManager{joysticks: []Joystick{f}}

	m.Send(Command{Type: ButtonType, Js: 0, Id: 1, Value: 1})
	m.Send(Command{Type: AxisType, Js: 0, Id: 2, Value: 2})
	m.Send(Command{Type: MouseType, Js: 0}) // unknown for joystick manager
	m.Send(Command{Type: ButtonType, Js: 3})

	if f.buttonCalls != 1 || f.axisCalls != 1 {
		t.Fatalf("unexpected send calls: button=%d axis=%d", f.buttonCalls, f.axisCalls)
	}

	m.Reload(0)
	if !f.closed {
		t.Fatal("reload should close previous joystick")
	}
	if len(m.joysticks) != 0 {
		t.Fatalf("reload(0) should clear joysticks, got len=%d", len(m.joysticks))
	}

	m.Close()
	if m.joysticks != nil {
		t.Fatal("close should nil joystick slice")
	}
}

func TestMousepadManagerMethods(t *testing.T) {
	f := &fakeMousepad{}
	m := &MousepadManager{mousepads: []Mousepad{f}}

	m.SendMove(0, 1, 2)
	m.SendWheel(0, 3)
	m.SendButton(0, MouseBtnLeft, 1)
	m.SendMove(2, 1, 2)
	m.SendWheel(2, 3)
	m.SendButton(2, MouseBtnLeft, 1)

	if f.moveCalls != 1 || f.wheelCalls != 1 || f.buttonCalls != 1 {
		t.Fatalf("unexpected mousepad calls: move=%d wheel=%d button=%d", f.moveCalls, f.wheelCalls, f.buttonCalls)
	}

	m.Reload(0)
	if !f.closed {
		t.Fatal("reload should close previous mousepad")
	}
	if len(m.mousepads) != 0 {
		t.Fatalf("reload(0) should clear mousepads, got len=%d", len(m.mousepads))
	}

	m.Close()
	if m.mousepads != nil {
		t.Fatal("close should nil mousepad slice")
	}
}

func TestKeyboardManagerMethods(t *testing.T) {
	f := &fakeKeyboard{}
	m := &KeyboardManager{keyboards: []Keyboard{f}}

	m.SendKey(0, 30, 1)
	m.SendCombo(0, []int{29, 30}, 1)
	m.SendKey(2, 30, 1)
	m.SendCombo(2, []int{29, 30}, 1)

	if f.keyCalls != 1 || f.comboCalls != 1 {
		t.Fatalf("unexpected keyboard calls: key=%d combo=%d", f.keyCalls, f.comboCalls)
	}

	m.Reload(0)
	if !f.closed {
		t.Fatal("reload should close previous keyboard")
	}
	if len(m.keyboards) != 0 {
		t.Fatalf("reload(0) should clear keyboards, got len=%d", len(m.keyboards))
	}

	m.Close()
	if m.keyboards != nil {
		t.Fatal("close should nil keyboard slice")
	}
}

func TestConstructorsWithZeroDevices(t *testing.T) {
	if got := New(0); got == nil {
		t.Fatal("New(0) returned nil")
	}
	if got := NewMousepad(0); got == nil {
		t.Fatal("NewMousepad(0) returned nil")
	}
	if got := NewKeyboard(0); got == nil {
		t.Fatal("NewKeyboard(0) returned nil")
	}
}

