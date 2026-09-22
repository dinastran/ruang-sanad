package services

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type statusFixture struct {
	db       *sql.DB
	service  *SantriStatusService
	santri   *SantriService
	kelasID  int64
	santriID int64
	adminID  int64
}

// 2026-09-22 10:00 WIB
var statusTestNow = time.Date(2026, 9, 22, 3, 0, 0, 0, time.UTC)

func setupSantriStatus(t *testing.T, kapasitas int) statusFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	_, err = db.Exec(`INSERT INTO users (email, name, role) VALUES ('ak@example.com', 'Admin Kelas', 'admin_kelas')`)
	require.NoError(t, err)
	adminID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri, created_at) VALUES ('S', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Senin', 1, 'Kelas Status', ?, 0, CURRENT_TIMESTAMP)`, kapasitas)
	require.NoError(t, err)
	kelasID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES ('Ahmad', ?, 'aktif')`, kelasID)
	require.NoError(t, err)
	santriID, err := lastID(db)
	require.NoError(t, err)

	querier := queries.NewQuerier(db)
	service := NewSantriStatusService(querier)
	service.now = func() time.Time { return statusTestNow }
	return statusFixture{db: db, service: service, santri: NewSantriService(querier, nil), kelasID: kelasID, santriID: santriID, adminID: adminID}
}

func (f statusFixture) santriRow(t *testing.T) queries.Santri {
	t.Helper()
	s, err := queries.NewQuerier(f.db).GetSantriByID(t.Context(), f.santriID)
	require.NoError(t, err)
	return s
}

func (f statusFixture) jumlahSantri(t *testing.T) int64 {
	t.Helper()
	k, err := queries.NewQuerier(f.db).GetKelasByID(t.Context(), f.kelasID)
	require.NoError(t, err)
	return k.JumlahSantri
}

func (f statusFixture) insertSantri(t *testing.T, nama, status string) {
	t.Helper()
	_, err := f.db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES (?, ?, ?)`, nama, f.kelasID, status)
	require.NoError(t, err)
}

func cutiReq(mulai, selesai string) models.UpdateSantriStatusRequest {
	return models.UpdateSantriStatusRequest{Status: "cuti", Alasan: "Sakit", CutiMulai: mulai, CutiSelesai: selesai}
}

func TestUbahStatusCutiMemegangKursiDanTercatat(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-09-20", "2026-10-05")))

	s := f.santriRow(t)
	require.Equal(t, "cuti", s.Status)
	require.Equal(t, "Sakit", s.StatusAlasan)
	require.Equal(t, "2026-09-20", s.CutiMulai)
	require.Equal(t, "2026-10-05", s.CutiSelesai)
	require.Equal(t, int64(1), f.jumlahSantri(t), "cuti tetap memegang kursi")

	aktif, err := queries.NewQuerier(f.db).GetSantriByKelasID(t.Context(), sql.NullInt64{Int64: f.kelasID, Valid: true})
	require.NoError(t, err)
	require.Empty(t, aktif, "santri cuti tidak ikut absensi/tagihan")

	riwayat, err := f.service.ListRiwayatKelas(f.kelasID)
	require.NoError(t, err)
	require.Len(t, riwayat, 1)
	require.Equal(t, "aktif", riwayat[0].StatusLama)
	require.Equal(t, "cuti", riwayat[0].StatusBaru)
	require.Equal(t, "Admin Kelas", riwayat[0].DibuatOlehNama)
}

func TestUbahStatusNonaktifMelepasKursi(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "nonaktif", Alasan: "Pindah kota", CutiMulai: "2026-09-01", CutiSelesai: "2026-09-30"}))
	s := f.santriRow(t)
	require.Equal(t, "nonaktif", s.Status)
	require.Empty(t, s.CutiMulai, "tanggal cuti diabaikan untuk nonaktif")
	require.Equal(t, int64(0), f.jumlahSantri(t))

	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "aktif"}))
	s = f.santriRow(t)
	require.Equal(t, "aktif", s.Status)
	require.Empty(t, s.StatusAlasan)
}

