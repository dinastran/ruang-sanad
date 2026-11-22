package services

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

// FilterRiwayatAbsensi mengikuti filter halaman Riwayat Absensi Guru.
// Mulai kosong = tanpa batas awal. Status hanya menyaring sheet Detail.
type FilterRiwayatAbsensi struct {
	Mulai    string // YYYY-MM-DD, opsional
	Sampai   string // YYYY-MM-DD
	Kegiatan string // "", pembinaan, rapat, kunjungan
	GuruID   int64
	Cari     string
	Status   string // "", hadir, izin, sakit, alfa
}

type RekapKehadiranGuru struct {
	Hadir, Izin, Sakit, Alfa, Total int
	// Persen hadir; -1 bila belum ada catatan absen.
	Persen float64
}

type RekapRiwayatGuru struct {
	GuruID              int64
	Nama                string
	Rekap               RekapKehadiranGuru
	Kunjungan           int
	KunjunganTerlaksana int
}

type LaporanRiwayatAbsensi struct {
	Filter              FilterRiwayatAbsensi
	Ringkasan           RekapKehadiranGuru
	Sesi                int
	SesiPembinaan       int
	Kunjungan           int
	KunjunganTerlaksana int
	PerGuru             []RekapRiwayatGuru
	// Bulan (YYYY-MM) urut naik, dan Matriks[guruID][bulan].
	Bulan   []string
	Matriks map[int64]map[string]RekapKehadiranGuru
	Detail  []models.RiwayatAbsensiGuru
}

var statusKehadiranValid = map[string]bool{"": true, "hadir": true, "izin": true, "sakit": true, "alfa": true}

// NormalisasiFilterRiwayat memvalidasi tanggal dan nilai filter.
func NormalisasiFilterRiwayat(f FilterRiwayatAbsensi, today time.Time) (FilterRiwayatAbsensi, error) {
	f.Mulai, f.Sampai = strings.TrimSpace(f.Mulai), strings.TrimSpace(f.Sampai)
	if f.Sampai == "" {
		f.Sampai = today.Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", f.Sampai); err != nil {
		return f, errors.New("tanggal sampai tidak valid")
	}
	if f.Mulai != "" {
		if _, err := time.Parse("2006-01-02", f.Mulai); err != nil {
			return f, errors.New("tanggal mulai tidak valid")
		}
		if f.Mulai > f.Sampai {
			return f, errors.New("tanggal mulai melewati tanggal sampai")
		}
	}
	switch f.Kegiatan {
	case "", "pembinaan", "rapat", "kunjungan":
	default:
		return f, errors.New("kegiatan tidak valid")
	}
	if !statusKehadiranValid[f.Status] {
		return f, errors.New("status tidak valid")
	}
	f.Cari = strings.ToLower(strings.TrimSpace(f.Cari))
	return f, nil
}

// StatusRiwayat sama dengan statusRiwayat di frontend (lib/riwayatAbsensi.ts).
func StatusRiwayat(r models.RiwayatAbsensiGuru) string {
	if r.Kegiatan == "kunjungan" {
		if r.Hadir {
			return "terlaksana"
		}
		if r.Alasan == "batal" {
			return "batal"
		}
		return "dijadwalkan"
	}
	switch r.Kehadiran {
	case "hadir", "izin", "sakit", "alfa":
		return r.Kehadiran
	}
	if r.Hadir {
		return "hadir"
	}
	return "alfa"
}

func isAbsensiRiwayat(r models.RiwayatAbsensiGuru) bool {
	return r.Kegiatan == "pembinaan" || r.Kegiatan == "rapat"
}

func (r *RekapKehadiranGuru) tambah(row models.RiwayatAbsensiGuru) {
	switch StatusRiwayat(row) {
	case "hadir":
		r.Hadir++
	case "izin":
		r.Izin++
	case "sakit":
		r.Sakit++
	default:
		r.Alfa++
	}
	r.Total++
}

func (r *RekapKehadiranGuru) hitungPersen() {
	r.Persen = -1
	if r.Total > 0 {
		r.Persen = float64(r.Hadir) / float64(r.Total) * 100
	}
}

