// Host microphone recording using malgo (cross-platform audio capture).
// Works on Linux (PulseAudio/ALSA), Windows (WASAPI), and macOS (CoreAudio)
// without platform-specific build tags.
package speech

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/gen2brain/malgo"
)

const (
	sampleRate  = 16000
	numChannels = 1
	format      = malgo.FormatS16
)

// HostRecorder captures audio from the system microphone using malgo.
type HostRecorder struct {
	device    *malgo.Device
	context   *malgo.AllocatedContext
	mu        sync.Mutex
	chunks    [][]byte
	recording bool
}

// NewHostRecorder creates a new host audio recorder.
func NewHostRecorder() (*HostRecorder, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("init malgo context: %w", err)
	}

	hr := &HostRecorder{
		context: ctx,
		chunks:  make([][]byte, 0),
	}

	return hr, nil
}

// Start begins recording from the default microphone.
func (hr *HostRecorder) Start() error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	if hr.recording {
		return fmt.Errorf("already recording")
	}

	hr.chunks = make([][]byte, 0)

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = format
	deviceConfig.Capture.Channels = numChannels
	deviceConfig.SampleRate = sampleRate
	deviceConfig.PeriodSizeInMilliseconds = 100

	sizeInBytes := uint32(malgo.SampleSizeInBytes(format))

	onRecv := func(output, input []byte, framecount uint32) {
		sampleCount := framecount * sizeInBytes
		if len(input) < int(sampleCount) {
			return
		}

		hr.mu.Lock()
		defer hr.mu.Unlock()
		if hr.recording {
			chunk := make([]byte, sampleCount)
			copy(chunk, input[:sampleCount])
			hr.chunks = append(hr.chunks, chunk)
		}
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecv,
	}

	device, err := malgo.InitDevice(hr.context.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return fmt.Errorf("init capture device: %w", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		return fmt.Errorf("start capture: %w", err)
	}

	hr.device = device
	hr.recording = true

	slog.Info("Host recording started", "samplerate", sampleRate, "channels", numChannels)
	return nil
}

// Stop stops recording and returns the captured audio as PCM.
// It releases the mutex before calling device.Stop() to avoid deadlock with
// the onRecv callback, which also acquires the mutex.
func (hr *HostRecorder) Stop() ([]byte, error) {
	hr.mu.Lock()
	if !hr.recording {
		hr.mu.Unlock()
		return nil, fmt.Errorf("not recording")
	}
	hr.recording = false

	device := hr.device
	hr.device = nil

	var pcm []byte
	for _, chunk := range hr.chunks {
		pcm = append(pcm, chunk...)
	}
	hr.chunks = nil
	hr.mu.Unlock()

	if device != nil {
		device.Stop()
		device.Uninit()
	}

	slog.Info("Host recording stopped", "bytes", len(pcm))
	return pcm, nil
}

// IsRecording returns whether recording is in progress.
func (hr *HostRecorder) IsRecording() bool {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	return hr.recording
}

// Close releases the recorder resources, including the malgo device and context.
// It releases the mutex before stopping the device to avoid deadlock with the
// onRecv callback, which also acquires the mutex.
func (hr *HostRecorder) Close() {
	hr.mu.Lock()
	device := hr.device
	hr.device = nil

	ctx := hr.context
	hr.context = nil
	hr.mu.Unlock()

	if device != nil {
		device.Stop()
		device.Uninit()
	}

	if ctx != nil {
		ctx.Free()
	}

	slog.Info("Host recorder closed")
}
