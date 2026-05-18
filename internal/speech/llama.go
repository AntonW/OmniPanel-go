// llama-cpp STT engine implementation.
// Uses llama-cpp-server's OpenAI-compatible API for speech recognition.
// Supports two modes:
//   - "transcriptions": Whisper-compatible /v1/audio/transcriptions endpoint
//   - "chat": OpenAI chat completions with audio content (multimodal models)
//
// PCM audio is wrapped in WAV format for API compatibility.
// Chat mode uses data URIs to embed audio in JSON requests.
package speech

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"

	"omnipanel-go/internal/config"
)

// llamaEngine implements STTEngine using llama-cpp-server's OpenAI-compatible API.
// Supports two modes:
//   - "transcriptions": Whisper-compatible /v1/audio/transcriptions endpoint
//   - "chat": OpenAI chat completions /v1/chat/completions with audio content (multimodal models like gemma)
type llamaEngine struct {
	url    string
	apiKey string
	mode   string
	model  string
	prompt string
	client *http.Client
}

// newLlamaEngine creates a llama-cpp STT engine.
func newLlamaEngine(cfg *config.SpeechConfig) *llamaEngine {
	baseURL := cfg.LlamaCppURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	mode := cfg.LlamaCppAPIMode
	if mode == "" {
		mode = "transcriptions"
	}

	model := cfg.LlamaCppModel
	if model == "" {
		model = "gemma-3-4b"
	}

	prompt := cfg.LlamaCppPrompt
	if prompt == "" {
		prompt = "Transcribe the following audio to text. Only output the transcribed text, nothing else."
	}

	return &llamaEngine{
		url:    baseURL,
		apiKey: cfg.LlamaCppAPIKey,
		mode:   mode,
		model:  model,
		prompt: prompt,
		client: &http.Client{},
	}
}

// Recognize sends audio to llama-cpp-server for transcription.
// Audio should be PCM 16kHz mono 16-bit, which we encode as WAV for the API.
func (e *llamaEngine) Recognize(pcm []byte) (string, error) {
	switch e.mode {
	case "chat":
		return e.recognizeChat(pcm)
	case "transcriptions":
		return e.recognizeTranscriptions(pcm)
	default:
		return e.recognizeTranscriptions(pcm)
	}
}

// recognizeChat uses the /v1/chat/completions endpoint with audio content.
// This is for multimodal models like gemma-3 that accept audio in chat messages.
func (e *llamaEngine) recognizeChat(pcm []byte) (string, error) {
	wavData, err := pcmToWav(pcm, 16000)
	if err != nil {
		return "", fmt.Errorf("convert PCM to WAV: %w", err)
	}

	audioBase64 := base64.StdEncoding.EncodeToString(wavData)

	requestBody := map[string]any{
		"model": e.model,
		"messages": []map[string]any{
			{
				"role":    "system",
				"content": e.prompt,
			},
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "audio_url",
						"audio_url": map[string]string{
							"url": "data:audio/wav;base64," + audioBase64,
						},
					},
				},
			},
		},
		"max_tokens":  512,
		"temperature": 0.0,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := e.url + "/v1/chat/completions"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llama-cpp chat error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	text := result.Choices[0].Message.Content
	slog.Info("llama-cpp chat transcription", "text", text)
	return text, nil
}

// recognizeTranscriptions uses the /v1/audio/transcriptions endpoint (Whisper-compatible).
func (e *llamaEngine) recognizeTranscriptions(pcm []byte) (string, error) {
	wavData, err := pcmToWav(pcm, 16000)
	if err != nil {
		return "", fmt.Errorf("convert PCM to WAV: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return "", fmt.Errorf("write audio data: %w", err)
	}

	modelName := e.model
	if modelName == "" {
		modelName = "whisper"
	}

	if err := writer.WriteField("model", modelName); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", fmt.Errorf("write format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	endpoint := e.url + "/v1/audio/transcriptions"
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llama-cpp transcription error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}

	slog.Info("llama-cpp transcription", "text", result.Text)
	return result.Text, nil
}

// Close releases resources (no-op for HTTP client).
func (e *llamaEngine) Close() {
	slog.Info("llama-cpp engine closed")
}

// SetGrammar is a no-op for the llama-cpp engine, which does not support
// grammar-constrained recognition. The method exists to satisfy the STTEngine
// interface.
func (e *llamaEngine) SetGrammar(grammar string) {
	slog.Debug("llama-cpp engine does not support grammar constraints")
}

// pcmToWav wraps raw PCM data in a WAV container.
func pcmToWav(pcm []byte, sampleRate int) ([]byte, error) {
	buf := &bytes.Buffer{}

	numChannels := 1
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := len(pcm)

	writeString(buf, "RIFF")
	writeUint32(buf, uint32(36+dataSize))
	writeString(buf, "WAVE")

	writeString(buf, "fmt ")
	writeUint32(buf, 16)
	writeUint16(buf, 1)
	writeUint16(buf, uint16(numChannels))
	writeUint32(buf, uint32(sampleRate))
	writeUint32(buf, uint32(byteRate))
	writeUint16(buf, uint16(blockAlign))
	writeUint16(buf, uint16(bitsPerSample))

	writeString(buf, "data")
	writeUint32(buf, uint32(dataSize))
	buf.Write(pcm)

	return buf.Bytes(), nil
}

func writeString(buf *bytes.Buffer, s string) {
	buf.WriteString(s)
}

func writeUint32(buf *bytes.Buffer, v uint32) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 24))
}

func writeUint16(buf *bytes.Buffer, v uint16) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
}
