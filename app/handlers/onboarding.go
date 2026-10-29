package handlers

import (
 "github.com/gofiber/fiber/v2"
 "github.com/maulanashalihin/laju-go/app/services"
)

type OnboardingHandler struct { service *services.OnboardingService }

func NewOnboardingHandler(s *services.OnboardingService) *OnboardingHandler {
 return &OnboardingHandler{service:s}
}

// List is read-only. Mutation endpoints arrive with validated moves in PR 2.
func (h *OnboardingHandler) List(c *fiber.Ctx) error {
 rows, err := h.service.List(c.UserContext())
 if err != nil { return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Gagal mengambil data onboarding"}) }
 return c.JSON(fiber.Map{"items":rows})
}
