// Package relay implements the central server mode for distributed deployment.
//
// The relay server serves the WebUI (static files, API routes) and acts as a
// WebSocket hub connecting browser clients to a single host agent. This enables
// deploying the WebUI on a central server (e.g., always-on machine or cloud)
// while the host agent runs on a machine behind a firewall that initiates an
// outbound WebSocket connection to the server.
//
// Authentication:
//
// All HTTP routes are protected by token-based authentication when auth_token
// is configured. The token is accepted via query parameter (?token=xxx) or
// Authorization header (Bearer xxx). Host WebSocket connections validate the
// token from the query parameter (?type=host&token=xxx). If auth_token is
// empty, authentication is disabled (backward compatible).
// The /health endpoint is exempt from authentication for Kubernetes probes.
// The /login page is exempt from authentication, providing a login form where
// users can enter their token. After successful validation, the token is stored
// in the browser (localStorage or sessionStorage) and included in all subsequent
// requests via Authorization headers and WebSocket URL query parameters.
// CSS files, JavaScript, images, and fonts are served without authentication
// so the login page can load styles and all pages can execute their JavaScript
// to handle auth detection and redirect. Block templates (/blocks/), theme CSS
// (/themes/), and user assets (/assets/) are also served without auth — they
// contain no sensitive data, only static files referenced by the frontend.
// HTML pages (/panel, /editor) are served without auth as templates only;
// sensitive data is protected by requiring authentication on all API endpoints
// and WebSocket connections.
//
// Architecture:
//
//	Fiber HTTP server serves static files and API routes (same as default mode)
//	WebSocket at /ws accepts two connection types (ws:// or wss://):
//	  - Browser: no query param, multiple allowed
//	  - Host: ?type=host query param, exactly one allowed (1:1)
//	Messages are relayed bidirectionally between the host and all browsers
//
// Connection flow:
//
//  1. Browser connects to ws://server/ws (or wss://) → added to browser pool
//  2. Host connects to ws://server/ws?type=host (or wss://) → accepted if no host exists
//  3. Browser messages forwarded to host
//  4. Host messages broadcast to all browsers
//
// Usage:
//
//	./omnipanel serve          # Start relay server on configured port
//	./omnipanel serve --port 8080  # Override port
//
// The host agent connects using:
//
//	./omnipanel connect <server-ip>:<port>        # ws:// (default)
//	./omnipanel connect wss://<server-ip>:<port>  # wss:// (secure, behind TLS proxy)
package relay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	ws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/auth"
	"omnipanel-go/internal/config"
)

// RelayServer holds the central server state for serve mode.
// It manages the Fiber HTTP app, a single host WebSocket connection,
// and multiple browser WebSocket connections. Messages are relayed
// bidirectionally between the host and all browsers.
//
// For MPRIS media player control in distributed deployments, the relay server
// forwards HTTP API requests to the host agent via WebSocket using a request-response
// pattern. Each request is assigned a unique ID, sent as an "mpris-request" message,
// and the response arrives as an "mpris-response" message that completes the pending
// request channel. Cover art is base64-encoded for transport over WebSocket.
type RelayServer struct {
	app       *fiber.App
	hostConn  *ws.Conn
	hostMu    sync.Mutex
	browsers  map[*ws.Conn]struct{}
	browserMu sync.Mutex
	staticDir string
	userPath  string
	config    *config.Config

	// pendingRequests tracks in-flight MPRIS requests forwarded to the host agent.
	// Each entry maps a unique request ID to a buffered channel that receives
	// the response payload when the host agent replies.
	pendingRequests   map[string]chan map[string]any
	pendingRequestsMu sync.Mutex
}

