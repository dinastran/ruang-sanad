package queries

import (
 "context"
 "database/sql"
 "fmt"
 "time"
)

// Dashboard uses distinct date semantics: created_at for acquisition, converted_at for closing.
func (q *Queries) ListLeadDashboardClosings(ctx context.Context, from, to time.Time) ([]time.Time, error) {
 rows, err := q.db.QueryContext(ctx, `SELECT converted_at FROM leads WHERE status = 'won' AND converted_at >= ? AND converted_at < ? ORDER BY converted_at`, from.UTC(), to.UTC())
 if err != nil { return nil, err }
 defer rows.Close()
 out := make([]time.Time, 0)
 for rows.Next() {
  var converted time.Time
  if err := rows.Scan(&converted); err != nil { return nil, err }
  out = append(out, converted)
 }
 return out, rows.Err()
}

func (q *Queries) LeadDashboardCreatedCount(ctx context.Context, from, to time.Time) (int64, error) {
 var n int64
 err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM leads WHERE created_at >= ? AND created_at < ?`, from.UTC(), to.UTC()).Scan(&n)
 return n, err
}

func (q *Queries) LeadDashboardReviewCount(ctx context.Context, from, to time.Time) (int64, error) {
 var n int64
 err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM leads WHERE status = 'open' AND needs_review = 1 AND created_at >= ? AND created_at < ?`, from.UTC(), to.UTC()).Scan(&n)
 return n, err
}

type LeadDashboardQueueCount struct {
 Status string
 Count int64
}

// Queue state is current workload, intentionally not filtered by lead acquisition date.
func (q *Queries) LeadDashboardQueue(ctx context.Context) ([]LeadDashboardQueueCount, error) {
 rows, err := q.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM lead_followups WHERE status IN ('pending_approval', 'approved', 'failed') GROUP BY status`)
 if err != nil { return nil, err }
 defer rows.Close()
 out := []LeadDashboardQueueCount{}
 for rows.Next() {
  var r LeadDashboardQueueCount
  if err := rows.Scan(&r.Status, &r.Count); err != nil { return nil, err }
  out = append(out, r)
 }
 return out, rows.Err()
}

type LeadDashboardPriorityRow struct {
 ID int64
 Nama string
 Stage string
 NeedsReview bool
 LastInbound sql.NullTime
 LastOutbound sql.NullTime
 CreatedAt time.Time
}

// A bounded global action queue, not a sample of a selected acquisition cohort.
func (q *Queries) LeadDashboardPriorities(ctx context.Context) ([]LeadDashboardPriorityRow, error) {
 rows, err := q.db.QueryContext(ctx, `SELECT id,
 COALESCE(NULLIF(nama, ''), NULLIF(wa_name, ''), 'Lead L-' || id),
 stage, needs_review, last_inbound_at, last_outbound_at, created_at
 FROM leads WHERE status = 'open' AND (
 stage IN ('bukti_masuk','terverifikasi') OR needs_review = 1 OR
 (last_inbound_at IS NOT NULL AND (last_outbound_at IS NULL OR last_inbound_at > last_outbound_at))
 )
 ORDER BY CASE WHEN stage = 'bukti_masuk' THEN 0
 WHEN stage = 'terverifikasi' THEN 1 WHEN needs_review = 1 THEN 2 ELSE 3 END,
 COALESCE(last_inbound_at, created_at) DESC, id DESC LIMIT 12`)
 if err != nil { return nil, err }
 defer rows.Close()
 out := []LeadDashboardPriorityRow{}
 for rows.Next() {
  var r LeadDashboardPriorityRow
  if err := rows.Scan(&r.ID, &r.Nama, &r.Stage, &r.NeedsReview, &r.LastInbound, &r.LastOutbound, &r.CreatedAt); err != nil { return nil, err }
  out = append(out, r)
 }
 return out, rows.Err()
}

// Count only follow-ups whose full 72-hour response window has elapsed.
func (q *Queries) LeadDashboardMaturedFollowups(ctx context.Context, from, to, cutoff time.Time) (eligible, replied int64, err error) {
 err = q.db.QueryRowContext(ctx, `SELECT COUNT(*),
 COALESCE(SUM(CASE WHEN EXISTS (
   SELECT 1 FROM lead_messages m WHERE m.lead_id = f.lead_id AND m.direction = 'in'
   AND datetime(m.sent_at) > datetime(f.sent_at)
   AND datetime(m.sent_at) <= datetime(f.sent_at, '+72 hours')
  ) THEN 1 ELSE 0 END), 0)
 FROM lead_followups f WHERE f.status = 'sent' AND f.sent_at >= ? AND f.sent_at < ? AND f.sent_at <= ?`,
 from.UTC(), to.UTC(), cutoff.UTC()).Scan(&eligible, &replied)
 return
}

// LeadDashboardPriorityPage returns the entire actionable queue in bounded pages.
// Reasons are mutually exclusive and follow the same precedence as the dashboard.
func (q *Queries) LeadDashboardPriorityPage(ctx context.Context, search, reason string, limit, offset int) ([]LeadDashboardPriorityRow, int64, error) {
 const category = `CASE WHEN stage = 'bukti_masuk' THEN 'verify'
 WHEN stage = 'terverifikasi' THEN 'convert'
 WHEN needs_review = 1 THEN 'review' ELSE 'reply' END`
 where := ` FROM leads WHERE status = 'open' AND (
 stage IN ('bukti_masuk','terverifikasi') OR needs_review = 1 OR
 (last_inbound_at IS NOT NULL AND (last_outbound_at IS NULL OR last_inbound_at > last_outbound_at))
 )`
 args := []any{}
 if reason != "" {
  where += " AND (" + category + ") = ?"
  args = append(args, reason)
 }
 if search != "" {
  where += " AND (LOWER(COALESCE(nama,'')) LIKE ? OR LOWER(COALESCE(wa_name,'')) LIKE ? OR CAST(id AS TEXT) LIKE ?)"
  term := "%" + search + "%"
  args = append(args, term, term, term)
 }
 var total int64
 if err := q.db.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&total); err != nil {
  return nil, 0, err
 }
 sqlQuery := `SELECT id, COALESCE(NULLIF(nama, ''), NULLIF(wa_name, ''), 'Lead L-' || id),
 stage, needs_review, last_inbound_at, last_outbound_at, created_at` +
  where + " ORDER BY " + category +
  ", COALESCE(last_inbound_at, created_at) ASC, id ASC LIMIT ? OFFSET ?"
 rows, err := q.db.QueryContext(ctx, sqlQuery, append(args, limit, offset)...)
 if err != nil { return nil, 0, fmt.Errorf("list lead priorities: %w", err) }
 defer rows.Close()
 out := make([]LeadDashboardPriorityRow, 0, limit)
 for rows.Next() {
  var r LeadDashboardPriorityRow
  if err := rows.Scan(&r.ID, &r.Nama, &r.Stage, &r.NeedsReview, &r.LastInbound, &r.LastOutbound, &r.CreatedAt); err != nil {
   return nil, 0, err
  }
  out = append(out, r)
 }
 return out, total, rows.Err()
}
