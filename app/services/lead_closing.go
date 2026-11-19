package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrLeadAlreadyConverted = errors.New("lead ini sudah dijadikan santri")
	ErrLeadNotVerified      = errors.New("lead harus berstatus Terverifikasi sebelum dijadikan santri")
)

// ErrLeadDuplicateSantri means a santri with the same WhatsApp number exists;
// the user must choose to link to it or explicitly create a new one.
type ErrLeadDuplicateSantri struct {
	Matches []models.LeadDuplicateSantri
}

func (e *ErrLeadDuplicateSantri) Error() string {
	return fmt.Sprintf("nomor ini sudah terdaftar sebagai santri %s", e.Matches[0].Nama)
}

// leadVerification is one payment confirmation: by staff (ActorUserID > 0,
// strict stage rules) or by the payment gateway (system, AnyOpenStage).
type leadVerification struct {
	Amount       int64
	Date         time.Time
	Note         string
	NoProof      bool
	ActorUserID  int64
	AnyOpenStage bool
}

// verifyLeadTx moves an open lead to terverifikasi inside the caller's
// transaction and returns the payment.verified event payload. The caller
// saves the lead and writes the event.
func verifyLeadTx(ctx context.Context, q *queries.Querier, l *queries.Lead, v leadVerification, now time.Time) (map[string]any, error) {
	if l.Status != LeadStatusOpen {
		return nil, errors.New("hanya lead yang masih berjalan yang bisa diverifikasi")
	}
	if l.Stage == LeadStageTerverifikasi {
		return nil, errors.New("lead sudah terverifikasi")
	}
	if !v.AnyOpenStage {
		switch l.Stage {
		case LeadStageBuktiMasuk:
		case LeadStageInfoBayarDikirim:
			if !v.NoProof {
				return nil, errors.New("bukti transfer belum tercatat dari WA; centang konfirmasi untuk tetap memverifikasi")
			}
		default:
			return nil, errors.New("verifikasi hanya untuk lead di tahap Info bayar dikirim atau Bukti transfer masuk")
		}
	}
	if v.Amount <= 0 {
		return nil, errors.New("nominal diterima wajib diisi")
	}
	if v.ActorUserID > 0 && l.PaymentAmount > 0 && v.Amount != l.PaymentAmount && v.Note == "" {
		return nil, fmt.Errorf("nominal berbeda dari tagihan (Rp%d); catatan wajib diisi", l.PaymentAmount)
	}
	if v.Date.After(now.In(wib)) {
		return nil, errors.New("tanggal terima tidak boleh di masa depan")
	}
	if v.ActorUserID > 0 {
		// Staff verified the transfer: waiting manual proofs are the payment.
		if err := markManualPaymentsPaid(ctx, q, l.ID, now); err != nil {
			return nil, err
		}
	} else if err := flagPendingManualPayments(ctx, q, l.ID, now); err != nil {
		return nil, err
	}
	from := l.Stage
	l.Stage = LeadStageTerverifikasi
	l.VerifiedAt = nullTime(now)
	l.VerifiedBy = sql.NullInt64{Int64: v.ActorUserID, Valid: v.ActorUserID > 0}
	return map[string]any{
		"stage_from": from, "stage_to": l.Stage, "amount": v.Amount, "expected_amount": l.PaymentAmount,
		"date": v.Date.In(wib).Format(time.DateOnly), "note": v.Note, "without_proof": from == LeadStageInfoBayarDikirim,
	}, nil
}

// VerifyPayment records that the transfer was received (PRD 6.3).
func (s *LeadService) VerifyPayment(ctx context.Context, id, actorID int64, req models.LeadVerifyRequest) error {
	note := truncateRunes(req.Note, 500)
	date, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(req.Date), wib)
	if err != nil {
		return errors.New("tanggal terima harus berformat YYYY-MM-DD")
	}
	return s.mutateLead(ctx, id, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		payload, err := verifyLeadTx(ctx, q, l, leadVerification{
			Amount: req.Amount, Date: date, Note: note, NoProof: req.NoProof, ActorUserID: actorID,
		}, now)
		if err != nil {
			return "", nil, err
		}
		if req.SendConfirmation {
			settings, err := q.GetCRMSettings(ctx)
			if err != nil {
				return "", nil, err
			}
			if strings.TrimSpace(settings.PaymentVerifiedTemplate) == "" {
				return "", nil, errors.New("template konfirmasi belum diisi di Integrasi WhatsApp; hilangkan centang kirim konfirmasi")
			}
			if _, err := q.CreateWaOutbox(ctx, queries.CreateWaOutboxParams{
				Purpose: OutboxPurposePaymentOK, Channel: OutboxChannelBridge, ToChatID: l.WaChatID,
				Body:          RenderPaymentVerifiedTemplate(settings.PaymentVerifiedTemplate, *l, req.Amount),
				ReferenceType: "lead", ReferenceID: sql.NullInt64{Int64: l.ID, Valid: true},
				NotBefore: now, CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return "", nil, err
			}
			payload["confirmation_queued"] = true
		}
		return "payment.verified", payload, nil
	})
}

