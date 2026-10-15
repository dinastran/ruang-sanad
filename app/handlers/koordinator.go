package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type KoordinatorGuruHandler struct {
	svc               *services.KoordinatorService
	monitoringService *services.KoordinatorMonitoringService
	pertemuanService  *services.PertemuanService
	store             *session.Store
	inertiaService    *services.InertiaService
}

func NewKoordinatorGuruHandler(svc *services.KoordinatorService, monitoringService *services.KoordinatorMonitoringService, pertemuanService *services.PertemuanService, store *session.Store, inertiaService *services.InertiaService) *KoordinatorGuruHandler {
	return &KoordinatorGuruHandler{
		svc:               svc,
		monitoringService: monitoringService,
		pertemuanService:  pertemuanService,
		store:             store,
		inertiaService:    inertiaService,
	}
}

// Dashboard is the koordinator's daily starting point. Each section loads
// independently so one failing source does not blank the whole page.
func (h *KoordinatorGuruHandler) Dashboard(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	now := time.Now()
	props := fiber.Map{"user": user}

	if dashboard, err := h.svc.GetDashboard(now); err != nil {
		props["error"] = "Gagal memuat ringkasan dashboard"
	} else {
		props["dashboard"] = dashboard
	}

	// Filter kosong = hari ini dalam WIB, sama dengan tanggal di GetDashboard.
	if monitoring, err := h.monitoringService.List(models.ClassMonitoringFilter{}); err != nil {
		props["kelas_error"] = "Gagal memuat kelas hari ini"
	} else {
		props["kelas_hari_ini"] = monitoring.Items
	}

	if belumDitutup, err := h.pertemuanService.ListBelumDitutup(nil, now); err != nil {
		props["belum_ditutup_error"] = "Gagal memuat pertemuan belum ditutup"
	} else {
		props["belum_ditutup"] = belumDitutup
	}

	return h.inertiaService.Render(c, "koordinator/Dashboard", props)
}
