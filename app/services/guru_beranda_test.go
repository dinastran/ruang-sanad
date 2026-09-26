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

func TestParseJadwalKelas(t *testing.T) {
	hari, jam := parseJadwalKelas("Senin, jam 20.30 WIB")
	require.Equal(t, map[time.Weekday]bool{time.Monday: true}, hari)
	require.Equal(t, "20:30", jam)

	hari, jam = parseJadwalKelas("Senin & Kamis, jam 7.00 WIB")
	require.Equal(t, map[time.Weekday]bool{time.Monday: true, time.Thursday: true}, hari)
	require.Equal(t, "07:00", jam)

	hari, _ = parseJadwalKelas("Jum'at pagi")
	require.True(t, hari[time.Friday])

	hari, jam = parseJadwalKelas("Mingguan")
	require.Empty(t, hari)
	require.Empty(t, jam)
}

type berandaFixture struct {
	db      *sql.DB
	service *GuruBerandaService
	guruA   int64
	guruB   int64
	today   time.Time
	ke      map[int64]int
}

func setupBeranda(t *testing.T) *berandaFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	q := queries.NewQuerier(db)
	f := &berandaFixture{db: db, service: NewGuruBerandaService(q, NewTSIService(q)), ke: map[int64]int{}}
	// Senin, 28 September 2026 pukul 08.00 WIB.
	f.today = time.Date(2026, 9, 28, 8, 0, 0, 0, ZonaWaktuRiayah)
	f.guruA = insertTestGuru(t, db, "Guru A", insertTestUser(t, db, "a@example.com", "Guru A"))
	f.guruB = insertTestGuru(t, db, "Guru B", insertTestUser(t, db, "b@example.com", "Guru B"))
	return f
}

func (f *berandaFixture) kelas(t *testing.T, nama, jadwal string, guruID int64, dibuat string, santri int) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri, created_at) VALUES (?, '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', ?, 1, ?, ?, 20, 0, ?)`, nama, jadwal, nama, guruID, dibuat+" 00:00:00")
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	for i := 0; i < santri; i++ {
		_, err := f.db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES (?, ?, 'aktif')`, nama+" santri", id)
		require.NoError(t, err)
	}
	return id
}

