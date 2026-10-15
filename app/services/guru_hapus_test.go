package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHapusGuruHanyaUntukGuruTanpaRiwayat(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	_, err := f.db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	svc := NewGuruService(f.querier)

	var guruLainID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM guru WHERE user_id = ?`, f.guruLainUser).Scan(&guruLainID))

	// Guru tanpa kelas tapi punya catatan riayah dari koordinator: riwayat, tidak boleh dihapus.
	_, err = f.db.Exec(`INSERT INTO guru (nama, is_aktif) VALUES ('Guru Dobel', 1)`)
	require.NoError(t, err)
	guruDobelID, err := lastID(f.db)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO catatan_riayah (author_user_id, target_type, target_id, catatan) VALUES (?, 'guru', ?, 'Pembinaan')`, f.guruUtamaUser, guruDobelID)
	require.NoError(t, err)
	info, err := svc.InfoHapus(guruDobelID)
	require.NoError(t, err)
	require.False(t, info.BisaDihapus)
	require.ErrorIs(t, svc.Hapus(guruDobelID), ErrGuruPunyaRiwayat)

	// Guru tanpa kelas & riwayat: boleh dihapus permanen.
	info, err = svc.InfoHapus(guruLainID)
	require.NoError(t, err)
	require.True(t, info.BisaDihapus)
	require.Empty(t, info.Riwayat)
	require.NoError(t, svc.Hapus(guruLainID))
	var n int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM guru WHERE id = ?`, guruLainID).Scan(&n))
	require.Zero(t, n)
	_, err = svc.InfoHapus(guruLainID)
	require.ErrorIs(t, err, ErrGuruTidakDitemukan)

	// Guru utama kelas aktif: tidak bisa dihapus maupun dinonaktifkan.
	info, err = svc.InfoHapus(f.guruUtamaID)
	require.NoError(t, err)
	require.False(t, info.BisaDihapus)
	require.Equal(t, []string{"Kelas Jadwal"}, info.KelasAktif)
	require.ErrorIs(t, svc.Hapus(f.guruUtamaID), ErrGuruMasihMengajar)
	require.ErrorIs(t, svc.Nonaktifkan(f.guruUtamaID), ErrGuruMasihMengajar)

	// Guru dengan riwayat (TSI): ditolak dihapus, tapi bisa dinonaktifkan dan riwayatnya tetap ada.
	_, err = f.db.Exec(`INSERT INTO tsi_periode (guru_id, bulan, status) VALUES (?, '2026-10', 'final')`, f.guruBadalID)
	require.NoError(t, err)
	info, err = svc.InfoHapus(f.guruBadalID)
	require.NoError(t, err)
	require.False(t, info.BisaDihapus)
	require.Empty(t, info.KelasAktif)
	require.Equal(t, "Penilaian TSI", info.Riwayat[0].Label)
	require.EqualValues(t, 1, info.Riwayat[0].Jumlah)
	require.ErrorIs(t, svc.Hapus(f.guruBadalID), ErrGuruPunyaRiwayat)

	require.NoError(t, svc.Nonaktifkan(f.guruBadalID))
	var aktif int64
	require.NoError(t, f.db.QueryRow(`SELECT is_aktif FROM guru WHERE id = ?`, f.guruBadalID).Scan(&aktif))
	require.Zero(t, aktif)
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM tsi_periode WHERE guru_id = ?`, f.guruBadalID).Scan(&n))
	require.EqualValues(t, 1, n)
}
