package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/stretchr/testify/require"
)

func TestAIAksesStoreUsesCurrentRole(t *testing.T) {
	_, tokens, db := setupMCP(t)
	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	h := NewAIAksesHandler(tokens, nil, services.NewUserService(q), store, inertia, "https://dashboard.example.test")
	app := fiber.New()
	app.Post("/app/ai-akses/token", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		// Session still claims super_admin; the DB role decides.
		sess.Set("user_id", int64(2))
		sess.Set("role", "super_admin")
		return h.Store(c)
	})
	post := func(body string) (int, map[string]any) {
		req := httptest.NewRequest("POST", "/app/ai-akses/token", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		return resp.StatusCode, out
	}

	status, out := post(`{"nama":"Hermes","masa_berlaku_hari":30}`)
	require.Equal(t, 200, status)
	require.True(t, strings.HasPrefix(out["plaintext"].(string), services.AITokenPrefix))

	status, out = post(`{"nama":"","masa_berlaku_hari":30}`)
	require.Equal(t, 422, status)
	require.Contains(t, out["error"], "nama token")

	_, err := db.Exec(`UPDATE users SET role = 'guru' WHERE id = 2`)
	require.NoError(t, err)
	status, _ = post(`{"nama":"Hermes","masa_berlaku_hari":30}`)
	require.Equal(t, 403, status)
}
