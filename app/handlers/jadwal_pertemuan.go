package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type JadwalPertemuanHandler struct {
	guruService    *services.GuruService
	jadwalService  *services.JadwalPertemuanService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewJadwalPertemuanHandler(guruService *services.GuruService, jadwalService *services.JadwalPertemuanService, store *session.Store, inertiaService *services.InertiaService) *JadwalPertemuanHandler {
	return &JadwalPertemuanHandler{
		guruService:    guruService,
		jadwalService:  jadwalService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *JadwalPertemuanHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru")
	}

	jadwal, err := h.jadwalService.List(guruID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat jadwal pertemuan")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	kelas, err := h.guruService.ListKelasForViewer(guruID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat daftar kelas")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	guruList, _ := h.guruService.ListDirectory()

	return h.inertiaService.Render(c, "guru/JadwalPertemuan", fiber.Map{
		"user":              user,
		"jadwal":            jadwal,
		"kelas":             kelas,
		"guruList":          guruList,
		"selected_kelas_id": c.QueryInt("kelas_id"),
	})
}

func (h *JadwalPertemuanHandler) Create(c *fiber.Ctx) error {
	var req models.BuatJadwalPertemuanRequest
	if err := c.BodyParser(&req); err != nil || req.KelasID == 0 {
		h.store.Flash(c, "error", "Kelas, tanggal, dan jam wajib diisi")
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	userID, err := h.ensureCanManageClass(c, req.KelasID)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	if _, err := h.jadwalService.Create(userID, req); err != nil {
		h.store.Flash(c, "error", "Gagal membuat jadwal: "+err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}

	h.store.Flash(c, "success", "Jadwal pertemuan berhasil dibuat")
	return h.inertiaService.Redirect(c, "/app/guru/jadwal-pertemuan")
}

func (h *JadwalPertemuanHandler) Reschedule(c *fiber.Ctx) error {
	id, kelasID := scheduleParams(c)
	if _, err := h.ensureCanManageClass(c, kelasID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	var req models.RescheduleRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data jadwal tidak valid")
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	if err := h.jadwalService.Reschedule(id, kelasID, req); err != nil {
		h.store.Flash(c, "error", "Gagal menjadwalkan ulang: "+err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}

	h.store.Flash(c, "success", "Pertemuan berhasil dijadwalkan ulang")
	return h.inertiaService.Redirect(c, "/app/guru/jadwal-pertemuan")
}

func (h *JadwalPertemuanHandler) Badal(c *fiber.Ctx) error {
	id, kelasID := scheduleParams(c)
	if _, err := h.ensureCanManageClass(c, kelasID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	var req models.BadalRequest
	if err := c.BodyParser(&req); err != nil || req.GuruPenggantiID == 0 {
		h.store.Flash(c, "error", "Guru badal wajib dipilih")
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	if err := h.jadwalService.Badal(id, kelasID, req); err != nil {
		h.store.Flash(c, "error", "Gagal menetapkan badal: "+err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}

	h.store.Flash(c, "success", "Guru badal berhasil ditetapkan")
	return h.inertiaService.Redirect(c, "/app/guru/jadwal-pertemuan")
}

func (h *JadwalPertemuanHandler) Cancel(c *fiber.Ctx) error {
	id, kelasID := scheduleParams(c)
	if _, err := h.ensureCanManageClass(c, kelasID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	if err := h.jadwalService.Cancel(id, kelasID); err != nil {
		h.store.Flash(c, "error", "Gagal membatalkan jadwal: "+err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}

	h.store.Flash(c, "success", "Jadwal pertemuan dibatalkan")
	return h.inertiaService.Redirect(c, "/app/guru/jadwal-pertemuan")
}

func (h *JadwalPertemuanHandler) Start(c *fiber.Ctx) error {
	id, kelasID := scheduleParams(c)
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	role := requestRole(c, user)
	privileged := role == models.RoleSuperAdmin || role == models.RoleAdminKelas

	pertemuan, err := h.jadwalService.Start(id, kelasID, userID, privileged)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memulai pertemuan: "+err.Error())
		return h.inertiaService.Back(c, "/app/guru/jadwal-pertemuan")
	}
	return h.inertiaService.Redirect(c, "/app/guru/kelas/"+strconv.FormatInt(kelasID, 10)+"/pertemuan/"+strconv.FormatInt(pertemuan.ID, 10)+"/selesai")
}

func (h *JadwalPertemuanHandler) ensureCanManageClass(c *fiber.Ctx, kelasID int64) (int64, error) {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))
	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		return 0, err
	}
	if err := h.guruService.EnsureCanAccessClass(guruID, kelasID); err != nil {
		return 0, err
	}
	return userID, nil
}

func scheduleParams(c *fiber.Ctx) (int64, int64) {
	id, _ := strconv.ParseInt(c.Params("jid"), 10, 64)
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	return id, kelasID
}
