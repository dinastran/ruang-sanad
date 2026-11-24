package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// ProgramKelasHandler manages the ikhwan/akhwat classes of short programs:
// teacher and Zoom link per class, moving a participant, deleting an empty
// class. Participants are placed automatically by the program service.
type ProgramKelasHandler struct {
	programs       *services.LeadProgramService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewProgramKelasHandler(programs *services.LeadProgramService, store *session.Store, inertiaService *services.InertiaService) *ProgramKelasHandler {
	return &ProgramKelasHandler{programs: programs, store: store, inertiaService: inertiaService}
}

func (h *ProgramKelasHandler) done(c *fiber.Ctx, programID int64, err error, success string) error {
	if err != nil {
		h.store.Flash(c, "error", err.Error())
	} else {
		h.store.Flash(c, "success", success)
	}
	if programID <= 0 {
		return h.inertiaService.Redirect(c, leadProgramsPath)
	}
	return h.inertiaService.Redirect(c, leadProgramPath(programID))
}

func programQuery(c *fiber.Ctx) int64 {
	id, _ := strconv.ParseInt(c.Query("program"), 10, 64)
	return id
}

// Update sets a class's teacher and Zoom link.
func (h *ProgramKelasHandler) Update(c *fiber.Ctx) error {
	kid, err := sourceIDParam(c, "kid")
	var req models.ProgramKelasRequest
	if err != nil || c.BodyParser(&req) != nil {
		return h.done(c, programQuery(c), errors.New("data tidak valid"), "")
	}
	pid, err := h.programs.UpdateKelas(kid, req)
	if pid == 0 {
		pid = programQuery(c)
	}
	return h.done(c, pid, err, "Kelas disimpan")
}

// Delete removes an empty class.
func (h *ProgramKelasHandler) Delete(c *fiber.Ctx) error {
	kid, err := sourceIDParam(c, "kid")
	if err != nil {
		return h.done(c, programQuery(c), err, "")
	}
	pid, err := h.programs.DeleteKelas(kid)
	if pid == 0 {
		pid = programQuery(c)
	}
	return h.done(c, pid, err, "Kelas kosong dihapus")
}

// SetPeserta sets a participant's gender (class picked automatically) or
// moves them to a chosen class.
func (h *ProgramKelasHandler) SetPeserta(c *fiber.Ctx) error {
	pid, err1 := sourceIDParam(c, "pid")
	oid, err2 := sourceIDParam(c, "oid")
	var req models.ProgramPesertaKelasRequest
	if err1 != nil || err2 != nil || c.BodyParser(&req) != nil {
		return h.done(c, pid, errors.New("data tidak valid"), "")
	}
	return h.done(c, pid, h.programs.SetPesertaKelas(pid, oid, req), "Kelas peserta diperbarui")
}
