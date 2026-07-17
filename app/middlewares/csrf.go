package middlewares

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CSRFConfig holds CSRF middleware configuration
type CSRFConfig struct {
	Secret      string        // Secret key for signing tokens
	CookieName  string        // Name of the CSRF token cookie
	HeaderName  string        // Name of the CSRF token header
	TokenLength int           // Length of the random token
	Expiry      time.Duration // Token expiry duration
	Secure      bool          // Secure cookie flag
	SameSite    string        // SameSite cookie attribute
	SkipPaths   []string      // Paths to skip CSRF check
	SkipMethods []string      // HTTP methods to skip CSRF check
}

// CSRFMiddleware implements CSRF protection using double-submit cookie pattern.
// The token is stored ONLY in the cookie (no session storage needed).
// Validation compares the X-XSRF-TOKEN header with the XSRF-TOKEN cookie value.
type CSRFMiddleware struct {
	config CSRFConfig
}

// DefaultCSRFConfig returns a default CSRF configuration
func DefaultCSRFConfig(secret string) CSRFConfig {
	return CSRFConfig{
		Secret:      secret,
		CookieName:  "XSRF-TOKEN",
		HeaderName:  "X-XSRF-TOKEN",
		TokenLength: 32,
		Expiry:      24 * time.Hour,
		Secure:      false, // Set to true in production with HTTPS
		SameSite:    "Lax",
		SkipMethods: []string{fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions},
	}
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(config CSRFConfig) *CSRFMiddleware {
	return &CSRFMiddleware{
		config: config,
	}
}

// Protect returns the CSRF protection middleware
func (csrf *CSRFMiddleware) Protect() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if path is in skip list
		for _, path := range csrf.config.SkipPaths {
			if strings.HasPrefix(c.Path(), path) {
				return c.Next()
			}
		}

		// Skip if method is in skip list
		for _, method := range csrf.config.SkipMethods {
			if c.Method() == method {
				return csrf.setToken(c)
			}
		}

		// Validate token for state-changing methods
		if err := csrf.validateToken(c); err != nil {
			return err
		}

		return c.Next()
	}
}

// setToken generates and sets a CSRF token using double-submit cookie pattern.
// No session write required — the token is stored in a cookie only.
func (csrf *CSRFMiddleware) setToken(c *fiber.Ctx) error {
	// Reuse existing cookie if still valid
	existingToken := c.Cookies(csrf.config.CookieName)
	if existingToken != "" {
		return c.Next()
	}

	// Generate new token
	token, err := csrf.generateToken()
	if err != nil {
		return err
	}

	// Set cookie with the token
	c.Cookie(&fiber.Cookie{
		Name:     csrf.config.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(csrf.config.Expiry.Seconds()),
		Secure:   csrf.config.Secure,
		HTTPOnly: false, // Must be false to allow JavaScript access
		SameSite: csrf.config.SameSite,
	})

	return c.Next()
}

// validateToken validates the CSRF token using double-submit cookie pattern.
// Compares the X-XSRF-TOKEN header with the XSRF-TOKEN cookie — no session lookup needed.
func (csrf *CSRFMiddleware) validateToken(c *fiber.Ctx) error {
	// Get token from header, form, or query
	token := c.Get(csrf.config.HeaderName)
	if token == "" {
		token = c.FormValue(csrf.config.CookieName)
	}
	if token == "" {
		token = c.Query(csrf.config.CookieName)
	}

	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "CSRF token missing")
	}

	// Get token from cookie (the source of truth)
	cookieToken := c.Cookies(csrf.config.CookieName)
	if cookieToken == "" {
		return fiber.NewError(fiber.StatusForbidden, "CSRF token invalid")
	}

	// Compare header token with cookie token (constant time comparison)
	if !csrf.constantTimeCompare(token, cookieToken) {
		return fiber.NewError(fiber.StatusForbidden, "CSRF token invalid")
	}

	return nil
}

// generateToken generates a random CSRF token
func (csrf *CSRFMiddleware) generateToken() (string, error) {
	bytes := make([]byte, csrf.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// constantTimeCompare compares two strings in constant time
func (csrf *CSRFMiddleware) constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

// GetToken retrieves the current CSRF token from a request context
func GetToken(c *fiber.Ctx) string {
	return c.Cookies("XSRF-TOKEN")
}
