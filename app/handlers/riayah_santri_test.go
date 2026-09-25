package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

type riayahHandlerFixture struct {
	app     *fiber.App
	db      *sql.DB
	santriA int64
}

func setupRiayahHandler(t *testing.T) *riayahHandlerFixture {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	insertUser := func(email, role string) int64 {
		res, err := db.Exec(`INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, ?)`, email, email, hashPW(t, "password123"), role)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	insertGuru := func(userID int64) int64 {
		res, err := db.Exec(`INSERT INTO guru (nama, user_id, is_aktif) VALUES (?, ?, 1)`, fmt.Sprintf("Guru %d", userID), userID)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	guruA := insertGuru(insertUser("guru-a@example.com", "guru"))
	insertGuru(insertUser("guru-b@example.com", "guru"))
	insertUser("koordinator@example.com", "koordinator_guru")

	res, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri) VALUES ('RA', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Senin', 1, 'Kelas A', ?, 20, 1)`, guruA)
	require.NoError(t, err)
	kelasA, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES ('Santri A', ?, 'aktif')`, kelasA)
	require.NoError(t, err)
	santriA, err := res.LastInsertId()
	require.NoError(t, err)

	q := queries.NewQuerier(db)
	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	userService := services.NewUserService(q)
	auth := NewAuthHandler(services.NewAuthService(q, services.AuthServiceConfig{SessionSecret: "test-secret-32-chars-long-for-testing!!"}), store, inertia)
	h := NewRiayahSantriHandler(services.NewGuruService(q), services.NewRiayahSantriService(q), store, inertia)

	app := fiber.New()
	app.Post("/login", middlewares.Guest(store), auth.Login)
	protected := app.Group("/app", middlewares.AuthRequired(store))
	readRole := middlewares.RoleRequired(store, userService, "guru", "admin_kelas", "super_admin", "koordinator_guru")
	writeRole := middlewares.RoleRequired(store, userService, "guru", "admin_kelas", "super_admin")
	protected.Get("/guru/riayah", readRole, h.Index)
	protected.Get("/guru/santri/:sid", readRole, h.Profil)
	protected.Post("/guru/santri/:sid/catatan", writeRole, h.CatatanCreate)
	protected.Post("/guru/santri/:sid/kontak", writeRole, h.KontakCreate)

	return &riayahHandlerFixture{app: app, db: db, santriA: santriA}
}

func (f *riayahHandlerFixture) login(t *testing.T, email string) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(fmt.Sprintf(`{"email":%q,"password":"password123"}`, email)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	for _, c := range resp.Cookies() {
		if c.Name == "session_id" {
			return c
		}
	}
	t.Fatalf("login %s tidak menghasilkan session", email)
	return nil
}

func (f *riayahHandlerFixture) do(t *testing.T, cookie *http.Cookie, method, path, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.AddCookie(cookie)
	req.Header.Set("X-Inertia", "true")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := f.app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func inertiaPage(t *testing.T, resp *http.Response) (string, map[string]any) {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var page struct {
		Component string         `json:"component"`
		Props     map[string]any `json:"props"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	return page.Component, page.Props
}

func TestRiayahSantriKoordinatorHanyaMembaca(t *testing.T) {
	f := setupRiayahHandler(t)
	cookie := f.login(t, "koordinator@example.com")

	component, props := inertiaPage(t, f.do(t, cookie, http.MethodGet, "/app/guru/riayah", ""))
	require.Equal(t, "guru/RiayahSantri", component)
	require.Equal(t, false, props["can_write"])
	require.Len(t, props["santri"], 1)

	component, _ = inertiaPage(t, f.do(t, cookie, http.MethodGet, fmt.Sprintf("/app/guru/santri/%d", f.santriA), ""))
	require.Equal(t, "guru/SantriProfil", component)

	resp := f.do(t, cookie, http.MethodPost, fmt.Sprintf("/app/guru/santri/%d/catatan", f.santriA), `{"catatan":"tidak boleh"}`)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp = f.do(t, cookie, http.MethodPost, fmt.Sprintf("/app/guru/santri/%d/kontak", f.santriA), `{"media":"wa"}`)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRiayahSantriGuruTerbatasPadaKelasnya(t *testing.T) {
	f := setupRiayahHandler(t)
	profil := fmt.Sprintf("/app/guru/santri/%d", f.santriA)

	guruB := f.login(t, "guru-b@example.com")
	resp := f.do(t, guruB, http.MethodGet, profil, "")
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.Equal(t, "/app/guru/riayah", resp.Header.Get("Location"))
	resp = f.do(t, guruB, http.MethodPost, profil+"/catatan", `{"catatan":"bukan santri saya"}`)
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	guruA := f.login(t, "guru-a@example.com")
	component, props := inertiaPage(t, f.do(t, guruA, http.MethodGet, profil, ""))
	require.Equal(t, "guru/SantriProfil", component)
	require.Equal(t, true, props["can_write"])
	resp = f.do(t, guruA, http.MethodPost, profil+"/catatan", `{"catatan":"Alhamdulillah mulai lancar"}`)
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	var jumlah int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM catatan_riayah WHERE target_type = 'santri' AND target_id = ?`, f.santriA).Scan(&jumlah))
	require.Equal(t, 1, jumlah)

	resp = f.do(t, guruA, http.MethodPost, profil+"/kontak?kembali=/app/guru/riayah", `{"media":"telepon","catatan":"Menanyakan kabar"}`)
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.Equal(t, "/app/guru/riayah", resp.Header.Get("Location"))
	resp = f.do(t, guruB, http.MethodPost, profil+"/kontak", `{"media":"wa"}`)
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM riayah_kontak WHERE santri_id = ?`, f.santriA).Scan(&jumlah))
	require.Equal(t, 1, jumlah)
}
