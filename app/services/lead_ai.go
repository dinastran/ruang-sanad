package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// CRM writes made by the AI agent through MCP (PRD 9.2). The AI only adds
// analysis, proposals and labels; stage, status, verification, conversion
// and sending stay human actions (PRD 9.4).

// LeadInputError is a validation error whose message is safe to return to
// the caller (staff or AI) as-is.
type LeadInputError struct{ Msg string }

func (e LeadInputError) Error() string { return e.Msg }

func leadInputErr(format string, a ...any) error { return LeadInputError{fmt.Sprintf(format, a...)} }

func aiActor(tokenID int64) string { return fmt.Sprintf("ai:%d", tokenID) }

// Persona schema (PRD 9.3). Unknown fields are rejected.
var (
	personaEnums = map[string][]string{
		"untuk_siapa":      {"diri_sendiri", "anak", "keluarga", "tidak_diketahui"},
		"level":            {"pemula", "pernah_belajar", "sedang_belajar", "tidak_diketahui"},
		"preferensi_waktu": {"pagi", "siang", "malam", "fleksibel", "tidak_diketahui"},
		"paket_diminati":   {"1x", "2x", "belum_jelas"},
		"gaya_komunikasi":  {"singkat", "detail", "formal", "santai", "tidak_diketahui"},
	}
	personaEnumDefaults = map[string]string{
		"untuk_siapa": "tidak_diketahui", "level": "tidak_diketahui", "preferensi_waktu": "tidak_diketahui",
		"paket_diminati": "belum_jelas", "gaya_komunikasi": "tidak_diketahui",
	}
	personaLists = map[string][]string{
		"tujuan":   {"makhraj", "tajwid", "percaya_diri", "rutinitas", "lainnya"},
		"hambatan": {"harga", "jadwal", "konsistensi", "keluarga", "ragu_online", "percaya_program", "belum_prioritas"},
	}
	// LeadObjections are the main_objection categories: the persona hambatan
	// plus "tidak_ada" and "lainnya".
	LeadObjections  = []string{"harga", "jadwal", "konsistensi", "keluarga", "ragu_online", "percaya_program", "belum_prioritas", "tidak_ada", "lainnya"}
	LeadConfidences = []string{"rendah", "sedang", "tinggi"}
)

const (
	personaCatatanMax     = 300
	analysisSummaryMax    = 600
	analysisNextActionMax = 300
)

func inList(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// keywordText lowercases s and turns every non letter/digit into a space,
// padded so " kata " matches whole words and phrases.
func keywordText(s string) string {
	var b strings.Builder
	b.WriteByte(' ')
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	b.WriteByte(' ')
	return " " + strings.Join(strings.Fields(b.String()), " ") + " "
}

// ParseBlockedKeywords splits the crm_settings list (one per line or comma).
func ParseBlockedKeywords(raw string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ',' || r == ';' }) {
		kw := strings.TrimSpace(keywordText(part))
		if kw == "" || seen[kw] {
			continue
		}
		seen[kw] = true
		out = append(out, kw)
	}
	return out
}

// matchBlockedKeyword returns the first blocked keyword found in text.
func matchBlockedKeyword(text string, keywords []string) string {
	t := keywordText(text)
	for _, kw := range keywords {
		if strings.Contains(t, " "+kw+" ") {
			return kw
		}
	}
	return ""
}

// ValidatePersona checks the persona against the fixed schema and returns its
// normalized JSON (all fields present, lists sorted and unique).
func ValidatePersona(raw json.RawMessage, blocked []string) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return "", leadInputErr("persona wajib diisi (objek sesuai skema)")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", leadInputErr("persona harus berupa objek JSON")
	}
	out := map[string]any{}
	for key, val := range fields {
		switch {
		case personaEnums[key] != nil:
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				return "", leadInputErr("persona.%s harus string", key)
			}
			v = strings.TrimSpace(v)
			if !inList(v, personaEnums[key]) {
				return "", leadInputErr("persona.%s harus salah satu dari: %s", key, strings.Join(personaEnums[key], ", "))
			}
			out[key] = v
		case personaLists[key] != nil:
			var vs []string
			if err := json.Unmarshal(val, &vs); err != nil {
				return "", leadInputErr("persona.%s harus array string", key)
			}
			uniq := map[string]bool{}
			list := []string{}
			for _, v := range vs {
				v = strings.TrimSpace(v)
				if !inList(v, personaLists[key]) {
					return "", leadInputErr("persona.%s hanya boleh berisi: %s", key, strings.Join(personaLists[key], ", "))
				}
				if !uniq[v] {
					uniq[v] = true
					list = append(list, v)
				}
			}
			sort.Strings(list)
			out[key] = list
		case key == "catatan":
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				return "", leadInputErr("persona.catatan harus string")
			}
			v = strings.TrimSpace(v)
			if utf8.RuneCountInString(v) > personaCatatanMax {
				return "", leadInputErr("persona.catatan maksimal %d karakter", personaCatatanMax)
			}
			if kw := matchBlockedKeyword(v, blocked); kw != "" {
				return "", leadInputErr("persona.catatan memuat kata terlarang %q: jangan menilai tingkat keagamaan, ekonomi, kesehatan, etnis, atau politik", kw)
			}
			out[key] = v
		default:
			return "", leadInputErr("persona.%s tidak dikenal; field yang diizinkan: untuk_siapa, level, tujuan, preferensi_waktu, paket_diminati, hambatan, gaya_komunikasi, catatan", key)
		}
	}
	for key, def := range personaEnumDefaults {
		if _, ok := out[key]; !ok {
			out[key] = def
		}
	}
	for key := range personaLists {
		if _, ok := out[key]; !ok {
			out[key] = []string{}
		}
	}
	if _, ok := out["catatan"]; !ok {
		out["catatan"] = ""
	}
	b, err := json.Marshal(out)
	return string(b), err
}

