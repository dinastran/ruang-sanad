package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// CRMIntegrationConfig holds the .env values for the WhatsApp integration.
// Secrets are used only for outgoing calls and never returned to the UI.
type CRMIntegrationConfig struct {
	IngestEnabled bool
	BridgeURL     string
	OutboundReady bool // CS_BRIDGE_URL and CS_BRIDGE_OUTBOUND_TOKEN set
	GowaURL       string
	GowaUser      string
	GowaPass      string
}

// CRMIntegrationService backs /app/integrasi-whatsapp (super_admin):
// connection status of GOWA and the bridge, plus crm_settings.
type CRMIntegrationService struct {
	querier *queries.Querier
	cfg     CRMIntegrationConfig
	client  *http.Client
	now     func() time.Time
}

func NewCRMIntegrationService(querier *queries.Querier, cfg CRMIntegrationConfig) *CRMIntegrationService {
	return &CRMIntegrationService{querier: querier, cfg: cfg, client: &http.Client{Timeout: 4 * time.Second}, now: time.Now}
}

func settingsResponse(s queries.CrmSetting) models.CRMSettingsResponse {
	mapping := map[string]string{}
	_ = json.Unmarshal([]byte(s.PaketKeKelasKode), &mapping)
	return models.CRMSettingsResponse{
		PaketKeKelasKode: mapping, FollowupEnabled: s.FollowupEnabled == 1,
		FollowupSendStart: s.FollowupSendStart, FollowupSendEnd: s.FollowupSendEnd,
		FollowupMinGapSeconds: s.FollowupMinGapSeconds, FollowupDailyCap: s.FollowupDailyCap,
		TranscriptRetentionDaysLost: s.TranscriptRetentionDaysLost, PaymentVerifiedTemplate: s.PaymentVerifiedTemplate,
		PersonaBlockedKeywords: s.PersonaBlockedKeywords, FollowupOnlyLabel: s.FollowupOnlyLabel,
	}
}

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
var paketKey = regexp.MustCompile(`^[0-9]{1,2}x$`)

func (s *CRMIntegrationService) UpdateSettings(ctx context.Context, actorID int64, req models.CRMSettingsRequest) error {
	if len(req.PaketKeKelasKode) == 0 || len(req.PaketKeKelasKode) > 10 {
		return errors.New("pemetaan paket wajib diisi (maks 10)")
	}
	mapping := map[string]string{}
	for paket, kode := range req.PaketKeKelasKode {
		paket = strings.ToLower(strings.TrimSpace(paket))
		kode = strings.TrimSpace(kode)
		if !paketKey.MatchString(paket) {
			return fmt.Errorf("kode paket %q tidak valid (contoh: 1x, 2x)", paket)
		}
		if _, err := s.querier.GetKodeKelasByKode(ctx, kode); err != nil {
			return fmt.Errorf("kode kelas %q tidak terdaftar", kode)
		}
		mapping[paket] = kode
	}
	if !hhmm.MatchString(req.FollowupSendStart) || !hhmm.MatchString(req.FollowupSendEnd) || req.FollowupSendStart >= req.FollowupSendEnd {
		return errors.New("jam kirim follow-up harus HH:MM dan jam mulai sebelum jam selesai")
	}
	if req.FollowupMinGapSeconds < 10 || req.FollowupMinGapSeconds > 3600 {
		return errors.New("jeda minimum follow-up harus 10–3600 detik")
	}
	if req.FollowupDailyCap < 1 || req.FollowupDailyCap > 500 {
		return errors.New("batas harian follow-up harus 1–500")
	}
	if req.TranscriptRetentionDaysLost < 30 || req.TranscriptRetentionDaysLost > 3650 {
		return errors.New("retensi transkrip harus 30–3650 hari")
	}
	if len([]rune(req.PaymentVerifiedTemplate)) > 1000 {
		return errors.New("template konfirmasi maksimal 1000 karakter")
	}
	keywords := ParseBlockedKeywords(req.PersonaBlockedKeywords)
	if len(keywords) == 0 {
		return errors.New("daftar kata terlarang persona tidak boleh kosong")
	}
	if len(keywords) > 300 {
		return errors.New("daftar kata terlarang persona maksimal 300 entri")
	}
	for _, kw := range keywords {
		if utf8.RuneCountInString(kw) > 60 {
			return fmt.Errorf("kata terlarang %q terlalu panjang (maks 60 karakter)", kw)
		}
	}
	onlyLabel := strings.TrimSpace(req.FollowupOnlyLabel)
	if onlyLabel != "" {
		if _, err := s.querier.GetLeadLabelByName(ctx, onlyLabel); err != nil {
			return fmt.Errorf("label %q belum ada; buat labelnya dulu atau kosongkan", onlyLabel)
		}
	}
	if strings.Contains(req.PaymentVerifiedTemplate, "[[") {
		return errors.New("template konfirmasi tidak boleh memuat marker [[...]]")
	}
	raw, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	return s.querier.UpdateCRMSettings(ctx, queries.UpdateCRMSettingsParams{
		PaketKeKelasKode: string(raw), FollowupEnabled: boolInt(req.FollowupEnabled),
		FollowupSendStart: req.FollowupSendStart, FollowupSendEnd: req.FollowupSendEnd,
		FollowupMinGapSeconds: req.FollowupMinGapSeconds, FollowupDailyCap: req.FollowupDailyCap,
		TranscriptRetentionDaysLost: req.TranscriptRetentionDaysLost,
		PaymentVerifiedTemplate:     strings.TrimSpace(req.PaymentVerifiedTemplate),
		PersonaBlockedKeywords:      strings.Join(keywords, "\n"),
		FollowupOnlyLabel:           onlyLabel,
		UpdatedBy:                   nullInt(actorID), UpdatedAt: s.now().UTC().Truncate(time.Second),
	})
}

