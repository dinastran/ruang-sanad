package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A successful CRM conversion must persist Purchase in the same transaction,
// without waiting for the two-minute reconciliation worker.
func TestLeadConvertQueuesPurchaseInTransaction(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.verified(t)
	meta := newTestCAPI(t, f, &fakeMeta{}, MetaCAPIConfig{
		DatasetID: "test-dataset", Token: "test-token", TestEventCode: "TEST_PURCHASE",
	})
	f.service.WithMetaCAPI(meta)
	req := convertRequest()
	req.Nominal = 200000 // purchase value must come from payment verification, not the santri form
	_, err := f.service.Convert(t.Context(), id, f.csID, req)
	require.NoError(t, err)

	lead := f.lead(t)
	require.True(t, lead.ConvertedAt.Valid)
	var eventID, status string
	var eventTime string
	var value int64
	var testMode int
	require.NoError(t, f.db.QueryRow(`SELECT event_id, CAST(event_time AS TEXT), value, status, test_mode
FROM meta_capi_events WHERE lead_id = ? AND event_name = 'Purchase'`, id).
		Scan(&eventID, &eventTime, &value, &status, &testMode))
	require.Equal(t, metaEventID(id, MetaEventPurchase), eventID)
	require.Contains(t, eventTime, "2026-10-24 05:00:00")
	require.EqualValues(t, 149000, value)
	require.Equal(t, "pending", status)
	require.Equal(t, 1, testMode, "test mode must not be labelled as production")

	_, err = f.service.Convert(t.Context(), id, f.csID, req)
	require.ErrorIs(t, err, ErrLeadAlreadyConverted)

	tick, err := meta.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 5, tick.Sent)
	require.Equal(t, 4, tick.Queued, "reconciliation creates only the other four milestones")
	var n int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM meta_capi_events
WHERE lead_id = ? AND event_name = 'Purchase'`, id).Scan(&n))
	require.Equal(t, 1, n, "worker must not duplicate Purchase")
}

func TestLeadConvertPurchaseOutboxFailureRollsBackEverything(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.verified(t)
	f.service.WithMetaCAPI(NewMetaCAPIService(f.querier, MetaCAPIConfig{DatasetID: "test-dataset", Token: "test-token"}))
	var before int
	require.NoError(t, f.db.QueryRow("SELECT COUNT(*) FROM santri").Scan(&before))
	_, err := f.db.Exec(`CREATE TRIGGER reject_purchase_outbox
BEFORE INSERT ON meta_capi_events
WHEN NEW.event_name = 'Purchase'
BEGIN SELECT RAISE(ABORT, 'test purchase outbox unavailable'); END`)
	require.NoError(t, err)

	_, err = f.service.Convert(t.Context(), id, f.csID, convertRequest())
	require.ErrorContains(t, err, "test purchase outbox unavailable")
	lead := f.lead(t)
	require.Equal(t, LeadStatusOpen, lead.Status)
	require.False(t, lead.SantriID.Valid)
	require.False(t, lead.ConvertedAt.Valid)
	var after int
	require.NoError(t, f.db.QueryRow("SELECT COUNT(*) FROM santri").Scan(&after))
	require.Equal(t, before, after, "santri creation must be rolled back with the outbox")
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM meta_capi_events WHERE lead_id = ?`, id).Scan(&after))
	require.Zero(t, after)
}

func TestLeadConvertDoesNotQueueCAPIWhenDisabled(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.verified(t)
	f.service.WithMetaCAPI(NewMetaCAPIService(f.querier, MetaCAPIConfig{}))
	_, err := f.service.Convert(t.Context(), id, f.csID, convertRequest())
	require.NoError(t, err)
	var n int
	require.NoError(t, f.db.QueryRow("SELECT COUNT(*) FROM meta_capi_events").Scan(&n))
	require.Zero(t, n, "preserve existing CAPI opt-in behavior")
}
