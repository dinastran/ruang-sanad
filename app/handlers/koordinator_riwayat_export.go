package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/xuri/excelize/v2"
)

// RiwayatExcel mengunduh analisis Riwayat Absensi Guru sesuai filter halaman:
// Ringkasan, Rekap per Guru, Guru × Bulan, dan Detail catatan.
func (h *KoordinatorAttendanceExportHandler) RiwayatExcel(c *fiber.Ctx) error {
	filter := services.FilterRiwayatAbsensi{
		Mulai:    c.Query("mulai"),
		Sampai:   c.Query("sampai"),
		Kegiatan: c.Query("kegiatan"),
		GuruID:   int64(c.QueryInt("guru")),
		Cari:     c.Query("q"),
		Status:   c.Query("status"),
	}
	lap, err := h.svc.LaporanRiwayatAbsensi(filter, time.Now().In(services.ZonaWaktuRiayah))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	buf, err := riwayatAbsensiExcel(lap)
	if err != nil {
		return err
	}
	nama := "riwayat-absensi-guru"
	if lap.Filter.Mulai != "" {
		nama += "-" + lap.Filter.Mulai
	}
	nama += "-sd-" + lap.Filter.Sampai + ".xlsx"
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", nama))
	return c.Send(buf)
}

var labelKegiatanRiwayat = map[string]string{"": "Semua", "pembinaan": "Pembinaan", "rapat": "Rapat", "kunjungan": "Kunjungan"}
var labelStatusRiwayat = map[string]string{"": "Semua", "hadir": "Hadir", "izin": "Izin", "sakit": "Sakit", "alfa": "Alfa", "terlaksana": "Terlaksana", "dijadwalkan": "Dijadwalkan", "batal": "Batal"}

func persenSel(p float64) any {
	if p < 0 {
		return "–"
	}
	return p / 100
}

func bulanLabel(ym string) string {
	t, err := time.Parse("2006-01", ym)
	if err != nil {
		return ym
	}
	bulan := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	return fmt.Sprintf("%s %d", bulan[t.Month()-1], t.Year())
}

