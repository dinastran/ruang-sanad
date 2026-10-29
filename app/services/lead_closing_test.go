package services

import (
	"errors"
	"sync"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// leadAtBuktiMasuk runs the example events from PRD Lampiran A.
func (f *leadFixture) leadAtBuktiMasuk(t *testing.T) int64 {
	t.Helper()
	f.ingest(t,
		leadEvent("evt_a1", "message.in", "2026-10-24T03:12:45Z", map[string]string{"wa_message_id": "A1", "body": "Assalamu'alaikum"}),
		leadEvent("evt_a3", "form.sent", "2026-10-24T03:20:10Z", map[string]string{}),
		leadEvent("evt_a3b", "message.in", "2026-10-24T03:25:00Z", map[string]string{"wa_message_id": "A3", "body": leadFormTemplateFilled}),
		leadEvent("evt_a4", "payment_info.sent", "2026-10-24T03:31:00Z", map[string]string{"paket_kode": "1x", "nominal_text": "Rp149.000"}),
		leadEvent("evt_a5", "proof.received", "2026-10-24T04:02:00Z", map[string]string{"media_type": "gambar"}),
	)
	return f.lead(t).ID
}

func (f *leadFixture) verified(t *testing.T) int64 {
	t.Helper()
	id := f.leadAtBuktiMasuk(t)
	require.NoError(t, f.service.VerifyPayment(t.Context(), id, f.csID, models.LeadVerifyRequest{Amount: 149000, Date: "2026-10-24"}))
	return id
}

func convertRequest() models.LeadConvertRequest {
	return models.LeadConvertRequest{
		Nama: "Fulanah binti Fulan", JenisKelamin: "P", Usia: 27, Domisili: "Bandung", NoWA: "6281234567890",
		KelasKode: "R 1X", Nominal: 149000, TanggalDaftar: "2026-10-24", Angkatan: "AKA39",
	}
}

func TestLeadVerifyPaymentRules(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.leadAtBuktiMasuk(t)

	err := f.service.VerifyPayment(t.Context(), id, f.csID, models.LeadVerifyRequest{Amount: 100000, Date: "2026-10-24"})
	require.ErrorContains(t, err, "catatan wajib")
	err = f.service.VerifyPayment(t.Context(), id, f.csID, models.LeadVerifyRequest{Amount: 149000, Date: "2026-12-01"})
	require.ErrorContains(t, err, "masa depan")

	require.NoError(t, f.service.VerifyPayment(t.Context(), id, f.keuanganID, models.LeadVerifyRequest{Amount: 100000, Date: "2026-10-24", Note: "transfer kurang, sisa via tunai"}))
	l := f.lead(t)
	require.Equal(t, LeadStageTerverifikasi, l.Stage)
	require.True(t, l.VerifiedAt.Valid)
	require.Equal(t, f.keuanganID, l.VerifiedBy.Int64)

	d, err := f.service.Detail(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, int64(100000), d.VerifiedAmount)
	require.Equal(t, "2026-10-24", d.VerifiedDate)
}

func TestLeadVerifyWithoutProofNeedsConfirmation(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("evt_1", "payment_info.sent", "2026-10-24T03:31:00Z", map[string]string{"paket_kode": "1x", "nominal_text": "Rp149.000"}))
	id := f.lead(t).ID
	err := f.service.VerifyPayment(t.Context(), id, f.csID, models.LeadVerifyRequest{Amount: 149000, Date: "2026-10-24"})
	require.ErrorContains(t, err, "bukti transfer belum tercatat")
	require.NoError(t, f.service.VerifyPayment(t.Context(), id, f.csID, models.LeadVerifyRequest{Amount: 149000, Date: "2026-10-24", NoProof: true}))
}

func TestLeadManualStageCannotReachTerverifikasi(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.leadAtBuktiMasuk(t)
	require.Error(t, f.service.ChangeStage(t.Context(), id, f.csID, LeadStageTerverifikasi, ""))
	// Manual moves may go backward.
	require.NoError(t, f.service.ChangeStage(t.Context(), id, f.csID, LeadStageDiscovery, ""))
	require.Equal(t, LeadStageDiscovery, f.lead(t).Stage)
	var payload string
	require.NoError(t, f.db.QueryRow(`SELECT payload FROM lead_events WHERE type = 'stage.changed'`).Scan(&payload))
	require.JSONEq(t, `{"stage_from":"bukti_masuk","stage_to":"discovery"}`, payload)
}