// New creates a new relay server with all HTTP routes and WebSocket hub initialized.
// The server serves static files, API endpoints, and manages browser/host connections.
// A /health endpoint is registered before authentication middleware to allow
// unauthenticated health checks for Kubernetes liveness and readiness probes.
func New(cfg *config.Config, userPath, baseDir string) *RelayServer {
	staticDir := filepath.Join(baseDir, "static")

	s := &RelayServer{
		browsers:        make(map[*ws.Conn]struct{}),
		pendingRequests: make(map[string]chan map[string]any),
		staticDir:       staticDir,
		userPath:        userPath,
		config:          cfg,
	}

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("relay", s)
		return c.Next()
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/login", s.serveLoginPage)

	// Redirect favicon.ico requests to the SVG logo so browsers display
	// the OmniPanel icon in the tab. The static directory is mounted at /,
	// so the file is served at /omnipanel-go-logo.svg. All HTML pages also
	// include a <link rel="icon"> for direct favicon discovery.
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.Redirect("/omnipanel-go-logo.svg", fiber.StatusMovedPermanently)
	})

	// Disable browser caching for JS/CSS so changes are always picked up.
	// This runs before static serving for unauthenticated assets.
	noCacheStaticMiddleware := func(c *fiber.Ctx) error {
		if strings.HasSuffix(c.Path(), ".js") || strings.HasSuffix(c.Path(), ".css") {
			c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		return c.Next()
	}
	app.Use(noCacheStaticMiddleware)

	// Serve static assets without authentication:
	// - CSS, JS, images, fonts from static/ directory
	// - Block templates from user/blocks/
	// - Theme CSS from user/themes/
	// - User assets from user/assets/
	// These are referenced by HTML/CSS and contain no sensitive data.
	// The Next function skips paths that should fall through to HTML handlers.
	app.Static("/", staticDir, fiber.Static{
		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/" || c.Path() == "/panel" || c.Path() == "/editor"
		},
	})

	blocksPath := filepath.Join(userPath, "blocks")
	if _, err := os.Stat(blocksPath); err == nil {
		app.Static("/blocks", blocksPath)
	}

	themesPath := filepath.Join(userPath, "themes")
	if _, err := os.Stat(themesPath); err == nil {
		app.Static("/themes", themesPath)
	}

	// User assets directory (images for block/workspace backgrounds).
	// Browse: true generates an HTML directory listing so the frontend
	// can parse available image filenames from the response. This route
	// is registered before auth middleware — assets contain no sensitive data.
	assetsPath := filepath.Join(userPath, "assets")
	if _, err := os.Stat(assetsPath); err == nil {
		app.Static("/assets", assetsPath, fiber.Static{
			Browse: true,
		})
	}

	app.Get("/", s.serveStartPage)
	app.Get("/panel", s.servePanel)
	app.Get("/editor", s.serveEditorUI)

	app.Use(auth.Middleware(cfg.AuthToken))

	app.Get("/ws", ws.New(func(c *ws.Conn) {
		s.handleWS(c)
	}))

	app.Get("/api/blocks", s.getBlocks)
	app.Get("/api/themes", s.getThemes)
	app.Get("/api/panels", s.getPanels)
	app.Post("/api/panel/save", s.savePanel)
	app.Get("/api/panel/load", s.loadPanel)
	app.Get("/api/panel/content", s.getPanelContent)
	app.Delete("/api/panel/delete", s.deletePanel)
	app.Post("/api/joystick-count", s.setJoystickCount)
	app.Get("/api/config", s.getConfig)
	app.Post("/api/data/push", s.pushData)

	// Media control endpoints (forwarded to host agent)
	app.Get("/api/media/players", s.listMPRISPlayers)
	app.Post("/api/media/control", s.controlMPRIS)
	app.Post("/api/media/select", s.selectMPRISPlayer)
	app.Get("/api/media/cover", s.serveMPRISCoverArt)

	s.app = app
	return s
}

// Listen starts the relay server on the given address and blocks until the server stops.
func (s *RelayServer) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the relay server with the given context timeout.
func (s *RelayServer) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

func (s *RelayServer) serveFile(c *fiber.Ctx, path string, contentType string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Failed to serve file", "path", path, "error", err)
		return c.Status(fiber.StatusNotFound).SendString("Not found")
	}
	c.Set("Content-Type", contentType)
	return c.Send(data)
}

func (s *RelayServer) serveStartPage(c *fiber.Ctx) error {
	path := filepath.Join(s.staticDir, "index.html")
	return s.serveFile(c, path, "text/html; charset=utf-8")
}

