package middlewares

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// AuthRequired is a middleware that checks if the user is authenticated
func AuthRequired(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slog.Debug("checking auth", "path", c.Path())

		// Skip auth for OPTIONS (CORS preflight) — browser doesn't send cookies
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		sess, err := store.Get(c)
		if err != nil {
			slog.Error("auth session error", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get session",
			})
		}

		userID := sess.Get("user_id")
		slog.Debug("auth user id", "user_id", userID)

		if userID == nil {
			slog.Warn("not authenticated, redirecting to login")
			// For Inertia requests, return redirect in JSON format
			if c.Get("X-Inertia") == "true" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"component": "Login",
					"props": fiber.Map{
						"error": "Please login to continue",
					},
				})
			}
			return c.Redirect("/login")
		}

		// Store user info in locals for handlers to use
		c.Locals("user_id", userID)
		c.Locals("email", sess.Get("email"))
		c.Locals("role", sess.Get("role"))
		slog.Debug("auth successful", "user_id", userID)

		return c.Next()
	}
}

// AdminRequired is a middleware that checks if the user is an admin.
// It verifies the role from the database (via UserService) instead of relying
// on the session-stored role, so role changes take effect immediately.
func AdminRequired(store *session.Store, userService *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get session",
			})
		}

		userID := sess.Get("user_id")
		if userID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		// Check admin role from DB/cache, not from session
		isAdmin, err := userService.IsAdmin(userID.(int64))
		if err != nil {
			slog.Error("admin check failed", "error", err, "user_id", userID)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to verify admin status",
			})
		}

		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}

		c.Locals("user_id", userID)
		c.Locals("email", sess.Get("email"))
		c.Locals("role", "admin")

		return c.Next()
	}
}

// Guest is a middleware that redirects authenticated users away from login/register pages
func Guest(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			slog.Error("guest session error", "error", err)
			return c.Next()
		}

		userID := sess.Get("user_id")
		slog.Debug("guest check", "user_id", userID)

		if userID != nil {
			slog.Debug("guest already authenticated, redirecting")
			return c.Redirect("/app")
		}

		return c.Next()
	}
}
