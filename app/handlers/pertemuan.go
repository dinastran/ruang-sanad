package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type PertemuanHandler struct {
	guruService      *services.GuruService
	pertemuanService *services.PertemuanService
	jadwalService    *services.JadwalPertemuanService
	store            *session.Store
	inertiaService   *services.InertiaService
}

func NewPertemuanHandler(guruService *services.GuruService, pertemuanService *services.PertemuanService, jadwalService *services.JadwalPertemuanService, store *session.Store, inertiaService *services.InertiaService) *PertemuanHandler {
	return &PertemuanHandler{
		guruService:      guruService,
		pertemuanService: pertemuanService,
		jadwalService:    jadwalService,
		store:            store,
		inertiaService:   inertiaService,
	}
}

func (h *PertemuanHandler) ensureClassAccess(c *fiber.Ctx, kelasID int64) (int64, error) {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, sessionUser(sess))
	if err != nil {
		return 0, err
	}
	if err := h.guruService.EnsureCanAccessClass(guruID, kelasID); err != nil {
		return 0, err
	}
	return userID, nil
}

func (h *PertemuanHandler) ensureMeetingAccess(c *fiber.Ctx, kelasID, pertemuanID int64) (int64, error) {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	role := requestRole(c, user)
	privileged := role == models.RoleSuperAdmin || role == models.RoleAdminKelas
	if err := h.pertemuanService.EnsureCanAccessPertemuan(pertemuanID, kelasID, userID, privileged); err != nil {
		return 0, err
	}
	return userID, nil
}

func (h *PertemuanHandler) canManageClass(c *fiber.Ctx, kelasID int64) bool {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		return false
	}
	return h.guruService.EnsureCanAccessClass(guruID, kelasID) == nil
}

func (h *PertemuanHandler) Mulai(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	userID, err := h.ensureClassAccess(c, kelasID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.MulaiPertemuanRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Back(c, "/app/guru/kelas/"+c.Params("id")+"/pertemuan/mulai")
	}

	var pertemuan *models.PertemuanResponse
	if req.JadwalID != 0 {
		sess, _ := h.store.Get(c)
		role := requestRole(c, sessionUser(sess))
		privileged := role == models.RoleSuperAdmin || role == models.RoleAdminKelas
		pertemuan, err = h.jadwalService.Start(req.JadwalID, kelasID, userID, privileged)
	} else {
		pertemuan, err = h.pertemuanService.MulaiPertemuan(kelasID, userID, req)
	}
	if err != nil {
		h.store.Flash(c, "error", "Gagal memulai pertemuan: "+err.Error())
		if errors.Is(err, services.ErrPertemuanBerlangsung) {
			if active, activeErr := h.pertemuanService.GetActivePertemuan(kelasID); activeErr == nil && active != nil {
				return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id")+"/pertemuan/"+strconv.FormatInt(active.ID, 10)+"/selesai")
			}
		}
		return h.inertiaService.Back(c, "/app/guru/kelas/"+c.Params("id")+"/pertemuan/mulai")
	}

	return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id")+"/pertemuan/"+strconv.FormatInt(pertemuan.ID, 10)+"/selesai")
}

func (h *PertemuanHandler) FormMulai(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)

	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app")
	}

	kelas, santri, err := h.guruService.GetDetailKelasForViewer(guruID, kelasID)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}
	active, err := h.pertemuanService.GetActivePertemuan(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat pertemuan aktif")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}
	if active != nil {
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id")+"/pertemuan/"+strconv.FormatInt(active.ID, 10)+"/selesai")
	}
	nextPertemuanKe, err := h.pertemuanService.GetNextPertemuanLevelKe(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat nomor pertemuan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}
	jadwalTersedia, err := h.jadwalService.ListDue(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat jadwal pertemuan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}

	return h.inertiaService.Render(c, "guru/PertemuanMulai", fiber.Map{
		"user":              user,
		"kelas":             kelas,
		"santri":            santri,
		"next_pertemuan_ke": nextPertemuanKe,
		"jadwal_tersedia":   jadwalTersedia,
	})
}

