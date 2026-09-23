package handlers

import (
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type SantriAnalyticsHandler struct {
	analyticsService *services.SantriAnalyticsService
	masterService    *services.MasterService
	store            *session.Store
	inertiaService   *services.InertiaService
}

func NewSantriAnalyticsHandler(analyticsService *services.SantriAnalyticsService, masterService *services.MasterService, store *session.Store, inertiaService *services.InertiaService) *SantriAnalyticsHandler {
	return &SantriAnalyticsHandler{analyticsService: analyticsService, masterService: masterService, store: store, inertiaService: inertiaService}
}

func (h *SantriAnalyticsHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	filters := models.SantriAnalyticsFilters{
		DateFrom: c.Query("date_from", ""),
		DateTo: c.Query("date_to", ""),
		AngkatanPendaftaran: c.Query("angkatan_pendaftaran", ""),
		AngkatanKelas: c.Query("angkatan_kelas", ""),
		Level: c.Query("level", ""),
		Tipe: c.Query("tipe", ""),
		SegmentAgeBucket: c.Query("segment_age", ""),
		SegmentGender: c.Query("segment_gender", ""),
		SegmentDomisili: c.Query("segment_domisili", ""),
		SegmentStatus: c.Query("segment_status", ""),
		SegmentRegistrationMo: c.Query("segment_month", ""),
		Page: page,
		Limit: 25,
	}

	normalized, normalizeErr := services.NormalizeSantriAnalyticsFilters(filters)
	if normalizeErr == nil {
		filters = normalized
	}

	angkatan, _ := h.masterService.ListAngkatan()
	levels, _ := h.masterService.ListLevel()
	kodeKelas, _ := h.masterService.ListKodeKelas()
	tipeSet := map[string]struct{}{}
	for _, item := range kodeKelas {
		if item.Tipe != "" {
			tipeSet[item.Tipe] = struct{}{}
		}
	}
	tipes := make([]string, 0, len(tipeSet))
	for tipe := range tipeSet {
		tipes = append(tipes, tipe)
	}
	sort.Strings(tipes)

	props := fiber.Map{
		"user": user,
		"filters": filters,
		"angkatan": angkatan,
		"levels": levels,
		"tipes": tipes,
		"analytics": models.SantriAnalyticsData{},
	}
	if normalizeErr != nil {
		props["error"] = "Filter analisis tidak valid: " + normalizeErr.Error()
		return h.inertiaService.Render(c, "app/SantriAnalytics", props)
	}

	analytics, err := h.analyticsService.Get(filters)
	if err != nil {
		props["error"] = "Gagal memuat analisis santri"
		return h.inertiaService.Render(c, "app/SantriAnalytics", props)
	}
	props["analytics"] = analytics
	return h.inertiaService.Render(c, "app/SantriAnalytics", props)
}
