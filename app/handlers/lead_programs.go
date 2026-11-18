package handlers

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// LeadProgramHandler serves CRM programs (Tahsin Reguler and short programs
// such as Sepekan Bersama Al-Fatihah): the program pages with sessions and
// attendance, the lead actions (program, participant closing, offer regular
// class) and the public attendance link /hadir/:token.
type LeadProgramHandler struct {
	leads          *services.LeadService
	programs       *services.LeadProgramService
	produk         *services.ProductCRMService
	store          *session.Store
	inertiaService *services.InertiaService
	editorRoles    []string
}

func NewLeadProgramHandler(leads *services.LeadService, programs *services.LeadProgramService, produk *services.ProductCRMService, store *session.Store, inertiaService *services.InertiaService, editorRoles []string) *LeadProgramHandler {
	return &LeadProgramHandler{leads: leads, programs: programs, produk: produk, store: store, inertiaService: inertiaService, editorRoles: editorRoles}
}

const leadProgramsPath = "/app/leads/program"

func leadProgramPath(id int64) string { return fmt.Sprintf("%s/%d", leadProgramsPath, id) }

func (h *LeadProgramHandler) canEdit(c *fiber.Ctx) bool {
	role := roleOf(c)
	for _, r := range h.editorRoles {
		if r == role {
			return true
		}
	}
	return false
}

func (h *LeadProgramHandler) done(c *fiber.Ctx, path string, err error, success string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", success)
	}
	return h.inertiaService.Redirect(c, path)
}

// produkProgram lists Produk CRM products of category program, for the
// program form (product and batch).
func (h *LeadProgramHandler) produkProgram() ([]models.Product, error) {
	items, err := h.produk.Products()
	if err != nil {
		return nil, err
	}
	out := make([]models.Product, 0)
	for _, p := range items {
		if p.Kategori == "program" {
			out = append(out, p)
		}
	}
	return out, nil
}

func (h *LeadProgramHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	programs, err := h.programs.List()
	if err != nil {
		return err
	}
	sources, err := h.leads.SourceOptions(c.UserContext())
	if err != nil {
		return err
	}
	produk, err := h.produkProgram()
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/LeadPrograms", fiber.Map{
		"user": sessionUser(sess), "programs": programs, "sources": sources, "produk": produk, "can_edit": h.canEdit(c),
	})
}

func (h *LeadProgramHandler) Show(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	id, err := sourceIDParam(c, "pid")
	if err != nil {
		return h.inertiaService.Redirect(c, leadProgramsPath)
	}
	detail, err := h.programs.Detail(id)
	if err != nil {
		h.store.Flash(c, "error", "Program tidak ditemukan")
		return h.inertiaService.Redirect(c, leadProgramsPath)
	}
	sources, err := h.leads.SourceOptions(c.UserContext())
	if err != nil {
		return err
	}
	produk, err := h.produkProgram()
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/LeadProgramDetail", fiber.Map{
		"user": sessionUser(sess), "program": detail.Program, "sesi": detail.Sesi, "peserta": detail.Peserta,
		"sources": sources, "produk": produk, "can_edit": h.canEdit(c),
	})
}

func (h *LeadProgramHandler) Create(c *fiber.Ctx) error {
	var req models.LeadProgramRequest
	if c.BodyParser(&req) != nil {
		return h.done(c, leadProgramsPath, errors.New("data tidak valid"), "")
	}
	id, err := h.programs.Create(req)
	if err != nil {
		return h.done(c, leadProgramsPath, err, "")
	}
	return h.done(c, leadProgramPath(id), nil, "Program dibuat. Petakan sumber lead dan tambahkan sesi.")
}

func (h *LeadProgramHandler) Update(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "pid")
	var req models.LeadProgramRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, leadProgramsPath, errors.New("data tidak valid"), "")
	}
	return h.done(c, leadProgramPath(id), h.programs.Update(id, req), "Program disimpan")
}

func (h *LeadProgramHandler) SetSources(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "pid")
	var req struct {
		SumberIDs []int64 `json:"sumber_ids"`
	}
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, leadProgramsPath, errors.New("data tidak valid"), "")
	}
	return h.done(c, leadProgramPath(id), h.programs.SetSources(id, req.SumberIDs), "Sumber lead program disimpan. Lead dari sumber itu kini masuk program ini.")
}

