package handlers

import (
	"io"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// UploadVoiceNote updates one santri's VN metadata and optionally replaces its audio file.
func (h *SantriHandler) UploadVoiceNote(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID santri tidak valid"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data voice note tidak valid"})
	}

	var file io.ReadSeeker
	var fileSize int64
	if files := form.File["file"]; len(files) > 0 {
		uploadedFile, err := files[0].Open()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Gagal membaca voice note"})
		}
		defer uploadedFile.Close()
		file = uploadedFile
		fileSize = files[0].Size
	}
	description := ""
	if values := form.Value["keterangan_vn"]; len(values) > 0 {
		description = values[0]
	}

	santri, err := h.santriService.UpdateVoiceNote(id, description, file, fileSize)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "santri": santri})
}

// ServeVoiceNote streams a voice note only to authorized class administrators.
func (h *SantriHandler) ServeVoiceNote(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID santri tidak valid"})
	}
	path, err := h.santriService.VoiceNotePath(id)
	if err != nil {
		if os.IsNotExist(err) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendFile(path)
}
