// Package config handles loading, saving, and discovering the application's
// configuration file (config.json). It uses Viper for configuration management,
// including OMNIPANEL_* environment variable support.
//
// Configuration precedence: environment variables > config file > defaults.
//
// Config discovery follows a fallback strategy:
//  1. Current working directory
//  2. Directory next to the executable binary
//  3. Fallback to the default path (caller handles missing file)
//
// See docs/tutorials/02-configuration.md for a detailed walkthrough.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// SpeechConfig holds speech recognition settings.
type SpeechConfig struct {
	// Enabled turns speech recognition on or off.
	Enabled bool `mapstructure:"enabled" json:"enabled"`
	// RecordingLoc selects where audio is captured: "client" or "host".
	RecordingLoc string `mapstructure:"recording_location" json:"recording_location"`
	// TriggerMode selects activation mode: "push-to-talk" or "wake-word".
	TriggerMode string `mapstructure:"trigger_mode" json:"trigger_mode"`
	// WakeWord is the phrase used to activate wake-word mode.
	WakeWord string `mapstructure:"wake_word" json:"wake_word"`
	// WakeWordListenSec is host-side follow-up listen duration after wake-word detection.
	WakeWordListenSec int `mapstructure:"wake_word_listen_sec" json:"wake_word_listen_sec"`
	// STTEngine selects speech-to-text backend, for example "vosk" or "llama-cpp".
	STTEngine string `mapstructure:"stt_engine" json:"stt_engine"`
	// VoskModelPath points to a Vosk model directory. Empty enables auto-download.
	VoskModelPath string `mapstructure:"vosk_model_path" json:"vosk_model_path"`
	// VoskRuntimeURL overrides the Windows Vosk runtime ZIP download URL.
	// If empty, the built-in default release URL is used.
	VoskRuntimeURL string `mapstructure:"vosk_runtime_url" json:"vosk_runtime_url"`
	// LlamaCppURL is the base URL of the llama-cpp server.
	LlamaCppURL string `mapstructure:"llama_cpp_url" json:"llama_cpp_url"`
	// LlamaCppAPIKey is an optional API key for llama-cpp requests.
	LlamaCppAPIKey string `mapstructure:"llama_cpp_api_key" json:"llama_cpp_api_key"`
	// LlamaCppAPIMode selects llama-cpp API mode: "transcriptions" or "chat".
	LlamaCppAPIMode string `mapstructure:"llama_cpp_api_mode" json:"llama_cpp_api_mode"`
	// LlamaCppModel selects the model name used in chat mode.
	LlamaCppModel string `mapstructure:"llama_cpp_model" json:"llama_cpp_model"`
	// LlamaCppPrompt provides system instructions used in chat mode.
	LlamaCppPrompt string `mapstructure:"llama_cpp_prompt" json:"llama_cpp_prompt"`
	// TTSEnabled toggles text-to-speech confirmations.
	TTSEnabled bool `mapstructure:"tts_enabled" json:"tts_enabled"`
	// SpeechAllowlist restricts allowed speech phrases and patterns.
	SpeechAllowlist []string `mapstructure:"speech_allowlist" json:"speech_allowlist"`
}

// MPRISConfig holds media player integration settings.
type MPRISConfig struct {
	// Enabled toggles MPRIS D-Bus monitoring on or off.
	Enabled bool `mapstructure:"enabled" json:"enabled"`
	// PollInterval is the D-Bus property polling frequency in milliseconds.
	// Minimum effective value is 500ms.
	PollInterval int `mapstructure:"poll_interval" json:"poll_interval"`
}

// Config holds the application's runtime settings.
// Environment values override matching keys from config.json.
//
// For distributed deployment, set ServerAddress to the relay server address
// (e.g., "10.0.0.1:3000") and run the binary with "connect" subcommand.
// Set AuthToken to protect serve/connect modes with token-based authentication.
type Config struct {
	// Port is the HTTP/WebSocket server port.
	Port uint16 `mapstructure:"port"`
	// NumJoysticks is the number of virtual joysticks to create.
	NumJoysticks uint8 `mapstructure:"numJoysticks"`
	// ServerAddress is the relay server address for distributed deployment (e.g., "10.0.0.1:3000").
	ServerAddress string `mapstructure:"server_address"`
	// AuthToken protects serve/connect modes with token-based authentication.
	// Empty string disables authentication (backward compatible).
	AuthToken string `mapstructure:"auth_token"`
	// Speech holds speech recognition settings.
	Speech SpeechConfig `mapstructure:"speech"`
	// MPRIS holds media player integration settings for Linux desktop environments.
	MPRIS MPRISConfig `mapstructure:"mpris"`
}

// Load reads configuration from the given path and applies OMNIPANEL_*
// environment overrides.
//
// Explicit env bindings are currently defined for:
// - OMNIPANEL_PORT
// - OMNIPANEL_NUMJOYSTICKS
// - OMNIPANEL_SERVER_ADDRESS
// - OMNIPANEL_AUTH_TOKEN
func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("json")

	v.SetEnvPrefix("OMNIPANEL")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	v.BindEnv("port")
	v.BindEnv("numJoysticks")
	v.BindEnv("server_address")
	v.BindEnv("auth_token")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the config to a JSON file if it does not already exist.
// The file write is delegated to Viper's SafeWriteConfig behavior.
func (c *Config) Save(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")

	v.Set("port", c.Port)
	v.Set("numJoysticks", c.NumJoysticks)
	v.Set("server_address", c.ServerAddress)
	v.Set("auth_token", c.AuthToken)
	v.Set("speech", c.Speech)
	v.Set("mpris", c.MPRIS)

	return v.SafeWriteConfig()
}

// FindConfigPath searches for config.json in order:
// 1. Current working directory
// 2. Directory next to the executable binary
// 3. Falls back to "config.json" (caller will get an error on Load)
func FindConfigPath() string {
	local := "config.json"
	if _, err := os.Stat(local); err == nil {
		return local
	}
	if exe, err := os.Executable(); err == nil {
		packaged := filepath.Join(filepath.Dir(exe), "config.json")
		if _, err := os.Stat(packaged); err == nil {
			return packaged
		}
	}
	return local
}

// FindUserPath searches for the user data directory in order:
// 1. "user" directory in current working directory
// 2. "user" directory next to the executable binary
// 3. Falls back to "user" (caller handles missing directory)
func FindUserPath() string {
	local := "user"
	if info, err := os.Stat(local); err == nil && info.IsDir() {
		return local
	}
	if exe, err := os.Executable(); err == nil {
		packaged := filepath.Join(filepath.Dir(exe), "user")
		if info, err := os.Stat(packaged); err == nil && info.IsDir() {
			return packaged
		}
	}
	return local
}
