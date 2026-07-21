package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type LaporanHandler struct {
	laporanService *services.LaporanService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLaporanHandler(laporanService *services.LaporanService, store *session.Store, inertiaService *services.InertiaService) *LaporanHandler {
	return &LaporanHandler{
		laporanService: laporanService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *LaporanHandler) Keuangan(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	stats, err := h.laporanService.GetDashboardStats()
	if err != nil {
		return h.inertiaService.Render(c, "app/LaporanKeuangan", fiber.Map{
			"user":  user,
			"error": "Gagal memuat laporan",
		})
	}

	tidakLanjut, _ := h.laporanService.ListTidakLanjut()
	infaqRekap, _ := h.laporanService.ListInfaqRekap()

	return h.inertiaService.Render(c, "app/LaporanKeuangan", fiber.Map{
		"user":             user,
		"nominal_angkatan": stats.NominalPerAngkatan,
		"tidak_lanjut":     tidakLanjut,
		"infaq_rekap":      infaqRekap,
	})
}
