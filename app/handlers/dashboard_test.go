package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestDashboardUsesCurrentRoleAndProtectsPayload(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec(`INSERT INTO users (id,email,name,role) VALUES (1,'dashboard@example.test','Dashboard','super_admin')`)
	require.NoError(t, err)
	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	overview := services.NewOverviewService(q, services.NewKeuanganDashboardService(q, services.NewUjrohService(q)), services.NewRiayahSantriService(q))
	h := NewDashboardHandler(overview, services.NewUserService(q), store, inertia)
	app := fiber.New()
	// Simulate an authenticated session that still holds the old elevated role.
	app.Get("/app", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		sess.Set("user_id", int64(1))
		sess.Set("role", "super_admin")
		sess.Set("name", "Dashboard")
		return h.Index(c)
	})
	for _, row := range []struct{ role, redirect string }{
		{"cs", "/app/santri"}, {"admin_kelas", "/app/kelas"}, {"guru", "/app/guru"}, {"koordinator_guru", "/app/koordinator-guru"}, {"user", "/app/profile"},
	} {
		_, err := db.Exec(`UPDATE users SET role = ? WHERE id = 1`, row.role)
		require.NoError(t, err)
		resp, err := app.Test(httptest.NewRequest("GET", "/app", nil))
		require.NoError(t, err)
		require.Equal(t, row.redirect, resp.Header.Get("Location"))
		require.Equal(t, 303, resp.StatusCode)
		resp.Body.Close()
	}
	for _, role := range []string{"super_admin", "admin", "keuangan"} {
		_, err := db.Exec(`UPDATE users SET role = ? WHERE id = 1`, role)
		require.NoError(t, err)
		req := httptest.NewRequest("GET", "/app", nil)
		req.Header.Set("X-Inertia", "true")
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		var page struct {
			Props struct {
				User struct {
					Role string `json:"role"`
				} `json:"user"`
				Overview map[string]interface{} `json:"overview"`
			} `json:"props"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
		resp.Body.Close()
		require.Equal(t, role, page.Props.User.Role)
		require.Contains(t, page.Props.Overview, "finance")
		if role == "super_admin" {
			require.Contains(t, page.Props.Overview, "operations")
		} else {
			for _, key := range []string{"santri", "operations", "riayah", "products", "stats", "role_data"} {
				require.NotContains(t, page.Props.Overview, key)
			}
		}
	}
	req := httptest.NewRequest("GET", "/app?bulan=not-a-month", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}
