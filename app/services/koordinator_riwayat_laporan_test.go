package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func riwayatContoh() []models.RiwayatAbsensiGuru {
	return []models.RiwayatAbsensiGuru{
		{GuruID: 1, GuruNama: "Ahmad", Kegiatan: "pembinaan", KegiatanID: 10, Tanggal: "2026-10-05", Judul: "Tahsin", Hadir: true, Kehadiran: "hadir"},
		{GuruID: 2, GuruNama: "Budi", Kegiatan: "pembinaan", KegiatanID: 10, Tanggal: "2026-10-05", Judul: "Tahsin", Hadir: false, Kehadiran: ""},
		{GuruID: 1, GuruNama: "Ahmad", Kegiatan: "rapat", KegiatanID: 20, Tanggal: "2026-11-03", Judul: "Rapat bulanan", Hadir: false, Kehadiran: "izin"},
		{GuruID: 2, GuruNama: "Budi", Kegiatan: "rapat", KegiatanID: 20, Tanggal: "2026-11-03", Judul: "Rapat bulanan", Hadir: false, Kehadiran: "alfa"},
		{GuruID: 2, GuruNama: "Budi", Kegiatan: "kunjungan", KegiatanID: 30, Tanggal: "2026-11-10", Judul: "Kelas A", Hadir: true},
		{GuruID: 1, GuruNama: "Ahmad", Kegiatan: "kunjungan", KegiatanID: 31, Tanggal: "2026-11-12", Judul: "Kelas B", Hadir: false, Alasan: "batal"},
		{GuruID: 1, GuruNama: "Ahmad", Kegiatan: "pembinaan", KegiatanID: 11, Tanggal: "2026-12-01", Judul: "Di luar rentang", Hadir: true, Kehadiran: "hadir"},
	}
}

func TestAnalisisRiwayatAbsensi(t *testing.T) {
	f, err := NormalisasiFilterRiwayat(FilterRiwayatAbsensi{Mulai: "2026-10-01", Sampai: "2026-11-30"}, time.Now())
	require.NoError(t, err)
	lap := AnalisisRiwayatAbsensi(riwayatContoh(), f)

	require.Equal(t, RekapKehadiranGuru{Hadir: 1, Izin: 1, Sakit: 0, Alfa: 2, Total: 4, Persen: 25}, lap.Ringkasan)
	require.Equal(t, 2, lap.Sesi)
	require.Equal(t, 1, lap.SesiPembinaan)
	require.Equal(t, 2, lap.Kunjungan)
	require.Equal(t, 1, lap.KunjunganTerlaksana)
	require.Len(t, lap.Detail, 6)

	// Budi: 2 alfa → urutan pertama.
	require.Len(t, lap.PerGuru, 2)
	require.Equal(t, "Budi", lap.PerGuru[0].Nama)
	require.Equal(t, 2, lap.PerGuru[0].Rekap.Alfa)
	require.EqualValues(t, 0, lap.PerGuru[0].Rekap.Persen)
	require.Equal(t, 1, lap.PerGuru[0].KunjunganTerlaksana)
	require.Equal(t, 1, lap.PerGuru[1].Kunjungan)
	require.Equal(t, 0, lap.PerGuru[1].KunjunganTerlaksana)

	require.Equal(t, []string{"2026-10", "2026-11"}, lap.Bulan)
	require.EqualValues(t, 100, lap.Matriks[1]["2026-10"].Persen)
	require.Equal(t, 1, lap.Matriks[1]["2026-11"].Izin)
	require.Equal(t, 1, lap.Matriks[2]["2026-11"].Alfa)
}

func TestAnalisisRiwayatAbsensiFilter(t *testing.T) {
	// Status hanya menyaring Detail; ringkasan tetap menghitung semua status.
	f, err := NormalisasiFilterRiwayat(FilterRiwayatAbsensi{Mulai: "2026-10-01", Sampai: "2026-11-30", Status: "alfa"}, time.Now())
	require.NoError(t, err)
	lap := AnalisisRiwayatAbsensi(riwayatContoh(), f)
	require.Equal(t, 4, lap.Ringkasan.Total)
	require.Len(t, lap.Detail, 2)
	for _, r := range lap.Detail {
		require.Equal(t, "alfa", StatusRiwayat(r))
	}

	f, err = NormalisasiFilterRiwayat(FilterRiwayatAbsensi{Sampai: "2026-11-30", Kegiatan: "rapat", Cari: " BUDI "}, time.Now())
	require.NoError(t, err)
	lap = AnalisisRiwayatAbsensi(riwayatContoh(), f)
	require.Len(t, lap.Detail, 1)
	require.Equal(t, int64(2), lap.Detail[0].GuruID)
	require.EqualValues(t, -1, AnalisisRiwayatAbsensi(nil, f).Ringkasan.Persen)

	_, err = NormalisasiFilterRiwayat(FilterRiwayatAbsensi{Mulai: "2026-12-01", Sampai: "2026-11-01"}, time.Now())
	require.Error(t, err)
	_, err = NormalisasiFilterRiwayat(FilterRiwayatAbsensi{Sampai: "2026-11-01", Status: "telat"}, time.Now())
	require.Error(t, err)
}
