package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithEnvOverride(t *testing.T) {
	t.Setenv("OMNIPANEL_PORT", "9123")
	t.Setenv("OMNIPANEL_NUMJOYSTICKS", "4")
	t.Setenv("OMNIPANEL_SERVER_ADDRESS", "ws://env-host:3000")
	t.Setenv("OMNIPANEL_AUTH_TOKEN", "env-token")
	t.Setenv("OMNIPANEL_USER_PATH", "env-user")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "port": 8080,
  "numJoysticks": 2,
  "server_address": "ws://file-host:3000",
  "auth_token": "file-token",
  "user_path": "file-user",
  "media_player": {"enabled": true, "poll_interval": 1000}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}

	if cfg.Port != 9123 {
		t.Fatalf("Port = %d, want 9123", cfg.Port)
	}
	if cfg.NumJoysticks != 4 {
		t.Fatalf("NumJoysticks = %d, want 4", cfg.NumJoysticks)
	}
	if cfg.ServerAddress != "ws://env-host:3000" {
		t.Fatalf("ServerAddress = %q", cfg.ServerAddress)
	}
	if cfg.AuthToken != "env-token" {
		t.Fatalf("AuthToken = %q", cfg.AuthToken)
	}
	if cfg.UserPath != "env-user" {
		t.Fatalf("UserPath = %q", cfg.UserPath)
	}
	if !cfg.MediaPlayer.Enabled || cfg.MediaPlayer.PollInterval != 1000 {
		t.Fatalf("MediaPlayer config not loaded from file: %#v", cfg.MediaPlayer)
	}
}

func TestLoad_ConfigNotFoundReturnsZeroConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	if _, err := Load(path); err == nil {
		t.Fatalf("Load missing explicit config should fail")
	}
}

func TestLoad_UnmarshalError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// "port" expects a number; object should trigger unmarshal type error.
	content := `{"port":{"unexpected":true}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatalf("Load should fail on invalid schema")
	}
}

func TestSave_WritesFileAndSafeWriteRejectsSecondWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &Config{
		Port:          3000,
		NumJoysticks:  2,
		ServerAddress: "ws://127.0.0.1:3000",
		AuthToken:     "abc",
		UserPath:      "user",
		Speech: SpeechConfig{
			Enabled:      true,
			RecordingLoc: "client",
		},
		MediaPlayer: MediaPlayerConfig{Enabled: true, PollInterval: 900},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("first Save error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
	if err := cfg.Save(path); err == nil {
		t.Fatalf("second Save should fail due SafeWriteConfig, got nil")
	}
}

func TestFindConfigPath_LocalPreferred(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	dir, err := os.MkdirTemp("", "cfgpath-local-")
	if err != nil {
		t.Fatalf("MkdirTemp error = %v", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.WriteFile("config.json", []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if got := FindConfigPath(); got != "config.json" {
		t.Fatalf("FindConfigPath() = %q, want config.json", got)
	}
}

func TestFindConfigPath_Fallback(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	dir, err := os.MkdirTemp("", "cfgpath-fallback-")
	if err != nil {
		t.Fatalf("MkdirTemp error = %v", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if got := FindConfigPath(); got != "config.json" {
		t.Fatalf("FindConfigPath fallback = %q, want config.json", got)
	}
}

func TestFindConfigPath_PackagedFallback(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable error = %v", err)
	}
	exeDir := filepath.Dir(exe)
	packaged := filepath.Join(exeDir, "config.json")

	if _, err := os.Stat(packaged); err == nil {
		t.Skip("packaged config.json already exists next to test binary")
	}
	if err := os.WriteFile(packaged, []byte("{}"), 0o644); err != nil {
		t.Skipf("cannot create packaged config near executable: %v", err)
	}
	defer os.Remove(packaged)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	dir, err := os.MkdirTemp("", "cfgpath-packaged-")
	if err != nil {
		t.Fatalf("MkdirTemp error = %v", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if got := FindConfigPath(); got != packaged {
		t.Fatalf("FindConfigPath packaged = %q, want %q", got, packaged)
	}
}

func TestFindUserPath_LocalPreferredAndFallback(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	dir, err := os.MkdirTemp("", "userpath-")
	if err != nil {
		t.Fatalf("MkdirTemp error = %v", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if got := FindUserPath(); got != "user" {
		t.Fatalf("FindUserPath fallback = %q, want user", got)
	}

	if err := os.Mkdir("user", 0o755); err != nil {
		t.Fatalf("Mkdir error = %v", err)
	}
	if got := FindUserPath(); got != "user" {
		t.Fatalf("FindUserPath local = %q, want user", got)
	}
}

func TestFindUserPath_PackagedFallback(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable error = %v", err)
	}
	exeDir := filepath.Dir(exe)
	packaged := filepath.Join(exeDir, "user")

	if info, err := os.Stat(packaged); err == nil && info.IsDir() {
		t.Skip("packaged user directory already exists next to test binary")
	}
	if err := os.Mkdir(packaged, 0o755); err != nil {
		t.Skipf("cannot create packaged user near executable: %v", err)
	}
	defer os.RemoveAll(packaged)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	dir, err := os.MkdirTemp("", "userpath-packaged-")
	if err != nil {
		t.Fatalf("MkdirTemp error = %v", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if got := FindUserPath(); got != packaged {
		t.Fatalf("FindUserPath packaged = %q, want %q", got, packaged)
	}
}