type ConvertAngkatanOption struct {
	Kode       string `json:"kode"`
	Keterangan string `json:"keterangan"`
}

// ConvertOptions is what the "Jadikan Santri" dialog needs besides the lead.
// Angkatan lists active cohorts, newest first (the default choice).
type ConvertOptions struct {
	Angkatan   []ConvertAngkatanOption      `json:"angkatan"`
	KodeKelas  []string                     `json:"kode_kelas"`
	Duplicates []models.LeadDuplicateSantri `json:"duplicates"`
}

// phoneVariants returns the digit-only forms a number may have in
// santri.no_wa (the query strips spaces, dashes, dots, brackets and "+").
func phoneVariants(phone string) []string {
	if phone == "" {
		return nil
	}
	local := strings.TrimPrefix(phone, "62")
	return []string{phone, "0" + local, local}
}

func findDuplicateSantri(ctx context.Context, q *queries.Querier, phone string) ([]models.LeadDuplicateSantri, error) {
	variants := phoneVariants(NormalizeLeadPhone(phone))
	if len(variants) == 0 {
		return []models.LeadDuplicateSantri{}, nil
	}
	rows, err := q.FindSantriByNoWA(ctx, variants)
	if err != nil {
		return nil, err
	}
	out := make([]models.LeadDuplicateSantri, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.LeadDuplicateSantri{ID: r.ID, Nama: r.Nama, IDMahasantri: r.IDMahasantri, NoWA: r.NoWa})
	}
	return out, nil
}

func (s *LeadService) ConvertOptions(ctx context.Context, id int64) (*ConvertOptions, error) {
	l, err := loadLead(ctx, s.querier, id)
	if err != nil {
		return nil, err
	}
	angkatan, err := s.querier.ListAngkatanAktif(ctx)
	if err != nil {
		return nil, err
	}
	kode, err := s.querier.ListKodeKelas(ctx)
	if err != nil {
		return nil, err
	}
	dups, err := findDuplicateSantri(ctx, s.querier, l.Phone)
	if err != nil {
		return nil, err
	}
	opts := &ConvertOptions{Angkatan: make([]ConvertAngkatanOption, 0, len(angkatan)), KodeKelas: make([]string, 0, len(kode)), Duplicates: dups}
	for _, a := range angkatan {
		opts.Angkatan = append(opts.Angkatan, ConvertAngkatanOption{Kode: a.Kode, Keterangan: a.Keterangan})
	}
	for _, k := range kode {
		opts.KodeKelas = append(opts.KodeKelas, k.Kode)
	}
	return opts, nil
}

