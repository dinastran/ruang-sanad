package middlewares

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestAffiliateRoleGuard(t *testing.T) {
	role := "affiliate"
	app := fiber.New()
	app.Use("/app", AffiliateRoleGuard(func(*fiber.Ctx) string { return role }))
	app.All("/app/*", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.All("/app", func(c *fiber.Ctx) error { return c.SendString("ok") })

	do := func(method, path string) (int, string) {
		resp, err := app.Test(httptest.NewRequest(method, path, nil), -1)
		require.NoError(t, err)
		return resp.StatusCode, resp.Header.Get("Location")
	}
	for _, p := range []string{"/app/affiliate", "/app/affiliate/rekening", "/app/profile", "/app/profile/password"} {
		code, _ := do("GET", p)
		require.Equal(t, 200, code, p)
	}
	for _, p := range []string{"/app", "/app/leads", "/app/tasbih", "/app/ai-akses", "/app/affiliatex"} {
		code, loc := do("GET", p)
		require.Equal(t, fiber.StatusFound, code, p)
		require.Equal(t, "/app/affiliate", loc)
	}
	code, _ := do("POST", "/app/upload/avatar")
	require.Equal(t, fiber.StatusForbidden, code)

	role = "cs"
	code, _ = do("GET", "/app/leads")
	require.Equal(t, 200, code, "role lain tidak terpengaruh")
}
