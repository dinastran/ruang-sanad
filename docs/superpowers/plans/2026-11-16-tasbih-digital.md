# Tasbih Digital Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tasbih digital pribadi (counter beranimasi, paket wirid, dzikir custom, target, streak, badge/level), bisa dipakai offline, plus dashboard grafik dan riwayat.

**Architecture:** Klien mencatat setiap sesi dzikir di IndexedDB dengan UUID dan mengirimnya dalam batch ke `POST /app/tasbih/sync`. Server melakukan insert idempoten ke `tasbih_sesi` dengan kolom `tanggal_wib`/`jam_wib`, lalu semua statistik dihitung lewat query sqlc yang di-GROUP BY. Alurnya Handler (`app/handlers/tasbih.go`) → `TasbihService` → sqlc (`queries/tasbih.sql`).

**Tech Stack:** Go Fiber, sqlc v1.31.1, goose, SQLite, Svelte 5 + Inertia, Tailwind v4 tokens, IndexedDB, chart SVG buatan sendiri.

**Spec:** `docs/superpowers/specs/2026-11-16-tasbih-digital-design.md`. Baca seluruhnya. Nilai di plan ini disalin dari spec.

## Global Constraints

- Handler → Service → Query. Hanya `app/queries` (hasil sqlc) yang mengeksekusi SQL. Test boleh memakai `db.Exec`.
- Jangan edit `app/queries/*.go` hasil generate secara manual. Regenerate: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`.
- Satu migrasi = satu tabel, nomor 0109–0115. Seed ikut di migrasi tabelnya.
- `go.sum` gitignored. Jalankan `go mod tidy` sekali sebelum `go test`.
- Jangan jalankan dev server. Untuk verifikasi E2E, build lalu jalankan binary sendiri di port bebas (Task 10).
- Batas hari dan jam memakai `wib` (`app/services/santri_status.go:55`).
- Validasi: `jumlah` 1..100000, target 1..100000, `target_harian` default 100, latin wajib ≤300 karakter, arab ≤500 karakter, maksimal 50 dzikir custom aktif per user, batch sync ≤100 sesi, `mulai_at` ≤ now+5 menit dan ≥ now−400 hari. Tidak ada batas laju.
- Semua route `/app/tasbih*` tanpa middleware role. Semua query wajib di-scope dengan `user_id` sesi.
- `fetch()` ke `/app/*` wajib menyertakan header `X-XSRF-TOKEN: getCSRFToken()`. Form memakai `router.*`. Link internal memakai `use:inertia`.
- Warna hanya dari token `@theme` (`brand-*`, `secondary-*`, `neutral-*`) di `frontend/src/app.css`. Tidak boleh hex hardcode.
- Svelte 5: `$derived` untuk state turunan, `$effect` hanya untuk side effect.
- Copy UI dalam Bahasa Indonesia, netral. Nama level berupa "Level N", tanpa gelar spiritual.
- Hormati `prefers-reduced-motion`.

## Review Focus

1. **Sesi melewati tengah malam WIB / zona waktu perangkat bukan WIB.** `tanggal_wib` harus dihitung dari `mulai_at` dalam WIB oleh server. Test `TestSyncTanggalWIBBoundary` (Task 3).
2. **Batch sync terkirim dua kali (retry setelah timeout).** Total tidak boleh dobel dan ID harus kembali dikonfirmasi di `diterima`. Test `TestSyncIdempoten` (Task 3).
3. **UUID sesi milik user lain dikirim ulang.** Harus ditolak tanpa menimpa data. Test `TestSyncIDMilikUserLain` (Task 3).
4. **Tab ditutup di tengah hitungan atau di tengah paket.** Setelah dibuka lagi, hitungan dan posisi paket dipulihkan. Cek E2E langkah 3 (Task 10).
5. **Hari ini belum mencapai target.** Streak tidak boleh langsung nol, harus dihitung sampai kemarin. Test `TestStreakHariIniBelumTercapai` (Task 4).

---

## File Structure

| File | Tanggung jawab |
|---|---|
| `migrations/0109_create_tasbih_dzikir_table.sql` … `0115_create_tasbih_target_user_table.sql` | Skema + seed |
| `queries/tasbih.sql` | Semua query tasbih (sqlc) |
| `app/models/tasbih.go` | DTO dan konstanta level/badge |
| `app/services/tasbih.go` | Katalog, dzikir custom, target, pengaturan |
| `app/services/tasbih_sync.go` | Validasi dan insert sesi |
| `app/services/tasbih_stats.go` | Ringkasan, streak, level, grafik, riwayat |
| `app/services/tasbih_badge.go` | Evaluasi badge |
| `app/services/tasbih_test.go`, `tasbih_sync_test.go`, `tasbih_stats_test.go`, `tasbih_badge_test.go` | Test service |
| `app/handlers/tasbih.go` (+ `tasbih_test.go`) | HTTP |
| `routes/web.go`, `cmd/laju-go/main.go`, `app/handlers/dashboard.go`, `app/handlers/auth.go`, `frontend/src/layouts/AppLayout.svelte` | Wiring, redirect, menu |
| `frontend/src/lib/tasbih/types.ts` | Tipe TS bersama |
| `frontend/src/lib/tasbih/db.ts` | IndexedDB |
| `frontend/src/lib/tasbih/sync.ts` | Antrean dan sync |
| `frontend/src/lib/tasbih/counter.svelte.ts` | Mesin sesi/paket (rune class) |
| `frontend/src/lib/tasbih/feedback.ts` | Getar, suara, wake lock |
| `frontend/src/pages/app/tasbih/Index.svelte` | Halaman counter |
| `frontend/src/pages/app/tasbih/Dashboard.svelte` | Dashboard |
| `frontend/src/components/tasbih/*.svelte` | ProgressRing, BeadStrip, DzikirPicker, DzikirFormSheet, BadgeToast, SyncIndicator, DailyBar, Heatmap, DonutKomposisi, HourBar, BadgeGrid, RiwayatList, KpiCard |

---

### Task 1: Migrasi + seed

**Files:**
- Create: `migrations/0109_create_tasbih_dzikir_table.sql`, `0110_create_tasbih_paket_table.sql`, `0111_create_tasbih_paket_item_table.sql`, `0112_create_tasbih_sesi_table.sql`, `0113_create_tasbih_pengaturan_table.sql`, `0114_create_tasbih_badge_user_table.sql`, `0115_create_tasbih_target_user_table.sql`
- Test: `app/services/tasbih_test.go`

**Interfaces:**
- Produces: tabel sesuai spec §6 dan 0115. Seed 12 dzikir (slug di spec §5, `urutan` 1..12 sesuai urutan tabel, `user_id` NULL) dan 3 paket dengan slug `badah-shalat` (urutan item: subhanallah 33, alhamdulillah 33, allahu-akbar 33, tahlil-lengkap 1), `tasbih-fatimah` (33, 33, 34), `istighfar-100` (istighfar 100). Item seed memakai subselect `(SELECT id FROM tasbih_dzikir WHERE slug=...)`.
- Produces test helper di `tasbih_test.go`: `newTasbihTestDB(t) (*sql.DB, *TasbihService)` (pola `guru_directory_test.go`: sqlite memory, `foreign_keys=ON`, goose up) dan `seedUser(t, db, role string) int64`.

- [ ] **Step 1:** Tulis `TestTasbihSeed`: setelah migrasi, `COUNT(*) FROM tasbih_dzikir WHERE user_id IS NULL` = 12, paket = 3, item `badah-shalat` = 4 dengan target item ke-4 = 1, item ke-3 `tasbih-fatimah` = 34. Setiap dzikir bawaan punya `teks_arab`, `arti`, dan `rujukan` tidak kosong.
- [ ] **Step 2:** `go mod tidy && go test ./app/services -run TestTasbihSeed` → FAIL (tabel tidak ada).
- [ ] **Step 3:** Tulis 7 migrasi dengan blok goose `-- +goose Up` / `-- +goose Down`. Teks Arab harus berharakat lengkap. Cocokkan dengan lafaz di riwayat yang dirujuk (cek ulang kata per kata). Arti ditulis dalam Bahasa Indonesia baku. `tasbih_sesi` mengikuti spec §6 persis, termasuk dua index.
- [ ] **Step 4:** Jalankan test → PASS. Lalu `go test ./...` untuk memastikan tidak ada test lain yang rusak karena migrasi.
- [ ] **Step 5:** Commit `feat(tasbih): migrations and seed`.

### Task 2: Query sqlc + service katalog, custom, target

**Files:**
- Create: `queries/tasbih.sql`, `app/models/tasbih.go`, `app/services/tasbih.go`
- Generate: `app/queries/tasbih.sql.go`
- Test: `app/services/tasbih_test.go`

**Interfaces:**
- Produces (models):
  ```go
  type TasbihDzikir struct { ID int64; Custom bool; Arab, Latin, Arti, Rujukan string; Target int64 /*efektif*/; TargetDefault int64 } // json snake_case
  type TasbihPaketItem struct { DzikirID int64; Target int64 }
  type TasbihPaket struct { ID int64; Slug, Nama, Deskripsi, Rujukan string; Items []TasbihPaketItem }
  type TasbihDzikirInput struct { Arab, Latin, Arti string; Target int64 }
  ```
- Produces (service):
  ```go
  func NewTasbihService(q *queries.Querier) *TasbihService
  func (s *TasbihService) Katalog(ctx, userID int64) (dzikir []models.TasbihDzikir, paket []models.TasbihPaket, err error) // bawaan+custom aktif, urut urutan lalu id
  func (s *TasbihService) BuatDzikir(ctx, userID int64, in models.TasbihDzikirInput) (int64, error)
  func (s *TasbihService) UbahDzikir(ctx, userID, dzikirID int64, in models.TasbihDzikirInput) error
  func (s *TasbihService) ArsipkanDzikir(ctx, userID, dzikirID int64) error
  func (s *TasbihService) SetTargetDzikir(ctx, userID, dzikirID, target int64) error // upsert 0115; dzikir harus bawaan/milik user
  func (s *TasbihService) TargetHarian(ctx, userID int64) (int64, error) // default 100 bila belum ada baris
  func (s *TasbihService) SetTargetHarian(ctx, userID, target int64) error
  var ErrTasbihValidasi, ErrTasbihTidakDitemukan error // dibungkus fmt.Errorf("%w: ...")
  ```

- [ ] **Step 1:** Tulis test-test berikut:
  - `TestKatalogGabungBawaanDanCustom`: user A membuat 1 custom; Katalog A = 13 item dan Katalog B = 12.
  - `TestTargetEfektifOverride`: `SetTargetDzikir` subhanallah ke 99 membuat Target = 99 dan TargetDefault = 33.
  - `TestUbahDzikirMilikLain`: menghasilkan `ErrTasbihTidakDitemukan`. Hal yang sama berlaku untuk mengubah dzikir bawaan.
  - `TestBuatDzikirValidasi`: latin kosong, latin 301 karakter, target 0, dan target 100001 semuanya menghasilkan `ErrTasbihValidasi`. Custom ke-51 ditolak.
  - `TestArsipkanDzikir`: dzikir hilang dari Katalog, tetapi baris tetap ada.
  - `TestTargetHarianDefault`: menghasilkan 100. Setelah Set 300, menghasilkan 300. Set 0 ditolak.
- [ ] **Step 2:** Jalankan → FAIL (compile).
- [ ] **Step 3:** Tulis query di `queries/tasbih.sql`, lalu jalankan sqlc generate. Implementasikan models + service. Query ownership harus memakai `WHERE id=? AND user_id=?`, sehingga dzikir bawaan otomatis tidak ditemukan.
- [ ] **Step 4:** `go test ./app/services -run 'Tasbih|Katalog|Dzikir|Target'` → PASS.
- [ ] **Step 5:** Commit `feat(tasbih): catalog, custom dzikir, targets`.

### Task 3: Sync sesi

**Files:**
- Create: `app/services/tasbih_sync.go`, `app/services/tasbih_sync_test.go`
- Modify: `queries/tasbih.sql` (+ regenerate)

**Interfaces:**
- Consumes: Task 2 service and errors.
- Produces:
  ```go
  type TasbihSesiInput struct { ID string; DzikirID int64; PaketID *int64; PaketRunID *string; Jumlah, Target int64; MulaiAt, SelesaiAt time.Time } // json: id,dzikir_id,paket_id,paket_run_id,jumlah,target,mulai_at,selesai_at (RFC3339)
  type TasbihSesiDitolak struct { ID, Alasan string }
  type TasbihSyncHasil struct { Diterima []string; Ditolak []TasbihSesiDitolak; BadgeBaru []TasbihBadge; HariIni TasbihHariIni }
  func (s *TasbihService) Sync(ctx, userID int64, sesi []models.TasbihSesiInput, now time.Time) (models.TasbihSyncHasil, error) // error hanya bila batch > 100 atau error DB
  ```
  `TasbihBadge` dan `TasbihHariIni` didefinisikan di Task 4/5. Untuk sementara deklarasikan dulu di models, isi menyusul.

- [ ] **Step 1:** Tulis test-test berikut:
  - `TestSyncIdempoten`: kirim batch 2 sesi dua kali. Kedua respons `Diterima` = 2 ID, dan `COUNT` = 2.
  - `TestSyncIDMilikUserLain`: ID yang sudah dipakai user A, lalu dikirim B, masuk `Ditolak`. Baris A tidak berubah.
  - `TestSyncValidasi`: UUID tidak valid, jumlah 0, jumlah 100001, selesai < mulai, mulai = now+6m, mulai = now−401d, dzikir custom user lain, dan paket_id tidak ada. Masing-masing masuk `Ditolak` dengan alasan tidak kosong. Sesi valid di batch yang sama tetap diterima.
  - `TestSyncDzikirDiarsipkanTetapDiterima`.
  - `TestSyncTanggalWIBBoundary`: mulai `2026-11-15T16:59:00Z` menghasilkan tanggal_wib `2026-11-15` jam 23. Mulai `17:01Z` menghasilkan `2026-11-16` jam 0.
  - `TestSyncPaketTargetTidakCocok`: paket badah-shalat, dzikir tahlil-lengkap, target 100 (item = 1). Sesi diterima dengan `paket_id` dan `paket_run_id` NULL.
  - `TestSyncBatchTerlaluBesar`: 101 sesi menghasilkan error.
  - `selesai_target` = 1 bila jumlah ≥ target.
- [ ] **Step 2:** Jalankan → FAIL.
- [ ] **Step 3:** Implementasi. Insert memakai `INSERT ... ON CONFLICT(id) DO NOTHING`. Bila 0 baris ter-insert, cek pemilik lewat `GetTasbihSesiOwner(id)`: pemilik sama berarti diterima, pemilik lain berarti ditolak dengan alasan "id tidak valid". Jalankan semua dalam satu transaksi (`q.WithTx` bila pola itu ada di Querier, jika tidak per-statement). Panggilan `EvaluasiBadge` ditambahkan di Task 5. Untuk sementara `BadgeBaru` = kosong dan `HariIni` dari Task 4 (isi nol sampai Task 4 selesai).
- [ ] **Step 4:** Jalankan → PASS.
- [ ] **Step 5:** Commit `feat(tasbih): idempotent session sync`.

### Task 4: Statistik (ringkasan, streak, level, grafik, riwayat)

**Files:**
- Create: `app/services/tasbih_stats.go`, `app/services/tasbih_stats_test.go`
- Modify: `queries/tasbih.sql`, `app/models/tasbih.go`

**Interfaces:**
- Produces:
  ```go
  var TasbihLevelAmbang = []int64{0,100,500,1500,4000,10000,25000,50000,100000,250000} // index+1 = level
  type TasbihLevel struct { Level int; Total, AmbangIni, AmbangBerikut int64 /*0 bila max*/ }
  type TasbihHariIni struct { Tanggal string; Total, Target int64 }
  type TasbihStreak struct { Saat, Terpanjang int; HariIniTercapai bool }
  type TasbihTitikHari struct { Tanggal string; Total int64 }
  type TasbihKomposisi struct { DzikirID int64; Latin string; Total int64 } // top 6 + {DzikirID:0, Latin:"Lainnya"}
  type TasbihPaketPutaran struct { PaketID int64; Nama string; Selesai int64 }
  type TasbihDashboard struct { Rentang int; HariIni TasbihHariIni; Streak TasbihStreak; Level TasbihLevel; RataRata float64; Harian []TasbihTitikHari /*lengkap tiap hari dalam rentang, nol diisi*/; TotalRentang, TotalRentangSebelum int64; Heatmap []TasbihTitikHari /*365 hari*/; Komposisi []TasbihKomposisi; PerJam [24]int64; Paket []TasbihPaketPutaran }
  type TasbihRiwayatItem struct { ID string; Tanggal string; Jam string /*HH:MM WIB*/; DzikirLatin string; Jumlah, Target int64; DurasiDetik int64; PaketNama string }
  func HitungLevel(total int64) models.TasbihLevel
  func HitungStreak(hariTercapai []string /*tanggal_wib asc*/, hariIni string) models.TasbihStreak // murni, tanpa DB
  func (s *TasbihService) HariIni(ctx, userID int64, now time.Time) (models.TasbihHariIni, error)
  func (s *TasbihService) Dashboard(ctx, userID int64, rentang int /*7|30|90|365, lainnya→30*/, now time.Time) (models.TasbihDashboard, error)
  func (s *TasbihService) Riwayat(ctx, userID int64, halaman int, dzikirID *int64) (items []models.TasbihRiwayatItem, adaLagi bool, err error) // 20/halaman, terbaru dulu
  func (s *TasbihService) JumlahPutaranPaket(ctx, userID int64, paketID *int64, sejak *string) (int64, error) // dipakai Task 5
  ```
- Definisi streak (spec §12): hari tercapai = total harian ≥ target harian *saat ini*. `Saat` dihitung mundur dari hari ini bila hari ini tercapai, jika tidak dari kemarin.
- Putaran paket selesai (spec §8): ada sesi untuk setiap `paket_item` dengan `paket_run_id` yang sama, `dzikir_id` cocok, dan `selesai_target=1`. Implementasi lewat query GROUP BY `paket_run_id` HAVING `COUNT(DISTINCT dzikir_id) = jumlah item distinct paket`.

- [ ] **Step 1:** Tulis test-test berikut:
  - `TestHitungLevel`: 0 → L1, 99 → L1, 100 → L2, 250000 → L10 dengan AmbangBerikut 0.
  - `TestStreakHariIniBelumTercapai`: hari tercapai 13, 14, 15; hari ini 16 belum tercapai. Hasil Saat = 3, HariIniTercapai false.
  - `TestStreakBolong`: hari 10, 11, 13, 14, 15, 16; hari ini 16. Hasil Saat = 4, Terpanjang = 4.
  - `TestStreakPutus`: terakhir tercapai tanggal 14, hari ini 16. Hasil Saat = 0.
  - `TestDashboardAgregat`: sync sesi di 3 hari dan 2 jam berbeda. Cek `Harian` panjangnya = rentang dan nol terisi, `PerJam[jam]`, `Komposisi` (8 dzikir menghasilkan 6 + "Lainnya"), `TotalRentangSebelum`, `RataRata` = TotalRentang/rentang.
  - `TestPutaranPaket`: satu run lengkap (4 item selesai) dan satu run kurang satu item menghasilkan Selesai = 1.
  - `TestRiwayatPaginasi`: 25 sesi menghasilkan halaman 1 = 20 dengan adaLagi true dan halaman 2 = 5. Filter dzikir berfungsi. Sesi user lain tidak ikut.
- [ ] **Step 2:** Jalankan → FAIL.
- [ ] **Step 3:** Implementasi query (GROUP BY `tanggal_wib`, `jam_wib`, `dzikir_id`) dan perakitan di service. Isi `HariIni` pada hasil `Sync` (Task 3).
- [ ] **Step 4:** Jalankan → PASS.
- [ ] **Step 5:** Commit `feat(tasbih): stats, streak, level, history`.

### Task 5: Badge

**Files:**
- Create: `app/services/tasbih_badge.go`, `app/services/tasbih_badge_test.go`
- Modify: `queries/tasbih.sql`, `app/models/tasbih.go`, `app/services/tasbih_sync.go`, `app/services/tasbih.go` (panggil dari BuatDzikir dan SetTargetHarian)

**Interfaces:**
- Produces:
  ```go
  type TasbihBadgeDef struct { Kode, Nama, Syarat, Ikon string } // Ikon = nama ikon lucide
  var TasbihBadges []TasbihBadgeDef // urutan tampil, kode persis spec §12
  type TasbihBadge struct { Kode, Nama, Syarat, Ikon string; DiraihAt *time.Time }
  func (s *TasbihService) EvaluasiBadge(ctx, userID int64, now time.Time) ([]models.TasbihBadge /*baru*/, error)
  func (s *TasbihService) DaftarBadge(ctx, userID int64) ([]models.TasbihBadge, error) // semua def, DiraihAt nil bila terkunci
  func (s *TasbihService) BuatDzikir(...) // signature tetap; badge baru diambil handler lewat EvaluasiBadge
  ```
  Keputusan: `BuatDzikir`/`SetTargetHarian` tidak mengembalikan badge. Handler memanggil `EvaluasiBadge` setelahnya (satu jalur). `Sync` memanggil `EvaluasiBadge` di dalam.
- Kode badge: `langkah-pertama`, `target-pertama`, `paket-pertama`, `total-1k`, `total-10k`, `total-100k`, `total-1m`, `streak-3`, `streak-7`, `streak-30`, `streak-100`, `badah-shalat-35`, `subuh-7`, `custom-pertama`, `istighfar-1k`. Kode spec `paket-ba'da-shalat-35` diganti `badah-shalat-35` (tanpa apostrof). `subuh-7` = 7 `tanggal_wib` berbeda dengan `jam_wib` 3..5. `istighfar-1k` = total slug `istighfar` + `istighfar-taubat`. `target-pertama` = Terpanjang ≥ 1. `badah-shalat-35` = ≥35 putaran selesai di paket `badah-shalat` dengan `tanggal_wib` ≥ hari ini−6.

