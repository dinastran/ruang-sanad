package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// OAuthService is the OAuth 2.1 authorization server that lets Claude.ai and
// ChatGPT connect to /mcp on behalf of a staff user (MCP authorization spec):
// Dynamic Client Registration, authorization code + PKCE (S256) with a
// consent page, rotating refresh tokens, and revocation. Each approved
// connection is an ai_tokens row (jenis 'oauth'), so it shows up and can be
// revoked on the Akses AI page and its tool calls land in the same audit log.
type OAuthService struct {
	querier       *queries.Querier
	now           func() time.Time
	redirectHosts map[string]bool
}

const (
	OAuthAccessPrefix  = "rsoa_"
	OAuthRefreshPrefix = "rsor_"
	OAuthScope         = "mcp"

	oauthClientIDPrefix     = "rsoc_"
	oauthClientSecretPrefix = "rsos_"
	oauthCodePrefix         = "rsok_"

	oauthCodeTTL    = 5 * time.Minute
	OAuthAccessTTL  = time.Hour
	OAuthRefreshTTL = 30 * 24 * time.Hour

	oauthMaxRedirectURIs = 10
	oauthMaxURILength    = 2000
	oauthClientNameMax   = 60
)

func NewOAuthService(q *queries.Querier, redirectHosts []string) *OAuthService {
	hosts := map[string]bool{}
	for _, h := range redirectHosts {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			hosts[h] = true
		}
	}
	return &OAuthService{querier: q, now: time.Now, redirectHosts: hosts}
}

// OAuthError is an error with an RFC 6749 error code. Status is the HTTP
// status for the token/registration endpoints.
type OAuthError struct {
	Code        string
	Description string
	Status      int
}

func (e *OAuthError) Error() string { return e.Code + ": " + e.Description }

func oauthErr(status int, code, desc string) *OAuthError {
	return &OAuthError{Code: code, Description: desc, Status: status}
}

