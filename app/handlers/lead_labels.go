package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (h *LeadHandler) SaveCatalogLabel(c *fiber.Ctx) error {
	leadID, ok := leadIDParam(c)
	var req struct {
		Nama  string `json:"nama"`
		Warna string `json:"warna"`
	}
	var labelID int64
	var err error
	if c.Params("labelID") != "" {
		labelID, err = strconv.ParseInt(c.Params("labelID"), 10, 64)
		if labelID <= 0 {
			err = errors.New("label tidak valid")
		}
	}
	if !ok || err != nil || c.BodyParser(&req) != nil {
		return h.finish(c, leadID, errors.New("data label tidak valid"), "")
	}
	return h.finish(c, leadID, h.leads.SaveLabel(c.UserContext(), labelID, req.Nama, req.Warna), "Katalog label diperbarui untuk seluruh tim")
}

func (h *LeadHandler) DeleteCatalogLabel(c *fiber.Ctx) error {
	leadID, ok := leadIDParam(c)
	labelID, err := strconv.ParseInt(c.Params("labelID"), 10, 64)
	if !ok || err != nil || labelID <= 0 {
		return h.finish(c, leadID, errors.New("label tidak valid"), "")
	}
	return h.finish(c, leadID, h.leads.DeleteLabel(c.UserContext(), labelID), "Label dihapus dari katalog dan seluruh lead")
}
