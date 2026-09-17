package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type RiayahHandler struct {
	guruService    *services.GuruService
	riayahService  *services.RiayahService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewRiayahHandler(guruService *services.GuruService, riayahService *services.RiayahService, store *session.Store, inertiaService *services.InertiaService) *RiayahHandler {
	return &RiayahHandler{
		guruService:    guruService,
		riayahService:  riayahService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *RiayahHandler) classViewer(c *fiber.Ctx, kelasID int64) (int64, *int64, error) {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	guruID, err := viewerGuruID(h.guruService, userID, sessionUser(sess))
	if err != nil {
		return 0, nil, err
	}
	if err := h.guruService.EnsureCanAccessClass(guruID, kelasID); err != nil {
		return 0, nil, err
	}
	return userID, guruID, nil
}

func (h *RiayahHandler) List(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)

	if _, _, err := h.classViewer(c, kelasID); err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.guruService.EnsureSantriInClass(kelasID, santriID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	list, err := h.riayahService.ListRiayahByTarget("santri", santriID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"riayah": list})
}

func (h *RiayahHandler) Create(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)

	userID, guruID, err := h.classViewer(c, kelasID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.guruService.EnsureSantriInClass(kelasID, santriID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.RiayahInput
	if err := c.BodyParser(&req); err != nil || req.Catatan == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Catatan wajib diisi"})
	}
	req.TargetType = "santri"
	req.TargetID = santriID

	if guruID == nil {
		_, err = h.riayahService.CreateUserRiayah(userID, req)
	} else {
		_, err = h.riayahService.CreateRiayah(*guruID, userID, req)
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	list, err := h.riayahService.ListRiayahByTarget("santri", santriID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"riayah": list})
}

func (h *RiayahHandler) Update(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)
	riayahID, _ := strconv.ParseInt(c.Params("rid"), 10, 64)

	_, guruID, err := h.classViewer(c, kelasID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.guruService.EnsureSantriInClass(kelasID, santriID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	if guruID == nil {
		err = h.riayahService.EnsureRiayahSantriTarget(riayahID, santriID)
	} else {
		err = h.riayahService.EnsureRiayahTarget(riayahID, *guruID, santriID)
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	var req struct {
		Catatan string `json:"catatan"`
	}
	if err := c.BodyParser(&req); err != nil || req.Catatan == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Catatan wajib diisi"})
	}

	if guruID == nil {
		err = h.riayahService.UpdateRiayahByID(riayahID, req.Catatan)
	} else {
		err = h.riayahService.UpdateRiayah(riayahID, *guruID, req.Catatan)
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *RiayahHandler) Delete(c *fiber.Ctx) error {
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)
	riayahID, _ := strconv.ParseInt(c.Params("rid"), 10, 64)

	_, guruID, err := h.classViewer(c, kelasID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.guruService.EnsureSantriInClass(kelasID, santriID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	if guruID == nil {
		err = h.riayahService.EnsureRiayahSantriTarget(riayahID, santriID)
	} else {
		err = h.riayahService.EnsureRiayahTarget(riayahID, *guruID, santriID)
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	if guruID == nil {
		err = h.riayahService.DeleteRiayahByID(riayahID)
	} else {
		err = h.riayahService.DeleteRiayah(riayahID, *guruID)
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *RiayahHandler) WALink(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	user := sessionUser(sess)
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	santriID, _ := strconv.ParseInt(c.Params("sid"), 10, 64)

	guruID, err := viewerGuruID(h.guruService, userID, user)
	if err != nil {
		h.store.Flash(c, "error", "Data guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app")
	}
	if err := h.guruService.EnsureCanAccessClass(guruID, kelasID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}

	_, santri, err := h.guruService.GetDetailKelasForViewer(guruID, kelasID)
	if err != nil {
		h.store.Flash(c, "error", "Kelas tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas")
	}

	var noWa, nama string
	for _, s := range santri {
		if s.ID == santriID {
			noWa = s.NoWa
			nama = s.Nama
			break
		}
	}
	if noWa == "" {
		h.store.Flash(c, "error", "Nomor WA santri tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/guru/kelas/"+c.Params("id"))
	}

	link := h.riayahService.GetLinkWA(noWa, "default", map[string]string{"nama": nama})
	return h.inertiaService.Location(c, link)
}

func (h *RiayahHandler) BroadcastWA(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := toInt64(sess.Get("user_id"))
	user := sessionUser(sess)
	kelasID, _ := strconv.ParseInt(c.Params("id"), 10, 64)

	guruID, err := viewerGuruID(h.guruService, userID, user)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Guru not found"})
	}
	if err := h.guruService.EnsureCanAccessClass(guruID, kelasID); err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	links, err := h.riayahService.GetBroadcastLinks(kelasID, "default")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(links)
}
