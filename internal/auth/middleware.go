// Package auth provides authentication middleware for Fiber HTTP routes and
// WebSocket connections in serve and connect modes. It validates a shared token
// passed via query parameter (?token=xxx) or Authorization header (Bearer xxx).
//
// Authentication is optional: if the configured token is empty, all requests
// pass through without validation. This keeps the default mode (single machine)
// open while protecting distributed deployments.
//
// Usage:
//
//	// Apply as Fiber middleware to all HTTP routes:
//	app.Use(auth.Middleware(cfg.AuthToken))
//
//	// Validate token for WebSocket host connections:
//	if !auth.ValidateToken(cfg.AuthToken, c.Query("token", "")) {
//	    // reject connection
//	}
package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Middleware returns a Fiber middleware that validates the shared auth token.
// If token is empty, the middleware is a no-op (authentication disabled).
// Token is accepted via query parameter (?token=xxx) or Authorization header
// (Bearer xxx). Returns 401 Unauthorized if token is configured but not
// provided or invalid.
func Middleware(token string) fiber.Handler {
	if token == "" {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	return func(c *fiber.Ctx) error {
		provided := c.Query("token")
		if provided == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				provided = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if provided != token {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		return c.Next()
	}
}

// ValidateToken checks if the provided token matches the expected token.
// Returns true if the expected token is empty (auth disabled) or if tokens
// match. Used for WebSocket host connection validation where HTTP middleware
// cannot be applied.
func ValidateToken(expected, provided string) bool {
	if expected == "" {
		return true
	}
	return provided == expected
}
