package services

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// PWA aplikasi Tasbih: manifest per halaman dan service worker yang daftar
// precache-nya dibangkitkan dari manifest Vite (versi berganti tiap build).

const (
	PWAManifestTasbih = "/public/tasbih.webmanifest"
	PWAManifestSitus  = "/public/site.webmanifest"
)

//go:embed pwa_sw.js
var pwaSWTemplate string

// Halaman yang ikut di-precache (beserta import & CSS-nya).
var pwaHalaman = []string{
	"src/pages/app/tasbih/Index.svelte",
	"src/pages/app/tasbih/Dashboard.svelte",
	"src/pages/app/Profile.svelte",
}

var pwaStatis = []string{
	"/public/offline.html",
	PWAManifestTasbih,
	"/public/icons/tasbih-192.png",
	"/public/icons/tasbih-512.png",
	"/public/icons/tasbih-maskable-512.png",
	"/public/icons/tasbih-apple-180.png",
	"/public/favicon.svg",
}

type PWAService struct {
	assets *AssetService
}

func NewPWAService(assets *AssetService) *PWAService {
	return &PWAService{assets: assets}
}

// ManifestUntuk memilih web manifest untuk sebuah halaman Inertia.
// Halaman tasbih (semua role) dan Profil milik role user memakai manifest Tasbih.
func ManifestUntuk(component, role string) string {
	if strings.HasPrefix(component, "app/tasbih/") || (component == "app/Profile" && role == "user") {
		return PWAManifestTasbih
	}
	return PWAManifestSitus
}

// Precache: aset statis + entry utama + halaman tasbih beserta seluruh import & CSS.
func (s *PWAService) Precache() []string {
	entries := s.assets.Entries()
	set := map[string]bool{}
	for _, f := range pwaStatis {
		set[f] = true
	}
	var kunjungi func(key string)
	dilihat := map[string]bool{}
	kunjungi = func(key string) {
		if dilihat[key] {
			return
		}
		dilihat[key] = true
		e, ok := entries[key]
		if !ok {
			return
		}
		if e.File != "" {
			set["/"+e.File] = true
		}
		for _, c := range e.CSS {
			set["/"+c] = true
		}
		for _, imp := range e.Imports {
			kunjungi(imp)
		}
	}
	kunjungi("src/main.ts")
	kunjungi("src/app.css")
	for _, h := range pwaHalaman {
		kunjungi(h)
	}
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// Versi: hash daftar precache — berubah setiap aset ber-hash berubah.
func (s *PWAService) Versi() string {
	h := sha256.Sum256([]byte(strings.Join(s.Precache(), "\n") + "\n" + pwaSWTemplate))
	return hex.EncodeToString(h[:])[:12]
}

// ServiceWorker mengembalikan isi /sw.js.
func (s *PWAService) ServiceWorker() string {
	daftar, _ := json.Marshal(s.Precache())
	js := strings.ReplaceAll(pwaSWTemplate, "__VERSION__", s.Versi())
	return strings.ReplaceAll(js, "__PRECACHE__", string(daftar))
}
