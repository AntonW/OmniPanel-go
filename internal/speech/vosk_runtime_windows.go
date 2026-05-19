//go:build windows

package speech

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultVoskRuntimeTag = "v0.3.45"
	defaultVoskRuntimeURL = "https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-win64-0.3.45.zip"
)

// ensureVoskWindowsRuntime makes sure required Vosk runtime DLLs are available
// on Windows and reachable via PATH. It downloads and extracts runtime files if
// missing. runtimeURL can override the default Vosk release URL.
func ensureVoskWindowsRuntime(userPath string, runtimeURL string) error {
	if strings.TrimSpace(runtimeURL) == "" {
		runtimeURL = defaultVoskRuntimeURL
	}

	if hasVoskWindowsRuntime(userPath) {
		return ensureVoskRuntimeOnPath(userPath)
	}

	runtimeDir := filepath.Join(userPath, "speech-runtime", "vosk")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}

	zipPath := filepath.Join(runtimeDir, "vosk-win64.zip")
	slog.Info("Downloading Vosk runtime for Windows", "tag", defaultVoskRuntimeTag, "url", runtimeURL)
	if err := downloadFile(runtimeURL, zipPath); err != nil {
		return fmt.Errorf("download vosk runtime: %w", err)
	}
	defer os.Remove(zipPath)

	if err := extractSelectedVoskWindowsFiles(zipPath, runtimeDir); err != nil {
		return fmt.Errorf("extract vosk runtime files: %w", err)
	}

	if !hasVoskWindowsRuntime(userPath) {
		return fmt.Errorf("vosk runtime still missing after extraction")
	}

	slog.Info("Vosk runtime ready", "dir", runtimeDir)
	return ensureVoskRuntimeOnPath(userPath)
}

func hasVoskWindowsRuntime(userPath string) bool {
	required := []string{"libvosk.dll", "libstdc++-6.dll"}
	for _, name := range required {
		if _, err := findVoskWindowsRuntimeFile(userPath, name); err != nil {
			return false
		}
	}
	return true
}

// findVoskWindowsRuntimeFile searches common runtime locations for a DLL file.
func findVoskWindowsRuntimeFile(userPath, name string) (string, error) {
	candidates := []string{}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, name))
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, name))
	}

	if userPath != "" {
		candidates = append(candidates, filepath.Join(userPath, "speech-runtime", "vosk", name))
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("%s not found", name)
}

// ensureVoskRuntimeOnPath prepends the runtime directory containing libvosk.dll
// to PATH so the Vosk CGO binding can load native libraries.
func ensureVoskRuntimeOnPath(userPath string) error {
	libPath, err := findVoskWindowsRuntimeFile(userPath, "libvosk.dll")
	if err != nil {
		return err
	}

	runtimeDir := filepath.Dir(libPath)
	pathParts := strings.Split(os.Getenv("Path"), ";")
	for _, p := range pathParts {
		if strings.EqualFold(filepath.Clean(p), filepath.Clean(runtimeDir)) {
			return nil
		}
	}

	if err := os.Setenv("Path", runtimeDir+";"+os.Getenv("Path")); err != nil {
		return fmt.Errorf("prepend runtime directory to Path: %w", err)
	}

	slog.Info("Added Vosk runtime directory to PATH", "dir", runtimeDir)
	return nil
}

// extractSelectedVoskWindowsFiles extracts only required runtime DLLs from the
// downloaded Vosk runtime ZIP archive.
func extractSelectedVoskWindowsFiles(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	needed := map[string]bool{
		"libvosk.dll":    false,
		"libstdc++-6.dll": false,
	}

	for _, f := range r.File {
		base := strings.ToLower(filepath.Base(f.Name))
		if base != "libvosk.dll" && base != "libstdc++-6.dll" {
			continue
		}

		if f.FileInfo().IsDir() {
			continue
		}

		target := filepath.Join(destDir, filepath.Base(f.Name))
		if err := copyZipFileEntry(f, target); err != nil {
			return err
		}
		needed[strings.ToLower(filepath.Base(f.Name))] = true
	}

	for fileName, found := range needed {
		if !found {
			return fmt.Errorf("required file %s not found in runtime zip", fileName)
		}
	}

	return nil
}

// copyZipFileEntry copies a single ZIP file entry to a target file.
func copyZipFileEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
