package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type DashboardHandler struct {
	santriService  *services.SantriService
	laporanService *services.LaporanService
	kelasService   *services.KelasService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewDashboardHandler(santriService *services.SantriService, laporanService *services.LaporanService, kelasService *services.KelasService, store *session.Store, inertiaService *services.InertiaService) *DashboardHandler {
	return &DashboardHandler{
		santriService:  santriService,
		laporanService: laporanService,
		kelasService:   kelasService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *DashboardHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	role := toStr(sess.Get("role"))

	stats, err := h.laporanService.GetDashboardStats()
	if err != nil {
		stats = &models.DashboardStats{}
	}

	var roleData fiber.Map
	switch role {
	case "cs":
		roleData = fiber.Map{}
	case "admin_kelas":
		perluLengkap, _ := h.santriService.ListPerluDilengkapi()
		roleData = fiber.Map{
			"perlu_dilengkapi": perluLengkap,
		}
	case "keuangan":
		roleData = fiber.Map{
			"nominal_per_angkatan": stats.NominalPerAngkatan,
		}
	case "super_admin":
		perluLengkap, _ := h.santriService.ListPerluDilengkapi()
		roleData = fiber.Map{
			"perlu_dilengkapi":     perluLengkap,
			"nominal_per_angkatan": stats.NominalPerAngkatan,
		}
	default:
		roleData = fiber.Map{}
	}

	return h.inertiaService.Render(c, "app/Dashboard", fiber.Map{
		"user":      user,
		"stats":     stats,
		"role_data": roleData,
	})
}
