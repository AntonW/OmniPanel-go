//go:build cgo

// Vosk STT engine implementation using grammar-constrained recognition.
// Uses CGO to call the Vosk C library for offline speech recognition.
// The recognizer is created with NewRecognizerGrm, which restricts output
// to a predefined set of phrases (commands, aliases, wake word). This
// dramatically improves accuracy compared to open-ended recognition.
// Requires libvosk.so (Linux), vosk.dll (Windows), or libvosk.dylib (macOS)
// at runtime, and a GCC/Clang compiler at build time.
package speech

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"omnipanel-go/internal/config"

	vosk "github.com/alphacep/vosk-api/go"
)

// voskEngine implements STTEngine using the Vosk library with grammar-constrained
// recognition. The grammar field stores the current JSON array of allowed phrases.
type voskEngine struct {
	model      *vosk.VoskModel
	recognizer *vosk.VoskRecognizer
	sampleRate float64
	grammar    string
}

// newVoskEngine creates a Vosk STT engine with grammar-constrained recognition.
// The grammar parameter is a JSON array of allowed phrases (commands, aliases,
// wake word). The recognizer is created with NewRecognizerGrm to restrict output.
func newVoskEngine(cfg *config.SpeechConfig, grammar string) (*voskEngine, error) {
	modelPath := cfg.VoskModelPath
	if modelPath == "" {
		return nil, fmt.Errorf("vosk model path not configured")
	}

	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load Vosk model from %s: %w", modelPath, err)
	}

	recognizer, err := vosk.NewRecognizerGrm(model, 16000.0, grammar)
	if err != nil {
		model.Free()
		return nil, fmt.Errorf("failed to create Vosk recognizer: %w", err)
	}

	slog.Info("Vosk engine initialized", "model", modelPath, "grammar_phrases", strings.Count(grammar, ",")+1)

	return &voskEngine{
		model:      model,
		recognizer: recognizer,
		sampleRate: 16000,
		grammar:    grammar,
	}, nil
}

// Recognize transcribes PCM audio (16kHz, mono, 16-bit) to text.
func (e *voskEngine) Recognize(pcm []byte) (string, error) {
	if len(pcm) == 0 {
		return "", fmt.Errorf("empty audio data")
	}

	e.recognizer.Reset()

	accepted := e.recognizer.AcceptWaveform(pcm)

	var result string
	if accepted != 0 {
		result = e.recognizer.Result()
	} else {
		result = e.recognizer.FinalResult()
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return result, nil
	}

	return parsed.Text, nil
}

// SetGrammar updates the recognizer's grammar at runtime by calling SetGrm on
// the underlying Vosk recognizer. This allows new phrases to be recognized
// without recreating the engine. Logs a warning if the recognizer is nil.
func (e *voskEngine) SetGrammar(grammar string) {
	if e.recognizer != nil {
		e.recognizer.SetGrm(grammar)
		e.grammar = grammar
		slog.Info("Vosk grammar updated", "phrases_count", strings.Count(grammar, ",")+1)
	} else {
		slog.Warn("Cannot set grammar: recognizer not initialized")
	}
}

// Close releases Vosk resources.
func (e *voskEngine) Close() {
	if e.recognizer != nil {
		e.recognizer.Free()
		e.recognizer = nil
	}
	if e.model != nil {
		e.model.Free()
		e.model = nil
	}
	slog.Info("Vosk engine closed")
}
