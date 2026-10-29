package handlers

import (
    "errors"
    "strconv"

    "github.com/gofiber/fiber/v2"
    "github.com/maulanashalihin/laju-go/app/services"
    "github.com/maulanashalihin/laju-go/app/session"
)

type OnboardingHandler struct {
    service *services.OnboardingService
    store *session.Store
    inertia *services.InertiaService
}

func NewOnboardingHandler(s *services.OnboardingService, store *session.Store, inertia *services.InertiaService) *OnboardingHandler {
    return &OnboardingHandler{service:s, store:store, inertia:inertia}
}

// List retains PR 1's read-only JSON endpoint for existing consumers.
func (h *OnboardingHandler) List(c *fiber.Ctx) error {
    rows, err := h.service.List(c.UserContext())
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Gagal mengambil data onboarding"})
    }
    return c.JSON(fiber.Map{"items":rows})
}

func (h *OnboardingHandler) Index(c *fiber.Ctx) error {
    sess, _ := h.store.Get(c)
    user := sessionUser(sess)
    page, err := strconv.ParseInt(c.Query("page", "1"), 10, 64)
    if err != nil || page < 1 { page = 1 }
    board, err := h.service.Board(c.UserContext(), page)
    if err != nil { return c.Status(fiber.StatusInternalServerError).SendString("Gagal memuat onboarding") }
    return h.inertia.Render(c, "app/OnboardingKanban", fiber.Map{"user":user, "board":board})
}

func onboardingCaseID(c *fiber.Ctx) (int64, error) {
    id, err := strconv.ParseInt(c.Params("id"), 10, 64)
    if err != nil || id <= 0 { return 0, services.ErrOnboardingNotFound }
    return id, nil
}

func onboardingHTTPError(c *fiber.Ctx, err error) error {
    switch {
    case errors.Is(err, services.ErrOnboardingNotFound):
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error":err.Error()})
    case errors.Is(err, services.ErrOnboardingInvalid):
        return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error":err.Error()})
    case errors.Is(err, services.ErrOnboardingConflict):
        return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error":err.Error()})
    default:
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Gagal memperbarui onboarding"})
    }
}

func (h *OnboardingHandler) UpdatePIC(c *fiber.Ctx) error {
    id, err := onboardingCaseID(c)
    if err != nil { return onboardingHTTPError(c,err) }
    var request struct { PICUserID int64 `json:"pic_user_id"` }
    if err := c.BodyParser(&request); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Input PIC tidak valid"})
    }
    if err := h.service.AssignPIC(c.UserContext(),id,request.PICUserID,c.Locals("user_id").(int64)); err != nil {
        return onboardingHTTPError(c,err)
    }
    h.store.Flash(c,"success","PIC onboarding berhasil diperbarui")
    return h.inertia.Redirect(c,"/app/onboarding")
}

func (h *OnboardingHandler) Move(c *fiber.Ctx) error {
    id, err := onboardingCaseID(c)
    if err != nil { return onboardingHTTPError(c,err) }
    var request struct {
        Stage string `json:"stage"`
        Reason string `json:"reason"`
        ActualStartedAt string `json:"actual_started_at"`
    }
    if err := c.BodyParser(&request); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Input perpindahan tidak valid"})
    }
    if err := h.service.Move(c.UserContext(),id,c.Locals("user_id").(int64),request.Stage,request.Reason,request.ActualStartedAt); err != nil {
        return onboardingHTTPError(c,err)
    }
    h.store.Flash(c,"success","Status onboarding berhasil diperbarui")
    return h.inertia.Redirect(c,"/app/onboarding")
}

func (h *OnboardingHandler) Activity(c *fiber.Ctx) error {
    id, err := onboardingCaseID(c)
    if err != nil { return onboardingHTTPError(c,err) }
    activity, err := h.service.Activity(c.UserContext(),id)
    if err != nil { return onboardingHTTPError(c,err) }
    return c.JSON(fiber.Map{"items":activity})
}
