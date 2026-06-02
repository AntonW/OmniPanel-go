package relay

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"omnipanel-go/internal/config"
)

func TestRelayHTTPRoutes(t *testing.T) {
	baseDir := t.TempDir()
	userPath := t.TempDir()

	mustWrite := func(path string, data string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	mustWrite(filepath.Join(baseDir, "static", "index.html"), "start")
	mustWrite(filepath.Join(baseDir, "static", "login.html"), "login")
	mustWrite(filepath.Join(baseDir, "static", "client", "index.html"), "panel")
	mustWrite(filepath.Join(baseDir, "static", "editor", "editor.html"), "editor")
	mustWrite(filepath.Join(baseDir, "static", "omnipanel-go-logo.svg"), "<svg/>")

	mustWrite(filepath.Join(userPath, "blocks", "a.html"), "<div></div>")
	mustWrite(filepath.Join(userPath, "themes", "default.css"), "body{}")
	mustWrite(filepath.Join(userPath, "panels", "p1.json"), `{"k":1}`)
	mustWrite(filepath.Join(userPath, "assets", "pic.png"), "png")

	s := New(&config.Config{Port: 3000, NumJoysticks: 4, AuthToken: ""}, userPath, baseDir)

	req := func(method, path string, body []byte) *http.Response {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		if len(body) > 0 {
			r.Header.Set("Content-Type", "application/json")
		}
		resp, err := s.app.Test(r)
		if err != nil {
			t.Fatalf("request %s %s failed: %v", method, path, err)
		}
		return resp
	}

	assertStatus := func(resp *http.Response, want int) {
		t.Helper()
		if resp.StatusCode != want {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, want, string(b))
		}
	}

	assertStatus(req(http.MethodGet, "/health", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/login", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/panel", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/editor", nil), http.StatusOK)

	fav := req(http.MethodGet, "/favicon.ico", nil)
	if fav.StatusCode != http.StatusMovedPermanently && fav.StatusCode != http.StatusFound {
		t.Fatalf("unexpected favicon redirect status: %d", fav.StatusCode)
	}

	assertStatus(req(http.MethodGet, "/api/config", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/api/blocks", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/api/themes", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/api/panels", nil), http.StatusOK)

	assertStatus(req(http.MethodPost, "/api/panel/save", []byte(`{"fileName":"newp","content":{"a":1}}`)), http.StatusOK)
	assertStatus(req(http.MethodGet, "/api/panel/load?name=newp", nil), http.StatusOK)
	assertStatus(req(http.MethodGet, "/api/panel/content?name=newp", nil), http.StatusOK)
	assertStatus(req(http.MethodDelete, "/api/panel/delete?name=newp", nil), http.StatusOK)

	assertStatus(req(http.MethodPost, "/api/joystick-count", []byte(`{"count":2}`)), http.StatusOK)
	assertStatus(req(http.MethodPost, "/api/data/push", []byte(`{"key":"x","value":1}`)), http.StatusOK)
	assertStatus(req(http.MethodPost, "/api/data/push", []byte(`{"value":1}`)), http.StatusOK)

	assertStatus(req(http.MethodGet, "/api/media/players", nil), http.StatusServiceUnavailable)
	assertStatus(req(http.MethodPost, "/api/media/control", []byte(`{bad`)), http.StatusBadRequest)
	assertStatus(req(http.MethodPost, "/api/media/control", []byte(`{"action":"play"}`)), http.StatusServiceUnavailable)
	assertStatus(req(http.MethodPost, "/api/media/select", []byte(`{"player":""}`)), http.StatusBadRequest)
	assertStatus(req(http.MethodGet, "/api/media/cover", nil), http.StatusBadRequest)
	assertStatus(req(http.MethodGet, "/api/media/cover?url=file:///tmp/x", nil), http.StatusServiceUnavailable)

	// Ensure /api/panels response stays JSON object with allPanels key.
	resp := req(http.MethodGet, "/api/panels", nil)
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode /api/panels: %v", err)
	}
	if _, ok := got["allPanels"]; !ok {
		t.Fatalf("missing allPanels in response: %v", got)
	}
}

