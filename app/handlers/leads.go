package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// LeadHandler serves the CRM lead list, lead detail and manual pipeline
// actions (/app/leads). Verification and conversion live in lead_closing.go.
type LeadHandler struct {
	leads          *services.LeadService
	ads            *services.MetaAdsService
	capi           *services.MetaCAPIService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLeadHandler(leads *services.LeadService, store *session.Store, inertiaService *services.InertiaService) *LeadHandler {
	return &LeadHandler{leads: leads, store: store, inertiaService: inertiaService}
}

// WithIntegrations enables optional read-only Meta status for the CRM home.
// No provider API calls or CAPI events are triggered by loading this page.
func (h *LeadHandler) WithIntegrations(ads *services.MetaAdsService, capi *services.MetaCAPIService) *LeadHandler {
	h.ads, h.capi = ads, capi
	return h
}

// /app/leads stays backward-compatible: links with list filters keep rendering
// the existing list; the empty URL opens the command center.
func leadOverviewRequested(c *fiber.Ctx) bool {
	if c.Query("view") == "list" { return false }
	for key := range c.Queries() {
		if key != "summary_days" && key != "view" { return false }
	}
	return true
}

func (h *LeadHandler) overview(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	days := 14
	switch c.Query("summary_days") {
	case "7": days = 7
	case "30": days = 30
	}
	loc := time.FixedZone("WIB", 7*60*60)
	now := time.Now().In(loc)
	last := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	from, to := last.AddDate(0, 0, 1-days).Format(time.DateOnly), last.Format(time.DateOnly)
	crm, err := h.leads.Dashboard(c.UserContext(), from, to)
	if err != nil { return err }
	props := fiber.Map{"user": sessionUser(sess), "dashboard": crm, "summary_days": days}
	role := roleOf(c)
	if h.ads != nil && (role == "cs" || role == "super_admin") {
		if report, err := h.ads.Report(c.UserContext(), from, to); err == nil {
			props["ads"] = report
		} else {
			props["ads_error"] = "Laporan iklan belum dapat dimuat."
		}
	}
	if h.capi != nil && role == "super_admin" {
		if status, err := h.capi.Status(c.UserContext()); err == nil {
			props["capi_status"] = fiber.Map{
				"enabled": status.Enabled, "test_mode": status.TestMode,
				"counts": status.Counts,
				"sources_without_url": len(status.SourcesWithoutURL),
				"has_default_url": status.EventSourceURL != "",
			}
			if analytics, err := h.capi.Analytics(c.UserContext(), from, to); err == nil {
				props["capi_analytics"] = analytics
			} else {
				props["capi_error"] = "Ringkasan event CAPI belum dapat dimuat."
			}
		} else {
			props["capi_error"] = "Status integrasi CAPI belum dapat dimuat."
		}
	}
	return h.inertiaService.Render(c, "app/LeadHome", props)
}

func leadIDParam(c *fiber.Ctx) (int64, bool) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

func roleOf(c *fiber.Ctx) string {
	role, _ := c.Locals("role").(string)
	return role
}

func leadPath(id int64) string { return fmt.Sprintf("/app/leads/%d", id) }

func (h *LeadHandler) Index(c *fiber.Ctx) error {
	if leadOverviewRequested(c) { return h.overview(c) }
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	var params models.LeadListParams
	_ = c.QueryParser(&params)
	params.UserID = c.Locals("user_id").(int64)
	list, err := h.leads.List(c.UserContext(), params)
	if err != nil {
		return err
	}
	labels, err := h.leads.Labels(c.UserContext())
	if err != nil {
		return err
	}
	sources, err := h.leads.SourceOptions(c.UserContext())
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/Leads", fiber.Map{
		"sources": sources,
		"user":    user,
		"leads":   list.Items,
		"next":    list.NextCursor,
		"total":   list.Total,
		"counts":  list.Counts,
		"labels":  labels,
		"stages":  services.LeadStages,
		"filters": params,
	})
}

