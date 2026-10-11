package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestBatalkanPertemuanMengembalikanJadwalDanMencatatRiwayat(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	today := startOfToday().Format("2006-01-02")
	jadwalID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{KelasID: f.kelasID, Tanggal: today, JamMulai: "23:59"})
	require.NoError(t, err)
	p, err := f.service.Start(jadwalID, f.kelasID, f.guruUtamaUser, false)
	require.NoError(t, err)

	require.ErrorContains(t, f.pertemuan.BatalkanPertemuan(p.ID, f.kelasID, f.guruUtamaUser, "  "), "alasan")
	require.ErrorContains(t, f.pertemuan.BatalkanPertemuan(p.ID, f.kelasID+1, f.guruUtamaUser, "salah klik"), "tidak ditemukan")
	require.NoError(t, f.pertemuan.BatalkanPertemuan(p.ID, f.kelasID, f.guruUtamaUser, "Salah klik Mulai"))

	var count int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM pertemuan WHERE id = ?`, p.ID).Scan(&count))
	require.Zero(t, count)
	var status string
	var pertemuanID *int64
	require.NoError(t, f.db.QueryRow(`SELECT status, pertemuan_id FROM jadwal_pertemuan WHERE id = ?`, jadwalID).Scan(&status, &pertemuanID))
	require.Equal(t, "dijadwalkan", status)
	require.Nil(t, pertemuanID)

	var jenis, alasan string
	require.NoError(t, f.db.QueryRow(`SELECT jenis, alasan FROM kelas_perubahan WHERE kelas_id = ?`, f.kelasID).Scan(&jenis, &alasan))
	require.Equal(t, PerubahanBatalPertemuan, jenis)
	require.Equal(t, "Salah klik Mulai", alasan)
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM schedule_activity_logs WHERE jadwal_pertemuan_id = ? AND action = 'pertemuan_dibatalkan'`, jadwalID).Scan(&count))
	require.Equal(t, 1, count)

	// The same schedule can be started again and gets the freed number.
	again, err := f.service.Start(jadwalID, f.kelasID, f.guruUtamaUser, false)
	require.NoError(t, err)
	require.Equal(t, p.PertemuanKe, again.PertemuanKe)
}

func TestBatalkanPertemuanSelesaiDitolak(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	p, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{KonfirmasiTambahan: true})
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE pertemuan SET status = 'selesai' WHERE id = ?`, p.ID)
	require.NoError(t, err)
	require.ErrorContains(t, f.pertemuan.BatalkanPertemuan(p.ID, f.kelasID, f.guruUtamaUser, "salah"), "sedang berlangsung")
}

func TestPertemuanLupaDitutupDitandaiDanDiingatkanSekali(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	p, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{KonfirmasiTambahan: true})
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE pertemuan SET guru_pengganti_id = ? WHERE id = ?`, f.guruBadalID, p.ID)
	require.NoError(t, err)

	var created time.Time
	require.NoError(t, f.db.QueryRow(`SELECT created_at FROM pertemuan WHERE id = ?`, p.ID).Scan(&created))
	baru := created.Add(30 * time.Minute)
	lama := created.Add(BatasLupaDitutup + time.Minute)

	list, err := f.pertemuan.ListBelumDitutup(nil, baru)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.False(t, list[0].Lupa)
	n, err := f.pertemuan.KirimPengingatLupaDitutup(baru)
	require.NoError(t, err)
	require.Zero(t, n)

	list, err = f.pertemuan.ListBelumDitutup(&f.guruUtamaID, lama)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].Lupa)
	list, err = f.pertemuan.ListBelumDitutup(&f.guruBadalID, lama)
	require.NoError(t, err)
	require.Len(t, list, 1, "guru badal juga melihat pertemuan yang ia gantikan")
	lainID := f.guruBadalID + 1
	list, err = f.pertemuan.ListBelumDitutup(&lainID, lama)
	require.NoError(t, err)
	require.Empty(t, list)

	n, err = f.pertemuan.KirimPengingatLupaDitutup(lama)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = f.pertemuan.KirimPengingatLupaDitutup(lama.Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, n, "pengingat hanya sekali")

	var notif int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type = 'pertemuan_lupa_ditutup' AND user_id IN (?, ?)`, f.guruUtamaUser, f.guruBadalUser).Scan(&notif))
	require.Equal(t, 2, notif)
}
