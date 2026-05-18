// Package routes provides HTTP handlers for the panel editor and REST API.
//
// The editor serves a modular JavaScript application (static/editor/) that
// replaces the legacy single-file editor-renderer.js with 12 separate modules.
// Panel definitions use a v2 JSON format with metadata, grid config, background
// image settings, and theme configuration.
//
// Block themes are stored as CSS files in user/themes/ and applied dynamically
// per block. Each block references a theme name (e.g., "star_citizen") and the
// editor/client loads the corresponding CSS with selectors scoped to that block.
package routes

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/state"
)

// serveEditorUI serves the panel editor HTML page at /editor.
// The editor is a modular JavaScript application with a three-panel layout:
// block library (left), workspace canvas (center), and properties/layers/bindings/speech tabs (right).
func serveEditorUI(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	path := filepath.Join(s.StaticDir, "editor", "editor.html")
	return serveFile(c, path, "text/html; charset=utf-8")
}

// getBlocks returns the block library as a flat list of .html files.
// Scans user/blocks/ for consolidated block templates (no theme subdirectories).
// Each block file contains a <settings> tag and HTML structure but no <style> tag;
// styling is provided by theme CSS files served from /themes/.
func getBlocks(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	blocksPath := filepath.Join(s.UserPath, "blocks")
	tree := buildDirectoryTree(blocksPath)
	if tree == nil {
		tree = []BlockNode{}
	}
	return c.JSON(tree)
}

// getPanels returns all available panels as a JSON object with an "allPanels" array.
// Panel names are derived from .json filenames in user/panels/, sorted alphabetically.
func getPanels(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	panelsPath := filepath.Join(s.UserPath, "panels")

	allPanels := listPanels(panelsPath)

	return c.JSON(map[string]any{
		"allPanels": allPanels,
	})
}

// getConfig returns the current server configuration (port and joystick count).
func getConfig(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	cfg := s.GetConfig()
	return c.JSON(map[string]any{
		"port":         cfg.Port,
		"numJoysticks": cfg.NumJoysticks,
	})
}

// savePanel writes a panel definition to user/panels/<fileName>.json.
// Accepts a JSON body with "fileName" (string) and "content" (any).
// Panel content uses v2 format (object with version, metadata, grid config).
func savePanel(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	panelsPath := filepath.Join(s.UserPath, "panels")

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

// loadPanel serves a panel JSON file (v2 format) to the client viewer.
// Also updates the SpeechManager with button/slider IDs from the panel.
func loadPanel(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	name := c.Query("name", "")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing panel name")
	}

	path := filepath.Join(s.UserPath, "panels", name+".json")
	panelJSON, err := os.ReadFile(path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read panel")
	}

	if s.SpeechManager != nil {
		s.SpeechManager.UpdateButtonIDsFromPanelJSON(panelJSON)
	}

	return c.Type("application/json").Send(panelJSON)
}

// getPanelContent serves a panel JSON file (v2 format) to the editor.
// Used by the modular editor to load and display panels.
func getPanelContent(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	name := c.Query("name", "")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing panel name")
	}

	path := filepath.Join(s.UserPath, "panels", name+".json")
	panelJSON, err := os.ReadFile(path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read panel")
	}

	if s.SpeechManager != nil {
		s.SpeechManager.UpdateButtonIDsFromPanelJSON(panelJSON)
	}

	return c.Type("application/json").Send(panelJSON)
}

// deletePanel removes a panel JSON file from user/panels/.
// Accepts the panel name as a query parameter (?name=X).
func deletePanel(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	name := c.Query("name", "")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": "Missing panel name"})
	}

	path := filepath.Join(s.UserPath, "panels", name+".json")
	if err := os.Remove(path); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	return c.JSON(map[string]any{"success": true})
}

// setJoystickCount updates the number of virtual joysticks at runtime.
func setJoystickCount(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	var data map[string]any
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	countFloat, _ := data["count"].(float64)
	count := uint8(countFloat)
	s.UpdateJoystickCount(count)

	return c.JSON(map[string]any{"success": true, "count": count})
}

// pushData adds a custom metric to the DataBus via HTTP POST.
func pushData(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	var data map[string]any
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": err.Error()})
	}

	key, _ := data["key"].(string)
	if key == "" {
		return c.JSON(map[string]any{"success": false, "error": "Missing key"})
	}

	value := data["value"]
	unit, _ := data["unit"].(string)
	source, _ := data["source"].(string)

	if source != "" {
		s.DataBus.SetSource(key, value, unit, source)
	} else {
		s.DataBus.Set(key, value, unit)
	}

	return c.JSON(map[string]any{"success": true, "key": key})
}

// getThemes returns available block themes as a JSON array of theme names.
// Scans user/themes/ for .css files and strips the extension. Themes are
// applied per block in the editor and client by fetching the CSS and scoping
// all selectors to the block's DOM element ID. Panel JSON stores a top-level
// "theme" field as the default, and each block can override it with its own
// "theme" field.
func getThemes(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	themesPath := filepath.Join(s.UserPath, "themes")

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

// writeFile ensures the parent directory exists before writing the file.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