- [ ] **Step 1:** Tulis test-test berikut:
  - `TestBadgeLangkahPertama`: sync 1 sesi menghasilkan `BadgeBaru` berisi `langkah-pertama`. Sync kedua tidak mengulangnya.
  - `TestBadgeTotal1k`.
  - `TestBadgeStreak3`.
  - `TestBadgeSubuh7`: 7 hari jam 04 WIB = dapat; 6 hari = belum.
  - `TestBadgeCustomPertama`: via `BuatDzikir` + `EvaluasiBadge`.
  - `TestBadgeTargetPertamaSetelahTurunkanTarget`: total hari ini 50, target 100 belum dapat. `SetTargetHarian(50)` + Evaluasi = dapat.
  - `TestBadgeIstighfar1k`: 600 + 400 dari dua slug.
  - `TestBadgeTidakDicabut`.
  - `TestDaftarBadge`: panjang = 15, urutan sama dengan `TasbihBadges`.
- [ ] **Step 2:** Jalankan → FAIL.
- [ ] **Step 3:** Implementasi. Hitung statistik yang dibutuhkan sekali per evaluasi, lalu `INSERT OR IGNORE` dan kembalikan yang ter-insert.
- [ ] **Step 4:** `go test ./app/services` → PASS.
- [ ] **Step 5:** Commit `feat(tasbih): badges`.

