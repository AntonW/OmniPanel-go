package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newTestApp(token string) *fiber.App {
	app := fiber.New()
	app.Use(Middleware(token))
	app.Get("/ok", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	return app
}

func doReq(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	return resp
}

func TestMiddleware_AuthDisabled(t *testing.T) {
	app := newTestApp("")
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	resp := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMiddleware_QueryToken(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodGet, "/ok?token=secret", nil)
	resp := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMiddleware_BearerToken(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMiddleware_Unauthorized(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	resp := doReq(t, app, req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body error = %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Fatalf("error body = %q, want unauthorized", body["error"])
	}
}

func TestValidateToken(t *testing.T) {
	if !ValidateToken("", "anything") {
		t.Fatalf("ValidateToken disabled auth should return true")
	}
	if !ValidateToken("secret", "secret") {
		t.Fatalf("ValidateToken matching token should return true")
	}
	if ValidateToken("secret", "wrong") {
		t.Fatalf("ValidateToken mismatching token should return false")
	}
}

