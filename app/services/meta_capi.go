package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// MetaCAPIService sends the CRM funnel of WhatsApp leads to Meta through the
// Conversions API, so ads can be measured (and optimized) up to the moment a
// calon santri becomes a santri.
//
// It does not hook into every code path that moves a lead. Instead Tick
// reconciles: for every lead changed in the last 7 days it derives which
// funnel milestones were reached and when, and queues one event per lead per
// milestone (the event_id is unique, so nothing is ever sent twice). Queued
// events are then sent one by one with retries.
//
// Personal data is only sent hashed (SHA-256), as Meta requires.
type MetaCAPIService struct {
	querier *queries.Querier
	cfg     MetaCAPIConfig
	client  *http.Client
	baseURL string
	now     func() time.Time
}

// MetaCAPIConfig comes from the environment. Sending is off until both the
// dataset (Pixel) ID and the token are set.
type MetaCAPIConfig struct {
	DatasetID     string
	Token         string
	APIVersion    string
	AppSecret     string
	TestEventCode string
	// EventSourceURL is the landing page sent as event_source_url when the
	// lead's source has no URL of its own. Meta blocks events without one
	// for businesses in restricted categories.
	EventSourceURL string
}

const (
	// Meta rejects server events older than 7 days.
	metaCAPIMaxAge      = 7 * 24 * time.Hour
	metaCAPIMaxAttempts = 5
	metaCAPIBatch       = 30
	metaCAPIDefaultVer  = "v23.0"
)

// Funnel events, in order. The first three CRM stages map to standard events;
// Lead is the first chat and Purchase is becoming a santri.
const (
	MetaEventLead                 = "Lead"
	MetaEventCompleteRegistration = "CompleteRegistration"
	MetaEventInitiateCheckout     = "InitiateCheckout"
	MetaEventAddPaymentInfo       = "AddPaymentInfo"
	MetaEventPurchase             = "Purchase"
)

// MetaCAPIStageEvents maps the CRM stage that triggers each mid-funnel event.
var MetaCAPIStageEvents = []struct{ Event, Stage string }{
	{MetaEventCompleteRegistration, LeadStageDataTerisi},
	{MetaEventInitiateCheckout, LeadStageInfoBayarDikirim},
	{MetaEventAddPaymentInfo, LeadStageBuktiMasuk},
}

func NewMetaCAPIService(q *queries.Querier, cfg MetaCAPIConfig) *MetaCAPIService {
	if cfg.APIVersion == "" {
		cfg.APIVersion = metaCAPIDefaultVer
	}
	return &MetaCAPIService{
		querier: q, cfg: cfg, client: &http.Client{Timeout: 20 * time.Second},
		baseURL: "https://graph.facebook.com", now: time.Now,
	}
}

func (s *MetaCAPIService) Enabled() bool { return s.cfg.DatasetID != "" && s.cfg.Token != "" }

func (s *MetaCAPIService) nowUTC() time.Time { return s.now().UTC().Truncate(time.Second) }

type MetaCAPITickResult struct{ Queued, Sent, Failed int }

// Tick queues newly reached milestones, then sends due events.
func (s *MetaCAPIService) Tick(ctx context.Context) (MetaCAPITickResult, error) {
	var res MetaCAPITickResult
	if !s.Enabled() {
		return res, nil
	}
	var err error
	if res.Queued, err = s.reconcile(ctx); err != nil {
		return res, err
	}
	due, err := s.querier.ListDueMetaCAPIEvents(ctx, queries.ListDueMetaCAPIEventsParams{Now: s.nowUTC(), Lim: metaCAPIBatch})
	if err != nil {
		return res, err
	}
	for _, ev := range due {
		sent, err := s.sendOne(ctx, ev)
		if err != nil {
			return res, err
		}
		if sent {
			res.Sent++
		} else {
			res.Failed++
		}
	}
	return res, nil
}

type metaMilestone struct {
	Event string
	At    time.Time
	Value int64
}