### Task 6: Handler, routes, redirect, menu

**Files:**
- Create: `app/handlers/tasbih.go`, `app/handlers/tasbih_test.go`
- Modify: `routes/web.go` (field `Tasbih *handlers.TasbihHandler` di `Handlers`, route di grup `protected`), `cmd/laju-go/main.go` (wire), `app/handlers/dashboard.go:36-37` dan `app/handlers/auth.go:198-199` (`/app/profile` → `/app/tasbih`), `frontend/src/layouts/AppLayout.svelte:90,97+` (homeHref user → `/app/tasbih`; 2 entri menu `show: role === "user"`, kategori "Ibadah", ikon lucide `CircleDot` dan `ChartColumn`)

**Interfaces:**
- Consumes: semua method service Task 2–5.
- Produces (route → handler → Inertia props / JSON), semuanya memakai `user_id` dari sesi:
  - `GET /app/tasbih` → `Index` → `app/tasbih/Index` props `{user, dzikir, paket, hari_ini, target_harian, streak, level, badge_baru}`
  - `GET /app/tasbih/dashboard?rentang=` → `Dashboard` → `app/tasbih/Dashboard` props `{user, dashboard, badges, riwayat:{items, ada_lagi}, dzikir, target_harian, badge_baru}`
  - `GET /app/tasbih/riwayat?halaman=&dzikir_id=` → `Riwayat` → JSON `{items, ada_lagi}`
  - `POST /app/tasbih/sync` → `Sync` → JSON `TasbihSyncHasil` (400 bila body rusak atau >100)
  - `POST /app/tasbih/dzikir`, `PUT /app/tasbih/dzikir/:id`, `DELETE /app/tasbih/dzikir/:id`, `PUT /app/tasbih/dzikir/:id/target`, `PUT /app/tasbih/target` → `inertiaService.Back(c)`. Error validasi dikirim lewat `store.Flash(c,"error",msg)`. Badge baru dikirim lewat `store.Flash(c,"tasbih_badge", kode dipisah koma)`.
  - `badge_baru` di Index/Dashboard = `GetFlash(c,"tasbih_badge")` yang dipetakan ke `[]TasbihBadge`.
  - `/app/tasbih/riwayat` dan `/sync` didaftarkan sebelum route berparameter. Tidak ada konflik dengan `/app/tasbih/dzikir/:id`.

