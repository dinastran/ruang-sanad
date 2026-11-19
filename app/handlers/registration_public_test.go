package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

type daftarFixture struct {
	db  *sql.DB
	q   *queries.Querier
	reg *services.RegistrationService
	app *fiber.App
}

func setupDaftarFixture(t *testing.T, mode, afterSubmit string) *daftarFixture {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	q := queries.NewQuerier(db)

	res, err := db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Tahsin Reguler', 'program', 150000)`)
	require.NoError(t, err)
	produkID, _ := res.LastInsertId()
	_, err = db.Exec(`UPDATE lead_programs SET produk_id = ? WHERE id = 1`, produkID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO jadwal_penawaran (kode, hari, jam_mulai, jam_selesai, kuota_per_gender, is_buka, periode_mulai, created_at, updated_at)
		VALUES ('tst1', '1', '19:30', '20:30', 1, 1, '2026-01-01 00:00:00', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	var sourceID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM lead_sources ORDER BY id LIMIT 1`).Scan(&sourceID))

	leads := services.NewLeadService(q, services.NewSantriService(q, services.NewKelasEngineService(q)))
	reg := services.NewRegistrationService(q, leads, services.NewJadwalPenawaranService(q), []byte("secret"))
	reg.SetProofDir(t.TempDir())
	ctx := t.Context()
	require.NoError(t, reg.SaveSettings(ctx, models.RegistrationSettings{CsWaNumber: "6281100000000", PaymentExpiryHours: 2, BankManualInfo: "BSI 1"}, 0))
	_, err = reg.CreateLink(ctx, models.RegistrationLinkRequest{
		Slug: "tahsin", Judul: "Daftar Tahsin", ProgramID: 1, LeadSourceID: sourceID,
		Mode: mode, AfterSubmit: afterSubmit, IzinkanManual: true, IsAktif: true,
	}, 0)
	require.NoError(t, err)

	store := session.New(q, nil, time.Hour)
	inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
	h := NewRegistrationPublicHandler(reg, inertia)
	app := fiber.New()
	app.Get("/daftar/bayar/:ref", h.BayarPage)
	app.Post("/daftar/bayar/:ref/bukti", h.UploadBukti)
	app.Get("/daftar/:slug", h.Page)
	app.Post("/daftar/:slug", h.Submit)
	return &daftarFixture{db: db, q: q, reg: reg, app: app}
}

func (f *daftarFixture) do(t *testing.T, method, path, contentType string, body io.Reader) (int, string, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := f.app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Location"), raw
}

func (f *daftarFixture) postJSON(t *testing.T, path string, v any) (int, []byte) {
	t.Helper()
	b, _ := json.Marshal(v)
	code, _, raw := f.do(t, "POST", path, "application/json", bytes.NewReader(b))
	return code, raw
}

// oldToken signs a form token 10s in the past so it passes the minimum age.
func (f *daftarFixture) oldToken(slug string) string {
	f.reg.SetNow(func() time.Time { return time.Now().Add(-10 * time.Second) })
	tok := f.reg.FormToken(slug)
	f.reg.SetNow(time.Now)
	return tok
}

func TestDaftarModeWARedirects(t *testing.T) {
	f := setupDaftarFixture(t, "wa", "wa")
	code, loc, _ := f.do(t, "GET", "/daftar/tahsin", "", nil)
	require.Equal(t, fiber.StatusFound, code)
	require.True(t, strings.HasPrefix(loc, "https://wa.me/6281100000000"))
	var klik int
	require.NoError(t, f.db.QueryRow(`SELECT COALESCE(SUM(klik), 0) FROM registration_link_stats`).Scan(&klik))
	require.Equal(t, 1, klik)
}

func TestDaftarInactiveLink404(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	_, err := f.db.Exec(`UPDATE registration_links SET is_aktif = 0`)
	require.NoError(t, err)
	code, _, _ := f.do(t, "GET", "/daftar/tahsin", "", nil)
	require.Equal(t, fiber.StatusNotFound, code)
	code, _, _ = f.do(t, "GET", "/daftar/tidak-ada", "", nil)
	require.Equal(t, fiber.StatusNotFound, code)
}

func TestDaftarPageRenders(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	code, _, raw := f.do(t, "GET", "/daftar/tahsin", "", nil)
	require.Equal(t, fiber.StatusOK, code)
	require.Contains(t, string(raw), "public/Daftar")
}

func TestDaftarSubmitFieldError422(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, jenis_kelamin, jadwal_kode, verified_at, created_at, updated_at)
		VALUES ('6289999999999@s.whatsapp.net', '6289999999999', 'P', 'tst1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	code, raw := f.postJSON(t, "/daftar/tahsin", map[string]any{
		"nama": "Fulanah", "no_wa": "081234567890", "jenis_kelamin": "P", "jadwal_kode": "tst1", "form_token": f.oldToken("tahsin"),
	})
	require.Equal(t, fiber.StatusUnprocessableEntity, code)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, "jadwal_kode", out["field"])
	require.NotEmpty(t, out["error"])
}

func TestDaftarSubmitOKAndUploadBukti(t *testing.T) {
	f := setupDaftarFixture(t, "short", "bayar")
	code, raw := f.postJSON(t, "/daftar/tahsin", map[string]any{
		"nama": "Fulanah", "no_wa": "081234567890", "jenis_kelamin": "P", "jadwal_kode": "tst1", "form_token": f.oldToken("tahsin"),
	})
	require.Equal(t, fiber.StatusOK, code, string(raw))
	var out struct {
		Next     string `json:"next"`
		BayarURL string `json:"bayar_url"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, "bayar", out.Next)
	require.True(t, strings.HasPrefix(out.BayarURL, "/daftar/bayar/"))

	code, _, page := f.do(t, "GET", out.BayarURL, "", nil)
	require.Equal(t, fiber.StatusOK, code)
	require.Contains(t, string(page), "public/DaftarBayar")

	upload := func(content []byte) int {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, err := w.CreateFormFile("bukti", "bukti.png")
		require.NoError(t, err)
		_, _ = part.Write(content)
		require.NoError(t, w.Close())
		code, _, _ := f.do(t, "POST", out.BayarURL+"/bukti", w.FormDataContentType(), &buf)
		return code
	}
	require.Equal(t, fiber.StatusUnprocessableEntity, upload([]byte("MZ not an image at all")))
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 64)...)
	require.Equal(t, fiber.StatusOK, upload(png))
	var stage string
	require.NoError(t, f.db.QueryRow(`SELECT stage FROM leads`).Scan(&stage))
	require.Equal(t, "bukti_masuk", stage)

	code, _, _ = f.do(t, "GET", "/daftar/bayar/tidak-ada", "", nil)
	require.Equal(t, fiber.StatusNotFound, code)
}

