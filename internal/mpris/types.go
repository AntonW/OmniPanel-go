// Package mpris provides media player integration.
// This file contains platform-independent types shared across all implementations.
package mpris

// PlayerState holds the current state of a media player.
// On Linux this is populated via MPRIS2/D-Bus; on Windows via SMTC (System Media Transport Controls).
type PlayerState struct {
	// Identity is the human-readable player name.
	Identity string
	// PlayerName is the D-Bus name suffix (Linux) or SMTC SourceAppUserModelId (Windows).
	PlayerName string
	// PlaybackStatus is one of "Playing", "Paused", or "Stopped".
	PlaybackStatus string
	// Title is the current track title.
	Title string
	// Artist is the current track artist. Multiple artists are joined with ", ".
	Artist string
	// Album is the current track album.
	Album string
	// ArtURL is the cover art URL. May be a file:// path to a local temp file.
	ArtURL string
	// Length is the track duration in microseconds.
	Length int64
	// Position is the current playback position in microseconds.
	Position int64
	// Volume is the player volume (0.0–1.0). On Windows this reflects system volume.
	Volume float64
	// CanPlay indicates whether Play is supported.
	CanPlay bool
	// CanPause indicates whether Pause is supported.
	CanPause bool
	// CanGoNext indicates whether Next/SkipNext is supported.
	CanGoNext bool
	// CanGoPrevious indicates whether Previous/SkipPrevious is supported.
	CanGoPrevious bool
	// CanControl indicates whether the player accepts control commands.
	CanControl bool
}

