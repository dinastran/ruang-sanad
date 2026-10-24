package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// IntegrasiWhatsAppHandler serves /app/integrasi-whatsapp (super_admin):
// GOWA/bridge status and CRM settings.
type IntegrasiWhatsAppHandler struct {
	integration    *services.CRMIntegrationService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewIntegrasiWhatsAppHandler(integration *services.CRMIntegrationService, store *session.Store, inertiaService *services.InertiaService) *IntegrasiWhatsAppHandler {
	return &IntegrasiWhatsAppHandler{integration: integration, store: store, inertiaService: inertiaService}
}

func (h *IntegrasiWhatsAppHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	page, err := h.integration.Page(c.UserContext())
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/IntegrasiWhatsApp", fiber.Map{
		"user": sessionUser(sess),
		"page": page,
	})
}

func (h *IntegrasiWhatsAppHandler) UpdateSettings(c *fiber.Ctx) error {
	var req models.CRMSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Pengaturan tidak valid")
		return h.inertiaService.Redirect(c, "/app/integrasi-whatsapp")
	}
	if err := h.integration.UpdateSettings(c.UserContext(), c.Locals("user_id").(int64), req); err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", "Pengaturan CRM disimpan")
	}
	return h.inertiaService.Redirect(c, "/app/integrasi-whatsapp")
}
