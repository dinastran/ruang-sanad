package handlers

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// RiayahSantriHandler melayani halaman Riayah Santri dan profil santri untuk guru.
type RiayahSantriHandler struct {
	guruService         *services.GuruService
	riayahSantriService *services.RiayahSantriService
	store               *session.Store
	inertiaService      *services.InertiaService
}

func NewRiayahSantriHandler(guruService *services.GuruService, riayahSantriService *services.RiayahSantriService, store *session.Store, inertiaService *services.InertiaService) *RiayahSantriHandler {
	return &RiayahSantriHandler{
		guruService:         guruService,
		riayahSantriService: riayahSantriService,
		store:               store,
		inertiaService:      inertiaService,
	}
}

// riayahViewer menerjemahkan role menjadi cakupan data:
// guru = santri di kelasnya; admin_kelas/super_admin = semua (tulis);
// koordinator_guru = semua (baca saja).
func riayahViewer(c *fiber.Ctx, guruService *services.GuruService, userID int64, user *models.UserResponse) (models.RiayahViewer, error) {
	viewer := models.RiayahViewer{UserID: userID}
	switch requestRole(c, user) {
	case models.RoleSuperAdmin, models.RoleAdminKelas:
		viewer.CanWrite = true
		return viewer, nil
	case models.RoleKoordinator:
		return viewer, nil
	case models.RoleGuru:
		guru, err := guruService.GetGuruByUserID(userID)
		if err != nil {
			return viewer, err
		}
		viewer.GuruID = &guru.ID
		viewer.CanWrite = true
		return viewer, nil
	}
	return viewer, services.ErrRiayahAksesDitolak
}

func (h *RiayahSantriHandler) viewer(c *fiber.Ctx) (models.RiayahViewer, *models.UserResponse, error) {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	viewer, err := riayahViewer(c, h.guruService, userID, user)
	return viewer, user, err
}

func (h *RiayahSantriHandler) Index(c *fiber.Ctx) error {
	viewer, user, err := h.viewer(c)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/profile")
	}

	items, ringkasan, err := h.riayahSantriService.ListPerhatian(viewer, time.Now())
	if err != nil {
		slog.Error("riayah santri list failed", "user_id", viewer.UserID, "error", err)
		h.store.Flash(c, "error", "Gagal memuat data riayah santri")
		return h.inertiaService.Redirect(c, "/app/guru")
	}

	return h.inertiaService.Render(c, "guru/RiayahSantri", fiber.Map{
		"user":      user,
		"santri":    items,
		"ringkasan": ringkasan,
		"can_write": viewer.CanWrite,
		"is_semua":  viewer.GuruID == nil,
	})
}

func (h *RiayahSantriHandler) Profil(c *fiber.Ctx) error {
	viewer, user, err := h.viewer(c)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/profile")
	}
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)

	profil, err := h.riayahSantriService.GetProfil(viewer, santriID, time.Now())
	if err != nil {
		if !errors.Is(err, services.ErrRiayahAksesDitolak) {
			slog.Error("riayah santri profil failed", "santri_id", santriID, "error", err)
		}
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/riayah")
	}

	return h.inertiaService.Render(c, "guru/SantriProfil", fiber.Map{
		"user":      user,
		"profil":    profil,
		"can_write": viewer.CanWrite,
	})
}

func (h *RiayahSantriHandler) CatatanCreate(c *fiber.Ctx) error {
	viewer, _, err := h.viewer(c)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)
	back := "/app/guru/santri/" + c.Params("sid")
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, back)
	}

	var req struct {
		Catatan string `json:"catatan" form:"catatan"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.riayahSantriService.TambahCatatan(viewer, santriID, req.Catatan); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, back)
	}
	h.store.Flash(c, "success", "Catatan riayah tersimpan")
	return h.inertiaService.Redirect(c, back)
}

func (h *RiayahSantriHandler) CatatanDelete(c *fiber.Ctx) error {
	viewer, _, err := h.viewer(c)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)
	catatanID, _ := strconv.ParseInt(c.Params("rid"), 10, 64)
	back := "/app/guru/santri/" + c.Params("sid")
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.riayahSantriService.HapusCatatan(viewer, santriID, catatanID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, back)
	}
	h.store.Flash(c, "success", "Catatan riayah dihapus")
	return h.inertiaService.Redirect(c, back)
}

func (h *RiayahSantriHandler) KontakCreate(c *fiber.Ctx) error {
	viewer, _, err := h.viewer(c)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)
	back := c.Query("kembali")
	if back != "/app/guru/riayah" {
		back = "/app/guru/santri/" + c.Params("sid")
	}
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, back)
	}

	var input models.RiayahKontakInput
	if err := c.BodyParser(&input); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.riayahSantriService.CatatKontak(viewer, santriID, input, time.Now()); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, back)
	}
	pesan := "Kontak dengan santri tercatat"
	if input.Jenis == "rapor" {
		pesan = "Pengiriman rapor tercatat"
	}
	h.store.Flash(c, "success", pesan)
	return h.inertiaService.Redirect(c, back)
}

func (h *RiayahSantriHandler) KontakDelete(c *fiber.Ctx) error {
	viewer, _, err := h.viewer(c)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)
	kontakID, _ := strconv.ParseInt(c.Params("kid"), 10, 64)
	back := "/app/guru/santri/" + c.Params("sid")
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, back)
	}
	if err := h.riayahSantriService.HapusKontak(viewer, santriID, kontakID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, back)
	}
	h.store.Flash(c, "success", "Log kontak dihapus")
	return h.inertiaService.Redirect(c, back)
}
