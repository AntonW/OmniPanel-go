//go:build !windows

// System volume control for Linux/macOS via PulseAudio/PipeWire (pactl).
package routes

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// setSystemVolume sets the system-wide volume via pactl (PulseAudio/PipeWire).
// The volume parameter is a float between 0.0 (muted) and 1.0 (max).
func setSystemVolume(volume float64) error {
	pct := int(volume * 100)
	cmd := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", strconv.Itoa(pct)+"%")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("media: system volume control failed", "error", err, "output", string(output))
		return err
	}
	slog.Debug("media: system volume set", "volume", volume, "percent", pct)
	return nil
}

// getSystemVolume reads the current system-wide volume via pactl (PulseAudio/PipeWire).
// Returns the volume as a float between 0.0 and 1.0. Falls back to 0.5 on error.
func getSystemVolume() (float64, error) {
	cmd := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("media: get system volume failed", "error", err, "output", string(output))
		return 0.5, err
	}
	// Output format: "Volume: front-left: 65536 / 100% / 0.00 dB"
	s := string(output)
	idx := strings.Index(s, "%")
	if idx == -1 {
		return 0.5, fmt.Errorf("media: unexpected pactl output format: %s", s)
	}
	start := idx
	for start > 0 && (s[start-1] == ' ' || (s[start-1] >= '0' && s[start-1] <= '9')) {
		start--
	}
	pctStr := strings.TrimSpace(s[start:idx])
	pct, err := strconv.Atoi(pctStr)
	if err != nil {
		return 0.5, fmt.Errorf("media: failed to parse volume percentage: %w", err)
	}
	return float64(pct) / 100.0, nil
}


