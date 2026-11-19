package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
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