- [ ] **Step 1:** Tulis handler test (pola `setupTestApp` di `auth_handler_test.go`, tetapi pakai goose migrations):
  - `TestTasbihSyncButuhLogin` → 401/redirect tanpa sesi.
  - `TestTasbihSyncRoleLain`: user role `cs` → 200.
  - `TestTasbihSyncBodyRusak` → 400.
  - `TestTasbihRiwayatScopedUser`.
  - `TestRoleHomeUser`: `roleHome(models.RoleUser)` = `/app/tasbih`.
- [ ] **Step 2:** Jalankan → FAIL.
- [ ] **Step 3:** Implementasi handler, routes, wiring, redirect, dan menu.
- [ ] **Step 4:** `go test ./...` → PASS. `go vet ./...` bersih. Cek tidak ada test lama yang mengharapkan `/app/profile` untuk role user (`grep -rn '/app/profile' app/**/*_test.go`). Kalau ada, sesuaikan karena itu perubahan yang disengaja.
- [ ] **Step 5:** Commit `feat(tasbih): handler, routes, user landing and menu`.

### Task 7: Lib klien (IndexedDB, mesin sesi, sync, feedback)

**Files:**
- Create: `frontend/src/lib/tasbih/types.ts`, `db.ts`, `sync.ts`, `counter.svelte.ts`, `feedback.ts`

