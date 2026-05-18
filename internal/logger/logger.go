// Package logger provides configurable structured logging for OmniPanel-go.
//
// It supports three output formats: color (default), text, and JSON.
// The color format uses the tint handler for human-readable terminal output
// with ANSI color coding. The text format uses slog's built-in TextHandler.
// The JSON format uses slog's built-in JSONHandler for machine-parseable output.
//
// Usage:
//
//	logger.Init("color")  // colored terminal output (default)
//	logger.Init("json")   // JSON lines for production/log aggregation
//	logger.Init("text")   // plain text with source info
//
// The format can also be controlled via the LOG_FORMAT environment variable
// or the --log-format command-line flag.
package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// Init configures the global slog logger with the specified format.
// Valid formats are "color" (default), "text", and "json".
// An empty string defaults to "color". Unknown formats fall back to text
// with a warning printed to stderr.
func Init(format string) {
	var handler slog.Handler

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, nil)
	case "text":
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})
	case "color", "":
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			AddSource:  true,
			Level:      slog.LevelDebug,
			TimeFormat: "15:04:05",
		})
	default:
		fmt.Fprintf(os.Stderr, "unknown log format %q, falling back to text\n", format)
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})
	}

	slog.SetDefault(slog.New(handler))
}
