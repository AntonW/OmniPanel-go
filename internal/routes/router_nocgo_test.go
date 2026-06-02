//go:build !cgo

package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestServeFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "index.html")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	app := fiber.New()
	app.Get("/ok", func(c *fiber.Ctx) error {
		return serveFile(c, file, "text/plain")
	})
	app.Get("/missing", func(c *fiber.Ctx) error {
		return serveFile(c, filepath.Join(dir, "missing.txt"), "text/plain")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "hello" {
		t.Fatalf("unexpected serveFile success response: code=%d body=%q", resp.StatusCode, string(body))
	}

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/missing", nil))
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing file should return 404, got %d", resp.StatusCode)
	}
}

func TestBuildDirectoryTreeAndListPanels(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.html"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write a.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write b.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "c.html"), []byte("c"), 0o644); err != nil {
		t.Fatalf("write c.html: %v", err)
	}

	tree := buildDirectoryTree(dir)
	if len(tree) != 2 {
		t.Fatalf("expected html file + folder only, got %d nodes", len(tree))
	}

	panelsDir := filepath.Join(dir, "panels")
	if err := os.MkdirAll(panelsDir, 0o755); err != nil {
		t.Fatalf("mkdir panels: %v", err)
	}
	_ = os.WriteFile(filepath.Join(panelsDir, "zeta.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(panelsDir, "alpha.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(panelsDir, "note.txt"), []byte("x"), 0o644)

	panels := listPanels(panelsDir)
	if len(panels) != 2 || panels[0] != "alpha" || panels[1] != "zeta" {
		t.Fatalf("unexpected panel list: %v", panels)
	}
}

