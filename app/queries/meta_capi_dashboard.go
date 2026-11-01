package queries

import (
 "context"
 "time"
)

// MetaCAPIAnalyticsRow groups the persisted outgoing queue, not Meta's
// attributed conversions. All timestamps are stored UTC and reported in WIB.
type MetaCAPIAnalyticsRow struct {
 Day string
 Event string
 Status string
 TestMode bool
 Count int64
 PurchaseValue int64
}

func (q *Queries) ListMetaCAPIAnalytics(ctx context.Context, start, end time.Time) ([]MetaCAPIAnalyticsRow, error) {
 rows, err := q.db.QueryContext(ctx, `
SELECT DATE(event_time, '+7 hours') AS day, event_name, status, test_mode,
 COUNT(*) AS total,
 COALESCE(SUM(CASE WHEN event_name = 'Purchase'
   AND status = 'sent' AND test_mode = 0 AND value > 0
   THEN value ELSE 0 END), 0) AS purchase_value
FROM meta_capi_events
WHERE event_time >= ? AND event_time < ?
GROUP BY day, event_name, status, test_mode
ORDER BY day, event_name, status, test_mode`, start.UTC(), end.UTC())
 if err != nil { return nil, err }
 defer rows.Close()
 out := make([]MetaCAPIAnalyticsRow,0)
 for rows.Next() {
  var item MetaCAPIAnalyticsRow
  var testMode int64
  if err := rows.Scan(&item.Day,&item.Event,&item.Status,&testMode,&item.Count,&item.PurchaseValue); err != nil {
   return nil, err
  }
  item.TestMode = testMode == 1
  out = append(out,item)
 }
 return out, rows.Err()
}

// MetaCAPIAuditRow exposes only persisted diagnostic fields. It never loads
// phone, hashed personal identifiers, access tokens, or complete payloads.
type MetaCAPIAuditRow struct {
 ID int64
 LeadID int64
 LeadName string
 EventName string
 EventID string
 EventTime time.Time
 Value int64
 Status string
 Attempts int64
 LastError string
 TestMode bool
 SentAt string
 FBTraceID string
}

func (q *Queries) ListMetaCAPIAudit(ctx context.Context, limit int) ([]MetaCAPIAuditRow,error) {
 rows,err := q.db.QueryContext(ctx,`
SELECT e.id,e.lead_id,
 COALESCE(NULLIF(l.nama,''),NULLIF(l.wa_name,''),'Lead ' || l.id),
 e.event_name,e.event_id,e.event_time,e.value,e.status,
 e.attempts,e.last_error,e.test_mode,COALESCE(CAST(e.sent_at AS TEXT),''),
 COALESCE(e.fbtrace_id,'')
FROM meta_capi_events e JOIN leads l ON l.id = e.lead_id
ORDER BY e.id DESC LIMIT ?`,limit)
 if err != nil { return nil,err }
 defer rows.Close()
 out := make([]MetaCAPIAuditRow,0,limit)
 for rows.Next() {
  var r MetaCAPIAuditRow
  var testMode int64
  if err := rows.Scan(&r.ID,&r.LeadID,&r.LeadName,&r.EventName,&r.EventID,&r.EventTime,&r.Value,&r.Status,&r.Attempts,&r.LastError,&testMode,&r.SentAt,&r.FBTraceID); err != nil {
   return nil,err
  }
  r.TestMode = testMode == 1
  out = append(out,r)
 }
 return out,rows.Err()
}
