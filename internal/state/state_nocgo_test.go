//go:build !cgo

package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
	"omnipanel-go/internal/devices"
)

func TestClientBroadcastAndData(t *testing.T) {
	s := &AppState{
		Config:          &config.Config{},
		ConfigPath:      filepath.Join(t.TempDir(), "config.json"),
		DataBus:         databus.New(),
		JoystickManager: devices.New(0),
		MousepadManager: devices.NewMousepad(0),
		KeyboardManager: devices.NewKeyboard(0),
		clients:         make(map[chan []byte]struct{}),
		clientIDs:       make(map[chan []byte]uint64),
	}

	ch := make(chan []byte, 2)
	id := s.RegisterClient(ch)
	if id == 0 {
		t.Fatal("expected non-zero client id")
	}

	s.Broadcast([]byte("x"))
	if string(<-ch) != "x" {
		t.Fatal("broadcast payload mismatch")
	}

	s.BroadcastJSON(map[string]any{"k": 1})
	var msg map[string]any
	if err := json.Unmarshal(<-ch, &msg); err != nil || msg["k"].(float64) != 1 {
		t.Fatalf("unexpected broadcast json: %v %v", err, msg)
	}

	s.PushData("a", 2, "%")
	if v, ok := s.DataBus.Get("a"); !ok || v.Unit != "%" {
		t.Fatalf("unexpected databus value: %#v ok=%v", v, ok)
	}

	s.broadcastToClient(id, map[string]any{"t": true})
	if err := json.Unmarshal(<-ch, &msg); err != nil || msg["t"] != true {
		t.Fatalf("unexpected targeted json: %v %v", err, msg)
	}

	s.UnregisterClient(ch)
}

func TestLoadPanelJSON(t *testing.T) {
	userPath := t.TempDir()
	panelDir := filepath.Join(userPath, "panels")
	if err := os.MkdirAll(panelDir, 0o755); err != nil {
		t.Fatalf("mkdir panels: %v", err)
	}
	panelPath := filepath.Join(panelDir, "demo.json")
	if err := os.WriteFile(panelPath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write panel: %v", err)
	}

	s := &AppState{UserPath: userPath}
	data, err := s.LoadPanelJSON("demo")
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("LoadPanelJSON failed: err=%v data=%q", err, string(data))
	}
}

func TestJoystickCountUpdate(t *testing.T) {
	s := &AppState{
		Config:          &config.Config{NumJoysticks: 2},
		JoystickManager: devices.New(2),
	}

	s.UpdateJoystickCount(4)
	if s.Config.NumJoysticks != 4 {
		t.Fatalf("expected Config.NumJoysticks=4, got %d", s.Config.NumJoysticks)
	}
}

