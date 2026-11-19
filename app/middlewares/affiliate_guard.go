package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// affiliateAllowed are the /app paths an external affiliate may open.
var affiliateAllowed = []string{"/app/affiliate", "/app/profile"}

// AffiliateRoleGuard keeps the "affiliate" role inside its own pages. Many
// /app routes only need a login (dashboard, tasbih, upload…), so a deny
// guard is needed in addition to per-route role lists.
func AffiliateRoleGuard(roleOf func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if roleOf(c) != "affiliate" {
			return c.Next()
		}
		p := strings.TrimRight(c.Path(), "/")
		for _, a := range affiliateAllowed {
			if p == a || strings.HasPrefix(p, a+"/") {
				return c.Next()
			}
		}
		if c.Method() == fiber.MethodGet {
			return c.Redirect("/app/affiliate", fiber.StatusFound)
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}
}

// SessionRole reads the current user's role from the database (the
// session's cached role may be stale).
func SessionRole(store *session.Store, users *services.UserService) func(*fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		sess, err := store.Get(c)
		if err != nil {
			return ""
		}
		id, ok := sess.Get("user_id").(int64)
		if !ok {
			return ""
		}
		role, err := users.GetUserRole(id)
		if err != nil {
			return ""
		}
		return role
	}
}
