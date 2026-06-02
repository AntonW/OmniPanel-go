// Package agent implements the host agent mode for distributed deployment.
//
// The host agent connects to a central relay server via WebSocket and runs all
// subsystems locally (joystick, mousepad, keyboard, databus, speech, media player, RSS).
// This allows the host machine to be behind a firewall — it initiates an outbound
// WebSocket connection to the relay server, which then forwards browser commands
// to the host and relays host responses back to browsers.
//
// Authentication:
//
// When auth_token is configured, the agent appends it to the WebSocket URL
// as a query parameter (ws://server/ws?type=host&token=xxx). The relay server
// validates this token before accepting the connection.
//
// Architecture:
//
//	WebSocket client to relay server (ws:// or wss://)
//	No HTTP server — all UI served by the central server
//	All subsystems run locally: joystick, mousepad, keyboard, databus, speech, media player, RSS
//	Receives forwarded browser commands from server, executes locally
//	Sends results back to server for broadcast to browsers
//
// RSS Updates in Connect Mode:
//
// Unlike default mode where broadcastToClient sends RSS updates to specific client
// channels, the agent uses BroadcastJSON because it has no direct browser connections.
// The relay server broadcasts RSS updates to all browsers, and each browser's client-side
// code processes only updates for matching block_id. Per-client "new entry" tracking is
// maintained server-side via seenPerClient map and client-side via rssSeenEntries Set.
//
// Connection flow:
//
//  1. Connect to ws://server/ws?type=host or wss://server/ws?type=host (with token if auth_token is set)
//  2. Send {"type": "host-register"} to register with server
//  3. Receive forwarded browser commands
//  4. Execute commands locally, send results back
//  5. On disconnect: auto-reconnect with exponential backoff (1s → 2s → 4s → max 30s)
//
// Usage:
//
//	./omnipanel connect 10.0.0.1:3000              # Connect to server at address (ws://)
//	./omnipanel connect wss://10.0.0.1:3000        # Connect with WebSocket Secure
//	./omnipanel connect                             # Uses server_address from config.json
//
// Configuration:
//
// Set "server_address": "10.0.0.1:3000" or "wss://10.0.0.1:3000" in config.json.
// Set "auth_token": "your-secret" in config.json or use OMNIPANEL_AUTH_TOKEN env var.
package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fasthttp/websocket"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
	"omnipanel-go/internal/devices"
	"omnipanel-go/internal/mediautil"
	"omnipanel-go/internal/mediacontrol"
	"omnipanel-go/internal/rssfeed"
	"omnipanel-go/internal/speech"
	wsHandler "omnipanel-go/internal/websocket"
)

// Agent holds the host agent state for connect mode.
// It maintains a WebSocket connection to the relay server and runs all
// subsystems locally. The agent auto-reconnects on disconnect with
// exponential backoff.
type Agent struct {
	conn   *websocket.Conn
	connMu sync.RWMutex

	Config     *config.Config
	ConfigPath string
	UserPath   string
	BaseDir    string

	JoystickManager *devices.JoystickManager
	MousepadManager *devices.MousepadManager
	KeyboardManager *devices.KeyboardManager
	DataBus         *databus.DataBus
	SpeechManager   *speech.SpeechManager
	MPRISWatcher    *mediacontrol.Watcher
	RSSManager      *rssfeed.Manager

	broadcastMu  sync.RWMutex
	clients      map[chan []byte]struct{}
	nextClientID uint64
	clientIDs    map[chan []byte]uint64

	stopChan chan struct{}
}

