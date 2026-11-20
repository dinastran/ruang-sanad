package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/stretchr/testify/require"
)

func TestAffiliateSignupApproveActivate(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	_, err := f.db.Exec(`INSERT INTO users (id, email, name, role) VALUES (1, 'admin@example.com', 'Admin', 'super_admin')`)
	require.NoError(t, err)
	leads := services.NewLeadService(f.q, services.NewSantriService(f.q, services.NewKelasEngineService(f.q)))
	aff := services.NewAffiliateService(f.q, leads, []byte("s"))
	store := session.New(f.q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	var mailed []string
	h := NewAffiliateHandler(aff, store, inertia).WithMailer(func(to, subject, body string) error { mailed = append(mailed, to+"|"+body); return nil })
	app := fiber.New()
	asAdmin := func(next fiber.Handler) fiber.Handler {
		return func(c *fiber.Ctx) error {
			sess, err := store.Get(c)
			if err != nil {
				return err
			}
			sess.Set("user_id", int64(1))
			sess.Set("role", "super_admin")
			c.Locals("user_id", int64(1))
			c.Locals("role", "super_admin")
			return next(c)
		}
	}
	app.Post("/affiliate/daftar", h.SignupSubmit)
	app.Post("/affiliate/aktivasi/:token", h.ActivateSubmit)
	app.Post("/app/leads/affiliate/:aid/setujui", asAdmin(h.Approve))

	post := func(path string, body any) (int, []byte) {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, raw
	}
	code, raw := post("/affiliate/daftar", map[string]any{"nama": "Dewi Lestari", "no_wa": "081200000009", "email": "dewi@x.com"})
	require.Equal(t, fiber.StatusUnprocessableEntity, code, "wajib setuju syarat")
	require.Contains(t, string(raw), "setuju")
	code, _ = post("/affiliate/daftar", map[string]any{"nama": "Dewi Lestari", "no_wa": "081200000009", "email": "dewi@x.com", "setuju": true})
	require.Equal(t, fiber.StatusOK, code)
	var id int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM affiliates WHERE email = 'dewi@x.com'`).Scan(&id))

	code, _ = post("/app/leads/affiliate/"+itoaTest(id)+"/setujui", map[string]any{})
	require.Equal(t, fiber.StatusSeeOther, code)
	require.Len(t, mailed, 1)
	require.True(t, strings.HasPrefix(mailed[0], "dewi@x.com|"))
	i := strings.Index(mailed[0], "/affiliate/aktivasi/")
	token := strings.SplitN(mailed[0][i+len("/affiliate/aktivasi/"):], "\"", 2)[0]

	code, _ = post("/affiliate/aktivasi/"+token, map[string]any{"password": "pendek"})
	require.Equal(t, fiber.StatusUnprocessableEntity, code)
	code, _ = post("/affiliate/aktivasi/"+token, map[string]any{"password": "rahasia-123"})
	require.Equal(t, fiber.StatusOK, code)
	code, _ = post("/affiliate/aktivasi/"+token, map[string]any{"password": "rahasia-123"})
	require.Equal(t, fiber.StatusUnprocessableEntity, code, "token sekali pakai")
}
