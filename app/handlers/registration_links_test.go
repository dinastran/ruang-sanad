package handlers

import (
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/stretchr/testify/require"
)

func TestRegistrationLinkAdmin(t *testing.T) {
	f := setupDaftarFixture(t, "short", "bayar")
	_, err := f.db.Exec(`INSERT INTO users (id, email, name, role) VALUES (1, 'admin@example.com', 'Admin', 'super_admin')`)
	require.NoError(t, err)
	store := session.New(f.q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	h := NewRegistrationLinkHandler(f.reg, store, inertia)
	app := fiber.New()
	asUser := func(next fiber.Handler) fiber.Handler {
		return func(c *fiber.Ctx) error {
			sess, err := store.Get(c)
			if err != nil {
				return err
			}
			sess.Set("user_id", int64(1))
			sess.Set("role", "super_admin")
			c.Locals("user_id", int64(1))
			return next(c)
		}
	}
	app.Get("/app/leads/link-pendaftaran", asUser(h.Index))
	app.Post("/app/leads/link-pendaftaran", asUser(h.Create))
	app.Get("/app/leads/:id/payments/:pid/bukti", asUser(h.Bukti))

	req := httptest.NewRequest("GET", "/app/leads/link-pendaftaran", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var sourceID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM lead_sources ORDER BY id LIMIT 1`).Scan(&sourceID))
	body := `{"slug":"tahsin-2","judul":"Tahsin","program_id":1,"lead_source_id":` + itoaTest(sourceID) + `,"mode":"full","after_submit":"wa","izinkan_manual":true,"is_aktif":true}`
	req = httptest.NewRequest("POST", "/app/leads/link-pendaftaran", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusSeeOther, resp.StatusCode)
	var n int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM registration_links WHERE slug = 'tahsin-2'`).Scan(&n))
	require.Equal(t, 1, n)

	// Bukti: payment of another lead → 404, never a file path from input.
	_, err = f.db.Exec(`INSERT INTO leads (id, wa_chat_id, phone, created_at, updated_at) VALUES (50, 'a@s.whatsapp.net', '6280000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		(51, 'b@s.whatsapp.net', '6280000000002', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO payments (id, lead_id, metode, amount, status, bukti_path, created_at, updated_at)
		VALUES (7, 50, 'manual', 1, 'menunggu_verifikasi', '../../etc/passwd', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	req = httptest.NewRequest("GET", "/app/leads/51/payments/7/bukti", nil)
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	req = httptest.NewRequest("GET", "/app/leads/50/payments/7/bukti", nil)
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNotFound, resp.StatusCode, "path traversal di DB tetap tidak keluar dari folder bukti")
}

func itoaTest(n int64) string { return strconv.FormatInt(n, 10) }
