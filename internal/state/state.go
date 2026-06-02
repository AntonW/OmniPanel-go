// Package state provides the central AppState hub that holds references to all
// subsystems (config, joystick/mousepad/keyboard managers, databus, speech manager,
// RSS feed manager) and manages WebSocket client connections.
//
// AppState uses two separate mutexes to protect different concerns:
//   - mu: protects Config (read/written by HTTP handlers)
//   - broadcastMu: protects the clients map and client ID tracking (read/written by WebSocket handlers)
//
// The broadcast mechanism uses a map[chan []byte]struct{} set pattern, where
// each WebSocket client has a dedicated channel. Broadcasting is non-blocking
// (select + default) to prevent slow clients from blocking the entire system.
//
// Each client is assigned a unique uint64 ID on connection, which is used for
// targeted message delivery (e.g., RSS updates are sent to specific clients
// based on their per-client seen-entry tracking).
//
// AppStateInterface defines the methods required by the websocket handler.
// Both AppState (default mode) and Agent (connect mode) implement this interface,
// allowing the same message handling code to work in both deployment modes.
//
// See docs/tutorials/03-state-and-concurrency.md for a detailed walkthrough.
package state

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
	"omnipanel-go/internal/devices"
	"omnipanel-go/internal/mpris"
	"omnipanel-go/internal/rssfeed"
	"omnipanel-go/internal/speech"
)

// AppStateInterface defines the methods required by the websocket handler.
// Both AppState (default mode) and Agent (connect mode) implement this interface.
type AppStateInterface interface {
	RegisterClient(ch chan []byte) uint64
	UnregisterClient(ch chan []byte)
	Broadcast(msg []byte)
	BroadcastJSON(msg map[string]any)
	UpdateJoystickCount(count uint8)
	GetConfig() *config.Config
	LoadPanelJSON(panelName string) ([]byte, error)
	StartDataBroadcast()
	PushData(key string, value any, unit string)
	GetJoystickManager() *devices.JoystickManager
	GetMousepadManager() *devices.MousepadManager
	GetKeyboardManager() *devices.KeyboardManager
	GetDataBus() *databus.DataBus
	GetSpeechManager() *speech.SpeechManager
	GetMPRISWatcher() *mpris.Watcher
	GetRSSManager() *rssfeed.Manager
}

// AppState is the central hub of the application.
// It holds references to all subsystems and manages WebSocket client connections.
// Two separate mutexes protect different concerns:
//   - mu: protects Config (read/written by HTTP handlers)
//   - broadcastMu: protects the clients map and client ID tracking (read/written by WebSocket handlers)
type AppState struct {
	mu              sync.RWMutex
	Config          *config.Config
	ConfigPath      string
	UserPath        string
	StaticDir       string
	JoystickManager *devices.JoystickManager
	MousepadManager *devices.MousepadManager
	KeyboardManager *devices.KeyboardManager
	DataBus         *databus.DataBus
	SpeechManager   *speech.SpeechManager
	// MPRISWatcher monitors media players (Spotify, VLC, etc.) via D-Bus on Linux
	// or SMTC on Windows, and publishes their state to the DataBus.
	MPRISWatcher *mpris.Watcher
	// RSSManager handles RSS/Atom feed polling and pushes updates to WebSocket clients.
	RSSManager *rssfeed.Manager

	broadcastMu  sync.RWMutex
	clients      map[chan []byte]struct{}
	nextClientID uint64
	clientIDs    map[chan []byte]uint64
}

// New creates and initializes the application state.
// It creates joystick/mousepad managers, the databus, and starts the data broadcast loop.
func New(cfg *config.Config, configPath, userPath, baseDir string) *AppState {
	staticDir := filepath.Join(baseDir, "static")
	jsMgr := devices.New(cfg.NumJoysticks)
	mpMgr := devices.NewMousepad(cfg.NumJoysticks)
	kbMgr := devices.NewKeyboard(cfg.NumJoysticks)
	db := databus.New()

	slog.Info("Created joystick manager", "count", cfg.NumJoysticks)
	slog.Info("Created mousepad manager", "count", cfg.NumJoysticks)
	slog.Info("Created keyboard manager", "count", cfg.NumJoysticks)
	slog.Info("Created data bus")

	app := &AppState{
		Config:          cfg,
		ConfigPath:      configPath,
		UserPath:        userPath,
		StaticDir:       staticDir,
		JoystickManager: jsMgr,
		MousepadManager: mpMgr,
		KeyboardManager: kbMgr,
		DataBus:         db,
		clients:         make(map[chan []byte]struct{}),
		clientIDs:       make(map[chan []byte]uint64),
	}

	sm := speech.New(&cfg.Speech, app, jsMgr, userPath)
	app.SpeechManager = sm

	mprisWatcher := mpris.New(&cfg.MediaPlayer, db, slog.Default())
	if mprisWatcher != nil {
		if err := mprisWatcher.Start(); err != nil {
			slog.Warn("MPRIS watcher failed to start", "error", err)
		}
	}
	app.MPRISWatcher = mprisWatcher

	app.RSSManager = rssfeed.New(func(clientID uint64, blockID string, entries []rssfeed.EntryWithNew) {
		app.broadcastToClient(clientID, map[string]any{
			"type": "rss-update",
			"data": map[string]any{
				"block_id": blockID,
				"entries":  entries,
			},
		})
	})
	app.RSSManager.Start()

	app.StartDataBroadcast()

	return app
}

