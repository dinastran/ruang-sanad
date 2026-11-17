# Tasbih PWA — Design

Tanggal: 2026-11-17 · Status: disetujui user (AskUserQuestion, 2026-11-17)

## Intent
Tasbih Digital bisa di-install di HP dan terasa seperti aplikasi mobile: ikon di layar utama, tanpa address bar, navigasi bawah, dan tetap bisa dibuka serta dipakai tanpa internet.

## Keputusan user
- **Cakupan**: aplikasi khusus Tasbih ("Tasbih", nama panjang "Tasbih Ruang Sanad"). Isinya Tasbih, Progres (dashboard tasbih), dan Profil. Dashboard staf tidak berubah.
- **Offline**: aplikasi harus bisa dibuka dan dipakai tanpa internet setelah pernah login online.
- **Ikon**: ikon tasbih baru (untaian butir), warna brand.
- **Tambahan**: hanya banner "Pasang aplikasi". Notifikasi pengingat dan shortcut ikon tidak termasuk.

## Approaches
| | Pendekatan | Trade-off |
|---|---|---|
| **A (dipilih)** | Service worker ditulis sendiri, dilayani Go di `/sw.js`. Daftar precache dan versi dibangkitkan dari manifest Vite. | Tanpa dependency. Kontrol penuh atas cache halaman Inertia dan pembersihan cache per user. |
| B | `vite-plugin-pwa`/Workbox | Ada dependency baru, dan aturan untuk halaman yang dirender server tetap harus ditulis manual. |

## Desain
1. **Manifest**
   - `public/tasbih.webmanifest`: `id` `/app/tasbih`, `start_url` `/app/tasbih?source=pwa`, `scope` `/`, `display` `standalone`, `theme_color` brand-600, `background_color` putih, ikon 192/512 (any) dan 512 (maskable).
   - Halaman `app/tasbih/*` dan `app/Profile` milik role user memakai manifest ini. Halaman lain memakai `public/site.webmanifest` yang diperbaiki (nama "Ruang Sanad", path ikon benar).
   - Meta iOS: `apple-mobile-web-app-capable`, `apple-mobile-web-app-title`, `apple-touch-icon` 180, `viewport-fit=cover`.
2. **Service worker** (`/sw.js`, `Cache-Control: no-cache`)
   - Versi = hash daftar aset. Setiap deploy menghasilkan versi baru dan cache lama dihapus saat `activate`.
   - Precache: main JS/CSS, chunk halaman tasbih beserta import dan CSS-nya, ikon, manifest, dan `/public/offline.html`.
   - `/assets/*` dan font Google memakai cache-first (file ber-hash dan immutable).
   - Navigasi dan GET Inertia ke `/app/tasbih` dan `/app/tasbih/dashboard` memakai network-first dan disimpan di cache **per user** (`pages-u<id>`; id dikirim halaman lewat `postMessage`). Hanya respons 200 yang tidak di-redirect yang disimpan.
   - Saat offline: dipakai cache halaman. Kalau tidak ada, navigasi diarahkan ke `offline.html`.
   - `POST /logout` menghapus semua cache halaman. Klien juga menghapusnya sebelum logout.
   - Request selain GET (termasuk `/app/tasbih/sync`) tidak pernah di-cache.
   - Pembaruan: SW baru menunggu, lalu muncul toast "Versi baru tersedia · Muat ulang". Tombolnya mengirim `SKIP_WAITING`, lalu halaman di-reload.
3. **Tampilan aplikasi** (role user, atau mode standalone di halaman tasbih)
   - AppLayout mendelegasikan ke `TasbihShell`: header ringkas (logo, judul, menu akun berisi mode gelap dan keluar) dan navigasi bawah Tasbih / Progres / Profil.
   - Memperhitungkan safe-area, `overscroll-behavior: none`, dan kolom konten maksimal `max-w-xl`.
4. **Banner pasang**
   - Android/Chrome: memakai `beforeinstallprompt`.
   - iOS Safari: panduan "Bagikan → Tambah ke Layar Utama".
   - Tidak tampil kalau sudah standalone. Kalau ditutup, tidak tampil 30 hari (`localStorage`). Hilang setelah event `appinstalled`.

## Non-goals
Push notification, shortcut ikon, background sync API (antrean sudah ditangani saat halaman dibuka atau online), dan halaman staf.

## Testing
- Go: `/sw.js` (content-type, no-cache, versi berubah saat aset berubah, daftar precache memuat chunk tasbih); pemilihan manifest di root HTML.
- E2E Playwright build produksi:
  - Manifest dan ikon bisa diambil, SW mengontrol halaman.
  - Reload offline `/app/tasbih` tetap tampil dan bisa menghitung, lalu online tersinkron.
  - Logout menghapus cache halaman.
  - Di lebar 390px tampil layout aplikasi (tanpa sidebar), banner pasang muncul pada event sintetis dan bisa ditutup.
