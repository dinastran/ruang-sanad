package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type DashboardHandler struct {
	overview       *services.OverviewService
	userService    *services.UserService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewDashboardHandler(overview *services.OverviewService, userService *services.UserService, store *session.Store, inertiaService *services.InertiaService) *DashboardHandler {
	return &DashboardHandler{overview: overview, userService: userService, store: store, inertiaService: inertiaService}
}

func (h *DashboardHandler) Index(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if sess.Get("user_id") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	user := sessionUser(sess)
	role, err := h.userService.GetUserRole(user.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	user.Role = models.UserRole(role)
	switch role {
	case "user":
		return h.inertiaService.Redirect(c, "/app/profile")
	case "cs":
		return h.inertiaService.Redirect(c, "/app/santri")
	case "admin_kelas":
		return h.inertiaService.Redirect(c, "/app/kelas")
	case "guru":
		return h.inertiaService.Redirect(c, "/app/guru")
	case "koordinator_guru":
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}

	if !services.OverviewCanFinance(role) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	overview, err := h.overview.Get(c.UserContext(), c.Query("bulan"), role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	return h.inertiaService.Render(c, "app/Dashboard", fiber.Map{
		"user": user, "overview": overview,
	})
}