// serveLoginPage serves the login page (static/login.html).
// This route is registered before auth middleware so it is always accessible,
// allowing users to enter their token when authentication is enabled.
func (s *RelayServer) serveLoginPage(c *fiber.Ctx) error {
	path := filepath.Join(s.staticDir, "login.html")
	return s.serveFile(c, path, "text/html; charset=utf-8")
}

func (s *RelayServer) servePanel(c *fiber.Ctx) error {
	path := filepath.Join(s.staticDir, "client", "index.html")
	return s.serveFile(c, path, "text/html; charset=utf-8")
}

func (s *RelayServer) serveEditorUI(c *fiber.Ctx) error {
	path := filepath.Join(s.staticDir, "editor", "editor.html")
	return s.serveFile(c, path, "text/html; charset=utf-8")
}

func (s *RelayServer) getBlocks(c *fiber.Ctx) error {
	blocksPath := filepath.Join(s.userPath, "blocks")
	tree := buildDirectoryTree(blocksPath)
	if tree == nil {
		tree = []BlockNode{}
	}
	return c.JSON(tree)
}

func (s *RelayServer) getPanels(c *fiber.Ctx) error {
	panelsPath := filepath.Join(s.userPath, "panels")
	allPanels := listPanels(panelsPath)
	return c.JSON(map[string]any{
		"allPanels": allPanels,
	})
}

func (s *RelayServer) getConfig(c *fiber.Ctx) error {
	return c.JSON(map[string]any{
		"port":         s.config.Port,
		"numJoysticks": s.config.NumJoysticks,
	})
}

func (s *RelayServer) savePanel(c *fiber.Ctx) error {
	panelsPath := filepath.Join(s.userPath, "panels")

	var data map[string]any
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	fileName, _ := data["fileName"].(string)
	if fileName == "" {
		fileName = "new_panel"
	}

	content := data["content"]
	filePath := filepath.Join(panelsPath, fileName+".json")

	jsonData, err := json.MarshalIndent(content, "", "    ")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	if err := writeFile(filePath, jsonData); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	return c.JSON(map[string]any{"success": true, "path": filePath})
}

func (s *RelayServer) loadPanel(c *fiber.Ctx) error {
	name := c.Query("name", "")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing panel name")
	}

	path := filepath.Join(s.userPath, "panels", name+".json")
	panelJSON, err := os.ReadFile(path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read panel")
	}

	return c.Type("application/json").Send(panelJSON)
}

func (s *RelayServer) getPanelContent(c *fiber.Ctx) error {
	name := c.Query("name", "")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing panel name")
	}

	path := filepath.Join(s.userPath, "panels", name+".json")
	panelJSON, err := os.ReadFile(path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read panel")
	}

	return c.Type("application/json").Send(panelJSON)
}

func (s *RelayServer) deletePanel(c *fiber.Ctx) error {
	name := c.Query("name", "")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": "Missing panel name"})
	}

	path := filepath.Join(s.userPath, "panels", name+".json")
	if err := os.Remove(path); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	return c.JSON(map[string]any{"success": true})
}

func (s *RelayServer) setJoystickCount(c *fiber.Ctx) error {
	var data map[string]any
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	countFloat, _ := data["count"].(float64)
	count := uint8(countFloat)

	if s.hostConn != nil {
		msg := map[string]any{
			"type": "save-joystick-count",
			"data": count,
		}
		if data, err := json.Marshal(msg); err == nil {
			s.hostConn.WriteMessage(ws.TextMessage, data)
		}
	}

	return c.JSON(map[string]any{"success": true, "count": count})
}

func (s *RelayServer) pushData(c *fiber.Ctx) error {
	var data map[string]any
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	key, _ := data["key"].(string)
	if key == "" {
		return c.JSON(map[string]any{"success": false, "error": "Missing key"})
	}

	if s.hostConn != nil {
		msg := map[string]any{
			"type": "push-data",
			"data": data,
		}
		if data, err := json.Marshal(msg); err == nil {
			s.hostConn.WriteMessage(ws.TextMessage, data)
		}
	}

	return c.JSON(map[string]any{"success": true, "key": key})
}