// Convert turns a verified lead into a santri (PRD 6.4). Creating the santri,
// linking it to the lead and the lead.converted event happen in one
// transaction; the santri_id IS NULL guard plus the UNIQUE constraint ensure
// a lead never yields two santri, even on concurrent clicks.
func (s *LeadService) Convert(ctx context.Context, id, actorID int64, req models.LeadConvertRequest) (int64, error) {
	var santriID int64
	created := false
	err := s.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, id)
		if err != nil {
			return err
		}
		if l.SantriID.Valid || l.Status == LeadStatusWon {
			return ErrLeadAlreadyConverted
		}
		if l.Status != LeadStatusOpen || l.Stage != LeadStageTerverifikasi {
			return ErrLeadNotVerified
		}
		// Short-program leads close into participants, never into santri.
		if p, err := s.effectiveProgram(ctx, q, id); err != nil {
			return err
		} else if p.Jenis == LeadProgramJenisProgram {
			return ErrLeadProgramPendek
		}
		now := s.nowUTC()

		if req.LinkSantriID > 0 {
			if _, err := q.GetSantriByID(ctx, req.LinkSantriID); err != nil {
				return errors.New("santri yang dipilih tidak ditemukan")
			}
			santriID = req.LinkSantriID
		} else {
			if !req.ForceNew {
				dups, err := findDuplicateSantri(ctx, q, req.NoWA)
				if err != nil {
					return err
				}
				if len(dups) > 0 {
					return &ErrLeadDuplicateSantri{Matches: dups}
				}
			}
			kelasKode := strings.TrimSpace(req.KelasKode)
			if kelasKode != "" {
				if _, err := q.GetKodeKelasByKode(ctx, kelasKode); err != nil {
					return fmt.Errorf("kode kelas %q tidak terdaftar", kelasKode)
				}
			}
			if req.Nominal < 0 {
				return errors.New("nominal tidak valid")
			}
			resp, err := s.santri.CreateWithQuerier(ctx, q, models.CreateSantriRequest{
				KelasKode: kelasKode, Nama: req.Nama, JenisKelamin: req.JenisKelamin, Nominal: req.Nominal,
				TanggalDaftar: req.TanggalDaftar, Angkatan: req.Angkatan, Usia: req.Usia,
				Domisili: strings.TrimSpace(req.Domisili), NoWA: strings.TrimSpace(req.NoWA),
			}, actorID)
			if err != nil {
				return err
			}
			santriID = resp.ID
			created = true
		}

		n, err := q.LinkLeadSantri(ctx, queries.LinkLeadSantriParams{
			SantriID: sql.NullInt64{Int64: santriID, Valid: true}, ConvertedAt: nullTime(now),
			ConvertedBy: sql.NullInt64{Int64: actorID, Valid: true}, UpdatedAt: now, ID: id,
		})
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errors.New("santri ini sudah tertaut ke lead lain")
			}
			return err
		}
		if n != 1 {
			return ErrLeadAlreadyConverted
		}
		if _, err := cancelActiveFollowups(ctx, q, id, "lead menjadi santri", now); err != nil {
			return err
		}
		if err := writeLeadEvent(ctx, q, id, newLeadEventID("usr"), "lead.converted", userActor(actorID), map[string]any{
			"santri_id": santriID, "linked_existing": !created, "status_from": LeadStatusOpen, "status_to": LeadStatusWon,
		}, now); err != nil {
			return err
		}
		if err := s.enqueuePurchase(ctx, q, id, l.Phone, l.PaymentAmount, now); err != nil {
			return err
		}
		if !created {
			return nil
		}
		admins, err := q.ListUserIDsByRole(ctx, "admin_kelas")
		if err != nil {
			return err
		}
		message := strings.TrimSpace(req.Nama) + " baru mendaftar lewat CS. Lengkapi data dan tempatkan di kelas."
		if _, label, ok, err := jadwalLabelByKode(ctx, q, l.JadwalKode); err != nil {
			return err
		} else if ok {
			message += " Jadwal pilihan (perlu dikonfirmasi): " + label + "."
		}
		return notifyUsers(ctx, q, admins, queries.CreateNotificationParams{
			Type:          "santri_baru_dari_lead",
			Title:         "Santri baru perlu penempatan kelas",
			Message:       message,
			ActionUrl:     "/app/perlu-dilengkapi",
			ReferenceType: "santri",
			ReferenceID:   sql.NullInt64{Int64: santriID, Valid: true},
			CreatedBy:     sql.NullInt64{Int64: actorID, Valid: true},
		})
	})
	if err != nil {
		return 0, err
	}
	return santriID, nil
}

// RenderPaymentVerifiedTemplate fills [NAMA] (first name, or "Kak") and
// [NOMINAL] (e.g. Rp149.000) in the confirmation template.
func RenderPaymentVerifiedTemplate(tpl string, l queries.Lead, amount int64) string {
	nama := aiFirstName(l.Nama, l.WaName)
	if nama == "" {
		nama = "Kak"
	}
	out := strings.ReplaceAll(strings.TrimSpace(tpl), "[NAMA]", nama)
	return strings.ReplaceAll(out, "[NOMINAL]", formatRupiahPlain(amount))
}

func formatRupiahPlain(n int64) string {
	digits := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return "Rp" + b.String()
}

// enqueuePurchase queues the Meta CAPI Purchase atomically with the CRM
// conversion, so an interrupted reconciliation worker cannot lose the
// milestone. The worker still owns HTTP delivery/retry. Preserves CAPI's
// opt-in and phone eligibility rules, plus test-mode attribution.
func (s *LeadService) enqueuePurchase(ctx context.Context, q *queries.Querier, id int64, phone string, paymentAmount int64, now time.Time) error {
	if s.metaCAPI == nil || !s.metaCAPI.Enabled() || NormalizeLeadPhone(phone) == "" {
		return nil
	}
	value, err := q.GetMetaCAPIPurchaseAmount(ctx, id)
	if err != nil {
		return err
	}
	if value <= 0 {
		value = paymentAmount
	}
	_, err = q.InsertMetaCAPIEvent(ctx, queries.InsertMetaCAPIEventParams{
		LeadID: id, EventName: MetaEventPurchase, EventID: metaEventID(id, MetaEventPurchase),
		EventTime: now, Value: max(value, int64(0)),
		TestMode:      boolInt(s.metaCAPI.cfg.TestEventCode != ""),
		NextAttemptAt: now, CreatedAt: now,
	})
	return err
}