func (h *LeadProgramHandler) CreateSesi(c *fiber.Ctx) error {
	id, err := sourceIDParam(c, "pid")
	var req models.ProgramSesiRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, leadProgramsPath, errors.New("data tidak valid"), "")
	}
	_, err = h.programs.CreateSesi(id, req)
	return h.done(c, leadProgramPath(id), err, "Sesi ditambahkan beserta link absensinya")
}

func (h *LeadProgramHandler) UpdateSesi(c *fiber.Ctx) error {
	sid, err := sourceIDParam(c, "sid")
	pid, _ := strconv.ParseInt(c.Query("program"), 10, 64)
	var req models.ProgramSesiRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, leadProgramPath(pid), errors.New("data tidak valid"), "")
	}
	return h.done(c, leadProgramPath(pid), h.programs.UpdateSesi(sid, req), "Sesi disimpan")
}

func (h *LeadProgramHandler) DeleteSesi(c *fiber.Ctx) error {
	sid, err := sourceIDParam(c, "sid")
	if err != nil {
		return h.done(c, leadProgramsPath, err, "")
	}
	pid, err := h.programs.DeleteSesi(sid)
	if pid == 0 {
		pid, _ = strconv.ParseInt(c.Query("program"), 10, 64)
	}
	return h.done(c, leadProgramPath(pid), err, "Sesi dihapus")
}

// SetHadir is the admin correction of one participant's attendance (JSON,
// called with fetch so the attendance grid updates in place).
func (h *LeadProgramHandler) SetHadir(c *fiber.Ctx) error {
	sid, err1 := sourceIDParam(c, "sid")
	oid, err2 := sourceIDParam(c, "oid")
	var req struct {
		Hadir bool `json:"hadir"`
	}
	if err1 != nil || err2 != nil || c.BodyParser(&req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if _, err := h.programs.SetHadir(sid, oid, req.Hadir); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// --- Lead actions -----------------------------------------------------------------

func (h *LeadProgramHandler) leadDone(c *fiber.Ctx, id int64, err error, success string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", success)
	}
	if pid, _ := strconv.ParseInt(c.Query("program"), 10, 64); pid > 0 {
		return h.inertiaService.Redirect(c, leadProgramPath(pid))
	}
	return h.inertiaService.Redirect(c, leadPath(id))
}

func (h *LeadProgramHandler) SetLeadProgram(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req struct {
		ProgramID int64 `json:"program_id"`
	}
	if !ok || c.BodyParser(&req) != nil {
		return h.leadDone(c, id, errors.New("data tidak valid"), "")
	}
	return h.leadDone(c, id, h.leads.SetProgram(c.UserContext(), id, req.ProgramID, c.Locals("user_id").(int64)), "Program lead diperbarui")
}

func (h *LeadProgramHandler) DaftarkanPeserta(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	var req models.LeadPesertaRequest
	if !ok || c.BodyParser(&req) != nil {
		return h.leadDone(c, id, errors.New("data tidak valid"), "")
	}
	_, err := h.leads.DaftarkanPeserta(c.UserContext(), id, c.Locals("user_id").(int64), req)
	return h.leadDone(c, id, err, "Lead didaftarkan sebagai peserta program dan tercatat di Produk CRM")
}

func (h *LeadProgramHandler) TawarkanReguler(c *fiber.Ctx) error {
	id, ok := leadIDParam(c)
	if !ok {
		return h.leadDone(c, id, errors.New("data tidak valid"), "")
	}
	return h.leadDone(c, id, h.leads.TawarkanReguler(c.UserContext(), id, c.Locals("user_id").(int64)),
		"Lead dibuka lagi di Tahsin Reguler (tahap rekomendasi). Silakan tawarkan kelas reguler.")
}

// --- Public attendance link ----------------------------------------------------------

func (h *LeadProgramHandler) HadirPage(c *fiber.Ctx) error {
	info, _, err := h.programs.HadirInfo(c.Params("token"))
	props := fiber.Map{"token": c.Params("token")}
	if err != nil {
		c.Status(fiber.StatusNotFound)
		props["galat"] = "Link absensi tidak dikenal. Pastikan link dari host Zoom disalin lengkap."
	} else {
		props["sesi"] = info
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return h.inertiaService.Render(c, "public/HadirProgram", props)
}

func (h *LeadProgramHandler) HadirSubmit(c *fiber.Ctx) error {
	var req struct {
		NoWA string `json:"no_wa"`
	}
	if c.BodyParser(&req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	hasil, err := h.programs.Hadir(c.Params("token"), req.NoWA)
	if err != nil {
		status := fiber.StatusUnprocessableEntity
		if errors.Is(err, services.ErrProgramSesiTidakAda) {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(hasil)
}
