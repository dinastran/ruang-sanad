package services

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

type fakeMeta struct {
	mu     sync.Mutex
	bodies []map[string]any
	paths  []string
	fail   bool
}

func (m *fakeMeta) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		m.mu.Lock()
		m.bodies = append(m.bodies, body)
		m.paths = append(m.paths, r.URL.Path)
		fail := m.fail
		m.mu.Unlock()
		if fail {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid parameter","code":100,"error_subcode":2804003,"fbtrace_id":"x"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"events_received":1,"messages":[],"fbtrace_id":"trace-ok"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestCAPI(t *testing.T, f *leadFixture, m *fakeMeta, cfg MetaCAPIConfig) *MetaCAPIService {
	svc := NewMetaCAPIService(f.querier, cfg)
	svc.baseURL = m.server(t).URL
	svc.now = func() time.Time { return f.clock }
	return svc
}

func TestLeadMilestonesFromStagesAndTimestamps(t *testing.T) {
	created := time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC)
	l := queries.ListMetaCAPICandidatesRow{
		ID: 1, CreatedAt: created, Status: LeadStatusWon, PaymentAmount: 350000,
		ProofReceivedAt: sql.NullTime{Time: created.Add(5 * time.Hour), Valid: true},
		ConvertedAt:     sql.NullTime{Time: created.Add(9 * time.Hour), Valid: true},
	}
	// The lead skipped data_terisi and jumped straight to info_bayar_dikirim:
	// CompleteRegistration takes the time of the later stage.
	stages := map[string]time.Time{
		LeadStageDiscovery:        created.Add(time.Hour),
		LeadStageInfoBayarDikirim: created.Add(3 * time.Hour),
		LeadStageBuktiMasuk:       created.Add(6 * time.Hour),
	}
	got := leadMilestones(l, stages)
	require.Equal(t, []metaMilestone{
		{Event: MetaEventLead, At: created},
		{Event: MetaEventCompleteRegistration, At: created.Add(3 * time.Hour)},
		{Event: MetaEventInitiateCheckout, At: created.Add(3 * time.Hour)},
		{Event: MetaEventAddPaymentInfo, At: created.Add(5 * time.Hour)},
		{Event: MetaEventPurchase, At: created.Add(9 * time.Hour), Value: 350000},
	}, got, "proof_received_at is earlier than the bukti_masuk stage event")

	open := queries.ListMetaCAPICandidatesRow{ID: 2, CreatedAt: created, Status: LeadStatusOpen}
	require.Equal(t, []metaMilestone{{Event: MetaEventLead, At: created}}, leadMilestones(open, nil))
}

func TestMetaCAPIDisabledDoesNothing(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("c0", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "C0", "body": "halo"}))
	m := &fakeMeta{}
	svc := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "123"})
	res, err := svc.Tick(t.Context())
	require.NoError(t, err)
	require.Zero(t, res)
	require.Empty(t, m.bodies)
	_, err = svc.RetryFailed(t.Context())
	require.ErrorIs(t, err, ErrMetaCAPIDisabled)
}

