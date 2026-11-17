package middlewares

import (
	"database/sql"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func appAuth(t *testing.T) *fiber.App {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE sessions (id TEXT PRIMARY KEY, user_id INTEGER NOT NULL, data TEXT NOT NULL,
		expires_at DATETIME NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);`)
	require.NoError(t, err)
	store := session.New(queries.NewQuerier(db), nil, time.Hour)
	app := fiber.New()
	app.Use("/app", AuthRequired(store))
	app.Get("/app/*", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Post("/app/*", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func cookieTujuan(t *testing.T, header []string) string {
	t.Helper()
	for _, h := range header {
		if strings.HasPrefix(h, session.CookieTujuanLogin+"=") {
			v := strings.SplitN(strings.TrimPrefix(h, session.CookieTujuanLogin+"="), ";", 2)[0]
			dec, err := url.QueryUnescape(v)
			require.NoError(t, err)
			return dec
		}
	}
	return ""
}

func TestAuthRequiredMenyimpanHalamanTujuan(t *testing.T) {
	app := appAuth(t)

	req := httptest.NewRequest("GET", "/app/kelas/7?tab=santri", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 302, resp.StatusCode)
	require.Equal(t, "/login", resp.Header.Get("Location"))
	require.Equal(t, "/app/kelas/7?tab=santri", cookieTujuan(t, resp.Header.Values("Set-Cookie")))

	req = httptest.NewRequest("GET", "/app/keuangan", nil)
	req.Header.Set("X-Inertia", "true")
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 409, resp.StatusCode)
	require.Equal(t, "/app/keuangan", cookieTujuan(t, resp.Header.Values("Set-Cookie")))
}

func TestAuthRequiredTidakMenyimpanTujuanUntukPOST(t *testing.T) {
	app := appAuth(t)
	req := httptest.NewRequest("POST", "/app/tasbih/sync", nil)
	req.Header.Set("X-Inertia", "true")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, "", cookieTujuan(t, resp.Header.Values("Set-Cookie")))
}