func (s *CRMIntegrationService) probe(ctx context.Context, name, target string, basicAuth bool) models.IntegrationStatus {
	st := models.IntegrationStatus{Name: name, Target: target}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		st.Message = "URL tidak valid"
		return st
	}
	if basicAuth {
		req.SetBasicAuth(s.cfg.GowaUser, s.cfg.GowaPass)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		st.Message = "Tidak bisa dihubungi"
		return st
	}
	defer resp.Body.Close()
	st.OK = resp.StatusCode >= 200 && resp.StatusCode < 300
	st.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		st.Message += " (akun dashboard ditolak atau tidak punya akses device ini)"
	}
	return st
}

// Page loads settings, ingest health and live connection checks.
func (s *CRMIntegrationService) Page(ctx context.Context) (*models.IntegrationPage, error) {
	settings, err := s.querier.GetCRMSettings(ctx)
	if err != nil {
		return nil, err
	}
	page := &models.IntegrationPage{Settings: settingsResponse(settings), IngestEnabled: s.cfg.IngestEnabled, OutboundReady: s.cfg.OutboundReady, Statuses: []models.IntegrationStatus{}}
	if last, err := s.querier.GetLatestBridgeEvent(ctx); err == nil && len(last) > 0 {
		page.LastBridgeEvent = formatLeadTime(last[0].CreatedAt)
	} else if err != nil {
		return nil, err
	}
	today := s.now().In(wib)
	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, wib)
	if page.BridgeEventsToday, err = s.querier.CountLeadEventsSince(ctx, dayStart.UTC()); err != nil {
		return nil, err
	}

	if s.cfg.BridgeURL != "" {
		page.Statuses = append(page.Statuses, s.probe(ctx, "Bridge CS AI", s.cfg.BridgeURL+"/healthz", false))
	} else {
		page.Statuses = append(page.Statuses, models.IntegrationStatus{Name: "Bridge CS AI", Message: "CS_BRIDGE_URL belum diisi di .env"})
	}
	devices := map[string]string{}
	_ = json.Unmarshal([]byte(settings.GowaDevices), &devices)
	keys := make([]string, 0, len(devices))
	for k := range devices {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, role := range keys {
		name := "GOWA device " + devices[role]
		if s.cfg.GowaURL == "" || s.cfg.GowaUser == "" {
			page.Statuses = append(page.Statuses, models.IntegrationStatus{Name: name, Message: "GOWA_URL / GOWA_USER belum diisi di .env"})
			continue
		}
		page.Statuses = append(page.Statuses, s.probe(ctx, name, s.cfg.GowaURL+"/devices/"+url.PathEscape(devices[role])+"/status", true))
	}
	return page, nil
}
