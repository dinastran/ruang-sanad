package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type KoordinatorGuruHandler struct {
	svc            *services.KoordinatorService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewKoordinatorGuruHandler(svc *services.KoordinatorService, store *session.Store, inertiaService *services.InertiaService) *KoordinatorGuruHandler {
	return &KoordinatorGuruHandler{
		svc:            svc,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *KoordinatorGuruHandler) Dashboard(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	dashboard, err := h.svc.GetDashboard()
	if err != nil {
		return h.inertiaService.Render(c, "koordinator/Dashboard", fiber.Map{"user": user, "error": "Gagal memuat dashboard"})
	}
	return h.inertiaService.Render(c, "koordinator/Dashboard", fiber.Map{
		"user":      user,
		"dashboard": dashboard,
	})
}
