package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	MaxLeadIngestEvents = 100
	maxLeadMessageBody  = 20000
)

var leadIngestTypes = map[string]bool{
	"message.in": true, "message.out": true, "form.sent": true, "payment_info.sent": true,
	"proof.received": true, "handoff": true, "ai.paused": true, "ai.resumed": true,
	"conversation.stage": true, "schedule.selected": true,
}

// errLeadSchema marks an event the bridge must not resend (bad schema).
type errLeadSchema struct{ msg string }

func (e errLeadSchema) Error() string { return e.msg }

type leadEventData struct {
	WaMessageID string `json:"wa_message_id"`
	Body        string `json:"body"`
	MediaType   string `json:"media_type"`
	Sender      string `json:"sender"`
	ReferenceID int64  `json:"reference_id"`
	Purpose     string `json:"purpose"`
	Stage       string `json:"stage"`
	PaketKode   string `json:"paket_kode"`
	NominalText string `json:"nominal_text"`
	Reason      string `json:"reason"`
	Summary     string `json:"summary"`
	Until       string `json:"until"`
	JadwalKode  string `json:"jadwal_kode"`
}

// IngestEvents applies a batch of bridge events (PRD 7.1). Each event runs in
// its own transaction: an event with a bad schema is rejected alone and the
// rest are processed; a repeated event_id is acknowledged as a duplicate
// without side effects. A database failure stops the batch and is returned so
// the bridge retries it (already applied events then come back as duplicates).
func (s *LeadService) IngestEvents(ctx context.Context, events []models.LeadIngestEvent) (models.LeadIngestResult, error) {
	result := models.LeadIngestResult{Rejected: []models.LeadIngestRejected{}}
	for _, ev := range events {
		dup, err := s.ingestOne(ctx, ev)
		var schemaErr errLeadSchema
		switch {
		case errors.As(err, &schemaErr):
			result.Rejected = append(result.Rejected, models.LeadIngestRejected{EventID: ev.EventID, Error: err.Error()})
		case err != nil:
			return result, err
		case dup:
			result.Duplicates++
		default:
			result.Accepted++
		}
	}
	return result, nil
}

func validateLeadIngestEvent(ev models.LeadIngestEvent) (time.Time, leadEventData, error) {
	var data leadEventData
	if ev.EventID == "" || len(ev.EventID) > 128 {
		return time.Time{}, data, errLeadSchema{"event_id wajib (maks 128 karakter)"}
	}
	if !leadIngestTypes[ev.Type] {
		return time.Time{}, data, errLeadSchema{fmt.Sprintf("type %q tidak dikenal", ev.Type)}
	}
	if ev.ChatID == "" || len(ev.ChatID) > 128 || !strings.Contains(ev.ChatID, "@") {
		return time.Time{}, data, errLeadSchema{"chat_id wajib berupa JID WhatsApp"}
	}
	occurred, err := time.Parse(time.RFC3339, ev.OccurredAt)
	if err != nil {
		return time.Time{}, data, errLeadSchema{"occurred_at harus RFC3339"}
	}
	if len(ev.Data) > 0 && string(ev.Data) != "null" {
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return time.Time{}, data, errLeadSchema{"data tidak valid: " + err.Error()}
		}
	}
	if ev.Type == "conversation.stage" {
		if data.Stage != LeadStageDiscovery && data.Stage != LeadStageRekomendasi {
			return time.Time{}, data, errLeadSchema{"conversation.stage hanya boleh discovery atau rekomendasi"}
		}
		if data.WaMessageID == "" || len(data.WaMessageID) > 128 {
			return time.Time{}, data, errLeadSchema{"conversation.stage memerlukan wa_message_id dari pesan terkirim"}
		}
		if data.Stage == LeadStageRekomendasi && !paketKey.MatchString(strings.ToLower(strings.TrimSpace(data.PaketKode))) {
			return time.Time{}, data, errLeadSchema{"conversation.stage rekomendasi memerlukan kode paket valid"}
		}
	}
	if ev.Type == "schedule.selected" {
		data.JadwalKode = strings.ToLower(strings.TrimSpace(data.JadwalKode))
		if !jadwalKodeRe.MatchString(data.JadwalKode) {
			return time.Time{}, data, errLeadSchema{"schedule.selected memerlukan jadwal_kode valid"}
		}
	}
	if len(data.Body) > maxLeadMessageBody {
		data.Body = data.Body[:maxLeadMessageBody]
	}
	return occurred.UTC().Truncate(time.Second), data, nil
}

