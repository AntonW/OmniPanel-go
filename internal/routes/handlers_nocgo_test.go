//go:build !cgo

package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
	"omnipanel-go/internal/devices"
	"omnipanel-go/internal/state"
)

func TestEditorHandlers(t *testing.T) {
	baseDir := t.TempDir()
	userPath := t.TempDir()

	for _, path := range []string{
		filepath.Join(baseDir, "static", "editor", "editor.html"),
		filepath.Join(userPath, "blocks", "demo.html"),
		filepath.Join(userPath, "themes", "default.css"),
		filepath.Join(userPath, "panels", "p1.json"),
	} {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte("test"), 0o644)
	}

	appState := &state.AppState{
		Config:     &config.Config{},
		StaticDir:  filepath.Join(baseDir, "static"),
		UserPath:   userPath,
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("state", appState)
		return c.Next()
	})

	app.Get("/editor", serveEditorUI)
	app.Get("/blocks", getBlocks)
	app.Get("/themes", getThemes)
	app.Get("/panels", getPanels)
	app.Post("/panel/save", savePanel)
	app.Get("/panel/load", loadPanel)
	app.Get("/panel/content", getPanelContent)
	app.Delete("/panel/delete", deletePanel)
	app.Get("/config", getConfig)

	req := func(method, path string, body []byte) (*http.Response, string) {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		if len(body) > 0 {
			r.Header.Set("Content-Type", "application/json")
		}
		resp, err := app.Test(r)
		if err != nil {
			t.Fatalf("request %s %s: %v", method, path, err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		return resp, string(respBody)
	}

	// Editor UI
	resp, _ := req(http.MethodGet, "/editor", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("editor GET failed: %d", resp.StatusCode)
	}

	// Blocks list
	resp, body := req(http.MethodGet, "/blocks", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("blocks GET failed: %d", resp.StatusCode)
	}

	// Themes list
	resp, body = req(http.MethodGet, "/themes", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("themes GET failed: %d", resp.StatusCode)
	}

	// Panels list
	resp, body = req(http.MethodGet, "/panels", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("panels GET failed: %d body=%s", resp.StatusCode, body)
	}
	var panelsResp map[string]any
	if err := json.Unmarshal([]byte(body), &panelsResp); err != nil {
		t.Fatalf("panels response not json: %v", err)
	}
	if _, ok := panelsResp["allPanels"]; !ok {
		t.Fatalf("missing allPanels key: %v", panelsResp)
	}

	// Config
	resp, _ = req(http.MethodGet, "/config", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("config GET failed: %d", resp.StatusCode)
	}

	// Save panel
	resp, _ = req(http.MethodPost, "/panel/save", []byte(`{"fileName":"test","content":{"x":1}}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save panel failed: %d", resp.StatusCode)
	}

	// Load panel (saved)
	resp, _ = req(http.MethodGet, "/panel/load?name=test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("load panel failed: %d", resp.StatusCode)
	}

	// Panel content
	resp, _ = req(http.MethodGet, "/panel/content?name=test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("panel content failed: %d", resp.StatusCode)
	}

	// Delete panel
	resp, _ = req(http.MethodDelete, "/panel/delete?name=test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete panel failed: %d", resp.StatusCode)
	}

	// Error cases
	resp, _ = req(http.MethodPost, "/panel/save", []byte(`{bad`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("save with bad json should fail: %d", resp.StatusCode)
	}

	resp, _ = req(http.MethodPost, "/panel/save", []byte(`{"content":{}}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save without fileName should use default: %d", resp.StatusCode)
	}

	resp, _ = req(http.MethodGet, "/panel/load?name=missing", nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("load nonexistent panel should 500: %d", resp.StatusCode)
	}

	resp, _ = req(http.MethodGet, "/panel/load", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("load without name should 400: %d", resp.StatusCode)
	}

	resp, _ = req(http.MethodDelete, "/panel/delete", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete without name should 400: %d", resp.StatusCode)
	}
}

func TestJoystickCountAndDataPush(t *testing.T) {
	s := &state.AppState{
		Config:          &config.Config{NumJoysticks: 2},
		ConfigPath:      filepath.Join(t.TempDir(), "config.json"),
		JoystickManager: devices.New(2),
		DataBus:         databus.New(),
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("state", s)
		return c.Next()
	})

	app.Post("/joystick-count", setJoystickCount)
	app.Post("/data/push", pushData)

	req := func(method, path string, body []byte) *http.Response {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(r)
		return resp
	}

	// Good joystick-count
	resp := req(http.MethodPost, "/joystick-count", []byte(`{"count":4}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("joystick-count failed: %d", resp.StatusCode)
	}

	// Bad joystick-count parse
	resp = req(http.MethodPost, "/joystick-count", []byte(`{bad`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad joystick-count parse should 400: %d", resp.StatusCode)
	}

	// Data push success
	resp = req(http.MethodPost, "/data/push", []byte(`{"key":"k","value":1,"unit":"u"}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("data push failed: %d", resp.StatusCode)
	}

	// Data push without key
	resp = req(http.MethodPost, "/data/push", []byte(`{"value":1}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("data push without key should 200: %d", resp.StatusCode)
	}

	// Data push parse error
	resp = req(http.MethodPost, "/data/push", []byte(`{bad`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("data push parse error should 400: %d", resp.StatusCode)
	}
}

