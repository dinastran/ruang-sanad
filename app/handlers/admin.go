package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type AdminHandler struct {
	userService    *services.UserService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewAdminHandler(userService *services.UserService, store *session.Store, inertiaService *services.InertiaService) *AdminHandler {
	return &AdminHandler{
		userService:    userService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *AdminHandler) Dashboard(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	return h.inertiaService.Render(c, "admin/Dashboard", fiber.Map{
		"user": user,
	})
}

func (h *AdminHandler) Users(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	users, err := h.userService.ListUsers()
	if err != nil {
		return h.inertiaService.Render(c, "admin/Users", fiber.Map{
			"user":  user,
			"error": "Gagal memuat user",
		})
	}

	return h.inertiaService.Render(c, "admin/Users", fiber.Map{
		"user":  user,
		"users": users,
	})
}

func (h *AdminHandler) UpdateUserRole(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/admin/users")
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/admin/users")
	}

	if err := h.userService.UpdateUserRole(id, req.Role); err != nil {
		h.store.Flash(c, "error", "Gagal update role")
		return h.inertiaService.Redirect(c, "/admin/users")
	}

	h.store.Flash(c, "success", "Role berhasil diupdate")
	return h.inertiaService.Redirect(c, "/admin/users")
}
