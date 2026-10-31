package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// PlanningReadiness is read-only and never creates classes, assigns teachers,
// changes student rosters, or publishes offered schedules.
func (h *OnboardingHandler) PlanningReadiness(c *fiber.Ctx) error {
	if h.planningReadiness == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Pemeriksaan kesiapan belum tersedia"})
	}
	report, err := h.planningReadiness.Report(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Belum bisa membaca kesiapan perencanaan kelas"})
	}
	return c.JSON(report)
}