// RegisterClient adds a channel to the broadcast set and assigns a unique client ID.
// The channel receives JSON messages broadcast to all connected clients.
// The returned client ID is used for targeted message delivery (e.g., RSS updates).
func (s *AppState) RegisterClient(ch chan []byte) uint64 {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	s.nextClientID++
	clientID := s.nextClientID
	s.clients[ch] = struct{}{}
	s.clientIDs[ch] = clientID
	return clientID
}

// UnregisterClient removes a channel from the broadcast set and closes it.
// Called when a WebSocket connection disconnects.
func (s *AppState) UnregisterClient(ch chan []byte) {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	delete(s.clients, ch)
	delete(s.clientIDs, ch)
	close(ch)
}

// Broadcast sends a message to all registered client channels.
// Uses non-blocking send (select + default) to prevent slow clients from blocking.
func (s *AppState) Broadcast(msg []byte) {
	s.broadcastMu.RLock()
	defer s.broadcastMu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// BroadcastJSON marshals a map to JSON and broadcasts it to all clients.
func (s *AppState) BroadcastJSON(msg map[string]any) {
	data, _ := json.Marshal(msg)
	s.Broadcast(data)
}

// UpdateJoystickCount changes the number of virtual joysticks at runtime.
// It reloads the joystick manager and persists the change to config.
func (s *AppState) UpdateJoystickCount(count uint8) {
	s.mu.Lock()
	s.Config.NumJoysticks = count
	s.mu.Unlock()

	s.JoystickManager.Reload(count)
	slog.Info("Joystick count updated", "count", count)

	_ = s.Config.Save(s.ConfigPath)
}

// GetConfig returns the current config under a read lock.
func (s *AppState) GetConfig() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Config
}

// LoadPanelJSON reads a panel definition from user/panels/<name>.json.
func (s *AppState) LoadPanelJSON(panelName string) ([]byte, error) {
	path := filepath.Join(s.UserPath, "panels", panelName+".json")
	return os.ReadFile(path)
}

// StartDataBroadcast kicks off two background goroutines:
// 1. DataBus metric collection (every 500ms)
// 2. Snapshot broadcast to all WebSocket clients (every 500ms)
func (s *AppState) StartDataBroadcast() {
	s.DataBus.StartMetrics(500 * time.Millisecond)

	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for range ticker.C {
			snapshot := s.DataBus.Snapshot()
			s.BroadcastJSON(map[string]any{
				"type": "data-update",
				"data": snapshot,
			})
		}
	}()
	slog.Info("Data broadcast started")
}

// PushData adds a custom metric to the DataBus.
// Used by the WebSocket handler for push-data messages.
func (s *AppState) PushData(key string, value any, unit string) {
	s.DataBus.Set(key, value, unit)
}

// broadcastToClient sends a JSON message to a specific client by ID.
// Used by the RSS manager for targeted per-client updates (e.g., new feed entries).
// Non-blocking: drops the message if the client's channel is full.
func (s *AppState) broadcastToClient(clientID uint64, msg map[string]any) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Warn("Failed to marshal RSS update", "error", err)
		return
	}

	s.broadcastMu.RLock()
	defer s.broadcastMu.RUnlock()
	for ch, id := range s.clientIDs {
		if id == clientID {
			select {
			case ch <- data:
			default:
			}
			return
		}
	}
}

// Close cleans up all virtual input devices and subsystems.
// Logs each subsystem shutdown to help identify hangs during graceful exit.
// Called via defer in main.go to ensure cleanup on exit.
func (s *AppState) Close() {
	slog.Info("Closing joystick manager...")
	s.JoystickManager.Close()
	slog.Info("Closing mousepad manager...")
	s.MousepadManager.Close()
	slog.Info("Closing keyboard manager...")
	s.KeyboardManager.Close()
	slog.Info("Closing speech manager...")
	s.SpeechManager.Close()
	slog.Info("Closing MPRIS watcher...")
	if s.MPRISWatcher != nil {
		s.MPRISWatcher.Close()
	}
	slog.Info("Closing RSS manager...")
	s.RSSManager.Close()
	slog.Info("All subsystems closed")
}

// GetJoystickManager returns the joystick manager.
func (s *AppState) GetJoystickManager() *devices.JoystickManager {
	return s.JoystickManager
}

// GetMousepadManager returns the mousepad manager.
func (s *AppState) GetMousepadManager() *devices.MousepadManager {
	return s.MousepadManager
}

// GetKeyboardManager returns the keyboard manager.
func (s *AppState) GetKeyboardManager() *devices.KeyboardManager {
	return s.KeyboardManager
}

// GetDataBus returns the databus.
func (s *AppState) GetDataBus() *databus.DataBus {
	return s.DataBus
}

// GetSpeechManager returns the speech manager.
func (s *AppState) GetSpeechManager() *speech.SpeechManager {
	return s.SpeechManager
}

// GetMPRISWatcher returns the media player watcher instance.
// On Linux this uses MPRIS; on Windows it uses SMTC.
func (s *AppState) GetMPRISWatcher() *mpris.Watcher {
	return s.MPRISWatcher
}

// GetRSSManager returns the RSS manager.
func (s *AppState) GetRSSManager() *rssfeed.Manager {
	return s.RSSManager
}
