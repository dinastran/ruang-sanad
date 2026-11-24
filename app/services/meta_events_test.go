package services

import (
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestNormalizeMetaEvents(t *testing.T) {
	d, err := NormalizeMetaEvents(nil)
	require.NoError(t, err)
	require.Equal(t, DefaultMetaEvents(), d)

	got, err := NormalizeMetaEvents(map[string][]string{
		MetaTriggerWA:     {"contact", "Contact", " KlikWA "},
		MetaTriggerSubmit: {"Lead", "SubmitApplication"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Contact", "KlikWA"}, got[MetaTriggerWA], "canonical case, deduped, trimmed")
	require.Empty(t, got[MetaTriggerLangkah1], "a trigger left out = no events")

	for name, in := range map[string]map[string][]string{
		"funnel at wrong trigger":  {MetaTriggerLangkah1: {"Lead"}},
		"proof funnel at submit":   {MetaTriggerSubmit: {"AddPaymentInfo"}},
		"purchase":                 {MetaTriggerSubmit: {"Purchase"}},
		"bad custom name":          {MetaTriggerWA: {"klik wa!"}},
		"more than five per event": {MetaTriggerWA: {"A1", "A2", "A3", "A4", "A5", "A6"}},
	} {
		_, err := NormalizeMetaEvents(in)
		require.Error(t, err, name)
	}
	require.Equal(t, DefaultMetaEvents(), ParseMetaEvents(""))
	require.Equal(t, DefaultMetaEvents(), ParseMetaEvents("{bad json"))
}

func (m *fakeMeta) eventNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []string{}
	for _, b := range m.bodies {
		ev := b["data"].([]any)[0].(map[string]any)
		out = append(out, ev["event_name"].(string))
	}
	return out
}

func TestSubmitCustomMetaEvents(t *testing.T) {
	f, linkID := setupSubmitFixture(t, "wa")
	m := &fakeMeta{}
	withMeta(t, f, m)
	req := f.linkReq("tahsin-reguler")
	req.MetaEvents = map[string][]string{
		MetaTriggerLangkah1: {"Contact", "IsiLangkah1"},
		MetaTriggerSubmit:   {"CompleteRegistration", "SubmitApplication", "DaftarTerkirim"},
	}
	require.NoError(t, f.reg.UpdateLink(t.Context(), linkID, req))

	link, err := f.reg.PublicLink(t.Context(), "tahsin-reguler")
	require.NoError(t, err)
	require.Len(t, link.Meta.Langkah1, 2)
	require.False(t, link.Meta.Langkah1[0].Custom)
	require.True(t, link.Meta.Langkah1[1].Custom)

	in := validSubmit("081234567890")
	in.FormToken = f.reg.FormToken("tahsin-reguler")
	f.clock = f.clock.Add(10 * time.Second)
	res, err := f.reg.Submit(t.Context(), "tahsin-reguler", in, models.RequestMeta{
		IP: "1.2.3.4", UserAgent: "Mozilla/5.0 test", PageURL: "https://dashboard.example.id/daftar/tahsin-reguler",
	})
	require.NoError(t, err)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)

	names := []string{}
	for _, ev := range res.Meta.Events {
		names = append(names, ev.Name)
		switch ev.Name {
		case MetaEventCompleteRegistration:
			require.Equal(t, metaEventID(l.ID, ev.Name), ev.ID, "funnel event shares the CRM event_id")
		default:
			require.True(t, strings.HasPrefix(ev.ID, "rs-ev-"), ev.Name)
		}
	}
	require.Equal(t, []string{"CompleteRegistration", "SubmitApplication", "DaftarTerkirim"}, names, "Lead not chosen → not fired")
	require.True(t, res.Meta.Events[2].Custom)

	// Non-funnel events are also sent through CAPI right away (async);
	// the funnel event goes through the CRM queue as before.
	require.Eventually(t, func() bool { return len(m.eventNames()) == 2 }, 3*time.Second, 20*time.Millisecond)
	require.ElementsMatch(t, []string{"SubmitApplication", "DaftarTerkirim"}, m.eventNames())
}

func TestWaTrackingOnlyWithEvents(t *testing.T) {
	f, linkID := setupSubmitFixture(t, "wa")
	m := &fakeMeta{}
	withMeta(t, f, m)
	req := f.linkReq("tahsin-reguler")
	req.Mode = "wa"
	require.NoError(t, f.reg.UpdateLink(t.Context(), linkID, req))
	sig := MetaWebSignals{UserAgent: "Mozilla/5.0", SourceURL: "https://x.id/daftar/tahsin-reguler"}
	require.Nil(t, f.reg.WaTracking(t.Context(), "tahsin-reguler", sig), "default: no event, plain redirect")

	req.MetaEvents = map[string][]string{MetaTriggerWA: {"Contact"}}
	require.NoError(t, f.reg.UpdateLink(t.Context(), linkID, req))
	tr := f.reg.WaTracking(t.Context(), "tahsin-reguler", sig)
	require.NotNil(t, tr)
	require.Equal(t, []string{"111222333"}, tr.PixelIDs)
	require.Len(t, tr.Events, 1)
	require.Equal(t, "Contact", tr.Events[0].Name)
	require.Eventually(t, func() bool { return len(m.eventNames()) == 1 }, 3*time.Second, 20*time.Millisecond)
}

func TestProofTrackingEvents(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	m := &fakeMeta{}
	withMeta(t, f, m)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))
	sig := MetaWebSignals{UserAgent: "Mozilla/5.0", SourceURL: "https://x.id/daftar/bayar/" + ref, Fbp: "fb.1.1.5"}

	tr := f.reg.ProofTracking(t.Context(), ref, sig)
	require.NotNil(t, tr)
	require.Len(t, tr.Events, 1)
	require.Equal(t, MetaEventAddPaymentInfo, tr.Events[0].Name)
	require.Equal(t, metaEventID(leadID, MetaEventAddPaymentInfo), tr.Events[0].ID)
	require.EqualValues(t, 150000, tr.Events[0].Value)
	saved, err := f.querier.GetLeadWebSignals(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, "fb.1.1.5", saved.Fbp, "signals stored so the queued AddPaymentInfo goes out as a website event")
	require.Nil(t, f.reg.ProofTracking(t.Context(), "tidak-ada", sig))
}
