package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

func (h *KoordinatorGuruDirectoryHandler) TeachingLevelsReplace(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 || h.teachingLevels == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	actor, ok := c.Locals("user_id").(int64)
	if !ok || actor <= 0 {
		return c.SendStatus(fiber.StatusForbidden)
	}
	var req struct {
		LevelCodes []string `json:"level_codes"`
	}
	if err := c.BodyParser(&req); err != nil || req.LevelCodes == nil {
		h.store.Flash(c, "error", "Daftar level mengajar tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
	}
	err = h.teachingLevels.Replace(c.UserContext(), id, actor, req.LevelCodes)
	if err != nil {
		message := "Gagal menyimpan izin mengajar level"
		if errors.Is(err, services.ErrTeachingLevelInvalid) || errors.Is(err, services.ErrTeachingLevelNotFound) {
			message = err.Error()
		}
		h.store.Flash(c, "error", message)
	} else {
		h.store.Flash(c, "success", "Izin mengajar level telah dikonfirmasi Koordinator Guru")
	}
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
}
