// Package routes provides HTTP handlers for media player control.
// On Linux, it uses MPRIS over D-Bus; on Windows, it uses SMTC (System Media Transport Controls).
// It exposes four endpoints:
//   - GET /api/media/players — list connected players and their current state
//   - POST /api/media/control — send playback commands (play, pause, next, etc.)
//   - POST /api/media/select — select which player's state is published to the DataBus
//   - GET /api/media/cover — proxy local cover art files for browser access
//
// System volume and cover-art path handling are implemented in platform-specific files
// (media_volume_linux.go / media_volume_windows.go).
package routes

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/mediautil"
	"omnipanel-go/internal/state"
)

// listMPRISPlayers returns connected media players and their current state.
// Works with both MPRIS on Linux and SMTC on Windows.
// The response also includes current system master volume for +/- volume buttons.
func listMPRISPlayers(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	sysVol, _ := getSystemVolume()
	return c.JSON(mediautil.BuildPlayersResponse(s.MPRISWatcher, sysVol))
}

// controlMPRIS handles /api/media/control playback commands.
// If no player is specified, the currently selected player is used.
// The volume action controls system master volume via platform-specific helpers
// in media_volume_linux.go and media_volume_windows.go.
func controlMPRIS(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	if s.MPRISWatcher == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "Media integration is not enabled",
		})
	}

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

	slog.Info("mpris control request", "player", req.Player, "action", req.Action, "volume", req.Volume)
	player, err := mediautil.ExecuteControl(s.MPRISWatcher, mediautil.ControlRequest{
		Player: req.Player,
		Action: req.Action,
		Volume: req.Volume,
	}, setSystemVolume)

	if err != nil {
		status := fiber.StatusInternalServerError
		switch err {
		case mediautil.ErrIntegrationDisabled:
			status = fiber.StatusServiceUnavailable
		case mediautil.ErrNoPlayersConnected:
			status = fiber.StatusNotFound
		case mediautil.ErrInvalidVolume:
			status = fiber.StatusBadRequest
		default:
			if strings.HasPrefix(err.Error(), "Unknown action:") {
				status = fiber.StatusBadRequest
			}
		}
		return c.Status(status).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"success": true,
		"player":  player,
		"action":  req.Action,
	})
}

// selectMPRISPlayer sets the active player for control and data updates.
// The selected player's state is published to mpris_* DataBus keys for frontend
// compatibility, and commands without explicit player target this selection.
func selectMPRISPlayer(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	if s.MPRISWatcher == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "Media integration is not enabled",
		})
	}

	var req struct {
		Player string `json:"player"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Invalid request body",
		})
	}

	if err := mediautil.SelectPlayer(s.MPRISWatcher, req.Player); err != nil {
		status := fiber.StatusNotFound
		if err == mediautil.ErrPlayerRequired {
			status = fiber.StatusBadRequest
		} else if err == mediautil.ErrIntegrationDisabled {
			status = fiber.StatusServiceUnavailable
		}
		return c.Status(status).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"success": true,
		"player":  req.Player,
	})
}

// serveMPRISCoverArt serves /api/media/cover via HTTP proxy.
// Browsers cannot access file:// URLs directly, so this endpoint
// reads the local file and serves it with the correct MIME type.
// URL format: /api/media/cover?url=<encoded-file-path>
func serveMPRISCoverArt(c *fiber.Ctx) error {
	filePath := c.Query("url")
	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Missing url parameter",
		})
	}

	// Strip file:// prefix if present
	filePath = mediautil.NormalizeFileURLPath(filePath)

	// Security: only allow files inside known temp/cache directories.
	// Covers Linux (/tmp, /var/tmp, $HOME/.cache) and Windows (%TEMP%).
	cleanPath := filepath.Clean(filePath)
	allowedPrefixes := mediautil.AllowedCoverPrefixes(os.Getenv("HOME"), os.TempDir())
	if !mediautil.IsAllowedCoverPath(cleanPath, allowedPrefixes) {
		return c.Status(fiber.StatusForbidden).JSON(map[string]any{
			"error": "Access denied",
		})
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(map[string]any{
			"error": "Cover art not found",
		})
	}

	c.Set("Content-Type", mediautil.ContentTypeForPath(cleanPath))
	c.Set("Cache-Control", "public, max-age=60")
	return c.Send(data)
}

