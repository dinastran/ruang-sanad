package handlers

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// OAuthHandler is the OAuth 2.1 authorization server that lets Claude.ai and
// ChatGPT connect to /mcp: discovery metadata, client registration, the
// consent page, and the token/revocation endpoints. Business rules live in
// services.OAuthService.
type OAuthHandler struct {
	oauth          *services.OAuthService
	agent          *services.AIAgentService
	userService    *services.UserService
	store          *session.Store
	inertiaService *services.InertiaService
	appURL         string
}

func NewOAuthHandler(oauth *services.OAuthService, agent *services.AIAgentService, userService *services.UserService, store *session.Store, inertiaService *services.InertiaService, appURL string) *OAuthHandler {
	return &OAuthHandler{oauth: oauth, agent: agent, userService: userService, store: store, inertiaService: inertiaService, appURL: appURL}
}

// oauthReturnCookie remembers the authorization request while the user logs
// in; the login handlers send the user back to it.
const oauthReturnCookie = "oauth_return"

func (h *OAuthHandler) base(c *fiber.Ctx) string {
	if h.appURL != "" {
		return h.appURL
	}
	return c.Protocol() + "://" + c.Hostname()
}

func (h *OAuthHandler) resourceURL(c *fiber.Ctx) string { return h.base(c) + "/mcp" }

// ProtectedResourceMetadata is RFC 9728 metadata for /mcp. MCP clients find
// it from the WWW-Authenticate header of a 401 response.
func (h *OAuthHandler) ProtectedResourceMetadata(c *fiber.Ctx) error {
	base := h.base(c)
	c.Set(fiber.HeaderAccessControlAllowOrigin, "*")
	return c.JSON(fiber.Map{
		"resource":                 base + "/mcp",
		"authorization_servers":    []string{base},
		"scopes_supported":         []string{services.OAuthScope},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Dashboard Ruang Sanad",
		"resource_documentation":   base + "/app/ai-akses",
	})
}

// AuthorizationServerMetadata is RFC 8414 metadata.
func (h *OAuthHandler) AuthorizationServerMetadata(c *fiber.Ctx) error {
	base := h.base(c)
	authMethods := []string{"none", "client_secret_basic", "client_secret_post"}
	c.Set(fiber.HeaderAccessControlAllowOrigin, "*")
	return c.JSON(fiber.Map{
		"issuer":                                         base,
		"authorization_endpoint":                         base + "/oauth/authorize",
		"token_endpoint":                                 base + "/oauth/token",
		"registration_endpoint":                          base + "/oauth/register",
		"revocation_endpoint":                            base + "/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"response_modes_supported":                       []string{"query"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          authMethods,
		"revocation_endpoint_auth_methods_supported":     authMethods,
		"scopes_supported":                               []string{services.OAuthScope},
		"service_documentation":                          base + "/app/ai-akses",
		"authorization_response_iss_parameter_supported": true,
	})
}

func oauthJSONError(c *fiber.Ctx, err error) error {
	var oe *services.OAuthError
	if errors.As(err, &oe) {
		if oe.Code == "invalid_client" && strings.HasPrefix(c.Get(fiber.HeaderAuthorization), "Basic ") {
			c.Set(fiber.HeaderWWWAuthenticate, `Basic realm="ruang-sanad"`)
		}
		return c.Status(oe.Status).JSON(fiber.Map{"error": oe.Code, "error_description": oe.Description})
	}
	slog.Error("oauth endpoint failed", "path", c.Path(), "error", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
}

func noStore(c *fiber.Ctx) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderPragma, "no-cache")
}

// Register is POST /oauth/register (RFC 7591 Dynamic Client Registration).
func (h *OAuthHandler) Register(c *fiber.Ctx) error {
	noStore(c)
	var req services.OAuthRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_client_metadata", "error_description": "body harus JSON"})
	}
	resp, err := h.oauth.RegisterClient(req)
	if err != nil {
		return oauthJSONError(c, err)
	}
	slog.Info("oauth client registered", "client_id", resp.ClientID, "client_name", resp.ClientName, "redirect_uris", resp.RedirectURIs)
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *OAuthHandler) authorizeParams(get func(string) string) services.OAuthAuthorizeParams {
	return services.OAuthAuthorizeParams{
		ResponseType:        get("response_type"),
		ClientID:            get("client_id"),
		RedirectURI:         get("redirect_uri"),
		CodeChallenge:       get("code_challenge"),
		CodeChallengeMethod: get("code_challenge_method"),
		State:               get("state"),
		Scope:               get("scope"),
		Resource:            get("resource"),
	}
}

