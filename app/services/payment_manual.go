package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const MaxProofSize = 2 << 20 // 2 MB

var (
	ErrRefTidakAda   = errors.New("tautan pembayaran tidak dikenal")
	ErrSudahDiproses = errors.New("pembayaran untuk pendaftaran ini sudah diproses, silakan hubungi CS")
)

// ProofFile is an uploaded payment proof: Head holds the first bytes for
// type sniffing; Save writes the whole file to dst.
type ProofFile struct {
	Name string
	Size int64
	Head []byte
	Save func(dst string) error
}

// DetectProofType sniffs JPG/PNG/WebP/PDF from magic bytes.
func DetectProofType(head []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}):
		return "jpg", true
	case bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return "png", true
	case len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return "webp", true
	case bytes.HasPrefix(head, []byte("%PDF-")):
		return "pdf", true
	}
	return "", false
}

func firstName(nama string) string {
	f := strings.Fields(nama)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

func (s *RegistrationService) BayarInfo(ctx context.Context, ref string) (*models.BayarInfo, error) {
	reg, err := s.querier.GetLeadRegistrationByRef(ctx, ref)
	if err != nil {
		return nil, ErrRefTidakAda
	}
	l, err := s.querier.GetLeadByID(ctx, reg.LeadID)
	if err != nil {
		return nil, ErrRefTidakAda
	}
	link, err := s.querier.GetRegistrationLink(ctx, reg.LinkID)
	if err != nil {
		return nil, ErrRefTidakAda
	}
	settings, err := s.Settings(ctx)
	if err != nil {
		return nil, err
	}
	out := &models.BayarInfo{
		LeadRef: leadRef(l.ID), NamaDepan: firstName(l.Nama), Nominal: l.PaymentAmount,
		BankManualInfo: settings.BankManualInfo, IzinkanManual: link.IzinkanManual == 1, Status: "belum",
	}
	if prog, err := s.querier.GetLeadProgram(ctx, link.ProgramID); err == nil && prog != nil {
		out.ProgramNama = prog.Nama
	}
	out.WaURL = waURL(settings.CsWaNumber, "Assalamu'alaikum, saya "+out.NamaDepan+" ("+out.LeadRef+") ingin konfirmasi pembayaran.")
	out.GatewayAktif = s.gateway != nil
	out.Meta = s.bayarMeta(link, l)
	if l.Stage == LeadStageTerverifikasi || l.Status == LeadStatusWon {
		out.Status = "lunas"
		return out, nil
	}
	if _, err := s.querier.GetActiveGatewayPaymentByLead(ctx, queries.GetActiveGatewayPaymentByLeadParams{
		LeadID: l.ID, Now: sql.NullTime{Time: s.nowUTC(), Valid: true},
	}); err == nil {
		out.SesiAktif = true
	}
	pays, err := s.querier.ListPaymentsByLead(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range pays {
		if p.Metode == "gateway" && p.Status != "paid" {
			continue
		}
		switch p.Status {
		case "menunggu_verifikasi":
			out.Status = "menunggu_verifikasi"
		case "ditolak":
			out.Status, out.CatatanTolak = "ditolak", p.CatatanSistem
		case "paid":
			out.Status = "lunas"
		}
		break
	}
	return out, nil
}

// SubmitManualProof stores a transfer proof for the registration's lead and
// moves it to bukti_masuk. A second upload while waiting replaces the file.
func (s *RegistrationService) SubmitManualProof(ctx context.Context, ref string, file ProofFile) error {
	if file.Size <= 0 || file.Size > MaxProofSize {
		return &FieldError{Field: "bukti", Pesan: "Ukuran file maksimal 2 MB"}
	}
	ext, ok := DetectProofType(file.Head)
	if !ok {
		return &FieldError{Field: "bukti", Pesan: "File harus berupa foto (JPG, PNG, WebP) atau PDF"}
	}
	reg, err := s.querier.GetLeadRegistrationByRef(ctx, ref)
	if err != nil {
		return ErrRefTidakAda
	}
	link, err := s.querier.GetRegistrationLink(ctx, reg.LinkID)
	if err != nil {
		return ErrRefTidakAda
	}
	if link.IzinkanManual != 1 {
		return errors.New("transfer manual tidak tersedia untuk pendaftaran ini")
	}
	if err := os.MkdirAll(s.proofDir, 0o750); err != nil {
		return err
	}
	name := newRegistrationRef()[:24] + "." + ext
	dst := filepath.Join(s.proofDir, name)
	if err := file.Save(dst); err != nil {
		return err
	}
	var oldFile string
	err = s.leads.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, reg.LeadID)
		if err != nil {
			return err
		}
		if l.Status != LeadStatusOpen || l.Stage == LeadStageTerverifikasi {
			return ErrSudahDiproses
		}
		now := s.nowUTC()
		pending, err := q.ListPendingManualPaymentsByLead(ctx, l.ID)
		if err != nil {
			return err
		}
		if len(pending) > 0 {
			oldFile = pending[0].BuktiPath
			return q.UpdatePaymentProof(ctx, queries.UpdatePaymentProofParams{BuktiPath: name, Amount: l.PaymentAmount, UpdatedAt: now, ID: pending[0].ID})
		}
		if _, err := q.CreatePayment(ctx, queries.CreatePaymentParams{
			LeadID: l.ID, LinkID: sql.NullInt64{Int64: link.ID, Valid: true}, Metode: "manual",
			Amount: l.PaymentAmount, Status: "menunggu_verifikasi", BuktiPath: name, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		l.ProofReceivedAt = sql.NullTime{Time: now, Valid: true}
		actor := formActor(link.ID)
		if leadStageIndex(l.Stage) < leadStageIndex(LeadStageBuktiMasuk) {
			from := l.Stage
			l.Stage = LeadStageBuktiMasuk
			if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("frm"), "stage.changed", actor,
				map[string]any{"stage_from": from, "stage_to": LeadStageBuktiMasuk, "via": "bukti_manual"}, now); err != nil {
				return err
			}
		}
		if err := saveLead(ctx, q, l, now); err != nil {
			return err
		}
		ids, err := q.ListUserIDsByRole(ctx, "keuangan")
		if err != nil {
			return err
		}
		for _, uid := range ids {
			if _, err := q.CreateNotification(ctx, queries.CreateNotificationParams{
				UserID: uid, Type: "bukti_transfer", Title: "Bukti transfer masuk",
				Message:   leadDisplayName(l) + " (" + leadRef(l.ID) + ") mengunggah bukti transfer " + rupiahLabel(l.PaymentAmount) + ".",
				ActionUrl: "/app/leads/" + strconv.FormatInt(l.ID, 10), ReferenceType: "lead", ReferenceID: sql.NullInt64{Int64: l.ID, Valid: true},
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_ = os.Remove(dst)
		return err
	}
	if oldFile != "" {
		_ = os.Remove(filepath.Join(s.proofDir, filepath.Base(oldFile)))
	}
	return nil
}

// RejectManualPayment marks a waiting manual proof as rejected and moves the
// lead back to info_bayar_dikirim so the registrant can upload again.
func (s *RegistrationService) RejectManualPayment(ctx context.Context, leadID, paymentID, actorID int64, alasan string) error {
	alasan = truncateRunes(strings.TrimSpace(alasan), 300)
	if alasan == "" {
		return errors.New("alasan penolakan wajib diisi")
	}
	p, err := s.querier.GetPayment(ctx, paymentID)
	if err != nil || p.LeadID != leadID {
		return errors.New("pembayaran tidak ditemukan")
	}
	if p.Metode != "manual" || p.Status != "menunggu_verifikasi" {
		return errors.New("hanya bukti transfer yang menunggu verifikasi yang bisa ditolak")
	}
	return s.leads.mutateLead(ctx, p.LeadID, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		if err := q.UpdatePaymentStatus(ctx, queries.UpdatePaymentStatusParams{
			Status: "ditolak", CatatanSistem: alasan, UpdatedAt: now, ID: p.ID,
		}); err != nil {
			return "", nil, err
		}
		payload := map[string]any{"payment_id": p.ID, "alasan": alasan}
		if l.Status == LeadStatusOpen && l.Stage == LeadStageBuktiMasuk {
			l.Stage = LeadStageInfoBayarDikirim
			l.ProofReceivedAt = sql.NullTime{}
			// Not stage_from/stage_to: Meta CAPI reconcile treats those as milestones.
			payload["tahap_dari"], payload["tahap_ke"] = LeadStageBuktiMasuk, LeadStageInfoBayarDikirim
		}
		return "payment.rejected", payload, nil
	})
}

// flagPendingManualPayments marks waiting manual proofs for review when the
// lead was paid another way (gateway): the registrant may have paid twice.
func flagPendingManualPayments(ctx context.Context, q *queries.Querier, leadID int64, now time.Time) error {
	pending, err := q.ListPendingManualPaymentsByLead(ctx, leadID)
	if err != nil {
		return err
	}
	for _, p := range pending {
		if err := q.FlagPaymentReview(ctx, queries.FlagPaymentReviewParams{
			CatatanSistem: "Lead sudah lunas lewat pembayaran online; cek kemungkinan transfer ganda.", UpdatedAt: now, ID: p.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// markManualPaymentsPaid is called by VerifyPayment inside its transaction.
func markManualPaymentsPaid(ctx context.Context, q *queries.Querier, leadID int64, now time.Time) error {
	pending, err := q.ListPendingManualPaymentsByLead(ctx, leadID)
	if err != nil {
		return err
	}
	for _, p := range pending {
		if err := q.UpdatePaymentStatus(ctx, queries.UpdatePaymentStatusParams{
			Status: "paid", PaidAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: now, ID: p.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}