func TestFixClientIPBehindProxy(t *testing.T) {
	f := setupDaftarFixture(t, "wa", "wa")
	for _, ip := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest("GET", "/daftar/tahsin", nil)
		req.Header.Set("X-Forwarded-For", ip)
		resp, err := f.app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusFound, resp.StatusCode)
	}
	var klik int
	require.NoError(t, f.db.QueryRow(`SELECT COALESCE(SUM(klik), 0) FROM registration_link_stats`).Scan(&klik))
	require.Equal(t, 2, klik, "pengunjung berbeda di balik proxy dihitung terpisah")
}

func TestFixSubmitInactiveLinkGivesWaURL(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	tok := f.oldToken("tahsin")
	_, err := f.db.Exec(`UPDATE registration_links SET is_aktif = 0`)
	require.NoError(t, err)
	code, raw := f.postJSON(t, "/daftar/tahsin", map[string]any{"nama": "Fulanah", "no_wa": "081234567890", "jenis_kelamin": "P", "jadwal_kode": "tst1", "form_token": tok})
	require.Equal(t, fiber.StatusNotFound, code)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Contains(t, out["wa_cs_url"], "https://wa.me/6281100000000")
}

func TestFixFullScheduleReturnsFreshJadwal(t *testing.T) {
	f := setupDaftarFixture(t, "short", "wa")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, jenis_kelamin, jadwal_kode, verified_at, created_at, updated_at)
		VALUES ('6289999999999@s.whatsapp.net', '6289999999999', 'P', 'tst1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	code, raw := f.postJSON(t, "/daftar/tahsin", map[string]any{"nama": "Fulanah", "no_wa": "081234567890", "jenis_kelamin": "P", "jadwal_kode": "tst1", "form_token": f.oldToken("tahsin")})
	require.Equal(t, fiber.StatusUnprocessableEntity, code)
	var out struct {
		Jadwal []struct {
			Kode       string `json:"kode"`
			SisaAkhwat int64  `json:"sisa_akhwat"`
		} `json:"jadwal"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	require.NotEmpty(t, out.Jadwal)
	for _, j := range out.Jadwal {
		if j.Kode == "tst1" {
			require.Zero(t, j.SisaAkhwat)
		}
	}
}