func TestUbahStatusValidasi(t *testing.T) {
	f := setupSantriStatus(t, 20)
	cases := []struct {
		name string
		req  models.UpdateSantriStatusRequest
	}{
		{"status tidak dikenal", models.UpdateSantriStatusRequest{Status: "tidak_lanjut", Alasan: "x"}},
		{"nonaktif tanpa alasan", models.UpdateSantriStatusRequest{Status: "nonaktif"}},
		{"cuti tanpa alasan", models.UpdateSantriStatusRequest{Status: "cuti", CutiMulai: "2026-09-22", CutiSelesai: "2026-09-30"}},
		{"cuti tanpa tanggal", models.UpdateSantriStatusRequest{Status: "cuti", Alasan: "x"}},
		{"selesai sebelum mulai", cutiReq("2026-09-30", "2026-09-25")},
		{"selesai sudah lewat", cutiReq("2026-09-01", "2026-09-21")},
		{"status sama", models.UpdateSantriStatusRequest{Status: "aktif"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, tc.req))
		})
	}
	require.Equal(t, "aktif", f.santriRow(t).Status)
}

func TestUbahStatusDitolakUntukTidakLanjutDanKelasLain(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.ErrorIs(t, f.service.UbahStatus(f.kelasID+99, f.santriID, f.adminID, cutiReq("2026-09-22", "2026-09-30")), ErrStatusSantriBukanDiKelas)

	_, err := f.db.Exec(`UPDATE santri SET status = 'tidak_lanjut' WHERE id = ?`, f.santriID)
	require.NoError(t, err)
	require.ErrorIs(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "aktif"}), ErrStatusSantriTerkunci)
}

func TestAktifkanDariNonaktifDitolakSaatKelasPenuh(t *testing.T) {
	f := setupSantriStatus(t, 2)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "nonaktif", Alasan: "x"}))
	f.insertSantri(t, "Budi", "aktif")
	f.insertSantri(t, "Umar", "cuti")
	err := f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "aktif"})
	require.ErrorContains(t, err, "penuh")
	require.Equal(t, "nonaktif", f.santriRow(t).Status)
}

func TestAktifkanCutiBerakhir(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-09-10", "2026-09-22")))

	total, err := f.service.AktifkanCutiBerakhir()
	require.NoError(t, err)
	require.Zero(t, total, "hari terakhir cuti belum lewat")

	// Keesokan harinya pukul 00:30 WIB (masih 22 Sep UTC).
	f.service.now = func() time.Time { return time.Date(2026, 9, 22, 17, 30, 0, 0, time.UTC) }
	total, err = f.service.AktifkanCutiBerakhir()
	require.NoError(t, err)
	require.Equal(t, 1, total)
	s := f.santriRow(t)
	require.Equal(t, "aktif", s.Status)
	require.Empty(t, s.CutiSelesai)

	total, err = f.service.AktifkanCutiBerakhir()
	require.NoError(t, err)
	require.Zero(t, total, "idempoten")

	riwayat, err := f.service.ListRiwayatKelas(f.kelasID)
	require.NoError(t, err)
	require.Len(t, riwayat, 2)
	require.Equal(t, alasanCutiBerakhir, riwayat[0].Alasan)
	require.Empty(t, riwayat[0].DibuatOlehNama, "dibuat oleh sistem")
}

func TestKeuanganTidakMenimpaStatusAdminKelas(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-09-22", "2026-10-22")))

	require.NoError(t, f.santri.UpdateByKeuangan(f.santriID, f.adminID, models.UpdateSantriKeuanganRequest{InfaqTerakhir: "Sep"}))
	require.Equal(t, "cuti", f.santriRow(t).Status)

	require.NoError(t, f.santri.UpdateByKeuangan(f.santriID, f.adminID, models.UpdateSantriKeuanganRequest{KeteranganTidakLanjut: "Berhenti"}))
	s := f.santriRow(t)
	require.Equal(t, "tidak_lanjut", s.Status)
	require.Empty(t, s.CutiSelesai)

	require.NoError(t, f.santri.UpdateByKeuangan(f.santriID, f.adminID, models.UpdateSantriKeuanganRequest{}))
	require.Equal(t, "aktif", f.santriRow(t).Status)

	riwayat, err := f.service.ListRiwayatKelas(f.kelasID)
	require.NoError(t, err)
	require.Len(t, riwayat, 3)
}