**Interfaces:**
- Produces:
  ```ts
  // types.ts — cermin JSON Go (snake_case)
  export interface Dzikir { id:number; custom:boolean; arab:string; latin:string; arti:string; rujukan:string; target:number; target_default:number }
  export interface Paket { id:number; slug:string; nama:string; deskripsi:string; rujukan:string; items:{dzikir_id:number; target:number}[] }
  export interface SesiTertutup { id:string; dzikir_id:number; paket_id:number|null; paket_run_id:string|null; jumlah:number; target:number; mulai_at:string; selesai_at:string }
  export interface StateAktif { mode:'dzikir'|'paket'; dzikir_id:number; paket_id:number|null; paket_run_id:string|null; item_index:number; sesi_id:string; jumlah:number; target:number; mulai_at:string; tap_terakhir_at:string }
  // db.ts — DB "ruang-sanad-tasbih" v1, store "antrean"(keyPath id) & "aktif"(key "current")
  export function antreanTambah(s:SesiTertutup):Promise<void>; antreanSemua():Promise<SesiTertutup[]>; antreanHapus(ids:string[]):Promise<void>
  export function aktifSimpan(a:StateAktif|null):Promise<void>; aktifAmbil():Promise<StateAktif|null>
  // sync.ts
  export type StatusSync = 'tersimpan'|'menunggu'|'offline'|'menyinkron'
  export const syncState: { status:StatusSync; pending:number } // $state di .svelte.ts bila perlu → beri nama file sync.svelte.ts
  export function syncSekarang():Promise<SyncHasil|null>; mulaiAutoSync(onHasil:(h:SyncHasil)=>void):()=>void // online event, interval 60s, saat load
  // counter.svelte.ts
  export class TasbihCounter { state:StateAktif|null; selesaiTarget:boolean; constructor(katalog:{dzikir:Dzikir[];paket:Paket[]}, cb:{onTarget():void; onItemPaketSelesai(next:number):void; onPaketSelesai():void; onSesiTutup():void}); pulihkan():Promise<void>; pilihDzikir(id:number):void; pilihPaket(id:number):void; tap():void; undo():void; reset():void; selesai():void; cekIdle(now:Date):void }
  // feedback.ts
  export function getar(pola:'tap'|'kelipatan'|'target'|'paket'):void; klik():void; kunciLayar(aktif:boolean):Promise<void> // pref di localStorage key "tasbih:pref" {getar,suara,layar}, try/catch
  ```
