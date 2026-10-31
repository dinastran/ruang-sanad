package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

func planningWorkflowError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrPlanWorkflowForbidden):
		return c.SendStatus(fiber.StatusForbidden)
	case errors.Is(err, services.ErrPlanWorkflowNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perencanaan tidak ditemukan"})
	case errors.Is(err, services.ErrPlanWorkflowConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Rencana atau kandidat berubah. Muat ulang sebelum melanjutkan."})
	case errors.Is(err, services.ErrPlanWorkflowInvalid):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses perencanaan kelas"})
	}
}

func (h *OnboardingHandler) PlanWorkflowList(c *fiber.Ctx) error {
	if h.planningWorkflow == nil {
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	actor, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.SendStatus(fiber.StatusForbidden)
	}
	items, err := h.planningWorkflow.List(c.UserContext(), actor)
	if err != nil {
		return planningWorkflowError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *OnboardingHandler) PlanWorkflowCreate(c *fiber.Ctx) error {
	if h.planningWorkflow == nil {
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	actor, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.SendStatus(fiber.StatusForbidden)
	}
	var req services.ClassPlanDraftInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Input draft tidak valid"})
	}
	id, err := h.planningWorkflow.Create(c.UserContext(), actor, req)
	if err != nil {
		return planningWorkflowError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "status": "draft", "version": 1})
}

func (h *OnboardingHandler) PlanWorkflowAction(c *fiber.Ctx) error {
	if h.planningWorkflow == nil {
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	actor, ok := c.Locals("user_id").(int64)
	if !ok {
		return c.SendStatus(fiber.StatusForbidden)
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	var req struct {
		Action  string `json:"action"`
		Version int64  `json:"version"`
		Reason  string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Aksi perencanaan tidak valid"})
	}
	if err := h.planningWorkflow.Action(c.UserContext(), id, req.Version, actor, req.Action, req.Reason); err != nil {
		return planningWorkflowError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
