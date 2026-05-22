//go:build !cgo

// Package speech host microphone recording stub for builds without CGO.
// When CGO is disabled (e.g., for fully static Linux builds or container images),
// the malgo library cannot be used. This stub provides the same HostRecorder
// interface so the rest of the code compiles, but all operations return errors.
//
// To enable host recording, build with CGO_ENABLED=1 and install the required
// audio development packages (e.g., libasound2-dev on Linux).
package speech

import (
	"fmt"
	"sync"
)

type HostRecorder struct {
	mu     sync.Mutex
	chunks [][]byte
}

func NewHostRecorder() (*HostRecorder, error) {
	return nil, fmt.Errorf("audio recording requires CGO (build with CGO_ENABLED=1)")
}

func (hr *HostRecorder) Start() error {
	return fmt.Errorf("audio recording requires CGO")
}

func (hr *HostRecorder) Stop() ([]byte, error) {
	return nil, fmt.Errorf("audio recording requires CGO")
}

func (hr *HostRecorder) IsRecording() bool {
	return false
}

func (hr *HostRecorder) Close() {}
