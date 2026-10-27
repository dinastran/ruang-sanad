package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/middlewares"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestTindakLanjutAlpaUntukAdminKelas(t *testing.T) {
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
	exec(`INSERT INTO guru (id, nama, is_aktif) VALUES (1, 'Guru', 1)`)
	exec(`INSERT INTO kelas (id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri) VALUES (1, 'K', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Senin', 1, 'Kelas K', 1, 20, 0)`)
	exec(`INSERT INTO santri (id, nama, kelas_id, status, mulai_belajar) VALUES (1, 'Santri Alpa', 1, 'aktif', ?)`, services.HariIniRiayah().AddDate(0, 0, -5).Format("2006-01-02"))
	for i, hariLalu := range []int{10, 3} {
		tanggal := services.HariIniRiayah().AddDate(0, 0, -hariLalu).Format("2006-01-02")
		exec(`INSERT INTO pertemuan (id, kelas_id, pertemuan_ke, pertemuan_level_ke, tanggal, status) VALUES (?, 1, ?, ?, ?, 'selesai')`, i+1, i+1, i+1, tanggal)
		exec(`INSERT INTO absensi (pertemuan_id, santri_id, status) VALUES (?, 1, 'alpa')`, i+1)
	}

	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	h := NewRiayahTindakLanjutHandler(services.NewRiayahSantriService(q), store, inertia)
	akRole := middlewares.RoleRequired(store, services.NewUserService(q), "admin_kelas", "super_admin")
	app := fiber.New()
	app.Get("/app/kelas/tindak-lanjut-alpa", akRole, h.Index)
	app.Get("/app/kelas/tindak-lanjut-alpa/jumlah", akRole, h.Jumlah)

	login := func(role string) string {
		res, err := db.Exec(`INSERT INTO users (email, name, role) VALUES (?, ?, ?)`, role+"@example.test", role, role)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		sid := fmt.Sprintf("session-%s", role)
		now := time.Now()
		exec(`INSERT INTO sessions (id, user_id, data, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			sid, id, fmt.Sprintf(`{"user_id":%d,"role":"%s"}`, id, role), now.Add(time.Hour), now, now)
		return sid
	}
	do := func(sid, path string, inertiaReq bool) *http.Response {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})
		if inertiaReq {
			req.Header.Set("X-Inertia", "true")
		}
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		return resp
	}

	guru := login("guru")
	koordinator := login("koordinator_guru")
	adminKelas := login("admin_kelas")
	for _, sid := range []string{guru, koordinator} {
		for _, path := range []string{"/app/kelas/tindak-lanjut-alpa", "/app/kelas/tindak-lanjut-alpa/jumlah"} {
			resp := do(sid, path, false)
			resp.Body.Close()
			require.Equal(t, fiber.StatusForbidden, resp.StatusCode, path)
		}
	}

	resp := do(adminKelas, "/app/kelas/tindak-lanjut-alpa/jumlah", false)
	require.Equal(t, 200, resp.StatusCode)
	var jumlah struct {
		Perlu int `json:"perlu"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&jumlah))
	resp.Body.Close()
	require.Equal(t, 1, jumlah.Perlu)

	resp = do(adminKelas, "/app/kelas/tindak-lanjut-alpa", true)
	require.Equal(t, 200, resp.StatusCode)
	var page struct {
		Component string `json:"component"`
		Props     struct {
			Perlu []struct {
				Nama             string `json:"nama"`
				TindakLanjutAlpa struct {
					AlpaBeruntun int `json:"alpa_beruntun"`
				} `json:"tindak_lanjut_alpa"`
			} `json:"perlu"`
			Sudah []json.RawMessage `json:"sudah"`
		} `json:"props"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	resp.Body.Close()
	require.Equal(t, "app/TindakLanjutAlpa", page.Component)
	require.Len(t, page.Props.Perlu, 1)
	require.Equal(t, "Santri Alpa", page.Props.Perlu[0].Nama)
	require.Equal(t, 2, page.Props.Perlu[0].TindakLanjutAlpa.AlpaBeruntun)
	require.NotNil(t, page.Props.Sudah)
}
