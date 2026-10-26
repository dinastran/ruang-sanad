package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// JadwalPenawaranHandler serves /app/jadwal-penawaran: the class schedules the
// CS AI offers to calon santri, with quota per gender.
type JadwalPenawaranHandler struct {
	jadwal         *services.JadwalPenawaranService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewJadwalPenawaranHandler(jadwal *services.JadwalPenawaranService, store *session.Store, inertiaService *services.InertiaService) *JadwalPenawaranHandler {
	return &JadwalPenawaranHandler{jadwal: jadwal, store: store, inertiaService: inertiaService}
}

const jadwalPenawaranPath = "/app/jadwal-penawaran"

func (h *JadwalPenawaranHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	list, err := h.jadwal.List(c.UserContext())
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/JadwalPenawaran", fiber.Map{
		"user":   sessionUser(sess),
		"jadwal": list,
	})
}

func (h *JadwalPenawaranHandler) done(c *fiber.Ctx, err error, ok string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", ok)
	}
	return h.inertiaService.Redirect(c, jadwalPenawaranPath)
}

func jadwalID(c *fiber.Ctx) (int64, bool) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

func (h *JadwalPenawaranHandler) Store(c *fiber.Ctx) error {
	var req models.JadwalPenawaranRequest
	if err := c.BodyParser(&req); err != nil {
		return h.done(c, fiber.NewError(fiber.StatusBadRequest, "data jadwal tidak valid"), "")
	}
	return h.done(c, h.jadwal.Create(c.UserContext(), req), "Jadwal ditambahkan")
}

func (h *JadwalPenawaranHandler) Update(c *fiber.Ctx) error {
	id, ok := jadwalID(c)
	var req models.JadwalPenawaranRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.done(c, fiber.NewError(fiber.StatusBadRequest, "data jadwal tidak valid"), "")
	}
	return h.done(c, h.jadwal.Update(c.UserContext(), id, req), "Jadwal disimpan")
}

func (h *JadwalPenawaranHandler) ResetPeriode(c *fiber.Ctx) error {
	id, ok := jadwalID(c)
	if !ok {
		return h.done(c, services.ErrJadwalNotFound, "")
	}
	return h.done(c, h.jadwal.ResetPeriode(c.UserContext(), id), "Periode baru dimulai; hitungan kuota kembali dari nol")
}

func (h *JadwalPenawaranHandler) Delete(c *fiber.Ctx) error {
	id, ok := jadwalID(c)
	if !ok {
		return h.done(c, services.ErrJadwalNotFound, "")
	}
	return h.done(c, h.jadwal.Delete(c.UserContext(), id), "Jadwal dihapus")
}
