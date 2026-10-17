package handlers

import (
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// UjrohKomponenHandler manages extra ujroh components (kuota, kesehatan, MC,
// Qiro'ah, ...): the master list, recurring amounts per guru, and per-month
// entries.
type UjrohKomponenHandler struct {
	ujroh          *services.UjrohService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewUjrohKomponenHandler(ujroh *services.UjrohService, store *session.Store, inertiaService *services.InertiaService) *UjrohKomponenHandler {
	return &UjrohKomponenHandler{ujroh: ujroh, store: store, inertiaService: inertiaService}
}

func (h *UjrohKomponenHandler) Create(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	var req models.SimpanUjrohKomponenRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", "Data komponen tidak valid")
	}
	if err := h.ujroh.CreateKomponen(req, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Komponen ujroh ditambahkan")
}

func (h *UjrohKomponenHandler) Update(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.SimpanUjrohKomponenRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", "Data komponen tidak valid")
	}
	if err := h.ujroh.UpdateKomponen(id, req, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Komponen ujroh diperbarui")
}

// SetGuru sets a guru's recurring amount for a tetap component (null removes it).
func (h *UjrohKomponenHandler) SetGuru(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	guruID, _ := strconv.ParseInt(c.Params("guruId"), 10, 64)
	var req models.SetUjrohKomponenGuruRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", "Data nominal tidak valid")
	}
	if err := h.ujroh.SetKomponenGuru(id, guruID, req.Nominal, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Penerima komponen diperbarui")
}

// SetPenerima saves the recurring amounts of many guru at once.
func (h *UjrohKomponenHandler) SetPenerima(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.SetUjrohKomponenPenerimaRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", "Data penerima tidak valid")
	}
	if err := h.ujroh.SetKomponenPenerima(id, req.Penerima, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, c.Query("bulan"), "success", "Penerima komponen disimpan")
}

func (h *UjrohKomponenHandler) SetBulan(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	bulan := c.Params("bulan")
	guruID, _ := strconv.ParseInt(c.Params("guruId"), 10, 64)
	komponenID, _ := strconv.ParseInt(c.Params("komponenId"), 10, 64)
	var req models.SetUjrohKomponenBulanRequest
	if err := c.BodyParser(&req); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, bulan, "error", "Data komponen tidak valid")
	}
	if err := h.ujroh.SetKomponenBulan(bulan, guruID, komponenID, req, toInt64(sess.Get("user_id"))); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, bulan, "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, bulan, "success", "Komponen bulan ini disimpan")
}

func (h *UjrohKomponenHandler) DeleteBulan(c *fiber.Ctx) error {
	bulan := c.Params("bulan")
	guruID, _ := strconv.ParseInt(c.Params("guruId"), 10, 64)
	komponenID, _ := strconv.ParseInt(c.Params("komponenId"), 10, 64)
	if err := h.ujroh.HapusKomponenBulan(bulan, guruID, komponenID); err != nil {
		return ujrohBack(c, h.store, h.inertiaService, bulan, "error", err.Error())
	}
	return ujrohBack(c, h.store, h.inertiaService, bulan, "success", "Komponen bulan ini dikembalikan")
}

// ujrohBack flashes message and returns to the Ujroh page for bulan.
func ujrohBack(c *fiber.Ctx, store *session.Store, inertia *services.InertiaService, bulan, kind, message string) error {
	store.Flash(c, kind, message)
	target := "/app/keuangan/ujroh"
	if bulan != "" {
		target += "?bulan=" + url.QueryEscape(bulan)
	}
	return inertia.Redirect(c, target)
}