// New creates a new host agent with all subsystems initialized.
// The agent does not connect to the server until Run() is called.
func New(cfg *config.Config, configPath, userPath, baseDir string) *Agent {
	jsMgr := devices.New(cfg.NumJoysticks)
	mpMgr := devices.NewMousepad(cfg.NumJoysticks)
	kbMgr := devices.NewKeyboard(cfg.NumJoysticks)
	db := databus.New()

	slog.Info("Created joystick manager", "count", cfg.NumJoysticks)
	slog.Info("Created mousepad manager", "count", cfg.NumJoysticks)
	slog.Info("Created keyboard manager", "count", cfg.NumJoysticks)
	slog.Info("Created data bus")

	a := &Agent{
		Config:          cfg,
		ConfigPath:      configPath,
		UserPath:        userPath,
		BaseDir:         baseDir,
		JoystickManager: jsMgr,
		MousepadManager: mpMgr,
		KeyboardManager: kbMgr,
		DataBus:         db,
		clients:         make(map[chan []byte]struct{}),
		clientIDs:       make(map[chan []byte]uint64),
		stopChan:        make(chan struct{}),
	}

	sm := speech.New(&cfg.Speech, a, jsMgr, userPath)
	a.SpeechManager = sm

	mprisWatcher := mediacontrol.New(&cfg.MediaPlayer, db, slog.Default())
	if mprisWatcher != nil {
		if err := mprisWatcher.Start(); err != nil {
			slog.Warn("MPRIS watcher failed to start", "error", err)
		}
	}
	a.MPRISWatcher = mprisWatcher

	a.RSSManager = rssfeed.New(func(clientID uint64, blockID string, entries []rssfeed.EntryWithNew) {
		a.BroadcastJSON(map[string]any{
			"type": "rss-update",
			"data": map[string]any{
				"block_id": blockID,
				"entries":  entries,
			},
		})
	})
	a.RSSManager.Start()

	a.StartDataBroadcast()

	return a
}

// buildWebSocketURL constructs the WebSocket URL from the server address.
// Supports full URLs (ws:// or wss://) or plain host:port (defaults to ws://).
func buildWebSocketURL(serverAddr string, authToken string) string {
	url := serverAddr
	if serverAddr != "" && !strings.HasPrefix(serverAddr, "ws://") && !strings.HasPrefix(serverAddr, "wss://") {
		url = "ws://" + serverAddr
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "ws?type=host"
	if authToken != "" {
		url += fmt.Sprintf("&token=%s", authToken)
	}
	return url
}

// Run connects to the relay server and starts the agent loop with auto-reconnect.
// It blocks until Stop() is called. Connection failures trigger exponential backoff
// (1s → 2s → 4s → max 30s) and automatic reconnection attempts.
func (a *Agent) Run(serverAddr string) {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-a.stopChan:
			return
		default:
		}

		url := buildWebSocketURL(serverAddr, a.Config.AuthToken)
		slog.Info("Connecting to server", "url", url)

		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			slog.Error("Failed to connect to server", "error", err, "retry_in", backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		slog.Info("Connected to server")

		a.connMu.Lock()
		a.conn = conn
		a.connMu.Unlock()

		a.sendJSON(map[string]any{"type": "host-register"})

		a.runConnection(conn)

		a.connMu.Lock()
		a.conn = nil
		a.connMu.Unlock()

		slog.Warn("Disconnected from server, reconnecting...", "retry_in", backoff)
		time.Sleep(backoff)
		backoff = min(backoff*2, maxBackoff)
	}
}

// runConnection handles a single WebSocket connection until disconnect.
// It runs a heartbeat ticker (every 15s) and processes incoming messages.
func (a *Agent) runConnection(conn *websocket.Conn) {
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-heartbeatTicker.C:
				a.sendJSON(map[string]any{"type": "heartbeat"})
			case <-done:
				return
			}
		}
	}()

	defer close(done)

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if msgType == websocket.BinaryMessage {
			wsHandler.HandleAudioChunk(a, msg)
		} else {
			var msgMap map[string]json.RawMessage
			if json.Unmarshal(msg, &msgMap) == nil {
				var msgTypeStr string
				json.Unmarshal(msgMap["type"], &msgTypeStr)

				if msgTypeStr == "mpris-request" {
					a.handleMPRISRequest(string(msg))
					continue
				}
			}

			wsHandler.HandleMessage(a, 0, string(msg))
		}
	}
}

