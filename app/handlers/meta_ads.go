package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// MetaAdsHandler serves /app/leads/iklan: Meta Ads performance next to CRM
// leads, campaign → lead source mapping, and a manual sync (super_admin).
type MetaAdsHandler struct {
	ads            *services.MetaAdsService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewMetaAdsHandler(ads *services.MetaAdsService, store *session.Store, inertiaService *services.InertiaService) *MetaAdsHandler {
	return &MetaAdsHandler{ads: ads, store: store, inertiaService: inertiaService}
}

func (h *MetaAdsHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	report, err := h.ads.Report(c.UserContext(), c.Query("from"), c.Query("to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	return h.inertiaService.Render(c, "app/MetaAds", fiber.Map{
		"user":   sessionUser(sess),
		"report": report,
	})
}

// back returns to the page, keeping the chosen period.
func (h *MetaAdsHandler) back(c *fiber.Ctx) error {
	return h.inertiaService.Back(c, "/app/leads/iklan")
}

// Sync pulls the last 28 days from Meta now.
func (h *MetaAdsHandler) Sync(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Minute)
	defer cancel()
	rows, err := h.ads.Sync(ctx, "manual")
	switch {
	case errors.Is(err, services.ErrMetaAdsDisabled), errors.Is(err, services.ErrMetaAdsBusy):
		h.store.Flash(c, "error", err.Error())
	case err != nil:
		h.store.Flash(c, "error", "Sinkron gagal: "+err.Error())
	default:
		h.store.Flash(c, "success", fmt.Sprintf("Sinkron selesai: %d baris data iklan (28 hari terakhir)", rows))
	}
	return h.back(c)
}

// MapCampaign links a campaign to a lead source (source_id 0 = lepas).
func (h *MetaAdsHandler) MapCampaign(c *fiber.Ctx) error {
	var req struct {
		SourceID int64 `json:"source_id"`
	}
	if c.BodyParser(&req) != nil {
		h.store.Flash(c, "error", "data tidak valid")
		return h.back(c)
	}
	if err := h.ads.SetCampaignSource(c.UserContext(), c.Params("cid"), req.SourceID); err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", "Kampanye dihubungkan ke sumber lead")
	}
	return h.back(c)
}
