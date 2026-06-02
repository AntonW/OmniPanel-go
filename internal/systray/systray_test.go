package systray

import "testing"

func TestNewAndIsHeadless(t *testing.T) {
	tray := New(nil, nil, nil)
	if tray == nil {
		t.Fatal("New returned nil")
	}
	_ = IsHeadless()
}

