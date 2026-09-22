package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type KoordinatorMonitoringHandler struct {
	monitoringService  *services.KoordinatorMonitoringService
	koordinatorService *services.KoordinatorService
	store              *session.Store
	inertiaService     *services.InertiaService
}

func NewKoordinatorMonitoringHandler(monitoringService *services.KoordinatorMonitoringService, koordinatorService *services.KoordinatorService, store *session.Store, inertiaService *services.InertiaService) *KoordinatorMonitoringHandler {
	return &KoordinatorMonitoringHandler{
		monitoringService:  monitoringService,
		koordinatorService: koordinatorService,
		store:              store,
		inertiaService:     inertiaService,
	}
}

func (h *KoordinatorMonitoringHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	today := time.Now().Format("2006-01-02")
	filter := models.ClassMonitoringFilter{
		StartDate: c.Query("start_date", today),
		EndDate:   c.Query("end_date", c.Query("start_date", today)),
		GuruID:    int64(c.QueryInt("guru_id")),
		KelasID:   int64(c.QueryInt("kelas_id")),
		Status:    c.Query("status"),
	}
	monitoring, err := h.monitoringService.List(filter)
	if err != nil {
		return h.inertiaService.Render(c, "koordinator/MonitoringKelas", fiber.Map{"user": user, "error": "Gagal memuat monitoring: " + err.Error(), "filters": filter})
	}
	guru, _ := h.koordinatorService.ListGuruSimple()
	kelas, _ := h.koordinatorService.ListKelasSimple()
	return h.inertiaService.Render(c, "koordinator/MonitoringKelas", fiber.Map{
		"user":       user,
		"monitoring": monitoring,
		"guru":       guru,
		"kelas":      kelas,
		"filters":    filter,
	})
}

func (h *KoordinatorMonitoringHandler) SendReminder(c *fiber.Ctx) error {
	scheduleID, kelasID := monitoringParams(c)
	var req models.MonitoringReminderRequest
	if err := c.BodyParser(&req); err != nil {
		return h.backWithError(c, "Data pengingat tidak valid")
	}
	if err := h.monitoringService.SendReminder(h.userID(c), scheduleID, kelasID, req); err != nil {
		return h.backWithError(c, "Gagal mengirim pengingat: "+err.Error())
	}
	h.store.Flash(c, "success", "Pengingat berhasil dikirim ke akun guru")
	return h.inertiaService.Back(c, "/app/koordinator-guru/monitoring-kelas")
}

func (h *KoordinatorMonitoringHandler) AddNote(c *fiber.Ctx) error {
	scheduleID, kelasID := monitoringParams(c)
	var req models.MonitoringNoteRequest
	if err := c.BodyParser(&req); err != nil {
		return h.backWithError(c, "Data catatan tidak valid")
	}
	if err := h.monitoringService.AddNote(h.userID(c), scheduleID, kelasID, req); err != nil {
		return h.backWithError(c, "Gagal menyimpan catatan: "+err.Error())
	}
	h.store.Flash(c, "success", "Catatan tindak lanjut tersimpan")
	return h.inertiaService.Back(c, "/app/koordinator-guru/monitoring-kelas")
}

func (h *KoordinatorMonitoringHandler) Reschedule(c *fiber.Ctx) error {
	scheduleID, kelasID := monitoringParams(c)
	var req models.RescheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return h.backWithError(c, "Data jadwal tidak valid")
	}
	if err := h.monitoringService.Reschedule(h.userID(c), scheduleID, kelasID, req); err != nil {
		return h.backWithError(c, "Gagal mengubah jadwal: "+err.Error())
	}
	h.store.Flash(c, "success", "Jadwal kelas berhasil diubah")
	return h.inertiaService.Back(c, "/app/koordinator-guru/monitoring-kelas")
}

func (h *KoordinatorMonitoringHandler) AssignSubstitute(c *fiber.Ctx) error {
	scheduleID, kelasID := monitoringParams(c)
	var req models.BadalRequest
	if err := c.BodyParser(&req); err != nil || req.GuruPenggantiID <= 0 {
		return h.backWithError(c, "Guru pengganti wajib dipilih")
	}
	if err := h.monitoringService.AssignSubstitute(h.userID(c), scheduleID, kelasID, req); err != nil {
		return h.backWithError(c, "Gagal menetapkan guru pengganti: "+err.Error())
	}
	h.store.Flash(c, "success", "Guru pengganti berhasil ditetapkan")
	return h.inertiaService.Back(c, "/app/koordinator-guru/monitoring-kelas")
}

func (h *KoordinatorMonitoringHandler) Cancel(c *fiber.Ctx) error {
	scheduleID, kelasID := monitoringParams(c)
	var req models.MonitoringCancelRequest
	if err := c.BodyParser(&req); err != nil {
		return h.backWithError(c, "Data pembatalan tidak valid")
	}
	if err := h.monitoringService.Cancel(h.userID(c), scheduleID, kelasID, req); err != nil {
		return h.backWithError(c, "Gagal membatalkan jadwal: "+err.Error())
	}
	h.store.Flash(c, "success", "Jadwal kelas berhasil dibatalkan")
	return h.inertiaService.Back(c, "/app/koordinator-guru/monitoring-kelas")
}

func (h *KoordinatorMonitoringHandler) userID(c *fiber.Ctx) int64 {
	sess, _ := h.store.Get(c)
	return toInt64(sess.Get("user_id"))
}

func (h *KoordinatorMonitoringHandler) backWithError(c *fiber.Ctx, message string) error {
	h.store.Flash(c, "error", message)
	return h.inertiaService.Back(c, "/app/koordinator-guru/monitoring-kelas")
}

func monitoringParams(c *fiber.Ctx) (int64, int64) {
	scheduleID, _ := strconv.ParseInt(c.Params("scheduleID"), 10, 64)
	kelasID, _ := strconv.ParseInt(c.Params("kelasID"), 10, 64)
	return scheduleID, kelasID
}
