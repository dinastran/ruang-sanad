package routes

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/handlers"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestLeadRoutesRoleGuard(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "migrations")))

	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	userService := services.NewUserService(q)
	leads := services.NewLeadService(q, services.NewSantriService(q, services.NewKelasEngineService(q)))
	h := Handlers{
		Leads:         handlers.NewLeadHandler(leads, store, nil),
		LeadClosing:   handlers.NewLeadClosingHandler(leads, store, nil),
		LeadLaporan:   handlers.NewLeadLaporanHandler(leads, store, nil),
		LeadFollowups: handlers.NewLeadFollowupHandler(leads, services.NewWAOutboxService(q, services.NewCSBridgeClient("", "")), store, nil),
		IntegrasiWhatsApp: handlers.NewIntegrasiWhatsAppHandler(
			services.NewCRMIntegrationService(q, services.CRMIntegrationConfig{}), store, nil),
	}
	app := fiber.New()
	setupLeadRoutes(app.Group("/app"), h, store, userService)

	login := func(role string) string {
		res, err := db.Exec(`INSERT INTO users (email, name, role) VALUES (?, ?, ?)`, role+"@example.test", role, role)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		sid := fmt.Sprintf("session-%s", role)
		now := time.Now()
		_, err = db.Exec(`INSERT INTO sessions (id, user_id, data, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			sid, id, fmt.Sprintf(`{"user_id":%d,"role":"%s"}`, id, role), now.Add(time.Hour), now, now)
		require.NoError(t, err)
		return sid
	}
	do := func(sid, method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		return resp.StatusCode
	}

	adminKelas := login("admin_kelas")
	guru := login("guru")
	keuangan := login("keuangan")
	cs := login("cs")

	for _, path := range []string{"/app/leads", "/app/leads/dashboard", "/app/leads/laporan", "/app/leads/laporan/export", "/app/leads/1"} {
		require.Equal(t, fiber.StatusForbidden, do(adminKelas, http.MethodGet, path), path)
		require.Equal(t, fiber.StatusForbidden, do(guru, http.MethodGet, path), path)
	}
	require.Equal(t, fiber.StatusForbidden, do(adminKelas, http.MethodPost, "/app/leads/1/convert"))
	for _, sid := range []string{adminKelas, guru} {
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodPost, "/app/leads/1/label-catalog"))
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodPut, "/app/leads/1/label-catalog/1"))
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodDelete, "/app/leads/1/label-catalog/1"))
	}
	// CSV export renders without Inertia, so it proves the allowed roles pass the guard.
	require.Equal(t, fiber.StatusOK, do(keuangan, http.MethodGet, "/app/leads/laporan/export"))
	require.Equal(t, fiber.StatusOK, do(cs, http.MethodGet, "/app/leads/laporan/export"))
	// Integration settings and hard delete are super_admin only.
	require.Equal(t, fiber.StatusForbidden, do(cs, http.MethodGet, "/app/integrasi-whatsapp"))
	require.Equal(t, fiber.StatusForbidden, do(keuangan, http.MethodDelete, "/app/leads/1"))
	// Follow-up approval is cs and super_admin only (PRD 8.6).
	for _, sid := range []string{keuangan, adminKelas, guru} {
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodGet, "/app/leads/followups"))
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodPut, "/app/leads/followups/1/approve"))
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodPost, "/app/leads/followups/bulk-approve"))
		require.Equal(t, fiber.StatusForbidden, do(sid, http.MethodPost, "/app/leads/outbox/1/retry"))
	}
	require.Equal(t, fiber.StatusForbidden, do(adminKelas, http.MethodPost, "/app/leads/1/followups"))
}
