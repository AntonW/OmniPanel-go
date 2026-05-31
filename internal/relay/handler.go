// WebSocket connection handling for the relay server.
//
// Connections are distinguished by the "type" query parameter:
//   - "host": Host agent connection (1:1, second host is rejected)
//   - any other value (or empty): Browser client connection (multiple allowed)
//
// Authentication:
//
// Host connections are validated against the configured auth_token via the
// "token" query parameter. If the token is configured but not provided or
// invalid, the connection is rejected with a WebSocket close message.
// Browser connections are protected by HTTP middleware (auth.Middleware)
// applied in server.go, which requires the token for all HTTP requests
// including the WebSocket upgrade.
//
// Message flow:
//   - Browser → Server → Host: input commands, speech config, RSS config, etc.
//   - Host → Server → Browsers: data updates, speech results, command results, etc.
//   - Control messages: host-register, heartbeat/heartbeat-ack
//   - MPRIS forwarding: mpris-request (server→host), mpris-response (host→server)
//
// MPRIS Request-Response Pattern:
//
// In distributed deployments (serve + connect mode), the relay server forwards
// MPRIS HTTP API requests to the host agent via WebSocket. The server generates
// a unique request ID, sends an "mpris-request" message, and waits for the
// matching "mpris-response" message. The response is delivered to a pending
// request channel that the HTTP handler is blocking on. This synchronous pattern
// enables the relay server to proxy MPRIS endpoints without running its own
// MPRIS watcher. Cover art is base64-encoded for transport over WebSocket.
package relay

import (
	"encoding/json"
	"log/slog"
	"time"

	ws "github.com/gofiber/contrib/websocket"

	"omnipanel-go/internal/auth"
)

// handleWS manages a WebSocket connection, distinguishing between browser and host.
func (s *RelayServer) handleWS(c *ws.Conn) {
	connType := c.Query("type", "")

	if connType == "host" {
		s.handleHostConn(c)
	} else {
		s.handleBrowserConn(c)
	}
}

// handleHostConn manages a single host agent connection (1:1).
// When a host connects, its IP address is captured via c.IP() and broadcast
// to all browser clients as a log-event message ("Host connected: <IP>").
// On disconnect, the IP is broadcast again ("Host disconnected: <IP>").
// If a second host attempts to connect, it is rejected with a WebSocket
// close message and the connection is immediately terminated.
func (s *RelayServer) handleHostConn(c *ws.Conn) {
	token := c.Query("token", "")
	if !auth.ValidateToken(s.config.AuthToken, token) {
		slog.Warn("Rejecting host connection: invalid token")
		c.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.ClosePolicyViolation, "unauthorized"))
		c.Close()
		return
	}

	s.hostMu.Lock()
	if s.hostConn != nil {
		s.hostMu.Unlock()
		slog.Warn("Rejecting second host connection")
		c.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.ClosePolicyViolation, "host already connected"))
		c.Close()
		return
	}
	s.hostConn = c
	s.hostMu.Unlock()

	hostIP := c.IP()
	slog.Info("Host agent connected", "ip", hostIP)
	s.broadcastToBrowsers(map[string]any{
		"type":      "log-event",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"data":      "Host connected: " + hostIP,
	})

	defer func() {
		s.hostMu.Lock()
		s.hostConn = nil
		hostIP := c.IP()
		s.hostMu.Unlock()
		slog.Info("Host agent disconnected", "ip", hostIP)
		s.broadcastToBrowsers(map[string]any{
			"type":      "log-event",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			"data":      "Host disconnected: " + hostIP,
		})
	}()

	for {
		msgType, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		if msgType == ws.BinaryMessage {
			s.broadcastToBrowsersBinary(msg)
		} else {
			s.handleHostMessage(msg)
		}
	}
}

// handleHostMessage processes messages from the host agent.
func (s *RelayServer) handleHostMessage(raw []byte) {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.Warn("Failed to parse host message", "error", err)
		return
	}

	msgType := ""
	if t, ok := parsed["type"]; ok {
		json.Unmarshal(t, &msgType)
	}

	switch msgType {
	case "host-register":
		slog.Info("Host registered")
		return
	case "heartbeat":
		ack := map[string]any{"type": "heartbeat-ack"}
		if data, err := json.Marshal(ack); err == nil {
			s.hostMu.Lock()
			if s.hostConn != nil {
				s.hostConn.WriteMessage(ws.TextMessage, data)
			}
			s.hostMu.Unlock()
		}
		return
	case "mpris-response":
		s.handleMPRISResponse(parsed)
		return
	}

	s.broadcastToBrowsersJSON(parsed)
}

