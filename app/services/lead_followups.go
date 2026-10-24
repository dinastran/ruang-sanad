package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Follow-up approval (PRD 8.3, 10): AI or CS proposes, CS/super_admin approves,
// the WAOutboxService worker sends. Every action is recorded in lead_events.

// FollowupApproverRoles may approve, edit, reschedule or cancel follow-ups (PRD 8.6).
var FollowupApproverRoles = []string{"cs", "super_admin"}

const followupQueueLimit = 200

// CanApproveFollowup reports whether the role may approve follow-ups.
func CanApproveFollowup(role string) bool { return inList(role, FollowupApproverRoles) }

// parseWIBLocal accepts "YYYY-MM-DDTHH:MM" (datetime-local) or "YYYY-MM-DD HH:MM" in WIB.
func parseWIBLocal(v string) (time.Time, error) {
	v = strings.Replace(strings.TrimSpace(v), "T", " ", 1)
	if len(v) > 16 {
		v = v[:16]
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", v, wib)
	if err != nil {
		return time.Time{}, leadInputErr("jadwal harus berformat tanggal dan jam (WIB)")
	}
	return t, nil
}

func (s *LeadService) loadFollowup(ctx context.Context, q *queries.Querier, id int64) (queries.LeadFollowup, error) {
	f, err := q.GetLeadFollowupByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return f, leadInputErr("follow-up tidak ditemukan")
	}
	return f, err
}

func (s *LeadService) queueItem(ctx context.Context, r queries.FollowupQueueRow, withMessages bool) (models.FollowupQueueItem, error) {
	item := models.FollowupQueueItem{
		ID: r.ID, LeadID: r.LeadID, LeadName: leadDisplayName(queries.Lead{ID: r.LeadID, Nama: r.LeadNama, WaName: r.LeadWaName}),
		LeadStage: r.LeadStage, Step: r.Step, ScheduledAt: formatLeadTime(r.ScheduledAt), DraftText: r.DraftText, Reason: r.Reason,
		Status: r.Status, CreatedByKind: r.CreatedByKind, Error: r.Error, OutboxStatus: r.OutboxStatus,
		Warnings:       FollowupDraftWarnings(r.Step, r.DraftText, r.Reason, r.FollowupPermission),
		RecentMessages: []models.LeadMessageResponse{},
	}
	if r.LeadStatus == LeadStatusDoNotContact && r.FollowupPermission != "no" {
		item.Warnings = append(item.Warnings, "Lead berstatus jangan hubungi (diblok)")
	}
	item.Blocked = r.FollowupPermission == "no" || r.LeadStatus != LeadStatusOpen
	if r.LeadStatus != LeadStatusOpen && r.LeadStatus != LeadStatusDoNotContact {
		item.Warnings = append(item.Warnings, "Lead tidak lagi berjalan ("+r.LeadStatus+")")
	}
	if withMessages {
		msgs, err := s.querier.ListLeadMessagesLatest(ctx, queries.ListLeadMessagesLatestParams{LeadID: r.LeadID, Limit: 3})
		if err != nil {
			return item, err
		}
		item.RecentMessages = messageResponses(msgs)
	}
	return item, nil
}