// redirectToClient sends the browser back to the AI app with the given
// query parameters (code or error).
func (h *OAuthHandler) redirectToClient(c *fiber.Ctx, redirectURI string, params url.Values) error {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return fiber.ErrBadRequest
	}
	q := u.Query()
	for k, v := range params {
		for _, item := range v {
			if item != "" {
				q.Set(k, item)
			}
		}
	}
	u.RawQuery = q.Encode()
	if c.Get("X-Inertia") == "true" {
		return h.inertiaService.Location(c, u.String())
	}
	return c.Redirect(u.String(), fiber.StatusFound)
}

func (h *OAuthHandler) redirectError(c *fiber.Ctx, req *services.OAuthAuthorizeRequest, code, desc string) error {
	return h.redirectToClient(c, req.RedirectURI, url.Values{
		"error": {code}, "error_description": {desc}, "state": {req.State}, "iss": {h.base(c)},
	})
}

func (h *OAuthHandler) renderInvalid(c *fiber.Ctx, user any) error {
	c.Status(fiber.StatusBadRequest)
	return h.inertiaService.Render(c, "oauth/Authorize", fiber.Map{
		"user":  user,
		"galat": "Permintaan koneksi tidak valid atau aplikasi tidak dikenal. Mulai ulang dari pengaturan connector di Claude atau ChatGPT.",
	})
}

// Authorize is GET /oauth/authorize: validates the request, asks the user to
// log in if needed, then shows the consent page.
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
	req, err := h.oauth.ValidateAuthorize(h.authorizeParams(func(k string) string { return c.Query(k) }), h.resourceURL(c))
	if errors.Is(err, services.ErrOAuthNoRedirect) {
		return h.renderInvalid(c, nil)
	}
	var oe *services.OAuthError
	if errors.As(err, &oe) {
		return h.redirectError(c, req, oe.Code, oe.Description)
	}
	if err != nil {
		return err
	}

	sess, err := h.store.Get(c)
	if err != nil || sess.Get("user_id") == nil {
		c.Cookie(&fiber.Cookie{
			Name: oauthReturnCookie, Value: c.OriginalURL(), Path: "/", MaxAge: 900,
			HTTPOnly: true, SameSite: "Lax", Secure: c.Protocol() == "https",
		})
		h.store.Flash(c, "success", "Masuk untuk menghubungkan "+req.Client.ClientName+" ke dashboard.")
		return c.Redirect("/login", fiber.StatusFound)
	}
	user := sessionUser(sess)
	role, err := h.userService.GetUserRole(user.ID)
	if err != nil {
		return err
	}
	tools := make([]fiber.Map, 0)
	for _, t := range h.agent.ToolsFor(role, role == "super_admin") {
		tools = append(tools, fiber.Map{"name": t.Name, "title": t.Title, "write": t.Write})
	}
	redirectHost := req.RedirectURI
	if u, err := url.Parse(req.RedirectURI); err == nil {
		redirectHost = u.Host
	}
	noStore(c)
	return h.inertiaService.Render(c, "oauth/Authorize", fiber.Map{
		"user":        user,
		"role":        role,
		"diizinkan":   services.AIRoleAllowed(role),
		"boleh_tulis": role == "super_admin",
		"tools":       tools,
		"klien":       fiber.Map{"nama": req.Client.ClientName, "redirect_host": redirectHost},
		"permintaan": fiber.Map{
			"client_id":      req.Client.ClientID,
			"redirect_uri":   req.RedirectURI,
			"code_challenge": req.CodeChallenge,
			"state":          req.State,
			"resource":       req.Resource,
		},
	})
}

