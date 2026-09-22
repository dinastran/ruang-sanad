package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/xuri/excelize/v2"
)

// KoordinatorAttendanceExportHandler downloads saved rapat and pembinaan attendance reports.
type KoordinatorAttendanceExportHandler struct {
	svc *services.KoordinatorService
}

func NewKoordinatorAttendanceExportHandler(svc *services.KoordinatorService) *KoordinatorAttendanceExportHandler {
	return &KoordinatorAttendanceExportHandler{svc: svc}
}

func (h *KoordinatorAttendanceExportHandler) PembinaanExcel(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	laporan, err := h.svc.GetLaporanPembinaanAbsensi(id)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return downloadLaporanAbsensiExcel(c, laporan, "absensi-pembinaan")
}

func (h *KoordinatorAttendanceExportHandler) RapatExcel(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	laporan, err := h.svc.GetLaporanRapatAbsensi(id)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return downloadLaporanAbsensiExcel(c, laporan, "absensi-rapat")
}

func downloadLaporanAbsensiExcel(c *fiber.Ctx, laporan *models.LaporanAbsensi, filenamePrefix string) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", laporan.JenisKegiatan)
	f.MergeCell(sheet, "A1", "F1")
	f.SetCellValue(sheet, "A2", "Kegiatan")
	f.SetCellValue(sheet, "B2", laporan.Judul)
	f.SetCellValue(sheet, "A3", "Tanggal")
	f.SetCellValue(sheet, "B3", laporan.Tanggal)
	f.SetCellValue(sheet, "A4", "Keterangan")
	f.SetCellValue(sheet, "B4", laporan.Keterangan)
	f.SetCellValue(sheet, "A5", "Total Guru")
	f.SetCellValue(sheet, "B5", laporan.Total)
	f.SetCellValue(sheet, "C5", "Hadir")
	f.SetCellValue(sheet, "D5", laporan.Hadir)
	f.SetCellValue(sheet, "E5", "Tidak Hadir")
	f.SetCellValue(sheet, "F5", laporan.TidakHadir)

	headers := []string{"No.", "Nama Guru", "Status Kehadiran", "Jam Masuk", "Alasan Tidak Hadir", "Keterangan"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 7)
		f.SetCellValue(sheet, cell, header)
	}
	for i, row := range laporan.Absen {
		excelRow := i + 8
		status := "Tidak Hadir"
		if row.Hadir {
			status = "Hadir"
		}
		f.SetSheetRow(sheet, fmt.Sprintf("A%d", excelRow), &[]interface{}{i + 1, row.Nama, status, row.JamMasuk, row.Alasan, row.Keterangan})
	}
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}, Alignment: &excelize.Alignment{Horizontal: "center"}})
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}})
	f.SetCellStyle(sheet, "A1", "F1", titleStyle)
	f.SetCellStyle(sheet, "A7", "F7", headerStyle)
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 28)
	f.SetColWidth(sheet, "C", "C", 20)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetColWidth(sheet, "E", "E", 28)
	f.SetColWidth(sheet, "F", "F", 40)
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: true, XSplit: 0, YSplit: 7, TopLeftCell: "A8", ActivePane: "bottomLeft"})

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s-%s.xlsx", filenamePrefix, laporan.Tanggal))
	return c.Send(buffer.Bytes())
}
