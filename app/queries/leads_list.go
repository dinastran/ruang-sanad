package queries

// Handwritten queries for the CRM lead list (dynamic filters + keyset cursor)
// and the funnel report. Simple CRUD lives in queries/leads.sql (sqlc).
import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// LeadListFilter narrows /app/leads. Empty fields are not applied.
type LeadListFilter struct {
	Status      string // "" = any
	Stage       string
	ReachedStage string
	LostReason string
	// SourceID > 0 filters one lead source; SourceNone keeps leads without one.
	SourceID   int64
	SourceNone bool
	LabelID     int64
	AssignedTo  int64
	NeedsReview bool
	Search      string
	PhoneSearch string
	DateField string // converted only when explicitly requested
	CreatedFrom time.Time
	CreatedTo   time.Time
	// Keyset cursor (exclusive): rows strictly after this position.
	AfterPrio int64
	AfterAct  string
	AfterID   int64
	HasCursor bool
	Limit     int64
}

type LeadListRow struct {
	ID             int64
	Nama           string
	WaName         string
	Phone          string
	Stage          string
	Status         string
	PaketKode      string
	NeedsReview    int64
	AssignedUserID sql.NullInt64
	AssignedName   string
	LastInboundAt  sql.NullTime
	ActivityRaw    string
	Prio           int64
	AIPaused       bool
}

const leadPrioExpr = `CASE WHEN l.stage = 'bukti_masuk' AND l.status = 'open' THEN 1 ELSE 0 END`
const leadActExpr = `CAST(COALESCE(l.last_inbound_at, l.created_at) AS TEXT)`

// leadListWhere builds the filter conditions shared by ListLeads and
// CountLeadsByColumn. The keyset cursor is not included.
func leadListWhere(f LeadListFilter) ([]string, []any) {
	where := []string{"1=1"}
	args := []any{}
	if f.Status != "" {
		where = append(where, "l.status = ?")
		args = append(args, f.Status)
	}
	if f.Stage != "" {
		where = append(where, "l.stage = ?")
		args = append(args, f.Stage)
	}
 if f.ReachedStage != "" {
  order := map[string]int{"baru":0,"discovery":1,"rekomendasi":2,"form_dikirim":3,"data_terisi":4,"info_bayar_dikirim":5,"bukti_masuk":6,"terverifikasi":7}
  if at, ok := order[f.ReachedStage]; ok {
   where = append(where, "(l.status = 'won' OR (CASE l.stage WHEN 'baru' THEN 0 WHEN 'discovery' THEN 1 WHEN 'rekomendasi' THEN 2 WHEN 'form_dikirim' THEN 3 WHEN 'data_terisi' THEN 4 WHEN 'info_bayar_dikirim' THEN 5 WHEN 'bukti_masuk' THEN 6 WHEN 'terverifikasi' THEN 7 ELSE -1 END) >= ?)")
   args = append(args, at)
  }
 }
 if f.SourceID > 0 {
		where = append(where, "l.lead_source_id = ?")
		args = append(args, f.SourceID)
	} else if f.SourceNone {
		where = append(where, "l.lead_source_id IS NULL")
	}
 if f.LostReason != "" {
  where = append(where, "l.status = 'lost' AND l.lost_reason = ?")
  args = append(args, f.LostReason)
 }
	if f.LabelID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM lead_label_links k WHERE k.lead_id = l.id AND k.label_id = ?)")
		args = append(args, f.LabelID)
	}
	if f.AssignedTo > 0 {
		where = append(where, "l.assigned_user_id = ?")
		args = append(args, f.AssignedTo)
	}
	if f.NeedsReview {
		where = append(where, "l.needs_review = 1")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		cond := "(l.nama LIKE ? COLLATE NOCASE OR l.wa_name LIKE ? COLLATE NOCASE"
		args = append(args, like, like)
		if f.PhoneSearch != "" {
			cond += " OR l.phone LIKE ?"
			args = append(args, "%"+f.PhoneSearch+"%")
		}
		where = append(where, cond+")")
	}
	dateColumn := "l.created_at"
 if f.DateField == "converted" { dateColumn = "l.converted_at" }
 if !f.CreatedFrom.IsZero() {
		where = append(where, dateColumn+" >= ?")
		args = append(args, f.CreatedFrom.UTC())
	}
	if !f.CreatedTo.IsZero() {
		where = append(where, dateColumn+" < ?")
		args = append(args, f.CreatedTo.UTC())
	}
	return where, args
}

