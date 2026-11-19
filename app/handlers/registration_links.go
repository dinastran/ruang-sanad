package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

const registrationLinksPath = "/app/leads/link-pendaftaran"

// RegistrationLinkHandler is the admin page for public registration links,
// plus the payment proof viewer/reject actions on a lead.
type RegistrationLinkHandler struct {
	reg            *services.RegistrationService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewRegistrationLinkHandler(reg *services.RegistrationService, store *session.Store, inertia *services.InertiaService) *RegistrationLinkHandler {
	return &RegistrationLinkHandler{reg: reg, store: store, inertiaService: inertia}
}

func (h *RegistrationLinkHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	ctx := c.UserContext()
	links, err := h.reg.ListLinks(ctx)
	if err != nil {
		return err
	}
	programs, sources, err := h.reg.LinkFormOptions(ctx)
	if err != nil {
		return err
	}
	settings, err := h.reg.Settings(ctx)
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/RegistrationLinks", fiber.Map{
		"user": sessionUser(sess), "links": links, "programs": programs, "sources": sources,
		"settings": settings, "base_url": c.BaseURL(),
	})
}

func (h *RegistrationLinkHandler) done(c *fiber.Ctx, err error, ok string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", ok)
	}
	return h.inertiaService.Redirect(c, registrationLinksPath)
}

func actorID(c *fiber.Ctx) int64 {
	id, _ := c.Locals("user_id").(int64)
	return id
}

func (h *RegistrationLinkHandler) Create(c *fiber.Ctx) error {
	var req models.RegistrationLinkRequest
	if c.BodyParser(&req) != nil {
		return h.done(c, errors.New("data link tidak valid"), "")
	}
	_, err := h.reg.CreateLink(c.UserContext(), req, actorID(c))
	return h.done(c, err, "Link pendaftaran dibuat")
}

func (h *RegistrationLinkHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("lid"), 10, 64)
	var req models.RegistrationLinkRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, errors.New("data link tidak valid"), "")
	}
	return h.done(c, h.reg.UpdateLink(c.UserContext(), id, req), "Link pendaftaran diperbarui")
}

func (h *RegistrationLinkHandler) SaveSettings(c *fiber.Ctx) error {
	var req models.RegistrationSettings
	if c.BodyParser(&req) != nil {
		return h.done(c, errors.New("pengaturan tidak valid"), "")
	}
	return h.done(c, h.reg.SaveSettings(c.UserContext(), req, actorID(c)), "Pengaturan disimpan")
}

func paymentParams(c *fiber.Ctx) (leadID, paymentID int64, ok bool) {
	leadID, err1 := strconv.ParseInt(c.Params("id"), 10, 64)
	paymentID, err2 := strconv.ParseInt(c.Params("pid"), 10, 64)
	return leadID, paymentID, err1 == nil && err2 == nil
}

// Bukti streams a stored payment proof to staff.
func (h *RegistrationLinkHandler) Bukti(c *fiber.Ctx) error {
	leadID, paymentID, ok := paymentParams(c)
	if !ok {
		return fiber.ErrNotFound
	}
	path, err := h.reg.ProofPath(c.UserContext(), leadID, paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Bukti tidak ditemukan")
	}
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	if err := c.SendFile(path); err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Bukti tidak ditemukan")
	}
	if c.Response().StatusCode() == fiber.StatusNotFound {
		return c.Status(fiber.StatusNotFound).SendString("Bukti tidak ditemukan")
	}
	return nil
}

func (h *RegistrationLinkHandler) Tolak(c *fiber.Ctx) error {
	leadID, paymentID, ok := paymentParams(c)
	var req struct {
		Alasan string `json:"alasan"`
	}
	back := "/app/leads/" + strconv.FormatInt(leadID, 10)
	if !ok || c.BodyParser(&req) != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.reg.RejectManualPayment(c.UserContext(), leadID, paymentID, actorID(c), req.Alasan); err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", "Bukti transfer ditolak; pendaftar bisa mengunggah ulang")
	}
	return h.inertiaService.Redirect(c, back)
}
