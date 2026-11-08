package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/middlewares"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oauthTestBase = "https://dashboard.example.test"

// setupOAuth wires the OAuth server, /mcp and a session-backed consent page
// the same way routes/web.go does (minus CSRF, covered by its own tests).
func setupOAuth(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec(`INSERT INTO users (id,email,name,role) VALUES (1,'boss@example.test','Boss','super_admin'),(2,'cs@example.test','CS','cs'),(3,'guru@example.test','Guru','guru')`)
	require.NoError(t, err)
	for id, role := range map[int]string{1: "super_admin", 2: "cs", 3: "guru"} {
		data, _ := json.Marshal(map[string]any{"user_id": id, "name": "U", "email": "u@example.test", "role": role})
		_, err = db.Exec(`INSERT INTO sessions (id,user_id,data,expires_at) VALUES (?,?,?,?)`, "sess"+string(rune('0'+id)), id, string(data), time.Now().Add(time.Hour).UTC())
		require.NoError(t, err)
	}

	q := queries.NewQuerier(db)
	store := session.New(q, nil, 24*time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	finance := services.NewKeuanganDashboardService(q, services.NewUjrohService(q))
	riayah := services.NewRiayahSantriService(q)
	santri := services.NewSantriService(q, services.NewKelasEngineService(q))
	agent := services.NewAIAgentService(q, services.NewOverviewService(q, finance, riayah), finance, riayah, services.NewTSIService(q), santri, services.NewLeadService(q, santri), nil, nil)
	tokens := services.NewAITokenService(q)
	users := services.NewUserService(q)

	oh := NewOAuthHandler(services.NewOAuthService(q, []string{"claude.ai", "chatgpt.com"}), agent, users, store, inertia, oauthTestBase)
	mh := NewMCPHandler(tokens, agent, oauthTestBase, nil, "test")
	app := fiber.New()
	app.Get("/.well-known/oauth-protected-resource/mcp", oh.ProtectedResourceMetadata)
	app.Get("/.well-known/oauth-authorization-server", oh.AuthorizationServerMetadata)
	app.Post("/oauth/register", oh.Register)
	app.Get("/oauth/authorize", oh.Authorize)
	app.Post("/oauth/authorize", middlewares.AuthRequired(store), oh.Approve)
	app.Post("/oauth/token", oh.Token)
	app.Post("/oauth/revoke", oh.Revoke)
	app.Post("/mcp", mh.Handle)
	return app, db
}

func do(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

func pkcePair() (string, string) {
	verifier := strings.Repeat("v3rifier-", 6) // 54 chars, allowed charset
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func registerClient(t *testing.T, app *fiber.App, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, raw := do(t, app, req)
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(raw))
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func tokenRequest(t *testing.T, app *fiber.App, form url.Values) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, raw := do(t, app, req)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	return resp.StatusCode, out
}

// approve runs the consent POST for the given session and returns the
// redirect back to the AI app.
func approve(t *testing.T, app *fiber.App, sess string, body map[string]any) *url.URL {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess})
	resp, out := do(t, app, req)
	require.Equal(t, http.StatusConflict, resp.StatusCode, string(out))
	loc, err := url.Parse(resp.Header.Get("X-Inertia-Location"))
	require.NoError(t, err)
	return loc
}

func TestOAuthDiscoveryAndUnauthorizedMCP(t *testing.T) {
	app, _ := setupOAuth(t)

	resp, raw := do(t, app, httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, string(raw))
	assert.Contains(t, resp.Header.Get("WWW-Authenticate"), `resource_metadata="`+oauthTestBase+`/.well-known/oauth-protected-resource/mcp"`)

	_, raw = do(t, app, httptest.NewRequest("GET", "/.well-known/oauth-protected-resource/mcp", nil))
	var prm map[string]any
	require.NoError(t, json.Unmarshal(raw, &prm))
	assert.Equal(t, oauthTestBase+"/mcp", prm["resource"])
	assert.Equal(t, []any{oauthTestBase}, prm["authorization_servers"])

	_, raw = do(t, app, httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil))
	var asm map[string]any
	require.NoError(t, json.Unmarshal(raw, &asm))
	assert.Equal(t, oauthTestBase, asm["issuer"])
	assert.Equal(t, oauthTestBase+"/oauth/register", asm["registration_endpoint"])
	assert.Equal(t, []any{"S256"}, asm["code_challenge_methods_supported"])
}