func TestCutiMasaDepanTerjadwalLaluDimulaiOtomatis(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-10-01", "2026-10-15")))

	s := f.santriRow(t)
	require.Equal(t, "aktif", s.Status, "cuti belum dimulai, santri tetap ikut absensi & tagihan")
	require.Equal(t, "2026-10-01", s.CutiMulai)
	require.Equal(t, "2026-10-15", s.CutiSelesai)
	riwayat, err := f.service.ListRiwayatKelas(f.kelasID)
	require.NoError(t, err)
	require.Equal(t, statusCutiTerjadwal, riwayat[0].StatusBaru)

	total, err := f.service.MulaiCutiTerjadwal()
	require.NoError(t, err)
	require.Zero(t, total)

	// 1 Okt 00:30 WIB.
	f.service.now = func() time.Time { return time.Date(2026, 9, 30, 17, 30, 0, 0, time.UTC) }
	total, err = f.service.MulaiCutiTerjadwal()
	require.NoError(t, err)
	require.Equal(t, 1, total)
	s = f.santriRow(t)
	require.Equal(t, "cuti", s.Status)
	require.Equal(t, "Sakit", s.StatusAlasan)
	require.Equal(t, "2026-10-15", s.CutiSelesai)
}

func TestCutiTerjadwalBisaDibatalkan(t *testing.T) {
	f := setupSantriStatus(t, 20)
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-10-01", "2026-10-15")))
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "aktif"}))
	s := f.santriRow(t)
	require.Equal(t, "aktif", s.Status)
	require.Empty(t, s.CutiMulai)

	err := f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, models.UpdateSantriStatusRequest{Status: "aktif"})
	require.ErrorContains(t, err, "sudah aktif")
}

func TestUbahRosterDitolakSaatPertemuanBerlangsung(t *testing.T) {
	f := setupSantriStatus(t, 20)
	_, err := f.db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status) VALUES (?, 1, '2026-09-22', 'berlangsung')`, f.kelasID)
	require.NoError(t, err)

	err = f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-09-22", "2026-10-05"))
	require.ErrorIs(t, err, ErrRosterPertemuanBerlangsung)
	require.Equal(t, "aktif", f.santriRow(t).Status)

	// Cuti terjadwal tidak mengubah roster saat ini, jadi tetap boleh.
	require.NoError(t, f.service.UbahStatus(f.kelasID, f.santriID, f.adminID, cutiReq("2026-10-01", "2026-10-05")))

	err = f.santri.UpdateByKeuangan(f.santriID, f.adminID, models.UpdateSantriKeuanganRequest{KeteranganTidakLanjut: "Berhenti"})
	require.ErrorIs(t, err, ErrRosterPertemuanBerlangsung)
}

func TestKeuanganAktifkanTidakLanjutDitolakSaatKelasPenuh(t *testing.T) {
	f := setupSantriStatus(t, 1)
	require.NoError(t, f.santri.UpdateByKeuangan(f.santriID, f.adminID, models.UpdateSantriKeuanganRequest{KeteranganTidakLanjut: "Berhenti"}))
	f.insertSantri(t, "Budi", "aktif")

	err := f.santri.UpdateByKeuangan(f.santriID, f.adminID, models.UpdateSantriKeuanganRequest{})
	require.ErrorContains(t, err, "penuh")
	require.Equal(t, "tidak_lanjut", f.santriRow(t).Status)
}

func TestPindahkanKelasDitolakSaatPertemuanBerlangsung(t *testing.T) {
	f := setupSantriStatus(t, 20)
	_, err := f.db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri, is_aktif, created_at) VALUES ('S2', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Selasa', 1, 'Kelas Tujuan', 20, 0, 1, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	tujuanID, err := lastID(f.db)
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE santri SET jenis_kelamin = 'L' WHERE id = ?`, f.santriID)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status) VALUES (?, 1, '2026-09-22', 'berlangsung')`, tujuanID)
	require.NoError(t, err)

	err = f.santri.PindahkanKelas(f.santriID, tujuanID)
	require.ErrorIs(t, err, ErrRosterPertemuanBerlangsung)
	require.Equal(t, f.kelasID, f.santriRow(t).KelasID.Int64)
}
