package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type TSIHandler struct {
	tsi            *services.TSIService
	guruService    *services.GuruService
	koordinator    *services.KoordinatorService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewTSIHandler(tsi *services.TSIService, guruService *services.GuruService, koordinator *services.KoordinatorService, store *session.Store, inertiaService *services.InertiaService) *TSIHandler {
	return &TSIHandler{tsi: tsi, guruService: guruService, koordinator: koordinator, store: store, inertiaService: inertiaService}
}

func bulanParam(c *fiber.Ctx) string {
	b := c.Query("bulan")
	if b == "" {
		return time.Now().Format("2006-01")
	}
	return b
}

// Rekap — matrix guru × kategori for a month (koordinator).
func (h *TSIHandler) Rekap(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := bulanParam(c)
	rekap, err := h.tsi.GetRekap(bulan)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	guru, _ := h.koordinator.ListGuruSimple()
	return h.inertiaService.Render(c, "koordinator/TSIRekap", fiber.Map{
		"user":     sessionUser(sess),
		"bulan":    bulan,
		"rekap":    rekap,
		"guruList": guru,
	})
}

// Penilaian — per-guru evaluation form (koordinator).
func (h *TSIHandler) Penilaian(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	guruID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	bulan := bulanParam(c)
	guru, err := h.guruService.GetDirectoryByID(guruID)
	if err != nil {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi")
	}
	penilaian, err := h.tsi.GetPenilaian(guruID, guru.Nama, bulan)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi")
	}
	return h.inertiaService.Render(c, "koordinator/TSIPenilaian", fiber.Map{
		"user":      sessionUser(sess),
		"penilaian": penilaian,
	})
}

func (h *TSIHandler) SaveNilai(c *fiber.Ctx) error {
	guruID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	bulan := bulanParam(c)
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))

	guru, err := h.guruService.GetDirectoryByID(guruID)
	if err != nil {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi")
	}
	penilaian, err := h.tsi.GetPenilaian(guruID, guru.Nama, bulan)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi")
	}
	var req models.TsiSaveRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.redirectPenilaian(c, guruID, bulan)
	}
	if err := h.tsi.SaveNilai(penilaian.PeriodeID, userID, req.Nilai); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.redirectPenilaian(c, guruID, bulan)
	}
	h.store.Flash(c, "success", "Nilai TSI tersimpan")
	return h.redirectPenilaian(c, guruID, bulan)
}

func (h *TSIHandler) Finalize(c *fiber.Ctx) error {
	return h.setStatus(c, "final", "Periode difinalisasi")
}
func (h *TSIHandler) Reopen(c *fiber.Ctx) error {
	return h.setStatus(c, "draft", "Periode dibuka kembali")
}

func (h *TSIHandler) setStatus(c *fiber.Ctx, status, msg string) error {
	guruID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	bulan := bulanParam(c)
	guru, err := h.guruService.GetDirectoryByID(guruID)
	if err != nil {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi")
	}
	penilaian, err := h.tsi.GetPenilaian(guruID, guru.Nama, bulan)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi")
	}
	if err := h.tsi.SetStatus(penilaian.PeriodeID, status); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.redirectPenilaian(c, guruID, bulan)
	}
	h.store.Flash(c, "success", msg)
	return h.redirectPenilaian(c, guruID, bulan)
}

func (h *TSIHandler) redirectPenilaian(c *fiber.Ctx, guruID int64, bulan string) error {
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/tsi/"+strconv.FormatInt(guruID, 10)+"?bulan="+bulan)
}

// Saya — a guru's own read-only TSI.
func (h *TSIHandler) Saya(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	bulan := bulanParam(c)

	guru, err := h.guruService.GetGuruByUserID(userID)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan untuk akun ini")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	penilaian, err := h.tsi.GetPenilaian(guru.ID, guru.Nama, bulan)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	return h.inertiaService.Render(c, "guru/TSISaya", fiber.Map{
		"user":      user,
		"penilaian": penilaian,
	})
}
