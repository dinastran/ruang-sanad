package handlers

import (
	"crypto/subtle"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

const maxWebhookBody = 64 << 10

// XenditWebhookHandler receives Xendit payment session webhooks. Xendit
// signs nothing; it sends the dashboard's callback token in
// x-callback-token, compared here in constant time.
type XenditWebhookHandler struct {
	reg   *services.RegistrationService
	token string
}

func NewXenditWebhookHandler(reg *services.RegistrationService, callbackToken string) *XenditWebhookHandler {
	return &XenditWebhookHandler{reg: reg, token: callbackToken}
}

func (h *XenditWebhookHandler) Handle(c *fiber.Ctx) error {
	if h.token == "" {
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	got := c.Get("x-callback-token")
	if subtle.ConstantTimeCompare([]byte(got), []byte(h.token)) != 1 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	body := c.Body()
	if len(body) > maxWebhookBody {
		return c.SendStatus(fiber.StatusRequestEntityTooLarge)
	}
	if err := h.reg.HandleXenditWebhook(c.UserContext(), append([]byte(nil), body...)); err != nil {
		if errors.Is(err, services.ErrWebhookTidakTerbaca) {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		slog.Error("xendit webhook failed", "error", err)
		return c.SendStatus(fiber.StatusInternalServerError) // Xendit retries
	}
	return c.SendStatus(fiber.StatusOK)
}
