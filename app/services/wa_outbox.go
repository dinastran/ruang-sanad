package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// WAOutboxService is the send worker for wa_outbox (PRD 10.1). One tick:
//  1. approved follow-ups whose time has come are queued (channel bridge_cs);
//  2. at most one queued message is sent through the bridge, after the global
//     switch, send window, daily cap, global gap and the PRD 10.2 rules pass.
//
// It runs in a single process; ClaimWaOutbox (queued → sending) guards
// against a double send.

const (
	OutboxChannelBridge    = "bridge_cs"
	OutboxPurposeFollowup  = "followup"
	OutboxPurposePaymentOK = "payment_verified"
	outboxMaxAttempts      = 3
	outboxStuckAfter       = 10 * time.Minute
)

// Waits after failed attempt 1 and 2; attempt 3 fails for good (PRD 11.3).
var outboxBackoff = []time.Duration{time.Minute, 5 * time.Minute}

type WAOutboxService struct {
	querier *queries.Querier
	bridge  OutboundSender
	now     func() time.Time
}

func NewWAOutboxService(q *queries.Querier, bridge OutboundSender) *WAOutboxService {
	return &WAOutboxService{querier: q, bridge: bridge, now: time.Now}
}

func (s *WAOutboxService) nowUTC() time.Time { return s.now().UTC().Truncate(time.Second) }

func (s *WAOutboxService) withTx(ctx context.Context, fn func(q *queries.Querier) error) error {
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.querier.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// sendWindow returns today's [start, end) send window in WIB.
func sendWindow(settings queries.CrmSetting, now time.Time) (time.Time, time.Time) {
	d := now.In(wib)
	parse := func(hm string, def int) time.Time {
		t, err := time.ParseInLocation("15:04", hm, wib)
		if err != nil {
			return time.Date(d.Year(), d.Month(), d.Day(), def, 0, 0, 0, wib)
		}
		return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, wib)
	}
	return parse(settings.FollowupSendStart, 8), parse(settings.FollowupSendEnd, 20)
}

// nextWindowStart is the next moment follow-ups may be sent, after the given day.
func nextWindowStart(settings queries.CrmSetting, now time.Time) time.Time {
	start, _ := sendWindow(settings, now)
	return start.AddDate(0, 0, 1)
}

// TickResult says what one run did (for logs and tests).
type TickResult struct {
	Queued int64
	Sent   int
	Failed int
	Note   string
}

// Tick runs one worker cycle.
func (s *WAOutboxService) Tick(ctx context.Context) (TickResult, error) {
	var res TickResult
	now := s.nowUTC()
	if _, err := s.querier.FailStuckWaOutbox(ctx, queries.FailStuckWaOutboxParams{
		Reason: "status kirim tidak pasti (server berhenti saat mengirim); cek WhatsApp CS sebelum Coba lagi",
		Now:    now, Before: now.Add(-outboxStuckAfter),
	}); err != nil {
		return res, err
	}
	settings, err := s.querier.GetCRMSettings(ctx)
	if err != nil {
		return res, err
	}
	if settings.FollowupEnabled == 1 {
		if res.Queued, err = s.querier.EnqueueDueFollowups(ctx, now); err != nil {
			return res, err
		}
	}
	due, err := s.querier.ListDueWaOutbox(ctx, OutboxChannelBridge, now, 50)
	if err != nil || len(due) == 0 {
		return res, err
	}
	last, err := s.querier.LastWaOutboxSentAt(ctx, OutboxChannelBridge)
	if err != nil {
		return res, err
	}
	if gap := time.Duration(settings.FollowupMinGapSeconds) * time.Second; last.Valid && now.Sub(last.Time) < gap {
		res.Note = "menunggu jeda minimum"
		return res, nil
	}
	for _, o := range due {
		if o.Purpose == OutboxPurposeFollowup {
			ok, note, err := s.followupMaySend(ctx, settings, o, now)
			if err != nil {
				return res, err
			}
			if !ok {
				res.Note = note
				continue
			}
		}
		sent, err := s.send(ctx, o, now)
		if err != nil {
			return res, err
		}
		if sent {
			res.Sent++
		} else {
			res.Failed++
		}
		return res, nil // one send attempt per tick keeps the gap
	}
	return res, nil
}

