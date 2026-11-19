package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// paidAffiliateLead: lead attributed to affiliate "AHMAD", verified 150000.
func paidAffiliateLead(t *testing.T, rule models.AffiliateRuleRequest) (*affiliateFixture, int64, int64) {
	t.Helper()
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	require.NoError(t, f.aff.SaveRule(t.Context(), rule, f.adminID))
	f.submitWith(t, "", "AHMAD")
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	f.clock = f.clock.Add(time.Minute)
	_, err = f.db.Exec(`UPDATE leads SET stage = 'bukti_masuk', payment_amount = 150000 WHERE id = ?`, l.ID)
	require.NoError(t, err)
	require.NoError(t, f.service.VerifyPayment(t.Context(), l.ID, f.keuanganID, models.LeadVerifyRequest{
		Amount: 150000, Date: f.clock.In(wib).Format("2006-01-02"),
	}))
	return f, a, l.ID
}

// convert links the lead to a new santri at the fixture clock.
func (f *affiliateFixture) convert(t *testing.T, leadID int64) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO santri (nama, status) VALUES ('Fulanah', 'aktif')`)
	require.NoError(t, err)
	sid, _ := res.LastInsertId()
	_, err = f.db.Exec(`UPDATE leads SET santri_id = ?, status = 'won', converted_at = ? WHERE id = ?`, sid, f.clock, leadID)
	require.NoError(t, err)
	return sid
}

func (f *affiliateFixture) tagihan(t *testing.T, santriID, bulanKe, nominal int64, status string, created time.Time) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO tagihan (santri_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, status, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?, ?, ?, ?)`, santriID, bulanKe, nominal, created.Format("2006-01-02"), status, created, created)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return id
}

func (f *affiliateFixture) reconcile(t *testing.T) {
	t.Helper()
	_, err := f.aff.Reconcile(t.Context())
	require.NoError(t, err)
}

