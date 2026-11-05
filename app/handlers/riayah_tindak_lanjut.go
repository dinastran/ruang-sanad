package handlers

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// RiayahTindakLanjutHandler melayani antrian Admin Kelas untuk santri yang
// alpa berturut-turut.
type RiayahTindakLanjutHandler struct {
	riayahSantriService *services.RiayahSantriService
	store               *session.Store
	inertiaService      *services.InertiaService
}

func NewRiayahTindakLanjutHandler(riayahSantriService *services.RiayahSantriService, store *session.Store, inertiaService *services.InertiaService) *RiayahTindakLanjutHandler {
	return &RiayahTindakLanjutHandler{riayahSantriService: riayahSantriService, store: store, inertiaService: inertiaService}
}

func (h *RiayahTindakLanjutHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	perlu, sudah, err := h.riayahSantriService.ListTindakLanjutAlpa(services.HariIniRiayah())
	if err != nil {
		slog.Error("tindak lanjut alpa list failed", "error", err)
		h.store.Flash(c, "error", "Gagal memuat tindak lanjut alpa")
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	return h.inertiaService.Render(c, "app/TindakLanjutAlpa", fiber.Map{
		"user":         sessionUser(sess),
		"perlu":        perlu,
		"sudah":        sudah,
		"wa_templates": h.riayahSantriService.ListTemplateWATindakLanjut(),
	})
}

// Jumlah dipakai badge menu dan kartu halaman Kelas.
func (h *RiayahTindakLanjutHandler) Jumlah(c *fiber.Ctx) error {
	n, err := h.riayahSantriService.JumlahTindakLanjutAlpa(services.HariIniRiayah())
	if err != nil {
		slog.Error("tindak lanjut alpa count failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghitung tindak lanjut alpa"})
	}
	return c.JSON(fiber.Map{"perlu": n})
}

// Selesaikan hanya tersedia bagi Admin Kelas dan Super Admin (middleware akRole).
func (h *RiayahTindakLanjutHandler) Selesaikan(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	santriID, err := strconv.ParseInt(c.Params("sid"), 10, 64)
	if err != nil || santriID <= 0 || userID <= 0 {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	var req struct {
		Alasan string `json:"alasan" form:"alasan"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	if err := h.riayahSantriService.SelesaikanTindakLanjutAlpa(userID, santriID, strings.TrimSpace(req.Alasan), services.HariIniRiayah()); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/kelas/tindak-lanjut-alpa")
	}
	h.store.Flash(c, "success", "Tindak lanjut alpa diselesaikan dengan alasan")
	return h.inertiaService.Redirect(c, "/app/kelas/tindak-lanjut-alpa")
}