// Stop signals the agent to stop reconnecting and closes the current WebSocket connection.
func (a *Agent) Stop() {
	close(a.stopChan)
	a.connMu.Lock()
	if a.conn != nil {
		a.conn.Close()
	}
	a.connMu.Unlock()
}

// sendJSON marshals a message to JSON and sends it to the server.
// If no connection is active, the message is silently dropped.
func (a *Agent) sendJSON(msg map[string]any) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Warn("Failed to marshal message", "error", err)
		return
	}

	a.connMu.RLock()
	conn := a.conn
	a.connMu.RUnlock()

	if conn == nil {
		return
	}

	conn.WriteMessage(websocket.TextMessage, data)
}

// sendBinary sends raw binary data to the server.
// If no connection is active, the data is silently dropped.
func (a *Agent) sendBinary(data []byte) {
	a.connMu.RLock()
	conn := a.conn
	a.connMu.RUnlock()

	if conn == nil {
		return
	}

	conn.WriteMessage(websocket.BinaryMessage, data)
}

// Broadcast sends a raw message to the server (which broadcasts to browsers).
// In agent mode, this is how data-updates and other broadcasts reach clients.
func (a *Agent) Broadcast(msg []byte) {
	a.sendBinary(msg)
}

// BroadcastJSON marshals a message to JSON and sends it to the server.
func (a *Agent) BroadcastJSON(msg map[string]any) {
	a.sendJSON(msg)
}

// RegisterClient adds a channel to the broadcast set (for internal use).
func (a *Agent) RegisterClient(ch chan []byte) uint64 {
	a.broadcastMu.Lock()
	defer a.broadcastMu.Unlock()
	a.nextClientID++
	clientID := a.nextClientID
	a.clients[ch] = struct{}{}
	a.clientIDs[ch] = clientID
	return clientID
}

// UnregisterClient removes a channel from the broadcast set.
func (a *Agent) UnregisterClient(ch chan []byte) {
	a.broadcastMu.Lock()
	defer a.broadcastMu.Unlock()
	delete(a.clients, ch)
	delete(a.clientIDs, ch)
	close(ch)
}

// LoadPanelJSON reads a panel definition from user/panels/<name>.json.
func (a *Agent) LoadPanelJSON(panelName string) ([]byte, error) {
	path := filepath.Join(a.UserPath, "panels", panelName+".json")
	return os.ReadFile(path)
}

// StartDataBroadcast starts background goroutines for metrics and snapshots.
func (a *Agent) StartDataBroadcast() {
	a.DataBus.StartMetrics(500 * time.Millisecond)

	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for range ticker.C {
			snapshot := a.DataBus.Snapshot()
			a.BroadcastJSON(map[string]any{
				"type": "data-update",
				"data": snapshot,
			})
		}
	}()
	slog.Info("Data broadcast started")
}

// PushData adds a custom metric to the DataBus.
func (a *Agent) PushData(key string, value any, unit string) {
	a.DataBus.Set(key, value, unit)
}

// GetConfig returns the current config.
func (a *Agent) GetConfig() *config.Config {
	return a.Config
}

// UpdateJoystickCount changes the number of virtual joysticks at runtime.
func (a *Agent) UpdateJoystickCount(count uint8) {
	a.Config.NumJoysticks = count
	a.JoystickManager.Reload(count)
	slog.Info("Joystick count updated", "count", count)
	_ = a.Config.Save(a.ConfigPath)
}

// Close cleans up all subsystems.
func (a *Agent) Close() {
	slog.Info("Closing joystick manager...")
	a.JoystickManager.Close()
	slog.Info("Closing mousepad manager...")
	a.MousepadManager.Close()
	slog.Info("Closing keyboard manager...")
	a.KeyboardManager.Close()
	slog.Info("Closing speech manager...")
	a.SpeechManager.Close()
	slog.Info("Closing MPRIS watcher...")
	if a.MPRISWatcher != nil {
		a.MPRISWatcher.Close()
	}
	slog.Info("Closing RSS manager...")
	a.RSSManager.Close()
	slog.Info("All subsystems closed")
}

