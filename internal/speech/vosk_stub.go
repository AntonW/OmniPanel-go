//go:build !cgo

// Vosk STT engine stub for builds without CGO.
// When CGO is disabled (e.g., for fully static Linux builds or container images),
// the Vosk C library cannot be linked. This stub provides the same voskEngine
// interface so the rest of the code compiles, but all operations return errors.
//
// To enable Vosk STT, build with CGO_ENABLED=1 and ensure libvosk is available
// at runtime (libvosk.so on Linux, vosk.dll on Windows, libvosk.dylib on macOS).
package speech

import (
	"fmt"

	"omnipanel-go/internal/config"
)

type voskEngine struct{}

func newVoskEngine(cfg *config.SpeechConfig, grammar string) (*voskEngine, error) {
	return nil, fmt.Errorf("Vosk STT requires CGO (build with CGO_ENABLED=1)")
}

func (e *voskEngine) Recognize(pcm []byte) (string, error) {
	return "", fmt.Errorf("Vosk STT requires CGO")
}

func (e *voskEngine) SetGrammar(grammar string) {}

func (e *voskEngine) Close() {}
