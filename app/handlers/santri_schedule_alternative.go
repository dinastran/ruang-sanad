package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

func (h *SantriHandler) UpdateScheduleAlternative(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 || h.scheduleAlternatives == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	actor, ok := c.Locals("user_id").(int64)
	if !ok || actor <= 0 {
		return c.SendStatus(fiber.StatusForbidden)
	}
	var req services.SantriScheduleAlternativeInput
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data alternatif jadwal tidak valid")
		return h.inertiaService.Redirect(c, "/app/santri/"+strconv.FormatInt(id, 10))
	}
	err = h.scheduleAlternatives.Save(c.UserContext(), id, actor, req)
	if err != nil {
		message := "Gagal menyimpan alternatif jadwal"
		if errors.Is(err, services.ErrAlternativeInvalid) || errors.Is(err, services.ErrAlternativeUnavailable) {
			message = err.Error()
		}
		h.store.Flash(c, "error", message)
	} else {
		h.store.Flash(c, "success", "Status alternatif jadwal tersimpan. Jadwal utama santri tidak berubah.")
	}
	return h.inertiaService.Redirect(c, "/app/santri/"+strconv.FormatInt(id, 10))
}
