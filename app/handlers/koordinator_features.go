package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// KoordinatorFeaturesHandler serves the koordinator domain pages: pembinaan,
// rapat, kunjungan kelas, Kalam Bersanad, kompetensi, WA templates.
type KoordinatorFeaturesHandler struct {
	svc            *services.KoordinatorService
	guruService    *services.GuruService
	riayahService  *services.RiayahService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewKoordinatorFeaturesHandler(svc *services.KoordinatorService, guruService *services.GuruService, riayahService *services.RiayahService, store *session.Store, inertiaService *services.InertiaService) *KoordinatorFeaturesHandler {
	return &KoordinatorFeaturesHandler{svc: svc, guruService: guruService, riayahService: riayahService, store: store, inertiaService: inertiaService}
}

func (h *KoordinatorFeaturesHandler) idParam(c *fiber.Ctx, name string) int64 {
	id, _ := strconv.ParseInt(c.Params(name), 10, 64)
	return id
}

// ============ Pembinaan ============

func (h *KoordinatorFeaturesHandler) PembinaanList(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.svc.ListPembinaan()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat pembinaan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	return h.inertiaService.Render(c, "koordinator/Pembinaan", fiber.Map{"user": sessionUser(sess), "pembinaan": list})
}

func (h *KoordinatorFeaturesHandler) PembinaanCreate(c *fiber.Ctx) error {
	var req models.PembinaanRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
	}
	if _, err := h.svc.CreatePembinaan(req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
	}
	h.store.Flash(c, "success", "Sesi pembinaan ditambahkan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
}

func (h *KoordinatorFeaturesHandler) PembinaanUpdate(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.PembinaanRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
	}
	if err := h.svc.UpdatePembinaan(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
	}
	h.store.Flash(c, "success", "Sesi pembinaan diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
}

func (h *KoordinatorFeaturesHandler) PembinaanDelete(c *fiber.Ctx) error {
	if err := h.svc.DeletePembinaan(h.idParam(c, "id")); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
	}
	h.store.Flash(c, "success", "Sesi pembinaan dihapus")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
}

func (h *KoordinatorFeaturesHandler) PembinaanAbsenPage(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id := h.idParam(c, "id")
	pembinaan, err := h.svc.GetPembinaan(id)
	if err != nil {
		h.store.Flash(c, "error", "Sesi tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan")
	}
	absen, _ := h.svc.GetPembinaanAbsen(id)
	laporan, _ := h.svc.GetLaporanPembinaanAbsensi(id)
	return h.inertiaService.Render(c, "koordinator/PembinaanAbsen", fiber.Map{"user": sessionUser(sess), "pembinaan": pembinaan, "absen": absen, "laporan": laporan})
}

func (h *KoordinatorFeaturesHandler) PembinaanAbsenSave(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req struct {
		Absen []models.AbsenInput `json:"absen"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan/"+strconv.FormatInt(id, 10)+"/absen")
	}
	if err := h.svc.SavePembinaanAbsen(id, req.Absen); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan/"+strconv.FormatInt(id, 10)+"/absen")
	}
	h.store.Flash(c, "success", "Absensi pembinaan tersimpan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/pembinaan/"+strconv.FormatInt(id, 10)+"/absen")
}

// ============ Rapat ============

func (h *KoordinatorFeaturesHandler) RapatList(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.svc.ListRapat()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat rapat")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	return h.inertiaService.Render(c, "koordinator/Rapat", fiber.Map{"user": sessionUser(sess), "rapat": list})
}

func (h *KoordinatorFeaturesHandler) RapatCreate(c *fiber.Ctx) error {
	var req models.RapatRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
	}
	if _, err := h.svc.CreateRapat(req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
	}
	h.store.Flash(c, "success", "Rapat ditambahkan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
}

func (h *KoordinatorFeaturesHandler) RapatUpdate(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.RapatRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
	}
	if err := h.svc.UpdateRapat(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
	}
	h.store.Flash(c, "success", "Rapat diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
}

func (h *KoordinatorFeaturesHandler) RapatDelete(c *fiber.Ctx) error {
	if err := h.svc.DeleteRapat(h.idParam(c, "id")); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
	}
	h.store.Flash(c, "success", "Rapat dihapus")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
}

func (h *KoordinatorFeaturesHandler) RapatAbsenPage(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id := h.idParam(c, "id")
	rapat, err := h.svc.GetRapat(id)
	if err != nil {
		h.store.Flash(c, "error", "Rapat tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat")
	}
	absen, _ := h.svc.GetRapatAbsen(id)
	laporan, _ := h.svc.GetLaporanRapatAbsensi(id)
	return h.inertiaService.Render(c, "koordinator/RapatAbsen", fiber.Map{"user": sessionUser(sess), "rapat": rapat, "absen": absen, "laporan": laporan})
}

func (h *KoordinatorFeaturesHandler) RapatAbsenSave(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req struct {
		Absen []models.AbsenInput `json:"absen"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat/"+strconv.FormatInt(id, 10)+"/absen")
	}
	if err := h.svc.SaveRapatAbsen(id, req.Absen); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat/"+strconv.FormatInt(id, 10)+"/absen")
	}
	h.store.Flash(c, "success", "Absensi rapat tersimpan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/rapat/"+strconv.FormatInt(id, 10)+"/absen")
}