// FollowupQueue loads /app/leads/followups.
func (s *LeadService) FollowupQueue(ctx context.Context, role string) (*models.FollowupQueuePage, error) {
	settings, err := s.querier.GetCRMSettings(ctx)
	if err != nil {
		return nil, err
	}
	page := &models.FollowupQueuePage{
		Pending: []models.FollowupQueueItem{}, Approved: []models.FollowupQueueItem{}, Failed: []models.FailedSendItem{},
		FollowupEnabled: settings.FollowupEnabled == 1, SendStart: settings.FollowupSendStart, SendEnd: settings.FollowupSendEnd,
		DailyCap: settings.FollowupDailyCap, OnlyLabel: settings.FollowupOnlyLabel, CanApprove: inList(role, FollowupApproverRoles),
	}
	dayStart, dayEnd := wibDayBounds(s.nowUTC())
	if page.SentToday, err = s.querier.CountWaOutboxSentBetween(ctx, OutboxPurposeFollowup, dayStart, dayEnd); err != nil {
		return nil, err
	}
	pending, err := s.querier.ListFollowupsByStatus(ctx, "pending_approval", followupQueueLimit)
	if err != nil {
		return nil, err
	}
	for _, r := range pending {
		item, err := s.queueItem(ctx, r, true)
		if err != nil {
			return nil, err
		}
		page.Pending = append(page.Pending, item)
	}
	approved, err := s.querier.ListFollowupsByStatus(ctx, "approved", followupQueueLimit)
	if err != nil {
		return nil, err
	}
	for _, r := range approved {
		if r.OutboxStatus == "failed" {
			continue // listed under "Gagal kirim"
		}
		item, err := s.queueItem(ctx, r, false)
		if err != nil {
			return nil, err
		}
		page.Approved = append(page.Approved, item)
	}
	failed, err := s.querier.ListFailedWaOutbox(ctx, 100)
	if err != nil {
		return nil, err
	}
	for _, r := range failed {
		page.Failed = append(page.Failed, models.FailedSendItem{
			OutboxID: r.ID, LeadID: r.LeadID, LeadName: leadDisplayName(queries.Lead{ID: r.LeadID, Nama: r.LeadNama, WaName: r.LeadWaName}),
			Purpose: r.Purpose, Step: r.Step, Body: r.Body, Error: r.LastError, Attempts: r.Attempts, UpdatedAt: formatLeadTime(r.UpdatedAt),
		})
	}
	return page, nil
}

// CreateManualFollowup is "Buat follow-up manual" on the lead page. It waits
// for approval like an AI proposal.
func (s *LeadService) CreateManualFollowup(ctx context.Context, leadID, actorID int64, req models.LeadFollowupRequest) (int64, error) {
	at, err := parseWIBLocal(req.ScheduledAt)
	if err != nil {
		return 0, err
	}
	step := strings.TrimSpace(req.Step)
	if step == "" {
		step = FollowupStepManual
	}
	return s.ProposeFollowup(ctx, leadID, LeadFollowupInput{Step: step, ScheduledAt: at, DraftText: req.DraftText, Reason: req.Reason}, "user", actorID)
}

// ApproveFollowup approves a pending follow-up, optionally with an edited
// draft and schedule. The PRD 10.2 rules are checked again.
func (s *LeadService) ApproveFollowup(ctx context.Context, id, actorID int64, req models.LeadFollowupRequest) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		f, err := s.loadFollowup(ctx, q, id)
		if err != nil {
			return err
		}
		if f.Status != "pending_approval" {
			return leadInputErr("follow-up ini sudah tidak menunggu persetujuan (status: %s)", f.Status)
		}
		draft, at, edited := f.DraftText, f.ScheduledAt, false
		if d := strings.TrimSpace(req.DraftText); d != "" && d != draft {
			draft, edited = d, true
		}
		if req.ScheduledAt != "" {
			t, err := parseWIBLocal(req.ScheduledAt)
			if err != nil {
				return err
			}
			if !t.Equal(at) {
				at, edited = t.UTC(), true
			}
		}
		now := s.nowUTC()
		if edited && at.Before(now.Add(-time.Minute)) {
			return leadInputErr("jadwal tidak boleh di masa lalu")
		}
		l, err := loadLead(ctx, q, f.LeadID)
		if err != nil {
			return err
		}
		block, err := checkFollowupRules(ctx, q, followupCheck{Lead: l, Step: f.Step, ScheduledAt: at, Draft: draft, Reason: f.Reason, ExcludeID: f.ID})
		if err != nil {
			return err
		}
		if block != nil {
			return block
		}
		n, err := q.ApproveLeadFollowup(ctx, queries.ApproveLeadFollowupParams{
			DraftText: draft, ScheduledAt: at, ApprovedBy: nullInt(actorID), ApprovedAt: nullTime(now), UpdatedAt: now, ID: f.ID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return leadInputErr("follow-up ini baru saja diubah orang lain; muat ulang halaman")
		}
		return writeLeadEvent(ctx, q, f.LeadID, newLeadEventID("usr"), "followup.approved", userActor(actorID),
			map[string]any{"followup_id": f.ID, "step": f.Step, "scheduled_at": formatWIB(at), "edited": edited}, now)
	})
}

