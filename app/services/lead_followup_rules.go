package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// Follow-up rules (PRD 10.2, Script 30 Hari). They run when a follow-up is
// proposed (AI or CS) and again right before it is sent (Fase D worker).

const (
	FollowupStepD1     = "d1"
	FollowupStepD3     = "d3"
	FollowupStepD7     = "d7"
	FollowupStepD14    = "d14"
	FollowupStepD30    = "d30"
	FollowupStepManual = "manual"

	followupDraftMax      = 1000
	followupReasonMin     = 20
	followupReasonMax     = 500
	followupMaxAheadDays  = 60
	followupPermissionCTA = "boleh kabari"
)

// FollowupSteps are the Script 30 Hari steps in order, with their day offset
// from the last meaningful outgoing message.
var FollowupSteps = []string{FollowupStepD1, FollowupStepD3, FollowupStepD7, FollowupStepD14, FollowupStepD30}

var followupStepDays = map[string]int{
	FollowupStepD1: 1, FollowupStepD3: 3, FollowupStepD7: 7, FollowupStepD14: 14, FollowupStepD30: 30,
}

func isFollowupStep(step string, allowManual bool) bool {
	if step == FollowupStepManual {
		return allowManual
	}
	_, ok := followupStepDays[step]
	return ok
}

// Kinds of a blocked follow-up. At send time the worker maps them to a status:
// skip → skipped, cancel → cancelled, reject → failed with the reason.
const (
	FollowupBlockSkip   = "skip"   // lead no longer open
	FollowupBlockCancel = "cancel" // lead asked to stop
	FollowupBlockReject = "reject" // rule not met (timing, content, ...)
)

// FollowupBlock is a rule violation. Its message is safe to show to staff and
// to return to the AI as-is.
type FollowupBlock struct {
	Kind string
	Msg  string
}

func (b *FollowupBlock) Error() string { return b.Msg }

func blockFollowup(kind, format string, a ...any) *FollowupBlock {
	return &FollowupBlock{Kind: kind, Msg: fmt.Sprintf(format, a...)}
}

var (
	// Rupiah amounts: "Rp149.000", "rp 149rb", "149.000", "150 ribu", "1,5 jt".
	followupRupiahRe = regexp.MustCompile(`(?i)\brp\.?\s*\d|\b\d{1,3}(?:[.,]\d{3})+\b|\b\d+(?:[.,]\d+)?\s*(?:rb|ribu|jt|juta)\b`)
	// Bank account hints: the words, or a run of 8+ digits (spaces/dashes allowed).
	followupAccountRe = regexp.MustCompile(`(?i)\b(?:rekening|norek|no\.?\s*rek)\b|transfer\s+ke|\d(?:[\s\-]?\d){7,}`)
	followupLinkRe    = regexp.MustCompile(`(?i)https?://|www\.|\bwa\.me\b|\bbit\.ly\b|\b[a-z0-9\-]+\.(?:com|id|net|org|co|ly|me)\b`)
)

// followupContentBlock checks the draft text itself (PRD 10.2 "Isi aman").
func followupContentBlock(step, draft, reason string) *FollowupBlock {
	draft = strings.TrimSpace(draft)
	if draft == "" {
		return blockFollowup(FollowupBlockReject, "draft_text wajib diisi")
	}
	if utf8.RuneCountInString(draft) > followupDraftMax {
		return blockFollowup(FollowupBlockReject, "draft_text maksimal %d karakter", followupDraftMax)
	}
	if strings.Contains(draft, "[[") {
		return blockFollowup(FollowupBlockReject, "draft tidak boleh memuat marker [[...]]")
	}
	if followupRupiahRe.MatchString(draft) {
		return blockFollowup(FollowupBlockReject, "draft tidak boleh memuat nominal rupiah; pembayaran hanya lewat alur CS AI")
	}
	if followupAccountRe.MatchString(draft) {
		return blockFollowup(FollowupBlockReject, "draft tidak boleh memuat nomor rekening atau deretan angka panjang")
	}
	if utf8.RuneCountInString(strings.TrimSpace(reason)) > followupReasonMax {
		return blockFollowup(FollowupBlockReject, "reason maksimal %d karakter", followupReasonMax)
	}
	switch step {
	case FollowupStepD7, FollowupStepD30:
		if utf8.RuneCountInString(strings.TrimSpace(reason)) < followupReasonMin {
			return blockFollowup(FollowupBlockReject, "%s wajib punya reason berisi update nyata (minimal %d karakter); tanpa update jangan kirim", step, followupReasonMin)
		}
	case FollowupStepD14:
		if !strings.Contains(normalizeChatText(draft), followupPermissionCTA) {
			return blockFollowup(FollowupBlockReject, "draft d14 wajib memuat opsi balas \"boleh kabari\"")
		}
	}
	return nil
}

