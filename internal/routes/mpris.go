// Package routes provides HTTP handlers for MPRIS media player control.
// It exposes four endpoints:
//   - GET /api/mpris/players — list connected players and their current state
//   - POST /api/mpris/control — send playback commands (play, pause, next, etc.)
//   - POST /api/mpris/select — select which player's state is published to the DataBus
//   - GET /api/mpris/cover — proxy local cover art files for browser access
//
// Control commands default to the selected player when no player is specified
// in the request body.
package routes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"omnipanel-go/internal/state"
)

// listMPRISPlayers returns a list of currently connected MPRIS media players
// with their current state (identity, playback status, metadata, volume, capabilities).
func listMPRISPlayers(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	if s.MPRISWatcher == nil {
		return c.JSON(map[string]any{
			"enabled": false,
			"players": []string{},
		})
	}

	players := s.MPRISWatcher.ListPlayers()
	playerStates := make([]map[string]any, 0, len(players))

	for _, name := range players {
		state := s.MPRISWatcher.GetPlayerState(name)
		if state != nil {
			playerStates = append(playerStates, map[string]any{
				"name":           state.PlayerName,
				"identity":       state.Identity,
				"playbackStatus": state.PlaybackStatus,
				"title":          state.Title,
				"artist":         state.Artist,
				"album":          state.Album,
				"artUrl":         state.ArtURL,
				"canControl":     state.CanControl,
				"volume":         state.Volume,
			})
		}
	}

	return c.JSON(map[string]any{
		"enabled": true,
		"players": playerStates,
	})
}

// controlMPRIS handles media control commands (play, pause, next, previous, stop, volume).
// If no player is specified in the request, the currently selected player is used.
func controlMPRIS(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	if s.MPRISWatcher == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "MPRIS is not enabled",
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

	if req.Player == "" {
		req.Player = s.MPRISWatcher.GetSelectedPlayer()
	}
	if req.Player == "" {
		players := s.MPRISWatcher.ListPlayers()
		if len(players) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(map[string]any{
				"error": "No media players connected",
			})
		}
		req.Player = players[0]
	}

	var err error
	switch req.Action {
	case "play":
		err = s.MPRISWatcher.CallMethod(req.Player, "Play")
	case "pause":
		err = s.MPRISWatcher.CallMethod(req.Player, "Pause")
	case "playpause":
		err = s.MPRISWatcher.CallMethod(req.Player, "PlayPause")
	case "stop":
		err = s.MPRISWatcher.CallMethod(req.Player, "Stop")
	case "next":
		err = s.MPRISWatcher.CallMethod(req.Player, "Next")
	case "previous":
		err = s.MPRISWatcher.CallMethod(req.Player, "Previous")
	case "volume":
		if req.Volume < 0 || req.Volume > 1 {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
				"error": "Volume must be between 0.0 and 1.0",
			})
		}
		err = s.MPRISWatcher.SetVolume(req.Player, req.Volume)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Unknown action: " + req.Action,
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"success": true,
		"player":  req.Player,
		"action":  req.Action,
	})
}

// selectMPRISPlayer sets the active MPRIS player for control and data updates.
// The selected player's state is published to the mpris_* DataBus keys, and
// control commands without an explicit player target this player.
func selectMPRISPlayer(c *fiber.Ctx) error {
	s := c.Locals("state").(*state.AppState)

	if s.MPRISWatcher == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{
			"error": "MPRIS is not enabled",
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

	if req.Player == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Player name is required",
		})
	}

	if err := s.MPRISWatcher.SetSelectedPlayer(req.Player); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"success": true,
		"player":  req.Player,
	})
}

// serveMPRISCoverArt serves MPRIS cover art files via HTTP proxy.
// Browsers cannot access file:// URLs directly, so this endpoint
// reads the local file and serves it with proper MIME type.
// URL format: /api/mpris/cover?url=<encoded-file-path>
func serveMPRISCoverArt(c *fiber.Ctx) error {
	filePath := c.Query("url")
	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "Missing url parameter",
		})
	}

	// Strip file:// prefix if present
	filePath = strings.TrimPrefix(filePath, "file://")

	// Security: only allow files under /tmp, /var/tmp, or user's cache directory
	cleanPath := filepath.Clean(filePath)
	allowedPrefixes := []string{"/tmp/", "/var/tmp/", os.Getenv("HOME") + "/.cache/"}
	allowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
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

	// Detect MIME type from extension
	ext := strings.ToLower(filepath.Ext(cleanPath))
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=60")
	return c.Send(data)
}