func (s *RelayServer) getThemes(c *fiber.Ctx) error {
	themesPath := filepath.Join(s.userPath, "themes")

	entries, err := os.ReadDir(themesPath)
	if err != nil {
		return c.JSON([]string{})
	}

	var themes []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".css" {
			themes = append(themes, entry.Name()[:len(entry.Name())-4])
		}
	}

	return c.JSON(themes)
}

type BlockNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Path     string      `json:"path"`
	Children []BlockNode `json:"children,omitempty"`
}

func buildDirectoryTree(path string) []BlockNode {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var result []BlockNode
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			children := buildDirectoryTree(fullPath)
			result = append(result, BlockNode{
				Name:     entry.Name(),
				Type:     "folder",
				Path:     fullPath,
				Children: children,
			})
		} else if strings.HasSuffix(entry.Name(), ".html") {
			result = append(result, BlockNode{
				Name: entry.Name(),
				Type: "file",
				Path: fullPath,
			})
		}
	}
	return result
}

func listPanels(panelsPath string) []string {
	entries, err := os.ReadDir(panelsPath)
	if err != nil {
		return nil
	}

	var panels []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			panels = append(panels, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(panels)
	return panels
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// registerPendingRequest creates a buffered channel for an MPRIS request-response
// pair and stores it in the pendingRequests map under the given requestID.
// The caller should read from the returned channel to receive the response.
func (s *RelayServer) registerPendingRequest(requestID string) chan map[string]any {
	ch := make(chan map[string]any, 1)
	s.pendingRequestsMu.Lock()
	s.pendingRequests[requestID] = ch
	s.pendingRequestsMu.Unlock()
	return ch
}

// completePendingRequest delivers the response payload to the waiting channel
// for the given requestID and removes the entry from pendingRequests.
// If no pending request exists for the ID, the response is silently dropped.
func (s *RelayServer) completePendingRequest(requestID string, response map[string]any) {
	s.pendingRequestsMu.Lock()
	ch, exists := s.pendingRequests[requestID]
	if exists {
		delete(s.pendingRequests, requestID)
	}
	s.pendingRequestsMu.Unlock()

	if exists {
		select {
		case ch <- response:
		default:
		}
	}
}

// cleanupPendingRequest removes a pending request entry without delivering a response.
// Called when a request times out or fails before the host agent replies.
func (s *RelayServer) cleanupPendingRequest(requestID string) {
	s.pendingRequestsMu.Lock()
	delete(s.pendingRequests, requestID)
	s.pendingRequestsMu.Unlock()
}

// generateRequestID creates a unique identifier for MPRIS request-response pairs.
// Uses nanosecond-precision timestamps to ensure uniqueness across concurrent requests.
func generateRequestID() string {
	return fmt.Sprintf("mpris-%d", time.Now().UnixNano())
}

// sendMPRISRequest forwards an MPRIS API request to the host agent via WebSocket
// and waits for the response. It generates a unique request ID, sends an
// "mpris-request" message, and blocks until the matching "mpris-response" arrives
// or the 5-second timeout expires. Returns the response payload or an error.
//
// This enables the relay server to proxy MPRIS HTTP endpoints to the host agent
// in distributed deployments (serve + connect mode). The host agent processes
// the request locally against the D-Bus session and sends back the result.
func (s *RelayServer) sendMPRISRequest(endpoint, method string, body map[string]any, query map[string]string) (map[string]any, error) {
	requestID := generateRequestID()
	respCh := s.registerPendingRequest(requestID)

	msg := map[string]any{
		"type": "mpris-request",
		"data": map[string]any{
			"request_id": requestID,
			"endpoint":   endpoint,
			"method":     method,
		},
	}

	if body != nil {
		msg["data"].(map[string]any)["body"] = body
	}
	if query != nil {
		msg["data"].(map[string]any)["query"] = query
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		s.cleanupPendingRequest(requestID)
		return nil, err
	}

	s.hostMu.Lock()
	if s.hostConn == nil {
		s.hostMu.Unlock()
		s.cleanupPendingRequest(requestID)
		return nil, fmt.Errorf("no host connected")
	}
	if err := s.hostConn.WriteMessage(ws.TextMessage, msgData); err != nil {
		s.hostMu.Unlock()
		s.cleanupPendingRequest(requestID)
		return nil, err
	}
	s.hostMu.Unlock()

	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(5 * time.Second):
		s.cleanupPendingRequest(requestID)
		return nil, fmt.Errorf("timeout waiting for MPRIS response")
	}
}

// listMPRISPlayers handles GET /api/media/players by forwarding the request
// to the host agent and returning the list of connected media players.
// Returns 503 if no host is connected or the request times out.
func (s *RelayServer) listMPRISPlayers(c *fiber.Ctx) error {
	resp, err := s.sendMPRISRequest("/players", "GET", nil, nil)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error":   "Media integration is not available",
			"enabled": false,
			"players": []string{},
		})
	}
	return c.JSON(resp)
}

