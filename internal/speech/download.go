package speech

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultModelURL  = "https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip"
	defaultModelName = "vosk-model-small-en-us-0.15"
)

// EnsureVoskModel checks if the model exists, downloads and extracts it if not.
// Returns the path to the model directory.
func EnsureVoskModel(modelPath string, userPath string) (string, error) {
	if modelPath != "" {
		if isValidVoskModel(modelPath) {
			return modelPath, nil
		}
		slog.Warn("Configured Vosk model path is invalid, attempting download", "path", modelPath)
	}

	modelsDir := filepath.Join(userPath, "speech-models")
	targetDir := filepath.Join(modelsDir, defaultModelName)

	if isValidVoskModel(targetDir) {
		slog.Info("Found existing Vosk model", "path", targetDir)
		return targetDir, nil
	}

	slog.Info("Downloading Vosk model", "url", defaultModelURL)
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return "", fmt.Errorf("create models directory: %w", err)
	}

	zipPath := filepath.Join(modelsDir, defaultModelName+".zip")
	if err := downloadFile(defaultModelURL, zipPath); err != nil {
		return "", fmt.Errorf("download model: %w", err)
	}
	defer os.Remove(zipPath)

	slog.Info("Extracting Vosk model", "zip", zipPath, "dest", modelsDir)
	if err := extractZip(zipPath, modelsDir); err != nil {
		return "", fmt.Errorf("extract model: %w", err)
	}

	if !isValidVoskModel(targetDir) {
		return "", fmt.Errorf("extracted model is invalid at %s", targetDir)
	}

	slog.Info("Vosk model downloaded and extracted", "path", targetDir)
	return targetDir, nil
}

// isValidVoskModel checks if a directory contains a valid Vosk model.
func isValidVoskModel(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	requiredFiles := []string{"am", "conf", "ivector", "graph"}
	for _, f := range requiredFiles {
		if _, err := os.Stat(filepath.Join(path, f)); err != nil {
			return false
		}
	}
	return true
}

// downloadFile downloads a file from url to dest with progress logging.
func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	slog.Info("Download complete", "file", dest, "bytes", written)
	return nil
}

// extractZip extracts a zip archive to the destination directory.
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// IsVoskAvailable checks if the Vosk shared library is loadable.
func IsVoskAvailable() bool {
	libNames := []string{"libvosk.so", "vosk.dll", "libvosk.dylib"}
	for _, lib := range libNames {
		if _, err := findLibrary(lib); err == nil {
			return true
		}
	}
	return false
}

// findLibrary searches common paths for a shared library.
func findLibrary(name string) (string, error) {
	searchPaths := []string{
		"/usr/local/lib",
		"/usr/lib",
		"/usr/local/lib/vosk",
		".",
	}

	for _, path := range searchPaths {
		fullPath := filepath.Join(path, name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}

	if os.Getenv("VOSK_LIB") != "" {
		fullPath := filepath.Join(os.Getenv("VOSK_LIB"), name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("library %s not found", name)
}

// NormalizeModelPath ensures the path uses OS-specific separators.
func NormalizeModelPath(path string) string {
	return filepath.Clean(path)
}

// ListAvailableModels returns a list of downloaded models in the models directory.
func ListAvailableModels(userPath string) []string {
	modelsDir := filepath.Join(userPath, "speech-models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var models []string
	for _, entry := range entries {
		if entry.IsDir() && isValidVoskModel(filepath.Join(modelsDir, entry.Name())) {
			models = append(models, entry.Name())
		}
	}
	return models
}

// CleanupOldModels removes models that don't match the current version.
func CleanupOldModels(userPath string, keepName string) error {
	modelsDir := filepath.Join(userPath, "speech-models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != keepName {
			fullPath := filepath.Join(modelsDir, entry.Name())
			slog.Info("Removing old model", "path", fullPath)
			if err := os.RemoveAll(fullPath); err != nil {
				slog.Warn("Failed to remove old model", "path", fullPath, "error", err)
			}
		}
	}

	zipFiles, _ := filepath.Glob(filepath.Join(modelsDir, "*.zip"))
	for _, zipFile := range zipFiles {
		if !strings.Contains(zipFile, keepName) {
			os.Remove(zipFile)
		}
	}

	return nil
}
