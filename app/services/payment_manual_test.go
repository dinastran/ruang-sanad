package services

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

var (
	pngHead  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	jpgHead  = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0, 0}
	webpHead = []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'}
	pdfHead  = []byte("%PDF-1.7\n%..")
)

func TestDetectProofType(t *testing.T) {
	for head, want := range map[string]string{string(pngHead): "png", string(jpgHead): "jpg", string(webpHead): "webp", string(pdfHead): "pdf"} {
		ext, ok := DetectProofType([]byte(head))
		require.True(t, ok)
		require.Equal(t, want, ext)
	}
	_, ok := DetectProofType([]byte("MZ\x90\x00 this is an exe"))
	require.False(t, ok)
	_, ok = DetectProofType(nil)
	require.False(t, ok)
}

func proofFile(head []byte, size int64) ProofFile {
	return ProofFile{
		Name: "bukti.png", Size: size, Head: head,
		Save: func(dst string) error { return os.WriteFile(dst, head, 0o600) },
	}
}

// bayarLead submits a "bayar" link and returns the bayar ref + lead id.
func bayarLead(t *testing.T) (*registrationFixture, string, int64) {
	t.Helper()
	f, _ := setupSubmitFixture(t, "bayar")
	res, err := f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	return f, res.BayarRef, l.ID
}

func TestBayarInfo(t *testing.T) {
	f, ref, _ := bayarLead(t)
	info, err := f.reg.BayarInfo(t.Context(), ref)
	require.NoError(t, err)
	require.Equal(t, "Fulanah", info.NamaDepan)
	require.EqualValues(t, 150000, info.Nominal)
	require.Equal(t, "belum", info.Status)
	require.Contains(t, info.BankManualInfo, "BSI")
	_, err = f.reg.BayarInfo(t.Context(), "tidak-ada")
	require.True(t, errors.Is(err, ErrRefTidakAda))
}

func TestSubmitManualProof(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))

	l, err := f.querier.GetLeadByID(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, LeadStageBuktiMasuk, l.Stage)
	require.True(t, l.ProofReceivedAt.Valid)
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Len(t, pays, 1)
	require.Equal(t, "menunggu_verifikasi", pays[0].Status)
	require.EqualValues(t, 150000, pays[0].Amount)
	_, err = os.Stat(filepath.Join(f.reg.proofDir, pays[0].BuktiPath))
	require.NoError(t, err)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND reference_type = 'lead' AND reference_id = ?`, f.keuanganID, leadID))

	info, err := f.reg.BayarInfo(t.Context(), ref)
	require.NoError(t, err)
	require.Equal(t, "menunggu_verifikasi", info.Status)
}

func TestSubmitManualProofRejectsBadFiles(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	var fe *FieldError
	err := f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, MaxProofSize+1))
	require.True(t, errors.As(err, &fe), "terlalu besar")
	err = f.reg.SubmitManualProof(t.Context(), ref, proofFile([]byte("MZ\x90\x00 exe file.."), 100))
	require.True(t, errors.As(err, &fe), "bukan gambar/pdf")
	l, err := f.querier.GetLeadByID(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, LeadStageInfoBayarDikirim, l.Stage)
	require.Zero(t, f.count(t, `SELECT COUNT(*) FROM payments`))
	entries, _ := os.ReadDir(f.reg.proofDir)
	require.Empty(t, entries)
}

func TestSubmitManualProofReplace(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pdfHead, 1000)))
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Len(t, pays, 1)
	require.True(t, bytes.HasSuffix([]byte(pays[0].BuktiPath), []byte(".pdf")))
	entries, _ := os.ReadDir(f.reg.proofDir)
	require.Len(t, entries, 1, "file lama dihapus")
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM notifications WHERE user_id = ?`, f.keuanganID))
}

func TestSubmitManualProofAfterVerified(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	_, err := f.db.Exec(`UPDATE leads SET stage = 'terverifikasi', verified_at = CURRENT_TIMESTAMP WHERE id = ?`, leadID)
	require.NoError(t, err)
	err = f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000))
	require.True(t, errors.Is(err, ErrSudahDiproses))
}

func TestVerifyPaymentMarksManualPaid(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))
	require.NoError(t, f.service.VerifyPayment(t.Context(), leadID, f.keuanganID, models.LeadVerifyRequest{
		Amount: 150000, Date: f.clock.In(wib).Format("2006-01-02"),
	}))
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, "paid", pays[0].Status)
	require.True(t, pays[0].PaidAt.Valid)
	info, err := f.reg.BayarInfo(t.Context(), ref)
	require.NoError(t, err)
	require.Equal(t, "lunas", info.Status)
}

func TestRejectManualPayment(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Error(t, f.reg.RejectManualPayment(t.Context(), pays[0].ID, f.keuanganID, " "), "alasan wajib")
	require.NoError(t, f.reg.RejectManualPayment(t.Context(), pays[0].ID, f.keuanganID, "Nominal kurang"))

	pays, err = f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, "ditolak", pays[0].Status)
	require.Equal(t, "Nominal kurang", pays[0].CatatanSistem)
	l, err := f.querier.GetLeadByID(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, LeadStageInfoBayarDikirim, l.Stage)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM lead_events WHERE lead_id = ? AND type = 'payment.rejected'`, leadID))

	info, err := f.reg.BayarInfo(t.Context(), ref)
	require.NoError(t, err)
	require.Equal(t, "ditolak", info.Status)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(jpgHead, 1000)), "boleh unggah ulang setelah ditolak")
}

func TestLeadRegistrationPanelAndProofPath(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))
	panel, err := f.reg.LeadPanel(t.Context(), leadID)
	require.NoError(t, err)
	require.Len(t, panel.Registrasi, 1)
	require.Equal(t, "tahsin-reguler", panel.Registrasi[0].LinkSlug)
	require.Len(t, panel.Payments, 1)
	require.Equal(t, "menunggu_verifikasi", panel.Payments[0].Status)
	require.True(t, panel.Payments[0].AdaBukti)

	path, err := f.reg.ProofPath(t.Context(), leadID, panel.Payments[0].ID)
	require.NoError(t, err)
	require.Equal(t, f.reg.proofDir, filepath.Dir(path))
	_, err = f.reg.ProofPath(t.Context(), leadID+1, panel.Payments[0].ID)
	require.Error(t, err, "pembayaran milik lead lain")
}
