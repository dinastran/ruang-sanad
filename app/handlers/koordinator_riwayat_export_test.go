package handlers

import (
	"bytes"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestRiwayatAbsensiExcelSheets(t *testing.T) {
	rows := []models.RiwayatAbsensiGuru{
		{GuruID: 1, GuruNama: "Ahmad", Kegiatan: "pembinaan", KegiatanID: 1, Tanggal: "2026-10-05", Judul: "Tahsin", Hadir: true, Kehadiran: "hadir"},
		{GuruID: 2, GuruNama: "Budi", Kegiatan: "rapat", KegiatanID: 2, Tanggal: "2026-11-03", Judul: "=SUM(A1)", Kehadiran: "alfa"},
	}
	f, err := services.NormalisasiFilterRiwayat(services.FilterRiwayatAbsensi{Mulai: "2026-10-01", Sampai: "2026-11-30"}, time.Now())
	require.NoError(t, err)
	buf, err := riwayatAbsensiExcel(services.AnalisisRiwayatAbsensi(rows, f))
	require.NoError(t, err)

	x, err := excelize.OpenReader(bytes.NewReader(buf))
	require.NoError(t, err)
	require.Equal(t, []string{"Ringkasan", "Rekap per Guru", "Guru x Bulan", "Detail"}, x.GetSheetList())

	v, _ := x.GetCellValue("Ringkasan", "B3")
	require.Equal(t, "2026-10-01 s.d. 2026-11-30", v)
	v, _ = x.GetCellValue("Rekap per Guru", "B2")
	require.Equal(t, "Budi", v, "alfa terbanyak di atas")
	v, _ = x.GetCellValue("Guru x Bulan", "B1")
	require.Equal(t, "Okt 2026", v)
	v, _ = x.GetCellValue("Guru x Bulan", "D1")
	require.Equal(t, "Nov 2026", v)

	// Teks dari data disimpan sebagai string, bukan formula.
	formula, _ := x.GetCellFormula("Detail", "E3")
	require.Empty(t, formula)
	v, _ = x.GetCellValue("Detail", "E3")
	require.Equal(t, "=SUM(A1)", v)
	v, _ = x.GetCellValue("Detail", "F3")
	require.Equal(t, "Alfa", v)
}