func TestOAuthRegisterRejectsUntrustedRedirect(t *testing.T) {
	app, _ := setupOAuth(t)
	for _, uri := range []string{"https://evil.example/callback", "http://claude.ai/api/mcp/auth_callback", "https://claude.ai/cb#frag"} {
		req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(`{"redirect_uris":["`+uri+`"]}`))
		req.Header.Set("Content-Type", "application/json")
		resp, raw := do(t, app, req)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, uri)
		assert.Contains(t, string(raw), "invalid_redirect_uri")
	}
	// Loopback (local MCP clients) is allowed.
	registerClient(t, app, `{"redirect_uris":["http://127.0.0.1:33418/callback"],"token_endpoint_auth_method":"none"}`)
}

func TestOAuthFullFlowPublicClient(t *testing.T) {
	app, db := setupOAuth(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	client := registerClient(t, app, `{"client_name":"Claude","redirect_uris":["`+redirect+`"],"grant_types":["authorization_code","refresh_token"],"token_endpoint_auth_method":"none"}`)
	clientID := client["client_id"].(string)
	assert.Empty(t, client["client_secret"])
	verifier, challenge := pkcePair()

	authorize := "/oauth/authorize?" + url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "state": {"xyz"},
		"resource": {oauthTestBase + "/mcp"}, "scope": {"mcp"},
	}.Encode()

	// Not logged in: remember the request and go to /login.
	resp, _ := do(t, app, httptest.NewRequest("GET", authorize, nil))
	require.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "/login", resp.Header.Get("Location"))
	var saved string
	for _, ck := range resp.Cookies() {
		if ck.Name == oauthReturnCookie {
			saved = ck.Value
		}
	}
	assert.Equal(t, authorize, saved)

	// Logged in: consent page.
	req := httptest.NewRequest("GET", authorize, nil)
	req.Header.Set("X-Inertia", "true")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess2"})
	resp, raw := do(t, app, req)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(raw))
	var page struct {
		Component string         `json:"component"`
		Props     map[string]any `json:"props"`
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	assert.Equal(t, "oauth/Authorize", page.Component)
	assert.Equal(t, true, page.Props["diizinkan"])
	assert.Equal(t, false, page.Props["boleh_tulis"])
	assert.Equal(t, "claude.ai", page.Props["klien"].(map[string]any)["redirect_host"])

	// CS approves (asking for write is refused: only super_admin may).
	consent := map[string]any{"client_id": clientID, "redirect_uri": redirect, "code_challenge": challenge, "state": "xyz", "resource": oauthTestBase + "/mcp", "izinkan": true, "can_write": true}
	loc := approve(t, app, "sess2", consent)
	assert.Equal(t, "access_denied", loc.Query().Get("error"))

	consent["can_write"] = false
	loc = approve(t, app, "sess2", consent)
	assert.Equal(t, "claude.ai", loc.Host)
	assert.Equal(t, "xyz", loc.Query().Get("state"))
	assert.Equal(t, oauthTestBase, loc.Query().Get("iss"))
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)

	// Wrong PKCE verifier fails and burns the code.
	status, out := tokenRequest(t, app, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {strings.Repeat("x", 50)}})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_grant", out["error"])
	status, _ = tokenRequest(t, app, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}})
	assert.Equal(t, http.StatusBadRequest, status, "code is single use")

	code = approve(t, app, "sess2", consent).Query().Get("code")
	status, out = tokenRequest(t, app, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}, "resource": {oauthTestBase + "/mcp"}})
	require.Equal(t, http.StatusOK, status, out)
	access, refresh := out["access_token"].(string), out["refresh_token"].(string)
	assert.True(t, strings.HasPrefix(access, services.OAuthAccessPrefix))
	assert.Equal(t, "Bearer", out["token_type"])
	assert.EqualValues(t, 3600, out["expires_in"])

	// The connection shows up as an ai_tokens row of jenis oauth.
	var jenis, nama string
	require.NoError(t, db.QueryRow(`SELECT jenis, nama FROM ai_tokens WHERE user_id = 2`).Scan(&jenis, &nama))
	assert.Equal(t, "oauth", jenis)
	assert.Equal(t, "Claude", nama)

	// The access token works on /mcp with the CS role's tools.
	code2, body, _ := mcpPost(t, app, access, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	require.Equal(t, 200, code2, body)
	assert.NotEmpty(t, body["result"].(map[string]any)["tools"])

	// Refresh rotates; the old refresh token cannot be reused, and reuse ends
	// the connection.
	status, out = tokenRequest(t, app, url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}})
	require.Equal(t, http.StatusOK, status, out)
	access2, refresh2 := out["access_token"].(string), out["refresh_token"].(string)
	assert.NotEqual(t, refresh, refresh2)
	code2, _, _ = mcpPost(t, app, access2, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, 200, code2)

	status, out = tokenRequest(t, app, url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_grant", out["error"])
	code2, _, _ = mcpPost(t, app, access2, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, 401, code2, "refresh token reuse revokes the whole connection")
	status, _ = tokenRequest(t, app, url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh2}})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestOAuthConfidentialClientAndRevoke(t *testing.T) {
	app, db := setupOAuth(t)
	redirect := "https://chatgpt.com/connector_platform_oauth_redirect"
	client := registerClient(t, app, `{"client_name":"ChatGPT","redirect_uris":["`+redirect+`"],"grant_types":["authorization_code","refresh_token"]}`)
	clientID, secret := client["client_id"].(string), client["client_secret"].(string)
	assert.Equal(t, "client_secret_basic", client["token_endpoint_auth_method"])
	require.NotEmpty(t, secret)
	verifier, challenge := pkcePair()

	// super_admin may grant write access.
	code := approve(t, app, "sess1", map[string]any{"client_id": clientID, "redirect_uri": redirect, "code_challenge": challenge, "state": "s", "izinkan": true, "can_write": true}).Query().Get("code")
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}}

	// Missing/wrong secret is rejected.
	status, out := tokenRequest(t, app, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "client_secret": {"rsos_wrong"}, "code": {code}, "code_verifier": {verifier}})
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, "invalid_client", out["error"])

	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(secret))
	resp, raw := do(t, app, req)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(raw))
	var tok map[string]any
	require.NoError(t, json.Unmarshal(raw, &tok))
	var canWrite bool
	require.NoError(t, db.QueryRow(`SELECT can_write FROM ai_tokens WHERE user_id = 1 AND jenis = 'oauth'`).Scan(&canWrite))
	assert.True(t, canWrite)

	// Revoking the refresh token ends the connection.
	rev := httptest.NewRequest("POST", "/oauth/revoke", strings.NewReader(url.Values{"token": {tok["refresh_token"].(string)}, "client_id": {clientID}, "client_secret": {secret}}.Encode()))
	rev.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ = do(t, app, rev)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	code2, _, _ := mcpPost(t, app, tok["access_token"].(string), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, 401, code2)
}