- Aturan (spec §7): tutup sesi jika ganti dzikir/paket, reset, selesai, idle > 5 menit (`cekIdle` dipanggil interval 30s dan saat `visibilitychange`), atau target item paket tercapai (otomatis lanjut ke item berikutnya dengan `sesi_id` baru dan `paket_run_id` sama). `jumlah=0` dibuang. Idle di tengah paket menutup sesi item tetapi state paket tetap ada di `aktif`, sehingga tap berikutnya membuka sesi baru di item yang sama dengan sisa target = target item − jumlah item yang sudah tercatat di run itu. Simpan `itemTercapai` per run di `StateAktif`, field opsional `progress_run?: Record<number, number>`. Ganti dzikir/paket atau `selesai()` meninggalkan run. `aktifSimpan` di-debounce 300ms dan di-flush saat `pagehide`. UUID memakai `crypto.randomUUID()`, dengan fallback bila tidak ada (HTTP non-secure).
- Sync: hapus dari antrean ID di `diterima` dan `ditolak` (ditolak di-`console.warn`). Backoff 5s→10s→…→5 menit saat gagal.

- [ ] **Step 1:** Implementasi kelima file.
- [ ] **Step 2:** `npx tsc --noEmit -p tsconfig.json` (atau `npx vite build`) → tidak ada error di file baru.
- [ ] **Step 3:** Commit `feat(tasbih): client offline queue, counter engine, feedback`.

