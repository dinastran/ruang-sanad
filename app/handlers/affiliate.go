package handlers

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

const affiliateAdminPath = "/app/leads/affiliate"

// AffiliateHandler serves "Affiliate Saya" (/app/affiliate), the admin page
// (/app/leads/affiliate), the lead panel action, and the public signup and
// activation pages.
type AffiliateHandler struct {
	aff            *services.AffiliateService
	store          *session.Store
	inertiaService *services.InertiaService
	sendMail       func(to, subject, body string) error // nil = SMTP off
}

func NewAffiliateHandler(aff *services.AffiliateService, store *session.Store, inertia *services.InertiaService) *AffiliateHandler {
	return &AffiliateHandler{aff: aff, store: store, inertiaService: inertia}
}

// WithMailer enables activation emails (only when SMTP is configured).
func (h *AffiliateHandler) WithMailer(send func(to, subject, body string) error) *AffiliateHandler {
	h.sendMail = send
	return h
}

func userIDOf(c *fiber.Ctx) int64 {
	id, _ := c.Locals("user_id").(int64)
	return id
}

// --- Affiliate Saya ---

func (h *AffiliateHandler) Me(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	ctx := c.UserContext()
	props := fiber.Map{"user": sessionUser(sess), "base_url": c.BaseURL()}
	a, err := h.aff.ForUser(ctx, userIDOf(c))
	if errors.Is(err, services.ErrAffiliateTidakAda) {
		props["belum_terdaftar"] = true
		return h.inertiaService.Render(c, "app/AffiliateSaya", props)
	}
	if err != nil {
		return err
	}
	d, err := h.aff.Dashboard(ctx, a.ID)
	if err != nil {
		return err
	}
	props["affiliate"] = d
	return h.inertiaService.Render(c, "app/AffiliateSaya", props)
}

func (h *AffiliateHandler) UpdateRekening(c *fiber.Ctx) error {
	var req struct {
		BankNama     string `json:"bank_nama"`
		BankRekening string `json:"bank_rekening"`
		BankAtasNama string `json:"bank_atas_nama"`
	}
	a, err := h.aff.ForUser(c.UserContext(), userIDOf(c))
	if err != nil || c.BodyParser(&req) != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/affiliate")
	}
	if err := h.aff.UpdateBank(c.UserContext(), a.ID, req.BankNama, req.BankRekening, req.BankAtasNama); err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", "Rekening disimpan")
	}
	return h.inertiaService.Redirect(c, "/app/affiliate")
}

// --- Admin ---

func (h *AffiliateHandler) Admin(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	ctx := c.UserContext()
	rows, err := h.aff.AdminRows(ctx)
	if err != nil {
		return err
	}
	rules, err := h.aff.Rules(ctx)
	if err != nil {
		return err
	}
	st, err := h.aff.Settings(ctx)
	if err != nil {
		return err
	}
	ledger, err := h.aff.Ledger(ctx, c.Query("status"))
	if err != nil {
		return err
	}
	opts, err := h.aff.AdminOptions(ctx)
	if err != nil {
		return err
	}
	props := fiber.Map{
		"user": sessionUser(sess), "affiliates": rows, "rules": rules, "ledger": ledger, "status_filter": c.Query("status"),
		"settings": fiber.Map{
			"masa_berlaku_hari": st.MasaBerlakuHari, "masa_tahan_hari": st.MasaTahanHari,
			"link_tujuan_default_id": st.LinkTujuanDefaultID.Int64, "template_bagikan": st.TemplateBagikan,
		},
		"programs": opts.Programs, "links": opts.Links, "base_url": c.BaseURL(), "tab": c.Query("tab", "affiliate"),
		"can_edit": roleOf(c) == "super_admin",
	}
	if inv := sess.Get("affiliate_invite"); inv != nil {
		props["invite"] = inv
		sess.Delete("affiliate_invite")
		_ = sess.Save()
	}
	return h.inertiaService.Render(c, "app/AffiliateAdmin", props)
}

