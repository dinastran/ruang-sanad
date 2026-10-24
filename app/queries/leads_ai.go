package queries

// Handwritten queries for the CRM tools of the AI agent (MCP) and for the
// follow-up rules (PRD 10.2): dynamic filters and MAX() lookups that sqlc
// types poorly on SQLite. Simple inserts live in queries/leads.sql (sqlc).
import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// LeadAIFilter narrows the daftar_lead tool. Empty fields are not applied.
type LeadAIFilter struct {
	Status        string
	Stages        []string
	UpdatedSince  time.Time
	NeedsAnalysis bool
	BeforeID      int64 // keyset cursor: rows with id < BeforeID
	Limit         int64
}

type LeadAIRow struct {
	ID                    int64
	Nama                  string
	WaName                string
	Stage                 string
	Status                string
	PaketKode             string
	LastInboundAt         sql.NullTime
	LastOutboundAt        sql.NullTime
	LastAnalyzedMessageID int64
	NewMessages           int64
}

// ListLeadsForAI returns leads newest first (by id) with the number of
// messages stored after the last analysed message.
func (q *Queries) ListLeadsForAI(ctx context.Context, f LeadAIFilter) ([]LeadAIRow, error) {
	where := []string{"1=1"}
	args := []any{}
	if f.Status != "" {
		where = append(where, "l.status = ?")
		args = append(args, f.Status)
	}
	if len(f.Stages) > 0 {
		where = append(where, "l.stage IN (?"+strings.Repeat(",?", len(f.Stages)-1)+")")
		for _, s := range f.Stages {
			args = append(args, s)
		}
	}
	if !f.UpdatedSince.IsZero() {
		where = append(where, "l.updated_at >= ?")
		args = append(args, f.UpdatedSince.UTC())
	}
	if f.NeedsAnalysis {
		where = append(where, "EXISTS (SELECT 1 FROM lead_messages m WHERE m.lead_id = l.id AND m.id > l.last_analyzed_message_id)")
	}
	if f.BeforeID > 0 {
		where = append(where, "l.id < ?")
		args = append(args, f.BeforeID)
	}
	args = append(args, f.Limit)
	rows, err := q.db.QueryContext(ctx, `SELECT l.id, l.nama, l.wa_name, l.stage, l.status, l.paket_kode,
	 l.last_inbound_at, l.last_outbound_at, l.last_analyzed_message_id,
	 (SELECT COUNT(*) FROM lead_messages m WHERE m.lead_id = l.id AND m.id > l.last_analyzed_message_id)
	 FROM leads l WHERE `+strings.Join(where, " AND ")+` ORDER BY l.id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadAIRow, 0)
	for rows.Next() {
		var r LeadAIRow
		if err := rows.Scan(&r.ID, &r.Nama, &r.WaName, &r.Stage, &r.Status, &r.PaketKode,
			&r.LastInboundAt, &r.LastOutboundAt, &r.LastAnalyzedMessageID, &r.NewMessages); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanLeadMessages(rows *sql.Rows) ([]LeadMessage, error) {
	defer rows.Close()
	out := make([]LeadMessage, 0)
	for rows.Next() {
		var m LeadMessage
		if err := rows.Scan(&m.ID, &m.LeadID, &m.WaMessageID, &m.Direction, &m.Sender, &m.Body, &m.MediaType, &m.SentAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

const leadMessageCols = `id, lead_id, wa_message_id, direction, sender, body, media_type, sent_at, created_at`

// ListLeadMessagesAfterID returns up to limit messages with id > afterID,
// oldest first.
func (q *Queries) ListLeadMessagesAfterID(ctx context.Context, leadID, afterID, limit int64) ([]LeadMessage, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+leadMessageCols+` FROM lead_messages
	 WHERE lead_id = ? AND id > ? ORDER BY id ASC LIMIT ?`, leadID, afterID, limit)
	if err != nil {
		return nil, err
	}
	return scanLeadMessages(rows)
}

// ListLeadMessagesLatestByID returns the last limit messages, oldest first.
func (q *Queries) ListLeadMessagesLatestByID(ctx context.Context, leadID, limit int64) ([]LeadMessage, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+leadMessageCols+` FROM (
	 SELECT * FROM lead_messages WHERE lead_id = ? ORDER BY id DESC LIMIT ?) ORDER BY id ASC`, leadID, limit)
	if err != nil {
		return nil, err
	}
	return scanLeadMessages(rows)
}

// LeadMaxMessageID returns the highest message id of a lead (0 if none).
func (q *Queries) LeadMaxMessageID(ctx context.Context, leadID int64) (int64, error) {
	var id int64
	err := q.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM lead_messages WHERE lead_id = ?`, leadID).Scan(&id)
	return id, err
}

// LeadLastMeaningfulOutbound is the time of the last outgoing message that is
// not itself a follow-up (the anchor of the Script 30 Hari sequence).
func (q *Queries) LeadLastMeaningfulOutbound(ctx context.Context, leadID int64) (sql.NullTime, error) {
	var at sql.NullTime
	row := q.db.QueryRowContext(ctx, `SELECT sent_at FROM lead_messages
	 WHERE lead_id = ? AND direction = 'out' AND sender <> 'followup' ORDER BY sent_at DESC, id DESC LIMIT 1`, leadID)
	err := row.Scan(&at)
	if err == sql.ErrNoRows {
		return sql.NullTime{}, nil
	}
	return at, err
}

// CountLeadFollowupsSentBetween counts follow-ups of a lead sent in [start, end).
func (q *Queries) CountLeadFollowupsSentBetween(ctx context.Context, leadID, excludeID int64, start, end time.Time) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lead_followups
	 WHERE lead_id = ? AND id <> ? AND status = 'sent' AND sent_at >= ? AND sent_at < ?`,
		leadID, excludeID, start.UTC(), end.UTC()).Scan(&n)
	return n, err
}

// CountLeadFollowupsPlannedBetween counts approved (and, when withPending,
// pending_approval) follow-ups of a lead scheduled in [start, end).
func (q *Queries) CountLeadFollowupsPlannedBetween(ctx context.Context, leadID, excludeID int64, start, end time.Time, withPending bool) (int64, error) {
	statuses := "'approved'"
	if withPending {
		statuses = "'approved', 'pending_approval'"
	}
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lead_followups
	 WHERE lead_id = ? AND id <> ? AND status IN (`+statuses+`) AND scheduled_at >= ? AND scheduled_at < ?`,
		leadID, excludeID, start.UTC(), end.UTC()).Scan(&n)
	return n, err
}

// LeadFollowupStepSentSince reports whether a follow-up of this step was sent
// at or after since.
func (q *Queries) LeadFollowupStepSentSince(ctx context.Context, leadID int64, step string, since time.Time) (bool, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lead_followups
	 WHERE lead_id = ? AND step = ? AND status = 'sent' AND sent_at >= ?`, leadID, step, since.UTC()).Scan(&n)
	return n > 0, err
}
