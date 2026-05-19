// OmniPanel-go is a web-based control panel system that runs on Linux or Windows.
// It lets you design custom control panels with draggable blocks, display them
// on any device with a web browser, simulate virtual joystick and mouse input
// on the host machine, execute shell commands and HTTP requests from panel
// buttons, monitor system metrics in real time, and control your panel with
// voice commands using speech recognition.
//
// Architecture:
//   - Go server (Fiber HTTP + WebSocket) handles all backend logic
//   - Web frontend (HTML/JS/CSS) provides panel, editor, and start page UIs
//   - Virtual input devices (uinput on Linux, vJoy/SendInput on Windows)
//     simulate hardware input from web panel interactions
//   - Speech recognition (Vosk offline or llama-cpp HTTP API) enables
//     voice-controlled commands
//
// Entry point flow:
//  1. Parse --log-format flag (fall back to LOG_FORMAT env var, then "color")
//  2. Initialize structured logger
//  3. Discover config and user data paths
//  4. Load configuration (with env var override support)
//  5. Create AppState (central hub for all subsystems)
//  6. Build HTTP router with all routes registered
//  7. Start server in a goroutine
//  8. Block until SIGINT or SIGTERM, then gracefully shut down HTTP server
//     and release all subsystem resources (virtual input devices, STT engine,
//     audio capture devices)
//
// See docs/tutorials/ for a guided tour of the codebase.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/logger"
	"omnipanel-go/internal/routes"
	"omnipanel-go/internal/state"
)

func main() {
	logFormat := flag.String("log-format", "", "log format: color (default), text, json (default: env LOG_FORMAT or color)")
	flag.Parse()

	format := *logFormat
	if format == "" {
		format = os.Getenv("LOG_FORMAT")
	}
	logger.Init(format)

	slog.Info("Starting OmniPanel-go server", "log_format", format)

	// Discover config and user data paths (local dir or next to executable)
	configPath := config.FindConfigPath()
	userPath := config.FindUserPath()
	baseDir, _ := os.Getwd()

	slog.Info("Loading config", "path", configPath)

	// Load config; fall back to defaults if file is missing or invalid
	// Environment variables (OMNIPANEL_PANEL, OMNIPANEL_PORT, OMNIPANEL_NUMJOYSTICKS)
	// override config file values automatically via viper
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "error", err)
		cfg = &config.Config{
			Port:         3000,
			NumJoysticks: 4,
		}
	}

	slog.Info("Configuration loaded",
		"port", cfg.Port,
		"joysticks", cfg.NumJoysticks,
		"user_path", userPath,
	)

	// Create the central application state (joystick managers, databus, etc.)
	appState := state.New(cfg, configPath, userPath, baseDir)
	defer appState.Close() // cleanup virtual input devices on exit

	// Build the HTTP router with all routes registered
	app := routes.NewRouter(appState)

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	slog.Info("Server starting", "addr", addr)

	// Start the HTTP server in a goroutine so we can listen for shutdown signals
	go func() {
		if err := app.Listen(addr); err != nil {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block until SIGINT (Ctrl+C) or SIGTERM is received
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down...")

	slog.Info("Closing HTTP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
	slog.Info("HTTP server closed")

	slog.Info("Closing app state...")
}
