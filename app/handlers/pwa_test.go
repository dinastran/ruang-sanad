package handlers

import (
	"database/sql"
	"io"
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

func setupMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	return db
}

func TestServiceWorkerHandler(t *testing.T) {
	app := fiber.New()
	h := NewPWAHandler(services.NewPWAService(services.NewAssetService("", "", false)))
	app.Get("/sw.js", h.ServiceWorker)
	resp, err := app.Test(httptest.NewRequest("GET", "/sw.js", nil), -1)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "javascript")
	require.Contains(t, resp.Header.Get("Cache-Control"), "no-cache")
	require.Equal(t, "/", resp.Header.Get("Service-Worker-Allowed"))
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "tasbih-static-")
}

func renderHTML(t *testing.T, component string) string {
	t.Helper()
	db := setupMemDB(t)
	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	app := fiber.New()
	app.Get("/x", func(c *fiber.Ctx) error { return inertia.Render(c, component, fiber.Map{}) })
	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil), -1)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func TestRootHTMLManifestPerHalaman(t *testing.T) {
	tasbih := renderHTML(t, "app/tasbih/Index")
	require.Contains(t, tasbih, `href="/public/tasbih.webmanifest"`)
	require.Contains(t, tasbih, `name="apple-mobile-web-app-capable" content="yes"`)
	require.Contains(t, tasbih, `name="apple-mobile-web-app-title" content="Tasbih"`)
	require.Contains(t, tasbih, `href="/public/icons/tasbih-apple-180.png"`)
	require.Contains(t, tasbih, "viewport-fit=cover")

	staf := renderHTML(t, "app/SantriList")
	require.Contains(t, staf, `href="/public/site.webmanifest"`)
	require.NotContains(t, staf, "tasbih.webmanifest")
}