func (h *AffiliateHandler) adminDone(c *fiber.Ctx, tab string, err error, ok string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else if ok != "" {
		h.store.Flash(c, "success", ok)
	}
	return h.inertiaService.Redirect(c, affiliateAdminPath+"?tab="+tab)
}

func affParam(c *fiber.Ctx, name string) int64 {
	id, _ := strconv.ParseInt(c.Params(name), 10, 64)
	return id
}

func (h *AffiliateHandler) Approve(c *fiber.Ctx) error {
	ctx := c.UserContext()
	inv, err := h.aff.Approve(ctx, affParam(c, "aid"), userIDOf(c))
	if err != nil {
		return h.adminDone(c, "affiliate", err, "")
	}
	return h.sendInvite(c, inv, "Affiliate disetujui")
}

func (h *AffiliateHandler) Invite(c *fiber.Ctx) error {
	inv, err := h.aff.NewInvite(c.UserContext(), affParam(c, "aid"))
	if err != nil {
		return h.adminDone(c, "affiliate", err, "")
	}
	return h.sendInvite(c, inv, "Link aktivasi baru dibuat")
}

// sendInvite emails the activation link when SMTP is on and keeps the link
// for the admin to copy or send by WhatsApp.
func (h *AffiliateHandler) sendInvite(c *fiber.Ctx, inv services.Invite, ok string) error {
	link := c.BaseURL() + inv.Path
	emailed := false
	if h.sendMail != nil && inv.Email != "" {
		body := "<p>Assalamu'alaikum,</p><p>Pendaftaran affiliate Ruang Sanad Anda sudah disetujui. Buat password untuk login lewat link berikut (berlaku 3 hari):</p>" +
			"<p><a href=\"" + link + "\">" + link + "</a></p>"
		if err := h.sendMail(inv.Email, "Aktivasi akun affiliate Ruang Sanad", body); err != nil {
			slog.Warn("affiliate: email aktivasi gagal", "error", err)
		} else {
			emailed = true
		}
	}
	if sess, err := h.store.Get(c); err == nil {
		sess.Set("affiliate_invite", map[string]any{"link": link, "wa_url": inv.WaURL(c.BaseURL()), "email": inv.Email, "emailed": emailed})
		_ = sess.Save()
	}
	return h.adminDone(c, "affiliate", nil, ok)
}

func (h *AffiliateHandler) SetStatus(c *fiber.Ctx) error {
	var req struct {
		Status string `json:"status"`
	}
	if c.BodyParser(&req) != nil {
		return h.adminDone(c, "affiliate", errors.New("data tidak valid"), "")
	}
	return h.adminDone(c, "affiliate", h.aff.SetStatus(c.UserContext(), affParam(c, "aid"), req.Status, userIDOf(c)), "Status affiliate diperbarui")
}

func (h *AffiliateHandler) UpdateKode(c *fiber.Ctx) error {
	var req struct {
		Kode string `json:"kode"`
	}
	if c.BodyParser(&req) != nil {
		return h.adminDone(c, "affiliate", errors.New("data tidak valid"), "")
	}
	return h.adminDone(c, "affiliate", h.aff.UpdateKode(c.UserContext(), affParam(c, "aid"), req.Kode), "Kode diperbarui")
}

func (h *AffiliateHandler) SaveRule(c *fiber.Ctx) error {
	var req models.AffiliateRuleRequest
	if c.BodyParser(&req) != nil {
		return h.adminDone(c, "aturan", errors.New("data aturan tidak valid"), "")
	}
	return h.adminDone(c, "aturan", h.aff.SaveRule(c.UserContext(), req, userIDOf(c)), "Aturan komisi disimpan")
}

func (h *AffiliateHandler) DeleteRule(c *fiber.Ctx) error {
	return h.adminDone(c, "aturan", h.aff.DeleteRule(c.UserContext(), affParam(c, "rid")), "Aturan dihapus")
}

