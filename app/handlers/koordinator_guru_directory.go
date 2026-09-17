package handlers

import (
	"net/mail"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type KoordinatorGuruDirectoryHandler struct {
	guruService        *services.GuruService
	koordinatorService *services.KoordinatorService
	riayahService      *services.RiayahService
	store              *session.Store
	inertiaService     *services.InertiaService
}

func NewKoordinatorGuruDirectoryHandler(guruService *services.GuruService, koordinatorService *services.KoordinatorService, riayahService *services.RiayahService, store *session.Store, inertiaService *services.InertiaService) *KoordinatorGuruDirectoryHandler {
	return &KoordinatorGuruDirectoryHandler{guruService: guruService, koordinatorService: koordinatorService, riayahService: riayahService, store: store, inertiaService: inertiaService}
}

func (h *KoordinatorGuruDirectoryHandler) List(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	gurus, err := h.guruService.ListDirectory()
	if err != nil {
		h.store.Flash(c, "error", "Gagal memuat data guru")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru")
	}
	return h.inertiaService.Render(c, "koordinator/GuruList", fiber.Map{"user": user, "gurus": gurus})
}

func (h *KoordinatorGuruDirectoryHandler) Detail(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	guru, err := h.guruService.GetDirectoryByID(id)
	if err != nil {
		h.store.Flash(c, "error", "Guru tidak ditemukan")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	linkableUsers, _ := h.guruService.ListLinkableUsers()
	kompetensi, _ := h.koordinatorService.GetKompetensi(id)
	riayah, _ := h.riayahService.ListRiayahByTarget("guru", id)
	return h.inertiaService.Render(c, "koordinator/GuruDetail", fiber.Map{
		"user":          user,
		"guru":          guru,
		"linkableUsers": linkableUsers,
		"canLink":       user != nil && user.Role == "super_admin",
		"kompetensi":    kompetensi,
		"riayah":        riayah,
	})
}

func (h *KoordinatorGuruDirectoryHandler) Create(c *fiber.Ctx) error {
	var req models.CreateGuruDirectoryRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	req.Nama = strings.TrimSpace(req.Nama)
	req.Gelar = strings.TrimSpace(req.Gelar)
	req.NoWa = strings.TrimSpace(req.NoWa)
	req.Email = strings.TrimSpace(req.Email)
	req.Foto = strings.TrimSpace(req.Foto)
	if req.Nama == "" || (req.JenisKelamin != "L" && req.JenisKelamin != "P") || (req.Status != "tetap" && req.Status != "part_time") {
		h.store.Flash(c, "error", "Data guru tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			h.store.Flash(c, "error", "Format email tidak valid")
			return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
		}
	}
	id, err := h.guruService.CreateDirectory(req)
	if err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	h.store.Flash(c, "success", "Guru baru ditambahkan. Hubungkan ke akun login, lalu Super Admin menetapkan role Guru.")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
}

func (h *KoordinatorGuruDirectoryHandler) LinkUser(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		h.store.Flash(c, "error", "Guru tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	var req models.LinkGuruUserRequest
	if err := c.BodyParser(&req); err != nil || req.UserID < 1 {
		h.store.Flash(c, "error", "Akun tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
	}
	if err := h.guruService.LinkUser(id, req.UserID); err != nil {
		h.store.Flash(c, "error", err.Error())
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
	}
	h.store.Flash(c, "success", "Akun berhasil dihubungkan. Selanjutnya tetapkan role akun menjadi Guru di menu Kelola User.")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
}

func (h *KoordinatorGuruDirectoryHandler) Unlink(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		h.store.Flash(c, "error", "Guru tidak valid")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru")
	}
	if err := h.guruService.UnlinkUser(id); err != nil {
		h.store.Flash(c, "error", "Gagal melepas akun")
		return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
	}
	h.store.Flash(c, "success", "Akun dilepas dari data guru")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
}

func (h *KoordinatorGuruDirectoryHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Guru tidak valid"})
	}
	var req models.UpdateGuruDirectoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}
	req.Nama = strings.TrimSpace(req.Nama)
	req.Gelar = strings.TrimSpace(req.Gelar)
	req.NoWa = strings.TrimSpace(req.NoWa)
	req.Email = strings.TrimSpace(req.Email)
	req.Foto = strings.TrimSpace(req.Foto)
	if req.Nama == "" || (req.JenisKelamin != "L" && req.JenisKelamin != "P") || (req.Status != "tetap" && req.Status != "part_time") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data guru tidak valid"})
	}
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format email tidak valid"})
		}
	}
	if err := h.guruService.UpdateDirectory(id, req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	h.store.Flash(c, "success", "Data guru diperbarui")
	return h.inertiaService.Redirect(c, "/app/koordinator-guru/guru/"+strconv.FormatInt(id, 10))
}
