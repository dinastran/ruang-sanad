package services

import (
	"database/sql"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// insertTagihanMassal creates a tagihan with time.Time params, the same way the
// generator stores tanggal_tagih, so date filters see production-shaped data.
func insertTagihanMassal(t *testing.T, db *sql.DB, kelasID, santriID, bulanKe int64, tanggal time.Time, status, catatan string) int64 {
	t.Helper()
	_, err := db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, jatuh_tempo, status, catatan, angkatan_kelas) VALUES (?, ?, ?, ?, 100000, ?, ?, ?, ?, '2026')`,
		santriID, kelasID, bulanKe, bulanKe*4, tanggal, tanggal.AddDate(0, 0, 7), status, catatan)
	require.NoError(t, err)
	id, err := lastID(db)
	require.NoError(t, err)
	return id
}

func tagihanStatusCatatan(t *testing.T, db *sql.DB, id int64) (string, string) {
	t.Helper()
	var status, catatan string
	require.NoError(t, db.QueryRow(`SELECT status, catatan FROM tagihan WHERE id = ?`, id).Scan(&status, &catatan))
	return status, catatan
}

func d(day int) time.Time { return time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC) }

func TestBatalMassalWajibRentangTanggal(t *testing.T) {
	_, service, _ := setupTagihanService(t, "1x/pekan", 0)
	_, err := service.PreviewBatalMassal(models.TagihanFilter{TanggalDari: "2026-01-01"})
	require.ErrorIs(t, err, ErrBatalMassalTanggalWajib)
	_, err = service.BatalMassal(models.TagihanFilter{}, "x", 0, 1)
	require.ErrorIs(t, err, ErrBatalMassalTanggalWajib)
	_, err = service.PreviewBatalMassal(models.TagihanFilter{TanggalDari: "2026-02-01", TanggalSampai: "2026-01-01"})
	require.ErrorIs(t, err, ErrBatalMassalTanggalTidak)
}

func TestBatalMassalHanyaBelumBayarDalamRentangDanBisaDipulihkan(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	awal := insertTagihanMassal(t, db, kelasID, santriID, 1, d(1), "belum_bayar", "catatan lama")
	akhir := insertTagihanMassal(t, db, kelasID, santriID, 2, d(31), "belum_bayar", "")
	lunas := insertTagihanMassal(t, db, kelasID, santriID, 3, d(10), "lunas", "")
	batal := insertTagihanMassal(t, db, kelasID, santriID, 4, d(11), "batal", "sudah batal")
	luar := insertTagihanMassal(t, db, kelasID, santriID, 5, time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC), "belum_bayar", "")

	filter := models.TagihanFilter{TanggalDari: "2026-01-01", TanggalSampai: "2026-01-31"}
	preview, err := service.PreviewBatalMassal(filter)
	require.NoError(t, err)
	require.Equal(t, models.BatalMassalPreview{Jumlah: 2, TotalNominal: 200000, DilewatiLunas: 1, DilewatiBatal: 1}, *preview)

	_, err = service.BatalMassal(filter, "  ", 0, 2)
	require.ErrorIs(t, err, ErrBatalMassalAlasanWajib)
	_, err = service.BatalMassal(filter, "Kedaluwarsa", 0, 3)
	require.ErrorIs(t, err, ErrBatalMassalBerubah)

	result, err := service.BatalMassal(filter, "Kedaluwarsa", 0, 2)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Jumlah)
	require.EqualValues(t, 200000, result.TotalNominal)
	for _, id := range []int64{awal, akhir} {
		status, catatan := tagihanStatusCatatan(t, db, id)
		require.Equal(t, "batal", status)
		require.Equal(t, "Kedaluwarsa", catatan)
	}
	status, _ := tagihanStatusCatatan(t, db, lunas)
	require.Equal(t, "lunas", status)
	_, catatan := tagihanStatusCatatan(t, db, batal)
	require.Equal(t, "sudah batal", catatan, "tagihan yang sudah batal tidak boleh ditimpa")
	status, _ = tagihanStatusCatatan(t, db, luar)
	require.Equal(t, "belum_bayar", status)

	history, err := service.ListBatalMassal(20)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Contains(t, history[0].FilterRingkas, "2026-01-01 s/d 2026-01-31")

	restored, err := service.PulihkanBatalMassal(result.BatchID, 0)
	require.NoError(t, err)
	require.EqualValues(t, 2, restored)
	status, catatan = tagihanStatusCatatan(t, db, awal)
	require.Equal(t, "belum_bayar", status)
	require.Equal(t, "catatan lama", catatan)
	_, catatan = tagihanStatusCatatan(t, db, batal)
	require.Equal(t, "sudah batal", catatan, "undo hanya menyentuh tagihan milik batch")

	_, err = service.PulihkanBatalMassal(result.BatchID, 0)
	require.ErrorIs(t, err, ErrBatalBatchSudahPulih)
	_, err = service.PulihkanBatalMassal(999, 0)
	require.ErrorIs(t, err, ErrBatalBatchTidakDitemukan)
}

func TestBatalMassalMengikutiFilterLain(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	_, err := db.Exec(`INSERT INTO santri (nama, jenis_kelamin, nominal, frekuensi, kelas_id, status, angkatan_kelas) VALUES ('Budi', 'L', 100000, '1x/pekan', ?, 'aktif', '2026')`, kelasID)
	require.NoError(t, err)
	budiID, err := lastID(db)
	require.NoError(t, err)
	ahmad := insertTagihanMassal(t, db, kelasID, santriID, 1, d(5), "belum_bayar", "")
	budi := insertTagihanMassal(t, db, kelasID, budiID, 1, d(5), "belum_bayar", "")

	filter := models.TagihanFilter{TanggalDari: "2026-01-01", TanggalSampai: "2026-01-31", Search: "Budi"}
	_, err = service.BatalMassal(filter, "Kedaluwarsa", 0, 1)
	require.NoError(t, err)
	status, _ := tagihanStatusCatatan(t, db, budi)
	require.Equal(t, "batal", status)
	status, _ = tagihanStatusCatatan(t, db, ahmad)
	require.Equal(t, "belum_bayar", status)
}
