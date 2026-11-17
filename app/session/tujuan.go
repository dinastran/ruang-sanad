package session

import (
	"net/url"
	"path"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CookieTujuanLogin menyimpan halaman yang sedang dibuka saat sesi berakhir,
// agar setelah login ulang user kembali ke sana (bukan ke beranda role).
const CookieTujuanLogin = "login_return"

const tujuanMaks = 512

// SimpanTujuan dipanggil saat request halaman ditolak karena belum login.
// Hanya navigasi halaman (GET HTML / Inertia) ke /app atau /admin yang disimpan.
func (s *Store) SimpanTujuan(c *fiber.Ctx) {
	if c.Method() != fiber.MethodGet || !isPageRequest(c) {
		return
	}
	tujuan := tujuanAman(c.OriginalURL())
	if tujuan == "" {
		return
	}
	c.Cookie(&fiber.Cookie{
		Name:     CookieTujuanLogin,
		Value:    url.QueryEscape(tujuan),
		Path:     "/",
		HTTPOnly: true,
		Secure:   s.secure,
		SameSite: "Lax",
		MaxAge:   15 * 60,
	})
}

// AmbilTujuan mengambil (dan menghapus) halaman tujuan yang aman, atau "".
func (s *Store) AmbilTujuan(c *fiber.Ctx) string {
	raw := c.Cookies(CookieTujuanLogin)
	if raw == "" {
		return ""
	}
	c.Cookie(&fiber.Cookie{Name: CookieTujuanLogin, Value: "", Path: "/", HTTPOnly: true, Secure: s.secure, SameSite: "Lax", MaxAge: -1})
	dec, err := url.QueryUnescape(raw)
	if err != nil {
		return ""
	}
	return tujuanAman(dec)
}

// tujuanAman menerima hanya path relatif di dalam /app atau /admin (anti open-redirect).
func tujuanAman(raw string) string {
	if raw == "" || len(raw) > tujuanMaks || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil {
		return ""
	}
	p := u.Path
	if path.Clean(p) != p {
		return ""
	}
	if !(p == "/app" || strings.HasPrefix(p, "/app/") || p == "/admin" || strings.HasPrefix(p, "/admin/")) {
		return ""
	}
	if u.RawQuery != "" {
		return p + "?" + u.RawQuery
	}
	return p
}
