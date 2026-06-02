package relay

import (
	"encoding/json"
	"testing"
)

func TestHostMessageParsingAndMPRISResponse(t *testing.T) {
	s := &RelayServer{pendingRequests: make(map[string]chan map[string]any)}

	// invalid JSON should be ignored without panic
	s.handleHostMessage([]byte("{bad"))

	// host-register and heartbeat without host connection should be safe no-ops
	s.handleHostMessage([]byte(`{"type":"host-register"}`))
	s.handleHostMessage([]byte(`{"type":"heartbeat"}`))

	ch := s.registerPendingRequest("req-1")
	msg := map[string]any{
		"type": "mpris-response",
		"data": map[string]any{
			"request_id": "req-1",
			"payload":    map[string]any{"ok": true},
		},
	}
	raw, _ := json.Marshal(msg)
	s.handleHostMessage(raw)

	resp := <-ch
	if resp["ok"] != true {
		t.Fatalf("unexpected pending response payload: %v", resp)
	}

	// malformed response payloads
	s.handleMPRISResponse(map[string]json.RawMessage{"data": json.RawMessage(`{bad`)})
	s.handleMPRISResponse(map[string]json.RawMessage{"data": json.RawMessage(`{"request_id":""}`)})
}

func TestForwardNoHostAndBroadcastHelpers(t *testing.T) {
	s := &RelayServer{}

	// invalid browser message parse
	s.forwardToHost([]byte("{bad"))
	// parsed but no host
	s.forwardToHost([]byte(`{"type":"x"}`))
	// binary with no host
	s.forwardToHostBinary([]byte{1, 2, 3})

	// marshal failures are not expected with plain values; still cover helper call paths
	s.browsers = nil
	s.broadcastToBrowsers(map[string]any{"type": "x"})
	s.broadcastToBrowsersJSON(map[string]json.RawMessage{"type": json.RawMessage(`"x"`)})
	s.broadcastToBrowsersBinary([]byte("{}"))
}

