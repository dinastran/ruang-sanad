package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type TagihanHandler struct {
	tagihan        *services.TagihanService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewTagihanHandler(tagihan *services.TagihanService, store *session.Store, inertiaService *services.InertiaService) *TagihanHandler {
	return &TagihanHandler{tagihan: tagihan, store: store, inertiaService: inertiaService}
}

func (h *TagihanHandler) Dashboard(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	now := time.Now()
	dari := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	sampai := dari.AddDate(0, 1, 0).Add(-time.Nanosecond)
	ringkasan, err := h.tagihan.RingkasanPeriode(dari, sampai)
	if err != nil {
		return err
	}
	belum, err := h.tagihan.List(models.TagihanFilter{Status: "belum_bayar", TanggalDari: dari.Format("2006-01-02"), TanggalSampai: sampai.Format("2006-01-02")})
	if err != nil {
		return err
	}
	lunas, err := h.tagihan.List(models.TagihanFilter{Status: "lunas", TanggalDari: dari.Format("2006-01-02"), TanggalSampai: sampai.Format("2006-01-02")})
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "keuangan/Dashboard", fiber.Map{"user": sessionUser(sess), "ringkasan": ringkasan, "belumBayar": belum, "lunas": lunas, "periode": dari.Format("January 2006")})
}

func (h *TagihanHandler) List(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	tagihan, err := h.tagihan.List(tagihanFilter(c))
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "keuangan/Tagihan", fiber.Map{"user": sessionUser(sess), "tagihan": tagihan, "filter": fiber.Map{"status": c.Query("status"), "search": c.Query("search"), "tanggal_dari": c.Query("tanggal_dari"), "tanggal_sampai": c.Query("tanggal_sampai"), "kelas_id": c.Query("kelas_id"), "angkatan_kelas": c.Query("angkatan_kelas"), "guru_id": c.Query("guru_id"), "frekuensi": c.Query("frekuensi"), "level": c.Query("level"), "gender": c.Query("gender"), "bulan_ke": c.Query("bulan_ke")}})
}

func (h *TagihanHandler) Detail(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	tagihan, err := h.tagihan.Get(id)
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "keuangan/DetailTagihan", fiber.Map{"user": sessionUser(sess), "tagihan": tagihan})
}

func (h *TagihanHandler) MarkLunas(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.MarkTagihanLunasRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "data pelunasan tidak valid")
	}
	sess, _ := h.store.Get(c)
	if err := h.tagihan.MarkLunas(id, toInt64(sess.Get("user_id")), req); err != nil {
		return err
	}
	h.store.Flash(c, "success", "Tagihan ditandai lunas")
	return h.inertiaService.Back(c, "/app/keuangan/tagihan")
}

func (h *TagihanHandler) Batal(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.BatalkanTagihanRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "data pembatalan tidak valid")
	}
	if err := h.tagihan.Batal(id, req.Catatan); err != nil {
		return err
	}
	h.store.Flash(c, "success", "Tagihan dibatalkan")
	return h.inertiaService.Back(c, "/app/keuangan/tagihan")
}

func (h *TagihanHandler) FollowUp(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	link, err := h.tagihan.FollowUpURL(id)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"url": link})
}

func (h *TagihanHandler) Sync(c *fiber.Ctx) error {
	if err := h.tagihan.Sync(); err != nil {
		return err
	}
	h.store.Flash(c, "success", "Tagihan dari pertemuan selesai telah disinkronkan")
	return h.inertiaService.Redirect(c, "/app/keuangan/tagihan")
}

func tagihanFilter(c *fiber.Ctx) models.TagihanFilter {
	parse := func(key string) int64 { n, _ := strconv.ParseInt(c.Query(key), 10, 64); return n }
	return models.TagihanFilter{Status: c.Query("status"), KelasID: parse("kelas_id"), AngkatanKelas: c.Query("angkatan_kelas"), GuruID: parse("guru_id"), Frekuensi: c.Query("frekuensi"), Level: c.Query("level"), Gender: c.Query("gender"), BulanKe: parse("bulan_ke"), TanggalDari: c.Query("tanggal_dari"), TanggalSampai: c.Query("tanggal_sampai"), Search: c.Query("search")}
}
