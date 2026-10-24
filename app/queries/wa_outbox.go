package queries

// Handwritten queries for the follow-up approval page and the WhatsApp send
// queue (wa_outbox): joins and MAX() lookups. Simple updates are in
// queries/leads.sql (sqlc).
import (
	"context"
	"database/sql"
	"time"
)

// EnqueueDueFollowups queues approved follow-ups whose time has come and that
// have no live outbox row yet. A cancelled row does not block re-queueing
// (used after a reschedule). Returns the number of rows queued.
func (q *Queries) EnqueueDueFollowups(ctx context.Context, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `INSERT INTO wa_outbox
	 (purpose, channel, to_chat_id, body, reference_type, reference_id, not_before, status, created_at, updated_at)
	 SELECT 'followup', 'bridge_cs', l.wa_chat_id, f.draft_text, 'lead_followup', f.id, f.scheduled_at, 'queued', ?, ?
	 FROM lead_followups f JOIN leads l ON l.id = f.lead_id
	 WHERE f.status = 'approved' AND f.scheduled_at <= ?
	   AND NOT EXISTS (SELECT 1 FROM wa_outbox o WHERE o.reference_type = 'lead_followup' AND o.reference_id = f.id
	                   AND o.status IN ('queued', 'sending', 'sent', 'failed'))`, now.UTC(), now.UTC(), now.UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

const waOutboxCols = `id, purpose, channel, to_chat_id, body, reference_type, reference_id, not_before, status,
 attempts, last_error, sent_at, wa_message_id, created_at, updated_at`

func scanWaOutbox(rows *sql.Rows) ([]WaOutbox, error) {
	defer rows.Close()
	out := make([]WaOutbox, 0)
	for rows.Next() {
		var o WaOutbox
		if err := rows.Scan(&o.ID, &o.Purpose, &o.Channel, &o.ToChatID, &o.Body, &o.ReferenceType, &o.ReferenceID, &o.NotBefore,
			&o.Status, &o.Attempts, &o.LastError, &o.SentAt, &o.WaMessageID, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListDueWaOutbox returns queued rows whose not_before has passed. Non
// follow-up sends (payment confirmations) come first so follow-ups waiting
// for the send window never hold them back; then oldest first.
func (q *Queries) ListDueWaOutbox(ctx context.Context, channel string, now time.Time, limit int64) ([]WaOutbox, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+waOutboxCols+` FROM wa_outbox
	 WHERE status = 'queued' AND channel = ? AND not_before <= ?
	 ORDER BY CASE WHEN purpose = 'followup' THEN 1 ELSE 0 END, not_before ASC, id ASC LIMIT ?`, channel, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	return scanWaOutbox(rows)
}

// LastWaOutboxSentAt is the latest send on a channel (global gap between sends).
func (q *Queries) LastWaOutboxSentAt(ctx context.Context, channel string) (sql.NullTime, error) {
	var at sql.NullTime
	err := q.db.QueryRowContext(ctx, `SELECT sent_at FROM wa_outbox WHERE channel = ? AND status = 'sent' AND sent_at IS NOT NULL
	 ORDER BY sent_at DESC LIMIT 1`, channel).Scan(&at)
	if err == sql.ErrNoRows {
		return sql.NullTime{}, nil
	}
	return at, err
}

// CountWaOutboxSentBetween counts sends of one purpose in [start, end).
func (q *Queries) CountWaOutboxSentBetween(ctx context.Context, purpose string, start, end time.Time) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_outbox WHERE purpose = ? AND status = 'sent' AND sent_at >= ? AND sent_at < ?`,
		purpose, start.UTC(), end.UTC()).Scan(&n)
	return n, err
}

// FollowupQueueRow is one follow-up with the lead fields the approval page needs.
type FollowupQueueRow struct {
	LeadFollowup
	LeadNama           string
	LeadWaName         string
	LeadStage          string
	LeadStatus         string
	FollowupPermission string
	OutboxStatus       string // latest outbox row for this follow-up ('' = none)
}

// ListFollowupsByStatus lists follow-ups of one status, earliest schedule first.
func (q *Queries) ListFollowupsByStatus(ctx context.Context, status string, limit int64) ([]FollowupQueueRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT f.id, f.lead_id, f.step, f.scheduled_at, f.draft_text, f.reason, f.status,
	 f.created_by_kind, f.created_by, f.approved_by, f.approved_at, f.sent_at, f.wa_message_id, f.error, f.created_at, f.updated_at,
	 l.nama, l.wa_name, l.stage, l.status, l.followup_permission,
	 COALESCE((SELECT o.status FROM wa_outbox o WHERE o.reference_type = 'lead_followup' AND o.reference_id = f.id ORDER BY o.id DESC LIMIT 1), '')
	 FROM lead_followups f JOIN leads l ON l.id = f.lead_id
	 WHERE f.status = ? ORDER BY f.scheduled_at ASC, f.id ASC LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]FollowupQueueRow, 0)
	for rows.Next() {
		var r FollowupQueueRow
		f := &r.LeadFollowup
		if err := rows.Scan(&f.ID, &f.LeadID, &f.Step, &f.ScheduledAt, &f.DraftText, &f.Reason, &f.Status,
			&f.CreatedByKind, &f.CreatedBy, &f.ApprovedBy, &f.ApprovedAt, &f.SentAt, &f.WaMessageID, &f.Error, &f.CreatedAt, &f.UpdatedAt,
			&r.LeadNama, &r.LeadWaName, &r.LeadStage, &r.LeadStatus, &r.FollowupPermission, &r.OutboxStatus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// FailedOutboxRow is a failed send with its lead (follow-up or confirmation).
type FailedOutboxRow struct {
	WaOutbox
	LeadID     int64
	LeadNama   string
	LeadWaName string
	Step       string
}

// ListFailedWaOutbox lists failed sends that can still be retried: follow-ups
// that are still approved, and other purposes referencing a lead.
func (q *Queries) ListFailedWaOutbox(ctx context.Context, limit int64) ([]FailedOutboxRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT o.id, o.purpose, o.channel, o.to_chat_id, o.body, o.reference_type, o.reference_id, o.not_before,
	 o.status, o.attempts, o.last_error, o.sent_at, o.wa_message_id, o.created_at, o.updated_at,
	 l.id, l.nama, l.wa_name, COALESCE(f.step, '')
	 FROM wa_outbox o
	 LEFT JOIN lead_followups f ON o.reference_type = 'lead_followup' AND f.id = o.reference_id
	 JOIN leads l ON l.id = CASE WHEN o.reference_type = 'lead_followup' THEN f.lead_id ELSE o.reference_id END
	 WHERE o.status = 'failed' AND (o.reference_type <> 'lead_followup' OR f.status = 'approved')
	 ORDER BY o.updated_at DESC, o.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]FailedOutboxRow, 0)
	for rows.Next() {
		var r FailedOutboxRow
		o := &r.WaOutbox
		if err := rows.Scan(&o.ID, &o.Purpose, &o.Channel, &o.ToChatID, &o.Body, &o.ReferenceType, &o.ReferenceID, &o.NotBefore,
			&o.Status, &o.Attempts, &o.LastError, &o.SentAt, &o.WaMessageID, &o.CreatedAt, &o.UpdatedAt,
			&r.LeadID, &r.LeadNama, &r.LeadWaName, &r.Step); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CancelLiveOutboxForFollowup cancels queued/failed outbox rows of a follow-up
// (after a reschedule or cancellation). A row being sent is left alone.
func (q *Queries) CancelLiveOutboxForFollowup(ctx context.Context, followupID int64, reason string, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE wa_outbox SET status = 'cancelled', last_error = ?, updated_at = ?
	 WHERE reference_type = 'lead_followup' AND reference_id = ? AND status IN ('queued', 'failed')`, reason, now.UTC(), followupID)
	return err
}
