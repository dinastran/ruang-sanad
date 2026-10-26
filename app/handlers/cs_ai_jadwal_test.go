package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestCsAiJadwalEndpoint(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	q := queries.NewQuerier(db)
	_, err = db.Exec(`UPDATE jadwal_penawaran SET is_buka = 0 WHERE kode = 'ahad-pagi'`)
	require.NoError(t, err)

	leads := services.NewLeadService(q, services.NewSantriService(q, services.NewKelasEngineService(q)))
	h := NewCsAiIngestHandler(leads, testIngestToken, []string{"0.0.0.0/0"}).WithJadwal(services.NewJadwalPenawaranService(q))
	app := fiber.New()
	app.Get("/integrations/cs-ai/jadwal", h.Guard, h.Jadwal)

	get := func(auth string, headers ...string) (int, []byte) {
		req := httptest.NewRequest("GET", "/integrations/cs-ai/jadwal", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		for i := 0; i+1 < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, raw
	}

	code, _ := get("")
	require.Equal(t, fiber.StatusUnauthorized, code)
	code, _ = get("Bearer wrong-token")
	require.Equal(t, fiber.StatusUnauthorized, code)
	code, _ = get("Bearer "+testIngestToken, "X-Forwarded-For", "1.2.3.4")
	require.Equal(t, fiber.StatusForbidden, code, "requests through the public proxy are refused")

	code, raw := get("Bearer " + testIngestToken)
	require.Equal(t, fiber.StatusOK, code)
	var out struct {
		Jadwal []struct {
			Kode       string `json:"kode"`
			Label      string `json:"label"`
			Level      string `json:"level"`
			SisaIkhwan int64  `json:"sisa_ikhwan"`
		} `json:"jadwal"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Len(t, out.Jadwal, 3, "closed schedules are not offered")
	require.Equal(t, "selasa-malam", out.Jadwal[0].Kode)
	require.Equal(t, "Selasa 19.30–21.00 WIB", out.Jadwal[0].Label)
	require.EqualValues(t, 10, out.Jadwal[0].SisaIkhwan)
}