// followupMaySend applies the global switch, window, daily cap and, for the
// follow-up itself, the PRD 10.2 rules. A follow-up that must not be sent is
// resolved here (skipped / cancelled / postponed).
func (s *WAOutboxService) followupMaySend(ctx context.Context, settings queries.CrmSetting, o queries.WaOutbox, now time.Time) (bool, string, error) {
	if settings.FollowupEnabled != 1 {
		return false, "follow-up dimatikan", nil
	}
	start, end := sendWindow(settings, now)
	if now.Before(start) || !now.Before(end) {
		return false, "di luar jam kirim", nil
	}
	dayStart, dayEnd := wibDayBounds(now)
	sentToday, err := s.querier.CountWaOutboxSentBetween(ctx, OutboxPurposeFollowup, dayStart, dayEnd)
	if err != nil {
		return false, "", err
	}
	if sentToday >= settings.FollowupDailyCap {
		return false, "batas harian tercapai", nil
	}

	allowed := false
	err = s.withTx(ctx, func(q *queries.Querier) error {
		f, err := q.GetLeadFollowupByID(ctx, o.ReferenceID.Int64)
		if errors.Is(err, sql.ErrNoRows) {
			return q.CancelWaOutbox(ctx, queries.CancelWaOutboxParams{LastError: "follow-up sudah dihapus", UpdatedAt: now, ID: o.ID})
		}
		if err != nil {
			return err
		}
		if f.Status != "approved" {
			return q.CancelWaOutbox(ctx, queries.CancelWaOutboxParams{LastError: "follow-up tidak lagi berstatus disetujui", UpdatedAt: now, ID: o.ID})
		}
		lead, err := loadLead(ctx, q, f.LeadID)
		if err != nil {
			return err
		}
		var block *FollowupBlock
		if settings.FollowupOnlyLabel != "" {
			n, err := q.LeadHasLabelNamed(ctx, queries.LeadHasLabelNamedParams{LeadID: lead.ID, Nama: settings.FollowupOnlyLabel})
			if err != nil {
				return err
			}
			if n == 0 {
				block = blockFollowup(FollowupBlockSkip, "uji bertahap: follow-up hanya dikirim ke lead berlabel %q", settings.FollowupOnlyLabel)
			}
		}
		if block == nil {
			if block, err = checkFollowupRules(ctx, q, followupCheck{
				Lead: lead, Step: f.Step, ScheduledAt: now, Draft: f.DraftText, Reason: f.Reason, ExcludeID: f.ID, AtSend: true,
			}); err != nil {
				return err
			}
		}
		if block == nil {
			allowed = true
			return nil
		}
		if block.Postpone {
			return q.MarkWaOutboxRetry(ctx, queries.MarkWaOutboxRetryParams{NotBefore: nextWindowStart(settings, now).UTC(), LastError: block.Msg, UpdatedAt: now, ID: o.ID})
		}
		status := "skipped"
		if block.Kind == FollowupBlockCancel {
			status = "cancelled"
		}
		if _, err := q.SetLeadFollowupStatus(ctx, queries.SetLeadFollowupStatusParams{Status: status, Error: block.Msg, UpdatedAt: now, ID: f.ID}); err != nil {
			return err
		}
		if err := q.CancelWaOutbox(ctx, queries.CancelWaOutboxParams{LastError: block.Msg, UpdatedAt: now, ID: o.ID}); err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, lead.ID, newLeadEventID("sys"), "followup."+status, "system",
			map[string]any{"followup_id": f.ID, "step": f.Step, "reason": block.Msg}, now)
	})
	return allowed, "", err
}

