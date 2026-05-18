package routes

import (
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/state"
)

// serveStartPage serves the start page HTML (panel list, editor link, host controls).
func serveStartPage(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	path := filepath.Join(s.StaticDir, "index.html")
	return serveFile(c, path, "text/html; charset=utf-8")
}

// servePanel serves the client panel HTML page (the main UI shown on tablets/browsers).
func servePanel(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)
	path := filepath.Join(s.StaticDir, "client", "index.html")
	return serveFile(c, path, "text/html; charset=utf-8")
}