func (s *LeadService) ingestOne(ctx context.Context, ev models.LeadIngestEvent) (bool, error) {
	occurred, data, err := validateLeadIngestEvent(ev)
	if err != nil {
		return false, err
	}
	duplicate := false
	err = s.withTx(ctx, func(q *queries.Querier) error {
		n, err := q.LeadEventExists(ctx, ev.EventID)
		if err != nil {
			return err
		}
		if n > 0 {
			duplicate = true
			return nil
		}
		now := s.nowUTC()
		lead, err := s.findOrCreateLead(ctx, q, ev, now)
		if err != nil {
			return err
		}
		payload, err := s.applyLeadEvent(ctx, q, &lead, ev, occurred, data, now)
		if err != nil {
			return err
		}
		if err := saveLead(ctx, q, lead, now); err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, lead.ID, ev.EventID, ev.Type, "bridge", payload, now)
	})
	return duplicate, err
}

func (s *LeadService) findOrCreateLead(ctx context.Context, q *queries.Querier, ev models.LeadIngestEvent, now time.Time) (queries.Lead, error) {
	phone := NormalizeLeadPhone(ev.Phone)
	if phone == "" {
		phone = phoneFromChatID(ev.ChatID)
	}
	waName := strings.TrimSpace(ev.WaName)
	if len(waName) > 120 {
		waName = waName[:120]
	}
	lead, err := q.GetLeadByChatID(ctx, ev.ChatID)
	if errors.Is(err, sql.ErrNoRows) {
		return q.CreateLead(ctx, queries.CreateLeadParams{WaChatID: ev.ChatID, Phone: phone, WaName: waName, CreatedAt: now, UpdatedAt: now})
	}
	if err != nil {
		return lead, err
	}
	if phone != "" && lead.Phone == "" {
		lead.Phone = phone
	}
	if waName != "" {
		lead.WaName = waName
	}
	return lead, nil
}

func laterTime(current sql.NullTime, t time.Time) sql.NullTime {
	if !current.Valid || t.After(current.Time) {
		return nullTime(t)
	}
	return current
}

func stageChange(payload map[string]any, from, to string) {
	payload["stage_from"] = from
	payload["stage_to"] = to
}

