// WebSocket connection handling for the relay server.
//
// Connections are distinguished by the "type" query parameter:
//   - "host": Host agent connection (1:1, second host is rejected)
//   - any other value (or empty): Browser client connection (multiple allowed)
//
// Message flow:
//   - Browser → Server → Host: input commands, speech config, RSS config, etc.
//   - Host → Server → Browsers: data updates, speech results, command results, etc.
//   - Control messages: host-register, heartbeat/heartbeat-ack
package relay

import (
	"encoding/json"
	"log/slog"
	"time"

	ws "github.com/gofiber/contrib/websocket"
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
func (s *RelayServer) handleHostConn(c *ws.Conn) {
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

	slog.Info("Host agent connected")
	s.broadcastToBrowsers(map[string]any{
		"type":      "log-event",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"data":      "Host connected",
	})

	defer func() {
		s.hostMu.Lock()
		s.hostConn = nil
		s.hostMu.Unlock()
		slog.Info("Host agent disconnected")
		s.broadcastToBrowsers(map[string]any{
			"type":      "log-event",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			"data":      "Host disconnected",
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
	}

	s.broadcastToBrowsersJSON(parsed)
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
