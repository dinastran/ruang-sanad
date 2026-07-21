package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type SantriHandler struct {
	santriService  *services.SantriService
	masterService  *services.MasterService
	kelasService   *services.KelasService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewSantriHandler(santriService *services.SantriService, masterService *services.MasterService, kelasService *services.KelasService, store *session.Store, inertiaService *services.InertiaService) *SantriHandler {
	return &SantriHandler{
		santriService:  santriService,
		masterService:  masterService,
		kelasService:   kelasService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *SantriHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit := int64(25)
	offset := (page - 1) * limit

	params := models.SantriListParams{
		Angkatan: c.Query("angkatan", ""),
		Level:    c.Query("level", ""),
		Tipe:     c.Query("tipe", ""),
		Jadwal:   c.Query("jadwal", ""),
		Gender:   c.Query("gender", ""),
		Status:   c.Query("status", "aktif"),
		Search:   c.Query("search", ""),
		Lengkap:  -1,
		Offset:   offset,
		Limit:    limit,
	}

	result, err := h.santriService.List(params)
	if err != nil {
		return h.inertiaService.Render(c, "app/SantriList", fiber.Map{
			"user":  user,
			"error": "Gagal memuat data santri",
		})
	}

	angkatanList, _ := h.masterService.ListAngkatan()
	levelList, _ := h.masterService.ListLevel()
	jadwalList, _ := h.masterService.ListJadwal()

	return h.inertiaService.Render(c, "app/SantriList", fiber.Map{
		"user":     user,
		"santri":   result.Data,
		"total":    result.Total,
		"page":     page,
		"limit":    limit,
		"angkatan": angkatanList,
		"levels":   levelList,
		"jadwals":  jadwalList,
	})
}

func (h *SantriHandler) New(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	angkatanList, _ := h.masterService.ListAngkatan()
	kodeKelasList, _ := h.masterService.ListKodeKelas()

	return h.inertiaService.Render(c, "app/SantriForm", fiber.Map{
		"user":       user,
		"santri":     nil,
		"angkatan":   angkatanList,
		"kode_kelas": kodeKelasList,
	})
}

func (h *SantriHandler) Store(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var req models.CreateSantriRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/santri/new")
	}

	if req.Nama == "" || req.JenisKelamin == "" {
		h.store.Flash(c, "error", "Nama dan jenis kelamin wajib diisi")
		return h.inertiaService.Redirect(c, "/app/santri/new")
	}

	_, err := h.santriService.Create(req, userID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal menyimpan santri: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/santri/new")
	}

	h.store.Flash(c, "success", "Santri berhasil ditambahkan")
	return h.inertiaService.Redirect(c, "/app/santri")
}

func (h *SantriHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/santri")
	}

	santri, err := h.santriService.GetByID(id)
	if err != nil {
		h.store.Flash(c, "error", "Santri tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/santri")
	}

	angkatanList, _ := h.masterService.ListAngkatan()
	levelList, _ := h.masterService.ListLevel()
	jadwalList, _ := h.masterService.ListJadwal()
	guruList, _ := h.masterService.ListGuru()
	kodeKelasList, _ := h.masterService.ListKodeKelas()

	return h.inertiaService.Render(c, "app/SantriForm", fiber.Map{
		"user":       user,
		"santri":     santri,
		"angkatan":   angkatanList,
		"levels":     levelList,
		"jadwals":    jadwalList,
		"gurus":      guruList,
		"kode_kelas": kodeKelasList,
	})
}

func (h *SantriHandler) UpdateCS(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/santri")
	}

	var req models.UpdateSantriCSRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/santri/"+c.Params("id"))
	}

	if err := h.santriService.UpdateByCS(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal update: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/santri/"+c.Params("id"))
	}

	h.store.Flash(c, "success", "Data santri berhasil diupdate")
	return h.inertiaService.Redirect(c, "/app/santri")
}

func (h *SantriHandler) UpdateAdminKelas(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/perlu-dilengkapi")
	}

	var req models.UpdateSantriAdminKelasRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/perlu-dilengkapi")
	}

	if err := h.santriService.UpdateByAdminKelas(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal update: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/perlu-dilengkapi")
	}

	h.store.Flash(c, "success", "Data kelas berhasil diupdate")
	return h.inertiaService.Redirect(c, "/app/perlu-dilengkapi")
}

// Keuangan renders the finance admin's monthly input page: a list of santri
// where infaq_terakhir & keterangan_tidak_lanjut are recorded per santri.
func (h *SantriHandler) Keuangan(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	if page < 1 {
		page = 1
	}
	limit := int64(25)
	offset := (page - 1) * limit

	params := models.SantriListParams{
		Angkatan: c.Query("angkatan", ""),
		Status:   c.Query("status", ""),
		Gender:   c.Query("gender", ""),
		Search:   c.Query("search", ""),
		Lengkap:  -1,
		Offset:   offset,
		Limit:    limit,
	}

	result, err := h.santriService.List(params)
	if err != nil {
		return h.inertiaService.Render(c, "app/Keuangan", fiber.Map{
			"user":  user,
			"error": "Gagal memuat data santri",
		})
	}

	angkatanList, _ := h.masterService.ListAngkatan()

	return h.inertiaService.Render(c, "app/Keuangan", fiber.Map{
		"user":     user,
		"santri":   result.Data,
		"total":    result.Total,
		"page":     page,
		"limit":    limit,
		"angkatan": angkatanList,
	})
}

func (h *SantriHandler) UpdateKeuangan(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/keuangan")
	}

	var req models.UpdateSantriKeuanganRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/keuangan")
	}

	if err := h.santriService.UpdateByKeuangan(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal update: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/keuangan")
	}

	h.store.Flash(c, "success", "Data keuangan berhasil diupdate")
	return h.inertiaService.Redirect(c, "/app/keuangan")
}

func (h *SantriHandler) PerluDilengkapi(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	list, err := h.santriService.ListPerluDilengkapi()
	if err != nil {
		return h.inertiaService.Render(c, "app/PerluDilengkapi", fiber.Map{
			"user":  user,
			"error": "Gagal memuat data",
		})
	}

	levelList, _ := h.masterService.ListLevel()
	jadwalList, _ := h.masterService.ListJadwal()
	guruList, _ := h.masterService.ListGuru()

	return h.inertiaService.Render(c, "app/PerluDilengkapi", fiber.Map{
		"user":    user,
		"santri":  list,
		"levels":  levelList,
		"jadwals": jadwalList,
		"gurus":   guruList,
	})
}

func (h *SantriHandler) PindahKelas(c *fiber.Ctx) error {
	santriID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	var req struct {
		KelasTujuanID int64 `json:"kelas_tujuan_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	if err := h.santriService.PindahkanKelas(santriID, req.KelasTujuanID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	h.store.Flash(c, "success", "Santri berhasil dipindahkan")
	return h.inertiaService.Redirect(c, "/app/santri/"+c.Params("id"))
}
