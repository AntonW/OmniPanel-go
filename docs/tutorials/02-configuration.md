# Chapter 2: Configuration System

## What This Package Does

The `config` package handles loading, saving, and discovering the application's configuration file (`config.json`). It uses [Viper](https://github.com/spf13/viper), a popular Go configuration library, which automatically supports environment variables, multiple config formats, and more. It also finds the user data directory where panels, blocks, assets, and speech commands live.

## The Config Struct

```go
// internal/config/config.go
type SpeechConfig struct {
    Enabled            bool     `mapstructure:"enabled" json:"enabled"`
    RecordingLoc       string   `mapstructure:"recording_location" json:"recording_location"`
    TriggerMode        string   `mapstructure:"trigger_mode" json:"trigger_mode"`
    WakeWord           string   `mapstructure:"wake_word" json:"wake_word"`
    STTEngine          string   `mapstructure:"stt_engine" json:"stt_engine"`
    VoskModelPath      string   `mapstructure:"vosk_model_path" json:"vosk_model_path"`
    LlamaCppURL        string   `mapstructure:"llama_cpp_url" json:"llama_cpp_url"`
    LlamaCppAPIKey     string   `mapstructure:"llama_cpp_api_key" json:"llama_cpp_api_key"`
    LlamaCppAPIMode    string   `mapstructure:"llama_cpp_api_mode" json:"llama_cpp_api_mode"`
    LlamaCppModel      string   `mapstructure:"llama_cpp_model" json:"llama_cpp_model"`
    LlamaCppPrompt     string   `mapstructure:"llama_cpp_prompt" json:"llama_cpp_prompt"`
    TTSEnabled         bool     `mapstructure:"tts_enabled" json:"tts_enabled"`
    SpeechAllowlist    []string `mapstructure:"speech_allowlist" json:"speech_allowlist"`
}

type Config struct {
    Port         uint16       `mapstructure:"port"`
    NumJoysticks uint8        `mapstructure:"numJoysticks"`
    Speech       SpeechConfig `mapstructure:"speech"`
}
```

> **Concept: Struct tags**
> The backtick strings like `` `mapstructure:"panel"` `` are **struct tags**. They're metadata attached to struct fields. Viper uses the `mapstructure` library to map config keys to Go struct fields. Without tags, Go would use the field name as-is (case-sensitive), which wouldn't match the lowercase JSON keys.

The types are chosen to match the data:
- `uint16` for port (valid range 0–65535)
- `uint8` for joystick count (valid range 0–255)
- `SpeechConfig` is a nested struct that groups all speech-related settings

## Loading Configuration with Viper

```go
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
```

Step by step:

1. `viper.New()` creates a new Viper instance
2. `SetConfigFile` and `SetConfigType` tell Viper where to find the config and its format
3. `SetEnvPrefix("OMNIPANEL")` sets the prefix for environment variables
4. `AutomaticEnv()` tells Viper to automatically read environment variables
5. `SetEnvKeyReplacer` handles key name transformations (e.g., `numJoysticks` → `NUMJOYSTICKS`)
6. `ReadInConfig()` loads the config file; we gracefully handle missing files
7. `BindEnv()` explicitly binds each config key to its environment variable
8. `Unmarshal()` populates the struct from Viper's merged configuration

> **Concept: Environment Variable Precedence**
> Viper follows a precedence order: environment variables > config file > defaults. This means `OMNIPANEL_PORT=8080` will override whatever port is set in `config.json`. This is useful for containerized deployments and different environments.

> **Concept: Error Type Assertion**
> `if _, ok := err.(viper.ConfigFileNotFoundError); !ok` uses a **type assertion** to check if the error is specifically a "config file not found" error. If it is, we ignore it (the app can run with env vars or defaults). If it's some other error (e.g., invalid JSON), we return it.

## Saving Configuration

```go
func (c *Config) Save(path string) error {
    v := viper.New()
    v.SetConfigFile(path)
    v.SetConfigType("json")

    v.Set("port", c.Port)
    v.Set("numJoysticks", c.NumJoysticks)
    v.Set("speech", c.Speech)

    return v.SafeWriteConfig()
}
```

