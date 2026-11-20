package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestXenditCreateSession(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "xnd_secret", user)
		require.Empty(t, pass)
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "/sessions", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &got))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payment_session_id":"ps-123","reference_id":"rs-1","status":"ACTIVE","amount":150000,
			"expires_at":"2026-11-19T05:00:00Z","payment_link_url":"https://dev.xen.to/abc"}`))
	}))
	defer srv.Close()

	gw := NewXenditGateway(srv.URL, "xnd_secret")
	exp := time.Date(2026, 11, 19, 5, 0, 0, 0, time.UTC)
	sess, err := gw.CreateSession(t.Context(), GatewaySessionRequest{
		ReferenceID: "rs-1", Amount: 150000, ExpiresAt: exp, Description: "Tahsin Reguler",
		CustomerRef: "lead-7", CustomerName: "Fulanah", CustomerPhone: "6281234567890",
		ReturnURL: "https://app.test/daftar/bayar/x",
	})
	require.NoError(t, err)
	require.Equal(t, "ps-123", sess.ID)
	require.Equal(t, "https://dev.xen.to/abc", sess.URL)
	require.Equal(t, "ACTIVE", sess.Status)
	require.True(t, sess.ExpiresAt.Equal(exp))

	require.Equal(t, "PAY", got["session_type"])
	require.Equal(t, "PAYMENT_LINK", got["mode"])
	require.Equal(t, "IDR", got["currency"])
	require.Equal(t, "ID", got["country"])
	require.EqualValues(t, 150000, got["amount"])
	require.Equal(t, "rs-1", got["reference_id"])
	require.Equal(t, "2026-11-19T05:00:00Z", got["expires_at"])
	require.Equal(t, "https://app.test/daftar/bayar/x", got["success_return_url"])
	cust := got["customer"].(map[string]any)
	require.Equal(t, "INDIVIDUAL", cust["type"])
	require.Equal(t, "+6281234567890", cust["mobile_number"])
	require.Equal(t, "Fulanah", cust["individual_detail"].(map[string]any)["given_names"])
}

func TestXenditGetSessionAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sessions/ps-ok" {
			_, _ = w.Write([]byte(`{"payment_session_id":"ps-ok","reference_id":"rs-1","status":"COMPLETED","amount":150000,"payment_id":"py-9"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_code":"INVALID_AMOUNT","message":"bad"}`))
	}))
	defer srv.Close()
	gw := NewXenditGateway(srv.URL, "k")
	sess, err := gw.GetSession(t.Context(), "ps-ok")
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", sess.Status)
	require.Equal(t, "py-9", sess.PaymentID)
	require.EqualValues(t, 150000, sess.Amount)
	_, err = gw.GetSession(t.Context(), "ps-bad")
	require.ErrorContains(t, err, "INVALID_AMOUNT")
}