func randomToken(prefix string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validRedirectURI accepts https callbacks on the allowed AI app hosts and
// http(s) loopback callbacks for local MCP clients.
func (s *OAuthService) validRedirectURI(raw string) bool {
	if raw == "" || len(raw) > oauthMaxURILength {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Fragment != "" || u.User != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if isLoopbackHost(host) {
		return u.Scheme == "http" || u.Scheme == "https"
	}
	return u.Scheme == "https" && s.redirectHosts[host]
}

// redirectMatches compares a requested redirect_uri with a registered one.
// Loopback URIs may use any port (RFC 8252 section 7.3).
func redirectMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	r, err1 := url.Parse(registered)
	q, err2 := url.Parse(requested)
	if err1 != nil || err2 != nil || !isLoopbackHost(strings.ToLower(r.Hostname())) {
		return false
	}
	return r.Scheme == q.Scheme && strings.EqualFold(r.Hostname(), q.Hostname()) && r.Path == q.Path && r.RawQuery == q.RawQuery
}

// ---------------------------------------------------------------------------
// Dynamic Client Registration (RFC 7591)

type OAuthRegisterRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

type OAuthRegisterResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   *int64   `json:"client_secret_expires_at,omitempty"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

func (s *OAuthService) RegisterClient(req OAuthRegisterRequest) (*OAuthRegisterResponse, error) {
	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > oauthMaxRedirectURIs {
		return nil, oauthErr(400, "invalid_redirect_uri", "redirect_uris wajib diisi (1–10 alamat)")
	}
	for _, u := range req.RedirectURIs {
		if !s.validRedirectURI(u) {
			return nil, oauthErr(400, "invalid_redirect_uri", "redirect_uri tidak diizinkan: "+u)
		}
	}
	grants := req.GrantTypes
	if len(grants) == 0 {
		grants = []string{"authorization_code"}
	}
	for _, g := range grants {
		if g != "authorization_code" && g != "refresh_token" {
			return nil, oauthErr(400, "invalid_client_metadata", "grant_type tidak didukung: "+g)
		}
	}
	responses := req.ResponseTypes
	if len(responses) == 0 {
		responses = []string{"code"}
	}
	for _, r := range responses {
		if r != "code" {
			return nil, oauthErr(400, "invalid_client_metadata", "response_type tidak didukung: "+r)
		}
	}
	method := req.TokenEndpointAuthMethod
	if method == "" {
		method = "client_secret_basic" // RFC 7591 default
	}
	if method != "none" && method != "client_secret_basic" && method != "client_secret_post" {
		return nil, oauthErr(400, "invalid_client_metadata", "token_endpoint_auth_method tidak didukung: "+method)
	}
	name := strings.Join(strings.Fields(req.ClientName), " ")
	if name == "" {
		name = "Aplikasi AI"
	}
	if utf8.RuneCountInString(name) > oauthClientNameMax {
		name = string([]rune(name)[:oauthClientNameMax])
	}

	clientID, err := randomToken(oauthClientIDPrefix)
	if err != nil {
		return nil, err
	}
	client := models.OAuthClient{ClientID: clientID, ClientName: name, RedirectURIs: req.RedirectURIs, TokenEndpointAuthMethod: method}
	resp := &OAuthRegisterResponse{
		ClientID: clientID, ClientIDIssuedAt: s.now().Unix(), ClientName: name, RedirectURIs: req.RedirectURIs,
		GrantTypes: grants, ResponseTypes: responses, TokenEndpointAuthMethod: method, Scope: OAuthScope,
	}
	if method != "none" {
		secret, err := randomToken(oauthClientSecretPrefix)
		if err != nil {
			return nil, err
		}
		client.ClientSecretHash = hashAIToken(secret)
		never := int64(0)
		resp.ClientSecret, resp.ClientSecretExpiresAt = secret, &never
	}
	if err := s.querier.CreateOAuthClient(context.Background(), client); err != nil {
		return nil, err
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Authorization endpoint

type OAuthAuthorizeParams struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
	Scope               string
	Resource            string
}

// OAuthAuthorizeRequest is a validated authorization request.
type OAuthAuthorizeRequest struct {
	Client        *models.OAuthClient
	RedirectURI   string
	CodeChallenge string
	State         string
	Scope         string
	Resource      string
}

// ErrOAuthNoRedirect marks authorization errors that must be shown to the
// user instead of redirected (unknown client or untrusted redirect_uri).
var ErrOAuthNoRedirect = errors.New("permintaan otorisasi tidak valid")

// ValidateAuthorize checks an authorization request. When the client and
// redirect_uri are valid but something else is wrong, it returns the request
// together with an *OAuthError so the caller can redirect the error back to
// the client. resourceURL is this server's MCP endpoint (RFC 8707).
func (s *OAuthService) ValidateAuthorize(p OAuthAuthorizeParams, resourceURL string) (*OAuthAuthorizeRequest, error) {
	if p.ClientID == "" {
		return nil, ErrOAuthNoRedirect
	}
	client, err := s.querier.GetOAuthClient(context.Background(), p.ClientID)
	if errors.Is(err, queries.ErrOAuthClientNotFound) {
		return nil, ErrOAuthNoRedirect
	}
	if err != nil {
		return nil, err
	}
	redirect := p.RedirectURI
	if redirect == "" {
		if len(client.RedirectURIs) != 1 {
			return nil, ErrOAuthNoRedirect
		}
		redirect = client.RedirectURIs[0]
	}
	matched := false
	for _, r := range client.RedirectURIs {
		if redirectMatches(r, redirect) {
			matched = true
			break
		}
	}
	if !matched || !s.validRedirectURI(redirect) {
		return nil, ErrOAuthNoRedirect
	}
	req := &OAuthAuthorizeRequest{Client: client, RedirectURI: redirect, State: p.State, Scope: OAuthScope, Resource: p.Resource}

	if p.ResponseType != "code" {
		return req, oauthErr(400, "unsupported_response_type", "hanya response_type=code yang didukung")
	}
	if p.CodeChallenge == "" || p.CodeChallengeMethod != "S256" {
		return req, oauthErr(400, "invalid_request", "PKCE wajib dengan code_challenge_method=S256")
	}
	if l := len(p.CodeChallenge); l < 43 || l > 128 {
		return req, oauthErr(400, "invalid_request", "code_challenge tidak valid")
	}
	if p.Resource != "" && !sameResource(p.Resource, resourceURL) {
		return req, oauthErr(400, "invalid_target", "resource harus "+resourceURL)
	}
	req.CodeChallenge = p.CodeChallenge
	return req, nil
}

func sameResource(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

var ErrOAuthTulisDitolak = errors.New("hanya super_admin yang bisa memberi izin menulis")

// Approve issues a one-time authorization code for a user who approved the
// request on the consent page. role is the user's current role from the DB.
func (s *OAuthService) Approve(userID int64, role string, req *OAuthAuthorizeRequest, canWrite bool) (string, error) {
	if !AIRoleAllowed(role) {
		return "", ErrAIRoleDitolak
	}
	if canWrite && role != "super_admin" {
		return "", ErrOAuthTulisDitolak
	}
	code, err := randomToken(oauthCodePrefix)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	now := s.now()
	_ = s.querier.DeleteExpiredOAuthAuthorizationCodes(ctx, now)
	err = s.querier.CreateOAuthAuthorizationCode(ctx, models.OAuthAuthorizationCode{
		CodeHash: hashAIToken(code), ClientID: req.Client.ClientID, UserID: userID, RedirectURI: req.RedirectURI,
		CodeChallenge: req.CodeChallenge, CanWrite: canWrite, ExpiresAt: now.Add(oauthCodeTTL),
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

// ---------------------------------------------------------------------------
// Token endpoint

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// authenticateClient checks client credentials at the token endpoint. Public
// clients (no secret) are identified by client_id and protected by PKCE.
func (s *OAuthService) authenticateClient(clientID, secret string) (*models.OAuthClient, error) {
	if clientID == "" {
		return nil, oauthErr(401, "invalid_client", "client_id wajib diisi")
	}
	client, err := s.querier.GetOAuthClient(context.Background(), clientID)
	if errors.Is(err, queries.ErrOAuthClientNotFound) {
		return nil, oauthErr(401, "invalid_client", "client tidak dikenal")
	}
	if err != nil {
		return nil, err
	}
	if client.ClientSecretHash != "" {
		if secret == "" || subtle.ConstantTimeCompare([]byte(hashAIToken(secret)), []byte(client.ClientSecretHash)) != 1 {
			return nil, oauthErr(401, "invalid_client", "client_secret salah")
		}
	}
	return client, nil
}

func pkceValid(verifier, challenge string) bool {
	if l := len(verifier); l < 43 || l > 128 {
		return false
	}
	for _, r := range verifier {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '.' || r == '_' || r == '~') {
			return false
		}
	}
	sum := sha256.Sum256([]byte(verifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(got), []byte(challenge)) == 1
}

// ExchangeCode trades an authorization code for tokens and creates the
// connection (ai_tokens row) shown on the Akses AI page.
func (s *OAuthService) ExchangeCode(clientID, clientSecret, code, redirectURI, verifier, resource, resourceURL string) (*OAuthTokenResponse, error) {
	client, err := s.authenticateClient(clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	if code == "" || verifier == "" {
		return nil, oauthErr(400, "invalid_request", "code dan code_verifier wajib diisi")
	}
	if resource != "" && !sameResource(resource, resourceURL) {
		return nil, oauthErr(400, "invalid_target", "resource harus "+resourceURL)
	}
	ctx := context.Background()
	now := s.now()
	c, err := s.querier.ConsumeOAuthAuthorizationCode(ctx, hashAIToken(code), now)
	if errors.Is(err, queries.ErrOAuthCodeNotFound) {
		return nil, oauthErr(400, "invalid_grant", "kode otorisasi tidak valid atau sudah dipakai")
	}
	if err != nil {
		return nil, err
	}
	if c.ClientID != client.ClientID || !c.ExpiresAt.After(now) {
		return nil, oauthErr(400, "invalid_grant", "kode otorisasi tidak valid atau kedaluwarsa")
	}
	if redirectURI != "" && redirectURI != c.RedirectURI {
		return nil, oauthErr(400, "invalid_grant", "redirect_uri tidak cocok")
	}
	if !pkceValid(verifier, c.CodeChallenge) {
		return nil, oauthErr(400, "invalid_grant", "code_verifier tidak cocok")
	}
	role, err := s.querier.GetUserRole(ctx, c.UserID)
	if err != nil || !AIRoleAllowed(role) {
		return nil, oauthErr(400, "invalid_grant", "role pengguna tidak punya akses AI")
	}
	canWrite := c.CanWrite && role == "super_admin"

	secret, err := randomToken("grant_")
	if err != nil {
		return nil, err
	}
	grantID, err := s.querier.CreateOAuthGrant(ctx, c.UserID, client.ClientName, hashAIToken(secret), client.ClientID, now.Add(OAuthRefreshTTL), canWrite)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, grantID, now)
}

func (s *OAuthService) issueTokens(ctx context.Context, grantID int64, now time.Time) (*OAuthTokenResponse, error) {
	access, err := randomToken(OAuthAccessPrefix)
	if err != nil {
		return nil, err
	}
	refresh, err := randomToken(OAuthRefreshPrefix)
	if err != nil {
		return nil, err
	}
	if err := s.querier.CreateOAuthToken(ctx, grantID, "access", hashAIToken(access), now.Add(OAuthAccessTTL)); err != nil {
		return nil, err
	}
	if err := s.querier.CreateOAuthToken(ctx, grantID, "refresh", hashAIToken(refresh), now.Add(OAuthRefreshTTL)); err != nil {
		return nil, err
	}
	_ = s.querier.DeleteExpiredOAuthTokens(ctx, now.Add(-24*time.Hour))
	return &OAuthTokenResponse{
		AccessToken: access, TokenType: "Bearer", ExpiresIn: int64(OAuthAccessTTL / time.Second),
		RefreshToken: refresh, Scope: OAuthScope,
	}, nil
}

// Refresh rotates a refresh token. Presenting a refresh token that was
// already used ends the whole connection (token theft detection).
func (s *OAuthService) Refresh(clientID, clientSecret, refreshToken string) (*OAuthTokenResponse, error) {
	client, err := s.authenticateClient(clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	invalid := oauthErr(400, "invalid_grant", "refresh token tidak valid, sudah dicabut, atau kedaluwarsa. Hubungkan ulang dashboard.")
	if !strings.HasPrefix(refreshToken, OAuthRefreshPrefix) {
		return nil, invalid
	}
	ctx := context.Background()
	now := s.now()
	t, err := s.querier.GetOAuthTokenAuthByHash(ctx, hashAIToken(refreshToken))
	if errors.Is(err, queries.ErrOAuthTokenNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}
	if t.Jenis != "refresh" || t.GrantClientID != client.ClientID {
		return nil, invalid
	}
	if t.RevokedAt != nil {
		if t.GrantRevokedAt == nil {
			_ = s.querier.RevokeOAuthGrant(ctx, t.GrantID, now)
		}
		return nil, invalid
	}
	if !t.ExpiresAt.After(now) || t.GrantRevokedAt != nil || !t.GrantExpiresAt.After(now) || !AIRoleAllowed(t.Role) {
		return nil, invalid
	}
	revoked, err := s.querier.RevokeOAuthToken(ctx, t.TokenID, now)
	if err != nil {
		return nil, err
	}
	if !revoked { // raced with another use of the same refresh token
		_ = s.querier.RevokeOAuthGrant(ctx, t.GrantID, now)
		return nil, invalid
	}
	if err := s.querier.ExtendOAuthGrant(ctx, t.GrantID, now.Add(OAuthRefreshTTL)); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, t.GrantID, now)
}

// Revoke implements RFC 7009. Revoking a refresh token ends the connection;
// unknown tokens are ignored (the endpoint always answers 200).
func (s *OAuthService) Revoke(clientID, clientSecret, token string) error {
	client, err := s.authenticateClient(clientID, clientSecret)
	if err != nil {
		return err
	}
	ctx := context.Background()
	t, err := s.querier.GetOAuthTokenAuthByHash(ctx, hashAIToken(strings.TrimSpace(token)))
	if errors.Is(err, queries.ErrOAuthTokenNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if t.GrantClientID != client.ClientID {
		return nil
	}
	now := s.now()
	if t.Jenis == "refresh" {
		return s.querier.RevokeOAuthGrant(ctx, t.GrantID, now)
	}
	_, err = s.querier.RevokeOAuthToken(ctx, t.TokenID, now)
	return err
}

// RedirectHosts lists the allowed redirect hosts (for the metadata/UI).
func (s *OAuthService) RedirectHosts() []string {
	out := make([]string, 0, len(s.redirectHosts))
	for h := range s.redirectHosts {
		out = append(out, h)
	}
	slices.Sort(out)
	return out
}
