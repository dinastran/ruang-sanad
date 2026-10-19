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

func setupMCP(t *testing.T) (*fiber.App, *services.AITokenService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec(`INSERT INTO users (id,email,name,role) VALUES (1,'boss@example.test','Boss','super_admin'),(2,'cs@example.test','CS','cs')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (id,nama,status,no_wa,email) VALUES (1,'Ahmad','aktif','081111111111','ahmad@mail.id')`)
	require.NoError(t, err)
	q := queries.NewQuerier(db)
	finance := services.NewKeuanganDashboardService(q, services.NewUjrohService(q))
	riayah := services.NewRiayahSantriService(q)
	overview := services.NewOverviewService(q, finance, riayah)
	agent := services.NewAIAgentService(q, overview, finance, riayah, services.NewTSIService(q), services.NewSantriService(q, services.NewKelasEngineService(q)))
	tokens := services.NewAITokenService(q)
	h := NewMCPHandler(tokens, agent, "https://dashboard.example.test", []string{"http://localhost:5173"}, "test")
	app := fiber.New()
	app.Post("/mcp", h.Handle)
	app.Get("/mcp", h.MethodNotAllowed)
	return app, tokens, db
}

func mcpPost(t *testing.T, app *fiber.App, token, body string, headers ...string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
	return resp.StatusCode, out, string(raw)
}

func TestMCPEndpointProtocolAndAuth(t *testing.T) {
	app, tokens, db := setupMCP(t)
	boss, err := tokens.Create(1, "super_admin", "Claude", 30)
	require.NoError(t, err)
	cs, err := tokens.Create(2, "cs", "Hermes", 30)
	require.NoError(t, err)

	// No / bad token → 401 with WWW-Authenticate.
	status, _, _ := mcpPost(t, app, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	require.Equal(t, 401, status)
	status, _, _ = mcpPost(t, app, "rsai_wrongwrongwrongwrongwrong", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	require.Equal(t, 401, status)

	// Cross-site browser origin is refused.
	status, _, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "Origin", "https://evil.example")
	require.Equal(t, 403, status)
	status, _, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "Origin", "https://dashboard.example.test")
	require.Equal(t, 200, status)

	// initialize negotiates a supported protocol and sends instructions.
	status, out, _ := mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	require.Equal(t, 200, status)
	result := out["result"].(map[string]any)
	require.Equal(t, "2025-06-18", result["protocolVersion"])
	require.Contains(t, result["instructions"], "BACA SAJA")
	_, out, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`)
	require.Equal(t, "2025-06-18", out["result"].(map[string]any)["protocolVersion"])

	// Notifications get 202 with no body.
	status, _, raw := mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	require.Equal(t, 202, status)
	require.Empty(t, raw)

	// tools/list is role-scoped.
	_, out, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	require.Len(t, out["result"].(map[string]any)["tools"], 15)
	_, out, _ = mcpPost(t, app, cs.Plaintext, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	csTools := out["result"].(map[string]any)["tools"].([]any)
	require.Len(t, csTools, 3)
	require.Equal(t, true, csTools[0].(map[string]any)["annotations"].(map[string]any)["readOnlyHint"])

	// tools/call returns sanitized text content with dashboard links.
	_, out, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"detail_santri","arguments":{"santri_id":1}}}`)
	res := out["result"].(map[string]any)
	require.Equal(t, false, res["isError"])
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	require.Contains(t, text, "Ahmad")
	require.Contains(t, text, "https://dashboard.example.test/app/guru/santri/1")
	require.NotContains(t, text, "081111111111")
	require.NotContains(t, text, "ahmad@mail.id")

	// A tool outside the role is an error result, not a protocol error.
	_, out, _ = mcpPost(t, app, cs.Plaintext, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"rekap_tunggakan","arguments":{}}}`)
	require.Equal(t, true, out["result"].(map[string]any)["isError"])

	// Unknown method / bad JSON.
	_, out, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":6,"method":"sampling/createMessage"}`)
	require.EqualValues(t, -32601, out["error"].(map[string]any)["code"])
	_, out, _ = mcpPost(t, app, boss.Plaintext, `{not json`)
	require.EqualValues(t, -32700, out["error"].(map[string]any)["code"])

	// Batch (older protocol revisions).
	status, _, raw = mcpPost(t, app, boss.Plaintext, `[{"jsonrpc":"2.0","id":7,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"}]`)
	require.Equal(t, 200, status)
	var batch []map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &batch))
	require.Len(t, batch, 1)

	// GET has no SSE stream.
	resp, err := app.Test(httptest.NewRequest("GET", "/mcp", nil))
	require.NoError(t, err)
	require.Equal(t, 405, resp.StatusCode)

	// Tool calls are audited; protocol chatter is not.
	var n int64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ai_audit_log`).Scan(&n))
	require.EqualValues(t, 2, n)

	// Revoked token is rejected.
	require.NoError(t, tokens.Revoke(1, boss.Token.ID))
	status, _, _ = mcpPost(t, app, boss.Plaintext, `{"jsonrpc":"2.0","id":8,"method":"ping"}`)
	require.Equal(t, 401, status)
}
