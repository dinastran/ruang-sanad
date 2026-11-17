package queries

// Handwritten queries for the follow-up history tab (/app/leads/followups?tab=riwayat):
// every follow-up with its latest send attempt, and the CRM events that make
// up one follow-up's timeline.
import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// followupHistoryStatus is the status shown in history: an approved follow-up
// whose latest send failed is reported as "failed".
const followupHistoryStatus = `CASE WHEN f.status = 'approved' AND COALESCE(o.status, '') = 'failed' THEN 'failed' ELSE f.status END`

// followupHistoryFrom joins each follow-up to its lead and latest outbox row.
const followupHistoryFrom = ` FROM lead_followups f
 JOIN leads l ON l.id = f.lead_id
 LEFT JOIN wa_outbox o ON o.id = (SELECT MAX(o2.id) FROM wa_outbox o2 WHERE o2.reference_type = 'lead_followup' AND o2.reference_id = f.id)`

// followupHistoryAt is the moment a follow-up last changed: sent time if sent.
const followupHistoryAt = `COALESCE(f.sent_at, f.updated_at)`

type FollowupHistoryFilter struct {
	Status string // derived status; empty = all
	Step   string
	From   time.Time // inclusive, zero = open
	To     time.Time // exclusive, zero = open
	Limit  int64
	Offset int64
}

type FollowupHistoryRow struct {
	ID            int64
	LeadID        int64
	LeadNama      string
	LeadWaName    string
	Step          string
	Status        string
	ScheduledAt   time.Time
	DraftText     string
	Reason        string
	CreatedByKind string
	Error         string
	SentAt        sql.NullTime
	CreatedAt     time.Time
	UpdatedAt     time.Time
	OutboxID      sql.NullInt64
	OutboxStatus  string
	Attempts      int64
	LastError     string
}

func followupHistoryWhere(f FollowupHistoryFilter, withStatus bool) (string, []any) {
	var conds []string
	var args []any
	if withStatus && f.Status != "" {
		conds = append(conds, followupHistoryStatus+" = ?")
		args = append(args, f.Status)
	}
	if f.Step != "" {
		conds = append(conds, "f.step = ?")
		args = append(args, f.Step)
	}
	if !f.From.IsZero() {
		conds = append(conds, followupHistoryAt+" >= ?")
		args = append(args, f.From.UTC())
	}
	if !f.To.IsZero() {
		conds = append(conds, followupHistoryAt+" < ?")
		args = append(args, f.To.UTC())
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListFollowupHistory lists follow-ups, most recently changed first.
func (q *Queries) ListFollowupHistory(ctx context.Context, f FollowupHistoryFilter) ([]FollowupHistoryRow, error) {
	where, args := followupHistoryWhere(f, true)
	args = append(args, f.Limit, f.Offset)
	rows, err := q.db.QueryContext(ctx, `SELECT f.id, f.lead_id, l.nama, l.wa_name, f.step, `+followupHistoryStatus+`,
	 f.scheduled_at, f.draft_text, f.reason, f.created_by_kind, f.error, f.sent_at, f.created_at, f.updated_at,
	 o.id, COALESCE(o.status, ''), COALESCE(o.attempts, 0), COALESCE(o.last_error, '')`+
		followupHistoryFrom+where+` ORDER BY `+followupHistoryAt+` DESC, f.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]FollowupHistoryRow, 0)
	for rows.Next() {
		var r FollowupHistoryRow
		if err := rows.Scan(&r.ID, &r.LeadID, &r.LeadNama, &r.LeadWaName, &r.Step, &r.Status,
			&r.ScheduledAt, &r.DraftText, &r.Reason, &r.CreatedByKind, &r.Error, &r.SentAt, &r.CreatedAt, &r.UpdatedAt,
			&r.OutboxID, &r.OutboxStatus, &r.Attempts, &r.LastError); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountFollowupHistoryByStatus counts follow-ups per derived status for the
// same step/date filter (the status filter itself is ignored).
func (q *Queries) CountFollowupHistoryByStatus(ctx context.Context, f FollowupHistoryFilter) (map[string]int64, error) {
	where, args := followupHistoryWhere(f, false)
	rows, err := q.db.QueryContext(ctx, `SELECT `+followupHistoryStatus+` AS st, COUNT(*)`+followupHistoryFrom+where+` GROUP BY st`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var st string
		var n int64
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}

// NextFollowupCreatedAt is when the lead's next follow-up after this one was
// created; it bounds which send events belong to this follow-up.
func (q *Queries) NextFollowupCreatedAt(ctx context.Context, leadID, followupID int64) (sql.NullTime, error) {
	var at sql.NullTime
	err := q.db.QueryRowContext(ctx, `SELECT MIN(created_at) FROM lead_followups WHERE lead_id = ? AND id > ?`, leadID, followupID).Scan(&at)
	return at, err
}

// ListLeadFollowupEvents returns a lead's follow-up and send events from a
// moment on, oldest first.
func (q *Queries) ListLeadFollowupEvents(ctx context.Context, leadID int64, since time.Time) ([]LeadEvent, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, lead_id, event_id, type, actor, payload, created_at FROM lead_events
	 WHERE lead_id = ? AND created_at >= ? AND (type LIKE 'followup.%' OR type LIKE 'outbox.%')
	 ORDER BY created_at, id LIMIT 200`, leadID, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadEvent, 0)
	for rows.Next() {
		var e LeadEvent
		if err := rows.Scan(&e.ID, &e.LeadID, &e.EventID, &e.Type, &e.Actor, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
