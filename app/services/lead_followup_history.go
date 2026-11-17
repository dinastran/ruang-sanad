package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const followupHistoryPageSize = 30

// FollowupHistoryStatuses are the statuses the history tab can filter on.
var FollowupHistoryStatuses = []string{"sent", "failed", "approved", "pending_approval", "skipped", "cancelled"}

func parseWIBDate(v string) (time.Time, bool) {
	t, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(v), wib)
	return t, err == nil
}

// FollowupHistory lists every follow-up with its outcome (history tab).
func (s *LeadService) FollowupHistory(ctx context.Context, in models.FollowupHistoryFilter) (*models.FollowupHistoryPage, error) {
	if !slices.Contains(FollowupHistoryStatuses, in.Status) {
		in.Status = ""
	}
	if in.Step != FollowupStepManual && !slices.Contains(FollowupSteps, in.Step) {
		in.Step = ""
	}
	if in.Page < 1 {
		in.Page = 1
	}
	f := queries.FollowupHistoryFilter{Status: in.Status, Step: in.Step, Limit: followupHistoryPageSize + 1, Offset: (in.Page - 1) * followupHistoryPageSize}
	if t, ok := parseWIBDate(in.From); ok {
		f.From = t
	} else {
		in.From = ""
	}
	if t, ok := parseWIBDate(in.To); ok {
		f.To = t.AddDate(0, 0, 1)
	} else {
		in.To = ""
	}
	rows, err := s.querier.ListFollowupHistory(ctx, f)
	if err != nil {
		return nil, err
	}
	counts, err := s.querier.CountFollowupHistoryByStatus(ctx, f)
	if err != nil {
		return nil, err
	}
	page := &models.FollowupHistoryPage{Items: []models.FollowupHistoryItem{}, Counts: counts, Filter: in, Page: in.Page}
	if len(rows) > followupHistoryPageSize {
		rows, page.HasMore = rows[:followupHistoryPageSize], true
	}
	for _, r := range rows {
		note := r.Error
		if r.Status == "failed" && r.LastError != "" {
			note = r.LastError
		}
		item := models.FollowupHistoryItem{
			ID: r.ID, LeadID: r.LeadID, LeadName: leadDisplayName(queries.Lead{ID: r.LeadID, Nama: r.LeadNama, WaName: r.LeadWaName}),
			Step: r.Step, Status: r.Status, ScheduledAt: formatLeadTime(r.ScheduledAt), DraftText: r.DraftText,
			CreatedByKind: r.CreatedByKind, Note: note, Attempts: r.Attempts, OutboxID: r.OutboxID.Int64,
			At: formatLeadTime(r.UpdatedAt),
		}
		if r.SentAt.Valid {
			item.SentAt = formatLeadTime(r.SentAt.Time)
			item.At = item.SentAt
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

// FollowupTimeline is the trail of one follow-up: proposed, approved, each
// send attempt and its outcome. Send events that predate followup_id in their
// payload are attributed by time (until the lead's next follow-up).
func (s *LeadService) FollowupTimeline(ctx context.Context, id int64) ([]models.FollowupTimelineEntry, error) {
	f, err := s.querier.GetLeadFollowupByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, leadInputErr("follow-up tidak ditemukan")
	}
	if err != nil {
		return nil, err
	}
	next, err := s.querier.NextFollowupCreatedAt(ctx, f.LeadID, f.ID)
	if err != nil {
		return nil, err
	}
	events, err := s.querier.ListLeadFollowupEvents(ctx, f.LeadID, f.CreatedAt.Add(-time.Minute))
	if err != nil {
		return nil, err
	}
	var mine []queries.LeadEvent
	for _, e := range events {
		var p map[string]any
		_ = json.Unmarshal([]byte(e.Payload), &p)
		if fid, ok := p["followup_id"].(float64); ok {
			if int64(fid) != f.ID {
				continue
			}
		} else {
			// Older outbox events carry no followup_id: only follow-up sends,
			// and only before the lead's next follow-up was created.
			if !strings.HasPrefix(e.Type, "outbox.") || p["purpose"] != OutboxPurposeFollowup {
				continue
			}
			if next.Valid && !e.CreatedAt.Before(next.Time) {
				continue
			}
		}
		mine = append(mine, e)
	}
	names := s.actorNames(ctx, mine)
	out := make([]models.FollowupTimelineEntry, 0, len(mine))
	for _, e := range mine {
		out = append(out, models.FollowupTimelineEntry{
			Type: e.Type, Actor: timelineActor(e.Actor, names), At: formatLeadTime(e.CreatedAt), Detail: timelineDetail(e),
		})
	}
	return out, nil
}

func timelineActor(actor string, names map[string]string) string {
	if n, ok := names[actor]; ok {
		return n
	}
	switch actor {
	case "ai":
		return "AI"
	case "system":
		return "Sistem"
	case "bridge":
		return "Bridge WA"
	}
	return actor
}

func timelineDetail(e queries.LeadEvent) string {
	var p map[string]any
	_ = json.Unmarshal([]byte(e.Payload), &p)
	str := func(k string) string { v, _ := p[k].(string); return v }
	switch e.Type {
	case "outbox.failed":
		d := str("reason")
		if n, ok := p["attempts"].(float64); ok {
			d = fmt.Sprintf("percobaan ke-%d: %s", int(n), d)
		}
		return d
	case "followup.skipped", "followup.cancelled":
		return str("reason")
	case "followup.proposed", "followup.rescheduled":
		if at := str("scheduled_at"); at != "" {
			return "jadwal " + at
		}
	case "followup.approved":
		d := ""
		if at := str("scheduled_at"); at != "" {
			d = "jadwal " + at
		}
		if v, _ := p["edited"].(bool); v {
			d = strings.TrimPrefix(d+" · pesan diedit sebelum disetujui", " · ")
		}
		return d
	case "followup.sent":
		return "pesan diterima server WhatsApp"
	}
	return ""
}
