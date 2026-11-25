package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestLeadListSearchByID(t *testing.T) {
	f := setupLeadFixture(t)
	now := time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC)
	insert := func(id int64, nama, phone string) int64 {
		_, err := f.db.Exec(`INSERT INTO leads (id, wa_chat_id, nama, phone, created_at, updated_at, stage, status) VALUES (?, ?, ?, ?, ?, ?, 'discovery', 'open')`, id, nama, nama, phone, now, now)
		require.NoError(t, err)
		return id
	}
	short := insert(171, "Nurdin", "6283800000001")
	long := insert(1711, "Habibullah", "6283800000002")
	// Higher id and more recent activity, so without pinning it would sort first.
	phoneMatch := insert(1800, "Ahmad", "6281711112222")
	_, err := f.db.Exec(`UPDATE leads SET last_inbound_at = ? WHERE id = ?`, now.Add(time.Hour), phoneMatch)
	require.NoError(t, err)
	insert(1900, "Budi", "6289900000000")

	ids := func(search string) []int64 {
		t.Helper()
		resp, err := f.service.List(t.Context(), models.LeadListParams{Search: search})
		require.NoError(t, err)
		out := []int64{}
		for _, it := range resp.Items {
			out = append(out, it.ID)
		}
		require.Equal(t, int64(len(out)), resp.Total, "counts must agree with items for %q", search)
		return out
	}

	// Short number (< 4 digits, no phone search): just the lead with that id.
	require.Equal(t, []int64{short}, ids("171"))
	// Longer number: the lead with that id pinned first, then phone matches.
	require.Equal(t, []int64{long, phoneMatch}, ids("1711"))
	// Lead ref: exactly that lead.
	require.Equal(t, []int64{long}, ids("L-1711"))
	require.Equal(t, []int64{short}, ids("l171"))
	// Names still work.
	require.Equal(t, []int64{phoneMatch}, ids("ahmad"))
}

func TestApplyLeadSearch(t *testing.T) {
	for _, tt := range []struct {
		q                 string
		only, searchID    int64
		search, phoneLike string
	}{
		{q: "171", searchID: 171, search: "171"},
		{q: "#171", searchID: 171, search: "#171"},
		{q: "L-171", only: 171},
		{q: "L 171", only: 171},
		{q: "081234", searchID: 81234, search: "081234", phoneLike: "81234"},
		{q: "Lukman", search: "Lukman"},
	} {
		var f queries.LeadListFilter
		applyLeadSearch(&f, tt.q)
		require.Equal(t, tt.only, f.OnlyID, tt.q)
		require.Equal(t, tt.searchID, f.SearchID, tt.q)
		require.Equal(t, tt.search, f.Search, tt.q)
		require.Equal(t, tt.phoneLike, f.PhoneSearch, tt.q)
	}
}

func TestStoredTimeToWIB(t *testing.T) {
	// payments.updated_at comes back as Go time text; the program page needs RFC3339 WIB.
	parsed, ok := queries.ParseStoredTime("2026-11-24 23:40:42 +0000 UTC")
	require.True(t, ok)
	require.Equal(t, "2026-11-25T06:40:42+07:00", formatLeadTime(parsed))
}
