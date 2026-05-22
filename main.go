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
// Deployment modes:
//   - Default: HTTP server + subsystems on the same machine (traditional)
//   - Serve: central relay server (WebUI + WebSocket hub, no subsystems)
//   - Connect: host agent (WebSocket client + subsystems, no HTTP server)
//
// Entry point flow:
//  1. Parse subcommand: default, serve, or connect
//  2. Parse --log-format flag (fall back to LOG_FORMAT env var, then "color")
//  3. Initialize structured logger
//  4. Discover config and user data paths
//  5. Load configuration (with env var override support)
//  6. Run selected mode:
//     - Default: HTTP server + subsystems (current behavior)
//     - Serve: central relay server (HTTP + WebSocket relay, no subsystems)
//     - Connect: host agent (WebSocket client + subsystems, no HTTP server)
//  7. Block until SIGINT or SIGTERM, then gracefully shut down
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

	"omnipanel-go/internal/agent"
	"omnipanel-go/internal/config"
	"omnipanel-go/internal/logger"
	"omnipanel-go/internal/relay"
	"omnipanel-go/internal/routes"
	"omnipanel-go/internal/starter"
	"omnipanel-go/internal/state"
)

func main() {
	subcommand := ""
	serverAddrArg := ""
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			subcommand = "serve"
		case "connect":
			subcommand = "connect"
			if len(os.Args) > 2 {
				serverAddrArg = os.Args[2]
			}
		}
	}

	logFormat := flag.String("log-format", "", "log format: color (default), text, json (default: env LOG_FORMAT or color)")
	flag.Parse()

	format := *logFormat
	if format == "" {
		format = os.Getenv("LOG_FORMAT")
	}
	logger.Init(format)

	configPath := config.FindConfigPath()
	userPath := config.FindUserPath()
	baseDir, _ := os.Getwd()

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "error", err)
		cfg = &config.Config{
			Port:         3000,
			NumJoysticks: 4,
		}
	}

	switch subcommand {
	case "serve":
		runServe(cfg, userPath, baseDir)
	case "connect":
		runConnect(cfg, configPath, userPath, baseDir, serverAddrArg)
	default:
		runDefault(cfg, configPath, userPath, baseDir)
	}
}

func runDefault(cfg *config.Config, configPath, userPath, baseDir string) {
	slog.Info("Starting OmniPanel-go server", "port", cfg.Port)

	appState := state.New(cfg, configPath, userPath, baseDir)
	defer appState.Close()

	app := routes.NewRouter(appState)

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	slog.Info("Server starting", "addr", addr)

	go func() {
		if err := app.Listen(addr); err != nil {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
	slog.Info("HTTP server closed")
}

func runServe(cfg *config.Config, userPath, baseDir string) {
	slog.Info("Starting OmniPanel-go relay server", "port", cfg.Port)

	starter.Init(userPath)

	srv := relay.New(cfg, userPath, baseDir)

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	slog.Info("Relay server starting", "addr", addr)
	slog.Info("Waiting for browser clients and host agent connection")

	go func() {
		if err := srv.Listen(addr); err != nil {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down relay server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
	slog.Info("Relay server closed")
}

func runConnect(cfg *config.Config, configPath, userPath, baseDir, serverAddrArg string) {
	serverAddr := serverAddrArg
	if serverAddr == "" {
		serverAddr = cfg.ServerAddress
	}
	if serverAddr == "" {
		slog.Error("No server address provided. Use 'connect <IP>:<PORT>' or set server_address in config.json")
		os.Exit(1)
	}

	slog.Info("Starting OmniPanel-go host agent", "server", serverAddr)

	agt := agent.New(cfg, configPath, userPath, baseDir)
	defer agt.Close()

	go func() {
		agt.Run(serverAddr)
	}()

	agt.WaitSignal()
	slog.Info("Shutting down host agent...")
	agt.Stop()
	slog.Info("Host agent stopped")
}