(Tidak ada infrastruktur unit test TS. Perilaku diverifikasi lewat E2E di Task 10.)

### Task 8: Halaman counter

Sebelum menulis UI: invoke skill `ui-ux-pro-max:ui-ux-pro-max` untuk arah visual dan animasi, dan baca `design-principles` di `.llm-wiki/` (AGENTS.md). Tetapkan satu arah visual yang tenang dan premium. Lalu jalankan pre-flight checklist.

**Files:**
- Create: `frontend/src/pages/app/tasbih/Index.svelte`, `frontend/src/components/tasbih/ProgressRing.svelte`, `BeadStrip.svelte`, `DzikirPicker.svelte`, `DzikirFormSheet.svelte`, `BadgeToast.svelte`, `SyncIndicator.svelte`
- Modify: `frontend/src/app.css` (font Arab: `@font-face` atau import Google Fonts `Amiri`; token `--font-family-arab`), `templates/` bila font dimuat via `<link>`

**Interfaces:**
- Consumes: props Task 6, lib Task 7.
- Perilaku wajib (spec §10):
  - Area tap besar dengan `onpointerdown` (`touch-action: manipulation`, mencegah zoom dan double-tap).
  - Keyboard: Spasi/Enter = tap (abaikan `repeat`), Backspace = undo.
  - Ring memakai `stroke-dashoffset`. Butir tasbih bergeser lewat `transform` dengan transisi sekitar 180ms. Angka pop memakai `scale`.
  - Perayaan target dan paket (ringan, CSS/SVG, tanpa library).
  - Ringkasan saat paket selesai.
  - Mode fokus (layar penuh, sembunyikan chrome; `document.documentElement.requestFullscreen` opsional).
  - Toggle getar, suara, layar menyala.
  - Bar "Hari ini x/target" + streak + level.
  - SyncIndicator.
  - BadgeToast dari respons sync maupun prop `badge_baru`.
  - Sheet kelola dzikir: tambah, ubah, arsip, dan ubah target, semua via `router.*`.
  - Ubah target harian via `router.put('/app/tasbih/target')`.
  - Link ke dashboard dengan `use:inertia`.
  - `prefers-reduced-motion`.
  - Dark mode.
  - Lebar 360px tanpa scroll horizontal.

