package services

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type kunjunganFixture struct {
	db       *sql.DB
	service  *KunjunganService
	guruID   int64
	guruUser int64
	adminID  int64
}

func setupKunjungan(t *testing.T) kunjunganFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	guruUser := insertTestUser(t, db, "guru@example.com", "Ustadz Ahmad")
	guruID := insertTestGuru(t, db, "Ustadz Ahmad", guruUser)
	_, err = db.Exec(`UPDATE guru SET no_wa = '0812-3456-7890' WHERE id = ?`, guruID)
	require.NoError(t, err)
	adminID := insertTestUser(t, db, "koor@example.com", "Koordinator")
	return kunjunganFixture{db: db, service: NewKunjunganService(queries.NewQuerier(db)), guruID: guruID, guruUser: guruUser, adminID: adminID}
}

func (f kunjunganFixture) terlaksana(nilai ...int64) models.KunjunganRequest {
	req := models.KunjunganRequest{GuruID: f.guruID, Tanggal: "2026-10-05", Jam: "05:00", Status: "terlaksana", Catatan: "Bacaan jelas, perlu perbaikan waktu mulai."}
	if len(nilai) == 4 {
		req.NilaiKedisiplinan, req.NilaiMateri, req.NilaiMetode, req.NilaiInteraksi = nilai[0], nilai[1], nilai[2], nilai[3]
	}
	return req
}

