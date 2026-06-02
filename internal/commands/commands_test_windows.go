//go:build windows

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installFakeSh(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := `@echo off
if "%1"=="-c" (
  if "%2"=="fail" exit /b 1
  if "%2"=="empty" exit /b 0
  echo %2
  exit /b 0
)
exit /b 1
`
	path := filepath.Join(dir, "sh.cmd")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatalf("write fake sh: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestExecuteShell(t *testing.T) {
	installFakeSh(t)

	success, out := ExecuteShell("hello")
	if !success || out != "hello" {
		t.Fatalf("expected successful shell call, got success=%v out=%q", success, out)
	}

	success, out = ExecuteShell("empty")
	if !success || out != "(no output)" {
		t.Fatalf("expected no-output placeholder, got success=%v out=%q", success, out)
	}

	t.Setenv("PATH", t.TempDir())
	success, out = ExecuteShell("fail")
	if success || !strings.Contains(out, "Failed to execute command") {
		t.Fatalf("expected failure message, got success=%v out=%q", success, out)
	}
}

