package starter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindStarterDir_KODataPathPreferred(t *testing.T) {
	root := t.TempDir()
	koStarter := filepath.Join(root, "starter")
	if err := os.MkdirAll(koStarter, 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	t.Setenv("KO_DATA_PATH", root)

	got := findStarterDir()
	if got != koStarter {
		t.Fatalf("findStarterDir() = %q, want %q", got, koStarter)
	}
}

func TestIsDirEmpty(t *testing.T) {
	empty := t.TempDir()
	ok, err := isDirEmpty(empty)
	if err != nil || !ok {
		t.Fatalf("isDirEmpty(empty) = (%v,%v), want (true,nil)", ok, err)
	}

	nonEmpty := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmpty, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	ok, err = isDirEmpty(nonEmpty)
	if err != nil || ok {
		t.Fatalf("isDirEmpty(nonEmpty) = (%v,%v), want (false,nil)", ok, err)
	}

	_, err = isDirEmpty(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatalf("isDirEmpty(missing) should return error")
	}
}

func TestCopyFileAndCopyDir(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	if err := os.MkdirAll(filepath.Join(srcRoot, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "sub", "b.txt"), []byte("B"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if err := copyDir(srcRoot, dstRoot); err != nil {
		t.Fatalf("copyDir error = %v", err)
	}

	dataA, err := os.ReadFile(filepath.Join(dstRoot, "a.txt"))
	if err != nil || string(dataA) != "A" {
		t.Fatalf("copied a.txt = %q, err=%v", string(dataA), err)
	}
	dataB, err := os.ReadFile(filepath.Join(dstRoot, "sub", "b.txt"))
	if err != nil || string(dataB) != "B" {
		t.Fatalf("copied sub/b.txt = %q, err=%v", string(dataB), err)
	}

	copyDst := filepath.Join(dstRoot, "single.txt")
	if err := copyFile(filepath.Join(srcRoot, "a.txt"), copyDst); err != nil {
		t.Fatalf("copyFile error = %v", err)
	}
	dataSingle, err := os.ReadFile(copyDst)
	if err != nil || string(dataSingle) != "A" {
		t.Fatalf("copied single.txt = %q, err=%v", string(dataSingle), err)
	}

	if err := copyFile(filepath.Join(srcRoot, "missing.txt"), filepath.Join(dstRoot, "x")); err == nil {
		t.Fatalf("copyFile missing src should error")
	}
}

func TestInit_PopulatesMissingAndSkipsNonEmpty(t *testing.T) {
	base := t.TempDir()
	koRoot := filepath.Join(base, "ko")
	starterUser := filepath.Join(koRoot, "starter", "user")
	if err := os.MkdirAll(filepath.Join(starterUser, "blocks"), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(starterUser, "blocks", "default.txt"), []byte("starter"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	t.Setenv("KO_DATA_PATH", koRoot)

	// Case 1: target missing -> should be created and populated.
	target := filepath.Join(base, "user-target")
	Init(target)
	data, err := os.ReadFile(filepath.Join(target, "blocks", "default.txt"))
	if err != nil || string(data) != "starter" {
		t.Fatalf("Init populate failed: data=%q err=%v", string(data), err)
	}

	// Case 2: target non-empty -> should be left untouched.
	if err := os.WriteFile(filepath.Join(target, "custom.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(starterUser, "blocks", "default.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	Init(target)
	still, err := os.ReadFile(filepath.Join(target, "custom.txt"))
	if err != nil || string(still) != "keep" {
		t.Fatalf("Init should keep existing files: data=%q err=%v", string(still), err)
	}
	copied, err := os.ReadFile(filepath.Join(target, "blocks", "default.txt"))
	if err != nil || string(copied) != "starter" {
		t.Fatalf("Init should not overwrite populated target: data=%q err=%v", string(copied), err)
	}
}

func TestInit_NoStarterDir_DoesNothing(t *testing.T) {
	base := t.TempDir()
	t.Setenv("KO_DATA_PATH", filepath.Join(base, "missing-ko"))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	target := filepath.Join(base, "user-target")
	Init(target)
	if _, err := os.Stat(target); err == nil {
		t.Fatalf("Init should do nothing when no starter dir exists")
	}
}

func TestInit_MkdirAllFailure(t *testing.T) {
	base := t.TempDir()
	koRoot := filepath.Join(base, "ko")
	starterUser := filepath.Join(koRoot, "starter", "user")
	if err := os.MkdirAll(starterUser, 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	t.Setenv("KO_DATA_PATH", koRoot)

	// target is a file, so MkdirAll(target) must fail.
	targetFile := filepath.Join(base, "user-target-file")
	if err := os.WriteFile(targetFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	Init(targetFile)
	if info, err := os.Stat(targetFile); err != nil || info.IsDir() {
		t.Fatalf("target file should remain a file, err=%v", err)
	}
}

func TestInit_CopyDirFailure(t *testing.T) {
	base := t.TempDir()
	koRoot := filepath.Join(base, "ko")
	// Create only starter root, but no starter/user subtree.
	if err := os.MkdirAll(filepath.Join(koRoot, "starter"), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	t.Setenv("KO_DATA_PATH", koRoot)

	target := filepath.Join(base, "target")
	Init(target)

	// Target is created before copyDir is attempted; missing starter/user triggers copy error branch.
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("target directory should exist even when copy fails, err=%v", err)
	}
}

func TestFindStarterDir_NoCandidates(t *testing.T) {
	base := t.TempDir()
	t.Setenv("KO_DATA_PATH", filepath.Join(base, "none"))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error = %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir error = %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if got := findStarterDir(); got != "" {
		t.Fatalf("findStarterDir() = %q, want empty", got)
	}
}

func TestCopyDirAndCopyFile_ErrorPaths(t *testing.T) {
	// copyDir should error when src is missing.
	if err := copyDir(filepath.Join(t.TempDir(), "missing"), t.TempDir()); err == nil {
		t.Fatalf("copyDir missing src should fail")
	}

	// copyDir should error when destination root cannot be created as directory.
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	dstFile := filepath.Join(t.TempDir(), "dst-file")
	if err := os.WriteFile(dstFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := copyDir(src, dstFile); err == nil {
		t.Fatalf("copyDir should fail when dst root is a file")
	}

	// copyFile should error when destination directory does not exist.
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := copyFile(filepath.Join(src, "f.txt"), filepath.Join(t.TempDir(), "missing", "x.txt")); err == nil {
		t.Fatalf("copyFile should fail when destination directory is missing")
	}
}


