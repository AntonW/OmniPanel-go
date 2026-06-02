// Package mediautil contains small, testable helpers shared by media HTTP handlers
// and distributed agent forwarding code.
package mediautil

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"omnipanel-go/internal/mpris"
)

// Watcher is the minimal media-watcher contract used by route and agent helpers.
type Watcher interface {
	ListPlayers() []string
	GetPlayerState(playerName string) *mpris.PlayerState
	GetSelectedPlayer() string
	SetSelectedPlayer(playerName string) error
	CallMethod(playerName, method string) error
}

// ControlRequest describes a media-control request.
type ControlRequest struct {
	Player string
	Action string
	Volume float64
}

var (
	// ErrIntegrationDisabled indicates that no media watcher is configured.
	ErrIntegrationDisabled = errors.New("Media integration is not enabled")
	// ErrNoPlayersConnected indicates that no player is currently available.
	ErrNoPlayersConnected = errors.New("No media players connected")
	// ErrPlayerRequired indicates that a player name is required for selection.
	ErrPlayerRequired = errors.New("Player name is required")
	// ErrInvalidVolume indicates that volume is outside the supported 0.0–1.0 range.
	ErrInvalidVolume = errors.New("Volume must be between 0.0 and 1.0")
)

// BuildPlayersResponse builds the JSON-ready response for /api/media/players.
func BuildPlayersResponse(w Watcher, systemVolume float64) map[string]any {
	if w == nil {
		return map[string]any{
			"enabled": false,
			"players": []string{},
		}
	}

	players := w.ListPlayers()
	playerStates := make([]map[string]any, 0, len(players))
	for _, name := range players {
		ps := w.GetPlayerState(name)
		if ps == nil {
			continue
		}
		playerStates = append(playerStates, map[string]any{
			"name":           ps.PlayerName,
			"identity":       ps.Identity,
			"playbackStatus": ps.PlaybackStatus,
			"title":          ps.Title,
			"artist":         ps.Artist,
			"album":          ps.Album,
			"artUrl":         ps.ArtURL,
			"canControl":     ps.CanControl,
			"volume":         ps.Volume,
		})
	}

	return map[string]any{
		"enabled":      true,
		"players":      playerStates,
		"systemVolume": systemVolume,
	}
}

// ResolvePlayer returns the requested player, the selected player, or the first
// available player in that order.
func ResolvePlayer(w Watcher, requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	if w == nil {
		return "", ErrIntegrationDisabled
	}
	if selected := w.GetSelectedPlayer(); selected != "" {
		return selected, nil
	}
	players := w.ListPlayers()
	if len(players) == 0 {
		return "", ErrNoPlayersConnected
	}
	return players[0], nil
}

// ExecuteControl executes a media control request using the watcher and system-volume callback.
func ExecuteControl(w Watcher, req ControlRequest, setSystemVolume func(float64) error) (string, error) {
	if w == nil {
		return "", ErrIntegrationDisabled
	}

	player, err := ResolvePlayer(w, req.Player)
	if err != nil {
		return "", err
	}

	switch req.Action {
	case "play":
		err = w.CallMethod(player, "Play")
	case "pause":
		err = w.CallMethod(player, "Pause")
	case "playpause":
		err = w.CallMethod(player, "PlayPause")
	case "stop":
		err = w.CallMethod(player, "Stop")
	case "next":
		err = w.CallMethod(player, "Next")
	case "previous":
		err = w.CallMethod(player, "Previous")
	case "volume":
		if req.Volume < 0 || req.Volume > 1 {
			return "", ErrInvalidVolume
		}
		err = setSystemVolume(req.Volume)
	default:
		return "", fmt.Errorf("Unknown action: %s", req.Action)
	}
	if err != nil {
		return "", err
	}
	return player, nil
}

// SelectPlayer validates and applies a player selection.
func SelectPlayer(w Watcher, player string) error {
	if w == nil {
		return ErrIntegrationDisabled
	}
	if player == "" {
		return ErrPlayerRequired
	}
	return w.SetSelectedPlayer(player)
}

// NormalizeFileURLPath strips a file:// prefix from a URL-like local path.
func NormalizeFileURLPath(fileURL string) string {
	return strings.TrimPrefix(fileURL, "file://")
}

// AllowedCoverPrefixes returns the trusted local directories from which cover art
// may be proxied.
func AllowedCoverPrefixes(homeDir, tempDir string) []string {
	return []string{
		"/tmp/",
		"/var/tmp/",
		homeDir + "/.cache/",
		tempDir + string(filepath.Separator),
	}
}

// IsAllowedCoverPath reports whether a cleaned file path is under one of the
// trusted cover-art prefixes.
func IsAllowedCoverPath(cleanPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			return true
		}
	}
	return false
}

// ContentTypeForPath infers an image content type from the file extension.
func ContentTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

