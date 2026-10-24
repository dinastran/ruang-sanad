package handlers

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// LeadClosingHandler handles payment verification and "Jadikan Santri".
type LeadClosingHandler struct {
	leads          *services.LeadService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLeadClosingHandler(leads *services.LeadService, store *session.Store, inertiaService *services.InertiaService) *LeadClosingHandler {
	return &LeadClosingHandler{leads: leads, store: store, inertiaService: inertiaService}
}

func (h *LeadClosingHandler) Verify(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadVerifyRequest
	if !ok || c.BodyParser(&req) != nil {
		h.store.Flash(c, "error", "Data verifikasi tidak valid")
		return h.inertiaService.Redirect(c, leadPath(id))
	}
	if err := h.leads.VerifyPayment(c.UserContext(), id, c.Locals("user_id").(int64), req); err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", "Pembayaran terverifikasi. Lanjutkan dengan Jadikan Santri.")
	}
	return h.inertiaService.Redirect(c, leadPath(id))
}

func (h *LeadClosingHandler) Convert(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadConvertRequest
	if !ok || c.BodyParser(&req) != nil {
		h.store.Flash(c, "error", "Data santri tidak valid")
		return h.inertiaService.Redirect(c, leadPath(id))
	}
	santriID, err := h.leads.Convert(c.UserContext(), id, c.Locals("user_id").(int64), req)
	var dup *services.ErrLeadDuplicateSantri
	switch {
	case errors.As(err, &dup):
		h.store.Flash(c, "error", dup.Error()+". Pilih tautkan ke santri itu, atau centang 'tetap buat santri baru'.")
		return h.inertiaService.Redirect(c, leadPath(id))
	case err != nil:
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, leadPath(id))
	}
	if req.LinkSantriID > 0 {
		h.store.Flash(c, "success", "Lead ditautkan ke santri yang sudah ada")
	} else {
		h.store.Flash(c, "success", "Santri baru dibuat. Admin Kelas sudah diberi notifikasi.")
	}
	// Keuangan cannot open /app/santri/:id, so it stays on the lead page.
	if role, _ := c.Locals("role").(string); role != "cs" && role != "super_admin" {
		return h.inertiaService.Redirect(c, leadPath(id))
	}
	return h.inertiaService.Redirect(c, fmt.Sprintf("/app/santri/%d", santriID))
}
