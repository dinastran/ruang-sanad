package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/stretchr/testify/require"
)

func TestXenditWebhookToken(t *testing.T) {
	f := setupDaftarFixture(t, "short", "bayar")
	post := func(h *XenditWebhookHandler, token, body string) int {
		app := fiber.New()
		app.Post("/webhooks/xendit", h.Handle)
		req := httptest.NewRequest("POST", "/webhooks/xendit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("x-callback-token", token)
		}
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		return resp.StatusCode
	}
	h := NewXenditWebhookHandler(f.reg, "rahasia")
	require.Equal(t, fiber.StatusUnauthorized, post(h, "", `{}`))
	require.Equal(t, fiber.StatusUnauthorized, post(h, "salah", `{}`))
	require.Equal(t, fiber.StatusOK, post(h, "rahasia", `{"event":"payment.capture","data":{}}`))
	require.Equal(t, fiber.StatusBadRequest, post(h, "rahasia", `bukan json`))
	require.Equal(t, fiber.StatusServiceUnavailable, post(NewXenditWebhookHandler(f.reg, ""), "", `{}`), "token belum diset: tolak semua")
}

type handlerFakeGateway struct{ n int }

func (g *handlerFakeGateway) CreateSession(_ context.Context, r services.GatewaySessionRequest) (services.GatewaySession, error) {
	g.n++
	return services.GatewaySession{ID: "ps-1", ReferenceID: r.ReferenceID, Status: "ACTIVE", Amount: r.Amount, URL: "https://dev.xen.to/x", ExpiresAt: r.ExpiresAt}, nil
}
func (g *handlerFakeGateway) GetSession(_ context.Context, id string) (services.GatewaySession, error) {
	return services.GatewaySession{}, nil
}

func TestDaftarGatewayStartAndStatus(t *testing.T) {
	f := setupDaftarFixture(t, "short", "bayar")
	code, raw := f.postJSON(t, "/daftar/tahsin", map[string]any{
		"nama": "Fulanah", "no_wa": "081234567890", "jenis_kelamin": "P", "jadwal_kode": "tst1", "form_token": f.oldToken("tahsin"),
	})
	require.Equal(t, fiber.StatusOK, code, string(raw))
	var out struct {
		BayarURL string `json:"bayar_url"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))

	// Gateway off → 422 with a message, never a redirect.
	code, _, _ = f.do(t, "POST", out.BayarURL+"/gateway", "", nil)
	require.Equal(t, fiber.StatusUnprocessableEntity, code)

	gw := &handlerFakeGateway{}
	f.reg.WithGateway(gw, "https://app.test")
	code, _, body := f.do(t, "POST", out.BayarURL+"/gateway", "", nil)
	require.Equal(t, fiber.StatusOK, code)
	require.Contains(t, string(body), "https://dev.xen.to/x")

	code, _, body = f.do(t, "GET", out.BayarURL+"/status", "", nil)
	require.Equal(t, fiber.StatusOK, code)
	var st map[string]any
	require.NoError(t, json.Unmarshal(body, &st))
	require.Equal(t, "belum", st["status"])
	require.Equal(t, true, st["sesi_aktif"])
}