func (h *PertemuanHandler) Selesai(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	pertemuanID, _ := strconv.ParseInt(c.Params("pid"), 10, 64)
	userID, err := h.ensureMeetingAccess(c, kelasID, pertemuanID)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}
	canManageClass := h.canManageClass(c, kelasID)

	var req models.SelesaiPertemuanRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Back(c, "/app/guru/kelas/"+c.Params("id"))
	}

	if err := h.pertemuanService.SelesaiPertemuan(pertemuanID, kelasID, req, userID); err != nil {
		h.store.Flash(c, "error", "Gagal menyelesaikan pertemuan: "+err.Error())
		return h.inertiaService.Back(c, "/app/guru/kelas/"+c.Params("id"))
	}

	h.store.Flash(c, "success", "Pertemuan selesai dan absensi tersimpan")
	if !canManageClass {
		return h.inertiaService.Redirect(c, "/app/guru/jadwal-pertemuan")
	}
	return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
}

func (h *PertemuanHandler) FormSelesai(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	user.Role = requestRole(c, user)
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	pertemuanID, _ := strconv.ParseInt(c.Params("pid"), 10, 64)

	if _, err := h.ensureMeetingAccess(c, kelasID, pertemuanID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru")
	}

	kelas, santri, err := h.guruService.GetDetailKelasForViewer(nil, kelasID)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}

	pertemuan, _, err := h.pertemuanService.GetPertemuanByID(pertemuanID, kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Pertemuan tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}
	if pertemuan.Status != "berlangsung" {
		h.store.Flash(c, "error", "Pertemuan ini sudah selesai")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}
	batasMateriTerakhir, err := h.pertemuanService.GetBatasMateriTerakhir(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat batas materi terakhir")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}

	return h.inertiaService.Render(c, "guru/PertemuanSelesai", fiber.Map{
		"user":                  user,
		"kelas":                 kelas,
		"santri":                santri,
		"pertemuan":             pertemuan,
		"can_manage_class":      h.canManageClass(c, kelasID),
		"batas_materi_terakhir": batasMateriTerakhir,
	})
}

func (h *PertemuanHandler) Detail(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	user.Role = requestRole(c, user)
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	pertemuanID, _ := strconv.ParseInt(c.Params("pid"), 10, 64)

	if _, err := h.ensureMeetingAccess(c, kelasID, pertemuanID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru")
	}

	pertemuan, _, err := h.pertemuanService.GetPertemuanByID(pertemuanID, kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Pertemuan tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}
	if pertemuan.Status != "berlangsung" {
		h.store.Flash(c, "error", "Pertemuan ini sudah selesai")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}
	batasMateriTerakhir, err := h.pertemuanService.GetBatasMateriTerakhir(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat batas materi terakhir")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}

	kelas, santri, err := h.guruService.GetDetailKelasForViewer(nil, kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Kelas tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}

	return h.inertiaService.Render(c, "guru/PertemuanSelesai", fiber.Map{
		"user":                  user,
		"kelas":                 kelas,
		"santri":                santri,
		"pertemuan":             pertemuan,
		"can_manage_class":      h.canManageClass(c, kelasID),
		"batas_materi_terakhir": batasMateriTerakhir,
	})
}

func (h *PertemuanHandler) RekapAbsensi(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)

	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app")
	}

	if err := h.guruService.EnsureCanAccessClass(guruID, kelasID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}
	kelas, _, err := h.guruService.GetDetailKelasForViewer(guruID, kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Kelas tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}

	santri, pertemuan, absensi, err := h.pertemuanService.GetRekapAbsensi(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat rekap absensi")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}

	return h.inertiaService.Render(c, "guru/RekapAbsensi", fiber.Map{
		"user":           user,
		"kelas":          kelas,
		"santri":         santri,
		"pertemuan_list": pertemuan,
		"absensi":        absensi,
	})
}

func (h *PertemuanHandler) Riwayat(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)

	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	kelas, _, err := h.guruService.GetDetailKelasForViewer(guruID, kelasID)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}
	pertemuan, err := h.pertemuanService.ListRiwayat(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat riwayat pertemuan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}

	return h.inertiaService.Render(c, "guru/RiwayatPertemuan", fiber.Map{
		"user":      user,
		"kelas":     kelas,
		"pertemuan": pertemuan,
	})
}

func (h *PertemuanHandler) EditAbsensi(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	absensiID, _ := strconv.ParseInt(c.Params("aid"), 10, 64)
	if _, err := h.ensureClassAccess(c, kelasID); err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	var req struct {
		Status      string `json:"status"`
		Catatan     string `json:"catatan"`
		BatasMateri string `json:"batas_materi"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	if err := h.pertemuanService.EditAbsensi(kelasID, absensiID, req.Status, req.Catatan, req.BatasMateri); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}