func TestMetaCAPISendsFunnelOnceWithHashedData(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("c1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "C1", "body": "Saya mau daftar"}))
	l := f.lead(t)
	at := func(h int) time.Time { return time.Date(2026, 10, 24, h, 0, 0, 0, time.UTC) }
	_, err := f.db.Exec(`UPDATE leads SET nama = 'Fulanah Binti Fulan', domisili = 'Jakarta Selatan, DKI', jenis_kelamin = 'P',
		stage = 'terverifikasi', status = 'won', payment_amount = 350000, payment_info_sent_at = ?, proof_received_at = ?, converted_at = ? WHERE id = ?`,
		at(4), at(4), at(5), l.ID)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO lead_events (lead_id, event_id, type, actor, payload, created_at) VALUES
		(?, 'st1', 'form.parsed', 'bridge', '{"stage_from":"form_dikirim","stage_to":"data_terisi"}', ?),
		(?, 'st2', 'payment.verified', 'user:1', '{"stage_from":"bukti_masuk","stage_to":"terverifikasi","amount":300000,"expected_amount":350000}', ?)`,
		l.ID, at(3), l.ID, at(4))
	require.NoError(t, err)

	m := &fakeMeta{}
	svc := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "987654321", Token: "tok", AppSecret: "sec", TestEventCode: "TEST123", APIVersion: "v99.0"})
	f.clock = at(6)
	res, err := svc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, MetaCAPITickResult{Queued: 5, Sent: 5}, res)

	names := []string{}
	for i, b := range m.bodies {
		require.Equal(t, "/v99.0/987654321/events", m.paths[i])
		require.Equal(t, "TEST123", b["test_event_code"])
		require.NotEmpty(t, b["appsecret_proof"])
		raw, _ := json.Marshal(b)
		require.NotContains(t, string(raw), "6281234567890", "raw phone never leaves the server")
		require.NotContains(t, strings.ToLower(string(raw)), "fulanah")
		ev := b["data"].([]any)[0].(map[string]any)
		names = append(names, ev["event_name"].(string))
		require.Equal(t, "system_generated", ev["action_source"])
		ud := ev["user_data"].(map[string]any)
		require.Equal(t, []any{metaHash("6281234567890")}, ud["ph"])
		require.Equal(t, []any{metaHash("fulanah")}, ud["fn"])
		require.Equal(t, []any{metaHash("fulan")}, ud["ln"])
		require.Equal(t, []any{metaHash("jakartaselatan")}, ud["ct"])
		require.Equal(t, []any{metaHash("f")}, ud["ge"])
		cd := ev["custom_data"].(map[string]any)
		require.Equal(t, "IDR", cd["currency"])
		if ev["event_name"] == MetaEventPurchase {
			require.EqualValues(t, 300000, cd["value"], "the verified amount, not the expected bill")
			require.EqualValues(t, at(5).Unix(), ev["event_time"])
			require.Equal(t, metaEventID(l.ID, MetaEventPurchase), ev["event_id"])
		}
	}
	require.ElementsMatch(t, []string{MetaEventLead, MetaEventCompleteRegistration, MetaEventInitiateCheckout, MetaEventAddPaymentInfo, MetaEventPurchase}, names)

	// The next tick finds nothing new: each funnel event is sent once.
	res, err = svc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, MetaCAPITickResult{}, res)

	st, err := svc.Status(t.Context())
	require.NoError(t, err)
	require.True(t, st.Enabled)
	require.True(t, st.TestMode)
	require.Equal(t, "••••4321", st.DatasetID)
	require.EqualValues(t, 5, st.Counts["sent"])
	require.Len(t, st.Events, 5)
}

func TestMetaCAPIRetriesThenFails(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("c2", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "C2", "body": "halo"}))
	m := &fakeMeta{fail: true}
	svc := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "1", Token: "tok"})

	for i := 0; i < metaCAPIMaxAttempts; i++ {
		res, err := svc.Tick(t.Context())
		require.NoError(t, err)
		require.Equal(t, 1, res.Failed, "attempt %d", i+1)
		f.clock = f.clock.Add(2 * time.Hour) // past every backoff
	}
	st, err := svc.Status(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 1, st.Counts["failed"])
	require.Contains(t, st.Events[0].LastError, "kode 100")

	m.fail = false
	n, err := svc.RetryFailed(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	res, err := svc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, res.Sent)
}

func TestMetaCAPISkipsOldAndDoNotContact(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("c3", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "C3", "body": "halo"}))
	m := &fakeMeta{fail: true}
	svc := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "1", Token: "tok"})
	res, err := svc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, res.Queued)

	require.Len(t, m.bodies, 1, "first attempt failed")

	// A week later the queued Lead event is too old for Meta: skipped, not sent.
	f.clock = f.clock.Add(8 * 24 * time.Hour)
	m.fail = false
	res, err = svc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, MetaCAPITickResult{Failed: 1}, res)
	require.Len(t, m.bodies, 1, "no new request to Meta")
	st, err := svc.Status(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 1, st.Counts["skipped"])

	// A lead that asked not to be contacted is never queued.
	other := leadEvent("c4", "message.in", "2026-11-01T03:00:00Z", map[string]string{"wa_message_id": "C4", "body": "halo"})
	other.ChatID, other.Phone = "6285555555555@s.whatsapp.net", "6285555555555"
	f.ingest(t, other)
	_, err = f.db.Exec(`UPDATE leads SET status = 'do_not_contact' WHERE wa_chat_id = ?`, other.ChatID)
	require.NoError(t, err)
	res, err = svc.Tick(t.Context())
	require.NoError(t, err)
	require.Zero(t, res.Queued)
}

func TestMetaCAPIResendTestEventsToProduction(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("t1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "T1", "body": "halo"}))
	other := leadEvent("t2", "message.in", "2026-10-15T03:00:00Z", map[string]string{"wa_message_id": "T2", "body": "halo"})
	other.ChatID, other.Phone = "6287777777777@s.whatsapp.net", "6287777777777"

	// Sent while test mode was on.
	m := &fakeMeta{}
	testSvc := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "1", Token: "tok", TestEventCode: "TEST12345"})
	res, err := testSvc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, res.Sent)
	require.Equal(t, "TEST12345", m.bodies[0]["test_event_code"])
	_, err = testSvc.ResendTestEvents(t.Context())
	require.ErrorIs(t, err, ErrMetaCAPITestModeOn, "refused while test mode is on")

	// An old test-mode event (beyond Meta's 7 days) cannot be rescued.
	f.ingest(t, other)
	l2, err := f.querier.GetLeadByChatID(t.Context(), other.ChatID)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO meta_capi_events (lead_id, event_name, event_id, event_time, status, attempts, test_mode, next_attempt_at, sent_at, created_at)
		VALUES (?, 'Lead', ?, ?, 'sent', 1, 1, ?, ?, ?)`, l2.ID, metaEventID(l2.ID, MetaEventLead), f.clock.Add(-9*24*time.Hour), f.clock, f.clock, f.clock)
	require.NoError(t, err)

	// After restart without the test code: the page offers the resend.
	prod := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "1", Token: "tok"})
	st, err := prod.Status(t.Context())
	require.NoError(t, err)
	require.False(t, st.TestMode)
	require.EqualValues(t, 1, st.ResendableTest)

	// A plain tick sends nothing: the event already exists.
	res, err = prod.Tick(t.Context())
	require.NoError(t, err)
	require.Zero(t, res.Sent)

	n, err := prod.ResendTestEvents(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	res, err = prod.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, res.Sent)
	last := m.bodies[len(m.bodies)-1]
	require.NotContains(t, last, "test_event_code", "sent to production")
	ev := last["data"].([]any)[0].(map[string]any)
	require.Equal(t, metaEventID(f.lead(t).ID, MetaEventLead), ev["event_id"], "same event_id")

	st, err = prod.Status(t.Context())
	require.NoError(t, err)
	require.Zero(t, st.ResendableTest)
	for _, e := range st.Events {
		if e.LeadID == f.lead(t).ID {
			require.False(t, e.TestMode, "no longer labelled uji")
		}
	}
	n, err = prod.ResendTestEvents(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "nothing left to resend; the old one stays as is")
}