- [ ] **Step 1:** Implementasi.
- [ ] **Step 2:** `npx vite build` → sukses.
- [ ] **Step 3:** Commit `feat(tasbih): counter page`.

### Task 9: Dashboard

Sebelum menulis UI: invoke skill `interface-design:interface-design` dan skill `dataviz`. Ikuti arah visual Task 8.

**Files:**
- Create: `frontend/src/pages/app/tasbih/Dashboard.svelte`, `frontend/src/components/tasbih/KpiCard.svelte`, `DailyBar.svelte`, `Heatmap.svelte`, `DonutKomposisi.svelte`, `HourBar.svelte`, `BadgeGrid.svelte`, `RiwayatList.svelte`

**Interfaces:**
- Consumes: props Task 6, `antreanSemua()` dan `syncSekarang()` Task 7, `GET /app/tasbih/riwayat`.
- Isi dan urutan sesuai spec §11 (1–9):
  - Filter rentang memakai `router.get` dengan `preserveScroll`.
  - DailyBar: garis target harian dan label perubahan "+x% vs n hari sebelumnya".
  - Heatmap: 53 kolom × 7, 5 tingkat intensitas `brand-*`, tooltip.
  - Donut: 6 + Lainnya.
  - HourBar: 24 batang dengan label jam WIB.
  - Paket: daftar jumlah putaran.
  - BadgeGrid: diraih/terkunci dengan syarat.
  - RiwayatList: "Muat lagi" + filter dzikir. Sesi antrean lokal ditampilkan di atas dengan label "belum tersinkron".
  - KPI "Hari ini" ditambah total antrean lokal yang `mulai_at` hari ini (WIB).
  - Catatan "n sesi menunggu sinkron".
  - Saat mount, jalankan `syncSekarang()`. Bila ada yang diterima, `router.reload()`.
  - Empty state untuk user tanpa sesi.
  - Semua chart: SVG, `role="img"` + `aria-label`, responsif di lebar 360px.

- [ ] **Step 1:** Implementasi.
- [ ] **Step 2:** `npx vite build` → sukses.
- [ ] **Step 3:** Commit `feat(tasbih): dashboard`.

### Task 10: Verifikasi E2E + review desain

**Files:** script sementara di scratchpad, tidak di-commit.

- [ ] **Step 1:** `npm run build:all` (vite lalu go build). Jalankan binary dengan DB salinan sementara di port bebas (cek `.env.example` untuk variabel `PORT`/DB). Ini bukan dev server user. Matikan setelah selesai.
- [ ] **Step 2:** Buat user role `user` dan inject sesi sesuai `.llm-wiki/wiki/concepts/agent-browser-testing.md`.
- [ ] **Step 3:** Playwright (viewport 390×844) dengan langkah berikut:
  - Login sebagai user → URL `/app/tasbih` dan menu "Tasbih" terlihat.
  - Tap 33× pada subhanallah → perayaan muncul.
  - `context.setOffline(true)` → tap 10× → indikator "Menunggu sinkron".
  - Reload saat offline tidak bisa (cold start, di luar scope). Sebagai gantinya, tutup halaman lalu buka lagi setelah online. Hitungan aktif harus dipulihkan dari IndexedDB.
  - `setOffline(false)` → indikator "Tersimpan". Baris di `tasbih_sesi` sesuai.
  - Jalankan paket Ba'da Shalat sampai selesai → ringkasan muncul, dan dashboard menampilkan 1 putaran.
  - Login sebagai role `cs` → menu Tasbih tidak ada, tetapi `/app/tasbih` bisa dibuka.
  - Dashboard: semua kartu tampil, mode gelap OK, tidak ada error di console.
  - Ambil screenshot counter dan dashboard (light dan dark).
- [ ] **Step 4:** Invoke `interface-design:design-review` pada screenshot. Perbaiki temuan dengan severity tinggi.
- [ ] **Step 5:** `go test ./...` dan `npx vite build` → hijau. Commit perbaikan, lalu push.
