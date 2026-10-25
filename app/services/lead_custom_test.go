package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestLeadPauseExpirationAndEventOrdering(t *testing.T) {
	for _, tt := range []struct {
		name, until string
		paused      bool
	}{
		{"future", "2026-10-24T06:00:00Z", true},
		{"expired", "2026-10-24T04:59:59Z", false},
		{"boundary", "2026-10-24T05:00:00Z", false},
		{"timezone", "2026-10-24T12:00:00+07:00", false},
		{"indefinite", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := setupLeadFixture(t)
			f.ingest(t, leadEvent("pause", "ai.paused", "2026-10-24T04:00:00Z", map[string]string{"until": tt.until}))
			check := func(want bool) {
				t.Helper()
				detail, err := f.service.Detail(t.Context(), f.lead(t).ID)
				require.NoError(t, err)
				require.Equal(t, want, detail.AIPaused)
				list, err := f.service.List(t.Context(), models.LeadListParams{})
				require.NoError(t, err)
				require.Len(t, list.Items, 1)
				require.Equal(t, want, list.Items[0].AIPaused)
			}
			check(tt.paused)
			// More than a full timeline page must not lose the pause state.
			for i := 0; i < 105; i++ {
				f.ingest(t, leadEvent(fmt.Sprintf("m%d", i), "message.in", "2026-10-24T04:10:00Z", map[string]string{"body": "Halo"}))
			}
			check(tt.paused)
			f.ingest(t, leadEvent("resume", "ai.resumed", "2026-10-24T04:30:00Z", nil))
			// Retry of an older pause arrives after resume: occurrence order wins.
			f.ingest(t, leadEvent("late-pause", "ai.paused", "2026-10-24T04:05:00Z", map[string]string{"until": "2026-10-24T08:00:00Z"}))
			check(false)
		})
	}
}

func TestSharedLeadLabelCRUD(t *testing.T) {
	f := setupLeadFixture(t)
	ctx := t.Context()
	f.ingest(t, leadEvent("hello", "message.in", f.clock.Format(time.RFC3339), nil))
	require.NoError(t, f.service.SaveLabel(ctx, 0, "  Minta   telepon  ", "brand"))
	label, err := f.querier.GetLeadLabelByName(ctx, "Minta telepon")
	require.NoError(t, err)
	require.Error(t, f.service.SaveLabel(ctx, 0, "minta telepon", "neutral"))
	require.Error(t, f.service.SaveLabel(ctx, 0, " ", "neutral"))
	require.Error(t, f.service.SaveLabel(ctx, 0, "Tes", "invalid"))
	require.Error(t, f.service.SaveLabel(ctx, 9999, "Tes", "neutral"))
	require.NoError(t, f.service.SetLabel(ctx, f.lead(t).ID, f.csID, label.ID, true))
	require.NoError(t, f.service.SaveLabel(ctx, label.ID, "Jadwalkan telepon", "secondary"))
	detail, err := f.service.Detail(ctx, f.lead(t).ID)
	require.NoError(t, err)
	require.Equal(t, "Jadwalkan telepon", detail.Labels[0].Nama)
	require.Equal(t, "secondary", detail.Labels[0].Warna)
	require.NoError(t, f.service.DeleteLabel(ctx, label.ID))
	detail, err = f.service.Detail(ctx, f.lead(t).ID)
	require.NoError(t, err)
	require.Empty(t, detail.Labels)
	pilot, err := f.querier.GetLeadLabelByName(ctx, pilotFollowupLabel)
	require.NoError(t, err)
	require.Error(t, f.service.DeleteLabel(ctx, pilot.ID))
	require.Error(t, f.service.SaveLabel(ctx, pilot.ID, "renamed", "neutral"))
	require.NoError(t, f.service.SaveLabel(ctx, pilot.ID, pilot.Nama, "brand"))
}