// leadMilestones derives the funnel events a lead has reached, with the time
// each was first reached. stageTimes holds the first time the lead moved to
// each stage. A milestone whose time is unknown is left out.
func leadMilestones(l queries.ListMetaCAPICandidatesRow, stageTimes map[string]time.Time) []metaMilestone {
	out := []metaMilestone{{Event: MetaEventLead, At: l.CreatedAt}}
	reached := func(stage string) time.Time {
		min := leadStageIndex(stage)
		var first time.Time
		for st, at := range stageTimes {
			if leadStageIndex(st) >= min && (first.IsZero() || at.Before(first)) {
				first = at
			}
		}
		return first
	}
	earliest := func(times ...time.Time) time.Time {
		var first time.Time
		for _, t := range times {
			if !t.IsZero() && (first.IsZero() || t.Before(first)) {
				first = t
			}
		}
		return first
	}
	for _, m := range MetaCAPIStageEvents {
		at := reached(m.Stage)
		switch m.Event {
		case MetaEventInitiateCheckout:
			at = earliest(at, l.PaymentInfoSentAt.Time)
		case MetaEventAddPaymentInfo:
			at = earliest(at, l.ProofReceivedAt.Time)
		}
		if !at.IsZero() {
			out = append(out, metaMilestone{Event: m.Event, At: at})
		}
	}
	if l.Status == LeadStatusWon && l.ConvertedAt.Valid {
		// The amount staff verified; the expected bill when none was recorded.
		value := l.VerifiedAmount
		if value <= 0 {
			value = l.PaymentAmount
		}
		out = append(out, metaMilestone{Event: MetaEventPurchase, At: l.ConvertedAt.Time, Value: value})
	}
	return out
}

func metaEventID(leadID int64, event string) string {
	return fmt.Sprintf("rs-lead-%d-%s", leadID, strings.ToLower(event))
}

func (s *MetaCAPIService) reconcile(ctx context.Context) (int, error) {
	now := s.nowUTC()
	since := now.Add(-metaCAPIMaxAge)
	leads, err := s.querier.ListMetaCAPICandidates(ctx, since)
	if err != nil {
		return 0, err
	}
	rows, err := s.querier.ListLeadStageReachedTimes(ctx, since)
	if err != nil {
		return 0, err
	}
	stageTimes := map[int64]map[string]time.Time{}
	for _, r := range rows {
		at, ok := queries.ParseStoredTime(r.ReachedAt)
		if !ok {
			continue
		}
		if stageTimes[r.LeadID] == nil {
			stageTimes[r.LeadID] = map[string]time.Time{}
		}
		stageTimes[r.LeadID][r.StageTo] = at
	}
	queued := 0
	for _, l := range leads {
		for _, m := range leadMilestones(l, stageTimes[l.ID]) {
			if m.At.Before(since) || m.At.After(now.Add(time.Hour)) {
				continue
			}
			n, err := s.querier.InsertMetaCAPIEvent(ctx, queries.InsertMetaCAPIEventParams{
				LeadID: l.ID, EventName: m.Event, EventID: metaEventID(l.ID, m.Event), EventTime: m.At.UTC(),
				Value: m.Value, TestMode: boolInt(s.cfg.TestEventCode != ""), NextAttemptAt: now, CreatedAt: now,
			})
			if err != nil {
				return queued, err
			}
			queued += int(n)
		}
	}
	return queued, nil
}

