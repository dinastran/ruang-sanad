package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKoordinatorDashboardMenampilkanAksiJatuhTempoDanAgenda(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	svc := NewKoordinatorService(f.querier)
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, wib)

	exec := func(query string, args ...any) {
		t.Helper()
		_, err := f.db.Exec(query, args...)
		require.NoError(t, err)
	}

	// To-do: lewat deadline, jatuh tempo hari ini, belum jatuh tempo, dan sudah selesai.
	exec(`INSERT INTO todo_koordinator (judul, deadline, status) VALUES ('Lewat', '2026-10-13', 'belum')`)
	exec(`INSERT INTO todo_koordinator (judul, deadline, status) VALUES ('Hari ini', '2026-10-15', 'proses')`)
	exec(`INSERT INTO todo_koordinator (judul, deadline, status) VALUES ('Besok', '2026-10-16', 'belum')`)
	exec(`INSERT INTO todo_koordinator (judul, deadline, status) VALUES ('Beres', '2026-10-10', 'selesai')`)

	// Kunjungan terjadwal dalam 7 hari + tindak lanjut terbuka yang lewat target.
	exec(`INSERT INTO kunjungan_kelas (guru_id, kelas_id, tanggal, jam, status) VALUES (?, ?, '2026-10-17', '08:00', 'dijadwalkan')`, f.guruUtamaID, f.kelasID)
	exec(`INSERT INTO kunjungan_kelas (guru_id, tanggal, status) VALUES (?, '2026-10-30', 'dijadwalkan')`, f.guruUtamaID)
	exec(`INSERT INTO kunjungan_tindak_lanjut (kunjungan_id, jenis, target_tanggal) VALUES (1, 'coaching', '2026-10-14')`)
	exec(`INSERT INTO kunjungan_tindak_lanjut (kunjungan_id, jenis, target_tanggal, status) VALUES (1, 'apresiasi', '2026-10-14', 'selesai')`)

	exec(`INSERT INTO pembinaan (tanggal, topik) VALUES ('2026-10-15', 'Adab mengajar')`)
	exec(`INSERT INTO rapat_guru (tanggal, judul) VALUES ('2026-10-19', 'Rapat pekanan')`)
	exec(`INSERT INTO rapat_guru (tanggal, judul, status) VALUES ('2026-10-18', 'Batal', 'batal')`)

	// Satu dari tiga guru aktif sudah final TSI bulan ini.
	exec(`INSERT INTO tsi_periode (guru_id, bulan, status) VALUES (?, '2026-10', 'final')`, f.guruUtamaID)

	d, err := svc.GetDashboard(now)
	require.NoError(t, err)

	require.Equal(t, "2026-10-15", d.Tanggal)
	require.Len(t, d.Todo, 2)
	require.Equal(t, "Lewat", d.Todo[0].Judul)
	require.True(t, d.Todo[0].Terlambat)
	require.False(t, d.Todo[1].Terlambat)

	require.Len(t, d.TindakLanjut, 1)
	require.Equal(t, "coaching", d.TindakLanjut[0].Jenis)
	require.Equal(t, "Guru Utama", d.TindakLanjut[0].GuruNama)
	require.True(t, d.TindakLanjut[0].Terlambat)

	require.Len(t, d.Agenda, 3)
	require.Equal(t, []string{"pembinaan", "kunjungan", "rapat"}, []string{d.Agenda[0].Jenis, d.Agenda[1].Jenis, d.Agenda[2].Jenis})
	require.Equal(t, "Kunjungan kelas · Guru Utama (Kelas Jadwal)", d.Agenda[1].Judul)

	require.Equal(t, "2026-10", d.TsiBulan)
	require.EqualValues(t, 2, d.TsiBelumFinal)
}