func (f *affiliateFixture) commissions(t *testing.T) []struct {
	Sumber        string
	BulanKe, Jml  int64
	Status        string
} {
	t.Helper()
	rows, err := f.db.Query(`SELECT sumber, bulan_ke, jumlah, status FROM affiliate_commissions ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var out []struct {
		Sumber       string
		BulanKe, Jml int64
		Status       string
	}
	for rows.Next() {
		var r struct {
			Sumber       string
			BulanKe, Jml int64
			Status       string
		}
		require.NoError(t, rows.Scan(&r.Sumber, &r.BulanKe, &r.Jml, &r.Status))
		out = append(out, r)
	}
	return out
}

func TestReconcilePendaftaranPersenIdempotent(t *testing.T) {
	f, _, _ := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "persen", Nilai: 1000, Pola: "sekali"})
	f.reconcile(t)
	f.reconcile(t)
	c := f.commissions(t)
	require.Len(t, c, 1)
	require.Equal(t, "pendaftaran", c[0].Sumber)
	require.EqualValues(t, 15000, c[0].Jml, "10% x 150000")
	require.Equal(t, "pending", c[0].Status)

	f.clock = f.clock.Add(15 * 24 * time.Hour)
	f.reconcile(t)
	require.Equal(t, "siap", f.commissions(t)[0].Status, "lewat masa tahan 14 hari")
}

func TestReconcileNoRuleNoCommission(t *testing.T) {
	f := setupAffiliateFixture(t)
	f.external(t, "Ahmad", "081200000001")
	f.submitWith(t, "", "AHMAD")
	f.reconcile(t)
	require.Empty(t, f.commissions(t))
}

func TestReconcileVerificationUndoneAndRedone(t *testing.T) {
	f, _, leadID := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "sekali"})
	f.reconcile(t)
	require.NoError(t, f.service.ChangeStage(t.Context(), leadID, f.csID, LeadStageBuktiMasuk, ""))
	f.reconcile(t)
	require.Equal(t, "batal", f.commissions(t)[0].Status)
	require.NoError(t, f.service.VerifyPayment(t.Context(), leadID, f.keuanganID, models.LeadVerifyRequest{Amount: 150000, Date: f.clock.In(wib).Format("2006-01-02")}))
	f.reconcile(t)
	c := f.commissions(t)
	require.Len(t, c, 1, "baris yang sama diaktifkan kembali")
	require.Equal(t, "pending", c[0].Status)
}

func TestReconcileSantriStopsDuringHold(t *testing.T) {
	f, _, leadID := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "sekali"})
	sid := f.convert(t, leadID)
	f.reconcile(t)
	_, err := f.db.Exec(`UPDATE santri SET status = 'tidak_lanjut' WHERE id = ?`, sid)
	require.NoError(t, err)
	f.clock = f.clock.Add(15 * 24 * time.Hour)
	f.reconcile(t)
	require.Equal(t, "batal", f.commissions(t)[0].Status)
}

func TestReconcileMonthlyFromConversion(t *testing.T) {
	f, _, leadID := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "persen", Nilai: 1000, Pola: "bulanan", MaksBulan: ptrInt64(3)})
	before := f.clock.Add(-30 * 24 * time.Hour)
	sid := f.convert(t, leadID)
	old := f.tagihan(t, sid, 13, 200000, "lunas", before) // returning santri: old bill
	_ = old
	t1 := f.tagihan(t, sid, 14, 200000, "lunas", f.clock.Add(24*time.Hour))
	_ = t1
	t15 := f.tagihan(t, sid, 15, 200000, "belum_bayar", f.clock.Add(48*time.Hour))
	f.tagihan(t, sid, 16, 200000, "lunas", f.clock.Add(72*time.Hour))
	f.tagihan(t, sid, 17, 200000, "lunas", f.clock.Add(96*time.Hour))
	f.reconcile(t)
	sppMonths := func() []int64 {
		var out []int64
		for _, r := range f.commissions(t) {
			if r.Sumber == "spp" {
				out = append(out, r.BulanKe)
			}
		}
		return out
	}
	require.Equal(t, []int64{2}, sppMonths(), "bulan 3 belum lunas; bulan 4 di atas batas 3")
	_, err := f.db.Exec(`UPDATE tagihan SET status = 'lunas' WHERE id = ?`, t15)
	require.NoError(t, err)
	f.reconcile(t)
	f.reconcile(t)
	c := f.commissions(t)
	var spp []int64
	for _, r := range c {
		if r.Sumber == "spp" {
			spp = append(spp, r.BulanKe)
			require.EqualValues(t, 20000, r.Jml)
		}
	}
	require.ElementsMatch(t, []int64{2, 3}, spp, "tagihan lama diabaikan; bulan dihitung sejak konversi; maks 3 bulan")
}

func TestReconcileSekaliNoMonthly(t *testing.T) {
	f, _, leadID := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "sekali"})
	sid := f.convert(t, leadID)
	f.tagihan(t, sid, 2, 200000, "lunas", f.clock.Add(24*time.Hour))
	f.reconcile(t)
	require.Len(t, f.commissions(t), 1)
}

func TestReconcileRuleSnapshot(t *testing.T) {
	f, _, leadID := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "bulanan"})
	f.reconcile(t)
	require.NoError(t, f.aff.SaveRule(t.Context(), models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 99000, Pola: "sekali"}, f.adminID))
	sid := f.convert(t, leadID)
	f.tagihan(t, sid, 2, 200000, "lunas", f.clock.Add(24*time.Hour))
	f.reconcile(t)
	c := f.commissions(t)
	require.Len(t, c, 2, "aturan baru tidak berlaku mundur")
	require.EqualValues(t, 50000, c[1].Jml)
}

func TestReconcileInactiveAffiliateNoNewCommission(t *testing.T) {
	f, a, _ := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "sekali"})
	require.NoError(t, f.aff.SetStatus(t.Context(), a, "nonaktif", f.adminID))
	f.reconcile(t)
	require.Empty(t, f.commissions(t))
}
