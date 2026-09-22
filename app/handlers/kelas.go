package handlers

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type KelasHandler struct {
	kelasService          *services.KelasService
	kelasPerubahanService *services.KelasPerubahanService
	santriStatusService   *services.SantriStatusService
	masterService         *services.MasterService
	store                 *session.Store
	inertiaService        *services.InertiaService
}

func NewKelasHandler(kelasService *services.KelasService, kelasPerubahanService *services.KelasPerubahanService, santriStatusService *services.SantriStatusService, masterService *services.MasterService, store *session.Store, inertiaService *services.InertiaService) *KelasHandler {
	return &KelasHandler{
		kelasService:          kelasService,
		kelasPerubahanService: kelasPerubahanService,
		santriStatusService:   santriStatusService,
		masterService:         masterService,
		store:                 store,
		inertiaService:        inertiaService,
	}
}

func (h *KelasHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	list, err := h.kelasService.ListByAngkatan("")
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
		"filters": fiber.Map{
			"q":        c.Query("q", ""),
			"guru_id":  c.Query("guru_id", ""),
			"angkatan": c.Query("angkatan", ""),
			"status":   c.Query("status", ""),
			"gender":   c.Query("gender", ""),
			"level":    c.Query("level", ""),
			"jadwal":   c.Query("jadwal", ""),
		},
	})
}

func (h *KelasHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	kelas, santriList, err := h.kelasService.GetDetail(id)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat detail kelas")
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	hasPertemuan, _ := h.kelasService.HasPertemuan(id)
	guruList, _ := h.masterService.ListGuruAll()
	kelasList, _ := h.kelasService.ListAll()
	kelasLain := make([]models.KelasResponse, 0, len(kelasList))
	for _, item := range kelasList {
		if item.ID != kelas.ID && item.IsAktif {
			kelasLain = append(kelasLain, item)
		}
	}

	h.fillGuruNama(kelasLain)

	if kelas.GuruID != nil {
		for _, g := range guruList {
			if g.ID == *kelas.GuruID {
				kelas.GuruNama = g.Nama
				break
			}
		}
	}

	levelList, _ := h.masterService.ListLevel()
	jadwalList, _ := h.masterService.ListJadwal()
	// Never send null: KelasDetail reads these lists' .length directly.
	riwayatPerubahan, err := h.kelasPerubahanService.ListRiwayat(id)
	if err != nil {
		riwayatPerubahan = []models.KelasPerubahanResponse{}
	}
	riwayatStatus, err := h.santriStatusService.ListRiwayatKelas(id)
	if err != nil {
		riwayatStatus = []models.SantriStatusLogResponse{}
	}

	return h.inertiaService.Render(c, "app/KelasDetail", fiber.Map{
		"user":              user,
		"kelas":             kelas,
		"santri":            santriList,
		"gurus":             guruList,
		"kelas_lain":        kelasLain,
		"has_pertemuan":     hasPertemuan,
		"levels":            levelList,
		"jadwals":           jadwalList,
		"riwayat_perubahan": riwayatPerubahan,
		"riwayat_status":    riwayatStatus,
		"return_to":         kelasListReturnURL(c),
	})
}

// kelasListReturnURL only permits returning to the internal class list.
func kelasListReturnURL(c *fiber.Ctx) string {
	returnTo := c.Query("return_to", "")
	parsed, err := url.Parse(returnTo)
	if err != nil || parsed.IsAbs() || parsed.Path != "/app/kelas" || parsed.Fragment != "" {
		return "/app/kelas"
	}
	if parsed.RawQuery == "" {
		return "/app/kelas"
	}
	return "/app/kelas?" + parsed.RawQuery
}

func kelasDetailReturnURL(c *fiber.Ctx, id string) string {
	returnTo := kelasListReturnURL(c)
	if returnTo == "/app/kelas" {
		return "/app/kelas/" + id
	}
	return "/app/kelas/" + id + "?return_to=" + url.QueryEscape(returnTo)
}

