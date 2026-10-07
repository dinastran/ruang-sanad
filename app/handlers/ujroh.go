package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/xuri/excelize/v2"
)

type UjrohHandler struct {
	ujroh          *services.UjrohService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewUjrohHandler(ujroh *services.UjrohService, store *session.Store, inertiaService *services.InertiaService) *UjrohHandler {
	return &UjrohHandler{ujroh: ujroh, store: store, inertiaService: inertiaService}
}

func (h *UjrohHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan, err := h.ujroh.NormalizeBulan(c.Query("bulan"))
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/keuangan/ujroh")
	}
	rekap, err := h.ujroh.GetRekap(bulan)
	if err != nil {
		return err
	}
	tarif, err := h.ujroh.GetTarif()
	if err != nil {
		return err
	}
	tarifGuru, err := h.ujroh.ListTarifGuru()
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "keuangan/Ujroh", fiber.Map{
		"user":       sessionUser(sess),
		"rekap":      rekap,
		"tarif":      tarif,
		"tarif_guru": tarifGuru,
	})
}

func (h *UjrohHandler) Kunci(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := c.Params("bulan")
	if err := h.ujroh.Kunci(bulan, toInt64(sess.Get("user_id"))); err != nil {
		return h.back(c, bulan, "error", err.Error())
	}
	return h.back(c, bulan, "success", "Ujroh bulan "+bulan+" dikunci")
}

func (h *UjrohHandler) Buka(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := c.Params("bulan")
	if err := h.ujroh.Buka(bulan, toInt64(sess.Get("user_id"))); err != nil {
		return h.back(c, bulan, "error", err.Error())
	}
	return h.back(c, bulan, "success", "Kunci ujroh bulan "+bulan+" dibuka")
}

func (h *UjrohHandler) TandaiDibayar(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := c.Params("bulan")
	guruID, _ := strconv.ParseInt(c.Params("guruId"), 10, 64)
	var req models.TandaiUjrohDibayarRequest
	if err := c.BodyParser(&req); err != nil {
		return h.back(c, bulan, "error", "Data pembayaran tidak valid")
	}
	if err := h.ujroh.TandaiDibayar(bulan, guruID, toInt64(sess.Get("user_id")), req); err != nil {
		return h.back(c, bulan, "error", err.Error())
	}
	return h.back(c, bulan, "success", "Ujroh ditandai dibayar")
}

func (h *UjrohHandler) BatalDibayar(c *fiber.Ctx) error {
	bulan := c.Params("bulan")
	guruID, _ := strconv.ParseInt(c.Params("guruId"), 10, 64)
	if err := h.ujroh.BatalDibayar(bulan, guruID); err != nil {
		return h.back(c, bulan, "error", err.Error())
	}
	return h.back(c, bulan, "success", "Tanda pembayaran dibatalkan")
}

func (h *UjrohHandler) UpdateTarif(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := c.Query("bulan")
	var req models.UpdateUjrohTarifRequest
	if err := c.BodyParser(&req); err != nil {
		return h.back(c, bulan, "error", "Data tarif tidak valid")
	}
	if err := h.ujroh.UpdateTarif(req, toInt64(sess.Get("user_id"))); err != nil {
		return h.back(c, bulan, "error", err.Error())
	}
	return h.back(c, bulan, "success", "Tarif ujroh diperbarui")
}

func (h *UjrohHandler) UpdateTarifGuru(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := c.Query("bulan")
	guruID, _ := strconv.ParseInt(c.Params("guruId"), 10, 64)
	var req models.UpdateUjrohTarifGuruRequest
	if err := c.BodyParser(&req); err != nil {
		return h.back(c, bulan, "error", "Data tarif guru tidak valid")
	}
	if err := h.ujroh.SetTarifGuru(guruID, req.Nominal, toInt64(sess.Get("user_id"))); err != nil {
		return h.back(c, bulan, "error", err.Error())
	}
	return h.back(c, bulan, "success", "Tarif khusus guru diperbarui")
}

