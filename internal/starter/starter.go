// Package starter copies default user content (blocks, panels, themes, assets,
// speech commands) into a mounted user directory on first run. It is used by
// the Docker serve container to populate an empty volume with starter files so
// users get a working setup immediately without manual file copying.
//
// Init checks whether the target user directory exists and is empty. If so, it
// copies all files from the embedded starter directory into it. If the directory
// already contains files, Init does nothing to preserve existing user data.
//
// The starter directory is searched in order:
//  1. $KO_DATA_PATH/starter (ko container runtime)
//  2. /var/run/ko/starter (ko default mount point)
//  3. starter (local development)
//  4. ./starter (local development, relative)
package starter

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// Init copies starter files into userPath if the directory is empty or does not
// exist. It searches for the starter directory using findStarterDir and copies
// all files and subdirectories recursively. If userPath already contains files,
// Init returns immediately without making changes.
func Init(userPath string) {
	starterDir := findStarterDir()
	if starterDir == "" {
		return
	}

	empty, err := isDirEmpty(userPath)
	if err != nil {
		slog.Info("Starter files: user directory does not exist, creating and populating", "path", userPath)
		if err := os.MkdirAll(userPath, 0755); err != nil {
			slog.Error("Starter files: failed to create user directory", "error", err)
			return
		}
		empty = true
	}

	if !empty {
		return
	}

	slog.Info("Starter files: populating user directory", "from", starterDir, "to", userPath)
	if err := copyDir(starterDir, userPath); err != nil {
		slog.Error("Starter files: failed to copy starter files", "error", err)
	} else {
		slog.Info("Starter files: done")
	}
}

func findStarterDir() string {
	candidates := []string{
		filepath.Join(os.Getenv("KO_DATA_PATH"), "starter"),
		"/var/run/ko/starter",
		"starter",
		"./starter",
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.ReadDir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
