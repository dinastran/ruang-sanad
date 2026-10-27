package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestPembinaanDanRapatMembawaAnalisisAlfa(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO users (id, email, name, role) VALUES (1, 'koor@example.test', 'Koordinator', 'koordinator_guru')`)
	exec(`INSERT INTO guru (id, nama, no_wa, is_aktif) VALUES (10, 'Guru Alfa', '0812', 1), (11, 'Guru Rajin', '', 1)`)
	for i, tanggal := range []string{"2026-09-01", "2026-09-08", "2026-09-15"} {
		id := i + 1
		exec(`INSERT INTO pembinaan (id, tanggal, topik, status) VALUES (?, ?, 'Topik', 'terlaksana')`, id, tanggal)
		exec(`INSERT INTO pembinaan_absen (pembinaan_id, guru_id, hadir, kehadiran) VALUES (?, 10, 0, 'alfa'), (?, 11, 1, 'hadir')`, id, id)
		exec(`INSERT INTO rapat_guru (id, tanggal, judul, status) VALUES (?, ?, 'Rapat', 'terlaksana')`, id, tanggal)
		exec(`INSERT INTO rapat_absen (rapat_id, guru_id, hadir, kehadiran) VALUES (?, 10, 0, 'izin'), (?, 11, 0, 'alfa')`, id, id)
	}

	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	h := NewKoordinatorFeaturesHandler(services.NewKoordinatorService(q), services.NewGuruService(q), services.NewRiayahService(q), store, inertia)
	app := fiber.New()
	login := func(next fiber.Handler) fiber.Handler {
		return func(c *fiber.Ctx) error {
			sess, err := store.Get(c)
			if err != nil {
				return err
			}
			sess.Set("user_id", int64(1))
			sess.Set("role", "koordinator_guru")
			sess.Set("name", "Koordinator")
			return next(c)
		}
	}
	app.Get("/app/koordinator-guru/pembinaan", login(h.PembinaanList))
	app.Get("/app/koordinator-guru/rapat", login(h.RapatList))
	app.Post("/app/koordinator-guru/guru/:id/riayah", login(h.RiayahGuruCreate))

	type analisisProps struct {
		Props struct {
			Analisis struct {
				Merah int `json:"merah"`
				Guru  []struct {
					Nama         string `json:"nama"`
					AlfaBeruntun int    `json:"alfa_beruntun"`
				} `json:"guru"`
			} `json:"analisis"`
		} `json:"props"`
	}
	get := func(path string) analisisProps {
		t.Helper()
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("X-Inertia", "true")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, 200, resp.StatusCode)
		var page analisisProps
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
		return page
	}

	pembinaan := get("/app/koordinator-guru/pembinaan")
	require.Equal(t, 1, pembinaan.Props.Analisis.Merah)
	require.Equal(t, "Guru Alfa", pembinaan.Props.Analisis.Guru[0].Nama)
	require.Equal(t, 3, pembinaan.Props.Analisis.Guru[0].AlfaBeruntun)

	rapat := get("/app/koordinator-guru/rapat")
	require.Equal(t, 1, rapat.Props.Analisis.Merah)
	require.Equal(t, "Guru Rajin", rapat.Props.Analisis.Guru[0].Nama, "izin tidak dihitung alfa")

	// Catat riayah dari halaman analisis kembali ke halaman asal.
	req := httptest.NewRequest("POST", "/app/koordinator-guru/guru/10/riayah", strings.NewReader(`{"catatan":"Alfa 3x, sudah ditanyakan"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Referer", "/app/koordinator-guru/pembinaan")
	resp, err := app.Test(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 303, resp.StatusCode)
	require.Equal(t, "/app/koordinator-guru/pembinaan", resp.Header.Get("Location"))
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM catatan_riayah WHERE target_type = 'guru' AND target_id = 10`).Scan(&n))
	require.Equal(t, 1, n)
}
