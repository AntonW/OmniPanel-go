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
// to handle auth detection and redirect. HTML pages (/panel, /editor) are also
// served without auth — they are templates only; sensitive data is protected by
// requiring authentication on all API endpoints and WebSocket connections.
//
// Architecture:
//
//	Fiber HTTP server serves static files and API routes (same as default mode)
//	WebSocket at /ws accepts two connection types:
//	  - Browser: no query param, multiple allowed
//	  - Host: ?type=host query param, exactly one allowed (1:1)
//	Messages are relayed bidirectionally between the host and all browsers
//
// Connection flow:
//
//  1. Browser connects to ws://server/ws → added to browser pool
//  2. Host connects to ws://server/ws?type=host → accepted if no host exists
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
//	./omnipanel connect <server-ip>:<port>
package relay

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	ws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/auth"
	"omnipanel-go/internal/config"
)

// RelayServer holds the central server state for serve mode.
// It manages the Fiber HTTP app, a single host WebSocket connection,
// and multiple browser WebSocket connections. Messages are relayed
// bidirectionally between the host and all browsers.
type RelayServer struct {
	app       *fiber.App
	hostConn  *ws.Conn
	hostMu    sync.Mutex
	browsers  map[*ws.Conn]struct{}
	browserMu sync.Mutex
	staticDir string
	userPath  string
	config    *config.Config
}

// New creates a new relay server with all HTTP routes and WebSocket hub initialized.
// The server serves static files, API endpoints, and manages browser/host connections.
// A /health endpoint is registered before authentication middleware to allow
// unauthenticated health checks for Kubernetes liveness and readiness probes.
func New(cfg *config.Config, userPath, baseDir string) *RelayServer {
	staticDir := filepath.Join(baseDir, "static")

	s := &RelayServer{
		browsers:  make(map[*ws.Conn]struct{}),
		staticDir: staticDir,
		userPath:  userPath,
		config:    cfg,
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
	// - CSS files: needed for login page and unauthenticated pages
	// - JS files: application code, auth enforced via API calls
	// - Images/fonts: referenced by HTML/CSS
	// The Next function skips paths that should fall through to HTML handlers.
	app.Static("/", staticDir, fiber.Static{
		Next: func(c *fiber.Ctx) bool {
			// Let HTML page handlers serve these paths
			return c.Path() == "/" || c.Path() == "/panel" || c.Path() == "/editor"
		},
	})

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

	blocksPath := filepath.Join(userPath, "blocks")
	if _, err := os.Stat(blocksPath); err == nil {
		app.Static("/blocks", blocksPath)
	}

	themesPath := filepath.Join(userPath, "themes")
	if _, err := os.Stat(themesPath); err == nil {
		app.Static("/themes", themesPath)
	}

	assetsPath := filepath.Join(userPath, "assets")
	if _, err := os.Stat(assetsPath); err == nil {
		app.Static("/assets", assetsPath)
	}

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
