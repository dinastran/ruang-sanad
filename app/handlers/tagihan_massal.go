package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// TagihanMassalHandler handles bulk cancellation of tagihan and its undo.
type TagihanMassalHandler struct {
	tagihan        *services.TagihanService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewTagihanMassalHandler(tagihan *services.TagihanService, store *session.Store, inertiaService *services.InertiaService) *TagihanMassalHandler {
	return &TagihanMassalHandler{tagihan: tagihan, store: store, inertiaService: inertiaService}
}

// Preview returns JSON for fetch(): how many tagihan the filter would cancel.
func (h *TagihanMassalHandler) Preview(c *fiber.Ctx) error {
	var req models.BatalMassalPreviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "filter tidak valid"})
	}
	preview, err := h.tagihan.PreviewBatalMassal(services.TagihanFilterFromRequest(req.Filter))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(preview)
}

func (h *TagihanMassalHandler) Batal(c *fiber.Ctx) error {
	var req models.BatalMassalRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "data pembatalan massal tidak valid")
		return h.inertiaService.Back(c, "/app/keuangan/tagihan")
	}
	sess, _ := h.store.Get(c)
	result, err := h.tagihan.BatalMassal(services.TagihanFilterFromRequest(req.Filter), req.Alasan, toInt64(sess.Get("user_id")), req.Konfirmasi)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, "/app/keuangan/tagihan")
	}
	h.store.Flash(c, "success", fmt.Sprintf("%d tagihan dibatalkan. Bisa dipulihkan dari Riwayat batal massal.", result.Jumlah))
	return h.inertiaService.Back(c, "/app/keuangan/tagihan")
}

func (h *TagihanMassalHandler) Pulihkan(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	sess, _ := h.store.Get(c)
	restored, err := h.tagihan.PulihkanBatalMassal(id, toInt64(sess.Get("user_id")))
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Back(c, "/app/keuangan/tagihan")
	}
	h.store.Flash(c, "success", fmt.Sprintf("%d tagihan dipulihkan ke belum bayar", restored))
	return h.inertiaService.Back(c, "/app/keuangan/tagihan")
}