func (h *AffiliateHandler) SaveSettings(c *fiber.Ctx) error {
	var req models.AffiliateSettings
	if c.BodyParser(&req) != nil {
		return h.adminDone(c, "pengaturan", errors.New("pengaturan tidak valid"), "")
	}
	return h.adminDone(c, "pengaturan", h.aff.SaveSettings(c.UserContext(), req, userIDOf(c)), "Pengaturan disimpan")
}

func (h *AffiliateHandler) Transfer(c *fiber.Ctx) error {
	var req struct {
		IDs     []int64 `json:"ids"`
		Catatan string  `json:"catatan"`
	}
	if c.BodyParser(&req) != nil || len(req.IDs) == 0 {
		return h.adminDone(c, "komisi", errors.New("pilih komisi yang sudah ditransfer"), "")
	}
	n, err := h.aff.MarkTransferred(c.UserContext(), req.IDs, userIDOf(c), req.Catatan)
	return h.adminDone(c, "komisi", err, strconv.Itoa(n)+" komisi ditandai sudah ditransfer")
}

func (h *AffiliateHandler) CancelCommission(c *fiber.Ctx) error {
	var req struct {
		Alasan string `json:"alasan"`
	}
	_ = c.BodyParser(&req)
	return h.adminDone(c, "komisi", h.aff.CancelCommission(c.UserContext(), affParam(c, "cid"), userIDOf(c), req.Alasan), "Komisi dibatalkan")
}

func (h *AffiliateHandler) Reconcile(c *fiber.Ctx) error {
	if _, err := h.aff.EnsureGuruAffiliates(c.UserContext()); err != nil {
		return h.adminDone(c, "komisi", err, "")
	}
	r, err := h.aff.Reconcile(c.UserContext())
	return h.adminDone(c, "komisi", err, "Dihitung ulang: "+strconv.Itoa(r.Dibuat)+" komisi baru, "+strconv.Itoa(r.Siap)+" siap cair, "+strconv.Itoa(r.Batal)+" batal")
}

// SetLeadAffiliate is the lead page action (CS / super_admin).
func (h *AffiliateHandler) SetLeadAffiliate(c *fiber.Ctx) error {
	leadID := affParam(c, "id")
	var req struct {
		AffiliateID int64 `json:"affiliate_id"`
	}
	back := "/app/leads/" + strconv.FormatInt(leadID, 10)
	if c.BodyParser(&req) != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.aff.SetLeadAffiliate(c.UserContext(), leadID, req.AffiliateID, userIDOf(c)); err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", "Affiliate lead diperbarui")
	}
	return h.inertiaService.Redirect(c, back)
}

// --- Public: signup & activation ---

func (h *AffiliateHandler) SignupPage(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	return h.inertiaService.Render(c, "public/AffiliateDaftar", fiber.Map{})
}

func (h *AffiliateHandler) SignupSubmit(c *fiber.Ctx) error {
	var in models.AffiliateSignup
	if c.BodyParser(&in) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}
	if in.Honeypot != "" {
		return c.JSON(fiber.Map{"ok": true})
	}
	if !in.Setuju {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"field": "setuju", "error": "Centang persetujuan syarat affiliate"})
	}
	if _, err := h.aff.Register(c.UserContext(), in); err != nil {
		return registrationError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AffiliateHandler) ActivatePage(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	return h.inertiaService.Render(c, "public/AffiliateAktivasi", fiber.Map{"token": c.Params("token")})
}

func (h *AffiliateHandler) ActivateSubmit(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}
	if c.BodyParser(&req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}
	if err := h.aff.Activate(c.UserContext(), c.Params("token"), req.Password); err != nil {
		var fe *services.FieldError
		if errors.As(err, &fe) || errors.Is(err, services.ErrUndanganTidakValid) {
			return registrationError(c, err)
		}
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}
