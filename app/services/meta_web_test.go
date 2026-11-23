package services

import (
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestParseMetaPixelIDs(t *testing.T) {
	require.Equal(t, []string{"123456789012345", "9876543"}, ParseMetaPixelIDs(" 123456789012345, abc ,9876543;123456789012345 12"))
	require.Empty(t, ParseMetaPixelIDs(""))
}

func TestMetaFbc(t *testing.T) {
	now := time.UnixMilli(1764000000123)
	require.Equal(t, "fb.1.111.AbC", MetaFbc("fb.1.111.AbC", "https://x.id/daftar/a?fbclid=zzzzzzzzzzzz", now), "cookie wins")
	require.Equal(t, "fb.1.1764000000123.IwAR0abcdefghij", MetaFbc("", "https://x.id/daftar/a?utm_source=fb&fbclid=IwAR0abcdefghij", now))
	require.Empty(t, MetaFbc("", "https://x.id/daftar/a", now))
	require.Empty(t, MetaFbc("", "https://x.id/daftar/a?fbclid=<script>", now))
}

func TestSafePageURL(t *testing.T) {
	fb := "https://dashboard.example.id/daftar/sbf"
	require.Equal(t, "https://dashboard.example.id/daftar/sbf?utm_source=ig", SafePageURL("https://dashboard.example.id/daftar/sbf?utm_source=ig#top", fb))
	require.Equal(t, fb, SafePageURL("https://evil.example.com/x", fb), "foreign host")
	require.Equal(t, fb, SafePageURL("javascript:alert(1)", fb))
}

func withMeta(t *testing.T, f *registrationFixture, m *fakeMeta) *MetaCAPIService {
	t.Helper()
	capi := newTestCAPI(t, f.leadFixture, m, MetaCAPIConfig{DatasetID: "111222333", Token: "tok"})
	f.reg.WithMeta(capi)
	return capi
}

func TestSubmitReturnsPixelEventsAndStoresSignals(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	m := &fakeMeta{}
	capi := withMeta(t, f, m)

	link, err := f.reg.PublicLink(t.Context(), "tahsin-reguler")
	require.NoError(t, err)
	require.NotNil(t, link.Meta)
	require.Equal(t, []string{"111222333"}, link.Meta.PixelIDs)
	require.True(t, strings.HasPrefix(link.Meta.ViewEventID, "rs-vc-"))

	in := validSubmit("081234567890")
	in.FormToken = f.reg.FormToken("tahsin-reguler")
	f.clock = f.clock.Add(10 * time.Second)
	page := "https://dashboard.example.id/daftar/tahsin-reguler?fbclid=IwAR0abcdefghij"
	res, err := f.reg.Submit(t.Context(), "tahsin-reguler", in, models.RequestMeta{
		IP: "1.2.3.4", UserAgent: "Mozilla/5.0 test", Fbp: "fb.1.1.222", PageURL: page,
	})
	require.NoError(t, err)
	require.NotNil(t, res.Meta)

	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	names := []string{}
	for _, ev := range res.Meta.Events {
		names = append(names, ev.Name)
		require.Equal(t, metaEventID(l.ID, ev.Name), ev.ID, "browser and CAPI share the event_id")
		require.Equal(t, "IDR", ev.Currency)
	}
	require.Equal(t, []string{MetaEventLead, MetaEventCompleteRegistration}, names)
	require.Equal(t, metaHash("6281234567890"), res.Meta.User["ph"], "hashed, never raw")
	require.NotContains(t, strings.Join(mapValues(res.Meta.User), ","), "6281234567890")

	sig, err := f.querier.GetLeadWebSignals(t.Context(), l.ID)
	require.NoError(t, err)
	require.Equal(t, "fb.1.1.222", sig.Fbp)
	require.True(t, strings.HasPrefix(sig.Fbc, "fb.1.") && strings.HasSuffix(sig.Fbc, ".IwAR0abcdefghij"))
	require.Equal(t, page, sig.SourceUrl)

	// The queued CRM events go out as website events with browser signals.
	res2, err := capi.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, res2.Sent)
	for _, b := range m.bodies {
		ev := b["data"].([]any)[0].(map[string]any)
		require.Equal(t, "website", ev["action_source"], ev["event_name"])
		require.Equal(t, page, ev["event_source_url"])
		ud := ev["user_data"].(map[string]any)
		require.Equal(t, "Mozilla/5.0 test", ud["client_user_agent"])
		require.Equal(t, "1.2.3.4", ud["client_ip_address"])
		require.Equal(t, "fb.1.1.222", ud["fbp"])
		require.Equal(t, sig.Fbc, ud["fbc"])
	}

	// A second submit by the same number: no new Lead / CompleteRegistration.
	in.FormToken = f.reg.FormToken("tahsin-reguler")
	f.clock = f.clock.Add(time.Minute)
	res, err = f.reg.Submit(t.Context(), "tahsin-reguler", in, models.RequestMeta{IP: "1.2.3.4", UserAgent: "x"})
	require.NoError(t, err)
	require.NotNil(t, res.Meta)
	require.Empty(t, res.Meta.Events)
	sig2, err := f.querier.GetLeadWebSignals(t.Context(), l.ID)
	require.NoError(t, err)
	require.Equal(t, sig.Fbc, sig2.Fbc, "an empty value does not erase the ad click id")
}

func TestSubmitWithoutTrackingHasNoPixel(t *testing.T) {
	f, linkID := setupSubmitFixture(t, "wa")
	withMeta(t, f, &fakeMeta{})
	req := f.linkReq("tahsin-reguler")
	off := false
	req.MetaTracking = &off
	require.NoError(t, f.reg.UpdateLink(t.Context(), linkID, req))

	link, err := f.reg.PublicLink(t.Context(), "tahsin-reguler")
	require.NoError(t, err)
	require.Nil(t, link.Meta)
	res, err := f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	require.Nil(t, res.Meta)
	require.Zero(t, f.count(t, `SELECT COUNT(*) FROM lead_web_signals`))
}

func TestLinkExtraPixelValidation(t *testing.T) {
	f, linkID := setupSubmitFixture(t, "wa")
	withMeta(t, f, &fakeMeta{})
	req := f.linkReq("tahsin-reguler")
	req.MetaPixelExtra = "bukan-angka"
	require.Error(t, f.reg.UpdateLink(t.Context(), linkID, req))
	req.MetaPixelExtra = "111222333, 444555666"
	require.NoError(t, f.reg.UpdateLink(t.Context(), linkID, req))
	link, err := f.reg.PublicLink(t.Context(), "tahsin-reguler")
	require.NoError(t, err)
	require.Equal(t, []string{"111222333", "444555666"}, link.Meta.PixelIDs, "main pixel not duplicated")
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
