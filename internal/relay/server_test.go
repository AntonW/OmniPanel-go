package relay

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPendingRequestLifecycle(t *testing.T) {
	s := &RelayServer{pendingRequests: make(map[string]chan map[string]any)}

	ch := s.registerPendingRequest("id-1")
	if _, ok := s.pendingRequests["id-1"]; !ok {
		t.Fatal("pending request should be registered")
	}

	payload := map[string]any{"ok": true}
	s.completePendingRequest("id-1", payload)
	select {
	case got := <-ch:
		if got["ok"] != true {
			t.Fatalf("unexpected payload: %v", got)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected response payload on pending request channel")
	}

	if _, ok := s.pendingRequests["id-1"]; ok {
		t.Fatal("pending request should be removed after completion")
	}

	// No panic and no-op for unknown IDs.
	s.completePendingRequest("missing", map[string]any{"x": 1})
	s.cleanupPendingRequest("missing")
}

func TestGenerateRequestID(t *testing.T) {
	a := generateRequestID()
	if !strings.HasPrefix(a, "mpris-") {
		t.Fatalf("request id prefix mismatch: %q", a)
	}
	tsPart := strings.TrimPrefix(a, "mpris-")
	if _, err := strconv.ParseInt(tsPart, 10, 64); err != nil {
		t.Fatalf("request id timestamp is not numeric: %q", a)
	}
}

func TestSendMPRISRequestNoHost(t *testing.T) {
	s := &RelayServer{pendingRequests: make(map[string]chan map[string]any)}
	resp, err := s.sendMPRISRequest("/players", "GET", nil, nil)
	if err == nil {
		t.Fatalf("expected error when no host is connected, got response=%v", resp)
	}
	if len(s.pendingRequests) != 0 {
		t.Fatalf("pending requests should be cleaned up on error, got %d", len(s.pendingRequests))
	}
}


func TestDirectoryAndPanelHelpers(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.html"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write a.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write x.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "b.html"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write nested html: %v", err)
	}

	tree := buildDirectoryTree(root)
	if len(tree) != 2 {
		t.Fatalf("expected folder + html file in tree, got %d", len(tree))
	}

	panelsDir := filepath.Join(root, "panels")
	if err := os.MkdirAll(panelsDir, 0o755); err != nil {
		t.Fatalf("mkdir panels: %v", err)
	}
	if err := os.WriteFile(filepath.Join(panelsDir, "z.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write z.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(panelsDir, "a.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write a.json: %v", err)
	}

	panels := listPanels(panelsDir)
	if len(panels) != 2 || panels[0] != "a" || panels[1] != "z" {
		t.Fatalf("unexpected panel list: %v", panels)
	}

	out := filepath.Join(root, "sub", "out.json")
	if err := writeFile(out, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected written file at %s: %v", out, err)
	}
}
