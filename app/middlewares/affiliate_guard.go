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
func AffiliateRoleGuard(roleOf func(*fiber.Ctx) (string, bool)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := roleOf(c)
		if !ok {
			// Fail closed: without the role we cannot tell an affiliate apart.
			return c.Status(fiber.StatusServiceUnavailable).SendString("Gagal memeriksa akses, coba lagi")
		}
		if role != "affiliate" {
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
// ok=false means the lookup failed (the guard then refuses the request).
func SessionRole(store *session.Store, users *services.UserService) func(*fiber.Ctx) (string, bool) {
	return func(c *fiber.Ctx) (string, bool) {
		sess, err := store.Get(c)
		if err != nil {
			return "", false
		}
		id, ok := sess.Get("user_id").(int64)
		if !ok {
			return "", true // not logged in: AuthRequired already handled it
		}
		role, err := users.GetUserRole(id)
		if err != nil {
			return "", false
		}
		return role, true
	}
}