> **Concept: Methods on types**
> `func (c *Config) Save(...)` defines a **method** on `*Config`. The `(c *Config)` part is the **receiver** — it's like `self` in Python or `this` in JavaScript. You call it as `cfg.Save(path)`.

The `Save` method now includes `v.Set("speech", c.Speech)` to persist speech configuration changes back to disk.

## The Config File

Here's what `config.json` looks like with all features:

```json
{
    "port": 3000,
    "numJoysticks": 4,
    "speech": {
        "enabled": false,
        "recording_location": "client",
        "trigger_mode": "push-to-talk",
        "wake_word": "omnipanel-go",
        "stt_engine": "vosk",
        "vosk_model_path": "",
        "llama_cpp_url": "http://localhost:8080",
        "llama_cpp_api_key": "",
        "llama_cpp_api_mode": "transcriptions",
        "llama_cpp_model": "",
        "llama_cpp_prompt": "",
        "tts_enabled": true,
        "speech_allowlist": []
    }
}
```

### Speech Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Master switch for speech recognition |
| `recording_location` | string | `"client"` | Where to record: `"client"` (browser) or `"host"` (server mic) |
| `trigger_mode` | string | `"push-to-talk"` | Activation: `"push-to-talk"` or `"wake-word"` |
| `wake_word` | string | `"omnipanel-go"` | Phrase to activate continuous listening |
| `stt_engine` | string | `"vosk"` | Backend: `"vosk"` (offline) or `"llama-cpp"` (HTTP API) |
| `vosk_model_path` | string | `""` | Path to Vosk model (auto-downloaded if empty) |
| `llama_cpp_url` | string | `"http://localhost:8080"` | llama-cpp-server endpoint |
| `llama_cpp_api_key` | string | `""` | API key for authentication |
| `llama_cpp_api_mode` | string | `"transcriptions"` | API mode: `"transcriptions"` or `"chat"` |
| `llama_cpp_model` | string | `""` | Model name for chat mode (e.g., `"gemma-3-4b"`) |
| `llama_cpp_prompt` | string | `""` | System prompt for chat mode |
| `tts_enabled` | boolean | `true` | Enable text-to-speech confirmations |
| `speech_allowlist` | array | `[]` | Allowed command patterns (empty = all allowed) |

## Environment Variables

The application can be configured via these environment variables:

| Variable | Type | Example | Description |
|----------|------|---------|-------------|
| `OMNIPANEL_PORT` | uint16 | `OMNIPANEL_PORT=8080` | HTTP server port |
| `OMNIPANEL_NUMJOYSTICKS` | uint8 | `OMNIPANEL_NUMJOYSTICKS=8` | Number of virtual joysticks |

Example usage:

```bash
# Run with custom port via environment variable
OMNIPANEL_PORT=8080 ./omnipanel-go

# Run with all config via environment variables
OMNIPANEL_PORT=9000 OMNIPANEL_NUMJOYSTICKS=2 ./omnipanel-go
```

## Finding the Config File

```go
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
```

This implements a **fallback strategy**:

1. Check for `config.json` in the current working directory
2. If not found, check next to the executable binary
3. If still not found, return `"config.json"` (the caller will get an error when trying to load it)

> **Concept: `os.Stat`**
> `os.Stat(path)` returns file info if the path exists, or an error if it doesn't. The idiom `if _, err := os.Stat(path); err == nil` checks existence without caring about the file info (discarded with `_`).

> **Concept: The blank identifier `_`**
> `_` discards a value you don't need. Go requires all declared variables to be used, so `_` is the escape hatch for "I know this function returns two values but I only care about one."

## Finding the User Directory

```go
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
```

Same fallback pattern, but with an extra check: `info.IsDir()` ensures the path is actually a directory, not a regular file.

## Key Takeaways

- Viper provides a unified configuration interface with automatic environment variable support
- Struct tags with `mapstructure` bridge Go field names and config keys
- Environment variables (`OMNIPANEL_*`) take precedence over config file values
- Fallback discovery (local → packaged → default) makes the app flexible for development and deployment
- Type assertions let you check specific error types for graceful handling
- Methods with receivers let you attach behavior to types
- Nested structs (`SpeechConfig`) group related settings logically

[← Back: Chapter 1](01-project-overview.md) · [Next: Chapter 3 →](03-state-and-concurrency.md)
