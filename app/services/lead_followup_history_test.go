package services

import (
	"context"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// History shows a failed send as "failed" with the bridge reason, and its
// timeline runs proposed → approved → failed → retried → sent.
func TestFollowupHistoryAndTimeline(t *testing.T) {
	ctx := context.Background()
	o := setupOutbox(t)
	_, fid := o.approvedD1(t, "6281200000031")
	_, okID := o.approvedD1(t, "6281200000032")

	o.bridge.err = &OutboundError{Msg: "bridge menolak pesan: HTTP 400 invalid: chat belum pernah masuk", Permanent: true}
	o.at(fuAt(25, 4))
	require.Equal(t, 1, o.tick(t).Failed)

	h, err := o.service.FollowupHistory(ctx, models.FollowupHistoryFilter{Status: "failed"})
	require.NoError(t, err)
	require.Len(t, h.Items, 1)
	failed := h.Items[0]
	require.Contains(t, failed.Note, "chat belum pernah masuk")
	require.EqualValues(t, 1, failed.Attempts)
	require.EqualValues(t, 1, h.Counts["failed"])
	require.EqualValues(t, 1, h.Counts["approved"])

	// Retry, then send succeeds.
	o.bridge.err = nil
	require.NoError(t, o.worker.Retry(ctx, failed.OutboxID, o.csID))
	o.at(fuAt(25, 5))
	require.Equal(t, 1, o.tick(t).Sent)

	h, err = o.service.FollowupHistory(ctx, models.FollowupHistoryFilter{Status: "sent"})
	require.NoError(t, err)
	require.Len(t, h.Items, 1)
	require.Equal(t, failed.ID, h.Items[0].ID)
	require.NotEmpty(t, h.Items[0].SentAt)

	tl, err := o.service.FollowupTimeline(ctx, failed.ID)
	require.NoError(t, err)
	var types []string
	for _, e := range tl {
		types = append(types, e.Type)
	}
	require.Equal(t, []string{"followup.proposed", "followup.approved", "outbox.failed", "outbox.retried", "followup.sent"}, types)
	require.Contains(t, tl[2].Detail, "chat belum pernah masuk")

	// The other lead's follow-up has its own, separate trail.
	other, err := o.service.FollowupTimeline(ctx, okID)
	require.NoError(t, err)
	require.Len(t, other, 2)
	_ = fid

	// Unknown filters are ignored, not errors.
	h, err = o.service.FollowupHistory(ctx, models.FollowupHistoryFilter{Status: "x", Step: "y", From: "bad"})
	require.NoError(t, err)
	require.Len(t, h.Items, 2)
	require.Empty(t, h.Filter.Status)
}
