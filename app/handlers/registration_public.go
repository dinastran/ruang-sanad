package handlers

import (
	"errors"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
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
	_ = h.reg.CountClick(ctx, link.ID, session.ClientIP(c), c.Get(fiber.HeaderUserAgent))
	if link.Mode == "wa" {
		waURL := h.reg.LinkWaURL(ctx, slug)
		// With events set for the WA trigger, show a short page that fires the
		// Pixel and then opens WhatsApp; otherwise redirect right away.
		if meta := h.reg.WaTracking(ctx, slug, metaSignals(c, c.BaseURL()+c.OriginalURL())); meta != nil {
			return h.inertiaService.Render(c, "public/DaftarWa", fiber.Map{"wa_url": waURL, "meta": meta, "judul": link.Judul})
		}
		return c.Redirect(waURL, fiber.StatusFound)
	}
	h.reg.TrackPageView(link, metaSignals(c, c.BaseURL()+c.OriginalURL()))
	props := fiber.Map{
		"link": link, "form_token": h.reg.FormToken(slug),
		"wa_cs_url": h.reg.CSWaURL(ctx, "Assalamu'alaikum, saya ingin bertanya dulu tentang "+link.ProgramNama+"."),
	}
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
		errors.Is(err, services.ErrFormTidakValid), errors.Is(err, services.ErrSudahDiproses),
		errors.Is(err, services.ErrUndanganTidakValid):
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
	ctx := c.UserContext()
	sig := metaSignals(c, in.PageURL)
	res, err := h.reg.Submit(ctx, c.Params("slug"), in, models.RequestMeta{
		IP: sig.IP, UserAgent: sig.UserAgent, AffCookie: c.Cookies(services.AffCookieName),
		Fbp: sig.Fbp, Fbc: sig.Fbc, PageURL: sig.SourceURL,
	})
	var fe *services.FieldError
	switch {
	case errors.Is(err, services.ErrLinkTidakAktif):
		// Link closed while the visitor was filling the form: give them CS.
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":     "Link pendaftaran ini sudah tidak aktif. Silakan lanjutkan lewat WhatsApp CS.",
			"wa_cs_url": h.reg.CSWaURL(ctx, "Assalamu'alaikum, saya ingin mendaftar."),
		})
	case errors.As(err, &fe) && fe.Field == "jadwal_kode":
		// Send fresh seat counts so the form can update without a reload.
		jadwal, jerr := h.reg.PublicJadwal(ctx)
		if jerr != nil {
			return registrationError(c, err)
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"field": fe.Field, "error": fe.Pesan, "jadwal": jadwal})
	case err != nil:
		return registrationError(c, err)
	}
	out := fiber.Map{"next": res.Next, "wa_url": res.WaURL, "lead_ref": res.LeadRef}
	if res.BayarRef != "" {
		out["bayar_url"] = "/daftar/bayar/" + res.BayarRef
	}
	if res.Meta != nil {
		out["meta"] = res.Meta
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

// StartGateway creates (or reuses) the online checkout session.
func (h *RegistrationPublicHandler) StartGateway(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	url, err := h.reg.StartGatewayPayment(c.UserContext(), c.Params("ref"))
	if errors.Is(err, services.ErrGatewayNonaktif) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Pembayaran online belum tersedia. Silakan gunakan transfer manual."})
	}
	if err != nil {
		var fe *services.FieldError
		if errors.As(err, &fe) || errors.Is(err, services.ErrSudahDiproses) || errors.Is(err, services.ErrRefTidakAda) {
			return registrationError(c, err)
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"url": url})
}

// BayarStatus is polled by the payment page while a session is open.
func (h *RegistrationPublicHandler) BayarStatus(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	info, err := h.reg.BayarInfo(c.UserContext(), c.Params("ref"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": info.Status, "sesi_aktif": info.SesiAktif})
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
	out := fiber.Map{"ok": true}
	// Best effort: tracking must never fail an accepted upload.
	if meta := h.reg.ProofTracking(c.UserContext(), c.Params("ref"), metaSignals(c, c.FormValue("page_url"))); meta != nil {
		out["meta"] = meta
	}
	return c.JSON(out)
}

// metaSignals reads the visitor's Meta browser identifiers. pageURL is the
// page the visitor is on; a foreign or invalid URL falls back to this
// request's page (for event_source_url).
func metaSignals(c *fiber.Ctx, pageURL string) services.MetaWebSignals {
	ref := c.Get(fiber.HeaderReferer)
	if ref == "" {
		ref = c.BaseURL() + c.Path()
	}
	src := services.SafePageURL(pageURL, services.SafePageURL(ref, c.BaseURL()+c.Path()))
	return services.MetaWebSignals{
		Fbp: services.MetaFbp(c.Cookies("_fbp")), Fbc: services.MetaFbc(c.Cookies("_fbc"), src, time.Now()),
		IP: session.ClientIP(c), UserAgent: c.Get(fiber.HeaderUserAgent), SourceURL: src,
	}
}