func kelasStatusReturnURL(c *fiber.Ctx, id string) string {
	if c.Query("from_detail") == "1" {
		return kelasDetailReturnURL(c, id)
	}
	return kelasListReturnURL(c)
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
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}

	if err := h.kelasService.AssignGuru(id, req.GuruID); err != nil {
		h.store.Flash(c, "error", "Gagal assign guru")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}

	h.store.Flash(c, "success", "Guru berhasil diassign")
	return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
}

// UbahStatusSantri lets Admin Kelas set a class member to aktif, cuti, or
// nonaktif from the class detail page.
func (h *KelasHandler) UbahStatusSantri(c *fiber.Ctx) error {
	kelasID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	returnURL := kelasDetailReturnURL(c, c.Params("id"))
	santriID, err := strconv.ParseInt(c.Params("santriId"), 10, 64)
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
	if err := h.santriStatusService.UbahStatus(kelasID, santriID, sessionUser(sess).ID, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, returnURL)
	}
	h.store.Flash(c, "success", "Status santri berhasil diubah")
	return h.inertiaService.Redirect(c, returnURL)
}

func (h *KelasHandler) SetPertemuanTerakhir(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	var req models.SetPertemuanTerakhirRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data nomor pertemuan tidak valid")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	if err := h.kelasService.SetPertemuanTerakhir(id, req.PertemuanTerakhir); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	h.store.Flash(c, "success", "Nomor pertemuan awal kelas tersimpan")
	return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
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
		return h.inertiaService.Redirect(c, kelasStatusReturnURL(c, c.Params("id")))
	}

	if err := h.kelasService.SetAktif(id, req.IsAktif); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah status kelas")
		return h.inertiaService.Redirect(c, kelasStatusReturnURL(c, c.Params("id")))
	}

	if req.IsAktif {
		h.store.Flash(c, "success", "Kelas diaktifkan")
	} else {
		h.store.Flash(c, "success", "Kelas dinonaktifkan")
	}
	return h.inertiaService.Redirect(c, kelasStatusReturnURL(c, c.Params("id")))
}

func (h *KelasHandler) SetMateriIndividual(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	var req models.SetKelasMateriIndividualRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data pengaturan progres materi tidak valid")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	if err := h.kelasService.SetMateriIndividual(id, req.MateriIndividual); err != nil {
		h.store.Flash(c, "error", "Gagal mengubah pengaturan progres materi")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	h.store.Flash(c, "success", "Pengaturan progres materi tersimpan")
	return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
}

func (h *KelasHandler) GantiLevel(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	var req models.GantiLevelKelasRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data level tidak valid")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	sess, _ := h.store.Get(c)
	if err := h.kelasPerubahanService.GantiLevelKelas(id, req.Level, sessionUser(sess).ID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	h.store.Flash(c, "success", "Level kelas berhasil diganti. Pertemuan berikutnya dimulai dari ke-1.")
	return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
}

func (h *KelasHandler) GantiJadwal(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	var req models.GantiJadwalKelasRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data jadwal tidak valid")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	sess, _ := h.store.Get(c)
	if err := h.kelasPerubahanService.GantiJadwalKelas(id, req.Jadwal, sessionUser(sess).ID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	h.store.Flash(c, "success", "Jadwal kelas berhasil diganti")
	return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
}

func (h *KelasHandler) GantiLevelSantri(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}
	var req models.GantiLevelSantriRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data ganti level santri tidak valid")
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	sess, _ := h.store.Get(c)
	if _, err := h.kelasPerubahanService.GantiLevelSantri(id, req, sessionUser(sess).ID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
	}
	h.store.Flash(c, "success", fmt.Sprintf("%d santri berhasil dipindahkan ke level baru", len(req.SantriIDs)))
	return h.inertiaService.Redirect(c, kelasDetailReturnURL(c, c.Params("id")))
}

func (h *KelasHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/kelas")
	}

	if err := h.kelasService.Delete(id); err != nil {
		h.store.Flash(c, "error", "Gagal menghapus kelas")
		return h.inertiaService.Redirect(c, kelasListReturnURL(c))
	}

	h.store.Flash(c, "success", "Kelas berhasil dihapus")
	return h.inertiaService.Redirect(c, kelasListReturnURL(c))
}
