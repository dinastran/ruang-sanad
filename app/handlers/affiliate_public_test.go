package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/stretchr/testify/require"
)

func TestAffiliateRedirect(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	leads := services.NewLeadService(f.q, services.NewSantriService(f.q, services.NewKelasEngineService(f.q)))
	aff := services.NewAffiliateService(f.q, leads, []byte("s"))
	id, err := aff.Register(t.Context(), models.AffiliateSignup{Nama: "Ahmad", NoWA: "081200000001"})
	require.NoError(t, err)
	require.NoError(t, aff.SetStatus(t.Context(), id, "aktif", 0))
	h := NewAffiliatePublicHandler(aff)
	app := fiber.New()
	app.Get("/r/:kode", h.Redirect)

	get := func(path, cookie string) (int, string, string) {
		req := httptest.NewRequest("GET", path, nil)
		if cookie != "" {
			req.Header.Set("Cookie", services.AffCookieName+"="+cookie)
		}
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		return resp.StatusCode, resp.Header.Get("Location"), resp.Header.Get("Set-Cookie")
	}
	code, loc, set := get("/r/ahmad?ke=tahsin", "")
	require.Equal(t, fiber.StatusFound, code)
	require.Equal(t, "/daftar/tahsin", loc)
	require.Contains(t, set, services.AffCookieName+"=AHMAD.")
	require.Contains(t, strings.ToLower(set), "httponly")

	val := strings.TrimPrefix(strings.Split(set, ";")[0], services.AffCookieName+"=")
	_, _, set2 := get("/r/ahmad?ke=tahsin", val)
	require.Empty(t, set2, "cookie berlaku tidak ditimpa")

	_, loc, set = get("/r/tidakada?ke=tahsin", "")
	require.Equal(t, "/daftar/tahsin", loc)
	require.Empty(t, set)

	_, err = f.db.Exec(`UPDATE registration_links SET mode = 'wa'`)
	require.NoError(t, err)
	_, loc, _ = get("/r/ahmad?ke=tahsin", "")
	require.True(t, strings.HasPrefix(loc, "https://wa.me/6281100000000?text="))
	require.Contains(t, loc, "ref%3A+AHMAD")

	_, loc, _ = get("/r/ahmad", "")
	require.Equal(t, "/", loc, "tanpa tujuan & tanpa default")
	_ = time.Now
}
