package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// MetaAdsService reads ad performance from the Meta Marketing API (Insights,
// read-only) into meta_ad_insights_daily, and reports it next to CRM leads.
//
// A sync pulls the last 28 days per ad per day and replaces that window,
// because Meta keeps revising recent numbers. It runs daily (after 02.00 WIB)
// and on demand. See docs/panduan/meta-ads-api.md, Langkah 1–11.
type MetaAdsService struct {
	querier *queries.Querier
	cfg     MetaAdsConfig
	client  *http.Client
	baseURL string
	now     func() time.Time
	running sync.Mutex
}

// MetaAdsConfig comes from the environment; syncing is off until a token and
// at least one ad account are set.
type MetaAdsConfig struct {
	Token      string
	AccountIDs []string
	APIVersion string
	AppSecret  string
}

const (
	metaAdsSyncDays   = 28
	metaAdsMaxPages   = 100
	metaAdsMessaging  = "onsite_conversion.messaging_conversation_started_7d"
	metaAdsReportDays = 30
)

var (
	ErrMetaAdsDisabled = errors.New("Meta Ads belum dikonfigurasi (META_ADS_TOKEN dan META_AD_ACCOUNT_IDS)")
	ErrMetaAdsBusy     = errors.New("sinkron data iklan sedang berjalan; coba lagi sebentar")
)

func NewMetaAdsService(q *queries.Querier, cfg MetaAdsConfig) *MetaAdsService {
	if cfg.APIVersion == "" {
		cfg.APIVersion = metaCAPIDefaultVer
	}
	ids := make([]string, 0, len(cfg.AccountIDs))
	for _, id := range cfg.AccountIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if !strings.HasPrefix(id, "act_") {
			id = "act_" + id
		}
		ids = append(ids, id)
	}
	cfg.AccountIDs = ids
	return &MetaAdsService{
		querier: q, cfg: cfg, client: &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://graph.facebook.com", now: time.Now,
	}
}

func (s *MetaAdsService) Enabled() bool { return s.cfg.Token != "" && len(s.cfg.AccountIDs) > 0 }

func (s *MetaAdsService) nowUTC() time.Time { return s.now().UTC().Truncate(time.Second) }

