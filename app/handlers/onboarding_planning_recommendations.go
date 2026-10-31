package handlers

import "github.com/gofiber/fiber/v2"

// PlanningRecommendations returns a read-only preview. No roster or class
// writes, teacher reservation, coordinator approval or publication occurs.
func (h *OnboardingHandler) PlanningRecommendations(c *fiber.Ctx) error {
	if h.planningMatching == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Rekomendasi belum tersedia"})
	}
	result, err := h.planningMatching.Recommend(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Belum bisa menghitung rekomendasi dari data terkini"})
	}
	return c.JSON(result)
}
