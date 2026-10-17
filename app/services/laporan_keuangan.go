package services

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var ErrLaporanPeriodeTidakValid = errors.New("periode laporan tidak valid: gunakan tanggal YYYY-MM-DD dan tanggal awal tidak boleh setelah tanggal akhir")

// maxHariLaporan caps the period so one request stays light.
const maxHariLaporan = 3 * 366

// NormalizeLaporanFilter validates the period; empty dates default to the
// current month (WIB).
func (s *LaporanService) NormalizeLaporanFilter(f models.LaporanKeuanganFilter) (models.LaporanKeuanganFilter, error) {
	hariIni := HariIniRiayah()
	if s.now != nil {
		hariIni = s.now()
	}
	f.Dari, f.Sampai = strings.TrimSpace(f.Dari), strings.TrimSpace(f.Sampai)
	if f.Dari == "" && f.Sampai == "" {
		awal := time.Date(hariIni.Year(), hariIni.Month(), 1, 0, 0, 0, 0, hariIni.Location())
		f.Dari = awal.Format("2006-01-02")
		f.Sampai = awal.AddDate(0, 1, -1).Format("2006-01-02")
	}
	dari, err1 := time.Parse("2006-01-02", f.Dari)
	sampai, err2 := time.Parse("2006-01-02", f.Sampai)
	if err1 != nil || err2 != nil || sampai.Before(dari) || sampai.Sub(dari).Hours()/24 > maxHariLaporan {
		return f, ErrLaporanPeriodeTidakValid
	}
	f.Status = strings.TrimSpace(f.Status)
	f.AngkatanKelas = strings.TrimSpace(f.AngkatanKelas)
	return f, nil
}

// GetLaporanKeuangan builds the cash report for a period: SPP and product
// income by payment/order date, ujroh paid and kasbon handed out, tagihan
// issued in the period, outstanding SPP today, and one row per santri.
func (s *LaporanService) GetLaporanKeuangan(f models.LaporanKeuanganFilter) (*models.LaporanKeuangan, error) {
	f, err := s.NormalizeLaporanFilter(f)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	out := &models.LaporanKeuangan{
		Filter: f, SppPerMetode: []models.LaporanKelompok{}, SppPerAngkatan: []models.LaporanKelompok{},
		Pembayaran: []models.LaporanPembayaran{}, Ujroh: []models.LaporanUjrohDibayar{}, Kasbon: []models.LaporanKasbon{},
		Santri: []models.LaporanSantri{}, TidakLanjut: []models.LaporanSantri{}, OpsiAngkatan: []string{},
	}

	bayar, err := s.querier.ListLaporanPembayaran(ctx, f.Dari, f.Sampai)
	if err != nil {
		return nil, err
	}
	perMetode, perAngkatan := map[string]*models.LaporanKelompok{}, map[string]*models.LaporanKelompok{}
	for _, r := range bayar {
		out.Pembayaran = append(out.Pembayaran, models.LaporanPembayaran{
			TagihanID: r.TagihanID, TanggalBayar: r.TanggalBayar, SantriID: r.SantriID, IDMahasantri: r.IDMahasantri,
			SantriNama: r.SantriNama, AngkatanKelas: r.AngkatanKelas, KelasNama: r.KelasNama, BulanKe: r.BulanKe,
			Nominal: r.Nominal, Metode: r.Metode, Catatan: r.Catatan, DicatatOleh: r.DicatatOleh,
		})
		out.Ringkasan.PemasukanSpp += r.Nominal
		out.Ringkasan.JumlahBayarSpp++
		tambahKelompok(perMetode, labelAtauDefault(r.Metode, "Tanpa keterangan metode"), r.Nominal)
		tambahKelompok(perAngkatan, labelAtauDefault(r.AngkatanKelas, "Tanpa angkatan"), r.Nominal)
	}
	out.SppPerMetode = urutkanKelompok(perMetode)
	out.SppPerAngkatan = urutkanKelompok(perAngkatan)

	produk, err := s.querier.LaporanOmzetProduk(ctx, f.Dari, f.Sampai)
	if err != nil {
		return nil, err
	}
	out.Ringkasan.PemasukanProduk, out.Ringkasan.JumlahOrderProduk = produk.Omzet, produk.JumlahOrder

	ujroh, err := s.querier.ListLaporanUjrohDibayar(ctx, f.Dari, f.Sampai)
	if err != nil {
		return nil, err
	}
	for _, r := range ujroh {
		out.Ujroh = append(out.Ujroh, models.LaporanUjrohDibayar{
			Bulan: r.Bulan, GuruNama: r.GuruNama, Pokok: r.Pokok, Komponen: r.Komponen, Kasbon: r.Kasbon,
			Diterima: r.Diterima, DibayarAt: r.DibayarAt, Catatan: r.Catatan,
		})
		out.Ringkasan.PengeluaranUjroh += r.Diterima
		out.Ringkasan.JumlahUjroh++
	}

	kasbon, err := s.querier.ListLaporanKasbon(ctx, f.Dari, f.Sampai)
	if err != nil {
		return nil, err
	}
	for _, r := range kasbon {
		out.Kasbon = append(out.Kasbon, models.LaporanKasbon{Tanggal: r.Tanggal, GuruNama: r.GuruNama, Nominal: r.Nominal, Keterangan: r.Keterangan})
		out.Ringkasan.PengeluaranKasbon += r.Nominal
		out.Ringkasan.JumlahKasbon++
	}

	out.Ringkasan.TotalPemasukan = out.Ringkasan.PemasukanSpp + out.Ringkasan.PemasukanProduk
	out.Ringkasan.TotalPengeluaran = out.Ringkasan.PengeluaranUjroh + out.Ringkasan.PengeluaranKasbon
	out.Ringkasan.Saldo = out.Ringkasan.TotalPemasukan - out.Ringkasan.TotalPengeluaran

	terbit, err := s.querier.LaporanTagihanTerbit(ctx, f.Dari, f.Sampai)
	if err != nil {
		return nil, err
	}
	out.Tagihan = models.LaporanTagihanTerbit{
		Jumlah: terbit.Jumlah, Nominal: terbit.Nominal, JumlahLunas: terbit.JumlahLunas,
		NominalLunas: terbit.NominalLunas, JumlahBatal: terbit.JumlahBatal,
	}
	if terbit.Nominal > 0 {
		out.Tagihan.PersenLunas = float64(terbit.NominalLunas) / float64(terbit.Nominal) * 100
	}

	hariIni := HariIniRiayah()
	if s.now != nil {
		hariIni = s.now()
	}
	// All santri, so piutang and the tidak lanjut list ignore the table filter.
	semua, err := s.querier.ListLaporanSantri(ctx, f.Dari, f.Sampai, hariIni.Format("2006-01-02"), "", "")
	if err != nil {
		return nil, err
	}
	angkatan := map[string]struct{}{}
	for _, r := range semua {
		row := laporanSantri(r)
		if r.AngkatanKelas != "" {
			angkatan[r.AngkatanKelas] = struct{}{}
		}
		if r.BelumBayar > 0 {
			out.Piutang.BelumBayar += r.BelumBayar
			out.Piutang.JumlahTagihan += r.JumlahBelumBayar
			out.Piutang.Terlambat += r.Terlambat
			out.Piutang.JumlahTerlambat += r.JumlahTerlambat
			out.Piutang.JumlahSantri++
		}
		if r.Status == StatusSantriTidakLanjut {
			out.TidakLanjut = append(out.TidakLanjut, row)
		}
		if (f.Status == "" || r.Status == f.Status) && (f.AngkatanKelas == "" || r.AngkatanKelas == f.AngkatanKelas) {
			out.Santri = append(out.Santri, row)
		}
	}
	for a := range angkatan {
		out.OpsiAngkatan = append(out.OpsiAngkatan, a)
	}
	sort.Strings(out.OpsiAngkatan)
	return out, nil
}