// controlMPRIS handles POST /api/media/control by forwarding playback commands
// (play, pause, next, previous, stop, volume) to the host agent. Parses the
// request body for player name, action, and optional volume value. Maps error
// messages from the host to appropriate HTTP status codes.
func (s *RelayServer) controlMPRIS(c *fiber.Ctx) error {
	var req struct {
		Player string  `json:"player"`
		Action string  `json:"action"`
		Volume float64 `json:"volume,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Invalid request body",
		})
	}

	body := map[string]any{
		"player": req.Player,
		"action": req.Action,
	}
	if req.Action == "volume" {
		body["volume"] = req.Volume
	}

	resp, err := s.sendMPRISRequest("/control", "POST", body, nil)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "Media integration is not available",
		})
	}

	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		status := fiber.StatusInternalServerError
		if strings.Contains(errMsg, "not found") {
			status = fiber.StatusNotFound
		} else if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "Unknown") {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(resp)
	}

	return c.JSON(resp)
}

// selectMPRISPlayer handles POST /api/media/select by forwarding the player
// selection request to the host agent. The selected player's state is then
// published to the DataBus and broadcast to all browsers via WebSocket.
func (s *RelayServer) selectMPRISPlayer(c *fiber.Ctx) error {
	var req struct {
		Player string `json:"player"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Invalid request body",
		})
	}

	if req.Player == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Player name is required",
		})
	}

	body := map[string]any{
		"player": req.Player,
	}

	resp, err := s.sendMPRISRequest("/select", "POST", body, nil)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "Media integration is not available",
		})
	}

	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		status := fiber.StatusInternalServerError
		if strings.Contains(errMsg, "not found") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(resp)
	}

	return c.JSON(resp)
}

// serveMPRISCoverArt handles GET /api/media/cover by forwarding the cover art
// request to the host agent. The agent reads the local file (restricted to
// /tmp/, /var/tmp/, and ~/.cache/ for security), base64-encodes it, and sends
// it back over WebSocket. The relay server decodes and serves it with the
// correct Content-Type header. Browsers cannot load file:// URLs directly,
// so this proxy endpoint is required for cover art display.
func (s *RelayServer) serveMPRISCoverArt(c *fiber.Ctx) error {
	fileURL := c.Query("url")
	if fileURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Missing url parameter",
		})
	}

	query := map[string]string{
		"url": fileURL,
	}

	resp, err := s.sendMPRISRequest("/cover", "GET", nil, query)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "Media integration is not available",
		})
	}

	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		status := fiber.StatusInternalServerError
		if strings.Contains(errMsg, "denied") {
			status = fiber.StatusForbidden
		} else if strings.Contains(errMsg, "not found") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(resp)
	}

	contentType, _ := resp["content_type"].(string)
	dataStr, _ := resp["data"].(string)

	if dataStr == "" {
		return c.Status(fiber.StatusNotFound).JSON(map[string]any{
			"error": "Cover art not found",
		})
	}

	data, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"error": "Failed to decode cover art",
		})
	}

	if contentType != "" {
		c.Set("Content-Type", contentType)
	}
	c.Set("Cache-Control", "public, max-age=60")
	return c.Send(data)
}
