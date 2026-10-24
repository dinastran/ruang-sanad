package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// LeadFollowupHandler serves the follow-up approval queue (/app/leads/followups),
// manual follow-ups on a lead, and "Coba lagi" for failed sends (PRD 8.3, 10).
type LeadFollowupHandler struct {
	leads          *services.LeadService
	outbox         *services.WAOutboxService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLeadFollowupHandler(leads *services.LeadService, outbox *services.WAOutboxService, store *session.Store, inertiaService *services.InertiaService) *LeadFollowupHandler {
	return &LeadFollowupHandler{leads: leads, outbox: outbox, store: store, inertiaService: inertiaService}
}

const followupQueuePath = "/app/leads/followups"

func followupIDParam(c *fiber.Ctx) (int64, bool) {
	id, err := strconv.ParseInt(c.Params("fid"), 10, 64)
	return id, err == nil && id > 0
}

func (h *LeadFollowupHandler) done(c *fiber.Ctx, back string, err error, success string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else if success != "" {
		h.store.Flash(c, "success", success)
	}
	return h.inertiaService.Redirect(c, back)
}

// back returns to the page the action came from (queue or lead detail).
func (h *LeadFollowupHandler) back(c *fiber.Ctx) string {
	if lead, err := strconv.ParseInt(c.Query("lead"), 10, 64); err == nil && lead > 0 {
		return leadPath(lead)
	}
	return followupQueuePath
}

func (h *LeadFollowupHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	role, _ := c.Locals("role").(string)
	page, err := h.leads.FollowupQueue(c.UserContext(), role)
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/LeadFollowups", fiber.Map{
		"user": sessionUser(sess),
		"page": page,
	})
}

// Create is "Buat follow-up manual" on /app/leads/:id.
func (h *LeadFollowupHandler) Create(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadFollowupRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.done(c, leadPath(id), errors.New("data follow-up tidak valid"), "")
	}
	_, err := h.leads.CreateManualFollowup(c.UserContext(), id, c.Locals("user_id").(int64), req)
	return h.done(c, leadPath(id), err, "Follow-up dibuat dan menunggu persetujuan")
}

func (h *LeadFollowupHandler) Approve(c *fiber.Ctx) error {
	id, ok := followupIDParam(c)
	var req models.LeadFollowupRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.done(c, h.back(c), errors.New("data tidak valid"), "")
	}
	return h.done(c, h.back(c), h.leads.ApproveFollowup(c.UserContext(), id, c.Locals("user_id").(int64), req), "Follow-up disetujui")
}

func (h *LeadFollowupHandler) BulkApprove(c *fiber.Ctx) error {
	var req models.FollowupBulkRequest
	if c.BodyParser(&req) != nil {
		return h.done(c, followupQueuePath, errors.New("data tidak valid"), "")
	}
	n, skipped, err := h.leads.BulkApprove(c.UserContext(), req.IDs, c.Locals("user_id").(int64))
	if err != nil {
		return h.done(c, followupQueuePath, err, "")
	}
	if len(skipped) > 0 {
		h.store.Flash(c, "error", fmt.Sprintf("%d dilewati: %s", len(skipped), strings.Join(skipped, "; ")))
	}
	return h.done(c, followupQueuePath, nil, fmt.Sprintf("%d follow-up disetujui", n))
}

func (h *LeadFollowupHandler) Reschedule(c *fiber.Ctx) error {
	id, ok := followupIDParam(c)
	var req models.LeadFollowupRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.done(c, h.back(c), errors.New("data tidak valid"), "")
	}
	return h.done(c, h.back(c), h.leads.RescheduleFollowup(c.UserContext(), id, c.Locals("user_id").(int64), req.ScheduledAt), "Jadwal follow-up diubah")
}

func (h *LeadFollowupHandler) Cancel(c *fiber.Ctx) error {
	id, ok := followupIDParam(c)
	if !ok {
		return h.done(c, h.back(c), errors.New("data tidak valid"), "")
	}
	return h.done(c, h.back(c), h.leads.CancelFollowup(c.UserContext(), id, c.Locals("user_id").(int64)), "Follow-up dibatalkan")
}

// Retry is "Coba lagi" for a failed send.
func (h *LeadFollowupHandler) Retry(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("oid"), 10, 64)
	if err != nil || id <= 0 {
		return h.done(c, followupQueuePath, errors.New("data tidak valid"), "")
	}
	return h.done(c, h.back(c), h.outbox.Retry(c.UserContext(), id, c.Locals("user_id").(int64)), "Dimasukkan lagi ke antrean kirim")
}