// LeadAnalysisInput is one analysis written by the AI (simpan_analisis_lead).
type LeadAnalysisInput struct {
	Summary                string
	Persona                json.RawMessage
	InterestScore          int64
	MainObjection          string
	SuggestedStage         string
	SuggestedNextAction    string
	Confidence             string
	AnalyzedUntilMessageID int64
}

// SaveAIAnalysis stores an analysis and moves last_analyzed_message_id
// forward. It never changes the lead's stage or status (PRD 9.4).
func (s *LeadService) SaveAIAnalysis(ctx context.Context, leadID, tokenID int64, in LeadAnalysisInput) (int64, error) {
	in.Summary = strings.TrimSpace(in.Summary)
	in.SuggestedNextAction = strings.TrimSpace(in.SuggestedNextAction)
	switch {
	case in.Summary == "":
		return 0, leadInputErr("summary wajib diisi")
	case utf8.RuneCountInString(in.Summary) > analysisSummaryMax:
		return 0, leadInputErr("summary maksimal %d karakter", analysisSummaryMax)
	case utf8.RuneCountInString(in.SuggestedNextAction) > analysisNextActionMax:
		return 0, leadInputErr("suggested_next_action maksimal %d karakter", analysisNextActionMax)
	case in.InterestScore < 0 || in.InterestScore > 100:
		return 0, leadInputErr("interest_score harus 0–100")
	case !inList(in.MainObjection, LeadObjections):
		return 0, leadInputErr("main_objection harus salah satu dari: %s", strings.Join(LeadObjections, ", "))
	case !inList(in.Confidence, LeadConfidences):
		return 0, leadInputErr("confidence harus rendah, sedang, atau tinggi")
	case in.SuggestedStage != "" && (!isValidLeadStage(in.SuggestedStage) || in.SuggestedStage == LeadStageTerverifikasi):
		return 0, leadInputErr("suggested_stage harus tahap pipeline selain terverifikasi (atau kosong)")
	case in.AnalyzedUntilMessageID <= 0:
		return 0, leadInputErr("analyzed_until_message_id wajib (id pesan terakhir yang Anda baca dari detail_lead)")
	}
	settings, err := s.querier.GetCRMSettings(ctx)
	if err != nil {
		return 0, err
	}
	persona, err := ValidatePersona(in.Persona, ParseBlockedKeywords(settings.PersonaBlockedKeywords))
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, leadID)
		if err != nil {
			return err
		}
		if l.Status != LeadStatusOpen {
			return leadInputErr("lead L-%d berstatus %s; analisis hanya untuk lead yang masih berjalan", l.ID, l.Status)
		}
		if _, err := q.GetLeadMessageByID(ctx, queries.GetLeadMessageByIDParams{ID: in.AnalyzedUntilMessageID, LeadID: leadID}); errors.Is(err, sql.ErrNoRows) {
			return leadInputErr("analyzed_until_message_id %d bukan pesan milik lead ini", in.AnalyzedUntilMessageID)
		} else if err != nil {
			return err
		}
		now := s.nowUTC()
		id, err = q.CreateLeadAnalysis(ctx, queries.CreateLeadAnalysisParams{
			LeadID: leadID, AiTokenID: nullInt(tokenID), Summary: in.Summary, Persona: persona,
			InterestScore: in.InterestScore, MainObjection: in.MainObjection, SuggestedStage: in.SuggestedStage,
			SuggestedNextAction: in.SuggestedNextAction, Confidence: in.Confidence,
			AnalyzedUntilMessageID: in.AnalyzedUntilMessageID, CreatedAt: now,
		})
		if err != nil {
			return err
		}
		if err := q.SetLeadLastAnalyzedMessage(ctx, queries.SetLeadLastAnalyzedMessageParams{MessageID: in.AnalyzedUntilMessageID, ID: leadID}); err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("ai"), "analysis.saved", aiActor(tokenID), map[string]any{
			"analysis_id": id, "interest_score": in.InterestScore, "suggested_stage": in.SuggestedStage, "confidence": in.Confidence,
		}, now)
	})
	return id, err
}