func (f *berandaFixture) pertemuan(t *testing.T, kelasID int64, tanggal, status string) {
	t.Helper()
	f.ke[kelasID]++
	_, err := f.db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, pertemuan_level_ke, tanggal, status) VALUES (?, ?, ?, ?, ?)`, kelasID, f.ke[kelasID], f.ke[kelasID], tanggal, status)
	require.NoError(t, err)
}

func (f *berandaFixture) jadwal(t *testing.T, kelasID int64, tanggal, jam, status, semula string, pengganti int64) {
	t.Helper()
	var p interface{}
	if pengganti != 0 {
		p = pengganti
	}
	reschedule := 0
	if semula != "" {
		reschedule = 1
	}
	_, err := f.db.Exec(`INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai, status, is_reschedule, jadwal_semula, guru_pengganti_id) VALUES (?, ?, ?, ?, ?, ?, ?)`, kelasID, tanggal, jam, status, reschedule, semula, p)
	require.NoError(t, err)
}

func slotKelas(slots []models.GuruSlotKelas, kelasID int64) []models.GuruSlotKelas {
	out := []models.GuruSlotKelas{}
	for _, s := range slots {
		if s.KelasID == kelasID {
			out = append(out, s)
		}
	}
	return out
}

func TestBerandaHariIniDanAbsensiTertunda(t *testing.T) {
	f := setupBeranda(t)
	senin := f.kelas(t, "Senin Rutin", "Senin, jam 20.30 WIB", f.guruA, "2026-09-01", 3)
	kamis := f.kelas(t, "Kamis Lupa", "Kamis, jam 19.00 WIB", f.guruA, "2026-09-01", 2)
	tidakTerbaca := f.kelas(t, "Tak Terbaca", "Mingguan", f.guruA, "2026-09-01", 2)
	pindah := f.kelas(t, "Selasa Pindah", "Selasa, jam 19.00 WIB", f.guruA, "2026-09-01", 2)
	batal := f.kelas(t, "Rabu Batal", "Rabu, jam 19.00 WIB", f.guruA, "2026-09-01", 2)
	kosong := f.kelas(t, "Tanpa Santri", "Senin, jam 10.00 WIB", f.guruA, "2026-09-01", 0)
	baru := f.kelas(t, "Kelas Baru", "Senin, jam 13.00 WIB", f.guruA, "2026-09-27", 1)
	berlangsung := f.kelas(t, "Sedang Jalan", "Senin, jam 07.30 WIB", f.guruA, "2026-09-01", 1)
	milikB := f.kelas(t, "Kelas Guru B", "Senin, jam 05.00 WIB", f.guruB, "2026-09-01", 2)

	f.pertemuan(t, senin, "2026-09-21", "selesai")
	f.pertemuan(t, berlangsung, "2026-09-21", "selesai")
	f.pertemuan(t, berlangsung, "2026-09-28", "berlangsung")
	f.jadwal(t, pindah, "2026-09-26", "16:00", "dijadwalkan", "2026-09-22 19:00", 0)
	f.jadwal(t, batal, "2026-09-23", "19:00", "dibatalkan", "", 0)
	f.jadwal(t, milikB, "2026-09-28", "05:00", "dijadwalkan", "", f.guruA)

	b, err := f.service.GetBeranda(&f.guruA, f.today)
	require.NoError(t, err)
	require.Equal(t, "2026-09-28", b.Tanggal)

	// Hari ini: kelas Senin milik A, kelas baru, kelas berlangsung, dan badal di kelas B.
	require.Len(t, slotKelas(b.HariIni, senin), 1)
	require.Equal(t, models.SlotBelum, slotKelas(b.HariIni, senin)[0].Status)
	require.Equal(t, "20:30", slotKelas(b.HariIni, senin)[0].Jam)
	require.Len(t, slotKelas(b.HariIni, baru), 1)
	require.Equal(t, models.SlotBerlangsung, slotKelas(b.HariIni, berlangsung)[0].Status)
	badal := slotKelas(b.HariIni, milikB)
	require.Len(t, badal, 1)
	require.True(t, badal[0].SebagaiBadal)
	require.Equal(t, "Menggantikan Guru B", badal[0].Keterangan)
	require.Empty(t, slotKelas(b.HariIni, kosong))
	// Urut berdasarkan jam: 05:00 (badal) lebih dulu.
	require.Equal(t, milikB, b.HariIni[0].KelasID)

	// Tertunda: Kamis 24/9 dan pertemuan pindahan Sabtu 26/9. Senin 21/9 sudah diabsen,
	// Selasa 22/9 dipindah, Rabu 23/9 dibatalkan, kelas baru belum ada pada 21/9.
	require.Len(t, b.Tertunda, 2)
	require.Equal(t, kamis, b.Tertunda[0].KelasID)
	require.Equal(t, "2026-09-24", b.Tertunda[0].Tanggal)
	require.Equal(t, pindah, b.Tertunda[1].KelasID)
	require.Equal(t, "2026-09-26", b.Tertunda[1].Tanggal)
	require.Equal(t, "jadwal", b.Tertunda[1].Sumber)
	require.Equal(t, "Dipindah dari 2026-09-22", b.Tertunda[1].Keterangan)
	require.Empty(t, slotKelas(b.Tertunda, batal))

	require.Len(t, b.JadwalTakTerbaca, 1)
	require.Equal(t, tidakTerbaca, b.JadwalTakTerbaca[0].KelasID)

	// Guru B melihat kelasnya dibadalkan, bukan tugas yang harus dikerjakan.
	bB, err := f.service.GetBeranda(&f.guruB, f.today)
	require.NoError(t, err)
	require.Len(t, bB.HariIni, 1)
	require.Equal(t, models.SlotDibadalkan, bB.HariIni[0].Status)
	require.Equal(t, "Dibadalkan oleh Guru A", bB.HariIni[0].Keterangan)
	// Hanya Senin lalu (21/9) yang tertunda; hari ini dibadalkan, bukan tertunda.
	require.Len(t, bB.Tertunda, 1)
	require.Equal(t, "2026-09-21", bB.Tertunda[0].Tanggal)

	// Admin (tanpa guru) melihat semua kelas tanpa kinerja pribadi.
	admin, err := f.service.GetBeranda(nil, f.today)
	require.NoError(t, err)
	require.Nil(t, admin.Kinerja)
	require.NotEmpty(t, slotKelas(admin.HariIni, milikB))
}

func TestBerandaPertemuanDimajukanSehariDianggapSelesai(t *testing.T) {
	f := setupBeranda(t)
	kelas := f.kelas(t, "Kamis", "Kamis, jam 19.00 WIB", f.guruA, "2026-09-01", 1)
	f.pertemuan(t, kelas, "2026-09-23", "selesai") // Rabu, sehari sebelum jadwal Kamis
	b, err := f.service.GetBeranda(&f.guruA, f.today)
	require.NoError(t, err)
	require.Empty(t, b.Tertunda)
}

func TestBerandaKinerjaDanAgenda(t *testing.T) {
	f := setupBeranda(t)
	for _, tgl := range []string{"2026-09-25", "2026-09-26", "2026-09-27", "2026-09-10"} {
		_, err := f.db.Exec(`INSERT INTO tilawah_harian (guru_id, tanggal) VALUES (?, ?)`, f.guruA, tgl)
		require.NoError(t, err)
	}
	_, err := f.db.Exec(`INSERT INTO pembinaan (tanggal, topik, status) VALUES ('2026-10-01', 'Adab guru', 'dijadwalkan'), ('2026-09-01', 'Lama', 'dijadwalkan')`)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO rapat_guru (tanggal, judul, status) VALUES ('2026-10-05', 'Rapat bulanan', 'dijadwalkan')`)
	require.NoError(t, err)

	b, err := f.service.GetBeranda(&f.guruA, f.today)
	require.NoError(t, err)
	require.NotNil(t, b.Kinerja)
	require.False(t, b.Kinerja.SudahTilawah)
	require.Equal(t, 3, b.Kinerja.TilawahStreak)
	require.Equal(t, int64(4), b.Kinerja.TilawahBulanIni)
	require.Nil(t, b.Kinerja.TsiTotal)

	require.Len(t, b.Agenda, 2)
	require.Equal(t, models.GuruAgendaItem{Jenis: "pembinaan", Tanggal: "2026-10-01", Judul: "Adab guru"}, b.Agenda[0])
	require.Equal(t, "rapat", b.Agenda[1].Jenis)
}