func (h *KoordinatorFeaturesHandler) RiwayatAbsensi(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	riwayat, err := h.svc.ListRiwayatAbsensiGuru()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat riwayat absensi")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	guru, _ := h.svc.ListGuruSimple()
	return h.inertiaService.Render(c, "koordinator/RiwayatAbsensi", fiber.Map{"user": sessionUser(sess), "riwayat": riwayat, "guruList": guru})
}

// ============ Kunjungan Kelas ============

func (h *KoordinatorFeaturesHandler) KunjunganList(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.svc.ListKunjungan()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat kunjungan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	guru, _ := h.svc.ListGuruSimple()
	kelas, _ := h.svc.ListKelasSimple()
	return h.inertiaService.Render(c, "koordinator/Kunjungan", fiber.Map{"user": sessionUser(sess), "kunjungan": list, "guruList": guru, "kelasList": kelas})
}

func (h *KoordinatorFeaturesHandler) KunjunganCreate(c *fiber.Ctx) error {
	var req models.KunjunganRequest
	if err := c.BodyParser(&req); err != nil || req.GuruID < 1 {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
	}
	if _, err := h.svc.CreateKunjungan(req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
	}
	h.store.Flash(c, "success", "Kunjungan dijadwalkan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
}

func (h *KoordinatorFeaturesHandler) KunjunganUpdate(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.KunjunganRequest
	if err := c.BodyParser(&req); err != nil || req.GuruID < 1 {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
	}
	if err := h.svc.UpdateKunjungan(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
	}
	h.store.Flash(c, "success", "Kunjungan diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
}

func (h *KoordinatorFeaturesHandler) KunjunganDelete(c *fiber.Ctx) error {
	if err := h.svc.DeleteKunjungan(h.idParam(c, "id")); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
	}
	h.store.Flash(c, "success", "Kunjungan dihapus")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kunjungan")
}

// ============ Kalam Bersanad ============

func (h *KoordinatorFeaturesHandler) KalamList(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.svc.ListKalam()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat Kalam Bersanad")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	return h.inertiaService.Render(c, "koordinator/Kalam", fiber.Map{"user": sessionUser(sess), "kalam": list})
}

func (h *KoordinatorFeaturesHandler) KalamCreate(c *fiber.Ctx) error {
	var req models.KalamRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	if _, err := h.svc.CreateKalam(req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	h.store.Flash(c, "success", "Kajian ditambahkan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
}

func (h *KoordinatorFeaturesHandler) KalamUpdate(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.KalamRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	if err := h.svc.UpdateKalam(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	h.store.Flash(c, "success", "Kajian diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
}

func (h *KoordinatorFeaturesHandler) KalamDelete(c *fiber.Ctx) error {
	if err := h.svc.DeleteKalam(h.idParam(c, "id")); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	h.store.Flash(c, "success", "Kajian dihapus")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
}

func (h *KoordinatorFeaturesHandler) KalamSharePage(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id := h.idParam(c, "id")
	kalam, err := h.svc.ListKalam()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	var current *models.KalamResponse
	for i := range kalam {
		if kalam[i].ID == id {
			current = &kalam[i]
			break
		}
	}
	if current == nil {
		h.store.Flash(c, "error", "Kajian tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam")
	}
	share, _ := h.svc.GetKalamShare(id)
	return h.inertiaService.Render(c, "koordinator/KalamShare", fiber.Map{"user": sessionUser(sess), "kalam": current, "share": share})
}

func (h *KoordinatorFeaturesHandler) KalamShareToggle(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req struct {
		GuruID int64 `json:"guru_id"`
		On     bool  `json:"on"`
	}
	if err := c.BodyParser(&req); err != nil || req.GuruID < 1 {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam/"+strconv.FormatInt(id, 10)+"/share")
	}
	if err := h.svc.ToggleKalamShare(id, req.GuruID, req.On); err != nil {
		h.store.Flash(c, "error", "Gagal memperbarui")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam/"+strconv.FormatInt(id, 10)+"/share")
	}
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/kalam/"+strconv.FormatInt(id, 10)+"/share")
}

// ============ WA Template ============

func (h *KoordinatorFeaturesHandler) WaTemplateList(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.svc.ListWaTemplate()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat template")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	return h.inertiaService.Render(c, "koordinator/WaTemplate", fiber.Map{"user": sessionUser(sess), "templates": list})
}

func (h *KoordinatorFeaturesHandler) WaTemplateCreate(c *fiber.Ctx) error {
	var req models.WaTemplateRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Nama) == "" {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
	}
	if _, err := h.svc.CreateWaTemplate(req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
	}
	h.store.Flash(c, "success", "Template ditambahkan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
}

func (h *KoordinatorFeaturesHandler) WaTemplateUpdate(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.WaTemplateRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Nama) == "" {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
	}
	if err := h.svc.UpdateWaTemplate(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
	}
	h.store.Flash(c, "success", "Template diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
}

func (h *KoordinatorFeaturesHandler) WaTemplateDelete(c *fiber.Ctx) error {
	if err := h.svc.DeleteWaTemplate(h.idParam(c, "id")); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
	}
	h.store.Flash(c, "success", "Template dihapus")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/wa-template")
}

// ============ Todo Koordinator ============

func (h *KoordinatorFeaturesHandler) TodoList(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.svc.ListTodo()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat todo")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	return h.inertiaService.Render(c, "koordinator/Todo", fiber.Map{"user": sessionUser(sess), "todos": list})
}

func (h *KoordinatorFeaturesHandler) TodoCreate(c *fiber.Ctx) error {
	var req models.TodoRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Judul) == "" {
		h.store.Flash(c, "error", "Judul todo wajib diisi")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	if _, err := h.svc.CreateTodo(req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	h.store.Flash(c, "success", "Todo ditambahkan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
}

func (h *KoordinatorFeaturesHandler) TodoUpdate(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.TodoRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Judul) == "" {
		h.store.Flash(c, "error", "Judul todo wajib diisi")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	if err := h.svc.UpdateTodo(id, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	h.store.Flash(c, "success", "Todo diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
}

func (h *KoordinatorFeaturesHandler) TodoStatus(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	if err := h.svc.SetTodoStatus(id, req.Status); err != nil {
		h.store.Flash(c, "error", "Gagal memperbarui status")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
}

func (h *KoordinatorFeaturesHandler) TodoDelete(c *fiber.Ctx) error {
	if err := h.svc.DeleteTodo(h.idParam(c, "id")); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
	}
	h.store.Flash(c, "success", "Todo dihapus")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/todo")
}

// ============ Kompetensi & Riayah (on guru detail) ============

func (h *KoordinatorFeaturesHandler) KompetensiSave(c *fiber.Ctx) error {
	id := h.idParam(c, "id")
	var req models.KompetensiRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
	}
	if err := h.svc.SaveKompetensi(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal menyimpan kompetensi")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
	}
	h.store.Flash(c, "success", "Matriks kompetensi tersimpan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
}

// RiayahGuruCreate lets a koordinator record a personal note about a guru.
func (h *KoordinatorFeaturesHandler) RiayahGuruCreate(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	targetGuruID := h.idParam(c, "id")
	if targetGuruID < 1 {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	if _, err := h.guruService.GetDirectoryByID(targetGuruID); err != nil {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}

	var req struct {
		Catatan string `json:"catatan"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Catatan) == "" {
		h.store.Flash(c, "error", "Catatan tidak boleh kosong")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(targetGuruID, 10))
	}
	input := models.RiayahInput{TargetType: "guru", TargetID: targetGuruID, Catatan: req.Catatan}
	author, err := h.guruService.GetGuruByUserID(userID)
	if err == nil {
		_, err = h.riayahService.CreateRiayah(author.ID, userID, input)
	} else {
		_, err = h.riayahService.CreateUserRiayah(userID, input)
	}
	if err != nil {
		h.store.Flash(c, "error", "Gagal menyimpan catatan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(targetGuruID, 10))
	}
	h.store.Flash(c, "success", "Catatan riayah tersimpan")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(targetGuruID, 10))
}
