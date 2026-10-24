package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// AIAksesHandler serves the "Akses AI" page where staff create and revoke the
// personal tokens their AI agents use to call /mcp.
type AIAksesHandler struct {
	tokens         *services.AITokenService
	agent          *services.AIAgentService
	userService    *services.UserService
	store          *session.Store
	inertiaService *services.InertiaService
	appURL         string
}

func NewAIAksesHandler(tokens *services.AITokenService, agent *services.AIAgentService, userService *services.UserService, store *session.Store, inertiaService *services.InertiaService, appURL string) *AIAksesHandler {
	return &AIAksesHandler{tokens: tokens, agent: agent, userService: userService, store: store, inertiaService: inertiaService, appURL: appURL}
}

// currentUser returns the session user with the role re-read from the DB.
func (h *AIAksesHandler) currentUser(c *fiber.Ctx) (*models.UserResponse, error) {
	sess, err := h.store.Get(c)
	if err != nil || sess.Get("user_id") == nil {
		return nil, fiber.ErrUnauthorized
	}
	user := sessionUser(sess)
	role, err := h.userService.GetUserRole(user.ID)
	if err != nil {
		return nil, err
	}
	user.Role = models.UserRole(role)
	return user, nil
}

func (h *AIAksesHandler) Index(c *fiber.Ctx) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	role := string(user.Role)
	tokens, err := h.tokens.List(user.ID)
	if err != nil {
		return err
	}
	audit, err := h.tokens.ListAudit(user.ID, role)
	if err != nil {
		return err
	}
	// super_admin also sees the CRM write tools a write token unlocks.
	tools := make([]fiber.Map, 0)
	for _, t := range h.agent.ToolsFor(role, role == "super_admin") {
		tools = append(tools, fiber.Map{"name": t.Name, "title": t.Title, "description": t.Description, "write": t.Write})
	}
	base := h.appURL
	if base == "" {
		base = c.Protocol() + "://" + c.Hostname()
	}
	return h.inertiaService.Render(c, "app/AksesAI", fiber.Map{
		"user":        user,
		"diizinkan":   services.AIRoleAllowed(role),
		"mcp_url":     base + "/mcp",
		"tokens":      tokens,
		"audit":       audit,
		"audit_semua": role == "super_admin",
		"tools":       tools,
		"boleh_tulis": role == "super_admin",
	})
}

// Store creates a token and returns its plaintext once as JSON (called with
// fetch, so the secret never passes through the session/flash store).
func (h *AIAksesHandler) Store(c *fiber.Ctx) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	var req struct {
		Nama            string `json:"nama"`
		MasaBerlakuHari int    `json:"masa_berlaku_hari"`
		CanWrite        bool   `json:"can_write"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	created, err := h.tokens.Create(user.ID, string(user.Role), req.Nama, req.MasaBerlakuHari, req.CanWrite)
	if err != nil {
		status := fiber.StatusUnprocessableEntity
		if errors.Is(err, services.ErrAIRoleDitolak) || errors.Is(err, services.ErrAITokenWriteDitolak) {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(created)
}

func (h *AIAksesHandler) Revoke(c *fiber.Ctx) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if id <= 0 {
		return fiber.ErrBadRequest
	}
	if err := h.tokens.Revoke(user.ID, id); err != nil {
		h.store.Flash(c, "error", "Token tidak ditemukan atau sudah dicabut")
	} else {
		h.store.Flash(c, "success", "Token dicabut. AI agent yang memakainya tidak bisa mengakses data lagi.")
	}
	return h.inertiaService.Redirect(c, "/app/ai-akses")
}