func (s *LeadService) applyLeadEvent(ctx context.Context, q *queries.Querier, lead *queries.Lead, ev models.LeadIngestEvent, occurred time.Time, data leadEventData, now time.Time) (map[string]any, error) {
	payload := map[string]any{}
	if data.WaMessageID != "" {
		payload["wa_message_id"] = data.WaMessageID
	}
	// A won lead is read-only (PRD 6.2 rule 4): keep the transcript and
	// timeline, but no automatic event changes its data columns.
	if lead.Status == LeadStatusWon && ev.Type != "message.in" && ev.Type != "message.out" {
		payload["ignored"] = "lead sudah menjadi santri"
		return payload, nil
	}
	switch ev.Type {
	case "message.in":
		return payload, s.applyMessageIn(ctx, q, lead, ev, occurred, data, now, payload)

	case "message.out":
		sender := data.Sender
		if sender != "ai" && sender != "admin" && sender != "followup" {
			sender = "ai"
		}
		payload["sender"] = sender
		if _, err := q.CreateLeadMessage(ctx, queries.CreateLeadMessageParams{
			LeadID: lead.ID, WaMessageID: data.WaMessageID, Direction: "out", Sender: sender,
			Body: data.Body, MediaType: data.MediaType, SentAt: occurred, CreatedAt: now,
		}); err != nil {
			return nil, err
		}
		lead.LastOutboundAt = laterTime(lead.LastOutboundAt, occurred)
		// Only a follow-up send carries a lead_followups id (other outbound
		// purposes, e.g. payment confirmation, send reference_id 0).
		if sender == "followup" && data.ReferenceID > 0 && (data.Purpose == "" || data.Purpose == OutboxPurposeFollowup) {
			payload["followup_id"] = data.ReferenceID
			if _, err := q.MarkLeadFollowupSent(ctx, queries.MarkLeadFollowupSentParams{
				SentAt: nullTime(occurred), WaMessageID: data.WaMessageID, UpdatedAt: now, ID: data.ReferenceID, LeadID: lead.ID,
			}); err != nil {
				return nil, err
			}
		}

	case "conversation.stage":
		// Bridge emits this milestone ONLY after GOWA confirms an actual
		// CS AI message was sent, with the referenced WhatsApp message id.
		// Auto advancement is forward-only and must not override won/lost/DNC
		// records. Staff can still correct the stage manually.
		payload["wa_message_id"] = data.WaMessageID
		payload["stage_source"] = "cs_ai"
		if data.Stage == LeadStageRekomendasi {
			payload["paket_kode"] = strings.ToLower(strings.TrimSpace(data.PaketKode))
		}
		if from, ok := advanceLeadStage(lead, data.Stage); ok {
			stageChange(payload, from, lead.Stage)
		}

	case "form.sent":
		if from, ok := advanceLeadStage(lead, LeadStageFormDikirim); ok {
			stageChange(payload, from, lead.Stage)
		}

	case "payment_info.sent":
		paket := strings.ToLower(strings.TrimSpace(data.PaketKode))
		if !paketKey.MatchString(paket) {
			paket = ""
		}
		if paket != "" {
			lead.PaketKode = paket
			settings, err := q.GetCRMSettings(ctx)
			if err != nil {
				return nil, err
			}
			if kode := paketToKelasKode(settings, paket); kode != "" {
				lead.KelasKode = kode
			}
		}
		if amount := ParseRupiah(data.NominalText); amount > 0 {
			lead.PaymentAmount = amount
		}
		lead.PaymentInfoSentAt = nullTime(occurred)
		payload["paket_kode"] = paket
		payload["nominal"] = lead.PaymentAmount
		if from, ok := advanceLeadStage(lead, LeadStageInfoBayarDikirim); ok {
			stageChange(payload, from, lead.Stage)
		}

	case "proof.received":
		payload["media_type"] = data.MediaType
		if err := s.applyProofReceived(ctx, q, lead, occurred, now, payload); err != nil {
			return nil, err
		}

	case "handoff":
		payload["reason"] = truncateRunes(data.Reason, 300)
		payload["summary"] = truncateRunes(data.Summary, 600)
		if strings.Contains(strings.ToLower(data.Reason), "bukti transfer") && !lead.ProofReceivedAt.Valid {
			if err := s.applyProofReceived(ctx, q, lead, occurred, now, payload); err != nil {
				return nil, err
			}
		}

	case "schedule.selected":
		// The calon santri chose a schedule in the chat. It is a preference:
		// Admin Kelas confirms it at placement. After payment is verified the
		// seat is counted, so only staff may change it from then on.
		payload["jadwal_kode"] = data.JadwalKode
		_, label, ok, err := jadwalLabelByKode(ctx, q, data.JadwalKode)
		if err != nil {
			return nil, err
		}
		switch {
		case !ok:
			payload["ignored"] = "jadwal tidak dikenal"
		case lead.VerifiedAt.Valid:
			payload["ignored"] = "pembayaran sudah diverifikasi; ubah jadwal manual"
		default:
			payload["jadwal_from"] = lead.JadwalKode
			payload["jadwal_label"] = label
			lead.JadwalKode = data.JadwalKode
		}

	case "ai.paused", "ai.resumed":
		payload["occurred_at"] = occurred.Format(time.RFC3339)
		payload["until"] = data.Until
		payload["reason"] = truncateRunes(data.Reason, 300)
	}
	return payload, nil
}

func (s *LeadService) applyMessageIn(ctx context.Context, q *queries.Querier, lead *queries.Lead, ev models.LeadIngestEvent, occurred time.Time, data leadEventData, now time.Time, payload map[string]any) error {
	if data.MediaType != "" {
		payload["media_type"] = data.MediaType
	}
	if _, err := q.CreateLeadMessage(ctx, queries.CreateLeadMessageParams{
		LeadID: lead.ID, WaMessageID: data.WaMessageID, Direction: "in", Sender: "lead",
		Body: data.Body, MediaType: data.MediaType, SentAt: occurred, CreatedAt: now,
	}); err != nil {
		return err
	}
	lead.LastInboundAt = laterTime(lead.LastInboundAt, occurred)

	switch lead.Status {
	case LeadStatusLost:
		// A lost lead that writes again is reopened; stage is kept (PRD 6.2 rule 5).
		lead.Status = LeadStatusOpen
		lead.LostReason = ""
		if err := writeLeadEvent(ctx, q, lead.ID, ev.EventID+":reopened", "lead.reopened", "bridge",
			map[string]any{"status_from": LeadStatusLost, "status_to": LeadStatusOpen}, now); err != nil {
			return err
		}
	case LeadStatusDoNotContact:
		payload["contacted_again"] = true
	}

	// The lead replied, so every pending follow-up of this lead stops (PRD 10.2).
	if n, err := cancelActiveFollowups(ctx, q, lead.ID, "lead membalas", now); err != nil {
		return err
	} else if n > 0 {
		payload["followups_cancelled"] = n
	}

	if IsLeadStopRequest(data.Body) {
		lead.FollowupPermission = "no"
		if err := writeLeadEvent(ctx, q, lead.ID, ev.EventID+":optout", "optout.detected", "bridge",
			map[string]any{"wa_message_id": data.WaMessageID}, now); err != nil {
			return err
		}
	} else if IsLeadFollowupPermission(data.Body) && lead.Status == LeadStatusOpen && lead.FollowupPermission != "no" {
		lead.FollowupPermission = "yes"
		payload["followup_permission"] = "yes"
	}

	if lead.Status == LeadStatusOpen && (lead.Stage == LeadStageFormDikirim || lead.Stage == LeadStageDataTerisi) {
		return s.applyFormParse(ctx, q, lead, ev, data, now)
	}
	return nil
}

