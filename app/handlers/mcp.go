package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
)

// MCPHandler serves the Model Context Protocol (Streamable HTTP transport) at
// /mcp so AI agents (Claude, Hermes, ...) can read dashboard data through the
// tools in services.AIAgentService. Stateless: every POST carries a bearer
// token, no session id is issued, and responses are plain JSON (no SSE).
type MCPHandler struct {
	tokens         *services.AITokenService
	agent          *services.AIAgentService
	appURL         string
	allowedOrigins map[string]bool
	version        string
}

func NewMCPHandler(tokens *services.AITokenService, agent *services.AIAgentService, appURL string, allowedOrigins []string, version string) *MCPHandler {
	origins := map[string]bool{}
	for _, o := range allowedOrigins {
		origins[strings.TrimRight(strings.TrimSpace(o), "/")] = true
	}
	if appURL != "" {
		origins[appURL] = true
	}
	return &MCPHandler{tokens: tokens, agent: agent, appURL: appURL, allowedOrigins: origins, version: version}
}

const mcpMaxBody = 1 << 20

// Protocol revisions this server speaks. The tools-only feature set is the
// same across them; the client's revision is echoed when supported.
var mcpProtocolVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

const mcpDefaultProtocol = "2025-06-18"

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

const (
	mcpParseError     = -32700
	mcpInvalidRequest = -32600
	mcpMethodNotFound = -32601
	mcpInvalidParams  = -32602
)

func (h *MCPHandler) baseURL(c *fiber.Ctx) string {
	if h.appURL != "" {
		return h.appURL
	}
	return c.Protocol() + "://" + c.Hostname()
}

// originAllowed rejects browser requests from other sites (DNS rebinding).
// Server-to-server MCP clients do not send Origin.
func (h *MCPHandler) originAllowed(c *fiber.Ctx) bool {
	origin := strings.TrimRight(c.Get(fiber.HeaderOrigin), "/")
	if origin == "" {
		return true
	}
	return h.allowedOrigins[origin] || origin == h.baseURL(c)
}

func mcpUnauthorized(c *fiber.Ctx, msg string) error {
	c.Set(fiber.HeaderWWWAuthenticate, `Bearer realm="ruang-sanad", error="invalid_token"`)
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": msg})
}

// Handle is POST /mcp.
func (h *MCPHandler) Handle(c *fiber.Ctx) error {
	if !h.originAllowed(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "origin tidak diizinkan"})
	}
	auth := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(auth, "Bearer ") {
		return mcpUnauthorized(c, "butuh header Authorization: Bearer <token AI dari halaman Akses AI>")
	}
	principal, err := h.tokens.Authenticate(c.UserContext(), strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		if errors.Is(err, services.ErrAITokenInvalid) || errors.Is(err, services.ErrAIRoleDitolak) {
			return mcpUnauthorized(c, err.Error())
		}
		slog.Error("mcp auth failed", "error", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	principal.BaseURL = h.baseURL(c)

	body := c.Body()
	if len(body) > mcpMaxBody {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "request terlalu besar"})
	}
	body = bytes.TrimSpace(body)

	// JSON-RPC batch (older protocol revisions) or a single message.
	if len(body) > 0 && body[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil || len(batch) == 0 {
			return c.JSON(mcpResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &mcpError{Code: mcpParseError, Message: "JSON tidak valid"}})
		}
		out := make([]mcpResponse, 0, len(batch))
		for _, raw := range batch {
			if resp := h.dispatch(c, principal, raw); resp != nil {
				out = append(out, *resp)
			}
		}
		if len(out) == 0 {
			return mcpAccepted(c)
		}
		return c.JSON(out)
	}
	resp := h.dispatch(c, principal, body)
	if resp == nil {
		return mcpAccepted(c)
	}
	return c.JSON(resp)
}

// mcpAccepted acknowledges notifications/responses: 202 with an empty body
// (SendStatus would write "Accepted" as the body).
func mcpAccepted(c *fiber.Ctx) error {
	c.Status(fiber.StatusAccepted)
	return nil
}

// MethodNotAllowed answers GET/DELETE /mcp: this server offers no SSE stream
// and keeps no session to terminate.
func (h *MCPHandler) MethodNotAllowed(c *fiber.Ctx) error {
	c.Set(fiber.HeaderAllow, fiber.MethodPost)
	return c.SendStatus(fiber.StatusMethodNotAllowed)
}

func (h *MCPHandler) dispatch(c *fiber.Ctx, p *models.AIPrincipal, raw []byte) *mcpResponse {
	var req mcpRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return &mcpResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &mcpError{Code: mcpParseError, Message: "JSON tidak valid"}}
	}
	isNotification := len(req.ID) == 0 || string(req.ID) == "null"
	if req.Method == "" {
		// A JSON-RPC response from the client (we never send requests) or garbage.
		if isNotification {
			return nil
		}
		return &mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{Code: mcpInvalidRequest, Message: "method wajib diisi"}}
	}
	if isNotification {
		return nil
	}
	resp := &mcpResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		version := mcpDefaultProtocol
		for _, v := range mcpProtocolVersions {
			if v == params.ProtocolVersion {
				version = v
			}
		}
		resp.Result = fiber.Map{
			"protocolVersion": version,
			"capabilities":    fiber.Map{"tools": fiber.Map{"listChanged": false}},
			"serverInfo":      fiber.Map{"name": "ruang-sanad", "title": "Dashboard Ruang Sanad", "version": h.version},
			"instructions":    h.agent.Instructions(p),
		}
	case "ping":
		resp.Result = fiber.Map{}
	case "tools/list":
		tools := h.agent.ToolsFor(p.Role, p.CanWrite)
		list := make([]fiber.Map, 0, len(tools))
		for _, t := range tools {
			list = append(list, fiber.Map{
				"name": t.Name, "title": t.Title, "description": t.Description, "inputSchema": t.InputSchema,
				"annotations": fiber.Map{"title": t.Title, "readOnlyHint": !t.Write, "destructiveHint": false, "idempotentHint": !t.Write, "openWorldHint": false},
			})
		}
		resp.Result = fiber.Map{"tools": list}
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
			resp.Error = &mcpError{Code: mcpInvalidParams, Message: "params.name wajib diisi"}
			return resp
		}
		r := h.agent.Call(c.UserContext(), p, params.Name, params.Arguments)
		resp.Result = fiber.Map{
			"content": []fiber.Map{{"type": "text", "text": r.Text}},
			"isError": r.IsError,
		}
	case "resources/list":
		resp.Result = fiber.Map{"resources": []any{}}
	case "prompts/list":
		resp.Result = fiber.Map{"prompts": []any{}}
	default:
		resp.Error = &mcpError{Code: mcpMethodNotFound, Message: "method tidak didukung: " + req.Method}
	}
	return resp
}
