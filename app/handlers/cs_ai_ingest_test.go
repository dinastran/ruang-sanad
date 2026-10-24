package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

const testIngestToken = "test-ingest-token-0123456789abcdef0123456789"

func setupCsAiIngest(t *testing.T, token string, cidrs []string) (*fiber.App, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	q := queries.NewQuerier(db)
	leads := services.NewLeadService(q, services.NewSantriService(q, services.NewKelasEngineService(q)))
	app := fiber.New()
	app.Post("/integrations/cs-ai/events", NewCsAiIngestHandler(leads, token, cidrs).Events)
	return app, db
}

const sampleIngestBody = `{"events":[
 {"event_id":"evt_a1","type":"message.in","occurred_at":"2026-10-24T03:12:45Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"wa_message_id":"3EB0A1","body":"Assalamu'alaikum, berapa biaya kelas tahsin?","media_type":""}},
 {"event_id":"evt_a2","type":"message.out","occurred_at":"2026-10-24T03:13:02Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"wa_message_id":"3EB0A2","body":"Wa'alaikumussalam","sender":"ai"}},
 {"event_id":"evt_a3","type":"form.sent","occurred_at":"2026-10-24T03:20:10Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{}},
 {"event_id":"evt_a4","type":"payment_info.sent","occurred_at":"2026-10-24T03:31:00Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"paket_kode":"1x","nominal_text":"Rp149.000"}},
 {"event_id":"evt_a5","type":"proof.received","occurred_at":"2026-10-24T04:02:00Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"media_type":"gambar"}}
]}`

func ingestPost(t *testing.T, app *fiber.App, auth, body string, headers ...string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/integrations/cs-ai/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

// fiber's app.Test connection reports 0.0.0.0 as the remote address.
var testCIDRs = []string{"0.0.0.0/32"}

func TestCsAiIngestDisabledWithoutToken(t *testing.T) {
	app, _ := setupCsAiIngest(t, "", testCIDRs)
	status, _ := ingestPost(t, app, "Bearer anything", sampleIngestBody)
	require.Equal(t, fiber.StatusNotFound, status)
}

func TestCsAiIngestWrongToken(t *testing.T) {
	app, _ := setupCsAiIngest(t, testIngestToken, testCIDRs)
	status, _ := ingestPost(t, app, "Bearer wrong", sampleIngestBody)
	require.Equal(t, fiber.StatusUnauthorized, status)
	status, _ = ingestPost(t, app, "", sampleIngestBody)
	require.Equal(t, fiber.StatusUnauthorized, status)
}

func TestCsAiIngestIPOutsideAllowlist(t *testing.T) {
	app, _ := setupCsAiIngest(t, testIngestToken, []string{"127.0.0.1/32", "172.16.0.0/12"})
	status, _ := ingestPost(t, app, "Bearer "+testIngestToken, sampleIngestBody)
	require.Equal(t, fiber.StatusForbidden, status)
}

func TestCsAiIngestRejectsProxiedRequest(t *testing.T) {
	app, _ := setupCsAiIngest(t, testIngestToken, testCIDRs)
	status, _ := ingestPost(t, app, "Bearer "+testIngestToken, sampleIngestBody, "X-Forwarded-For", "203.0.113.9")
	require.Equal(t, fiber.StatusForbidden, status)
}

func TestCsAiIngestAcceptsAndDeduplicates(t *testing.T) {
	app, db := setupCsAiIngest(t, testIngestToken, testCIDRs)
	status, out := ingestPost(t, app, "Bearer "+testIngestToken, sampleIngestBody)
	require.Equal(t, fiber.StatusOK, status)
	require.EqualValues(t, 5, out["accepted"])

	var stage string
	var amount int64
	require.NoError(t, db.QueryRow(`SELECT stage, payment_amount FROM leads WHERE wa_chat_id = '6281234567890@s.whatsapp.net'`).Scan(&stage, &amount))
	require.Equal(t, "bukti_masuk", stage)
	require.Equal(t, int64(149000), amount)

	status, out = ingestPost(t, app, "Bearer "+testIngestToken, sampleIngestBody)
	require.Equal(t, fiber.StatusOK, status)
	require.EqualValues(t, 0, out["accepted"])
	require.EqualValues(t, 5, out["duplicates"])
}

func TestCsAiIngestRejectsOversizedBatch(t *testing.T) {
	app, _ := setupCsAiIngest(t, testIngestToken, testCIDRs)
	events := make([]string, 101)
	for i := range events {
		events[i] = `{"event_id":"e","type":"form.sent","occurred_at":"2026-10-24T03:20:10Z","chat_id":"1@s.whatsapp.net","data":{}}`
	}
	status, _ := ingestPost(t, app, "Bearer "+testIngestToken, `{"events":[`+strings.Join(events, ",")+`]}`)
	require.Equal(t, fiber.StatusBadRequest, status)
}
