package session

import "strings"

// famili meringkas User-Agent menjadi "sistem-operasi/browser" tanpa nomor versi.
// Pemeriksaan sesi membandingkan famili, bukan teks utuh, supaya update browser
// atau mode "situs desktop" tidak membuat user logout, sementara cookie yang
// dipakai dari browser/OS lain tetap ditolak.
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