func TestMetaCAPIEventSourceURLAndValue(t *testing.T) {
	f := setupLeadFixture(t)
	reguler := sourceIDByName(t, f, "LP Kelas Reguler Tahsin")
	require.ErrorContains(t, f.service.UpdateSource(t.Context(), reguler, models.LeadSourceRequest{Nama: "LP Kelas Reguler Tahsin", URL: "ruangsanad.com/tahsin", IsAktif: true}), "URL LP harus lengkap")
	require.NoError(t, f.service.UpdateSource(t.Context(), reguler, models.LeadSourceRequest{Nama: "LP Kelas Reguler Tahsin", URL: " https://ruangsanad.com/tahsin ", IsAktif: true}))

	// Lead A comes from the reguler LP; lead B has no source.
	f.ingest(t, leadEvent("u1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "U1", "body": "Assalamu'alaikum, saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad."}))
	b := leadEvent("u2", "message.in", "2026-10-24T03:01:00Z", map[string]string{"wa_message_id": "U2", "body": "halo"})
	b.ChatID, b.Phone = "6286666666666@s.whatsapp.net", "6286666666666"
	f.ingest(t, b)
	_, err := f.db.Exec(`UPDATE leads SET payment_amount = 250000 WHERE wa_chat_id = ?`, b.ChatID)
	require.NoError(t, err)

	m := &fakeMeta{}
	svc := newTestCAPI(t, f, m, MetaCAPIConfig{DatasetID: "1", Token: "tok", EventSourceURL: "https://ruangsanad.com"})
	res, err := svc.Tick(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, res.Sent)
	got := map[string]map[string]any{}
	for _, body := range m.bodies {
		ev := body["data"].([]any)[0].(map[string]any)
		got[ev["event_id"].(string)] = ev
	}
	a, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	lb, err := f.querier.GetLeadByChatID(t.Context(), b.ChatID)
	require.NoError(t, err)
	evA, evB := got[metaEventID(a.ID, MetaEventLead)], got[metaEventID(lb.ID, MetaEventLead)]
	require.Equal(t, "https://ruangsanad.com/tahsin", evA["event_source_url"], "the lead source's own landing page")
	require.Equal(t, "https://ruangsanad.com", evB["event_source_url"], "default for leads without a source URL")
	cdA, cdB := evA["custom_data"].(map[string]any), evB["custom_data"].(map[string]any)
	require.Equal(t, "IDR", cdA["currency"])
	require.EqualValues(t, 0, cdA["value"], "no bill yet: value 0 with a valid currency")
	require.EqualValues(t, 250000, cdB["value"], "the lead's bill when known")

	st, err := svc.Status(t.Context())
	require.NoError(t, err)
	require.Equal(t, "https://ruangsanad.com", st.EventSourceURL)
	require.Contains(t, st.SourcesWithoutURL, "Iklan Meta (teks default)")
	require.NotContains(t, st.SourcesWithoutURL, "LP Kelas Reguler Tahsin")

	// Without any URL the field is left out rather than sent empty.
	noURL := newTestCAPI(t, f, &fakeMeta{}, MetaCAPIConfig{DatasetID: "1", Token: "tok"})
	ev := noURL.eventPayload(queries.MetaCapiEvent{EventName: MetaEventLead, EventID: "x"}, lb, "")
	require.NotContains(t, ev, "event_source_url")
}

func TestMetaCAPIAnalyticsSeparatesProductionTestAndPurchaseValue(t *testing.T) {
 f := setupLeadFixture(t)
 f.clock = time.Date(2026,11,1,12,0,0,0,time.UTC)
 f.ingest(t, leadEvent("audit-1","message.in","2026-10-31T18:20:00Z",map[string]string{"wa_message_id":"A1","body":"Halo"}))
 lead := f.lead(t)
 created := f.clock
 seed := func(name,id,stamp,status string,testMode int,value int64) {
  t.Helper()
  moment,err := time.Parse(time.RFC3339,stamp)
  require.NoError(t,err)
  _,err=f.db.Exec(`INSERT INTO meta_capi_events
   (lead_id,event_name,event_id,event_time,value,status,test_mode,next_attempt_at,created_at,fbtrace_id)
   VALUES(?,?,?,?,?,?,?,?,?,?)`,
   lead.ID,name,id,moment,value,status,testMode,created,created,"trace-"+id)
  require.NoError(t,err)
 }
 // 18:20 UTC on Oct 31 belongs to November 1 in WIB.
 seed("Purchase","purchase-prod","2026-10-31T18:20:00Z","sent",0,450000)
 seed("Lead","lead-test","2026-11-01T01:00:00Z","sent",1,0)
 seed("Lead","lead-failed","2026-11-01T02:00:00Z","failed",0,0)
 seed("Lead","lead-pending","2026-11-01T03:00:00Z","pending",0,0)
 seed("CompleteRegistration","registration-skipped","2026-11-01T04:00:00Z","skipped",0,0)
 seed("Purchase","purchase-old","2026-10-29T12:00:00Z","sent",0,900000)

 svc := NewMetaCAPIService(f.querier,MetaCAPIConfig{DatasetID:"123",Token:"token"})
 svc.now=func()time.Time{return f.clock}
 report,err:=svc.Analytics(t.Context(),"2026-11-01","2026-11-01")
 require.NoError(t,err)
 require.Equal(t,int64(5),report.Total)
 require.Equal(t,int64(1),report.Sent)
 require.Equal(t,int64(1),report.TestSent)
 require.Equal(t,int64(1),report.Failed)
 require.Equal(t,int64(1),report.Pending)
 require.Equal(t,int64(1),report.Skipped)
 require.Equal(t,int64(450000),report.PurchaseValue)
 require.Len(t,report.Days,1)
 require.Equal(t,"2026-11-01",report.Days[0].Day)
 require.Equal(t,int64(1),report.Days[0].Sent)
 require.Equal(t,int64(1),report.Days[0].TestSent)

 st,err:=svc.Status(t.Context())
 require.NoError(t,err)
 require.Len(t,st.Events,6)
 require.NotEmpty(t,st.Events[0].EventID)
 require.NotEmpty(t,st.Events[0].FBTraceID)
 require.Equal(t,int64(2),st.Counts["sent"])
 // Historical totals remain available, separate from the selected period.
 _,err=svc.Analytics(t.Context(),"2026-11-02","2026-11-01")
 require.Error(t,err)
 _,err=svc.Analytics(t.Context(),"2026-07-01","2026-11-01")
 require.Error(t,err)
}