// FollowupDraftWarnings are the red warnings on the approval page (PRD 8.3).
// A follow-up with warnings cannot be bulk-approved.
func FollowupDraftWarnings(step, draft, reason, permission string) []string {
	out := []string{}
	if followupRupiahRe.MatchString(draft) || followupAccountRe.MatchString(draft) {
		out = append(out, "Draft memuat nominal rupiah atau nomor rekening")
	}
	if followupLinkRe.MatchString(draft) {
		out = append(out, "Draft memuat link")
	}
	if (step == FollowupStepD7 || step == FollowupStepD30) && utf8.RuneCountInString(strings.TrimSpace(reason)) < followupReasonMin {
		out = append(out, "Step "+step+" tanpa alasan update nyata")
	}
	if permission == "no" {
		out = append(out, "Lead meminta berhenti dihubungi (diblok)")
	}
	return out
}

// followupCheck holds what one rule evaluation needs.
type followupCheck struct {
	Lead        queries.Lead
	Step        string
	ScheduledAt time.Time // proposed send time (propose) or now (send)
	Draft       string
	Reason      string
	ExcludeID   int64 // the follow-up itself, when re-checking an existing row
	AtSend      bool
}

func wibDayBounds(t time.Time) (time.Time, time.Time) {
	d := t.In(wib)
	start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, wib)
	return start, start.AddDate(0, 0, 1)
}

func formatWIB(t time.Time) string { return t.In(wib).Format("2006-01-02 15:04") + " WIB" }

// checkFollowupRules evaluates PRD 10.2 for one follow-up. It returns nil when
// the follow-up may be proposed (AtSend=false) or sent now (AtSend=true).
func checkFollowupRules(ctx context.Context, q *queries.Querier, c followupCheck) (*FollowupBlock, error) {
	l := c.Lead
	if l.Status == LeadStatusDoNotContact || l.FollowupPermission == "no" {
		return blockFollowup(FollowupBlockCancel, "lead meminta berhenti dihubungi; follow-up tidak boleh dikirim"), nil
	}
	if l.Status != LeadStatusOpen {
		return blockFollowup(FollowupBlockSkip, "follow-up hanya untuk lead yang masih berjalan (status sekarang: %s)", l.Status), nil
	}
	if b := followupContentBlock(c.Step, c.Draft, c.Reason); b != nil {
		return b, nil
	}
	if c.Step == FollowupStepD30 && l.FollowupPermission != "yes" {
		return blockFollowup(FollowupBlockReject, "d30 hanya untuk lead yang pernah membalas \"boleh kabari\""), nil
	}

	if c.Step != FollowupStepManual {
		anchor, err := q.LeadLastMeaningfulOutbound(ctx, l.ID)
		if err != nil {
			return nil, err
		}
		if !anchor.Valid {
			return blockFollowup(FollowupBlockReject, "belum ada pesan keluar dari CS untuk lead ini, jadi belum ada yang perlu di-follow-up"), nil
		}
		if l.LastInboundAt.Valid && !l.LastInboundAt.Time.Before(anchor.Time) {
			return blockFollowup(FollowupBlockReject, "lead sudah membalas pesan terakhir CS; urutan follow-up berhenti"), nil
		}
		earliest := anchor.Time.AddDate(0, 0, followupStepDays[c.Step])
		if c.ScheduledAt.Before(earliest) {
			return blockFollowup(FollowupBlockReject, "terlalu cepat: %s paling cepat %s (%d hari setelah pesan terakhir CS)", c.Step, formatWIB(earliest), followupStepDays[c.Step]), nil
		}
		sent, err := q.LeadFollowupStepSentSince(ctx, l.ID, c.Step, anchor.Time)
		if err != nil {
			return nil, err
		}
		if sent {
			return blockFollowup(FollowupBlockReject, "follow-up %s sudah terkirim untuk percakapan ini", c.Step), nil
		}
	}

	// One touch per lead per WIB day (sent or planned).
	dayStart, dayEnd := wibDayBounds(c.ScheduledAt)
	n, err := q.CountLeadFollowupsSentBetween(ctx, l.ID, c.ExcludeID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	if n == 0 && !c.AtSend {
		if n, err = q.CountLeadFollowupsPlannedBetween(ctx, l.ID, c.ExcludeID, dayStart, dayEnd, true); err != nil {
			return nil, err
		}
	}
	if n > 0 {
		return blockFollowup(FollowupBlockReject, "sudah ada follow-up untuk lead ini pada %s; maksimal satu sentuhan per hari", dayStart.Format("2006-01-02")), nil
	}
	return nil, nil
}
