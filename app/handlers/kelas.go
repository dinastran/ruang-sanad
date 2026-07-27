package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type KelasHandler struct {
	kelasService   *services.KelasService
	masterService  *services.MasterService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewKelasHandler(kelasService *services.KelasService, masterService *services.MasterService, store *session.Store, inertiaService *services.InertiaService) *KelasHandler {
	return &KelasHandler{
		kelasService:   kelasService,
		masterService:  masterService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *KelasHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	angkatan := c.Query("angkatan", "")
	list, err := h.kelasService.ListByAngkatan(angkatan)
	if err != nil {
		return h.inertiaService.Render(c, "app/KelasList", fiber.Map{
			"user":  user,
			"error": "Gagal memuat kelas",
		})
	}

	h.fillGuruNama(list)

	angkatanList, _ := h.masterService.ListAngkatan()

	return h.inertiaService.Render(c, "app/KelasList", fiber.Map{
		"user":     user,
		"kelas":    list,
		"angkatan": angkatanList,
	})
}

func (h *KelasHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	kelas, err := h.kelasService.GetByID(id)
	if err != nil {
		h.store.Flash(c, "error", "Kelas tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	santriList, _ := h.kelasService.GetSantriByKelasID(id)
	guruList, _ := h.masterService.ListGuruAll()
	kelasList, _ := h.kelasService.ListByAngkatan(kelas.Angkatan)
	kelasLain := make([]models.KelasResponse, 0, len(kelasList))
	for _, item := range kelasList {
		if item.ID != kelas.ID && item.IsAktif {
			kelasLain = append(kelasLain, item)
		}
	}

	if kelas.GuruID != nil {
		for _, g := range guruList {
			if g.ID == *kelas.GuruID {
				kelas.GuruNama = g.Nama
				break
			}
		}
	}

	return h.inertiaService.Render(c, "app/KelasDetail", fiber.Map{
		"user":       user,
		"kelas":      kelas,
		"santri":     santriList,
		"gurus":      guruList,
		"kelas_lain": kelasLain,
	})
}

// fillGuruNama populates GuruNama on each kelas from the guru master list.
func (h *KelasHandler) fillGuruNama(list []models.KelasResponse) {
	guruList, _ := h.masterService.ListGuruAll()
	names := make(map[int64]string, len(guruList))
	for _, g := range guruList {
		names[g.ID] = g.Nama
	}
	for i := range list {
		if list[i].GuruID != nil {
			list[i].GuruNama = names[*list[i].GuruID]
		}
	}
}

func (h *KelasHandler) AssignGuru(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	var req struct {
		GuruID int64 `json:"guru_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/kelas/"+c.Params("id"))
	}

	if err := h.kelasService.AssignGuru(id, req.GuruID); err != nil {
		h.store.Flash(c, "error", "Gagal assign guru")
		return h.inertiaService.Redirect(c, "/app/kelas/"+c.Params("id"))
	}

	h.store.Flash(c, "success", "Guru berhasil diassign")
	return h.inertiaService.Redirect(c, "/app/kelas/"+c.Params("id"))
}

func (h *KelasHandler) SetAktif(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	var req struct {
		IsAktif bool `json:"is_aktif"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	if err := h.kelasService.SetAktif(id, req.IsAktif); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah status kelas")
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	if req.IsAktif {
		h.store.Flash(c, "success", "Kelas diaktifkan")
	} else {
		h.store.Flash(c, "success", "Kelas dinonaktifkan")
	}
	return h.inertiaService.Redirect(c, c.Get("Referer", "/app/kelas"))
}

func (h *KelasHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	if err := h.kelasService.Delete(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus kelas")
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	h.store.Flash(c, "success", "Kelas berhasil dihapus")
	return h.inertiaService.Redirect(c, "/app/kelas")
}