// send claims one outbox row and posts it to the bridge. It reports whether
// the bridge accepted the message.
func (s *WAOutboxService) send(ctx context.Context, o queries.WaOutbox, now time.Time) (bool, error) {
	n, err := s.querier.ClaimWaOutbox(ctx, queries.ClaimWaOutboxParams{UpdatedAt: now, ID: o.ID})
	if err != nil || n == 0 {
		return false, err
	}
	req := OutboundRequest{ChatID: o.ToChatID, Text: o.Body, Purpose: o.Purpose}
	var f queries.LeadFollowup
	if o.Purpose == OutboxPurposeFollowup {
		if f, err = s.querier.GetLeadFollowupByID(ctx, o.ReferenceID.Int64); err != nil {
			return false, err
		}
		req.Text, req.ReferenceID = f.DraftText, f.ID // the approved text, also after edits
	}
	leadID := f.LeadID
	if o.Purpose != OutboxPurposeFollowup {
		leadID = o.ReferenceID.Int64
	}

	waID, sendErr := s.bridge.SendOutbound(ctx, req)
	done := s.nowUTC()
	if sendErr == nil {
		return true, s.withTx(ctx, func(q *queries.Querier) error {
			if err := q.MarkWaOutboxSent(ctx, queries.MarkWaOutboxSentParams{SentAt: nullTime(done), WaMessageID: waID, UpdatedAt: done, ID: o.ID}); err != nil {
				return err
			}
			typ := "payment_confirmation.sent"
			payload := map[string]any{"wa_message_id": waID}
			if o.Purpose == OutboxPurposeFollowup {
				typ = "followup.sent"
				payload["followup_id"], payload["step"] = f.ID, f.Step
				if _, err := q.MarkLeadFollowupSent(ctx, queries.MarkLeadFollowupSentParams{
					SentAt: nullTime(done), WaMessageID: waID, UpdatedAt: done, ID: f.ID, LeadID: f.LeadID,
				}); err != nil {
					return err
				}
			}
			return writeLeadEvent(ctx, q, leadID, newLeadEventID("sys"), typ, "system", payload, done)
		})
	}

	msg, permanent := outboundErrorInfo(sendErr)
	attempts := o.Attempts + 1
	slog.Warn("kirim WA lewat bridge gagal", "outbox_id", o.ID, "purpose", o.Purpose, "percobaan", attempts, "error", msg)
	return false, s.withTx(ctx, func(q *queries.Querier) error {
		if !permanent && attempts < outboxMaxAttempts {
			wait := outboxBackoff[min(int(attempts)-1, len(outboxBackoff)-1)]
			return q.MarkWaOutboxRetry(ctx, queries.MarkWaOutboxRetryParams{NotBefore: done.Add(wait), LastError: msg, UpdatedAt: done, ID: o.ID})
		}
		if err := q.MarkWaOutboxFailed(ctx, queries.MarkWaOutboxFailedParams{LastError: msg, UpdatedAt: done, ID: o.ID}); err != nil {
			return err
		}
		if o.Purpose == OutboxPurposeFollowup {
			if err := q.SetLeadFollowupError(ctx, queries.SetLeadFollowupErrorParams{Error: "Gagal kirim: " + msg, UpdatedAt: done, ID: f.ID}); err != nil {
				return err
			}
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("sys"), "outbox.failed", "system",
			map[string]any{"purpose": o.Purpose, "reason": msg, "attempts": attempts}, done)
	})
}

// Retry puts a failed send back in the queue ("Coba lagi").
func (s *WAOutboxService) Retry(ctx context.Context, outboxID, actorID int64) error {
	now := s.nowUTC()
	return s.withTx(ctx, func(q *queries.Querier) error {
		o, err := q.GetWaOutboxByID(ctx, outboxID)
		if errors.Is(err, sql.ErrNoRows) {
			return leadInputErr("kiriman tidak ditemukan")
		}
		if err != nil {
			return err
		}
		leadID := o.ReferenceID.Int64
		if o.Purpose == OutboxPurposeFollowup {
			f, err := q.GetLeadFollowupByID(ctx, o.ReferenceID.Int64)
			if err != nil {
				return err
			}
			if f.Status != "approved" {
				return leadInputErr("follow-up ini sudah tidak berstatus disetujui")
			}
			leadID = f.LeadID
			if err := q.SetLeadFollowupError(ctx, queries.SetLeadFollowupErrorParams{Error: "", UpdatedAt: now, ID: f.ID}); err != nil {
				return err
			}
		}
		n, err := q.RequeueFailedWaOutbox(ctx, queries.RequeueFailedWaOutboxParams{NotBefore: now, UpdatedAt: now, ID: outboxID})
		if err != nil {
			return err
		}
		if n == 0 {
			return leadInputErr("kiriman ini tidak berstatus gagal")
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("usr"), "outbox.retried", userActor(actorID), map[string]any{"purpose": o.Purpose}, now)
	})
}
