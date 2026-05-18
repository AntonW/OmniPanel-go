// Package config handles loading, saving, and discovering the application's
// configuration file (config.json). It uses Viper for configuration management,
// which automatically supports environment variables with the OMNIPANEL_ prefix.
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
	Enabled           bool     `mapstructure:"enabled" json:"enabled"`
	RecordingLoc      string   `mapstructure:"recording_location" json:"recording_location"`
	TriggerMode       string   `mapstructure:"trigger_mode" json:"trigger_mode"`
	WakeWord          string   `mapstructure:"wake_word" json:"wake_word"`
	WakeWordListenSec int      `mapstructure:"wake_word_listen_sec" json:"wake_word_listen_sec"`
	STTEngine         string   `mapstructure:"stt_engine" json:"stt_engine"`
	VoskModelPath     string   `mapstructure:"vosk_model_path" json:"vosk_model_path"`
	LlamaCppURL       string   `mapstructure:"llama_cpp_url" json:"llama_cpp_url"`
	LlamaCppAPIKey    string   `mapstructure:"llama_cpp_api_key" json:"llama_cpp_api_key"`
	LlamaCppAPIMode   string   `mapstructure:"llama_cpp_api_mode" json:"llama_cpp_api_mode"`
	LlamaCppModel     string   `mapstructure:"llama_cpp_model" json:"llama_cpp_model"`
	LlamaCppPrompt    string   `mapstructure:"llama_cpp_prompt" json:"llama_cpp_prompt"`
	TTSEnabled        bool     `mapstructure:"tts_enabled" json:"tts_enabled"`
	SpeechAllowlist   []string `mapstructure:"speech_allowlist" json:"speech_allowlist"`
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
// Environment variables (OMNIPANEL_PANEL, OMNIPANEL_PORT, OMNIPANEL_NUMJOYSTICKS)
// override values from the config file.
type Config struct {
	Port         uint16       `mapstructure:"port"`
	NumJoysticks uint8        `mapstructure:"numJoysticks"`
	Speech       SpeechConfig `mapstructure:"speech"`
	// MPRIS holds media player integration settings for Linux desktop environments.
	MPRIS MPRISConfig `mapstructure:"mpris"`
}

// Load reads configuration using viper from the given path.
// Viper automatically binds environment variables with the OMNIPANEL_ prefix.
// Environment variables take precedence over config file values.
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

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the config to a JSON file with 4-space indentation.
// The file is created with permissions 0644 (owner rw, group/others r).
func (c *Config) Save(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")

	v.Set("port", c.Port)
	v.Set("numJoysticks", c.NumJoysticks)
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
