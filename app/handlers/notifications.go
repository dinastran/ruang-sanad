package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type NotificationHandler struct {
	service        *services.NotificationService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewNotificationHandler(service *services.NotificationService, store *session.Store, inertiaService *services.InertiaService) *NotificationHandler {
	return &NotificationHandler{service: service, store: store, inertiaService: inertiaService}
}

func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	notificationID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if notificationID <= 0 {
		return fiber.ErrBadRequest
	}
	if err := h.service.MarkRead(userID, notificationID); err != nil {
		h.store.Flash(c, "error", "Gagal memperbarui notifikasi")
	}
	return h.inertiaService.Back(c, "/app/guru")
}