func (h *UjrohHandler) Export(c *fiber.Ctx) error {
	rekap, err := h.ujroh.GetRekap(c.Query("bulan"))
	if err != nil {
		if errors.Is(err, services.ErrUjrohBulanTidakValid) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return err
	}

	f := excelize.NewFile()
	defer f.Close()
	ringkasan := f.GetSheetName(0)
	_ = f.SetSheetName(ringkasan, "Ringkasan")
	ringkasan = "Ringkasan"
	status := "Belum dikunci (angka masih bisa berubah)"
	if rekap.Terkunci {
		status = "Dikunci " + rekap.DikunciAt
	}
	f.SetCellValue(ringkasan, "A1", "Ujroh Guru "+rekap.Bulan)
	f.SetCellValue(ringkasan, "A2", "Status")
	f.SetCellValue(ringkasan, "B2", status)
	headers := []string{"No.", "Nama Guru", "Status Guru", "Tarif", "Pertemuan", "Badal", "Total", "Dibayar", "Tanggal Bayar", "Catatan"}
	f.SetSheetRow(ringkasan, "A4", &headers)
	for i, g := range rekap.Guru {
		dibayar := "Belum"
		if g.Dibayar {
			dibayar = "Sudah"
		}
		f.SetSheetRow(ringkasan, fmt.Sprintf("A%d", i+5), &[]interface{}{i + 1, g.GuruNama, statusGuruLabel(g.GuruStatus), g.Tarif, g.JumlahPertemuan, g.JumlahBadal, g.Total, dibayar, g.DibayarAt, g.CatatanBayar})
	}
	totalRow := len(rekap.Guru) + 5
	f.SetCellValue(ringkasan, fmt.Sprintf("B%d", totalRow), "Total")
	f.SetCellValue(ringkasan, fmt.Sprintf("E%d", totalRow), rekap.Ringkasan.JumlahPertemuan)
	f.SetCellValue(ringkasan, fmt.Sprintf("G%d", totalRow), rekap.Ringkasan.TotalUjroh)

	rincian := "Rincian"
	_, _ = f.NewSheet(rincian)
	detailHeaders := []string{"Nama Guru", "Tanggal", "Kelas", "Pertemuan Ke", "Badal", "Tarif"}
	f.SetSheetRow(rincian, "A1", &detailHeaders)
	row := 2
	for _, g := range rekap.Guru {
		for _, p := range g.Pertemuan {
			badal := ""
			if p.IsBadal {
				badal = "Ya"
			}
			f.SetSheetRow(rincian, fmt.Sprintf("A%d", row), &[]interface{}{g.GuruNama, p.Tanggal, p.KelasNama, p.PertemuanKe, badal, p.Tarif})
			row++
		}
	}

	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	money, _ := f.NewStyle(&excelize.Style{NumFmt: 3})
	f.SetCellStyle(ringkasan, "A1", "A1", bold)
	f.SetCellStyle(ringkasan, "A4", "J4", bold)
	f.SetCellStyle(ringkasan, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("J%d", totalRow), bold)
	f.SetCellStyle(ringkasan, "D5", fmt.Sprintf("D%d", totalRow), money)
	f.SetCellStyle(ringkasan, "G5", fmt.Sprintf("G%d", totalRow), money)
	f.SetCellStyle(rincian, "A1", "F1", bold)
	f.SetCellStyle(rincian, "F2", fmt.Sprintf("F%d", row), money)
	f.SetColWidth(ringkasan, "B", "B", 30)
	f.SetColWidth(ringkasan, "C", "J", 14)
	f.SetColWidth(rincian, "A", "A", 30)
	f.SetColWidth(rincian, "C", "C", 45)

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=ujroh-guru-%s.xlsx", rekap.Bulan))
	return c.Send(buffer.Bytes())
}

func (h *UjrohHandler) back(c *fiber.Ctx, bulan, kind, message string) error {
	h.store.Flash(c, kind, message)
	target := "/app/keuangan/ujroh"
	if bulan != "" {
		target += "?bulan=" + url.QueryEscape(bulan)
	}
	return h.inertiaService.Redirect(c, target)
}

func statusGuruLabel(status string) string {
	switch status {
	case "tetap":
		return "Tetap"
	case "part_time":
		return "Part Time"
	}
	return status
}
