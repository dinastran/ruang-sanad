package session

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// catatUAberubah hanya MENCATAT bila browser/OS berbeda dari saat login; sesi
// tidak dihapus. User-Agent mudah dipalsukan sehingga perlindungannya kecil,
// sementara mode HP di DevTools dan Chrome iOS (request service worker memakai
// UA "Safari") membuat user asli ter-logout. Keamanan sesi bersandar pada
// cookie HttpOnly + Secure + SameSite.
func catatUAberubah(c *fiber.Ctx, tersimpan string) {
	if tersimpan == "" || !isPageRequest(c) {
		return
	}
	if a, b := famili(tersimpan), famili(c.Get("User-Agent")); a != b {
		slog.Info("sesi dipakai dari browser/OS berbeda (dibiarkan)", "tersimpan", a, "sekarang", b)
	}
}

// famili meringkas User-Agent menjadi "sistem-operasi/browser" tanpa nomor versi
// (dipakai untuk log).
func famili(ua string) string {
	return keluargaOS(ua) + "/" + keluargaBrowser(ua)
}

func keluargaOS(ua string) string {
	switch {
	case strings.Contains(ua, "Windows"):
		return "windows"
	case strings.Contains(ua, "CrOS"):
		return "chromeos"
	// iPad "situs desktop" melaporkan Macintosh — satu keluarga Apple.
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iPod"), strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		return "apple"
	// Chrome Android "situs desktop" melaporkan X11; Linux — satu keluarga.
	case strings.Contains(ua, "Android"), strings.Contains(ua, "Linux"), strings.Contains(ua, "X11"):
		return "linux"
	}
	return "lain"
}

func keluargaBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "Edg/"), strings.Contains(ua, "EdgA/"), strings.Contains(ua, "EdgiOS/"):
		return "edge"
	case strings.Contains(ua, "OPR/"), strings.Contains(ua, "OPiOS/"):
		return "opera"
	case strings.Contains(ua, "SamsungBrowser/"):
		return "samsung"
	case strings.Contains(ua, "Firefox/"), strings.Contains(ua, "FxiOS/"):
		return "firefox"
	case strings.Contains(ua, "CriOS/"), strings.Contains(ua, "Chrome/"):
		return "chrome"
	case strings.Contains(ua, "Safari/"):
		return "safari"
	}
	return "lain"
}