// Approve is POST /oauth/authorize: the user's decision on the consent page.
// Everything is validated again; nothing from the page is trusted.
func (h *OAuthHandler) Approve(c *fiber.Ctx) error {
	var body struct {
		ClientID      string `json:"client_id"`
		RedirectURI   string `json:"redirect_uri"`
		CodeChallenge string `json:"code_challenge"`
		State         string `json:"state"`
		Resource      string `json:"resource"`
		Izinkan       bool   `json:"izinkan"`
		CanWrite      bool   `json:"can_write"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.ErrBadRequest
	}
	req, err := h.oauth.ValidateAuthorize(services.OAuthAuthorizeParams{
		ResponseType: "code", ClientID: body.ClientID, RedirectURI: body.RedirectURI,
		CodeChallenge: body.CodeChallenge, CodeChallengeMethod: "S256", State: body.State, Resource: body.Resource,
	}, h.resourceURL(c))
	if errors.Is(err, services.ErrOAuthNoRedirect) {
		h.store.Flash(c, "error", "Permintaan koneksi tidak valid. Mulai ulang dari Claude atau ChatGPT.")
		return h.inertiaService.Redirect(c, "/app/ai-akses")
	}
	var oe *services.OAuthError
	if errors.As(err, &oe) {
		return h.redirectError(c, req, oe.Code, oe.Description)
	}
	if err != nil {
		return err
	}
	if !body.Izinkan {
		return h.redirectError(c, req, "access_denied", "pengguna menolak akses")
	}

	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	role, err := h.userService.GetUserRole(user.ID)
	if err != nil {
		return err
	}
	code, err := h.oauth.Approve(user.ID, role, req, body.CanWrite)
	if errors.Is(err, services.ErrAIRoleDitolak) || errors.Is(err, services.ErrOAuthTulisDitolak) {
		return h.redirectError(c, req, "access_denied", err.Error())
	}
	if err != nil {
		return err
	}
	slog.Info("oauth access approved", "user_id", user.ID, "client_id", req.Client.ClientID, "client_name", req.Client.ClientName, "can_write", body.CanWrite)
	return h.redirectToClient(c, req.RedirectURI, url.Values{"code": {code}, "state": {req.State}, "iss": {h.base(c)}})
}

// clientCredentials reads client_id/client_secret from HTTP Basic auth or
// the form body (RFC 6749 section 2.3.1).
func clientCredentials(c *fiber.Ctx) (string, string) {
	if auth := c.Get(fiber.HeaderAuthorization); strings.HasPrefix(auth, "Basic ") {
		if raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic ")); err == nil {
			if id, secret, ok := strings.Cut(string(raw), ":"); ok {
				id, _ = url.QueryUnescape(id)
				secret, _ = url.QueryUnescape(secret)
				return id, secret
			}
		}
	}
	return c.FormValue("client_id"), c.FormValue("client_secret")
}

// Token is POST /oauth/token.
func (h *OAuthHandler) Token(c *fiber.Ctx) error {
	noStore(c)
	clientID, secret := clientCredentials(c)
	var (
		resp *services.OAuthTokenResponse
		err  error
	)
	switch c.FormValue("grant_type") {
	case "authorization_code":
		resp, err = h.oauth.ExchangeCode(clientID, secret, c.FormValue("code"), c.FormValue("redirect_uri"),
			c.FormValue("code_verifier"), c.FormValue("resource"), h.resourceURL(c))
	case "refresh_token":
		resp, err = h.oauth.Refresh(clientID, secret, c.FormValue("refresh_token"))
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_grant_type", "error_description": "gunakan authorization_code atau refresh_token"})
	}
	if err != nil {
		return oauthJSONError(c, err)
	}
	return c.JSON(resp)
}

// Revoke is POST /oauth/revoke (RFC 7009).
func (h *OAuthHandler) Revoke(c *fiber.Ctx) error {
	noStore(c)
	clientID, secret := clientCredentials(c)
	if err := h.oauth.Revoke(clientID, secret, c.FormValue("token")); err != nil {
		return oauthJSONError(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}

// popOAuthReturn returns the pending authorization URL saved before login
// (and clears it), or "" when there is none. Only /oauth/authorize is
// accepted so the cookie cannot become an open redirect.
func popOAuthReturn(c *fiber.Ctx) string {
	v := c.Cookies(oauthReturnCookie)
	if v == "" {
		return ""
	}
	c.Cookie(&fiber.Cookie{Name: oauthReturnCookie, Value: "", Path: "/", MaxAge: -1, HTTPOnly: true, SameSite: "Lax"})
	if !strings.HasPrefix(v, "/oauth/authorize?") {
		return ""
	}
	return v
}
