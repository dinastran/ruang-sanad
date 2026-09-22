package handlers

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type GuruHandler struct {
	guruService         *services.GuruService
	pertemuanService    *services.PertemuanService
	riayahService       *services.RiayahService
	notificationService *services.NotificationService
	store               *session.Store
	inertiaService      *services.InertiaService
}

func NewGuruHandler(guruService *services.GuruService, pertemuanService *services.PertemuanService, riayahService *services.RiayahService, store *session.Store, inertiaService *services.InertiaService, notificationServices ...*services.NotificationService) *GuruHandler {
	h := &GuruHandler{
		guruService:      guruService,
		pertemuanService: pertemuanService,
		riayahService:    riayahService,
		store:            store,
		inertiaService:   inertiaService,
	}
	if len(notificationServices) > 0 {
		h.notificationService = notificationServices[0]
	}
	return h
}

func (h *GuruHandler) Dashboard(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))

	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		slog.Error("guru dashboard viewer lookup failed", "user_id", userID, "error", err)
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		// /app routes teachers back here; use a destination independent of guru data.
		return h.inertiaService.Redirect(c, "/app/profile")
	}

	dashboard, err := h.guruService.GetDashboardForViewer(guruID)
	if err != nil {
		slog.Error("guru dashboard load failed", "user_id", userID, "error", err)
		h.store.Flash(c, "error", "Gagal memuat dashboard")
		return h.inertiaService.Redirect(c, "/app/profile")
	}

	var tilawah interface{}
	if guruID != nil {
		tilawah, _ = h.guruService.GetTilawahStatus(*guruID)
	}

	// Never send null: the dashboard reads notifications.length directly.
	notifications := []models.NotificationResponse{}
	if h.notificationService != nil {
		if list, err := h.notificationService.ListForUser(userID); err == nil {
			notifications = list
		}
	}

	return h.inertiaService.Render(c, "guru/Dashboard", fiber.Map{
		"user":          user,
		"dashboard":     dashboard,
		"tilawah":       tilawah,
		"notifications": notifications,
	})
}

// TilawahCheckin marks today's tilawah done for the logged-in guru.
func (h *GuruHandler) TilawahCheckin(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	guru, err := h.guruService.GetGuruByUserID(userID)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	if err := h.guruService.CheckinTilawah(guru.ID); err != nil {
		h.store.Flash(c, "error", "Gagal mencatat tilawah")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	h.store.Flash(c, "success", "Tilawah hari ini tercatat")
	return h.inertiaService.Redirect(c, "/app/guru")
}

// TilawahUncheck removes today's tilawah check-in (undo).
func (h *GuruHandler) TilawahUncheck(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	guru, err := h.guruService.GetGuruByUserID(userID)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	if err := h.guruService.UncheckTilawah(guru.ID); err != nil {
		h.store.Flash(c, "error", "Gagal membatalkan tilawah")
		return h.inertiaService.Redirect(c, "/app/guru")
	}
	h.store.Flash(c, "success", "Tilawah hari ini dibatalkan")
	return h.inertiaService.Redirect(c, "/app/guru")
}

func (h *GuruHandler) KelasSaya(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	userID := toInt64(sess.Get("user_id"))

	guruID, err := viewerGuruIDForRequest(c, h.guruService, userID, user)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app")
	}

	kelas, err := h.guruService.ListKelasForViewer(guruID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat daftar kelas")
		return h.inertiaService.Redirect(c, "/app/guru")
	}

	return h.inertiaService.Render(c, "guru/KelasList", fiber.Map{
		"user":  user,
		"kelas": kelas,
	})
}

func (h *GuruHandler) DetailKelas(c *fiber.Ctx) error {
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
	activePertemuan, err := h.pertemuanService.GetActivePertemuan(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat pertemuan aktif")
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}
	lastPertemuan, lastAbsensi, err := h.pertemuanService.GetLastCompletedAbsensi(kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat absensi pertemuan terakhir")
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}

	return h.inertiaService.Render(c, "guru/KelasDetail", fiber.Map{
		"user":             user,
		"kelas":            kelas,
		"santri":           santri,
		"active_pertemuan": activePertemuan,
		"last_pertemuan":   lastPertemuan,
		"last_absensi":     lastAbsensi,
	})
}

func viewerGuruID(guruService *services.GuruService, userID int64, user *models.UserResponse) (*int64, error) {
	return guruService.GetViewerGuruID(userID, user.Role == models.RoleSuperAdmin || user.Role == models.RoleAdminKelas)
}

func viewerGuruIDForRequest(c *fiber.Ctx, guruService *services.GuruService, userID int64, user *models.UserResponse) (*int64, error) {
	role := requestRole(c, user)
	user.Role = role
	return guruService.GetViewerGuruID(userID, role == models.RoleSuperAdmin || role == models.RoleAdminKelas)
}

func requestRole(c *fiber.Ctx, user *models.UserResponse) models.UserRole {
	if role, ok := c.Locals("role").(string); ok && role != "" {
		return models.UserRole(role)
	}
	return user.Role
}

func (h *GuruHandler) RiwayatSantri(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)

	guru, err := h.guruService.GetGuruByUserID(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Guru not found"})
	}

	if err := h.guruService.EnsureOwnsClass(guru.ID, kelasID); err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	riwayat, err := h.guruService.GetRiwayatSantri(guru.ID, santriID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(riwayat)
}
