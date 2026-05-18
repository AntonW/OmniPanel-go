// Route registration and directory tree builder.
// See editor.go for the full package documentation.
package routes

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/state"
	"omnipanel-go/internal/websocket"
)

// NewRouter builds the Fiber HTTP server with all routes registered.
// It sets up middleware to share AppState with all handlers via c.Locals.
func NewRouter(s *state.AppState) *fiber.App {
	app := fiber.New()

	// Middleware: store AppState in request context for all handlers
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("state", s)
		return c.Next()
	})

	// UI pages (friendly URLs)
	app.Get("/", serveStartPage)
	app.Get("/panel", servePanel)
	app.Get("/editor", serveEditorUI)

	// Static files (JS, CSS, assets served directly from static/ directory)
	app.Static("/", s.StaticDir)

	app.Get("/ws", ws.New(func(c *ws.Conn) {
		handleWS(c, s)
	}))

	// REST API endpoints
	app.Get("/api/blocks", getBlocks)
	app.Get("/api/themes", getThemes)
	app.Get("/api/panels", getPanels)
	app.Post("/api/panel/save", savePanel)
	app.Get("/api/panel/load", loadPanel)
	app.Get("/api/panel/content", getPanelContent)
	app.Delete("/api/panel/delete", deletePanel)
	app.Post("/api/joystick-count", setJoystickCount)
	app.Get("/api/config", getConfig)
	app.Post("/api/data/push", pushData)

	// MPRIS media control endpoints
	app.Get("/api/mpris/players", listMPRISPlayers)
	app.Post("/api/mpris/control", controlMPRIS)
	app.Post("/api/mpris/select", selectMPRISPlayer)
	app.Get("/api/mpris/cover", serveMPRISCoverArt)

	// Static file directories (only if they exist)
	blocksPath := filepath.Join(s.UserPath, "blocks")
	if _, err := os.Stat(blocksPath); err == nil {
		app.Static("/blocks", blocksPath)
	}

	themesPath := filepath.Join(s.UserPath, "themes")
	if _, err := os.Stat(themesPath); err == nil {
		app.Static("/themes", themesPath)
	}

	assetsPath := filepath.Join(s.UserPath, "assets")
	if _, err := os.Stat(assetsPath); err == nil {
		app.Static("/assets", assetsPath)
	}

	return app
}

// handleWS manages a single WebSocket connection lifecycle:
// 1. Register a broadcast channel and assign a unique client ID
// 2. Register the client with the RSS manager for per-client seen-entry tracking
// 3. Launch a write goroutine (reads from channel, writes to socket)
// 4. Read loop (reads from socket, dispatches to HandleMessage with clientID)
// 5. Cleanup on disconnect (unregister from RSS manager, unregister channel, close it)
func handleWS(c *ws.Conn, s *state.AppState) {
	ch := make(chan []byte, 256)
	clientID := s.RegisterClient(ch)

	if s.RSSManager != nil {
		s.RSSManager.RegisterClient(clientID)
	}

	clientIP := c.IP()
	s.BroadcastJSON(map[string]any{
		"type":      "log-event",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"data":      "Client connected: " + clientIP,
	})

	if s.Config != nil {
		s.BroadcastJSON(map[string]any{
			"type": "speech-config-init",
			"data": map[string]any{
				"enabled":           s.Config.Speech.Enabled,
				"recordingLocation": s.Config.Speech.RecordingLoc,
				"triggerMode":       s.Config.Speech.TriggerMode,
				"wakeWord":          s.Config.Speech.WakeWord,
				"wakeWordListenSec": s.Config.Speech.WakeWordListenSec,
				"ttsEnabled":        s.Config.Speech.TTSEnabled,
			},
		})
	}

	defer func() {
		if s.RSSManager != nil {
			s.RSSManager.UnregisterClient(clientID)
		}
		s.UnregisterClient(ch)
		s.BroadcastJSON(map[string]any{
			"type":      "log-event",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			"data":      "Client disconnected: " + clientIP,
		})
	}()

	// Write goroutine: reads from channel, writes to WebSocket
	go func() {
		for msg := range ch {
			if err := c.WriteMessage(ws.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Read loop: reads from WebSocket, dispatches to message handler
	for {
		msgType, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		if msgType == ws.BinaryMessage {
			websocket.HandleAudioChunk(s, msg)
		} else {
			websocket.HandleMessage(s, clientID, string(msg))
		}
	}

	slog.Info("WebSocket disconnected")
}

// serveFile reads a file from disk and sends it with the correct Content-Type.
// Returns 404 if the file doesn't exist.
func serveFile(c *fiber.Ctx, path string, contentType string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Failed to serve file", "path", path, "error", err)
		return c.Status(fiber.StatusNotFound).SendString("Not found")
	}
	c.Set("Content-Type", contentType)
	return c.Send(data)
}

// BlockNode represents a file or folder in the block library tree.
type BlockNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Path     string      `json:"path"`
	Children []BlockNode `json:"children,omitempty"`
}

// buildDirectoryTree recursively walks a directory and builds a nested JSON tree.
// Only includes .html files (block templates) and directories.
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

// listPanels returns sorted names of all .json files in the panels directory.
// Strips the .json extension from each name.
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
