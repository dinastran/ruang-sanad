package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// fakeInsights serves GET /{ver}/{act}/insights with two pages.
type fakeInsights struct {
	mu       sync.Mutex
	queries  []string
	pages    [][]string // JSON objects per page
	failWith string
}

func (f *fakeInsights) server(t *testing.T) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.queries = append(f.queries, r.URL.Path+"?"+r.URL.RawQuery)
		fail, pages := f.failWith, f.pages
		f.mu.Unlock()
		if fail != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fail))
			return
		}
		page := 0
		if r.URL.Query().Get("after") == "p2" {
			page = 1
		}
		next := ""
		if page+1 < len(pages) {
			next = fmt.Sprintf(`,"paging":{"next":"%s%s?after=p2"}`, srv.URL, r.URL.Path)
		}
		_, _ = fmt.Fprintf(w, `{"data":[%s]%s}`, strings.Join(pages[page], ","), next)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func insightJSON(day, campaign, ad, spend string, impressions, clicks, messaging int) string {
	return fmt.Sprintf(`{"date_start":"%s","date_stop":"%s","campaign_id":"c%s","campaign_name":"Kampanye %s","adset_id":"s1","adset_name":"Set","ad_id":"a%s","ad_name":"Iklan %s","spend":"%s","impressions":"%d","reach":"%d","clicks":"%d","inline_link_clicks":"%d","actions":[{"action_type":"link_click","value":"%d"},{"action_type":"onsite_conversion.messaging_conversation_started_7d","value":"%d"}]}`,
		day, day, campaign, campaign, ad, ad, spend, impressions, impressions/2, clicks, clicks-1, clicks-1, messaging)
}

func newTestAds(t *testing.T, f *leadFixture, fake *fakeInsights, cfg MetaAdsConfig) *MetaAdsService {
	svc := NewMetaAdsService(f.querier, cfg)
	svc.baseURL = fake.server(t).URL
	svc.now = func() time.Time { return f.clock }
	return svc
}

func TestMetaAdsSyncPagesAndReplacesWindow(t *testing.T) {
	f := setupLeadFixture(t) // clock: 2026-10-24 05:00 UTC = 12.00 WIB
	fake := &fakeInsights{pages: [][]string{
		{insightJSON("2026-10-23", "1", "1", "150000.40", 1000, 30, 4), insightJSON("2026-10-24", "1", "1", "99999.60", 800, 20, 2)},
		{insightJSON("2026-10-24", "2", "2", "50000", 500, 10, 1)},
	}}
	require.Equal(t, []string{"act_123", "act_456"}, NewMetaAdsService(f.querier, MetaAdsConfig{AccountIDs: []string{"123", " act_456 ", ""}}).cfg.AccountIDs)
	svc := newTestAds(t, f, fake, MetaAdsConfig{Token: "tok", AccountIDs: []string{"123"}, APIVersion: "v99.0", AppSecret: "sec"})

	_, err := NewMetaAdsService(f.querier, MetaAdsConfig{}).Sync(t.Context(), "manual")
	require.ErrorIs(t, err, ErrMetaAdsDisabled)

	rows, err := svc.Sync(t.Context(), "manual")
	require.NoError(t, err)
	require.Equal(t, 3, rows)
	require.Len(t, fake.queries, 2, "follows paging.next")
	first := fake.queries[0]
	require.True(t, strings.HasPrefix(first, "/v99.0/act_123/insights?"))
	require.Contains(t, first, "level=ad")
	require.Contains(t, first, "time_increment=1")
	require.Contains(t, first, "appsecret_proof=")
	require.Contains(t, first, "%22since%22%3A%222026-09-27%22", "28 days ending today (WIB)")

	var spend, messaging, link int64
	require.NoError(t, f.db.QueryRow(`SELECT SUM(spend), SUM(messaging_started), SUM(link_clicks) FROM meta_ad_insights_daily`).Scan(&spend, &messaging, &link))
	require.EqualValues(t, 150000+100000+50000, spend, "spend rounded to whole rupiah")
	require.EqualValues(t, 7, messaging)
	require.EqualValues(t, 29+19+9, link)

	// Re-sync: ad a2 is gone from Meta and must disappear; a1 is revised.
	fake.pages = [][]string{{insightJSON("2026-10-24", "1", "1", "120000", 900, 25, 3)}}
	_, err = svc.Sync(t.Context(), "otomatis")
	require.NoError(t, err)
	var n int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM meta_ad_insights_daily`).Scan(&n))
	require.Equal(t, 1, n)

	// A failing sync keeps the stored data and records the error.
	fake.failWith = `{"error":{"message":"Invalid OAuth access token","code":190}}`
	_, err = svc.Sync(t.Context(), "manual")
	require.ErrorContains(t, err, "kode 190")
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM meta_ad_insights_daily`).Scan(&n))
	require.Equal(t, 1, n)
	rep, err := svc.Report(t.Context(), "", "")
	require.NoError(t, err)
	require.Equal(t, "failed", rep.LastRun.Status)
	require.Contains(t, rep.LastRun.Error, "Invalid OAuth")
	require.Equal(t, "success", rep.LastSuccess.Status)
	require.Equal(t, []string{"act_123"}, rep.Accounts, "IDs of 4 digits or fewer are not masked")
}

