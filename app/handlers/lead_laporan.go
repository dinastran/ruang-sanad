package handlers

import (
	"bytes"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// LeadLaporanHandler serves the CRM funnel report and its CSV export.
type LeadLaporanHandler struct {
	leads          *services.LeadService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLeadLaporanHandler(leads *services.LeadService, store *session.Store, inertiaService *services.InertiaService) *LeadLaporanHandler {
	return &LeadLaporanHandler{leads: leads, store: store, inertiaService: inertiaService}
}

func (h *LeadLaporanHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	report, err := h.leads.Report(c.UserContext(), c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/LeadLaporan", fiber.Map{
		"user":   sessionUser(sess),
		"report": report,
	})
}

func (h *LeadLaporanHandler) Export(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	var buf bytes.Buffer
	if err := h.leads.ExportCSV(c.UserContext(), &buf, c.Query("from"), c.Query("to"), role == "super_admin"); err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="laporan-leads-%s_%s.csv"`, sanitizeDate(c.Query("from")), sanitizeDate(c.Query("to"))))
	return c.Send(buf.Bytes())
}

func sanitizeDate(v string) string {
	out := make([]rune, 0, 10)
	for _, r := range v {
		if (r >= '0' && r <= '9') || r == '-' {
			out = append(out, r)
		}
		if len(out) == 10 {
			break
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return string(out)
}