// LeadFollowupInput is a proposed follow-up (AI tool or CS form).
type LeadFollowupInput struct {
	Step        string
	ScheduledAt time.Time
	DraftText   string
	Reason      string
}

// ProposeFollowup inserts a follow-up awaiting approval after it passes the
// PRD 10.2 rules. createdByKind is "ai" (createdBy = token id) or "user".
// A rule violation is returned as *FollowupBlock with a readable reason.
func (s *LeadService) ProposeFollowup(ctx context.Context, leadID int64, in LeadFollowupInput, createdByKind string, createdBy int64) (int64, error) {
	in.Step = strings.TrimSpace(in.Step)
	in.DraftText = strings.TrimSpace(in.DraftText)
	in.Reason = strings.TrimSpace(in.Reason)
	if !isFollowupStep(in.Step, createdByKind == "user") {
		if createdByKind == "user" {
			return 0, leadInputErr("step harus d1, d3, d7, d14, d30, atau manual")
		}
		return 0, leadInputErr("step harus d1, d3, d7, d14, atau d30")
	}
	now := s.nowUTC()
	if in.ScheduledAt.IsZero() {
		return 0, leadInputErr("scheduled_at wajib diisi")
	}
	in.ScheduledAt = in.ScheduledAt.UTC().Truncate(time.Second)
	if in.ScheduledAt.Before(now.Add(-time.Minute)) {
		return 0, leadInputErr("scheduled_at tidak boleh di masa lalu")
	}
	if in.ScheduledAt.After(now.AddDate(0, 0, followupMaxAheadDays)) {
		return 0, leadInputErr("scheduled_at maksimal %d hari ke depan", followupMaxAheadDays)
	}
	var id int64
	err := s.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, leadID)
		if err != nil {
			return err
		}
		block, err := checkFollowupRules(ctx, q, followupCheck{Lead: l, Step: in.Step, ScheduledAt: in.ScheduledAt, Draft: in.DraftText, Reason: in.Reason})
		if err != nil {
			return err
		}
		if block != nil {
			return block
		}
		id, err = q.CreateLeadFollowup(ctx, queries.CreateLeadFollowupParams{
			LeadID: leadID, Step: in.Step, ScheduledAt: in.ScheduledAt, DraftText: in.DraftText, Reason: in.Reason,
			CreatedByKind: createdByKind, CreatedBy: nullInt(createdBy), CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return blockFollowup(FollowupBlockReject, "follow-up %s untuk lead ini sudah menunggu persetujuan atau sudah disetujui", in.Step)
			}
			return err
		}
		actor := userActor(createdBy)
		if createdByKind == "ai" {
			actor = aiActor(createdBy)
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("fu"), "followup.proposed", actor, map[string]any{
			"followup_id": id, "step": in.Step, "scheduled_at": formatWIB(in.ScheduledAt),
		}, now)
	})
	return id, err
}

// AddLabelByAI links an existing label (by name) to a lead.
func (s *LeadService) AddLabelByAI(ctx context.Context, leadID, tokenID, ownerUserID int64, labelName string) (bool, error) {
	labelName = strings.TrimSpace(labelName)
	if labelName == "" {
		return false, leadInputErr("label wajib diisi")
	}
	added := false
	err := s.withTx(ctx, func(q *queries.Querier) error {
		if _, err := loadLead(ctx, q, leadID); err != nil {
			return err
		}
		label, err := q.GetLeadLabelByName(ctx, labelName)
		if errors.Is(err, sql.ErrNoRows) {
			labels, lerr := q.ListLeadLabels(ctx)
			if lerr != nil {
				return lerr
			}
			names := make([]string, 0, len(labels))
			for _, lb := range labels {
				names = append(names, lb.Nama)
			}
			return leadInputErr("label %q belum ada; AI hanya boleh memakai label yang sudah ada: %s", labelName, strings.Join(names, ", "))
		}
		if err != nil {
			return err
		}
		now := s.nowUTC()
		n, err := q.AddLeadLabelLink(ctx, queries.AddLeadLabelLinkParams{LeadID: leadID, LabelID: label.ID, CreatedBy: nullInt(ownerUserID), CreatedAt: now})
		if err != nil || n == 0 {
			return err
		}
		added = true
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("ai"), "label.added", aiActor(tokenID), map[string]any{"label": label.Nama}, now)
	})
	return added, err
}
