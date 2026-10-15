package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

const guruListPath = "/app/koordinator-guru/guru"

// HapusInfo returns JSON for the delete dialog: whether the guru can be
// deleted, and otherwise which classes and history block it.
func (h *KoordinatorGuruDirectoryHandler) HapusInfo(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Guru tidak valid"})
	}
	info, err := h.guruService.InfoHapus(id)
	if errors.Is(err, services.ErrGuruTidakDitemukan) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memeriksa riwayat guru"})
	}
	return c.JSON(info)
}

func (h *KoordinatorGuruDirectoryHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		h.store.Flash(c, "error", "Guru tidak valid")
		return h.inertiaService.Redirect(c, guruListPath)
	}
	if err := h.guruService.Hapus(id); err != nil {
		h.store.Flash(c, "error", guruHapusErrorMessage(err, "Gagal menghapus guru"))
		return h.inertiaService.Back(c, guruListPath)
	}
	h.store.Flash(c, "success", "Data guru dihapus")
	return h.inertiaService.Redirect(c, guruListPath)
}

func (h *KoordinatorGuruDirectoryHandler) Nonaktifkan(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		h.store.Flash(c, "error", "Guru tidak valid")
		return h.inertiaService.Redirect(c, guruListPath)
	}
	if err := h.guruService.Nonaktifkan(id); err != nil {
		h.store.Flash(c, "error", guruHapusErrorMessage(err, "Gagal menonaktifkan guru"))
		return h.inertiaService.Back(c, guruListPath)
	}
	h.store.Flash(c, "success", "Guru dinonaktifkan. Riwayatnya tetap tersimpan.")
	return h.inertiaService.Back(c, guruListPath)
}

func guruHapusErrorMessage(err error, fallback string) string {
	switch {
	case errors.Is(err, services.ErrGuruTidakDitemukan),
		errors.Is(err, services.ErrGuruMasihMengajar),
		errors.Is(err, services.ErrGuruPunyaRiwayat):
		return err.Error()
	default:
		return fallback
	}
}
