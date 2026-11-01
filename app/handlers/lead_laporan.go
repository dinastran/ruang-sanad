package handlers

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// LeadLaporanHandler serves the CRM funnel report and its CSV export.
type LeadLaporanHandler struct {
	leads          *services.LeadService
	ads            *services.MetaAdsService
	capi           *services.MetaCAPIService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLeadLaporanHandler(leads *services.LeadService, store *session.Store, inertiaService *services.InertiaService) *LeadLaporanHandler {
	return &LeadLaporanHandler{leads: leads, store: store, inertiaService: inertiaService}
}

// WithIntegrations attaches optional read-only Meta reports for authorized CRM roles.
func (h *LeadLaporanHandler) WithIntegrations(ads *services.MetaAdsService, capi *services.MetaCAPIService) *LeadLaporanHandler {
 h.ads, h.capi = ads, capi
 return h
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

 
// Dashboard renders CRM metrics, optionally augmented with Meta data.
// Ads spend is restricted to the roles permitted on /leads/iklan.
// The CAPI diagnostics remain super_admin only.
func (h *LeadLaporanHandler) Dashboard(c *fiber.Ctx) error {
 sess, _ := h.store.Get(c)
 dashboard, err := h.leads.Dashboard(c.UserContext(), c.Query("from"), c.Query("to"))
 if err != nil {
  return c.Status(fiber.StatusBadRequest).SendString(err.Error())
 }
 props := fiber.Map{"user": sessionUser(sess), "dashboard": dashboard}
 role := roleOf(c)
 if h.ads != nil && (role == "cs" || role == "super_admin") {
  if report, err := h.ads.Report(c.UserContext(), dashboard.From, dashboard.To); err == nil {
   props["ads"] = report
  } else {
   props["ads_error"] = "Data iklan tidak dapat dimuat. Ringkasan CRM tetap tersedia."
  }
 }
 if h.capi != nil && role == "super_admin" {
  if status, err := h.capi.Status(c.UserContext()); err == nil {
   props["capi"] = fiber.Map{
    "enabled": status.Enabled, "test_mode": status.TestMode,
    "counts": status.Counts, "sources_without_url": len(status.SourcesWithoutURL),
    "has_default_url": status.EventSourceURL != "",
   }
  } else {
   props["capi_error"] = "Status CAPI tidak dapat dimuat."
  }
 }
 return h.inertiaService.Render(c, "app/LeadDashboard", props)
}

// Priorities is the complete, paginated action queue, independent of acquisition dates.
func (h *LeadLaporanHandler) Priorities(c *fiber.Ctx) error {
 sess, _ := h.store.Get(c)
 q := c.Query("q")
 reason := c.Query("reason")
 page, err := strconv.Atoi(c.Query("page", "1"))
 if err != nil || page < 1 { page = 1 }
 if page > 100000 { page = 100000 }
 result, err := h.leads.PriorityPage(c.UserContext(), q, reason, page)
 if err != nil {
  return c.Status(fiber.StatusBadRequest).SendString(err.Error())
 }
 return h.inertiaService.Render(c, "app/LeadPriorities", fiber.Map{
  "user": sessionUser(sess), "queue": result,
 })
}
