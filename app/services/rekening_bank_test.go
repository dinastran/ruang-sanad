package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestRekeningDariTeks(t *testing.T) {
	got := RekeningDariTeks("🏦 Bank Muamalat\nNo. Rek: 1110012352\nAtas Nama: Ruang Sanad Indonesia")
	require.Equal(t, []models.RekeningBank{{Bank: "Bank Muamalat", NoRekening: "1110012352", AtasNama: "Ruang Sanad Indonesia"}}, got)

	got = RekeningDariTeks("BSI 7123-456-789\na.n. Yayasan Ruang Sanad")
	require.Equal(t, []models.RekeningBank{{Bank: "BSI", NoRekening: "7123-456-789", AtasNama: "Yayasan Ruang Sanad"}}, got)

	require.Nil(t, RekeningDariTeks("Transfer ke rekening yayasan, hubungi CS"))
	require.Nil(t, RekeningDariTeks(""))
}

func TestNormalizeRekening(t *testing.T) {
	got, err := normalizeRekening([]models.RekeningBank{
		{Bank: " Bank  Muamalat ", NoRekening: "1110 0123 52", AtasNama: "Ruang Sanad"},
		{}, // empty row dropped
	})
	require.NoError(t, err)
	require.Equal(t, "Bank Muamalat", got[0].Bank)
	require.Equal(t, "1110012352", onlyDigits(got[0].NoRekening))
	require.Len(t, got, 1)

	_, err = normalizeRekening([]models.RekeningBank{{Bank: "BSI", NoRekening: "12ab34", AtasNama: "X"}})
	require.Error(t, err)
	_, err = normalizeRekening([]models.RekeningBank{{Bank: "BSI", NoRekening: "1234567", AtasNama: ""}})
	require.Error(t, err)

	// Settings round trip: structured accounts win over the free text.
	require.Equal(t, "BSI", rekeningEfektif(`[{"bank":"BSI","no_rekening":"7123456789","atas_nama":"Yayasan"}]`, "Bank Muamalat 1110012352")[0].Bank)
	require.Equal(t, "Bank Muamalat", rekeningEfektif("", "Bank Muamalat\nNo. Rek: 1110012352\nAtas Nama: RS")[0].Bank)
}

func TestProgramMenungguVerifikasiShowsUploadedProof(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	programs := NewLeadProgramService(f.querier)
	link, err := f.querier.GetRegistrationLinkBySlug(t.Context(), "tahsin-reguler")
	require.NoError(t, err)

	d, err := programs.Detail(link.ProgramID)
	require.NoError(t, err)
	require.Empty(t, d.MenungguVerifikasi, "no proof yet")

	require.NoError(t, f.reg.SubmitManualProof(t.Context(), ref, proofFile(pngHead, 1000)))
	d, err = programs.Detail(link.ProgramID)
	require.NoError(t, err)
	require.Len(t, d.MenungguVerifikasi, 1)
	m := d.MenungguVerifikasi[0]
	require.Equal(t, leadID, m.LeadID)
	require.Equal(t, "Fulanah", m.Nama)
	require.EqualValues(t, 150000, m.Amount)
	require.NotZero(t, m.PaymentID)

	proof, err := f.querier.LatestProofByLead(t.Context(), []int64{leadID, 999})
	require.NoError(t, err)
	require.Equal(t, map[int64]int64{leadID: m.PaymentID}, proof)
}

func TestBayarInfoRekeningFallback(t *testing.T) {
	f, ref, _ := bayarLead(t)
	info, err := f.reg.BayarInfo(t.Context(), ref)
	require.NoError(t, err)
	require.Empty(t, info.Rekening, `"BSI 123" has no real account number: page shows the free text`)
	require.Equal(t, "BSI 123 a.n. Yayasan", info.BankManualInfo)

	s, err := f.reg.Settings(t.Context())
	require.NoError(t, err)
	s.Rekening = []models.RekeningBank{
		{Bank: "Bank Muamalat", NoRekening: "1110012352", AtasNama: "Ruang Sanad Indonesia"},
		{Bank: "BSI", NoRekening: "7123456789", AtasNama: "Ruang Sanad Indonesia"},
	}
	require.NoError(t, f.reg.SaveSettings(t.Context(), s, f.adminID))
	info, err = f.reg.BayarInfo(t.Context(), ref)
	require.NoError(t, err)
	require.Equal(t, s.Rekening, info.Rekening)

	s.Rekening = []models.RekeningBank{{Bank: "BSI", NoRekening: "abc", AtasNama: "X"}}
	require.Error(t, f.reg.SaveSettings(t.Context(), s, f.adminID))
}