func TestMetaAdsDueForDailySync(t *testing.T) {
	f := setupLeadFixture(t)
	fake := &fakeInsights{pages: [][]string{{}}}
	svc := newTestAds(t, f, fake, MetaAdsConfig{Token: "tok", AccountIDs: []string{"1"}})

	f.clock = time.Date(2026, 10, 23, 18, 30, 0, 0, time.UTC) // 01.30 WIB
	due, err := svc.DueForDailySync(t.Context())
	require.NoError(t, err)
	require.False(t, due, "not before 02.00 WIB")

	f.clock = time.Date(2026, 10, 23, 19, 30, 0, 0, time.UTC) // 02.30 WIB
	due, err = svc.DueForDailySync(t.Context())
	require.NoError(t, err)
	require.True(t, due, "no successful sync yet")
	_, err = svc.Sync(t.Context(), "otomatis")
	require.NoError(t, err)
	due, err = svc.DueForDailySync(t.Context())
	require.NoError(t, err)
	require.False(t, due, "synced today")

	f.clock = f.clock.Add(21 * time.Hour)
	due, err = svc.DueForDailySync(t.Context())
	require.NoError(t, err)
	require.True(t, due, "next day")
}

func TestMetaAdsReportJoinsCampaignsWithLeadSources(t *testing.T) {
	f := setupLeadFixture(t)
	fake := &fakeInsights{pages: [][]string{{
		insightJSON("2026-10-23", "1", "1", "200000", 1000, 30, 5),
		insightJSON("2026-10-24", "1", "3", "100000", 500, 10, 2),
		insightJSON("2026-10-24", "2", "2", "50000", 300, 5, 1),
	}}}
	svc := newTestAds(t, f, fake, MetaAdsConfig{Token: "tok", AccountIDs: []string{"1"}})
	_, err := svc.Sync(t.Context(), "manual")
	require.NoError(t, err)

	reguler := sourceIDByName(t, f, "LP Kelas Reguler Tahsin")
	require.NoError(t, svc.SetCampaignSource(t.Context(), "c1", reguler))
	require.ErrorIs(t, svc.SetCampaignSource(t.Context(), "c2", 999), ErrLeadSourceNotFound)

	// Three leads in the period: two from the mapped source (one became a
	// santri), one without a source.
	for i, body := range []string{
		"Assalamu'alaikum, saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad.",
		"saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad",
		"halo",
	} {
		ev := leadEvent(fmt.Sprintf("r%d", i), "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": fmt.Sprintf("R%d", i), "body": body})
		ev.ChatID, ev.Phone = fmt.Sprintf("62812%d@s.whatsapp.net", i), fmt.Sprintf("62812%d", i)
		f.ingest(t, ev)
	}
	_, err = f.db.Exec(`UPDATE leads SET status = 'won' WHERE wa_chat_id = '628120@s.whatsapp.net'`)
	require.NoError(t, err)

	rep, err := svc.Report(t.Context(), "2026-10-18", "2026-10-24")
	require.NoError(t, err)
	require.Equal(t, "2026-10-11", rep.PrevFrom, "previous period has the same length")
	require.Equal(t, "2026-10-17", rep.PrevTo)
	require.Len(t, rep.Rows, 3)
	require.Equal(t, models.MetaAdsRow{Day: "2026-10-23", AdID: "a1", AdsetID: "s1", CampaignID: "c1", Spend: 200000, Impressions: 1000, Clicks: 30, LinkClicks: 29, Messaging: 5}, rep.Rows[0])
	require.Len(t, rep.Ads, 3)
	require.Len(t, rep.Campaigns, 2)
	mapped := map[string]int64{}
	for _, c := range rep.Campaigns {
		mapped[c.ID] = c.SourceID
	}
	require.Equal(t, map[string]int64{"c1": reguler, "c2": 0}, mapped)
	require.Equal(t, []models.MetaAdsLeadDay{
		{Day: "2026-10-24", SourceID: 0, Leads: 1},
		{Day: "2026-10-24", SourceID: reguler, Leads: 2, Won: 1},
	}, rep.LeadDays)
	require.NotEmpty(t, rep.Sources)

	// Removing the mapping is reflected in the campaign list.
	require.NoError(t, svc.SetCampaignSource(t.Context(), "c1", 0))
	rep, err = svc.Report(t.Context(), "2026-10-18", "2026-10-24")
	require.NoError(t, err)
	for _, c := range rep.Campaigns {
		require.Zero(t, c.SourceID)
	}

	_, err = svc.Report(t.Context(), "2026-10-24", "2026-10-01")
	require.Error(t, err)
}
