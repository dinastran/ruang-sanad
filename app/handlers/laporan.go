package handlers

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/xuri/excelize/v2"
)

type LaporanHandler struct {
	laporanService *services.LaporanService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewLaporanHandler(laporanService *services.LaporanService, store *session.Store, inertiaService *services.InertiaService) *LaporanHandler {
	return &LaporanHandler{
		laporanService: laporanService,
		store:          store,
		inertiaService: inertiaService,
	}
}

// laporanFilter reads the query string. The per-santri table shows aktif
// santri unless status is given; status=semua shows every status.
func laporanFilter(c *fiber.Ctx) models.LaporanKeuanganFilter {
	status := c.Query("status", services.StatusSantriAktif)
	if status == "semua" {
		status = ""
	}
	return models.LaporanKeuanganFilter{
		Dari:          c.Query("dari"),
		Sampai:        c.Query("sampai"),
		Status:        status,
		AngkatanKelas: c.Query("angkatan_kelas"),
	}
}

// Keuangan renders the periodic finance report.
func (h *LaporanHandler) Keuangan(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	laporan, err := h.laporanService.GetLaporanKeuangan(laporanFilter(c))
	if errors.Is(err, services.ErrLaporanPeriodeTidakValid) {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/laporan/keuangan")
	}
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/LaporanKeuangan", fiber.Map{
		"user":    sessionUser(sess),
		"laporan": laporan,
	})
}

