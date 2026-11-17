package handlers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// TasbihHandler: tasbih digital pribadi. Terbuka untuk semua role yang login;
// semua data di-scope ke user_id sesi.
type TasbihHandler struct {
	tasbih         *services.TasbihService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewTasbihHandler(tasbih *services.TasbihService, store *session.Store, inertiaService *services.InertiaService) *TasbihHandler {
	return &TasbihHandler{tasbih: tasbih, store: store, inertiaService: inertiaService}
}

const tasbihFlashBadge = "tasbih_badge"

func (h *TasbihHandler) userID(c *fiber.Ctx) (int64, *session.Session) {
	sess, err := h.store.Get(c)
	if err != nil || sess == nil {
		return 0, nil
	}
	return toInt64(sess.Get("user_id")), sess
}

func (h *TasbihHandler) badgeBaru(c *fiber.Ctx) []models.TasbihBadge {
	raw := h.store.GetFlash(c, tasbihFlashBadge)
	if raw == "" {
		return []models.TasbihBadge{}
	}
	return services.BadgeDariKode(strings.Split(raw, ","))
}

// kembali: redirect back setelah aksi router.*; error validasi lewat flash.
func (h *TasbihHandler) kembali(c *fiber.Ctx, userID int64, err error) error {
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTasbihValidasi), errors.Is(err, services.ErrTasbihTidakDitemukan):
			h.store.Flash(c, "error", err.Error())
		default:
			return c.Status(fiber.StatusInternalServerError).SendString("gagal menyimpan")
		}
		return h.inertiaService.Back(c, "/app/tasbih")
	}
	if baru, err := h.tasbih.EvaluasiBadge(c.UserContext(), userID, time.Now()); err == nil && len(baru) > 0 {
		kode := make([]string, 0, len(baru))
		for _, b := range baru {
			kode = append(kode, b.Kode)
		}
		h.store.Flash(c, tasbihFlashBadge, strings.Join(kode, ","))
	}
	return h.inertiaService.Back(c, "/app/tasbih")
}

func (h *TasbihHandler) Index(c *fiber.Ctx) error {
	uid, sess := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	ctx, now := c.UserContext(), time.Now()
	dzikir, paket, err := h.tasbih.Katalog(ctx, uid)
	if err != nil {
		return err
	}
	hariIni, err := h.tasbih.HariIni(ctx, uid, now)
	if err != nil {
		return err
	}
	streak, err := h.tasbih.Streak(ctx, uid, now)
	if err != nil {
		return err
	}
	level, err := h.tasbih.Level(ctx, uid)
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/tasbih/Index", fiber.Map{
		"user": sessionUser(sess), "dzikir": dzikir, "paket": paket, "hari_ini": hariIni,
		"target_harian": hariIni.Target, "streak": streak, "level": level, "badge_baru": h.badgeBaru(c),
	})
}

func (h *TasbihHandler) Dashboard(c *fiber.Ctx) error {
	uid, sess := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	ctx := c.UserContext()
	rentang, _ := strconv.Atoi(c.Query("rentang", "30"))
	dash, err := h.tasbih.Dashboard(ctx, uid, rentang, time.Now())
	if err != nil {
		return err
	}
	badges, err := h.tasbih.DaftarBadge(ctx, uid)
	if err != nil {
		return err
	}
	items, adaLagi, err := h.tasbih.Riwayat(ctx, uid, 1, nil)
	if err != nil {
		return err
	}
	dzikir, _, err := h.tasbih.Katalog(ctx, uid)
	if err != nil {
		return err
	}
	return h.inertiaService.Render(c, "app/tasbih/Dashboard", fiber.Map{
		"user": sessionUser(sess), "dashboard": dash, "badges": badges,
		"riwayat": fiber.Map{"items": items, "ada_lagi": adaLagi},
		"dzikir":  dzikir, "target_harian": dash.HariIni.Target, "badge_baru": h.badgeBaru(c),
	})
}

func (h *TasbihHandler) Riwayat(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	halaman, _ := strconv.Atoi(c.Query("halaman", "1"))
	var dzikirID *int64
	if v, err := strconv.ParseInt(c.Query("dzikir_id"), 10, 64); err == nil && v > 0 {
		dzikirID = &v
	}
	items, adaLagi, err := h.tasbih.Riwayat(c.UserContext(), uid, halaman, dzikirID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal memuat riwayat"})
	}
	return c.JSON(fiber.Map{"items": items, "ada_lagi": adaLagi})
}

type tasbihSyncReq struct {
	Sesi []models.TasbihSesiInput `json:"sesi"`
}

func (h *TasbihHandler) Sync(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var req tasbihSyncReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "format data tidak valid"})
	}
	hasil, err := h.tasbih.Sync(c.UserContext(), uid, req.Sesi, time.Now())
	if errors.Is(err, services.ErrTasbihValidasi) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal menyimpan sesi"})
	}
	return c.JSON(hasil)
}

func (h *TasbihHandler) parseDzikir(c *fiber.Ctx) (models.TasbihDzikirInput, error) {
	var in models.TasbihDzikirInput
	if err := c.BodyParser(&in); err != nil {
		return in, services.ErrTasbihValidasi
	}
	return in, nil
}

func (h *TasbihHandler) BuatDzikir(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	in, err := h.parseDzikir(c)
	if err == nil {
		_, err = h.tasbih.BuatDzikir(c.UserContext(), uid, in)
	}
	return h.kembali(c, uid, err)
}

func (h *TasbihHandler) UbahDzikir(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	in, err := h.parseDzikir(c)
	if err == nil {
		err = h.tasbih.UbahDzikir(c.UserContext(), uid, id, in)
	}
	return h.kembali(c, uid, err)
}

func (h *TasbihHandler) ArsipkanDzikir(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	return h.kembali(c, uid, h.tasbih.ArsipkanDzikir(c.UserContext(), uid, id))
}

type tasbihTargetReq struct {
	Target int64 `json:"target"`
}

func (h *TasbihHandler) SetTargetDzikir(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req tasbihTargetReq
	err := c.BodyParser(&req)
	if err != nil {
		err = services.ErrTasbihValidasi
	} else {
		err = h.tasbih.SetTargetDzikir(c.UserContext(), uid, id, req.Target)
	}
	return h.kembali(c, uid, err)
}

func (h *TasbihHandler) SetTargetHarian(c *fiber.Ctx) error {
	uid, _ := h.userID(c)
	if uid == 0 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var req tasbihTargetReq
	err := c.BodyParser(&req)
	if err != nil {
		err = services.ErrTasbihValidasi
	} else {
		err = h.tasbih.SetTargetHarian(c.UserContext(), uid, req.Target)
	}
	return h.kembali(c, uid, err)
}
