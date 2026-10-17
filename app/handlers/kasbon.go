package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// KasbonHandler records guru cash advances that are deducted from ujroh.
type KasbonHandler struct {
	ujroh          *services.UjrohService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewKasbonHandler(ujroh *services.UjrohService, store *session.Store, inertiaService *services.InertiaService) *KasbonHandler {
	return &KasbonHandler{ujroh: ujroh, store: store, inertiaService: inertiaService}
}

func (h *KasbonHandler) Create(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	var req models.CreateKasbonRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", "Data kasbon tidak valid")
	}
	if err := h.ujroh.CreateKasbon(req, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Kasbon dicatat")
}

func (h *KasbonHandler) Update(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.UpdateKasbonRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", "Data kasbon tidak valid")
	}
	if err := h.ujroh.UpdateKasbon(id, req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Kasbon diperbarui")
}

func (h *KasbonHandler) Batal(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.ujroh.BatalkanKasbon(id, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Kasbon dibatalkan")
}