// Export downloads the same report as an Excel workbook.
func (h *LaporanHandler) Export(c *fiber.Ctx) error {
	l, err := h.laporanService.GetLaporanKeuangan(laporanFilter(c))
	if errors.Is(err, services.ErrLaporanPeriodeTidakValid) {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err != nil {
		return err
	}

	f := excelize.NewFile()
	defer f.Close()
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	money, _ := f.NewStyle(&excelize.Style{NumFmt: 3})
	header := func(sheet string, cols []string) {
		f.SetSheetRow(sheet, "A1", &cols)
		last, _ := excelize.CoordinatesToCellName(len(cols), 1)
		f.SetCellStyle(sheet, "A1", last, bold)
	}

	ringkasan := "Ringkasan"
	_ = f.SetSheetName(f.GetSheetName(0), ringkasan)
	r := l.Ringkasan
	baris := [][]interface{}{
		{"Laporan Keuangan", l.Filter.Dari + " s.d. " + l.Filter.Sampai},
		{"Catatan", "Belum termasuk biaya operasional di luar sistem."},
		{},
		{"PEMASUKAN", ""},
		{fmt.Sprintf("SPP / infaq (%d pembayaran)", r.JumlahBayarSpp), r.PemasukanSpp},
		{fmt.Sprintf("Produk (%d order)", r.JumlahOrderProduk), r.PemasukanProduk},
		{"Total pemasukan", r.TotalPemasukan},
		{},
		{"PENGELUARAN", ""},
		{fmt.Sprintf("Ujroh guru dibayar (%d guru-bulan, bersih)", r.JumlahUjroh), r.PengeluaranUjroh},
		{fmt.Sprintf("Kasbon dicairkan (%d)", r.JumlahKasbon), r.PengeluaranKasbon},
		{"Total pengeluaran", r.TotalPengeluaran},
		{},
		{"SALDO PERIODE", r.Saldo},
		{},
		{"TAGIHAN TERBIT DI PERIODE", ""},
		{fmt.Sprintf("Terbit (%d tagihan)", l.Tagihan.Jumlah), l.Tagihan.Nominal},
		{fmt.Sprintf("Sudah lunas (%d tagihan)", l.Tagihan.JumlahLunas), l.Tagihan.NominalLunas},
		{},
		{"PIUTANG PER HARI INI", ""},
		{fmt.Sprintf("Belum dibayar (%d tagihan, %d santri)", l.Piutang.JumlahTagihan, l.Piutang.JumlahSantri), l.Piutang.BelumBayar},
		{fmt.Sprintf("Lewat jatuh tempo (%d tagihan)", l.Piutang.JumlahTerlambat), l.Piutang.Terlambat},
	}
	for i, row := range baris {
		f.SetSheetRow(ringkasan, fmt.Sprintf("A%d", i+1), &row)
	}
	for _, i := range []int{1, 4, 7, 9, 12, 14, 16, 20} {
		f.SetCellStyle(ringkasan, fmt.Sprintf("A%d", i), fmt.Sprintf("B%d", i), bold)
	}
	f.SetCellStyle(ringkasan, "B5", fmt.Sprintf("B%d", len(baris)), money)
	f.SetColWidth(ringkasan, "A", "A", 48)
	f.SetColWidth(ringkasan, "B", "B", 22)

	pembayaran := "Pembayaran SPP"
	_, _ = f.NewSheet(pembayaran)
	header(pembayaran, []string{"Tanggal Bayar", "ID Mahasantri", "Nama", "Angkatan Kelas", "Kelas", "Bulan Ke", "Nominal", "Metode", "Catatan", "Dicatat Oleh"})
	for i, p := range l.Pembayaran {
		f.SetSheetRow(pembayaran, fmt.Sprintf("A%d", i+2), &[]interface{}{p.TanggalBayar, p.IDMahasantri, p.SantriNama, p.AngkatanKelas, p.KelasNama, p.BulanKe, p.Nominal, p.Metode, p.Catatan, p.DicatatOleh})
	}
	f.SetCellStyle(pembayaran, "G2", fmt.Sprintf("G%d", len(l.Pembayaran)+1), money)
	f.SetColWidth(pembayaran, "C", "C", 30)

	pengeluaran := "Pengeluaran"
	_, _ = f.NewSheet(pengeluaran)
	header(pengeluaran, []string{"Tanggal", "Jenis", "Guru", "Bulan Ujroh", "Pokok", "Komponen", "Potongan Kasbon", "Nominal Keluar", "Keterangan"})
	row := 2
	for _, u := range l.Ujroh {
		f.SetSheetRow(pengeluaran, fmt.Sprintf("A%d", row), &[]interface{}{u.DibayarAt, "Ujroh", u.GuruNama, u.Bulan, u.Pokok, u.Komponen, u.Kasbon, u.Diterima, u.Catatan})
		row++
	}
	for _, k := range l.Kasbon {
		f.SetSheetRow(pengeluaran, fmt.Sprintf("A%d", row), &[]interface{}{k.Tanggal, "Kasbon", k.GuruNama, "", "", "", "", k.Nominal, k.Keterangan})
		row++
	}
	f.SetCellStyle(pengeluaran, "E2", fmt.Sprintf("H%d", row), money)
	f.SetColWidth(pengeluaran, "C", "C", 28)

	perSantri := "Per Santri"
	_, _ = f.NewSheet(perSantri)
	header(perSantri, []string{"ID Mahasantri", "Nama", "Angkatan Daftar", "Angkatan Kelas", "Kelas", "Status", "Nominal SPP",
		"Terbit di Periode", "Dibayar di Periode", "Belum Dibayar", "Lewat Jatuh Tempo", "Pembayaran Terakhir", "Bulan Ke", "Nominal Terakhir"})
	for i, s := range l.Santri {
		var tgl, bulanKe, nominal interface{} = s.CatatanInfaqLama, "", ""
		if s.PembayaranTerakhir != nil {
			tgl, bulanKe, nominal = s.PembayaranTerakhir.Tanggal, s.PembayaranTerakhir.BulanKe, s.PembayaranTerakhir.Nominal
		}
		f.SetSheetRow(perSantri, fmt.Sprintf("A%d", i+2), &[]interface{}{s.IDMahasantri, s.Nama, s.Angkatan, s.AngkatanKelas, s.KelasNama, s.Status, s.NominalSpp,
			s.TerbitPeriode, s.DibayarPeriode, s.BelumBayar, s.Terlambat, tgl, bulanKe, nominal})
	}
	f.SetCellStyle(perSantri, "G2", fmt.Sprintf("K%d", len(l.Santri)+1), money)
	f.SetCellStyle(perSantri, "N2", fmt.Sprintf("N%d", len(l.Santri)+1), money)
	f.SetColWidth(perSantri, "B", "B", 30)

	tidakLanjut := "Tidak Lanjut"
	_, _ = f.NewSheet(tidakLanjut)
	header(tidakLanjut, []string{"ID Mahasantri", "Nama", "Angkatan Daftar", "Angkatan Kelas", "Keterangan", "Belum Dibayar"})
	for i, s := range l.TidakLanjut {
		f.SetSheetRow(tidakLanjut, fmt.Sprintf("A%d", i+2), &[]interface{}{s.IDMahasantri, s.Nama, s.Angkatan, s.AngkatanKelas, s.KeteranganTidakLanjut, s.BelumBayar})
	}
	f.SetCellStyle(tidakLanjut, "F2", fmt.Sprintf("F%d", len(l.TidakLanjut)+1), money)
	f.SetColWidth(tidakLanjut, "B", "B", 30)
	f.SetColWidth(tidakLanjut, "E", "E", 40)

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	nama := fmt.Sprintf("laporan-keuangan-%s_%s.xlsx", l.Filter.Dari, l.Filter.Sampai)
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, "attachment; filename="+url.PathEscape(nama))
	return c.Send(buffer.Bytes())
}