func riwayatAbsensiExcel(lap *services.LaporanRiwayatAbsensi) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	judul, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	label, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	header, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}})
	persen, _ := f.NewStyle(&excelize.Style{NumFmt: 10, Alignment: &excelize.Alignment{Horizontal: "right"}}) // 0.00%
	alfaMerah, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "C00000", Bold: true}})

	fl := lap.Filter
	periode := "Semua s.d. " + fl.Sampai
	if fl.Mulai != "" {
		periode = fl.Mulai + " s.d. " + fl.Sampai
	}
	guru := "Semua"
	if fl.GuruID != 0 {
		guru = fmt.Sprintf("ID %d", fl.GuruID)
		if len(lap.PerGuru) > 0 {
			guru = lap.PerGuru[0].Nama
		}
	}

	// --- Sheet 1: Ringkasan ---
	s := "Ringkasan"
	f.SetSheetName(f.GetSheetName(0), s)
	f.SetCellValue(s, "A1", "Riwayat Absensi Guru")
	f.SetCellStyle(s, "A1", "A1", judul)
	r := lap.Ringkasan
	rows := [][]any{
		{"Periode", periode},
		{"Kegiatan", labelKegiatanRiwayat[fl.Kegiatan]},
		{"Guru", guru},
		{"Pencarian", fl.Cari},
		{"Status (sheet Detail)", labelStatusRiwayat[fl.Status]},
		{},
		{"Tingkat kehadiran", persenSel(r.Persen)},
		{"Catatan absen", r.Total},
		{"Hadir", r.Hadir},
		{"Izin", r.Izin},
		{"Sakit", r.Sakit},
		{"Alfa", r.Alfa},
		{"Sesi tercatat", lap.Sesi},
		{"  Pembinaan", lap.SesiPembinaan},
		{"  Rapat", lap.Sesi - lap.SesiPembinaan},
		{"Kunjungan kelas terlaksana", lap.KunjunganTerlaksana},
		{"Kunjungan kelas dijadwalkan", lap.Kunjungan},
		{},
		{"Dibuat", time.Now().In(services.ZonaWaktuRiayah).Format("2006-01-02 15:04") + " WIB"},
	}
	for i, row := range rows {
		f.SetSheetRow(s, fmt.Sprintf("A%d", i+3), &row)
		if len(row) > 0 {
			f.SetCellStyle(s, fmt.Sprintf("A%d", i+3), fmt.Sprintf("A%d", i+3), label)
		}
	}
	f.SetCellStyle(s, "B9", "B9", persen)
	f.SetColWidth(s, "A", "A", 30)
	f.SetColWidth(s, "B", "B", 32)

	// --- Sheet 2: Rekap per Guru ---
	s = "Rekap per Guru"
	f.NewSheet(s)
	hdr := []any{"No.", "Nama Guru", "Catatan", "Hadir", "Izin", "Sakit", "Alfa", "% Hadir", "Kunjungan Terlaksana", "Kunjungan Dijadwalkan"}
	f.SetSheetRow(s, "A1", &hdr)
	f.SetCellStyle(s, "A1", "J1", header)
	for i, g := range lap.PerGuru {
		n := i + 2
		row := []any{i + 1, g.Nama, g.Rekap.Total, g.Rekap.Hadir, g.Rekap.Izin, g.Rekap.Sakit, g.Rekap.Alfa, persenSel(g.Rekap.Persen), g.KunjunganTerlaksana, g.Kunjungan}
		f.SetSheetRow(s, fmt.Sprintf("A%d", n), &row)
		f.SetCellStyle(s, fmt.Sprintf("H%d", n), fmt.Sprintf("H%d", n), persen)
		if g.Rekap.Alfa > 0 {
			f.SetCellStyle(s, fmt.Sprintf("G%d", n), fmt.Sprintf("G%d", n), alfaMerah)
		}
	}
	f.SetColWidth(s, "A", "A", 6)
	f.SetColWidth(s, "B", "B", 30)
	f.SetColWidth(s, "C", "H", 10)
	f.SetColWidth(s, "I", "J", 14)
	f.SetPanes(s, &excelize.Panes{Freeze: true, Split: true, YSplit: 1, XSplit: 2, TopLeftCell: "C2", ActivePane: "bottomRight"})

	// --- Sheet 3: Guru × Bulan (% hadir dan alfa per bulan) ---
	s = "Guru x Bulan"
	f.NewSheet(s)
	f.SetCellValue(s, "A1", "Nama Guru")
	f.MergeCell(s, "A1", "A2")
	for i, ym := range lap.Bulan {
		c1, _ := excelize.CoordinatesToCellName(2+i*2, 1)
		c2, _ := excelize.CoordinatesToCellName(3+i*2, 1)
		f.SetCellValue(s, c1, bulanLabel(ym))
		f.MergeCell(s, c1, c2)
		h1, _ := excelize.CoordinatesToCellName(2+i*2, 2)
		h2, _ := excelize.CoordinatesToCellName(3+i*2, 2)
		f.SetCellValue(s, h1, "% Hadir")
		f.SetCellValue(s, h2, "Alfa")
	}
	akhir, _ := excelize.CoordinatesToCellName(1+len(lap.Bulan)*2, 2)
	f.SetCellStyle(s, "A1", akhir, header)
	n := 3
	for _, g := range lap.PerGuru {
		per := lap.Matriks[g.GuruID]
		if len(per) == 0 {
			continue // guru tanpa catatan absen (hanya kunjungan)
		}
		f.SetCellValue(s, fmt.Sprintf("A%d", n), g.Nama)
		for i, ym := range lap.Bulan {
			m, ok := per[ym]
			if !ok {
				continue
			}
			c1, _ := excelize.CoordinatesToCellName(2+i*2, n)
			c2, _ := excelize.CoordinatesToCellName(3+i*2, n)
			f.SetCellValue(s, c1, persenSel(m.Persen))
			f.SetCellStyle(s, c1, c1, persen)
			f.SetCellValue(s, c2, m.Alfa)
			if m.Alfa > 0 {
				f.SetCellStyle(s, c2, c2, alfaMerah)
			}
		}
		n++
	}
	f.SetColWidth(s, "A", "A", 30)
	if len(lap.Bulan) > 0 {
		colAkhir, _ := excelize.ColumnNumberToName(1 + len(lap.Bulan)*2)
		f.SetColWidth(s, "B", colAkhir, 10)
	}
	f.SetPanes(s, &excelize.Panes{Freeze: true, Split: true, YSplit: 2, XSplit: 1, TopLeftCell: "B3", ActivePane: "bottomRight"})

	// --- Sheet 4: Detail ---
	s = "Detail"
	f.NewSheet(s)
	hdr = []any{"No.", "Tanggal", "Nama Guru", "Kegiatan", "Judul / Kelas", "Status", "Jam Masuk", "Keterangan", "Alasan"}
	f.SetSheetRow(s, "A1", &hdr)
	f.SetCellStyle(s, "A1", "I1", header)
	for i, d := range lap.Detail {
		row := []any{i + 1, d.Tanggal, d.GuruNama, labelKegiatanRiwayat[d.Kegiatan], d.Judul, labelStatusRiwayat[services.StatusRiwayat(d)], d.JamMasuk, d.Keterangan, d.Alasan}
		f.SetSheetRow(s, fmt.Sprintf("A%d", i+2), &row)
	}
	f.SetColWidth(s, "A", "A", 6)
	f.SetColWidth(s, "B", "B", 12)
	f.SetColWidth(s, "C", "C", 28)
	f.SetColWidth(s, "D", "D", 12)
	f.SetColWidth(s, "E", "E", 32)
	f.SetColWidth(s, "F", "G", 12)
	f.SetColWidth(s, "H", "I", 36)
	f.SetPanes(s, &excelize.Panes{Freeze: true, Split: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	if len(lap.Detail) > 0 {
		f.AutoFilter(s, fmt.Sprintf("A1:I%d", len(lap.Detail)+1), nil)
	}

	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
