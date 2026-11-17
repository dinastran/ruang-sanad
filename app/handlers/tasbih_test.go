package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupTasbihApp: sesi login disuntik lewat header X-Test-User (0 = tanpa login).
func setupTasbihApp(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	h := NewTasbihHandler(services.NewTasbihService(q), store, inertia)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if uid := c.Get("X-Test-User"); uid != "" && uid != "0" {
			sess, err := store.Get(c)
			if err != nil {
				return err
			}
			var id int64
			fmt.Sscan(uid, &id)
			sess.Set("user_id", id)
			sess.Set("role", c.Get("X-Test-Role"))
		}
		return c.Next()
	})
	app.Post("/app/tasbih/sync", h.Sync)
	app.Get("/app/tasbih/riwayat", h.Riwayat)
	return app, db
}

func tasbihUser(t *testing.T, db *sql.DB, email, role string) int64 {
	t.Helper()
	res, err := db.Exec("INSERT INTO users (email, name, role) VALUES (?, ?, ?)", email, "U", role)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return id
}

func tasbihReq(t *testing.T, app *fiber.App, method, path, body string, uid int64, role string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User", fmt.Sprint(uid))
	req.Header.Set("X-Test-Role", role)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func sesiJSON(id string, dzikirID int64) string {
	m := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	return fmt.Sprintf(`{"id":%q,"dzikir_id":%d,"jumlah":33,"target":33,"mulai_at":%q,"selesai_at":%q}`, id, dzikirID, m, m)
}

func TestTasbihSyncButuhLogin(t *testing.T) {
	app, _ := setupTasbihApp(t)
	status, _ := tasbihReq(t, app, "POST", "/app/tasbih/sync", `{"sesi":[]}`, 0, "")
	require.Equal(t, 401, status)
}

func TestTasbihSyncRoleLain(t *testing.T) {
	app, db := setupTasbihApp(t)
	cs := tasbihUser(t, db, "cs@test.local", "cs")
	status, out := tasbihReq(t, app, "POST", "/app/tasbih/sync",
		`{"sesi":[`+sesiJSON("00000000-0000-4000-8000-000000000001", 1)+`]}`, cs, "cs")
	require.Equal(t, 200, status)
	require.Len(t, out["diterima"], 1)
	require.NotEmpty(t, out["badge_baru"])
}

func TestTasbihSyncBodyRusak(t *testing.T) {
	app, db := setupTasbihApp(t)
	u := tasbihUser(t, db, "u@test.local", "user")
	status, _ := tasbihReq(t, app, "POST", "/app/tasbih/sync", `{"sesi":`, u, "user")
	require.Equal(t, 400, status)
	parts := make([]string, 101)
	for i := range parts {
		parts[i] = sesiJSON(fmt.Sprintf("00000000-0000-4000-8000-%012d", i), 1)
	}
	status, _ = tasbihReq(t, app, "POST", "/app/tasbih/sync", `{"sesi":[`+strings.Join(parts, ",")+`]}`, u, "user")
	require.Equal(t, 400, status)
}

func TestTasbihRiwayatScopedUser(t *testing.T) {
	app, db := setupTasbihApp(t)
	a := tasbihUser(t, db, "a@test.local", "user")
	b := tasbihUser(t, db, "b@test.local", "user")
	status, _ := tasbihReq(t, app, "POST", "/app/tasbih/sync", `{"sesi":[`+sesiJSON("00000000-0000-4000-8000-000000000009", 1)+`]}`, a, "user")
	require.Equal(t, 200, status)
	status, out := tasbihReq(t, app, "GET", "/app/tasbih/riwayat?halaman=1", "", b, "user")
	require.Equal(t, 200, status)
	require.Empty(t, out["items"])
	status, out = tasbihReq(t, app, "GET", "/app/tasbih/riwayat?halaman=1", "", a, "user")
	require.Equal(t, 200, status)
	require.Len(t, out["items"], 1)
	require.Equal(t, false, out["ada_lagi"])
}

func TestRoleHomeUser(t *testing.T) {
	require.Equal(t, "/app/tasbih", roleHome(models.RoleUser))
}
