package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


func TestSubstituteParams(t *testing.T) {
	got := SubstituteParams("echo {name} {num}", json.RawMessage(`{"name":"World","num":42}`))
	if got != "echo World " {
		t.Fatalf("unexpected substitution: %q", got)
	}

	bad := SubstituteParams("echo {name}", json.RawMessage(`{invalid`))
	if bad != "echo {name}" {
		t.Fatalf("invalid json should keep template, got %q", bad)
	}
}


func TestExecuteHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/empty":
			w.WriteHeader(http.StatusNoContent)
		case "/err":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad"))
		}
	}))
	defer ts.Close()

	success, out := ExecuteHTTP("GET", ts.URL+"/ok", "")
	if !success || !strings.Contains(out, "Status: 200 OK") || !strings.Contains(out, "ok") {
		t.Fatalf("unexpected OK response: success=%v out=%q", success, out)
	}

	success, out = ExecuteHTTP("POST", ts.URL+"/empty", "body")
	if !success || !strings.Contains(out, "(no response body)") {
		t.Fatalf("unexpected empty response handling: success=%v out=%q", success, out)
	}

	success, out = ExecuteHTTP("GET", ts.URL+"/err", "")
	if success || !strings.Contains(out, "Status: 400 Bad Request") {
		t.Fatalf("unexpected error response handling: success=%v out=%q", success, out)
	}

	success, out = ExecuteHTTP("\n", "http://example.com", "")
	if success || !strings.Contains(out, "Failed to create request") {
		t.Fatalf("expected request creation error, got success=%v out=%q", success, out)
	}

	success, out = ExecuteHTTP("GET", "http://127.0.0.1:1/unreachable", "")
	if success || !strings.Contains(out, "HTTP request failed") {
		t.Fatalf("expected transport error, got success=%v out=%q", success, out)
	}
}