// DueForDailySync is true once a day after 02.00 WIB: when there is no
// successful sync in the last 20 hours.
func (s *MetaAdsService) DueForDailySync(ctx context.Context) (bool, error) {
	if !s.Enabled() || s.now().In(wib).Hour() < 2 {
		return false, nil
	}
	last, err := s.querier.GetLastSuccessfulMetaAdsSyncRun(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return last.StartedAt.Before(s.nowUTC().Add(-20 * time.Hour)), nil
}

// insightRow is one ad on one day from GET /{act}/insights.
type insightRow struct {
	DateStart        string `json:"date_start"`
	CampaignID       string `json:"campaign_id"`
	CampaignName     string `json:"campaign_name"`
	AdsetID          string `json:"adset_id"`
	AdsetName        string `json:"adset_name"`
	AdID             string `json:"ad_id"`
	AdName           string `json:"ad_name"`
	Spend            string `json:"spend"`
	Impressions      string `json:"impressions"`
	Reach            string `json:"reach"`
	Clicks           string `json:"clicks"`
	InlineLinkClicks string `json:"inline_link_clicks"`
	Actions          []struct {
		ActionType string `json:"action_type"`
		Value      string `json:"value"`
	} `json:"actions"`
}

type insightPage struct {
	Data   []insightRow `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
		Subcode int    `json:"error_subcode"`
	} `json:"error"`
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// rupiah turns Meta's decimal spend string ("125000.50") into whole rupiah.
func rupiah(s string) int64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(f))
}

func (s *MetaAdsService) appSecretProof() string {
	if s.cfg.AppSecret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.AppSecret))
	mac.Write([]byte(s.cfg.Token))
	return hex.EncodeToString(mac.Sum(nil))
}

// fetchAccount pulls every insight row of one ad account for the range,
// following paging.next.
func (s *MetaAdsService) fetchAccount(ctx context.Context, account, from, to string) ([]insightRow, error) {
	q := url.Values{}
	q.Set("level", "ad")
	q.Set("time_increment", "1")
	q.Set("time_range", fmt.Sprintf(`{"since":"%s","until":"%s"}`, from, to))
	q.Set("fields", "campaign_id,campaign_name,adset_id,adset_name,ad_id,ad_name,spend,impressions,reach,clicks,inline_link_clicks,actions")
	q.Set("limit", "500")
	q.Set("access_token", s.cfg.Token)
	if proof := s.appSecretProof(); proof != "" {
		q.Set("appsecret_proof", proof)
	}
	next := fmt.Sprintf("%s/%s/%s/insights?%s", s.baseURL, s.cfg.APIVersion, account, q.Encode())
	var out []insightRow
	for page := 0; next != ""; page++ {
		if page >= metaAdsMaxPages {
			return nil, fmt.Errorf("terlalu banyak halaman data untuk %s; persempit rentang", account)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			return nil, err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("koneksi ke Meta gagal: %w", err)
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		var p insightPage
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("respons Meta tidak terbaca (HTTP %d)", resp.StatusCode)
		}
		if p.Error != nil {
			return nil, fmt.Errorf("Meta menolak untuk %s (kode %d/%d): %s", account, p.Error.Code, p.Error.Subcode, p.Error.Message)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("respons Meta tidak terduga (HTTP %d)", resp.StatusCode)
		}
		out = append(out, p.Data...)
		next = p.Paging.Next
	}
	return out, nil
}

// Sync pulls the last 28 days for every ad account and replaces them. It
// records the run, also when it fails, so the page can show the error.
func (s *MetaAdsService) Sync(ctx context.Context, trigger string) (int, error) {
	if !s.Enabled() {
		return 0, ErrMetaAdsDisabled
	}
	if !s.running.TryLock() {
		return 0, ErrMetaAdsBusy
	}
	defer s.running.Unlock()

	today := s.now().In(wib)
	to := today.Format(time.DateOnly)
	from := today.AddDate(0, 0, -(metaAdsSyncDays - 1)).Format(time.DateOnly)
	runID, err := s.querier.CreateMetaAdsSyncRun(ctx, queries.CreateMetaAdsSyncRunParams{
		Trigger: trigger, DateFrom: from, DateTo: to, StartedAt: s.nowUTC(),
	})
	if err != nil {
		return 0, err
	}
	rows, syncErr := s.syncRange(ctx, from, to)
	status, msg := "success", ""
	if syncErr != nil {
		status, msg = "failed", truncateRunes(syncErr.Error(), 500)
	}
	if err := s.querier.FinishMetaAdsSyncRun(ctx, queries.FinishMetaAdsSyncRunParams{
		Status: status, Rows: int64(rows), Error: msg, FinishedAt: sql.NullTime{Time: s.nowUTC(), Valid: true}, ID: runID,
	}); err != nil {
		return rows, err
	}
	return rows, syncErr
}

func (s *MetaAdsService) syncRange(ctx context.Context, from, to string) (int, error) {
	// Fetch everything first: a failure on any account leaves stored data as is.
	fetched := map[string][]insightRow{}
	for _, account := range s.cfg.AccountIDs {
		rows, err := s.fetchAccount(ctx, account, from, to)
		if err != nil {
			return 0, err
		}
		fetched[account] = rows
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)
	now := s.nowUTC()
	total := 0
	for account, rows := range fetched {
		if err := q.DeleteMetaAdInsightsRange(ctx, queries.DeleteMetaAdInsightsRangeParams{AdAccountID: account, Tanggal: from, Tanggal_2: to}); err != nil {
			return 0, err
		}
		for _, r := range rows {
			if r.AdID == "" || r.DateStart == "" {
				continue
			}
			var messaging, linkClicks int64
			for _, a := range r.Actions {
				switch a.ActionType {
				case metaAdsMessaging:
					messaging = atoi64(a.Value)
				case "link_click":
					linkClicks = atoi64(a.Value)
				}
			}
			if v := atoi64(r.InlineLinkClicks); v > 0 {
				linkClicks = v
			}
			if err := q.UpsertMetaAdInsight(ctx, queries.UpsertMetaAdInsightParams{
				Tanggal: r.DateStart, AdAccountID: account, CampaignID: r.CampaignID, CampaignName: r.CampaignName,
				AdsetID: r.AdsetID, AdsetName: r.AdsetName, AdID: r.AdID, AdName: r.AdName,
				Spend: rupiah(r.Spend), Impressions: atoi64(r.Impressions), Reach: atoi64(r.Reach), Clicks: atoi64(r.Clicks),
				LinkClicks: linkClicks, MessagingStarted: messaging, SyncedAt: now,
			}); err != nil {
				return 0, err
			}
			total++
		}
	}
	return total, tx.Commit()
}

// SetCampaignSource maps a campaign to a lead source; sourceID 0 removes it.
func (s *MetaAdsService) SetCampaignSource(ctx context.Context, campaignID string, sourceID int64) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || sourceID < 0 {
		return errors.New("data tidak valid")
	}
	if sourceID == 0 {
		return s.querier.DeleteLeadSourceMetaCampaign(ctx, campaignID)
	}
	if _, err := s.querier.GetLeadSource(ctx, sourceID); errors.Is(err, sql.ErrNoRows) {
		return ErrLeadSourceNotFound
	} else if err != nil {
		return err
	}
	return s.querier.SetLeadSourceMetaCampaign(ctx, queries.SetLeadSourceMetaCampaignParams{
		CampaignID: campaignID, LeadSourceID: sourceID, CreatedAt: s.nowUTC(),
	})
}

// reportRange validates an inclusive WIB date range; default is the last 30 days.
func (s *MetaAdsService) reportRange(from, to string) (time.Time, time.Time, error) {
	today := s.now().In(wib)
	end := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, wib)
	start := end.AddDate(0, 0, -(metaAdsReportDays - 1))
	if from != "" || to != "" {
		var ok bool
		if start, ok = wibDayStart(from); !ok {
			return start, end, errors.New("tanggal mulai tidak valid")
		}
		if end, ok = wibDayStart(to); !ok {
			return start, end, errors.New("tanggal akhir tidak valid")
		}
	}
	if start.After(end) || end.Sub(start) > 366*24*time.Hour {
		return start, end, errors.New("rentang tanggal harus 1 sampai 366 hari")
	}
	return start, end, nil
}

func syncInfo(r queries.MetaAdsSyncRun) *models.MetaAdsSyncInfo {
	return &models.MetaAdsSyncInfo{
		Trigger: r.Trigger, Status: r.Status, Rows: r.Rows, Error: r.Error, DateFrom: r.DateFrom, DateTo: r.DateTo,
		StartedAt: r.StartedAt.UTC().Format(time.RFC3339), FinishedAt: formatLeadNullTime(r.FinishedAt),
	}
}

// Report returns the raw data of the Performa Iklan dashboard for a period
// and the period of the same length right before it.
func (s *MetaAdsService) Report(ctx context.Context, from, to string) (*models.MetaAdsReport, error) {
	start, end, err := s.reportRange(from, to)
	if err != nil {
		return nil, err
	}
	days := int(end.Sub(start).Hours()/24) + 1
	prevStart, prevEnd := start.AddDate(0, 0, -days), start.AddDate(0, 0, -1)
	rep := &models.MetaAdsReport{
		From: start.Format(time.DateOnly), To: end.Format(time.DateOnly),
		PrevFrom: prevStart.Format(time.DateOnly), PrevTo: prevEnd.Format(time.DateOnly),
		Enabled: s.Enabled(), Accounts: make([]string, 0, len(s.cfg.AccountIDs)),
		Rows: []models.MetaAdsRow{}, Ads: []models.MetaAdsAdInfo{}, Campaigns: []models.MetaAdsCampaignInfo{},
		LeadDays: []models.MetaAdsLeadDay{}, Sources: []models.LeadSourceOption{},
	}
	for _, a := range s.cfg.AccountIDs {
		rep.Accounts = append(rep.Accounts, "act_"+maskID(strings.TrimPrefix(a, "act_")))
	}
	if last, err := s.querier.GetLastMetaAdsSyncRun(ctx); err == nil {
		rep.LastRun = syncInfo(last)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if last, err := s.querier.GetLastSuccessfulMetaAdsSyncRun(ctx); err == nil {
		rep.LastSuccess = syncInfo(last)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	sources, err := s.querier.ListLeadSources(ctx)
	if err != nil {
		return nil, err
	}
	for _, src := range sources {
		rep.Sources = append(rep.Sources, models.LeadSourceOption{ID: src.ID, Nama: src.Nama, IsAktif: src.IsAktif == 1})
	}
	mapping, err := s.querier.ListLeadSourceMetaCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	campaignSource := map[string]int64{}
	for _, m := range mapping {
		campaignSource[m.CampaignID] = m.LeadSourceID
	}
	campaigns, err := s.querier.ListMetaCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range campaigns {
		rep.Campaigns = append(rep.Campaigns, models.MetaAdsCampaignInfo{ID: c.CampaignID, Name: c.CampaignName, LastDate: c.LastDate, SourceID: campaignSource[c.CampaignID]})
	}

	f, t := rep.PrevFrom, rep.To
	rows, err := s.querier.MetaAdsRows(ctx, queries.MetaAdsRowsParams{Tanggal: f, Tanggal_2: t})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		rep.Rows = append(rep.Rows, models.MetaAdsRow{
			Day: r.Tanggal, AdID: r.AdID, AdsetID: r.AdsetID, CampaignID: r.CampaignID, Spend: r.Spend,
			Impressions: r.Impressions, Clicks: r.Clicks, LinkClicks: r.LinkClicks, Messaging: r.MessagingStarted,
		})
	}
	names, err := s.querier.MetaAdNames(ctx, queries.MetaAdNamesParams{Tanggal: f, Tanggal_2: t})
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		rep.Ads = append(rep.Ads, models.MetaAdsAdInfo{ID: n.AdID, Name: n.AdName, AdsetID: n.AdsetID, AdsetName: n.AdsetName, CampaignID: n.CampaignID})
	}

	// CRM leads that came in during both periods, per WIB day and source.
	leads, err := s.querier.ListLeadSourceRows(ctx, queries.ListLeadSourceRowsParams{CreatedAt: prevStart.UTC(), CreatedAt_2: end.AddDate(0, 0, 1).UTC()})
	if err != nil {
		return nil, err
	}
	type key struct {
		day string
		src int64
	}
	counts := map[key]*models.MetaAdsLeadDay{}
	var order []key
	for _, l := range leads {
		k := key{l.CreatedAt.In(wib).Format(time.DateOnly), l.LeadSourceID.Int64}
		c := counts[k]
		if c == nil {
			c = &models.MetaAdsLeadDay{Day: k.day, SourceID: k.src}
			counts[k] = c
			order = append(order, k)
		}
		c.Leads++
		if l.Status == LeadStatusWon {
			c.Won++
		}
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].day != order[j].day {
			return order[i].day < order[j].day
		}
		return order[i].src < order[j].src
	})
	for _, k := range order {
		rep.LeadDays = append(rep.LeadDays, *counts[k])
	}
	return rep, nil
}
