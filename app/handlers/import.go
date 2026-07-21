package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type ImportHandler struct {
	importService  *services.ImportService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewImportHandler(importService *services.ImportService, store *session.Store, inertiaService *services.InertiaService) *ImportHandler {
	return &ImportHandler{
		importService:  importService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *ImportHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	return h.inertiaService.Render(c, "app/ImportCSV", fiber.Map{
		"user": user,
	})
}

func (h *ImportHandler) Upload(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	file, err := c.FormFile("file")
	if err != nil {
		h.store.Flash(c, "error", "File tidak ditemukan")
		return h.inertiaService.Redirect(c, "/admin/import")
	}

	f, err := file.Open()
	if err != nil {
		h.store.Flash(c, "error", "Gagal membuka file")
		return h.inertiaService.Redirect(c, "/admin/import")
	}
	defer f.Close()

	result, err := h.importService.ProcessCSV(f, userID, file.Filename)
	if err != nil {
		h.store.Flash(c, "error", "Import gagal: "+err.Error())
		return h.inertiaService.Redirect(c, "/admin/import")
	}

	return h.inertiaService.Render(c, "app/ImportCSV", fiber.Map{
		"user":    sessionUser(nil),
		"result":  result,
		"success": "Import selesai",
	})
}

func (h *ImportHandler) Template(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=template-import-santri.csv")
	c.Write([]byte("kelas_kode,nama,jenis_kelamin,nominal,tanggal_daftar,angkatan,usia,domisili\n"))
	c.Write([]byte("R 1X,Nama Santri,L,100000,2026-01-15,AKA38,25,Jakarta\n"))
	return nil
}
