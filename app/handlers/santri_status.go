package handlers

import (
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// SantriStatusHandler lets Admin Kelas change a santri's status (aktif, cuti,
// nonaktif) straight from the Data Santri list.
type SantriStatusHandler struct {
	santriStatusService *services.SantriStatusService
	store               *session.Store
	inertiaService      *services.InertiaService
}

func NewSantriStatusHandler(santriStatusService *services.SantriStatusService, store *session.Store, inertiaService *services.InertiaService) *SantriStatusHandler {
	return &SantriStatusHandler{
		santriStatusService: santriStatusService,
		store:               store,
		inertiaService:      inertiaService,
	}
}

// santriListReturnURL keeps the Data Santri filters and page after a change,
// accepting only a relative /app/santri URL.
func santriListReturnURL(c *fiber.Ctx) string {
	parsed, err := url.Parse(c.Query("return_to", ""))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path != "/app/santri" || parsed.Fragment != "" || parsed.RawQuery == "" {
		return "/app/santri"
	}
	return "/app/santri?" + parsed.RawQuery
}

func (h *SantriStatusHandler) Update(c *fiber.Ctx) error {
	returnURL := santriListReturnURL(c)
	santriID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		h.store.Flash(c, "error", "Santri tidak valid")
		return h.inertiaService.Redirect(c, returnURL)
	}
	var req models.UpdateSantriStatusRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data status tidak valid")
		return h.inertiaService.Redirect(c, returnURL)
	}
	sess, _ := h.store.Get(c)
	if err := h.santriStatusService.UbahStatusSantri(santriID, sessionUser(sess).ID, req); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, returnURL)
	}
	h.store.Flash(c, "success", "Status santri berhasil diubah")
	return h.inertiaService.Redirect(c, returnURL)
}