func (f kunjunganFixture) countNotif(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = 'kunjungan_kelas'`, f.guruUser).Scan(&n))
	return n
}

func TestKunjunganKirimMenguncidanMemberiNotifikasi(t *testing.T) {
	f := setupKunjungan(t)
	today := time.Date(2026, 10, 8, 0, 0, 0, 0, time.Local)

	id, err := f.service.Create(f.terlaksana(4, 3, 3, 2))
	require.NoError(t, err)

	k, err := f.service.Detail(id, today)
	require.NoError(t, err)
	require.True(t, k.NilaiLengkap)
	require.Equal(t, 3.0, k.NilaiRataRata)
	require.Equal(t, "Baik", k.Predikat)
	require.Equal(t, "draft", k.StatusKirim)
	require.Empty(t, k.WALink, "WA hanya untuk hasil yang sudah dikirim")
	require.Equal(t, "Sangat baik", k.Aspek[0].Predikat)

	require.NoError(t, f.service.Kirim(id, f.adminID))
	require.Equal(t, 1, f.countNotif(t))
	require.ErrorContains(t, f.service.Kirim(id, f.adminID), "sudah dikirim")

	err = f.service.Update(id, f.terlaksana(1, 1, 1, 1))
	require.ErrorContains(t, err, "buka kunci")
	require.ErrorContains(t, f.service.Delete(id), "buka kunci")

	k, err = f.service.Detail(id, today)
	require.NoError(t, err)
	require.Equal(t, "terkirim", k.StatusKirim)
	require.Contains(t, k.WALink, "https://wa.me/6281234567890?text=")
	text, err := url.QueryUnescape(k.WALink[len("https://wa.me/6281234567890?text="):])
	require.NoError(t, err)
	require.Contains(t, text, "Kedisiplinan & adab: 4 (Sangat baik)")
	require.Contains(t, text, "Rata-rata: 3,00 (Baik)")

	require.NoError(t, f.service.BukaKunci(id))
	require.NoError(t, f.service.Update(id, f.terlaksana(4, 4, 3, 3)))
	require.NoError(t, f.service.Kirim(id, f.adminID))
	require.Equal(t, 2, f.countNotif(t))
}

func TestKunjunganKirimMenolakHasilBelumLengkap(t *testing.T) {
	f := setupKunjungan(t)
	id, err := f.service.Create(f.terlaksana(4, 3, 0, 2))
	require.NoError(t, err)
	require.ErrorContains(t, f.service.Kirim(id, f.adminID), "lengkapi nilai")

	jadwal, err := f.service.Create(models.KunjunganRequest{GuruID: f.guruID, Status: "dijadwalkan", NilaiMateri: 3})
	require.NoError(t, err)
	require.ErrorContains(t, f.service.Kirim(jadwal, f.adminID), "Terlaksana")
	k, err := f.service.Detail(jadwal, time.Now())
	require.NoError(t, err)
	require.Zero(t, k.NilaiMateri, "nilai dikosongkan bila kunjungan belum terlaksana")

	_, err = f.service.Create(f.terlaksana(5, 3, 3, 3))
	require.ErrorContains(t, err, "1 sampai 4")
	require.Equal(t, 0, f.countNotif(t))
}

func TestKunjunganGuruMembacaDanMenanggapi(t *testing.T) {
	f := setupKunjungan(t)
	today := time.Now()
	id, err := f.service.Create(f.terlaksana(3, 3, 3, 3))
	require.NoError(t, err)

	_, err = f.service.DetailUntukGuru(id, f.guruID, today)
	require.ErrorContains(t, err, "tidak ditemukan", "draft tidak terlihat oleh guru")
	list, err := f.service.ListUntukGuru(f.guruID, today)
	require.NoError(t, err)
	require.Empty(t, list)

	require.NoError(t, f.service.TambahTindakLanjut(id, models.TindakLanjutRequest{Jenis: TindakLanjutKoordinasi, Catatan: "Koordinasi dengan admin kelas"}, f.adminID))
	require.NoError(t, f.service.TambahTindakLanjut(id, models.TindakLanjutRequest{Jenis: TindakLanjutCoaching, Catatan: "Coaching manajemen waktu", TargetTanggal: "2026-10-20"}, f.adminID))
	require.NoError(t, f.service.Kirim(id, f.adminID))

	otherUser := insertTestUser(t, f.db, "lain@example.com", "Guru Lain")
	otherGuru := insertTestGuru(t, f.db, "Guru Lain", otherUser)
	_, err = f.service.DetailUntukGuru(id, otherGuru, today)
	require.Error(t, err)

	k, err := f.service.DetailUntukGuru(id, f.guruID, today)
	require.NoError(t, err)
	require.Equal(t, "dibaca", k.StatusKirim)
	require.Len(t, k.TindakLanjut, 1, "koordinasi bersifat internal")
	require.Equal(t, TindakLanjutCoaching, k.TindakLanjut[0].Jenis)

	require.ErrorContains(t, f.service.SimpanTanggapan(id, f.guruID, "   "), "kosong")
	require.ErrorContains(t, f.service.SimpanTanggapan(id, otherGuru, "Siap"), "tidak ditemukan")
	require.NoError(t, f.service.SimpanTanggapan(id, f.guruID, "Siap, akan datang 10 menit lebih awal."))

	detail, err := f.service.Detail(id, today)
	require.NoError(t, err)
	require.Equal(t, "ditanggapi", detail.StatusKirim)
	require.Len(t, detail.TindakLanjut, 2, "koordinator melihat semua tindak lanjut")
	text, err := url.QueryUnescape(detail.WALink)
	require.NoError(t, err)
	require.NotContains(t, text, "Koordinasi dengan admin kelas")
	require.Contains(t, text, "Coaching manajemen waktu")
}

func TestKunjunganTindakLanjutMonitoringMenjadwalkanKunjunganBerikutnya(t *testing.T) {
	f := setupKunjungan(t)
	ctx := context.Background()
	today := time.Date(2026, 10, 8, 0, 0, 0, 0, time.Local)
	id, err := f.service.Create(f.terlaksana(2, 2, 2, 2))
	require.NoError(t, err)
	require.NoError(t, f.service.Kirim(id, f.adminID))

	require.ErrorContains(t, f.service.TambahTindakLanjut(id, models.TindakLanjutRequest{Jenis: TindakLanjutMonitoring}, f.adminID), "target tanggal")
	require.ErrorContains(t, f.service.TambahTindakLanjut(id, models.TindakLanjutRequest{Jenis: "hukuman"}, f.adminID), "tidak valid")
	before := f.countNotif(t)
	require.NoError(t, f.service.TambahTindakLanjut(id, models.TindakLanjutRequest{Jenis: TindakLanjutMonitoring, Catatan: "Cek manajemen waktu", TargetTanggal: "2026-10-01"}, f.adminID))
	require.Equal(t, before+1, f.countNotif(t), "tindak lanjut yang terlihat guru dikabarkan setelah hasil dikirim")

	k, err := f.service.Detail(id, today)
	require.NoError(t, err)
	require.Len(t, k.TindakLanjut, 1)
	tl := k.TindakLanjut[0]
	require.True(t, tl.Terlambat)
	require.NotNil(t, tl.KunjunganBerikutnyaID)

	next, err := f.service.Detail(*tl.KunjunganBerikutnyaID, today)
	require.NoError(t, err)
	require.Equal(t, "dijadwalkan", next.Status)
	require.Equal(t, f.guruID, next.GuruID)
	require.Equal(t, "2026-10-01", next.TargetMulai)
	require.Contains(t, next.Catatan, "Cek manajemen waktu")

	dash, err := f.service.Dashboard(today)
	require.NoError(t, err)
	require.Len(t, dash.TindakLanjutTerbuka, 1)
	require.Equal(t, "Ustadz Ahmad", dash.TindakLanjutTerbuka[0].GuruNama)

	// Sending the scheduled monitoring visit closes the follow-up.
	require.NoError(t, f.service.Update(next.ID, f.terlaksana(3, 3, 3, 4)))
	require.NoError(t, f.service.Kirim(next.ID, f.adminID))
	tlRow, err := f.service.querier.GetTindakLanjut(ctx, tl.ID)
	require.NoError(t, err)
	require.Equal(t, "selesai", tlRow.Status)

	dash, err = f.service.Dashboard(today)
	require.NoError(t, err)
	require.Empty(t, dash.TindakLanjutTerbuka)
	require.Len(t, dash.Rekap, 1)
	r := dash.Rekap[0]
	require.Equal(t, 2, r.JumlahKunjungan)
	require.Equal(t, 3.25, r.NilaiTerakhir)
	require.Equal(t, 2.0, r.NilaiSebelumnya)
	require.Equal(t, "naik", r.Tren)
	require.Equal(t, 3.0, r.RataRataAspek[3].RataRata)
	require.Len(t, dash.BelumDitanggapi, 2)
}

func TestKunjunganHapusMonitoringMenghapusJadwalYangBelumDipakai(t *testing.T) {
	f := setupKunjungan(t)
	today := time.Now()
	id, err := f.service.Create(f.terlaksana(3, 3, 3, 3))
	require.NoError(t, err)
	require.NoError(t, f.service.TambahTindakLanjut(id, models.TindakLanjutRequest{Jenis: TindakLanjutMonitoring, TargetTanggal: "2026-11-01"}, f.adminID))
	k, err := f.service.Detail(id, today)
	require.NoError(t, err)
	nextID := *k.TindakLanjut[0].KunjunganBerikutnyaID

	require.NoError(t, f.service.SetStatusTindakLanjut(id, k.TindakLanjut[0].ID, "selesai"))
	require.ErrorContains(t, f.service.SetStatusTindakLanjut(nextID, k.TindakLanjut[0].ID, "selesai"), "tidak ditemukan")
	require.NoError(t, f.service.HapusTindakLanjut(id, k.TindakLanjut[0].ID))
	_, err = f.service.Detail(nextID, today)
	require.ErrorContains(t, err, "tidak ditemukan")

	jadwal, err := f.service.Create(models.KunjunganRequest{GuruID: f.guruID})
	require.NoError(t, err)
	require.ErrorContains(t, f.service.TambahTindakLanjut(jadwal, models.TindakLanjutRequest{Jenis: TindakLanjutApresiasi}, f.adminID), "terlaksana")
}
