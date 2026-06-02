//go:build !cgo

package websocket

import (
	"encoding/json"
	"testing"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
	"omnipanel-go/internal/devices"
	"omnipanel-go/internal/mpris"
	"omnipanel-go/internal/rssfeed"
	"omnipanel-go/internal/speech"
)

type mockState struct {
	cfg       *config.Config
	db        *databus.DataBus
	jm        *devices.JoystickManager
	mm        *devices.MousepadManager
	km        *devices.KeyboardManager
	broadcast []map[string]any
}

func newMockState() *mockState {
	return &mockState{
		cfg: &config.Config{},
		db:  databus.New(),
		jm:  devices.New(0),
		mm:  devices.NewMousepad(0),
		km:  devices.NewKeyboard(0),
	}
}

func (m *mockState) RegisterClient(ch chan []byte) uint64          { return 1 }
func (m *mockState) UnregisterClient(ch chan []byte)                {}
func (m *mockState) Broadcast(msg []byte)                           {}
func (m *mockState) BroadcastJSON(msg map[string]any)               { m.broadcast = append(m.broadcast, msg) }
func (m *mockState) UpdateJoystickCount(count uint8)                {}
func (m *mockState) GetConfig() *config.Config                      { return m.cfg }
func (m *mockState) LoadPanelJSON(panelName string) ([]byte, error) { return nil, nil }
func (m *mockState) StartDataBroadcast()                             {}
func (m *mockState) PushData(key string, value any, unit string)    { m.db.Set(key, value, unit) }
func (m *mockState) GetJoystickManager() *devices.JoystickManager   { return m.jm }
func (m *mockState) GetMousepadManager() *devices.MousepadManager   { return m.mm }
func (m *mockState) GetKeyboardManager() *devices.KeyboardManager   { return m.km }
func (m *mockState) GetDataBus() *databus.DataBus                   { return m.db }
func (m *mockState) GetSpeechManager() *speech.SpeechManager        { return nil }
func (m *mockState) GetMPRISWatcher() *mpris.Watcher                { return nil }
func (m *mockState) GetRSSManager() *rssfeed.Manager                { return nil }

func TestParseHelpersAndBasicRouting(t *testing.T) {
	if got := parseUintField(json.RawMessage(`{"v":5}`), "v"); got != 5 {
		t.Fatalf("parseUintField number got %d", got)
	}
	if got := parseUintField(json.RawMessage(`{"v":"7"}`), "v"); got != 7 {
		t.Fatalf("parseUintField string got %d", got)
	}
	if got := parseStringField(json.RawMessage(`{"s":"ok"}`), "s"); got != "ok" {
		t.Fatalf("parseStringField got %q", got)
	}

	m := newMockState()
	HandleMessage(m, 1, "{bad")
	HandleMessage(m, 1, `{"type":"heartbeat"}`)
	if len(m.broadcast) == 0 || m.broadcast[len(m.broadcast)-1]["type"] != "heartbeat-ack" {
		t.Fatalf("expected heartbeat-ack broadcast, got %#v", m.broadcast)
	}

	HandleMessage(m, 1, `{"type":"push-data","data":{"key":"k","value":12,"unit":"u"}}`)
	v, ok := m.db.Get("k")
	if !ok || v.Unit != "u" {
		t.Fatalf("expected databus value for key k, got %#v ok=%v", v, ok)
	}

	HandleMessage(m, 1, `{"type":"enter-fullscreen"}`)
	HandleMessage(m, 1, `{"type":"exit-fullscreen"}`)
	HandleMessage(m, 1, `{"type":"open-url","data":{}}`)
	HandleMessage(m, 1, `{"type":"unknown"}`)
}

func TestSimulationCommands(t *testing.T) {
	m := newMockState()

	// Button simulation
	HandleMessage(m, 1, `{"type":"simulate-button","data":{"js":"0","id":"0","state":1}}`)

	// Slider
	HandleMessage(m, 1, `{"type":"simulate-slider","data":{"js":"0","id":"1","value":128}}`)

	// Joystick 2D
	HandleMessage(m, 1, `{"type":"simulate-joystick","data":{"js":"0","id":"0","value":{"x":100,"y":50}}}`)

	// Mousepad move
	HandleMessage(m, 1, `{"type":"simulate-mousepad","data":{"js":"0","value":{"x":10,"y":20}}}`)

	// Mousewheel
	HandleMessage(m, 1, `{"type":"simulate-mousewheel","data":{"js":"0","delta":3}}`)

	// Mouse button
	HandleMessage(m, 1, `{"type":"simulate-mousebtn","data":{"js":"0","btn":"left","state":1}}`)
	HandleMessage(m, 1, `{"type":"simulate-mousebtn","data":{"js":"0","btn":"right","state":0}}`)
	HandleMessage(m, 1, `{"type":"simulate-mousebtn","data":{"js":"0","btn":"middle","state":1}}`)
	HandleMessage(m, 1, `{"type":"simulate-mousebtn","data":{"js":"0","btn":"unknown","state":0}}`)

	// Keyboard key
	HandleMessage(m, 1, `{"type":"simulate-keyboard","data":{"keyboard_index":"0","key":"a","state":1}}`)

	// Keyboard combo
	HandleMessage(m, 1, `{"type":"simulate-keyboard","data":{"keyboard_index":"0","key":"ctrl+a","state":1}}`)

	// Save joystick count
	HandleMessage(m, 1, `{"type":"save-joystick-count","data":3}`)

	// Execute command (shell)
	HandleMessage(m, 1, `{"type":"execute-command","data":{"block_id":"b1","command_type":"shell","command":"echo test","params":{"x":"y"}}}`)

	// Unhandled message should not crash
	HandleMessage(m, 1, `{"type":"simulate-button","data":{}}`)
	HandleMessage(m, 1, `{"type":"simulate-button","data":{"js":"","id":"","state":1}}`)
	HandleMessage(m, 1, `{"type":"simulate-keyboard","data":{"key":""}}`)
	HandleMessage(m, 1, `{"type":"unknown-new-type"}`)
}