// AnalisisRiwayatAbsensi menghitung ringkasan, rekap per guru, matriks guru×bulan,
// dan detail dari baris riwayat yang lolos filter (logika sama dengan dashboard).
func AnalisisRiwayatAbsensi(rows []models.RiwayatAbsensiGuru, f FilterRiwayatAbsensi) *LaporanRiwayatAbsensi {
	lap := &LaporanRiwayatAbsensi{Filter: f, Matriks: map[int64]map[string]RekapKehadiranGuru{}, Detail: []models.RiwayatAbsensiGuru{}}
	perGuru := map[int64]*RekapRiwayatGuru{}
	sesi, sesiPembinaan := map[string]bool{}, map[string]bool{}
	bulan := map[string]bool{}

	for _, r := range rows {
		t := r.Tanggal
		if len(t) > 10 {
			t = t[:10]
		}
		if (f.Mulai != "" && t < f.Mulai) || t > f.Sampai {
			continue
		}
		if f.Kegiatan != "" && r.Kegiatan != f.Kegiatan {
			continue
		}
		if f.GuruID != 0 && r.GuruID != f.GuruID {
			continue
		}
		if f.Cari != "" && !strings.Contains(strings.ToLower(r.GuruNama+" "+r.Judul), f.Cari) {
			continue
		}
		g := perGuru[r.GuruID]
		if g == nil {
			g = &RekapRiwayatGuru{GuruID: r.GuruID, Nama: r.GuruNama}
			perGuru[r.GuruID] = g
		}
		if isAbsensiRiwayat(r) {
			lap.Ringkasan.tambah(r)
			g.Rekap.tambah(r)
			key := r.Kegiatan + "-" + strconv.FormatInt(r.KegiatanID, 10)
			sesi[key] = true
			if r.Kegiatan == "pembinaan" {
				sesiPembinaan[key] = true
			}
			ym := t[:7]
			bulan[ym] = true
			if lap.Matriks[r.GuruID] == nil {
				lap.Matriks[r.GuruID] = map[string]RekapKehadiranGuru{}
			}
			m := lap.Matriks[r.GuruID][ym]
			m.tambah(r)
			lap.Matriks[r.GuruID][ym] = m
		} else {
			lap.Kunjungan++
			g.Kunjungan++
			if r.Hadir {
				lap.KunjunganTerlaksana++
				g.KunjunganTerlaksana++
			}
		}
		if f.Status == "" || StatusRiwayat(r) == f.Status {
			lap.Detail = append(lap.Detail, r)
		}
	}

	lap.Ringkasan.hitungPersen()
	lap.Sesi, lap.SesiPembinaan = len(sesi), len(sesiPembinaan)
	for _, g := range perGuru {
		g.Rekap.hitungPersen()
		lap.PerGuru = append(lap.PerGuru, *g)
	}
	// Alfa terbanyak dulu, lalu % hadir terendah, lalu nama (sama dengan "Guru alfa terbanyak").
	sort.Slice(lap.PerGuru, func(i, j int) bool {
		a, b := lap.PerGuru[i], lap.PerGuru[j]
		if a.Rekap.Alfa != b.Rekap.Alfa {
			return a.Rekap.Alfa > b.Rekap.Alfa
		}
		if a.Rekap.Persen != b.Rekap.Persen {
			return a.Rekap.Persen < b.Rekap.Persen
		}
		return a.Nama < b.Nama
	})
	for ym := range bulan {
		lap.Bulan = append(lap.Bulan, ym)
	}
	sort.Strings(lap.Bulan)
	for id, per := range lap.Matriks {
		for ym, m := range per {
			m.hitungPersen()
			lap.Matriks[id][ym] = m
		}
	}
	return lap
}

// LaporanRiwayatAbsensi memuat riwayat lalu menganalisisnya sesuai filter.
func (s *KoordinatorService) LaporanRiwayatAbsensi(f FilterRiwayatAbsensi, today time.Time) (*LaporanRiwayatAbsensi, error) {
	f, err := NormalisasiFilterRiwayat(f, today)
	if err != nil {
		return nil, err
	}
	rows, err := s.ListRiwayatAbsensiGuru()
	if err != nil {
		return nil, err
	}
	return AnalisisRiwayatAbsensi(rows, f), nil
}
