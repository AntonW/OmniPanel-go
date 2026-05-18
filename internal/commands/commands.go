// Package commands provides execution backends for panel buttons:
//   - Shell commands: runs arbitrary commands via `sh -c`
//   - HTTP requests: makes HTTP calls and captures responses
//
// Both support parameter substitution, allowing panel buttons to be templated
// with {key} placeholders that are replaced at execution time.
//
// Parameter substitution only replaces string-typed values; numbers and
// booleans in the params JSON are ignored.
//
// See docs/tutorials/07-commands.md for a detailed walkthrough.
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
)

// SubstituteParams replaces {key} placeholders in a template string
// with values from a JSON object. Only string-typed values are substituted.
// Example: template="echo {name}", params={"name":"World"} → "echo World"
func SubstituteParams(template string, params json.RawMessage) string {
	result := template
	var paramsObj map[string]any
	if err := json.Unmarshal(params, &paramsObj); err != nil {
		return result
	}
	for key, value := range paramsObj {
		placeholder := "{" + key + "}"
		valueStr := ""
		if s, ok := value.(string); ok {
			valueStr = s
		}
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}
	return result
}

// ExecuteShell runs a command via `sh -c` and captures combined stdout+stderr.
// Returns (success, output). Success is true if the command exits with status 0.
func ExecuteShell(command string) (bool, string) {
	slog.Info("Executing shell command", "command", command)

	output, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		slog.Error("Shell execution failed", "error", err)
		return false, fmt.Sprintf("Failed to execute command: %s", err)
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		result = "(no output)"
	}

	return true, result
}

// ExecuteHTTP makes an HTTP request and captures the response.
// Returns (success, output). Success is true for 2xx status codes.
func ExecuteHTTP(method, url, body string) (bool, string) {
	slog.Info("Executing HTTP request", "method", method, "url", url)

	var req *http.Request
	var err error

	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("HTTP request failed", "error", err)
		return false, fmt.Sprintf("HTTP request failed: %s", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respStr := strings.TrimSpace(string(respBody))
	if respStr == "" {
		respStr = "(no response body)"
	}

	result := fmt.Sprintf("Status: %s\n\n%s", resp.Status, respStr)
	return resp.StatusCode >= 200 && resp.StatusCode < 300, result
}