// metaHash returns the SHA-256 hex Meta expects; callers normalize first.
func metaHash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// metaLetters keeps lowercase letters only ("Jakarta Selatan" → "jakartaselatan").
func metaLetters(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// metaUserData builds the hashed customer information Meta uses to match the
// event to an account. Raw values never leave the server.
func metaUserData(l queries.Lead) map[string]any {
	ud := map[string]any{
		"external_id": []string{metaHash(fmt.Sprintf("ruangsanad-lead-%d", l.ID))},
		"country":     []string{metaHash("id")},
	}
	if ph := NormalizeLeadPhone(l.Phone); ph != "" {
		ud["ph"] = []string{metaHash(ph)}
	}
	if words := strings.Fields(NormalizeLeadSourceText(l.Nama)); len(words) > 0 {
		ud["fn"] = []string{metaHash(metaLetters(words[0]))}
		if len(words) > 1 {
			ud["ln"] = []string{metaHash(metaLetters(words[len(words)-1]))}
		}
	}
	if city := metaLetters(strings.SplitN(l.Domisili, ",", 2)[0]); city != "" {
		ud["ct"] = []string{metaHash(city)}
	}
	switch l.JenisKelamin {
	case "L":
		ud["ge"] = []string{metaHash("m")}
	case "P":
		ud["ge"] = []string{metaHash("f")}
	}
	return ud
}

// eventPayload builds one server event. Every event carries value + IDR
// (Meta flags events without a valid currency): Purchase uses the verified
// amount, earlier steps the lead's bill when known, otherwise 0.
func (s *MetaCAPIService) eventPayload(ev queries.MetaCapiEvent, l queries.Lead, sourceURL string) map[string]any {
	value := l.PaymentAmount
	if ev.EventName == MetaEventPurchase {
		value = ev.Value
	}
	custom := map[string]any{
		"currency":          "IDR",
		"value":             max(value, 0),
		"lead_event_source": "Ruang Sanad Dashboard",
		"event_source":      "crm",
	}
	out := map[string]any{
		"event_name":    ev.EventName,
		"event_time":    ev.EventTime.Unix(),
		"event_id":      ev.EventID,
		"action_source": "system_generated",
		"user_data":     metaUserData(l),
		"custom_data":   custom,
	}
	if sourceURL != "" {
		out["event_source_url"] = sourceURL
	}
	return out
}

type metaAPIError struct {
	Error *struct {
		Message   string `json:"message"`
		Code      int    `json:"code"`
		Subcode   int    `json:"error_subcode"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
	EventsReceived int    `json:"events_received"`
	FBTraceID      string `json:"fbtrace_id"`
}

// sendOne sends one queued event. It returns whether it was sent; a failed
// attempt is retried later (backoff) up to metaCAPIMaxAttempts.
func (s *MetaCAPIService) sendOne(ctx context.Context, ev queries.MetaCapiEvent) (bool, error) {
	now := s.nowUTC()
	if ev.EventTime.Before(now.Add(-metaCAPIMaxAge)) {
		return false, s.querier.MarkMetaCAPIEventDone(ctx, queries.MarkMetaCAPIEventDoneParams{
			Status: "skipped", Attempts: 0, LastError: "lebih dari 7 hari; Meta tidak menerima event selama itu", ID: ev.ID,
		})
	}
	lead, err := s.querier.GetLeadByID(ctx, ev.LeadID)
	if err != nil {
		return false, err
	}
	if lead.Status == LeadStatusDoNotContact {
		return false, s.querier.MarkMetaCAPIEventDone(ctx, queries.MarkMetaCAPIEventDoneParams{
			Status: "skipped", Attempts: 0, LastError: "lead minta tidak dihubungi", ID: ev.ID,
		})
	}
	sourceURL := s.cfg.EventSourceURL
	if lead.LeadSourceID.Valid {
		if src, err := s.querier.GetLeadSource(ctx, lead.LeadSourceID.Int64); err == nil && src.Url != "" {
			sourceURL = src.Url
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	traceID, sendErr := s.post(ctx, s.eventPayload(ev, lead, sourceURL))
	if sendErr == nil {
		return true, s.querier.MarkMetaCAPIEventSent(ctx, queries.MarkMetaCAPIEventSentParams{
			FbtraceID: traceID, SentAt: sql.NullTime{Time: now, Valid: true}, ID: ev.ID,
		})
	}
	msg := truncateRunes(sendErr.Error(), 500)
	if ev.Attempts+1 >= metaCAPIMaxAttempts {
		return false, s.querier.MarkMetaCAPIEventDone(ctx, queries.MarkMetaCAPIEventDoneParams{Status: "failed", Attempts: 1, LastError: msg, ID: ev.ID})
	}
	backoff := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}[min(int(ev.Attempts), 3)]
	return false, s.querier.MarkMetaCAPIEventRetry(ctx, queries.MarkMetaCAPIEventRetryParams{LastError: msg, NextAttemptAt: now.Add(backoff), ID: ev.ID})
}

func (s *MetaCAPIService) post(ctx context.Context, event map[string]any) (string, error) {
	body := map[string]any{"data": []any{event}, "access_token": s.cfg.Token}
	if s.cfg.TestEventCode != "" {
		body["test_event_code"] = s.cfg.TestEventCode
	}
	if s.cfg.AppSecret != "" {
		mac := hmac.New(sha256.New, []byte(s.cfg.AppSecret))
		mac.Write([]byte(s.cfg.Token))
		body["appsecret_proof"] = hex.EncodeToString(mac.Sum(nil))
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/%s/%s/events", s.baseURL, s.cfg.APIVersion, s.cfg.DatasetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("koneksi ke Meta gagal: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var parsed metaAPIError
	_ = json.Unmarshal(data, &parsed)
	if parsed.Error != nil {
		return "", fmt.Errorf("Meta menolak (kode %d/%d): %s", parsed.Error.Code, parsed.Error.Subcode, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || parsed.EventsReceived < 1 {
		return "", fmt.Errorf("respons Meta tidak terduga (HTTP %d): %s", resp.StatusCode, truncateRunes(string(data), 200))
	}
	return parsed.FBTraceID, nil
}

// RetryFailed puts failed events that Meta can still accept back in the queue.
func (s *MetaCAPIService) RetryFailed(ctx context.Context) (int64, error) {
	if !s.Enabled() {
		return 0, ErrMetaCAPIDisabled
	}
	now := s.nowUTC()
	return s.querier.RetryFailedMetaCAPIEvents(ctx, queries.RetryFailedMetaCAPIEventsParams{Now: now, Since: now.Add(-metaCAPIMaxAge)})
}

// resendSince is how far back test-mode events can still go to production:
// Meta's 7 days, minus a margin so an event does not expire while queued.
func (s *MetaCAPIService) resendSince() time.Time {
	return s.nowUTC().Add(-metaCAPIMaxAge + 15*time.Minute)
}

// ErrMetaCAPITestModeOn blocks resending while test mode is still on: the
// events would only reach Test events again.
var ErrMetaCAPITestModeOn = errors.New("mode uji masih aktif (META_CAPI_TEST_EVENT_CODE terisi); kosongkan lalu restart aplikasi sebelum mengirim ulang ke produksi")

// ResendTestEvents queues the events that were sent in test mode (so Meta
// only shows them in Test events) to be sent again to production, as long as
// Meta still accepts them (7 days). The event_id stays the same.
func (s *MetaCAPIService) ResendTestEvents(ctx context.Context) (int64, error) {
	if !s.Enabled() {
		return 0, ErrMetaCAPIDisabled
	}
	if s.cfg.TestEventCode != "" {
		return 0, ErrMetaCAPITestModeOn
	}
	return s.querier.ResendTestMetaCAPIEvents(ctx, queries.ResendTestMetaCAPIEventsParams{Now: s.nowUTC(), Since: s.resendSince()})
}

// Status is what the settings page shows: configuration (without secrets),
// queue counts, and the latest events.
func (s *MetaCAPIService) Status(ctx context.Context) (*models.MetaCAPIStatus, error) {
	out := &models.MetaCAPIStatus{
		Enabled: s.Enabled(), DatasetID: maskID(s.cfg.DatasetID), APIVersion: s.cfg.APIVersion,
		TestMode: s.cfg.TestEventCode != "", AppSecretProof: s.cfg.AppSecret != "",
		Counts: map[string]int64{}, Events: []models.MetaCAPIEventRow{}, Mapping: []models.MetaCAPIMapping{
			{Event: MetaEventLead, Tahap: "Lead masuk (chat pertama)"},
		},
	}
	for _, m := range MetaCAPIStageEvents {
		out.Mapping = append(out.Mapping, models.MetaCAPIMapping{Event: m.Event, Tahap: m.Stage})
	}
	out.Mapping = append(out.Mapping, models.MetaCAPIMapping{Event: MetaEventPurchase, Tahap: "Jadi santri (nilai = nominal yang diverifikasi)"})
	counts, err := s.querier.CountMetaCAPIEventsByStatus(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range counts {
		out.Counts[c.Status] = c.Total
	}
	if out.ResendableTest, err = s.querier.CountResendableTestMetaCAPIEvents(ctx, s.resendSince()); err != nil {
		return nil, err
	}
	out.EventSourceURL = s.cfg.EventSourceURL
	out.SourcesWithoutURL = []string{}
	sources, err := s.querier.ListLeadSources(ctx)
	if err != nil {
		return nil, err
	}
	for _, src := range sources {
		if src.IsAktif == 1 && src.Url == "" {
			out.SourcesWithoutURL = append(out.SourcesWithoutURL, src.Nama)
		}
	}
	rows, err := s.querier.ListMetaCAPIAudit(ctx, 50)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out.Events = append(out.Events, models.MetaCAPIEventRow{
			ID: r.ID, LeadID: r.LeadID, LeadNama: r.LeadName, Event: r.EventName,
			EventID: r.EventID, FBTraceID: r.FBTraceID,
			EventTime: r.EventTime.UTC().Format(time.RFC3339), Value: r.Value, Status: r.Status,
			Attempts: r.Attempts, LastError: r.LastError, TestMode: r.TestMode,
			SentAt: formatLeadNullTime(r.SentAt),
		})
	}
	return out, nil
}

// maskID shows only the last 4 digits of an ID.
func maskID(id string) string {
	if len(id) <= 4 {
		return id
	}
	return strings.Repeat("•", 4) + id[len(id)-4:]
}

// ErrMetaCAPIDisabled is returned by actions that need a configured dataset.
var ErrMetaCAPIDisabled = errors.New("Meta CAPI belum dikonfigurasi (META_CAPI_DATASET_ID dan META_CAPI_TOKEN)")

// Analytics aggregates persisted events in a WIB reporting window.
// Daily/event totals are period-scoped; Status().Counts remains the lifetime queue.
func (s *MetaCAPIService) Analytics(ctx context.Context, from, to string) (*models.MetaCAPIAnalytics,error) {
 today := s.now().In(wib)
 lastDay := time.Date(today.Year(),today.Month(),today.Day(),0,0,0,0,wib)
 firstDay := lastDay.AddDate(0,0,-13)
 if from != "" || to != "" {
  if from == "" || to == "" { return nil, errors.New("tanggal awal dan akhir wajib diisi") }
  var ok bool
  if firstDay,ok = wibDayStart(from); !ok { return nil, errors.New("tanggal awal tidak valid") }
  if lastDay,ok = wibDayStart(to); !ok { return nil, errors.New("tanggal akhir tidak valid") }
 }
 end := lastDay.AddDate(0,0,1)
 if firstDay.After(lastDay) || end.After(firstDay.AddDate(0,0,91)) {
  return nil,errors.New("rentang CAPI harus 1 sampai 90 hari")
 }
 rows,err := s.querier.ListMetaCAPIAnalytics(ctx,firstDay,end)
 if err != nil { return nil,err }
 out := &models.MetaCAPIAnalytics{
  From:firstDay.Format(time.DateOnly),To:lastDay.Format(time.DateOnly),
  Days:[]models.MetaCAPIDay{}, Events:[]models.MetaCAPIEventSummary{},
 }
 daily := map[string]*models.MetaCAPIDay{}
 for d:=firstDay;d.Before(end);d=d.AddDate(0,0,1) {
  day := d.Format(time.DateOnly)
  daily[day] = &models.MetaCAPIDay{Day:day}
 }
 order := []string{MetaEventLead,MetaEventCompleteRegistration,MetaEventInitiateCheckout,MetaEventAddPaymentInfo,MetaEventPurchase}
 byEvent := map[string]*models.MetaCAPIEventSummary{}
 for _,name := range order { byEvent[name] = &models.MetaCAPIEventSummary{Event:name} }
 for _,r := range rows {
  day := daily[r.Day]
  if day == nil { continue }
  by := byEvent[r.Event]
  if by == nil {
   by = &models.MetaCAPIEventSummary{Event:r.Event}
   byEvent[r.Event] = by
   order = append(order,r.Event)
  }
  out.Total += r.Count
  by.Total += r.Count
  switch r.Status {
  case "sent":
   if r.TestMode {
    out.TestSent += r.Count
    day.TestSent += r.Count
    by.TestSent += r.Count
   } else {
    out.Sent += r.Count
    day.Sent += r.Count
    by.Sent += r.Count
    out.PurchaseValue += r.PurchaseValue
    by.PurchaseValue += r.PurchaseValue
   }
  case "pending":
   out.Pending += r.Count; day.Pending += r.Count; by.Pending += r.Count
  case "failed":
   out.Failed += r.Count; day.Failed += r.Count; by.Failed += r.Count
  case "skipped":
   out.Skipped += r.Count; day.Skipped += r.Count; by.Skipped += r.Count
  }
 }
 for d:=firstDay;d.Before(end);d=d.AddDate(0,0,1) {
  out.Days=append(out.Days,*daily[d.Format(time.DateOnly)])
 }
 for _,name:=range order {out.Events=append(out.Events,*byEvent[name])}
 return out,nil
}