func TestLeadConvertCreatesOneSantriAndNotifiesAdminKelas(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.verified(t)

	santriID, err := f.service.Convert(t.Context(), id, f.csID, convertRequest())
	require.NoError(t, err)
	require.NotZero(t, santriID)

	l := f.lead(t)
	require.Equal(t, LeadStatusWon, l.Status)
	require.Equal(t, santriID, l.SantriID.Int64)
	require.True(t, l.ConvertedAt.Valid)

	// CRM conversion must enter the same onboarding pipeline as manual CS creation.
	var onboardingCount int
	require.NoError(t, f.db.QueryRow("SELECT COUNT(*) FROM onboarding_cases WHERE santri_id = ?", santriID).Scan(&onboardingCount))
	require.Equal(t, 1, onboardingCount)

	var nama, noWA string
	var lengkap int
	require.NoError(t, f.db.QueryRow(`SELECT nama, no_wa, is_lengkap FROM santri WHERE id = ?`, santriID).Scan(&nama, &noWA, &lengkap))
	require.Equal(t, "Fulanah binti Fulan", nama)
	require.Equal(t, "6281234567890", noWA)

	perlu, err := f.service.santri.ListPerluDilengkapi()
	require.NoError(t, err)
	found := false
	for _, s := range perlu {
		found = found || s.ID == santriID
	}
	require.True(t, found, "new santri appears in Perlu Dilengkapi")

	for _, admin := range []int64{f.adminKelas1, f.adminKelas2} {
		var n int
		require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = 'santri_baru_dari_lead' AND action_url = '/app/perlu-dilengkapi'`, admin).Scan(&n))
		require.Equal(t, 1, n)
	}

	_, err = f.service.Convert(t.Context(), id, f.csID, convertRequest())
	require.ErrorIs(t, err, ErrLeadAlreadyConverted)
	require.ErrorIs(t, f.service.UpdateData(t.Context(), id, f.csID, models.LeadUpdateRequest{Nama: "X"}), ErrLeadReadOnly)
}

func TestLeadConvertConcurrentClicksMakeOneSantri(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.verified(t)
	var before int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM santri`).Scan(&before))

	var wg sync.WaitGroup
	results := make([]error, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = f.service.Convert(t.Context(), id, f.csID, convertRequest())
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, err := range results {
		if err == nil {
			ok++
		} else {
			require.ErrorIs(t, err, ErrLeadAlreadyConverted)
		}
	}
	require.Equal(t, 1, ok)
	var after int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM santri`).Scan(&after))
	require.Equal(t, before+1, after)
}

func TestLeadConvertWarnsOnDuplicateNoWA(t *testing.T) {
	f := setupLeadFixture(t)
	_, err := f.db.Exec(`INSERT INTO santri (id_mahasantri, nama, jenis_kelamin, status, no_wa) VALUES ('MHS.X.0001.012026', 'Fulanah Lama', 'P', 'aktif', '081234567890')`)
	require.NoError(t, err)
	var existingID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM santri WHERE nama = 'Fulanah Lama'`).Scan(&existingID))
	id := f.verified(t)

	opts, err := f.service.ConvertOptions(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, opts.Duplicates, 1)
	require.Equal(t, "AKA39", opts.Angkatan[0].Kode, "newest active angkatan first")

	_, err = f.service.Convert(t.Context(), id, f.csID, convertRequest())
	var dup *ErrLeadDuplicateSantri
	require.True(t, errors.As(err, &dup))
	require.Contains(t, err.Error(), "Fulanah Lama")
	require.Equal(t, LeadStatusOpen, f.lead(t).Status, "nothing changed")

	// Link to the existing santri: no new santri, no admin_kelas notification.
	santriID, err := f.service.Convert(t.Context(), id, f.csID, models.LeadConvertRequest{LinkSantriID: existingID})
	require.NoError(t, err)
	require.Equal(t, existingID, santriID)
	var notif int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type = 'santri_baru_dari_lead'`).Scan(&notif))
	require.Zero(t, notif)
	var cases int
	require.NoError(t, f.db.QueryRow("SELECT COUNT(*) FROM onboarding_cases WHERE santri_id = ?", existingID).Scan(&cases))
	require.Zero(t, cases, "linking a historical santri must not create onboarding")
}

func TestLeadConvertForceNewDespiteDuplicate(t *testing.T) {
	f := setupLeadFixture(t)
	_, err := f.db.Exec(`INSERT INTO santri (id_mahasantri, nama, jenis_kelamin, status, no_wa) VALUES ('MHS.X.0001.012026', 'Saudara', 'L', 'aktif', '6281234567890')`)
	require.NoError(t, err)
	id := f.verified(t)
	req := convertRequest()
	req.ForceNew = true
	_, err = f.service.Convert(t.Context(), id, f.csID, req)
	require.NoError(t, err)
}

func TestLeadConvertRequiresVerification(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.leadAtBuktiMasuk(t)
	_, err := f.service.Convert(t.Context(), id, f.csID, convertRequest())
	require.ErrorIs(t, err, ErrLeadNotVerified)
}

func TestLeadStatusLostAndReopen(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.leadAtBuktiMasuk(t)
	require.Error(t, f.service.ChangeStatus(t.Context(), id, f.csID, models.LeadStatusRequest{Status: LeadStatusLost}))
	require.NoError(t, f.service.ChangeStatus(t.Context(), id, f.csID, models.LeadStatusRequest{Status: LeadStatusLost, LostReason: "harga"}))
	l := f.lead(t)
	require.Equal(t, LeadStatusLost, l.Status)
	require.Equal(t, LeadStageBuktiMasuk, l.Stage, "stage kept")
	require.NoError(t, f.service.ChangeStatus(t.Context(), id, f.csID, models.LeadStatusRequest{Status: LeadStatusDoNotContact}))
	require.Equal(t, "no", f.lead(t).FollowupPermission)
	require.NoError(t, f.service.ChangeStatus(t.Context(), id, f.csID, models.LeadStatusRequest{Status: LeadStatusOpen}))
	require.Equal(t, LeadStatusOpen, f.lead(t).Status)
}

func TestLeadAssignRules(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.leadAtBuktiMasuk(t)
	require.Error(t, f.service.Assign(t.Context(), id, f.csID, "cs", f.keuanganID), "cs can only take for self")
	require.NoError(t, f.service.Assign(t.Context(), id, f.csID, "cs", f.csID))
	require.Equal(t, f.csID, f.lead(t).AssignedUserID.Int64)
	require.Error(t, f.service.Assign(t.Context(), id, f.csID, "super_admin", f.adminKelas1), "admin_kelas cannot own leads")
}

func TestLeadListFiltersAndCursor(t *testing.T) {
	f := setupLeadFixture(t)
	for i := 0; i < 35; i++ {
		ev := leadEvent("evt_"+string(rune('a'+i%26))+string(rune('a'+i/26)), "message.in", "2026-10-24T03:00:00Z", map[string]string{"body": "halo"})
		ev.ChatID = "62812000000" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + "@s.whatsapp.net"
		ev.Phone = ""
		ev.OccurredAt = "2026-10-2" + string(rune('0'+i%5)) + "T0" + string(rune('0'+i%9)) + ":00:00Z"
		f.ingest(t, ev)
	}
	// One lead with transfer proof always sorts first.
	proof := leadEvent("evt_proof", "proof.received", "2026-10-20T01:00:00Z", map[string]string{})
	proof.ChatID = "6281200000007@s.whatsapp.net"
	f.ingest(t, proof)

	page1, err := f.service.List(t.Context(), models.LeadListParams{})
	require.NoError(t, err)
	require.Len(t, page1.Items, leadListPageSize)
	require.NotEmpty(t, page1.NextCursor)
	require.Equal(t, LeadStageBuktiMasuk, page1.Items[0].Stage)

	page2, err := f.service.List(t.Context(), models.LeadListParams{Cursor: page1.NextCursor})
	require.NoError(t, err)
	require.Len(t, page2.Items, 5)
	require.Empty(t, page2.NextCursor)
	seen := map[int64]bool{}
	for _, it := range append(page1.Items, page2.Items...) {
		require.False(t, seen[it.ID], "no duplicates across pages")
		seen[it.ID] = true
	}

	byStage, err := f.service.List(t.Context(), models.LeadListParams{Stage: LeadStageBuktiMasuk})
	require.NoError(t, err)
	require.Len(t, byStage.Items, 1)

	search, err := f.service.List(t.Context(), models.LeadListParams{Search: "0812000000 07"})
	require.NoError(t, err)
	require.Len(t, search.Items, 1)

	mine, err := f.service.List(t.Context(), models.LeadListParams{Mine: true, UserID: f.csID})
	require.NoError(t, err)
	require.Empty(t, mine.Items)
}

func TestLeadReportFunnel(t *testing.T) {
	f := setupLeadFixture(t)
	id := f.verified(t)
	_, err := f.service.Convert(t.Context(), id, f.csID, convertRequest())
	require.NoError(t, err)
	other := leadEvent("evt_other", "message.in", "2026-10-24T06:00:00Z", map[string]string{"body": "halo"})
	other.ChatID = "6289999999999@s.whatsapp.net"
	f.ingest(t, other)

	rep, err := f.service.Report(t.Context(), "2026-10-01", "2026-10-31")
	require.NoError(t, err)
	require.Equal(t, int64(2), rep.TotalLeads)
	require.Equal(t, int64(1), rep.Won)
	require.Equal(t, int64(2), rep.Funnel[0].Reached)
	require.Equal(t, int64(1), rep.Funnel[len(rep.Funnel)-1].Reached)
}
