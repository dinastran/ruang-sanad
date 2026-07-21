package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type MasterHandler struct {
	masterService  *services.MasterService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewMasterHandler(masterService *services.MasterService, store *session.Store, inertiaService *services.InertiaService) *MasterHandler {
	return &MasterHandler{
		masterService:  masterService,
		store:          store,
		inertiaService: inertiaService,
	}
}

// ---- JSON API endpoints (used by dropdowns) ----

func (h *MasterHandler) Angkatan(c *fiber.Ctx) error {
	list, err := h.masterService.ListAngkatan()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

func (h *MasterHandler) Level(c *fiber.Ctx) error {
	list, err := h.masterService.ListLevel()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

func (h *MasterHandler) Jadwal(c *fiber.Ctx) error {
	list, err := h.masterService.ListJadwal()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

func (h *MasterHandler) Guru(c *fiber.Ctx) error {
	list, err := h.masterService.ListGuru()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

func (h *MasterHandler) KodeKelas(c *fiber.Ctx) error {
	list, err := h.masterService.ListKodeKelas()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

// ---- Management page (Inertia) ----

func (h *MasterHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	angkatan, _ := h.masterService.ListAngkatan()
	levels, _ := h.masterService.ListLevel()
	jadwals, _ := h.masterService.ListJadwal()
	gurus, _ := h.masterService.ListGuruAll()
	kodeKelas, _ := h.masterService.ListKodeKelas()

	return h.inertiaService.Render(c, "app/MasterData", fiber.Map{
		"user":       user,
		"angkatan":   angkatan,
		"levels":     levels,
		"jadwals":    jadwals,
		"gurus":      gurus,
		"kode_kelas": kodeKelas,
	})
}

// ---- Create ----

func (h *MasterHandler) CreateAngkatan(c *fiber.Ctx) error {
	var req models.CreateAngkatanRequest
	if err := c.BodyParser(&req); err != nil || req.Kode == "" {
		h.store.Flash(c, "error", "Kode angkatan wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.CreateAngkatan(req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah angkatan: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Angkatan berhasil ditambahkan")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) CreateLevel(c *fiber.Ctx) error {
	var req models.CreateLevelRequest
	if err := c.BodyParser(&req); err != nil || req.Kode == "" {
		h.store.Flash(c, "error", "Kode level wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.CreateLevel(req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah level: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Level berhasil ditambahkan")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) CreateJadwal(c *fiber.Ctx) error {
	var req models.CreateJadwalRequest
	if err := c.BodyParser(&req); err != nil || req.Nama == "" {
		h.store.Flash(c, "error", "Nama jadwal wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.CreateJadwal(req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah jadwal: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Jadwal berhasil ditambahkan")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) CreateGuru(c *fiber.Ctx) error {
	var req models.CreateGuruRequest
	if err := c.BodyParser(&req); err != nil || req.Nama == "" {
		h.store.Flash(c, "error", "Nama guru wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.CreateGuru(req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah guru: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Guru berhasil ditambahkan")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) CreateKodeKelas(c *fiber.Ctx) error {
	var req models.CreateKodeKelasRequest
	if err := c.BodyParser(&req); err != nil || req.Kode == "" {
		h.store.Flash(c, "error", "Kode kelas wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.CreateKodeKelas(req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah kode kelas: "+err.Error())
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Kode kelas berhasil ditambahkan")
	return h.inertiaService.Redirect(c, "/app/master")
}

// ---- Update ----

func (h *MasterHandler) UpdateAngkatan(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.UpdateAngkatanRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.UpdateAngkatan(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah angkatan")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Angkatan diperbarui")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) UpdateLevel(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.UpdateLevelRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.UpdateLevel(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah level")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Level diperbarui")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) UpdateJadwal(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.UpdateJadwalRequest
	if err := c.BodyParser(&req); err != nil || req.Nama == "" {
		h.store.Flash(c, "error", "Nama jadwal wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.UpdateJadwal(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah jadwal")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Jadwal diperbarui")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) UpdateGuru(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.UpdateGuruRequest
	if err := c.BodyParser(&req); err != nil || req.Nama == "" {
		h.store.Flash(c, "error", "Nama guru wajib diisi")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.UpdateGuru(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah guru")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Guru diperbarui")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) UpdateKodeKelas(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.UpdateKodeKelasRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	if err := h.masterService.UpdateKodeKelas(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah kode kelas")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Kode kelas diperbarui")
	return h.inertiaService.Redirect(c, "/app/master")
}

// ---- Delete ----

func (h *MasterHandler) DeleteAngkatan(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.masterService.DeleteAngkatan(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus angkatan")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Angkatan dihapus")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) DeleteLevel(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.masterService.DeleteLevel(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus level")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Level dihapus")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) DeleteJadwal(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.masterService.DeleteJadwal(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus jadwal")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Jadwal dihapus")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) DeleteGuru(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.masterService.DeleteGuru(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus guru")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Guru dihapus")
	return h.inertiaService.Redirect(c, "/app/master")
}

func (h *MasterHandler) DeleteKodeKelas(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.masterService.DeleteKodeKelas(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus kode kelas")
		return h.inertiaService.Redirect(c, "/app/master")
	}
	h.store.Flash(c, "success", "Kode kelas dihapus")
	return h.inertiaService.Redirect(c, "/app/master")
}
