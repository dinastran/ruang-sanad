package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// Lead pipeline stages for status=open, in pipeline order (PRD 6.1).
const (
	LeadStageBaru             = "baru"
	LeadStageDiscovery        = "discovery"
	LeadStageRekomendasi      = "rekomendasi"
	LeadStageFormDikirim      = "form_dikirim"
	LeadStageDataTerisi       = "data_terisi"
	LeadStageInfoBayarDikirim = "info_bayar_dikirim"
	LeadStageBuktiMasuk       = "bukti_masuk"
	LeadStageTerverifikasi    = "terverifikasi"
)

const (
	LeadStatusOpen         = "open"
	LeadStatusWon          = "won"
	LeadStatusLost         = "lost"
	LeadStatusDoNotContact = "do_not_contact"
)

// LeadStages lists every stage in pipeline order.
var LeadStages = []string{
	LeadStageBaru, LeadStageDiscovery, LeadStageRekomendasi, LeadStageFormDikirim,
	LeadStageDataTerisi, LeadStageInfoBayarDikirim, LeadStageBuktiMasuk, LeadStageTerverifikasi,
}

var LeadLostReasons = []string{"harga", "jadwal", "belum_prioritas", "keluarga", "tidak_cocok", "tidak_merespons", "lainnya"}

var (
	ErrLeadNotFound     = errors.New("lead tidak ditemukan")
	ErrLeadReadOnly     = errors.New("lead sudah menjadi santri dan tidak bisa diubah")
	ErrLeadInvalidStage = errors.New("tahap tidak valid")
)

func leadStageIndex(stage string) int {
	for i, s := range LeadStages {
		if s == stage {
			return i
		}
	}
	return -1
}

func isValidLeadStage(stage string) bool { return leadStageIndex(stage) >= 0 }

// LeadService owns the CRM pipeline: ingest from the CS AI bridge, manual
// actions, verification and conversion to santri.
type LeadService struct {
	querier *queries.Querier
	santri  *SantriService
	now     func() time.Time
}

func NewLeadService(querier *queries.Querier, santri *SantriService) *LeadService {
	return &LeadService{querier: querier, santri: santri, now: time.Now}
}

// nowUTC returns the service clock in UTC, truncated to seconds so stored
// DATETIME strings sort consistently.
func (s *LeadService) nowUTC() time.Time {
	return s.now().UTC().Truncate(time.Second)
}

func newLeadEventID(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: !t.IsZero()} }

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func saveLead(ctx context.Context, q *queries.Querier, l queries.Lead, now time.Time) error {
	return q.UpdateLead(ctx, queries.UpdateLeadParams{
		Phone: l.Phone, WaName: l.WaName, Nama: l.Nama, Usia: l.Usia, Domisili: l.Domisili,
		JenisKelamin: l.JenisKelamin, PaketKode: l.PaketKode, KelasKode: l.KelasKode,
		WaktuBelajar: l.WaktuBelajar, RiwayatBelajar: l.RiwayatBelajar, BelajarDimana: l.BelajarDimana,
		UntukSiapa: l.UntukSiapa, Stage: l.Stage, Status: l.Status, LostReason: l.LostReason,
		FollowupPermission: l.FollowupPermission, AssignedUserID: l.AssignedUserID,
		PaymentAmount: l.PaymentAmount, PaymentInfoSentAt: l.PaymentInfoSentAt,
		ProofReceivedAt: l.ProofReceivedAt, VerifiedAt: l.VerifiedAt, VerifiedBy: l.VerifiedBy,
		NeedsReview: l.NeedsReview, LastInboundAt: l.LastInboundAt, LastOutboundAt: l.LastOutboundAt,
		JadwalKode: l.JadwalKode, UpdatedAt: now, ID: l.ID,
	})
}

func writeLeadEvent(ctx context.Context, q *queries.Querier, leadID int64, eventID, typ, actor string, payload map[string]any, at time.Time) error {
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.CreateLeadEvent(ctx, queries.CreateLeadEventParams{
		LeadID: leadID, EventID: eventID, Type: typ, Actor: actor, Payload: string(raw), CreatedAt: at,
	})
}

// advanceLeadStage moves an open lead forward only (PRD 6.2 rule 1). Automatic
// events can never reach terverifikasi.
func advanceLeadStage(l *queries.Lead, target string) (from string, changed bool) {
	if l.Status != LeadStatusOpen || target == LeadStageTerverifikasi {
		return l.Stage, false
	}
	if leadStageIndex(target) <= leadStageIndex(l.Stage) {
		return l.Stage, false
	}
	from = l.Stage
	l.Stage = target
	return from, true
}

// cancelActiveFollowups cancels pending/approved follow-ups of a lead and any
// outbox rows still queued for them.
func cancelActiveFollowups(ctx context.Context, q *queries.Querier, leadID int64, reason string, now time.Time) (int64, error) {
	n, err := q.CancelActiveLeadFollowups(ctx, queries.CancelActiveLeadFollowupsParams{Error: reason, UpdatedAt: now, LeadID: leadID})
	if err != nil {
		return 0, err
	}
	if _, err := q.CancelQueuedOutboxForLead(ctx, queries.CancelQueuedOutboxForLeadParams{LastError: reason, UpdatedAt: now, LeadID: leadID}); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *LeadService) withTx(ctx context.Context, fn func(q *queries.Querier) error) error {
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.querier.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func loadLead(ctx context.Context, q *queries.Querier, id int64) (queries.Lead, error) {
	l, err := q.GetLeadByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrLeadNotFound
	}
	return l, err
}

// paketToKelasKode maps a bridge package code (1x, 2x) to a kode_kelas using
// crm_settings.paket_ke_kelas_kode.
func paketToKelasKode(settings queries.CrmSetting, paket string) string {
	mapping := map[string]string{}
	if err := json.Unmarshal([]byte(settings.PaketKeKelasKode), &mapping); err != nil {
		return ""
	}
	return mapping[strings.ToLower(strings.TrimSpace(paket))]
}

func leadDisplayName(l queries.Lead) string {
	switch {
	case l.Nama != "":
		return l.Nama
	case l.WaName != "":
		return l.WaName
	case l.Phone != "":
		return l.Phone
	}
	return fmt.Sprintf("Lead L-%d", l.ID)
}

// notifyUsers inserts one dashboard notification per distinct user id.
func notifyUsers(ctx context.Context, q *queries.Querier, userIDs []int64, params queries.CreateNotificationParams) error {
	seen := map[int64]bool{}
	for _, id := range userIDs {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		p := params
		p.UserID = id
		if _, err := q.CreateNotification(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// PurgeClosedLeadTranscripts deletes chat transcripts of lost/do_not_contact
// leads untouched for crm_settings.transcript_retention_days_lost days
// (PRD 11.2). The lead profile stays for aggregate reports.
func (s *LeadService) PurgeClosedLeadTranscripts(ctx context.Context) (int64, error) {
	settings, err := s.querier.GetCRMSettings(ctx)
	if err != nil {
		return 0, err
	}
	if settings.TranscriptRetentionDaysLost <= 0 {
		return 0, nil
	}
	cutoff := s.nowUTC().AddDate(0, 0, -int(settings.TranscriptRetentionDaysLost))
	// Free-text notes in the timeline (handoff summaries, verification notes)
	// can repeat chat content, so they expire with the transcript.
	if _, err := s.querier.ClearLeadEventNotesForClosedLeads(ctx, cutoff); err != nil {
		return 0, err
	}
	return s.querier.DeleteLeadMessagesForClosedLeads(ctx, cutoff)
}
