package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const pwaManifestA = `{
 "src/main.ts": {"file": "assets/main-AAA.js", "isEntry": true, "css": ["assets/main-AAA.css"]},
 "src/pages/app/tasbih/Index.svelte": {"file": "assets/Index-T1.js", "isDynamicEntry": true, "imports": ["src/main.ts", "_AppLayout-L1.js"], "css": ["assets/Index-T1.css"]},
 "src/pages/app/tasbih/Dashboard.svelte": {"file": "assets/Dashboard-T2.js", "isDynamicEntry": true, "imports": ["src/main.ts", "_AppLayout-L1.js", "_flame-F1.js"]},
 "src/pages/app/Profile.svelte": {"file": "assets/Profile-P1.js", "isDynamicEntry": true, "imports": ["_AppLayout-L1.js"]},
 "src/pages/app/SantriList.svelte": {"file": "assets/SantriList-S1.js", "isDynamicEntry": true, "imports": ["_AppLayout-L1.js"]},
 "_AppLayout-L1.js": {"file": "assets/AppLayout-L1.js", "imports": ["_Logo-G1.js"], "css": ["assets/AppLayout-L1.css"]},
 "_Logo-G1.js": {"file": "assets/Logo-G1.js"},
 "_flame-F1.js": {"file": "assets/flame-F1.js"}
}`

func pwaService(t *testing.T, manifest string) *PWAService {
	t.Helper()
	p := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(p, []byte(manifest), 0o644))
	return NewPWAService(NewAssetService(p, "", false))
}

func TestServiceWorkerPrecacheTasbih(t *testing.T) {
	js := pwaService(t, pwaManifestA).ServiceWorker()
	for _, f := range []string{
		"/assets/main-AAA.js", "/assets/main-AAA.css",
		"/assets/Index-T1.js", "/assets/Index-T1.css", "/assets/Dashboard-T2.js", "/assets/Profile-P1.js",
		"/assets/AppLayout-L1.js", "/assets/AppLayout-L1.css", "/assets/Logo-G1.js", "/assets/flame-F1.js",
		"/public/offline.html", "/public/tasbih.webmanifest", "/public/icons/tasbih-192.png",
	} {
		require.Contains(t, js, `"`+f+`"`, f)
	}
	require.NotContains(t, js, "SantriList", "halaman staf tidak di-precache")
	require.NotContains(t, js, "__VERSION__")
	require.NotContains(t, js, "__PRECACHE__")
	require.Equal(t, 1, strings.Count(js, `"/assets/AppLayout-L1.js"`), "tanpa duplikat")
}

func TestServiceWorkerVersiBerubahSaatAsetBerubah(t *testing.T) {
	a := pwaService(t, pwaManifestA)
	b := pwaService(t, strings.ReplaceAll(pwaManifestA, "Index-T1.js", "Index-T9.js"))
	require.Equal(t, a.Versi(), pwaService(t, pwaManifestA).Versi(), "deterministik")
	require.NotEqual(t, a.Versi(), b.Versi())
	require.Contains(t, a.ServiceWorker(), a.Versi())
}

func TestServiceWorkerTanpaManifest(t *testing.T) {
	js := NewPWAService(NewAssetService(filepath.Join(t.TempDir(), "x.json"), "", false)).ServiceWorker()
	require.Contains(t, js, `"/public/offline.html"`)
}

func TestManifestUntukHalaman(t *testing.T) {
	require.Equal(t, PWAManifestTasbih, ManifestUntuk("app/tasbih/Index", "cs"))
	require.Equal(t, PWAManifestTasbih, ManifestUntuk("app/tasbih/Dashboard", "user"))
	require.Equal(t, PWAManifestTasbih, ManifestUntuk("app/Profile", "user"))
	require.Equal(t, PWAManifestSitus, ManifestUntuk("app/Profile", "cs"))
	require.Equal(t, PWAManifestSitus, ManifestUntuk("app/SantriList", "super_admin"))
	require.Equal(t, PWAManifestSitus, ManifestUntuk("auth/Login", ""))
}
