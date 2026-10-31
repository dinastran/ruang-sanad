package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

func guruAvailabilityIDs(c *fiber.Ctx, withSlot bool) (int64, int64, bool) {
	guruID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || guruID < 1 {
		return 0, 0, false
	}
	if !withSlot {
		return guruID, 0, true
	}
	slotID, err := strconv.ParseInt(c.Params("slotId"), 10, 64)
	return guruID, slotID, err == nil && slotID > 0
}

func (h *KoordinatorGuruDirectoryHandler) availabilityRedirect(c *fiber.Ctx, guruID int64, err error, success string) error {
	if err != nil {
		message := "Gagal menyimpan jadwal available guru"
		if errors.Is(err, services.ErrGuruAvailabilityInvalid) ||
			errors.Is(err, services.ErrGuruAvailabilityConflict) ||
			errors.Is(err, services.ErrGuruAvailabilityNotFound) {
			message = err.Error()
		}
		h.store.Flash(c, "error", message)
	} else {
		h.store.Flash(c, "success", success)
	}
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(guruID, 10))
}

func (h *KoordinatorGuruDirectoryHandler) AvailabilityCreate(c *fiber.Ctx) error {
	guruID, _, ok := guruAvailabilityIDs(c, false)
	if !ok || h.availability == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	var req services.GuruAvailabilityInput
	if err := c.BodyParser(&req); err != nil {
		return h.availabilityRedirect(c, guruID, services.ErrGuruAvailabilityInvalid, "")
	}
	actorID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.SendStatus(fiber.StatusForbidden)
	}
	err := h.availability.Create(c.UserContext(), guruID, actorID, req)
	return h.availabilityRedirect(c, guruID, err, "Slot available guru ditambahkan dan dikonfirmasi")
}

func (h *KoordinatorGuruDirectoryHandler) AvailabilityUpdate(c *fiber.Ctx) error {
	guruID, slotID, ok := guruAvailabilityIDs(c, true)
	if !ok || h.availability == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	var req services.GuruAvailabilityInput
	if err := c.BodyParser(&req); err != nil {
		return h.availabilityRedirect(c, guruID, services.ErrGuruAvailabilityInvalid, "")
	}
	actorID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.SendStatus(fiber.StatusForbidden)
	}
	err := h.availability.Update(c.UserContext(), guruID, slotID, actorID, req)
	return h.availabilityRedirect(c, guruID, err, "Slot available guru diperbarui dan dikonfirmasi")
}

func (h *KoordinatorGuruDirectoryHandler) AvailabilityDelete(c *fiber.Ctx) error {
	guruID, slotID, ok := guruAvailabilityIDs(c, true)
	if !ok || h.availability == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	actorID, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.SendStatus(fiber.StatusForbidden)
	}
	err := h.availability.Delete(c.UserContext(), guruID, slotID, actorID)
	return h.availabilityRedirect(c, guruID, err, "Slot available guru dihapus")
}