func (q *Queries) ListLeads(ctx context.Context, f LeadListFilter) ([]LeadListRow, error) {
	where, args := leadListWhere(f)
	if f.HasCursor {
		where = append(where, "("+leadPrioExpr+" < ? OR ("+leadPrioExpr+" = ? AND "+leadActExpr+" < ?) OR ("+leadPrioExpr+" = ? AND "+leadActExpr+" = ? AND l.id < ?))")
		args = append(args, f.AfterPrio, f.AfterPrio, f.AfterAct, f.AfterPrio, f.AfterAct, f.AfterID)
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	args = append(args, limit)
	query := `SELECT l.id, l.nama, l.wa_name, l.phone, l.stage, l.status, l.paket_kode, l.needs_review,
		l.assigned_user_id, COALESCE(u.name, ''), l.last_inbound_at, ` + leadActExpr + `, ` + leadPrioExpr + `,
		COALESCE((SELECT e.type FROM lead_events e WHERE e.lead_id = l.id AND e.type IN ('ai.paused', 'ai.resumed') ORDER BY e.id DESC LIMIT 1), '') = 'ai.paused'
	FROM leads l LEFT JOIN users u ON u.id = l.assigned_user_id
	WHERE ` + strings.Join(where, " AND ") + `
	ORDER BY ` + leadPrioExpr + ` DESC, ` + leadActExpr + ` DESC, l.id DESC
	LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadListRow, 0)
	for rows.Next() {
		var r LeadListRow
		if err := rows.Scan(&r.ID, &r.Nama, &r.WaName, &r.Phone, &r.Stage, &r.Status, &r.PaketKode, &r.NeedsReview,
			&r.AssignedUserID, &r.AssignedName, &r.LastInboundAt, &r.ActivityRaw, &r.Prio, &r.AIPaused); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LeadColumnCount is how many leads match the list filter in one column:
// the current stage for open leads, otherwise the status (won, lost, ...).
type LeadColumnCount struct {
	Column string
	Count  int64
}

// CountLeadsByColumn counts every lead matching the filter (all pages).
func (q *Queries) CountLeadsByColumn(ctx context.Context, f LeadListFilter) ([]LeadColumnCount, error) {
	where, args := leadListWhere(f)
	rows, err := q.db.QueryContext(ctx, `SELECT CASE WHEN l.status = 'open' THEN l.stage ELSE l.status END AS col, COUNT(*)
	FROM leads l WHERE `+strings.Join(where, " AND ")+` GROUP BY col`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadColumnCount, 0)
	for rows.Next() {
		var r LeadColumnCount
		if err := rows.Scan(&r.Column, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ParseStoredTime reads a DATETIME value that came back as text (e.g. from an
// expression column), as written by the driver or by CURRENT_TIMESTAMP.
func ParseStoredTime(raw string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02 15:04:05.999999999 -0700 MST", "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05", time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ListLabelsForLeads returns labels keyed by lead id for one page of leads.
func (q *Queries) ListLabelsForLeads(ctx context.Context, leadIDs []int64) (map[int64][]LeadLabel, error) {
	out := map[int64][]LeadLabel{}
	if len(leadIDs) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(leadIDs)), ",")
	args := make([]any, len(leadIDs))
	for i, id := range leadIDs {
		args[i] = id
	}
	rows, err := q.db.QueryContext(ctx, `SELECT k.lead_id, l.id, l.nama, l.warna, l.created_at
	 FROM lead_label_links k JOIN lead_labels l ON l.id = k.label_id
	 WHERE k.lead_id IN (`+placeholders+`) ORDER BY l.nama`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var leadID int64
		var l LeadLabel
		if err := rows.Scan(&leadID, &l.ID, &l.Nama, &l.Warna, &l.CreatedAt); err != nil {
			return nil, err
		}
		out[leadID] = append(out[leadID], l)
	}
	return out, rows.Err()
}

// LeadReportRow is the minimum per-lead data the funnel report aggregates.
type LeadReportRow struct {
	ID          int64
	Stage       string
	Status      string
	LostReason  string
	CreatedAt   time.Time
	ConvertedAt sql.NullTime
	Objection   string
}

func (q *Queries) ListLeadReportRows(ctx context.Context, from, to time.Time) ([]LeadReportRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT l.id, l.stage, l.status, l.lost_reason, l.created_at, l.converted_at,
		COALESCE((SELECT a.main_objection FROM lead_analyses a WHERE a.lead_id = l.id ORDER BY a.id DESC LIMIT 1), '')
	 FROM leads l WHERE l.created_at >= ? AND l.created_at < ? ORDER BY l.created_at`, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadReportRow, 0)
	for rows.Next() {
		var r LeadReportRow
		if err := rows.Scan(&r.ID, &r.Stage, &r.Status, &r.LostReason, &r.CreatedAt, &r.ConvertedAt, &r.Objection); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LeadSentFollowupRow is one sent follow-up and whether the lead answered.
type LeadSentFollowupRow struct {
	LeadID     int64
	SentAt     time.Time
	LeadStatus string
}

func (q *Queries) ListSentLeadFollowups(ctx context.Context, from, to time.Time) ([]LeadSentFollowupRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT f.lead_id, f.sent_at, l.status FROM lead_followups f
	 JOIN leads l ON l.id = f.lead_id
	 WHERE f.status = 'sent' AND f.sent_at >= ? AND f.sent_at < ?`, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadSentFollowupRow, 0)
	for rows.Next() {
		var r LeadSentFollowupRow
		if err := rows.Scan(&r.LeadID, &r.SentAt, &r.LeadStatus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LeadHasInboundBetween reports whether the lead wrote between two instants.
func (q *Queries) LeadHasInboundBetween(ctx context.Context, leadID int64, from, to time.Time) (bool, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lead_messages
	 WHERE lead_id = ? AND direction = 'in' AND sent_at > ? AND sent_at <= ?`, leadID, from.UTC(), to.UTC()).Scan(&n)
	return n > 0, err
}

// LeadExportRow is one CSV line of /app/leads/laporan/export.
type LeadExportRow struct {
	ID         int64
	Nama       string
	WaName     string
	Phone      string
	Stage      string
	Status     string
	LostReason string
	PaketKode  string
	CreatedAt  time.Time
}

func (q *Queries) ListLeadExportRows(ctx context.Context, from, to time.Time) ([]LeadExportRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, nama, wa_name, phone, stage, status, lost_reason, paket_kode, created_at
	 FROM leads WHERE created_at >= ? AND created_at < ? ORDER BY created_at`, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LeadExportRow, 0)
	for rows.Next() {
		var r LeadExportRow
		if err := rows.Scan(&r.ID, &r.Nama, &r.WaName, &r.Phone, &r.Stage, &r.Status, &r.LostReason, &r.PaketKode, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type FindSantriByNoWARow struct {
	ID           int64
	Nama         string
	IDMahasantri string
	NoWa         string
}

// FindSantriByNoWA matches santri whose no_wa, stripped of spaces, dashes,
// dots, brackets and "+", equals one of the given digit strings.
func (q *Queries) FindSantriByNoWA(ctx context.Context, numbers []string) ([]FindSantriByNoWARow, error) {
	out := make([]FindSantriByNoWARow, 0)
	if len(numbers) == 0 {
		return out, nil
	}
	args := make([]any, len(numbers))
	for i, n := range numbers {
		args[i] = n
	}
	rows, err := q.db.QueryContext(ctx, `SELECT id, nama, id_mahasantri, no_wa FROM santri
	 WHERE REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(no_wa, ' ', ''), '-', ''), '+', ''), '.', ''), '(', ''), ')', '')
	 IN (`+strings.TrimSuffix(strings.Repeat("?,", len(numbers)), ",")+`) ORDER BY id LIMIT 5`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r FindSantriByNoWARow
		if err := rows.Scan(&r.ID, &r.Nama, &r.IDMahasantri, &r.NoWa); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