func laporanSantri(r queries.LaporanSantriRow) models.LaporanSantri {
	row := models.LaporanSantri{
		SantriID: r.SantriID, IDMahasantri: r.IDMahasantri, Nama: r.Nama, Angkatan: r.Angkatan,
		AngkatanKelas: r.AngkatanKelas, KelasNama: r.KelasNama, Status: r.Status, NominalSpp: r.NominalSpp,
		KeteranganTidakLanjut: r.KeteranganTidakLanjut, TerbitPeriode: r.TerbitPeriode, DibayarPeriode: r.DibayarPeriode,
		BelumBayar: r.BelumBayar, JumlahBelumBayar: r.JumlahBelumBayar, Terlambat: r.Terlambat, JumlahTerlambat: r.JumlahTerlambat,
	}
	if r.TerakhirTanggal.Valid {
		row.PembayaranTerakhir = &models.LaporanPembayaranTerakhir{
			Tanggal: r.TerakhirTanggal.String, BulanKe: r.TerakhirBulanKe.Int64,
			Nominal: r.TerakhirNominal.Int64, Metode: r.TerakhirMetode.String,
		}
	} else {
		row.CatatanInfaqLama = strings.TrimSpace(r.InfaqTerakhirManual)
	}
	return row
}

func labelAtauDefault(s, def string) string {
	if s = strings.TrimSpace(s); s != "" {
		return s
	}
	return def
}

func tambahKelompok(m map[string]*models.LaporanKelompok, nama string, nominal int64) {
	// Group case-insensitively ("Transfer BSI" and "transfer bsi" are one method).
	key := strings.ToLower(nama)
	g, ok := m[key]
	if !ok {
		g = &models.LaporanKelompok{Nama: nama}
		m[key] = g
	}
	g.Jumlah++
	g.Nominal += nominal
}

func urutkanKelompok(m map[string]*models.LaporanKelompok) []models.LaporanKelompok {
	out := make([]models.LaporanKelompok, 0, len(m))
	for _, g := range m {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Nominal != out[j].Nominal {
			return out[i].Nominal > out[j].Nominal
		}
		return out[i].Nama < out[j].Nama
	})
	return out
}