// WaitSignal blocks until SIGINT or SIGTERM is received.
func (a *Agent) WaitSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// GetJoystickManager returns the joystick manager.
func (a *Agent) GetJoystickManager() *devices.JoystickManager {
	return a.JoystickManager
}

// GetMousepadManager returns the mousepad manager.
func (a *Agent) GetMousepadManager() *devices.MousepadManager {
	return a.MousepadManager
}

// GetKeyboardManager returns the keyboard manager.
func (a *Agent) GetKeyboardManager() *devices.KeyboardManager {
	return a.KeyboardManager
}

// GetDataBus returns the databus.
func (a *Agent) GetDataBus() *databus.DataBus {
	return a.DataBus
}

// GetSpeechManager returns the speech manager.
func (a *Agent) GetSpeechManager() *speech.SpeechManager {
	return a.SpeechManager
}

// GetMPRISWatcher returns the media player watcher instance.
// On Linux this uses MPRIS; on Windows it uses SMTC.
func (a *Agent) GetMPRISWatcher() *mediacontrol.Watcher {
	return a.MPRISWatcher
}

// GetRSSManager returns the RSS manager.
func (a *Agent) GetRSSManager() *rssfeed.Manager {
	return a.RSSManager
}

// handleMPRISRequest processes an "mpris-request" message forwarded from the
// relay server. It parses the endpoint, method, body, and query parameters,
// dispatches to the appropriate local MPRIS watcher method, and sends an
// "mpris-response" message back with the result. This enables the relay server
// to proxy MPRIS HTTP endpoints to the host agent in distributed deployments.
func (a *Agent) handleMPRISRequest(raw string) {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		slog.Warn("Failed to parse mpris-request", "error", err)
		return
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(parsed["data"], &data); err != nil {
		slog.Warn("Failed to parse mpris-request data", "error", err)
		return
	}

	var requestID, endpoint, method string
	json.Unmarshal(data["request_id"], &requestID)
	json.Unmarshal(data["endpoint"], &endpoint)
	json.Unmarshal(data["method"], &method)

	var response map[string]any

	switch endpoint {
	case "/players":
		response = a.handleMPRISListPlayers()
	case "/control":
		var body map[string]any
		if raw, ok := data["body"]; ok {
			json.Unmarshal(raw, &body)
		}
		response = a.handleMPRISControl(body)
	case "/select":
		var body map[string]any
		if raw, ok := data["body"]; ok {
			json.Unmarshal(raw, &body)
		}
		response = a.handleMPRISSelect(body)
	case "/cover":
		var query map[string]string
		if raw, ok := data["query"]; ok {
			json.Unmarshal(raw, &query)
		}
		response = a.handleMPRISCover(query)
	default:
		response = map[string]any{
			"error": "Unknown endpoint: " + endpoint,
		}
	}

	respMsg := map[string]any{
		"type": "mpris-response",
		"data": map[string]any{
			"request_id": requestID,
			"payload":    response,
		},
	}

	a.sendJSON(respMsg)
}

// handleMPRISListPlayers returns the list of connected MPRIS media players
// with their current state (identity, playback status, metadata, capabilities).
// Also includes the current system volume for accurate volume button calculations.
// Called by handleMPRISRequest for the "/players" endpoint.
func (a *Agent) handleMPRISListPlayers() map[string]any {
	sysVol, _ := getSystemVolume()
	return mediautil.BuildPlayersResponse(a.MPRISWatcher, sysVol)
}