func TestOAuthAuthorizeErrors(t *testing.T) {
	app, _ := setupOAuth(t)
	redirect := "https://claude.ai/api/mcp/auth_callback"
	clientID := registerClient(t, app, `{"redirect_uris":["`+redirect+`"],"token_endpoint_auth_method":"none"}`)["client_id"].(string)
	_, challenge := pkcePair()

	// Unknown client or a redirect_uri that was not registered: shown, not redirected.
	for _, q := range []url.Values{
		{"response_type": {"code"}, "client_id": {"rsoc_unknown"}, "redirect_uri": {redirect}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}},
		{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {"https://claude.ai/other"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}},
	} {
		resp, _ := do(t, app, httptest.NewRequest("GET", "/oauth/authorize?"+q.Encode(), nil))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Location"))
	}

	// Missing PKCE: redirected back with an error.
	resp, _ := do(t, app, httptest.NewRequest("GET", "/oauth/authorize?"+url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "state": {"s1"}}.Encode(), nil))
	require.Equal(t, http.StatusFound, resp.StatusCode)
	loc, _ := url.Parse(resp.Header.Get("Location"))
	assert.Equal(t, "invalid_request", loc.Query().Get("error"))
	assert.Equal(t, "s1", loc.Query().Get("state"))

	// Wrong resource indicator.
	resp, _ = do(t, app, httptest.NewRequest("GET", "/oauth/authorize?"+url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}, "resource": {"https://other.example/mcp"}}.Encode(), nil))
	loc, _ = url.Parse(resp.Header.Get("Location"))
	assert.Equal(t, "invalid_target", loc.Query().Get("error"))

	// guru is not allowed AI access: consent page says so and approval is denied.
	q := url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
	req := httptest.NewRequest("GET", "/oauth/authorize?"+q.Encode(), nil)
	req.Header.Set("X-Inertia", "true")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess3"})
	_, raw := do(t, app, req)
	assert.Contains(t, string(raw), `"diizinkan":false`)
	loc = approve(t, app, "sess3", map[string]any{"client_id": clientID, "redirect_uri": redirect, "code_challenge": challenge, "izinkan": true})
	assert.Equal(t, "access_denied", loc.Query().Get("error"))

	// Deny.
	loc = approve(t, app, "sess2", map[string]any{"client_id": clientID, "redirect_uri": redirect, "code_challenge": challenge, "state": "d", "izinkan": false})
	assert.Equal(t, "access_denied", loc.Query().Get("error"))
	assert.Equal(t, "d", loc.Query().Get("state"))
}
