package handlers

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// SantriStatusHandler lets Admin Kelas (Data Santri) and Keuangan (Daftar
// Tagihan) change a santri's status (aktif, cuti, nonaktif).
type SantriStatusHandler struct {
	santriStatusService *services.SantriStatusService
	store               *session.Store
	inertiaService      *services.InertiaService
}

func NewSantriStatusHandler(santriStatusService *services.SantriStatusService, store *session.Store, inertiaService *services.InertiaService) *SantriStatusHandler {
	return &SantriStatusHandler{
		santriStatusService: santriStatusService,
		store:               store,
		inertiaService:      inertiaService,
	}
}

// localReturnURL returns the return_to query when it is a relative URL whose
// path passes allowed, otherwise fallback.
func localReturnURL(c *fiber.Ctx, fallback string, allowed func(path string) bool) string {
	parsed, err := url.Parse(c.Query("return_to", ""))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" || !allowed(parsed.Path) {
		return fallback
	}
	if parsed.RawQuery == "" {
		return parsed.Path
	}
	return parsed.Path + "?" + parsed.RawQuery
}

// santriListReturnURL keeps the Data Santri filters and page after a change.
func santriListReturnURL(c *fiber.Ctx) string {
	return localReturnURL(c, "/app/santri", func(path string) bool { return path == "/app/santri" })
}

// keuanganReturnURL returns to Daftar Tagihan (with its filters) or a Detail Tagihan page.
func keuanganReturnURL(c *fiber.Ctx) string {
	return localReturnURL(c, "/app/keuangan/tagihan", func(path string) bool {
		if path == "/app/keuangan/tagihan" {
			return true
		}
		id, ok := strings.CutPrefix(path, "/app/keuangan/tagihan/")
		if !ok {
			return false
		}
		_, err := strconv.ParseInt(id, 10, 64)
		return err == nil
	})
}

func (h *SantriStatusHandler) Update(c *fiber.Ctx) error {
	return h.update(c, santriListReturnURL(c), h.santriStatusService.UbahStatusSantri)
}

// UpdateKeuangan lets Keuangan change a santri's status from Daftar Tagihan.
func (h *SantriStatusHandler) UpdateKeuangan(c *fiber.Ctx) error {
	return h.update(c, keuanganReturnURL(c), h.santriStatusService.UbahStatusOlehKeuangan)
}

func (h *SantriStatusHandler) update(c *fiber.Ctx, returnURL string, ubah func(santriID, actorID int64, req models.UpdateSantriStatusRequest) error) error {
	santriID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		h.store.Flash(c, "error", "Santri tidak valid")
		return h.inertiaService.Redirect(c, returnURL)
	}
	var req models.UpdateSantriStatusRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data status tidak valid")
		return h.inertiaService.Redirect(c, returnURL)
	}
	sess, _ := h.store.Get(c)
	if err := ubah(santriID, sessionUser(sess).ID, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, returnURL)
	}
	h.store.Flash(c, "success", "Status santri berhasil diubah")
	return h.inertiaService.Redirect(c, returnURL)
}