// handleMPRISResponse processes an "mpris-response" message from the host agent.
// It extracts the request_id and payload from the message, then delivers the
// payload to the pending request channel registered by sendMPRISRequest.
// This unblocks the HTTP handler that was waiting for the response.
// If the request_id is missing or doesn't match any pending request, the response
// is silently dropped.
func (s *RelayServer) handleMPRISResponse(parsed map[string]json.RawMessage) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(parsed["data"], &data); err != nil {
		slog.Warn("Failed to parse mpris-response data", "error", err)
		return
	}

	var requestID string
	if raw, ok := data["request_id"]; ok {
		json.Unmarshal(raw, &requestID)
	}

	if requestID == "" {
		slog.Warn("mpris-response missing request_id")
		return
	}

	var payload map[string]any
	if raw, ok := data["payload"]; ok {
		json.Unmarshal(raw, &payload)
	}

	s.completePendingRequest(requestID, payload)
}

// handleBrowserConn manages a browser client connection.
func (s *RelayServer) handleBrowserConn(c *ws.Conn) {
	s.browserMu.Lock()
	s.browsers[c] = struct{}{}
	s.browserMu.Unlock()

	clientIP := c.IP()
	slog.Info("Browser client connected", "ip", clientIP)
	s.broadcastToBrowsers(map[string]any{
		"type":      "log-event",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"data":      "Client connected: " + clientIP,
	})

	defer func() {
		s.browserMu.Lock()
		delete(s.browsers, c)
		s.browserMu.Unlock()
		slog.Info("Browser client disconnected", "ip", clientIP)
		s.broadcastToBrowsers(map[string]any{
			"type":      "log-event",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			"data":      "Client disconnected: " + clientIP,
		})
	}()

	for {
		msgType, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		if msgType == ws.BinaryMessage {
			s.forwardToHostBinary(msg)
		} else {
			s.forwardToHost(msg)
		}
	}
}

// forwardToHost sends a message from browser to host.
func (s *RelayServer) forwardToHost(raw []byte) {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.Warn("Failed to parse browser message", "error", err)
		return
	}

	msgType := ""
	if t, ok := parsed["type"]; ok {
		json.Unmarshal(t, &msgType)
	}

	s.hostMu.Lock()
	hostConn := s.hostConn
	s.hostMu.Unlock()

	if hostConn == nil {
		slog.Warn("No host connected, dropping message", "type", msgType)
		return
	}

	if err := hostConn.WriteMessage(ws.TextMessage, raw); err != nil {
		slog.Error("Failed to forward to host", "error", err)
	}
}

// forwardToHostBinary sends binary data from browser to host.
func (s *RelayServer) forwardToHostBinary(data []byte) {
	s.hostMu.Lock()
	hostConn := s.hostConn
	s.hostMu.Unlock()

	if hostConn == nil {
		slog.Warn("No host connected, dropping audio data")
		return
	}

	if err := hostConn.WriteMessage(ws.BinaryMessage, data); err != nil {
		slog.Error("Failed to forward audio to host", "error", err)
	}
}

// broadcastToBrowsers sends a JSON message to all browser clients.
func (s *RelayServer) broadcastToBrowsers(msg map[string]any) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Warn("Failed to marshal broadcast message", "error", err)
		return
	}
	s.broadcastToBrowsersBinary(data)
}

// broadcastToBrowsersJSON sends pre-parsed JSON to all browser clients.
func (s *RelayServer) broadcastToBrowsersJSON(msg map[string]json.RawMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Warn("Failed to marshal broadcast message", "error", err)
		return
	}
	s.broadcastToBrowsersBinary(data)
}

// broadcastToBrowsersBinary sends raw binary data to all browser clients.
func (s *RelayServer) broadcastToBrowsersBinary(data []byte) {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	for conn := range s.browsers {
		conn.WriteMessage(ws.TextMessage, data)
	}
}
