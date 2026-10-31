package handlers

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

const leadSourcesPath = "/app/leads/sumber"

// LeadSourceHandler serves /app/leads/sumber: the lead source catalog
// (landing page / iklan) and its trigger texts, plus setting a lead's source.
type LeadSourceHandler struct {
	leads          *services.LeadService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLeadSourceHandler(leads *services.LeadService, store *session.Store, inertiaService *services.InertiaService) *LeadSourceHandler {
	return &LeadSourceHandler{leads: leads, store: store, inertiaService: inertiaService}
}

func (h *LeadSourceHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	sources, err := h.leads.Sources(c.UserContext())
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/LeadSources", fiber.Map{
		"user":    sessionUser(sess),
		"sources": sources,
	})
}

// done flashes the outcome and returns to the catalog.
func (h *LeadSourceHandler) done(c *fiber.Ctx, err error, success string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", success)
	}
	return h.inertiaService.Redirect(c, leadSourcesPath)
}

func sourceIDParam(c *fiber.Ctx, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Params(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("data tidak valid")
	}
	return id, nil
}

func (h *LeadSourceHandler) Create(c *fiber.Ctx) error {
	var req models.LeadSourceRequest
	if c.BodyParser(&req) != nil {
		return h.done(c, errors.New("data tidak valid"), "")
	}
	_, err := h.leads.CreateSource(c.UserContext(), req)
	return h.done(c, err, "Sumber lead ditambahkan")
}

func (h *LeadSourceHandler) Update(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "sid")
	var req models.LeadSourceRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, errors.New("data tidak valid"), "")
	}
	return h.done(c, h.leads.UpdateSource(c.UserContext(), id, req), "Sumber lead disimpan")
}

func (h *LeadSourceHandler) Delete(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "sid")
	if err != nil {
		return h.done(c, err, "")
	}
	return h.done(c, h.leads.DeleteSource(c.UserContext(), id), "Sumber lead dihapus")
}

func (h *LeadSourceHandler) AddPattern(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "sid")
	var req models.LeadSourcePatternRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, errors.New("data tidak valid"), "")
	}
	return h.done(c, h.leads.AddSourcePattern(c.UserContext(), id, req.Teks), "Teks pemicu ditambahkan. Lead baru dengan teks ini otomatis tercatat.")
}

func (h *LeadSourceHandler) DeletePattern(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "pid")
	if err != nil {
		return h.done(c, err, "")
	}
	return h.done(c, h.leads.DeleteSourcePattern(c.UserContext(), id), "Teks pemicu dihapus. Lead yang sudah tercatat tidak berubah.")
}

func (h *LeadSourceHandler) Rematch(c *fiber.Ctx) error {
	n, err := h.leads.RematchSources(c.UserContext())
	return h.done(c, err, fmt.Sprintf("%d lead tanpa sumber berhasil dicocokkan", n))
}

// SetForLead corrects one lead's source from the lead page.
func (h *LeadSourceHandler) SetForLead(c *fiber.Ctx) error {
	leadID, ok := leadIDParam(c)
	var req models.LeadSourceSetRequest
	if ok && c.BodyParser(&req) == nil {
		err := h.leads.SetLeadSourceManual(c.UserContext(), leadID, c.Locals("user_id").(int64), req.SourceID)
		if err != nil {
			h.store.Flash(c, "error", err.Error())
		} else {
			h.store.Flash(c, "success", "Sumber lead diperbarui")
		}
	} else {
		h.store.Flash(c, "error", "data tidak valid")
	}
	if !ok {
		return h.inertiaService.Redirect(c, "/app/leads")
	}
	return h.inertiaService.Redirect(c, leadPath(leadID))
}
