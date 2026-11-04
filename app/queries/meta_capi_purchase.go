package queries

import "context"

// GetMetaCAPIPurchaseAmount returns the last payment verification amount used
// by the CAPI reconciliation path. Zero means there is no positive verified
// amount and callers should fall back to the lead's expected payment.
func (q *Queries) GetMetaCAPIPurchaseAmount(ctx context.Context, leadID int64) (int64, error) {
	var amount int64
	err := q.db.QueryRowContext(ctx, `
SELECT COALESCE((
  SELECT CAST(json_extract(payload, '$.amount') AS INTEGER)
  FROM lead_events
  WHERE lead_id = ? AND type = 'payment.verified'
  ORDER BY id DESC LIMIT 1
), 0)`, leadID).Scan(&amount)
	return amount, err
}