// BulkApprove approves the selected follow-ups that have no warning. Items
// with a warning or a rule violation are left pending and reported.
func (s *LeadService) BulkApprove(ctx context.Context, ids []int64, actorID int64) (int, []string, error) {
	if len(ids) == 0 || len(ids) > followupQueueLimit {
		return 0, nil, leadInputErr("pilih 1–%d follow-up", followupQueueLimit)
	}
	approved, skipped := 0, []string{}
	for _, id := range ids {
		f, err := s.loadFollowup(ctx, s.querier, id)
		if err != nil {
			skipped = append(skipped, err.Error())
			continue
		}
		l, err := loadLead(ctx, s.querier, f.LeadID)
		if err != nil {
			return approved, skipped, err
		}
		if w := FollowupDraftWarnings(f.Step, f.DraftText, f.Reason, l.FollowupPermission); len(w) > 0 {
			skipped = append(skipped, leadDisplayName(l)+": "+w[0])
			continue
		}
		err = s.ApproveFollowup(ctx, id, actorID, models.LeadFollowupRequest{})
		var le LeadInputError
		var fb *FollowupBlock
		switch {
		case err == nil:
			approved++
		case errors.As(err, &le), errors.As(err, &fb):
			skipped = append(skipped, leadDisplayName(l)+": "+err.Error())
		default:
			return approved, skipped, err
		}
	}
	return approved, skipped, nil
}

// RescheduleFollowup moves a pending or approved follow-up. An already queued
// send is cancelled and re-queued at the new time by the worker.
func (s *LeadService) RescheduleFollowup(ctx context.Context, id, actorID int64, scheduledAt string) error {
	at, err := parseWIBLocal(scheduledAt)
	if err != nil {
		return err
	}
	return s.withTx(ctx, func(q *queries.Querier) error {
		f, err := s.loadFollowup(ctx, q, id)
		if err != nil {
			return err
		}
		if f.Status != "pending_approval" && f.Status != "approved" {
			return leadInputErr("jadwal hanya bisa diubah untuk follow-up yang menunggu atau disetujui")
		}
		now := s.nowUTC()
		if at.Before(now.Add(-time.Minute)) {
			return leadInputErr("jadwal tidak boleh di masa lalu")
		}
		l, err := loadLead(ctx, q, f.LeadID)
		if err != nil {
			return err
		}
		block, err := checkFollowupRules(ctx, q, followupCheck{Lead: l, Step: f.Step, ScheduledAt: at, Draft: f.DraftText, Reason: f.Reason, ExcludeID: f.ID})
		if err != nil {
			return err
		}
		if block != nil {
			return block
		}
		if _, err := q.RescheduleLeadFollowup(ctx, queries.RescheduleLeadFollowupParams{ScheduledAt: at.UTC(), UpdatedAt: now, ID: f.ID}); err != nil {
			return err
		}
		if err := q.CancelLiveOutboxForFollowup(ctx, f.ID, "jadwal diubah", now); err != nil {
			return err
		}
		if err := q.SetLeadFollowupError(ctx, queries.SetLeadFollowupErrorParams{Error: "", UpdatedAt: now, ID: f.ID}); err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, f.LeadID, newLeadEventID("usr"), "followup.rescheduled", userActor(actorID),
			map[string]any{"followup_id": f.ID, "step": f.Step, "scheduled_at": formatWIB(at)}, now)
	})
}

// CancelFollowup cancels a pending or approved follow-up and its queued send.
func (s *LeadService) CancelFollowup(ctx context.Context, id, actorID int64) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		f, err := s.loadFollowup(ctx, q, id)
		if err != nil {
			return err
		}
		now := s.nowUTC()
		n, err := q.SetLeadFollowupStatus(ctx, queries.SetLeadFollowupStatusParams{Status: "cancelled", Error: "dibatalkan", UpdatedAt: now, ID: f.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return leadInputErr("follow-up ini sudah tidak bisa dibatalkan (status: %s)", f.Status)
		}
		if err := q.CancelLiveOutboxForFollowup(ctx, f.ID, "dibatalkan", now); err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, f.LeadID, newLeadEventID("usr"), "followup.cancelled", userActor(actorID),
			map[string]any{"followup_id": f.ID, "step": f.Step}, now)
	})
}
