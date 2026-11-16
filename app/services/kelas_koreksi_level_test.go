package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type labelPertemuan struct {
	global  int64
	level   string
	levelKe int64
}

func (f perubahanFixture) labelPertemuan(t *testing.T) []labelPertemuan {
	t.Helper()
	rows, err := f.db.Query(`SELECT pertemuan_ke, level_nama, pertemuan_level_ke FROM pertemuan WHERE kelas_id = ? ORDER BY pertemuan_ke`, f.kelasID)
	require.NoError(t, err)
	defer rows.Close()
	var out []labelPertemuan
	for rows.Next() {
		var l labelPertemuan
		require.NoError(t, rows.Scan(&l.global, &l.level, &l.levelKe))
		out = append(out, l)
	}
	require.NoError(t, rows.Err())
	return out
}

// The real case: the class moved to Talaqqi 4 meetings before Admin Kelas
// changed its level, so Talaqqi restarted at 1 and the dashboard showed
// "Talaqqi ke-2" while the real number is 6. Correcting it relabels the 4
// meetings as Talaqqi 1–4 without touching global numbers or billing.
func TestKoreksiNomorLevelMundurkanAwalLevel(t *testing.T) {
	f := setupKelasPerubahan(t)
	ctx := context.Background()
	require.NoError(t, NewKelasService(f.querier).SetPertemuanTerakhir(f.kelasID, 10))
	for range 6 { // PT 11–16; the last 4 were actually Talaqqi
		f.buatPertemuan(t, "selesai")
	}
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "03", f.adminID))
	f.buatPertemuan(t, "selesai") // TALAQQI ke-1 (should be 5)
	f.buatPertemuan(t, "selesai") // TALAQQI ke-2 (should be 6)
	_, err := f.db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, status) VALUES (?, ?, 1, 17, 150000, '2026-09-01', 'belum_bayar')`, f.santri[0], f.kelasID)
	require.NoError(t, err)

	state, err := f.service.GetKoreksiLevelState(f.kelasID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, "TALAQQI", state.LevelNama)
	require.EqualValues(t, 2, state.TerakhirSekarang)
	require.EqualValues(t, 18, state.TerakhirGlobal)
	require.EqualValues(t, 18, state.MaksTerakhir, "no earlier level change: may go back to the legacy anchor")

	require.ErrorContains(t, f.service.KoreksiNomorLevel(f.kelasID, 6, f.adminID, " "), "alasan")
	require.ErrorContains(t, f.service.KoreksiNomorLevel(f.kelasID, 2, f.adminID, "x"), "belum berubah")
	require.ErrorContains(t, f.service.KoreksiNomorLevel(f.kelasID, -15, f.adminID, "x"), "negatif")

	require.NoError(t, f.service.KoreksiNomorLevel(f.kelasID, 6, f.adminID, "Talaqqi sudah mulai 4 pertemuan sebelum level diganti"))

	require.Equal(t, []labelPertemuan{
		{11, "PT", 11}, {12, "PT", 12},
		{13, "TALAQQI", 1}, {14, "TALAQQI", 2}, {15, "TALAQQI", 3}, {16, "TALAQQI", 4},
		{17, "TALAQQI", 5}, {18, "TALAQQI", 6},
	}, f.labelPertemuan(t))

	next, err := f.querier.GetNextPertemuanLevelKe(ctx, f.kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 7, next)
	p := f.buatPertemuan(t, "selesai")
	require.EqualValues(t, 19, p.PertemuanKe, "global numbering unchanged")
	require.EqualValues(t, 7, p.PertemuanLevelKe)

	var tagihanKe, boundary int64
	require.NoError(t, f.db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE kelas_id = ?`, f.kelasID).Scan(&tagihanKe))
	require.EqualValues(t, 17, tagihanKe, "billing untouched")
	require.NoError(t, f.db.QueryRow(`SELECT pertemuan_ke FROM kelas_perubahan WHERE kelas_id = ? AND jenis = 'level_kelas'`, f.kelasID).Scan(&boundary))
	require.EqualValues(t, 12, boundary, "level change record follows the real start")

	var lama, baru, alasan string
	require.NoError(t, f.db.QueryRow(`SELECT nilai_lama, nilai_baru, alasan FROM kelas_perubahan WHERE kelas_id = ? AND jenis = ?`, f.kelasID, PerubahanKoreksiLevel).Scan(&lama, &baru, &alasan))
	require.Equal(t, "TALAQQI ke-2", lama)
	require.Equal(t, "TALAQQI ke-6", baru)
	var notif int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND title = 'Nomor pertemuan dikoreksi'`, f.guruUser).Scan(&notif))
	require.EqualValues(t, 1, notif)

	// The old global correction keeps working afterwards.
	require.NoError(t, f.service.KoreksiNomorPertemuan(f.kelasID, 20, f.adminID, "anchor arsip"))
	got := f.labelPertemuan(t)
	require.Equal(t, labelPertemuan{21, "PT", 21}, got[0])
	require.Equal(t, labelPertemuan{23, "TALAQQI", 1}, got[2])
	require.Equal(t, labelPertemuan{29, "TALAQQI", 7}, got[8])
}

// The opposite mistake: the level was changed too early, so the first
// recorded "Talaqqi" meetings were really the previous level.
func TestKoreksiNomorLevelMajukanAwalLevel(t *testing.T) {
	f := setupKelasPerubahan(t)
	for range 3 {
		f.buatPertemuan(t, "selesai") // PT 1–3
	}
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))
	for range 4 {
		f.buatPertemuan(t, "selesai") // TQ 1–4, really PT 4–5 + TQ 1–2
	}
	require.NoError(t, f.service.KoreksiNomorLevel(f.kelasID, 2, f.adminID, "Level diganti terlalu cepat"))
	require.Equal(t, []labelPertemuan{
		{1, "PT", 1}, {2, "PT", 2}, {3, "PT", 3}, {4, "PT", 4}, {5, "PT", 5},
		{6, "TQ", 1}, {7, "TQ", 2},
	}, f.labelPertemuan(t))
}

// With two level changes, the current level cannot swallow the whole
// previous level, and a class that never changed level has no such tool.
func TestKoreksiNomorLevelBatas(t *testing.T) {
	f := setupKelasPerubahan(t)
	state, err := f.service.GetKoreksiLevelState(f.kelasID)
	require.NoError(t, err)
	require.Nil(t, state)
	require.ErrorContains(t, f.service.KoreksiNomorLevel(f.kelasID, 3, f.adminID, "x"), "belum pernah ganti level")

	f.buatPertemuan(t, "selesai") // PT 1
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))
	f.buatPertemuan(t, "selesai") // TQ 1
	f.buatPertemuan(t, "selesai") // TQ 2
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "03", f.adminID))
	f.buatPertemuan(t, "selesai") // TALAQQI 1

	state, err = f.service.GetKoreksiLevelState(f.kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 2, state.MaksTerakhir, "TQ must keep at least one meeting")
	require.Equal(t, "TQ", state.LevelSebelum)
	require.ErrorContains(t, f.service.KoreksiNomorLevel(f.kelasID, 3, f.adminID, "x"), "maksimal pertemuan ke-2")

	f.buatPertemuan(t, "berlangsung")
	require.ErrorContains(t, f.service.KoreksiNomorLevel(f.kelasID, 2, f.adminID, "x"), "sedang berlangsung")
}