func (s *LeadService) applyFormParse(ctx context.Context, q *queries.Querier, lead *queries.Lead, ev models.LeadIngestEvent, data leadEventData, now time.Time) error {
	parsed := ParseLeadForm(data.Body)
	if len(parsed.Recognized) == 0 {
		return nil
	}
	if !parsed.Filled() {
		// Looks like a form attempt but too little was readable: a human checks it.
		lead.NeedsReview = 1
		return writeLeadEvent(ctx, q, lead.ID, ev.EventID+":form", "form.unreadable", "bridge",
			map[string]any{"fields": parsed.Recognized}, now)
	}
	setStr := func(dst *string, v string) {
		if v != "" {
			*dst = truncateRunes(v, 200)
		}
	}
	setStr(&lead.Nama, parsed.Nama)
	setStr(&lead.Domisili, parsed.Domisili)
	setStr(&lead.JenisKelamin, parsed.JenisKelamin)
	setStr(&lead.WaktuBelajar, parsed.WaktuBelajar)
	setStr(&lead.RiwayatBelajar, parsed.RiwayatBelajar)
	setStr(&lead.BelajarDimana, parsed.BelajarDimana)
	if parsed.Usia > 0 {
		lead.Usia = sql.NullInt64{Int64: parsed.Usia, Valid: true}
	}
	if parsed.PaketKode != "" {
		lead.PaketKode = parsed.PaketKode
		settings, err := q.GetCRMSettings(ctx)
		if err != nil {
			return err
		}
		if kode := paketToKelasKode(settings, parsed.PaketKode); kode != "" {
			lead.KelasKode = kode
		}
	}
	if len(parsed.Review) > 0 {
		lead.NeedsReview = 1
	}
	payload := map[string]any{"fields": parsed.Recognized, "review_fields": parsed.Review}
	if from, ok := advanceLeadStage(lead, LeadStageDataTerisi); ok {
		stageChange(payload, from, lead.Stage)
	}
	return writeLeadEvent(ctx, q, lead.ID, ev.EventID+":form", "form.parsed", "bridge", payload, now)
}

func (s *LeadService) applyProofReceived(ctx context.Context, q *queries.Querier, lead *queries.Lead, occurred, now time.Time, payload map[string]any) error {
	firstProof := !lead.ProofReceivedAt.Valid
	if firstProof {
		lead.ProofReceivedAt = nullTime(occurred)
	}
	if from, ok := advanceLeadStage(lead, LeadStageBuktiMasuk); ok {
		stageChange(payload, from, lead.Stage)
	}
	if !firstProof || lead.Status != LeadStatusOpen {
		return nil
	}
	recipients, err := q.ListUserIDsByRole(ctx, "keuangan")
	if err != nil {
		return err
	}
	if lead.AssignedUserID.Valid {
		recipients = append(recipients, lead.AssignedUserID.Int64)
	}
	return notifyUsers(ctx, q, recipients, queries.CreateNotificationParams{
		Type:          "lead_bukti_transfer",
		Title:         "Bukti transfer masuk dari " + leadDisplayName(*lead),
		Message:       "Cek bukti di WhatsApp CS lalu verifikasi pembayaran di halaman lead.",
		ActionUrl:     fmt.Sprintf("/app/leads/%d", lead.ID),
		ReferenceType: "lead",
		ReferenceID:   sql.NullInt64{Int64: lead.ID, Valid: true},
	})
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max])
}
