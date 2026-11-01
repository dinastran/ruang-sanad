package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

const metaCAPIPath = "/app/leads/meta-capi"

// MetaCAPIHandler serves /app/leads/meta-capi: the status of the Meta
// Conversions API queue (super_admin only).
type MetaCAPIHandler struct {
	capi           *services.MetaCAPIService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewMetaCAPIHandler(capi *services.MetaCAPIService, store *session.Store, inertiaService *services.InertiaService) *MetaCAPIHandler {
	return &MetaCAPIHandler{capi: capi, store: store, inertiaService: inertiaService}
}

func (h *MetaCAPIHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	status, err := h.capi.Status(c.UserContext())
	if err != nil {
		return err
	}
	analytics, err := h.capi.Analytics(c.UserContext(), c.Query("from"), c.Query("to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	return h.inertiaService.Render(c, "app/MetaCAPI", fiber.Map{
		"user":   sessionUser(sess),
		"status": status,
		"analytics": analytics,
	})
}

// ResendTest sends the events that went out in test mode again, to production.
func (h *MetaCAPIHandler) ResendTest(c *fiber.Ctx) error {
	n, err := h.capi.ResendTestEvents(c.UserContext())
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", fmt.Sprintf("%d event uji dimasukkan ke antrean produksi; terkirim bertahap dalam beberapa menit", n))
	}
	return h.inertiaService.Redirect(c, metaCAPIPath)
}

// RetryFailed puts failed events (still within Meta's 7-day window) back in the queue.
func (h *MetaCAPIHandler) RetryFailed(c *fiber.Ctx) error {
	n, err := h.capi.RetryFailed(c.UserContext())
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", fmt.Sprintf("%d event dimasukkan lagi ke antrean kirim", n))
	}
	return h.inertiaService.Redirect(c, metaCAPIPath)
}
