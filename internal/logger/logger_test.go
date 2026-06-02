package logger

import (
	"log/slog"
	"testing"
)

func TestInit_AllFormats(t *testing.T) {
	formats := []string{"json", "text", "color", "", "unknown-format"}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			Init(f)
			if slog.Default() == nil {
				t.Fatalf("slog default logger is nil after Init(%q)", f)
			}
		})
	}
}