// handleMPRISControl executes a playback command (play, pause, playpause,
// stop, next, previous, volume) on the selected media session.
// If no player is specified, it uses the currently selected player.
// Called by handleMPRISRequest for the "/control" endpoint.
func (a *Agent) handleMPRISControl(body map[string]any) map[string]any {
	if a.MPRISWatcher == nil {
		return map[string]any{
			"error": "Media integration is not enabled",
		}
	}

	action, _ := body["action"].(string)
	player, _ := body["player"].(string)
	slog.Info("mpris control (agent)", "player", player, "action", action, "volume", body["volume"])
	volume, _ := body["volume"].(float64)
	player, err := mediautil.ExecuteControl(a.MPRISWatcher, mediautil.ControlRequest{
		Player: player,
		Action: action,
		Volume: volume,
	}, setSystemVolume)
	if err != nil {
		return map[string]any{
			"error": err.Error(),
		}
	}

	return map[string]any{
		"success": true,
		"player":  player,
		"action":  action,
	}
}

// handleMPRISSelect changes the active player whose state is published
// to the DataBus. After selection, the new player's metadata is immediately
// fetched and broadcast. Called by handleMPRISRequest for the "/select" endpoint.
func (a *Agent) handleMPRISSelect(body map[string]any) map[string]any {
	if a.MPRISWatcher == nil {
		return map[string]any{
			"error": "Media integration is not enabled",
		}
	}

	player, _ := body["player"].(string)
	if err := mediautil.SelectPlayer(a.MPRISWatcher, player); err != nil {
		return map[string]any{
			"error": err.Error(),
		}
	}

	return map[string]any{
		"success": true,
		"player":  player,
	}
}

// handleMPRISCover reads a local cover art file and returns it as a base64-encoded
// string with the correct MIME type. File access is restricted to trusted temp/cache
// directories (/tmp/, /var/tmp/, ~/.cache/, and os.TempDir()) for security.
// Browsers cannot load file:// URLs directly,
// so the relay server decodes this and serves it over HTTP.
// Called by handleMPRISRequest for the "/cover" endpoint.
func (a *Agent) handleMPRISCover(query map[string]string) map[string]any {
	filePath, _ := query["url"]
	if filePath == "" {
		return map[string]any{
			"error": "Missing url parameter",
		}
	}

	filePath = mediautil.NormalizeFileURLPath(filePath)

	cleanPath := filepath.Clean(filePath)
	allowedPrefixes := mediautil.AllowedCoverPrefixes(os.Getenv("HOME"), os.TempDir())
	if !mediautil.IsAllowedCoverPath(cleanPath, allowedPrefixes) {
		return map[string]any{
			"error": "Access denied",
		}
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return map[string]any{
			"error": "Cover art not found",
		}
	}

	contentType := mediautil.ContentTypeForPath(cleanPath)

	encoded := base64.StdEncoding.EncodeToString(data)

	return map[string]any{
		"content_type": contentType,
		"data":         encoded,
	}
}

// setSystemVolume sets the system-wide volume via pactl (PulseAudio/PipeWire).
// The volume parameter is a float between 0.0 (muted) and 1.0 (max).
func setSystemVolume(volume float64) error {
	pct := int(volume * 100)
	cmd := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", strconv.Itoa(pct)+"%")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("mpris: system volume control failed", "error", err, "output", string(output))
		return err
	}
	slog.Debug("mpris: system volume set", "volume", volume, "percent", pct)
	return nil
}

// getSystemVolume reads the current system-wide volume via pactl (PulseAudio/PipeWire).
// Returns the volume as a float between 0.0 and 1.0. Falls back to 0.5 on error.
func getSystemVolume() (float64, error) {
	cmd := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("mpris: get system volume failed", "error", err, "output", string(output))
		return 0.5, err
	}
	s := string(output)
	idx := strings.Index(s, "%")
	if idx == -1 {
		return 0.5, fmt.Errorf("mpris: unexpected pactl output format: %s", s)
	}
	start := idx
	for start > 0 && (s[start-1] == ' ' || (s[start-1] >= '0' && s[start-1] <= '9')) {
		start--
	}
	pctStr := strings.TrimSpace(s[start:idx])
	pct, err := strconv.Atoi(pctStr)
	if err != nil {
		return 0.5, fmt.Errorf("mpris: failed to parse volume percentage: %w", err)
	}
	return float64(pct) / 100.0, nil
}