func (h *LeadHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	id, ok := leadIDParam(c)
	if !ok {
		return h.inertiaService.Redirect(c, "/app/leads")
	}
	lead, err := h.leads.Detail(c.UserContext(), id)
	if errors.Is(err, services.ErrLeadNotFound) {
		h.store.Flash(c, "error", "Lead tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/leads")
	}
	if err != nil {
		return err
	}
	labels, err := h.leads.Labels(c.UserContext())
	if err != nil {
		return err
	}
	users, err := h.leads.AssignableUsers(c.UserContext())
	if err != nil {
		return err
	}
	props := fiber.Map{
		"user":         user,
		"lead":         lead,
		"labels":       labels,
		"users":        users,
		"stages":       services.LeadStages,
		"lost_reasons": services.LeadLostReasons,
		"followup": fiber.Map{
			"can_approve": services.CanApproveFollowup(roleOf(c)),
			"steps":       append(append([]string{}, services.FollowupSteps...), services.FollowupStepManual),
		},
	}
	if lead.Status == services.LeadStatusOpen && lead.Stage == services.LeadStageTerverifikasi && lead.SantriID == 0 {
		opts, err := h.leads.ConvertOptions(c.UserContext(), id)
		if err != nil {
			return err
		}
		props["convert"] = opts
	}
	return h.inertiaService.Render(c, "app/LeadDetail", props)
}

// Messages returns older transcript messages as JSON (infinite scroll up).
func (h *LeadHandler) Messages(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	before, err := strconv.ParseInt(c.Query("before"), 10, 64)
	if !ok || err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "parameter tidak valid"})
	}
	msgs, more, err := h.leads.OlderMessages(c.UserContext(), id, before)
	if errors.Is(err, services.ErrLeadNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pesan tidak ditemukan"})
	}
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"messages": msgs, "has_more": more})
}

// finish flashes the outcome of an action and returns to the lead page.
func (h *LeadHandler) finish(c *fiber.Ctx, id int64, err error, success string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", success)
	}
	// The kanban board moves cards without leaving the list.
	if c.Query("back") == "list" {
		return h.inertiaService.Back(c, "/app/leads")
	}
	return h.inertiaService.Redirect(c, leadPath(id))
}

func (h *LeadHandler) UpdateData(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadUpdateRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.finish(c, id, errors.New("data tidak valid"), "")
	}
	return h.finish(c, id, h.leads.UpdateData(c.UserContext(), id, c.Locals("user_id").(int64), req), "Data calon santri disimpan")
}

func (h *LeadHandler) Stage(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadStageRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.finish(c, id, errors.New("data tidak valid"), "")
	}
	eventType := ""
	if c.Query("from") == "ai" {
		eventType = "stage.ai_suggestion_applied"
	}
	return h.finish(c, id, h.leads.ChangeStage(c.UserContext(), id, c.Locals("user_id").(int64), req.Stage, eventType), "Tahap diperbarui")
}

func (h *LeadHandler) Status(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadStatusRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.finish(c, id, errors.New("data tidak valid"), "")
	}
	return h.finish(c, id, h.leads.ChangeStatus(c.UserContext(), id, c.Locals("user_id").(int64), req), "Status lead diperbarui")
}

func (h *LeadHandler) Assign(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadAssignRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.finish(c, id, errors.New("data tidak valid"), "")
	}
	role, _ := c.Locals("role").(string)
	return h.finish(c, id, h.leads.Assign(c.UserContext(), id, c.Locals("user_id").(int64), role, req.UserID), "Penanggung jawab diperbarui")
}

func (h *LeadHandler) AddLabel(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadLabelRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.finish(c, id, errors.New("data tidak valid"), "")
	}
	return h.finish(c, id, h.leads.SetLabel(c.UserContext(), id, c.Locals("user_id").(int64), req.LabelID, true), "Label ditambahkan")
}

func (h *LeadHandler) RemoveLabel(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	labelID, err := strconv.ParseInt(c.Params("labelID"), 10, 64)
	if !ok || err != nil {
		return h.finish(c, id, errors.New("data tidak valid"), "")
	}
	return h.finish(c, id, h.leads.SetLabel(c.UserContext(), id, c.Locals("user_id").(int64), labelID, false), "Label dihapus")
}

// Delete erases a lead and all its data (super_admin only).
func (h *LeadHandler) Delete(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	if !ok {
		return h.inertiaService.Redirect(c, "/app/leads")
	}
	if err := h.leads.Delete(c.UserContext(), id, c.Locals("user_id").(int64)); err != nil {
		return h.finish(c, id, err, "")
	}
	h.store.Flash(c, "success", "Data lead dihapus permanen")
	return h.inertiaService.Redirect(c, "/app/leads?status=all")
}
