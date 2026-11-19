package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestUjrohIncludesGuruAffiliateCommission(t *testing.T) {
	f := setupUjroh(t)
	now := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	res, err := f.db.Exec(`INSERT INTO affiliates (kode, tipe, guru_id, nama, status, created_at, updated_at) VALUES ('GURU', 'guru', ?, 'Guru Badal', 'aktif', ?, ?)`, f.guruBadalID, now, now)
	require.NoError(t, err)
	affID, _ := res.LastInsertId()
	for i, jml := range []int64{50000, 25000} {
		res, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, created_at, updated_at) VALUES (?, '62800', ?, ?)`, "c"+string(rune('a'+i))+"@s.whatsapp.net", now, now)
		require.NoError(t, err)
		lid, _ := res.LastInsertId()
		_, err = f.db.Exec(`INSERT INTO affiliate_commissions (affiliate_id, lead_id, sumber, sumber_id, jumlah, status, siap_at, created_at, updated_at)
			VALUES (?, ?, 'pendaftaran', ?, ?, 'siap', ?, ?, ?)`, affID, lid, lid, jml, now, now, now)
		require.NoError(t, err)
	}
	var komponenID int64
	require.NoError(t, f.db.QueryRow(`SELECT komponen_ujroh_id FROM affiliate_settings WHERE id = 1`).Scan(&komponenID))

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	g := findGuru(t, rekap, f.guruBadalID)
	var komisi int64
	for _, k := range g.Komponen {
		if k.KomponenID == komponenID {
			komisi = k.Total
		}
	}
	require.EqualValues(t, 75000, komisi)

	require.Error(t, f.svc.SetKomponenBulan("2026-09", f.guruBadalID, komponenID, models.SetUjrohKomponenBulanRequest{Jumlah: 1, Tarif: 1}, 1), "baris sistem tidak bisa diedit")

	require.NoError(t, f.svc.Kunci("2026-09", f.guruUtamaUser))
	var dibayar int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM affiliate_commissions WHERE status = 'dibayar' AND ujroh_bulan = '2026-09' AND metode_bayar = 'ujroh'`).Scan(&dibayar))
	require.Equal(t, 2, dibayar)
	rekap, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	g = findGuru(t, rekap, f.guruBadalID)
	require.EqualValues(t, 75000, g.TotalKomponen, "snapshot terkunci memuat komisi")

	require.NoError(t, f.svc.Buka("2026-09", f.guruUtamaUser))
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM affiliate_commissions WHERE status = 'siap' AND ujroh_bulan = ''`).Scan(&dibayar))
	require.Equal(t, 2, dibayar, "buka bulan: komisi kembali siap")
}

func TestMarkExternalTransferred(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	f.submitWith(t, "", "AHMAD")
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	now := f.clock
	res, err := f.db.Exec(`INSERT INTO affiliate_commissions (affiliate_id, lead_id, sumber, sumber_id, jumlah, status, created_at, updated_at) VALUES (?, ?, 'pendaftaran', ?, 50000, 'siap', ?, ?)`, a, l.ID, l.ID, now, now)
	require.NoError(t, err)
	siap, _ := res.LastInsertId()
	res, err = f.db.Exec(`INSERT INTO affiliate_commissions (affiliate_id, lead_id, sumber, sumber_id, jumlah, status, created_at, updated_at) VALUES (?, ?, 'spp', 999, 20000, 'pending', ?, ?)`, a, l.ID, now, now)
	require.NoError(t, err)
	pending, _ := res.LastInsertId()

	n, err := f.aff.MarkTransferred(t.Context(), []int64{siap, pending}, f.keuanganID, "TRF-01")
	require.NoError(t, err)
	require.Equal(t, 1, n, "hanya komisi siap yang bisa dicairkan")
	c, err := f.querier.GetAffiliateCommission(t.Context(), siap)
	require.NoError(t, err)
	require.Equal(t, "dibayar", c.Status)
	require.Equal(t, "transfer", c.MetodeBayar)
	require.Equal(t, "TRF-01", c.Catatan)
	n, err = f.aff.MarkTransferred(t.Context(), []int64{siap}, f.keuanganID, "")
	require.NoError(t, err)
	require.Zero(t, n, "idempoten")
}

func TestCancelCommissionManual(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	f.submitWith(t, "", "AHMAD")
	l, _ := f.querier.GetLeadByChatID(t.Context(), testChatID)
	res, err := f.db.Exec(`INSERT INTO affiliate_commissions (affiliate_id, lead_id, sumber, sumber_id, jumlah, status, created_at, updated_at) VALUES (?, ?, 'pendaftaran', ?, 50000, 'siap', ?, ?)`, a, l.ID, l.ID, f.clock, f.clock)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	require.Error(t, f.aff.CancelCommission(t.Context(), id, f.adminID, " "))
	require.NoError(t, f.aff.CancelCommission(t.Context(), id, f.adminID, "Santri ternyata titipan"))
	c, _ := f.querier.GetAffiliateCommission(t.Context(), id)
	require.Equal(t, "batal", c.Status)
}
