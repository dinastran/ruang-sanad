package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// KunjunganHandler serves Kunjungan Kelas: the Koordinator Guru side (jadwal,
// penilaian, kirim ke guru, tindak lanjut, rekap) and the guru's own results.
type KunjunganHandler struct {
	svc            *services.KunjunganService
	koordinator    *services.KoordinatorService
	guruService    *services.GuruService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewKunjunganHandler(svc *services.KunjunganService, koordinator *services.KoordinatorService, guruService *services.GuruService, store *session.Store, inertiaService *services.InertiaService) *KunjunganHandler {
	return &KunjunganHandler{svc: svc, koordinator: koordinator, guruService: guruService, store: store, inertiaService: inertiaService}
}

const kunjunganListURL = "/app/koordinator-guru/kunjungan"

func kunjunganDetailURL(id int64) string {
	return fmt.Sprintf("%s/%d", kunjunganListURL, id)
}

func paramID(c *fiber.Ctx, name string) int64 {
	id, _ := strconv.ParseInt(c.Params(name), 10, 64)
	return id
}

// ============ Koordinator ============

func (h *KunjunganHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	dashboard, err := h.svc.Dashboard(time.Now())
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat kunjungan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	guru, _ := h.koordinator.ListGuruSimple()
	kelas, _ := h.koordinator.ListKelasSimple()
	return h.inertiaService.Render(c, "koordinator/Kunjungan", fiber.Map{
		"user":      sessionUser(sess),
		"dashboard": dashboard,
		"guruList":  guru,
		"kelasList": kelas,
		"tab":       c.Query("tab"),
	})
}

func (h *KunjunganHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	k, err := h.svc.Detail(paramID(c, "id"), time.Now())
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganListURL)
	}
	guru, _ := h.koordinator.ListGuruSimple()
	kelas, _ := h.koordinator.ListKelasSimple()
	return h.inertiaService.Render(c, "koordinator/KunjunganDetail", fiber.Map{
		"user":      sessionUser(sess),
		"kunjungan": k,
		"guruList":  guru,
		"kelasList": kelas,
	})
}

func (h *KunjunganHandler) Create(c *fiber.Ctx) error {
	var req models.KunjunganRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, kunjunganListURL)
	}
	id, err := h.svc.Create(req)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganListURL)
	}
	if req.Status == "terlaksana" {
		h.store.Flash(c, "success", "Kunjungan tersimpan. Lengkapi penilaian lalu kirim ke guru.")
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	h.store.Flash(c, "success", "Kunjungan dijadwalkan")
	return h.inertiaService.Redirect(c, kunjunganListURL)
}

func (h *KunjunganHandler) Update(c *fiber.Ctx) error {
	id := paramID(c, "id")
	var req models.KunjunganRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	if err := h.svc.Update(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	h.store.Flash(c, "success", "Kunjungan diperbarui")
	return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
}

func (h *KunjunganHandler) Delete(c *fiber.Ctx) error {
	id := paramID(c, "id")
	if err := h.svc.Delete(id); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, kunjunganListURL)
	}
	h.store.Flash(c, "success", "Kunjungan dihapus")
	return h.inertiaService.Redirect(c, kunjunganListURL)
}

func (h *KunjunganHandler) Kirim(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id := paramID(c, "id")
	if err := h.svc.Kirim(id, sessionUser(sess).ID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	h.store.Flash(c, "success", "Hasil kunjungan terkirim ke guru")
	return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
}

func (h *KunjunganHandler) BukaKunci(c *fiber.Ctx) error {
	id := paramID(c, "id")
	if err := h.svc.BukaKunci(id); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	h.store.Flash(c, "success", "Kunci dibuka. Kirim ulang setelah selesai mengubah.")
	return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
}

func (h *KunjunganHandler) TambahTindakLanjut(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id := paramID(c, "id")
	var req models.TindakLanjutRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tindak lanjut tidak valid")
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	if err := h.svc.TambahTindakLanjut(id, req, sessionUser(sess).ID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	msg := "Tindak lanjut ditambahkan"
	if req.Jenis == services.TindakLanjutMonitoring {
		msg = "Tindak lanjut ditambahkan dan kunjungan monitoring berikutnya sudah dijadwalkan"
	}
	h.store.Flash(c, "success", msg)
	return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
}

func (h *KunjunganHandler) StatusTindakLanjut(c *fiber.Ctx) error {
	id := paramID(c, "id")
	var req models.TindakLanjutStatusRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Back(c, kunjunganDetailURL(id))
	}
	if err := h.svc.SetStatusTindakLanjut(id, paramID(c, "tid"), req.Status); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, kunjunganDetailURL(id))
	}
	h.store.Flash(c, "success", "Status tindak lanjut diperbarui")
	return h.inertiaService.Back(c, kunjunganDetailURL(id))
}

func (h *KunjunganHandler) HapusTindakLanjut(c *fiber.Ctx) error {
	id := paramID(c, "id")
	if err := h.svc.HapusTindakLanjut(id, paramID(c, "tid")); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
	}
	h.store.Flash(c, "success", "Tindak lanjut dihapus")
	return h.inertiaService.Redirect(c, kunjunganDetailURL(id))
}

// ============ Guru ============

const guruKunjunganURL = "/app/guru/kunjungan"

func (h *KunjunganHandler) GuruIndex(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	guru, err := h.guruService.GetGuruByUserID(toInt64(sess.Get("user_id")))
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan untuk akun ini")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	list, err := h.svc.ListUntukGuru(guru.ID, time.Now())
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat hasil kunjungan")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	return h.inertiaService.Render(c, "guru/Kunjungan", fiber.Map{"user": sessionUser(sess), "kunjungan": list})
}

func (h *KunjunganHandler) GuruShow(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	guru, err := h.guruService.GetGuruByUserID(toInt64(sess.Get("user_id")))
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan untuk akun ini")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	k, err := h.svc.DetailUntukGuru(paramID(c, "id"), guru.ID, time.Now())
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, guruKunjunganURL)
	}
	return h.inertiaService.Render(c, "guru/KunjunganDetail", fiber.Map{"user": sessionUser(sess), "kunjungan": k})
}

func (h *KunjunganHandler) GuruTanggapan(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id := paramID(c, "id")
	back := fmt.Sprintf("%s/%d", guruKunjunganURL, id)
	guru, err := h.guruService.GetGuruByUserID(toInt64(sess.Get("user_id")))
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan untuk akun ini")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	var req models.TanggapanKunjunganRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tanggapan tidak valid")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.svc.SimpanTanggapan(id, guru.ID, req.Tanggapan); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, back)
	}
	h.store.Flash(c, "success", "Tanggapan tersimpan")
	return h.inertiaService.Redirect(c, back)
}
