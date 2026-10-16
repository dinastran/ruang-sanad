package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// KeuanganDashboardHandler serves the finance dashboard at /app/keuangan.
type KeuanganDashboardHandler struct {
	dashboard      *services.KeuanganDashboardService
	tagihan        *services.TagihanService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewKeuanganDashboardHandler(dashboard *services.KeuanganDashboardService, tagihan *services.TagihanService, store *session.Store, inertiaService *services.InertiaService) *KeuanganDashboardHandler {
	return &KeuanganDashboardHandler{dashboard: dashboard, tagihan: tagihan, store: store, inertiaService: inertiaService}
}

func (h *KeuanganDashboardHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	data, err := h.dashboard.Get(c.Query("bulan"))
	if errors.Is(err, services.ErrUjrohBulanTidakValid) {
		return h.inertiaService.Redirect(c, "/app/keuangan")
	}
	if err != nil {
		return err
	}
	templates, err := h.tagihan.ListTagihanTemplates()
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "keuangan/Dashboard", fiber.Map{"user": sessionUser(sess), "dashboard": data, "templates": templates})
}
