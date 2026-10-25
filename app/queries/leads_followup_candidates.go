package queries

import (
 "context"
 "database/sql"
 "time"
)

// DormantFollowupCandidate is an intentionally contact-free input to the MCP
// follow-up eligibility evaluator. The query is hard-gated to the pilot label.
type DormantFollowupCandidate struct {
 ID int64
 Nama string
 WaName string
 Stage string
 Permission string
 Anchor time.Time
 LastStep sql.NullString
 LastSentAt sql.NullTime
}

// ListPilotDormantFollowups excludes replied, closed, opted-out and already
// queued leads. Cursor follows lead ID, not an offset over a changing inbox.
func (q *Queries) ListPilotDormantFollowups(ctx context.Context, beforeID int64, now, todayWIBStart, tomorrowWIBStart time.Time, limit int64) ([]DormantFollowupCandidate, error) {
 rows, err := q.db.QueryContext(ctx, `
 SELECT l.id, l.nama, l.wa_name, l.stage, l.followup_permission, outbound.sent_at,
   last_sent.step, last_sent.sent_at
 FROM leads l
 JOIN lead_messages outbound ON outbound.id = (
   SELECT m.id FROM lead_messages m WHERE m.lead_id = l.id
     AND m.direction = 'out' AND m.sender <> 'followup'
   ORDER BY m.sent_at DESC, m.id DESC LIMIT 1
 )
 LEFT JOIN lead_followups last_sent ON last_sent.id = (
   SELECT f.id FROM lead_followups f
   WHERE f.lead_id = l.id AND f.status = 'sent'
     AND f.step IN ('d1','d3','d7','d14','d30') AND f.sent_at >= outbound.sent_at
   ORDER BY f.sent_at DESC, f.id DESC LIMIT 1
 )
 WHERE l.status = 'open' AND l.followup_permission <> 'no'
   AND outbound.sent_at <= ?
   AND (l.last_inbound_at IS NULL OR l.last_inbound_at < outbound.sent_at)
   AND (? = 0 OR l.id < ?)
   AND EXISTS (
     SELECT 1 FROM lead_label_links link
     JOIN lead_labels label ON label.id = link.label_id
     WHERE link.lead_id = l.id AND label.nama = 'uji-followup'
   )
   AND NOT EXISTS (
     SELECT 1 FROM lead_followups pending
     WHERE pending.lead_id = l.id AND pending.status IN ('pending_approval', 'approved')
   )
   AND NOT EXISTS (
     SELECT 1 FROM lead_followups sent
     WHERE sent.lead_id = l.id AND sent.status = 'sent'
       AND sent.sent_at >= ? AND sent.sent_at < ?
   )
 ORDER BY l.id DESC LIMIT ?`,
  now.UTC().Add(-24*time.Hour), beforeID, beforeID,
  todayWIBStart.UTC(), tomorrowWIBStart.UTC(), limit)
 if err != nil { return nil, err }
 defer rows.Close()
 result := make([]DormantFollowupCandidate, 0)
 for rows.Next() {
  var r DormantFollowupCandidate
  if err := rows.Scan(&r.ID, &r.Nama, &r.WaName, &r.Stage, &r.Permission,
   &r.Anchor, &r.LastStep, &r.LastSentAt); err != nil { return nil, err }
  result = append(result, r)
 }
 return result, rows.Err()
}
