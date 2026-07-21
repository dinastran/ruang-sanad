package middlewares

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

func RoleRequired(store *session.Store, userService *services.UserService, roles ...string) fiber.Handler {
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

		userRole, err := userService.GetUserRole(userID.(int64))
		if err != nil {
			slog.Error("role check failed", "error", err, "user_id", userID)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to verify role",
			})
		}

		for _, role := range roles {
			if userRole == role {
				c.Locals("user_id", userID)
				c.Locals("email", sess.Get("email"))
				c.Locals("role", userRole)
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Akses ditolak. Role tidak memiliki izin.",
		})
	}
}
