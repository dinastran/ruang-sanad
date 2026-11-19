package handlers

import (
	"errors"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
)

// RegistrationPublicHandler serves the public registration form
// (/daftar/:slug) and the manual payment page. No login, no CSRF: the form
// carries a signed render token and the routes are rate limited.
type RegistrationPublicHandler struct {
	reg            *services.RegistrationService
	inertiaService *services.InertiaService
}

func NewRegistrationPublicHandler(reg *services.RegistrationService, inertia *services.InertiaService) *RegistrationPublicHandler {
	return &RegistrationPublicHandler{reg: reg, inertiaService: inertia}
}

func (h *RegistrationPublicHandler) Page(c *fiber.Ctx) error {
	ctx := c.UserContext()
	slug := c.Params("slug")
	c.Set(fiber.HeaderCacheControl, "no-store")
	link, err := h.reg.PublicLink(ctx, slug)
	if err != nil {
		c.Status(fiber.StatusNotFound)
		return h.inertiaService.Render(c, "public/Daftar", fiber.Map{
			"galat":     "Link pendaftaran ini sudah tidak aktif.",
			"wa_cs_url": h.reg.CSWaURL(ctx, "Assalamu'alaikum, saya ingin mendaftar."),
		})
	}
	_ = h.reg.CountClick(ctx, link.ID, c.IP(), c.Get(fiber.HeaderUserAgent))
	if link.Mode == "wa" {
		return c.Redirect(h.reg.LinkWaURL(ctx, slug), fiber.StatusFound)
	}
	props := fiber.Map{"link": link, "form_token": h.reg.FormToken(slug)}
	if link.ButuhJadwal {
		jadwal, err := h.reg.PublicJadwal(ctx)
		if err != nil {
			return err
		}
		props["jadwal"] = jadwal
	}
	return h.inertiaService.Render(c, "public/Daftar", props)
}

func registrationError(c *fiber.Ctx, err error) error {
	var fe *services.FieldError
	switch {
	case errors.As(err, &fe):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"field": fe.Field, "error": fe.Pesan})
	case errors.Is(err, services.ErrLinkTidakAktif), errors.Is(err, services.ErrRefTidakAda):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, services.ErrFormTerlaluCepat), errors.Is(err, services.ErrFormKedaluwarsa),
		errors.Is(err, services.ErrFormTidakValid), errors.Is(err, services.ErrSudahDiproses):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Maaf, pendaftaran belum berhasil. Silakan coba lagi atau hubungi CS lewat WhatsApp."})
}

func (h *RegistrationPublicHandler) Submit(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	var in models.RegistrationSubmit
	if c.BodyParser(&in) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}
	res, err := h.reg.Submit(c.UserContext(), c.Params("slug"), in, models.RequestMeta{IP: c.IP(), UserAgent: c.Get(fiber.HeaderUserAgent)})
	if err != nil {
		return registrationError(c, err)
	}
	out := fiber.Map{"next": res.Next, "wa_url": res.WaURL, "lead_ref": res.LeadRef}
	if res.BayarRef != "" {
		out["bayar_url"] = "/daftar/bayar/" + res.BayarRef
	}
	return c.JSON(out)
}

func (h *RegistrationPublicHandler) BayarPage(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	ref := c.Params("ref")
	info, err := h.reg.BayarInfo(c.UserContext(), ref)
	if err != nil {
		c.Status(fiber.StatusNotFound)
		return h.inertiaService.Render(c, "public/DaftarBayar", fiber.Map{"galat": "Tautan pembayaran tidak dikenal."})
	}
	return h.inertiaService.Render(c, "public/DaftarBayar", fiber.Map{"info": info, "ref": ref})
}

func (h *RegistrationPublicHandler) UploadBukti(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	fh, err := c.FormFile("bukti")
	if err != nil {
		return registrationError(c, &services.FieldError{Field: "bukti", Pesan: "Pilih file bukti transfer"})
	}
	src, err := fh.Open()
	if err != nil {
		return registrationError(c, err)
	}
	head := make([]byte, 16)
	n, _ := io.ReadFull(src, head)
	_ = src.Close()
	err = h.reg.SubmitManualProof(c.UserContext(), c.Params("ref"), services.ProofFile{
		Name: fh.Filename, Size: fh.Size, Head: head[:n],
		Save: func(dst string) error { return c.SaveFile(fh, dst) },
	})
	if err != nil {
		return registrationError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
