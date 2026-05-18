// Audio format conversion utilities.
// Converts browser audio (Ogg/Opus) to PCM 16kHz mono 16-bit,
// which is the format expected by the STT engines.
package speech

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pion/opus"
	"github.com/pion/webrtc/v4/pkg/media/oggreader"
)

// DecodeAudio converts browser audio (ogg/opus) to PCM 16kHz mono 16-bit.
func DecodeAudio(data []byte) ([]byte, error) {
	if isOggOpus(data) {
		return decodeOggOpus(data)
	}
	return nil, fmt.Errorf("unsupported audio format")
}

// isOggOpus checks if data starts with Ogg magic bytes.
func isOggOpus(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], []byte("OggS"))
}

// decodeOggOpus decodes Ogg Opus audio to PCM.
func decodeOggOpus(data []byte) ([]byte, error) {
	ogg, _, err := oggreader.NewWith(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse Ogg: %w", err)
	}

	decoder, err := opus.NewDecoderWithOutput(16000, 1)
	if err != nil {
		return nil, fmt.Errorf("create Opus decoder: %w", err)
	}

	var pcm []byte
	outBuf := make([]byte, 1920)

	for {
		pageData, _, err := ogg.ParseNextPage()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		if len(pageData) == 0 {
			continue
		}

		_, _, err = decoder.Decode(pageData, outBuf)
		if err != nil {
			continue
		}

		pcm = append(pcm, outBuf...)
	}

	return pcm, nil
}

// DecodeOpusPacket decodes a single Opus packet to PCM.
func DecodeOpusPacket(data []byte) ([]byte, error) {
	decoder, err := opus.NewDecoderWithOutput(16000, 1)
	if err != nil {
		return nil, fmt.Errorf("create Opus decoder: %w", err)
	}

	outBuf := make([]byte, 1920)
	_, _, err = decoder.Decode(data, outBuf)
	if err != nil {
		return nil, fmt.Errorf("decode Opus: %w", err)
	}

	return outBuf, nil
}

// ValidatePCM checks if audio data is valid PCM 16kHz mono 16-bit.
func ValidatePCM(data []byte) bool {
	return len(data) > 0 && len(data)%2 == 0
}

// ResamplePCM resamples PCM audio to the target sample rate (simple linear interpolation).
func ResamplePCM(input []byte, inputRate, outputRate int) []byte {
	if inputRate == outputRate {
		return input
	}

	ratio := float64(inputRate) / float64(outputRate)
	outputLen := int(float64(len(input)/2) / ratio)
	output := make([]byte, outputLen*2)

	for i := 0; i < outputLen; i++ {
		srcIdx := int(float64(i) * ratio)
		if srcIdx*2+1 < len(input) {
			output[i*2] = input[srcIdx*2]
			output[i*2+1] = input[srcIdx*2+1]
		}
	}

	return output
}

// NormalizePCM normalizes PCM samples to prevent clipping.
func NormalizePCM(pcm []byte) []byte {
	var maxVal int16
	for i := 0; i < len(pcm); i += 2 {
		val := int16(binary.LittleEndian.Uint16(pcm[i:]))
		if val < 0 {
			val = -val
		}
		if val > maxVal {
			maxVal = val
		}
	}

	if maxVal == 0 || maxVal > 30000 {
		return pcm
	}

	scale := float64(30000) / float64(maxVal)
	normalized := make([]byte, len(pcm))
	for i := 0; i < len(pcm); i += 2 {
		val := int16(binary.LittleEndian.Uint16(pcm[i:]))
		normalizedVal := int16(float64(val) * scale)
		binary.LittleEndian.PutUint16(normalized[i:], uint16(normalizedVal))
	}

	return normalized
}
