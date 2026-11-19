# Form Pendaftaran Dinamis — Fase 1a Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Admin bisa membuat link pendaftaran (mode WA / short / full) per program dengan harga dari produk + varian. Calon santri mendaftar lewat halaman publik, lead tercipta tanpa duplikat, dan untuk mode bayar mereka mengunggah bukti transfer manual yang diverifikasi staf.

**Architecture:** Tabel baru (`produk_varian`, `registration_links`, `registration_link_stats`, `lead_registrations`, `payments`), query lewat sqlc. `RegistrationService` memakai `LeadService.mutateLeadAs` untuk semua perubahan lead, sehingga `lead_events` dan worker CAPI tetap jadi satu-satunya jalur. Halaman publik Svelte/Inertia di `/daftar/:slug` mengikuti pola `/hadir/:token` (`HadirProgram.svelte`).

**Tech Stack:** Go 1.26 + Fiber, sqlc v1.31.1, goose, SQLite, Svelte 5 + Inertia, Tailwind `@theme` tokens.

**Spec:** `docs/superpowers/specs/2026-11-19-form-pendaftaran-pembayaran-design.md` (fase 1a = §5.6; gateway Xendit dan penahanan kursi = fase 1b, plan terpisah).

## Global Constraints

- Handler → Service → Query → DB. Handler tidak memanggil query. Satu file handler per fitur.
- `app/queries/*.sql.go` hanya dari `sqlc generate`. Jalankan dengan `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` (binary `sqlc` tidak terpasang).
- Satu migrasi = satu tabel. Format goose `-- +goose Up/Down` + `StatementBegin/End` seperti `migrations/0121_create_program_kehadiran_table.sql`. Nomor mulai `0122`.
- Jangan jalankan dev server (`npm run dev:all` dll.).
- Svelte 5: `$derived`, bukan `$effect`, untuk state turunan. Link internal pakai `use:inertia`. `fetch()` ke `/app/*` wajib `X-XSRF-TOKEN` dari `getCSRFToken()`. Warna hanya dari token `app.css`. Baca `design-principles` wiki sebelum menulis halaman.
- Harga: `nominal_override` link > `produk_varian.harga` > `produk.harga`. Tidak pernah dari input pendaftar.
- Nomor WA dinormalisasi dengan `NormalizeLeadPhone` (`app/services/lead_parser.go:205`). `wa_chat_id` lead baru = `<nomor>@s.whatsapp.net`.
- Actor event dari form: `form:<link_id>`. Event id prefix: `frm`.
- Upload bukti: JPG/PNG/WebP/PDF, ≤ 2 MB, cek magic bytes, nama file acak, disimpan di `storage/payment_proofs/` (bukan folder publik).
- `ref` halaman bayar: 32 byte acak, base64url.
- Rate limit publik: submit 5 per 10 menit per IP; upload 5 per 10 menit per IP; per nomor WA 3 submit per jam.
- Token render form: HMAC dari `slug|unix_ts` dengan session secret yang ada di config. Submit < 3 detik atau > 2 jam sejak render ditolak.
- Pesan error ke pendaftar dalam Bahasa Indonesia, ramah, tanpa detail teknis.

## Review Focus

1. Nomor WA dalam format aneh (`+62 812-3456-7890`, `0812…`, `62812…`, spasi) harus menghasilkan satu lead yang sama → test di Task 5.
2. Pendaftar submit dua kali (klik ganda / refresh) → satu lead, dua baris `lead_registrations`, tahap tidak mundur → test di Task 5.
3. Link dinonaktifkan, atau program/produk dinonaktifkan setelah link dibagikan → halaman "link tidak aktif" + tombol WA, bukan 500 → test di Task 7.
4. Jadwal penuh di antara render dan submit → error ramah, data form tetap terisi → test di Task 5 (service) + Task 7 (handler status 422).
5. Upload file yang diganti ekstensinya (`.exe` → `.jpg`) atau > 2 MB → ditolak tanpa menyentuh lead → test di Task 8.

---

## File Structure

| File | Tanggung jawab |
|---|---|
| `migrations/0122_create_produk_varian_table.sql` | tabel varian harga |
| `migrations/0123_create_registration_links_table.sql` | konfigurasi link |
| `migrations/0124_create_registration_link_stats_table.sql` | agregat harian klik/submit/bayar |
| `migrations/0125_create_lead_registrations_table.sql` | jejak submit + `ref` |
| `migrations/0126_create_payments_table.sql` | pembayaran (manual sekarang, gateway nanti) |
| `migrations/0127_add_registration_settings_to_crm_settings.sql` | `cs_wa_number`, `payment_expiry_hours`, `bank_manual_info` |
| `migrations/0128_add_phone_index_to_leads.sql` | index `leads(phone)` untuk dedup |
| `queries/produk_varian.sql`, `queries/registration.sql`, `queries/payments.sql` | query sqlc |
| `app/models/registration.go` | request/response struct |
| `app/services/produk_varian.go` | CRUD varian |
| `app/services/registration_links.go` | CRUD link, harga, stats, settings |
| `app/services/registration_guard.go` | token render, limit per nomor |
| `app/services/registration_submit.go` | submit → lead |
| `app/services/payment_manual.go` | bukti manual, tolak, tandai lunas |
| `app/handlers/registration_links.go` | admin link pendaftaran |
| `app/handlers/registration_public.go` | halaman publik + upload |
| `frontend/src/pages/public/Daftar.svelte` | form / sukses / link tidak aktif |
| `frontend/src/pages/public/DaftarBayar.svelte` | halaman bayar manual |
| `frontend/src/pages/app/RegistrationLinks.svelte` + `frontend/src/components/registration/LinkForm.svelte` | admin |
| `frontend/src/components/leads/RegistrationPanel.svelte` | panel di detail lead |

---

### Task 1: Skema + query

**Files:**
- Create: migrasi `0122`–`0128` (lihat File Structure), `queries/produk_varian.sql`, `queries/registration.sql`, `queries/payments.sql`
- Generate: `app/queries/*.sql.go`
- Test: `app/services/registration_schema_test.go`

Kolom persis seperti spec §5.1, ditambah:
- `lead_registrations.ref TEXT NOT NULL UNIQUE` (token halaman bayar, keputusan plan: satu ref per submit).
- `registration_links.mode` CHECK IN (`wa`,`short`,`full`), `after_submit` CHECK IN (`wa`,`bayar`), `field_config TEXT NOT NULL DEFAULT '{}'`, `izinkan_manual INTEGER NOT NULL DEFAULT 1`.
- `payments.status` CHECK IN (`pending`,`paid`,`expired`,`failed`,`menunggu_verifikasi`,`ditolak`), `metode` CHECK IN (`gateway`,`manual`), `external_id TEXT UNIQUE` (NULL untuk manual).
- `registration_link_stats` PK (`link_id`,`tanggal`).

**Interfaces:**
- Produces (nama query sqlc): `CreateProdukVarian`, `UpdateProdukVarian`, `ListProdukVarianByProduk`, `GetProdukVarian`; `CreateRegistrationLink`, `UpdateRegistrationLink`, `GetRegistrationLinkBySlug`, `GetRegistrationLink`, `ListRegistrationLinks`, `BumpRegistrationStat(link_id, tanggal, klik, submit, bayar)` (UPSERT tambah), `ListRegistrationStatsSince(since)`; `CreateLeadRegistration`, `GetLeadRegistrationByRef`, `ListLeadRegistrationsByLead`; `CreatePayment`, `GetPayment`, `ListPaymentsByLead`, `UpdatePaymentStatus(id, status, paid_at, catatan_sistem, updated_at)`, `ListPendingManualPaymentsByLead`; `GetLeadByPhone` (status apa pun, terbaru dulu); `GetRegistrationSettings`, `UpdateRegistrationSettings`.

- [ ] **Step 1: Tulis test** `TestRegistrationSchemaRoundTrip`: dengan `setupSantriDatabase(t)`, buat produk → varian → link (slug `tahsin-reguler`) → lead → lead_registration → payment manual. Assert `GetRegistrationLinkBySlug` mengembalikan link, `BumpRegistrationStat` dua kali dengan klik=1 → klik=2, dan insert link dengan slug sama → error UNIQUE.
- [ ] **Step 2:** `go test ./app/services -run TestRegistrationSchemaRoundTrip` → FAIL (query belum ada).
- [ ] **Step 3:** Tulis migrasi + query, lalu `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`.
- [ ] **Step 4:** Test → PASS. `go build ./...` → sukses.
- [ ] **Step 5:** Commit `feat(daftar): skema produk varian, link pendaftaran, payments`.

### Task 2: Varian harga di Produk CRM

**Files:**
- Create: `app/services/produk_varian.go`, test `app/services/produk_varian_test.go`
- Modify: `app/handlers/product_crm.go` (method baru), `routes/web.go` (di blok `/produk-crm`, role `akRole`), halaman Produk CRM yang ada (`frontend/src/pages/app/` — cari komponen yang me-render `/produk-crm`) untuk panel varian per produk.

**Interfaces:**
- Produces: `func (s *ProductCRMService) ListVarian(ctx context.Context, produkID int64) ([]models.ProdukVarian, error)`, `CreateVarian(ctx, produkID int64, req models.ProdukVarianRequest) error`, `UpdateVarian(ctx, id int64, req models.ProdukVarianRequest) error`. `models.ProdukVarian{ID, ProdukID, Nama, Harga, PaketKode, Urutan, IsAktif}`; request tanpa ID/ProdukID.
- Routes: `POST /app/produk-crm/produk/:id/varian`, `PUT /app/produk-crm/varian/:id`. Redirect 303 balik ke `/app/produk-crm`.

- [ ] **Step 1: Tulis test** `TestProdukVarianValidation`: nama kosong → error; harga < 0 → error; `paket_kode` hanya `''` atau kunci yang ada di `crm_settings.paket_ke_kelas_kode` (default `1x`,`2x`) → `3x` error; produk tidak ada → error; list urut `urutan, id`.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi service + handler + route + UI (daftar varian, tambah, ubah, aktif/nonaktif; tidak ada hapus).
- [ ] **Step 4:** `go test ./app/services -run Varian` → PASS; `npm run build` → sukses.
- [ ] **Step 5:** Commit `feat(produk): varian harga per produk`.

### Task 3: Link pendaftaran — service (CRUD, harga, settings, stats)

**Files:**
- Create: `app/services/registration_links.go`, `app/models/registration.go`, test `app/services/registration_links_test.go`

**Interfaces:**
- Produces:
  - `type RegistrationService struct{ querier *queries.Querier; leads *LeadService; jadwal *JadwalPenawaranService; now func() time.Time; secret []byte }`
  - `func NewRegistrationService(q *queries.Querier, leads *LeadService, jadwal *JadwalPenawaranService, secret []byte) *RegistrationService`
  - `CreateLink(ctx, req models.RegistrationLinkRequest, actorID int64) (int64, error)`, `UpdateLink(ctx, id int64, req models.RegistrationLinkRequest) error`, `ListLinks(ctx) ([]models.RegistrationLinkRow, error)` (dengan klik/submit/bayar 30 hari WIB)
  - `PublicLink(ctx, slug string) (*models.PublicRegistrationLink, error)`, yang mengembalikan `ErrLinkTidakAktif` bila link/program/produk nonaktif atau tidak ada. Isinya judul, deskripsi, mode, after_submit, program (nama, jenis), varian aktif + harga final, field_config, `butuh_jadwal` (= program jenis `reguler`).
  - `ResolveAmount(link queries.RegistrationLink, varianID int64) (amount int64, varian *queries.ProdukVarian, err error)`
  - `Settings(ctx) / SaveSettings(ctx, models.RegistrationSettings) error`
  - `var ErrLinkTidakAktif = errors.New("link pendaftaran tidak aktif")`
- `RegistrationLinkRequest`: `Slug, Judul, Deskripsi, ProgramID, LeadSourceID, Mode, AfterSubmit, NominalOverride *int64, FieldConfig map[string]FieldRule, WaTemplate, IzinkanManual, IsAktif`. Field opsional yang boleh diatur: `usia, domisili, riwayat_belajar, belajar_dimana, untuk_siapa, waktu_belajar`. `jenis_kelamin` selalu wajib bila program reguler.

- [ ] **Step 1: Tulis test**:
  - `TestCreateLinkValidation`: slug `Tahsin Reguler` → error (hanya `[a-z0-9-]`, 3–60); program tanpa `produk_id` → error "program belum terhubung ke produk"; `lead_source_id` kosong → error; mode `wa` tanpa `cs_wa_number` di settings → error.
  - `TestResolveAmount`: produk harga 150000 tanpa varian → 150000; dengan varian 1x=200000, 2x=350000, pilih 2x → 350000; varian dari produk lain → error; varian wajib dipilih bila ada varian aktif; `nominal_override=99000` → 99000 untuk varian apa pun.
  - `TestPublicLinkInactive`: link nonaktif, program nonaktif, produk nonaktif → `ErrLinkTidakAktif`.
  - `TestListLinksStats`: stats 3 hari terakhir dijumlah; stats 40 hari lalu tidak.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi.
- [ ] **Step 4:** Run → PASS.
- [ ] **Step 5:** Commit `feat(daftar): service link pendaftaran dan harga`.

### Task 4: Guard anti-spam

**Files:**
- Create: `app/services/registration_guard.go`, test `app/services/registration_guard_test.go`
- Modify: `app/middlewares/rate-limit.go` (tambah `DaftarRateLimit` 5/10 menit, `DaftarUploadRateLimit` 5/10 menit, pola seperti `HadirRateLimit`)

**Interfaces:**
- Produces: `func (s *RegistrationService) FormToken(slug string) string`, `func (s *RegistrationService) CheckFormToken(slug, token string) error` (`ErrFormTerlaluCepat`, `ErrFormKedaluwarsa`, `ErrFormTidakValid`), `func (s *RegistrationService) allowPhone(phone string) bool` (in-memory, 3 per jam per nomor, mutex + map, bersih otomatis entri > 1 jam).

- [ ] **Step 1: Tulis test** `TestFormToken` (clock disuntik lewat `s.now`): token untuk slug A ditolak di slug B; umur 2 detik → `ErrFormTerlaluCepat`; 5 detik → ok; 2 jam 1 detik → `ErrFormKedaluwarsa`; token diubah satu karakter → `ErrFormTidakValid`. `TestAllowPhone`: submit ke-4 dalam 1 jam → false; setelah 61 menit → true.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi (HMAC-SHA256, `hmac.Equal`).
- [ ] **Step 4:** Run → PASS.
- [ ] **Step 5:** Commit `feat(daftar): token form dan limit per nomor`.

### Task 5: Submit → lead (inti)

**Files:**
- Create: `app/services/registration_submit.go`, test `app/services/registration_submit_test.go` (fixture: `setupLeadFixture` + `NewRegistrationService`)

**Interfaces:**
- Consumes: `LeadService.mutateLeadAs`, `LeadService.withTx`, `JadwalPenawaranService.Open(ctx, gender)`, Task 3, Task 4.
- Produces: `func (s *RegistrationService) Submit(ctx context.Context, slug string, in models.RegistrationSubmit, meta models.RequestMeta) (*models.RegistrationResult, error)`
  - `RegistrationSubmit`: `Nama, NoWA, JenisKelamin (L/P), Usia, Domisili, VarianID, JadwalKode, RiwayatBelajar, BelajarDimana, UntukSiapa, WaktuBelajar, Honeypot, FormToken`
  - `RequestMeta`: `IP, UserAgent`
  - `RegistrationResult`: `LeadRef string ("L-<id>"), Next string ("wa"|"bayar"), WaURL string, BayarRef string`
  - Error terstruktur `*FieldError{Field, Pesan}` untuk validasi per field (dipakai handler untuk 422).
- Aturan (spec §5.3):
  1. Honeypot terisi → kembalikan hasil "sukses" palsu tanpa menulis apa pun (bot tidak diberi sinyal).
  2. Token, `allowPhone`, validasi field sesuai mode + `field_config`. Usia 5–90. Nama 2–80 rune.
  3. Program reguler: `JadwalKode` wajib, harus ada di `Open(gender)` dengan sisa > 0 untuk gender itu, kalau tidak → `FieldError{"jadwal_kode", "Jadwal ini sudah penuh, silakan pilih jadwal lain"}`.
  4. Dalam `withTx`: cari `GetLeadByChatID(<nomor>@s.whatsapp.net)`, lalu `GetLeadByPhone`. Perlakuan per status sesuai spec (baru / open / lost dibuka kembali / won & DNC → `needs_review=1`, `Next="wa"`, tanpa ubah tahap).
  5. Isi field lead yang kosong saja. Set `program_id`, `paket_kode` (dari varian), `jadwal_kode` bila kosong. `lead_source_id` di-set bila kosong dengan `lead_source_by = LeadSourceByForm` (konstanta baru `"form"` di `lead_sources.go`). Pastikan `RematchSources` tidak menimpa sumber ber-`by` `form` (tambah assert di test).
  6. Tahap: naikkan ke `data_terisi` bila lebih awal. Bila `after_submit=bayar` (dan bukan won/DNC): naikkan ke `info_bayar_dikirim`, set `payment_amount` = `ResolveAmount`. Tidak pernah mundur. Urutan tahap ambil dari konstanta `LeadStage*` di `lead.go`.
  7. Tulis `lead_registrations` (payload JSON input tanpa token/honeypot, ref acak) dan `BumpRegistrationStat(submit=1)`.
  8. `WaURL` = `https://wa.me/<cs_wa_number>?text=` + `wa_template` terisi (`{nama} {program} {jadwal} {ref}` → `{ref}` = `L-<id>`). Bila template kosong: `Assalamu'alaikum, saya {nama} sudah isi form pendaftaran {program} ({ref}).`

- [ ] **Step 1: Tulis test** (satu fungsi per baris):
  - `TestSubmitCreatesLead`: lead baru `wa_chat_id=6281234567890@s.whatsapp.net`, tahap `data_terisi`, `lead_source_id` = sumber link, satu `lead_events` dengan actor `form:<id>`.
  - `TestSubmitPhoneFormatsDedup`: submit `0812-3456-7890` lalu `+62 812 3456 7890` → `COUNT(*) FROM leads` = 1, `lead_registrations` = 2.
  - `TestSubmitMatchesLeadByPhoneColumn`: lead lama dengan `wa_chat_id='abc@lid'`, `phone='6281234567890'` → submit memakai lead itu.
  - `TestSubmitDoesNotOverwrite`: lead dengan `nama='Fulan (CS)'`, `usia` kosong → setelah submit nama tetap, usia terisi.
  - `TestSubmitStageNeverRegresses`: lead di `bukti_masuk` → tetap `bukti_masuk`.
  - `TestSubmitBayarSetsAmount`: after_submit bayar, varian 2x 350000 → tahap `info_bayar_dikirim`, `payment_amount=350000`, `Next="bayar"`, `BayarRef` ≠ "".
  - `TestSubmitLostReopens`, `TestSubmitWonFlagsReview` (status tetap won, `needs_review=1`, `Next="wa"`), `TestSubmitDNCFlagsReview`.
  - `TestSubmitFullScheduleRejected`: kuota 1, satu lead terverifikasi gender L → submit L → `FieldError` `jadwal_kode`; tidak ada lead/registrasi tertulis.
  - `TestSubmitProgramPendekNoJadwal`: program jenis `program` → `JadwalKode` diabaikan, tidak wajib.
  - `TestSubmitHoneypotWritesNothing`.
  - `TestSubmitWaURL`: template default → URL berisi nomor CS dan `L-<id>` ter-encode.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi.
- [ ] **Step 4:** `go test ./app/services -run 'Submit|Registration'` → PASS; `go test ./app/services` tetap hijau.
- [ ] **Step 5:** Commit `feat(daftar): submit form membuat atau memperbarui lead`.

### Task 6: Pembayaran manual (service)

**Files:**
- Create: `app/services/payment_manual.go`, test `app/services/payment_manual_test.go`
- Modify: `app/services/lead_closing.go` — di `VerifyPayment` (di dalam fn `mutateLead`, sebelum return), tandai semua payment `menunggu_verifikasi` milik lead itu `paid` dengan `paid_at=now`.

**Interfaces:**
- Produces:
  - `func (s *RegistrationService) BayarInfo(ctx, ref string) (*models.BayarInfo, error)`: nama depan, program, varian, nominal, info rekening, `izinkan_manual`, status pembayaran terakhir (`belum` / `menunggu_verifikasi` / `lunas` / `ditolak`). Ref tidak dikenal → `ErrRefTidakAda`.
  - `func (s *RegistrationService) SubmitManualProof(ctx, ref string, file ProofFile) error` dengan `type ProofFile struct{ Name string; Size int64; Head []byte; Save func(dst string) error }`.
  - `func (s *RegistrationService) RejectManualPayment(ctx, paymentID, actorID int64, alasan string) error`
  - `func DetectProofType(head []byte) (ext string, ok bool)`, berdasarkan magic bytes JPG/PNG/WebP/PDF.
- Aturan: ukuran > 2 MB atau tipe tidak dikenal → `FieldError{"bukti", ...}` tanpa efek. Sukses → simpan `storage/payment_proofs/<random>.<ext>`, `payments(metode=manual, status=menunggu_verifikasi, amount=lead.payment_amount, bukti_path)`, lead → `bukti_masuk` + `proof_received_at` lewat `mutateLeadAs(actor form:<link_id>)`, lalu `CreateNotification` ke semua user role `keuangan` (judul "Bukti transfer masuk", link `/app/leads/<id>`). Bila sudah ada payment `menunggu_verifikasi` → ganti bukti (update row, file lama dihapus), tanpa notifikasi kedua. Lead won/DNC/sudah terverifikasi → `ErrSudahDiproses`. `Reject` → payment `ditolak` + `catatan_sistem=alasan`, lead kembali ke `info_bayar_dikirim` (event `payment.rejected`).

- [ ] **Step 1: Tulis test**: `TestDetectProofType` (PNG/JPG/WebP/PDF header asli → ok; teks `MZ…` → !ok); `TestSubmitManualProof` (tahap `bukti_masuk`, satu notifikasi keuangan, file ada di tmp dir); `TestSubmitManualProofTooLarge` (2 MB + 1 → error, lead tidak berubah); `TestSubmitManualProofReplace` (dua upload → satu payment, satu notifikasi); `TestVerifyPaymentMarksManualPaid` (setelah `VerifyPayment` staf → payment `paid`); `TestRejectManualPayment`.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi. Direktori penyimpanan disuntik lewat field `proofDir` (test memakai `t.TempDir()`).
- [ ] **Step 4:** Run → PASS. Test `lead_closing` yang ada tetap hijau.
- [ ] **Step 5:** Commit `feat(daftar): bukti transfer manual dan verifikasi`.

### Task 7: Halaman publik

**Files:**
- Create: `app/handlers/registration_public.go`, test `app/handlers/registration_public_test.go`, `frontend/src/pages/public/Daftar.svelte`, `frontend/src/pages/public/DaftarBayar.svelte`
- Modify: `routes/web.go` (fungsi `setupRegistrationPublicRoutes(app, h)` dipanggil di samping `setupProgramPublicRoutes`), `cmd/laju-go/main.go` (wiring `NewRegistrationService`, handler, `os.MkdirAll("storage/payment_proofs")`), `routes/web.go` `Handlers` struct.

**Interfaces:**
- Routes:
  - `GET /daftar/:slug`: mode `wa` → bump klik, 302 ke WaURL (pesan tanpa data). Selain itu bump klik, lalu render `public/Daftar` dengan props `{link, form_token, jadwal (bila reguler: semua jadwal Open, dengan sisa per gender), galat?}`. Link tidak aktif → 404 + `galat` + `wa_cs_url`.
  - `POST /daftar/:slug` (`DaftarRateLimit`): JSON → `Submit`. 200 `{next, wa_url, bayar_url}`. `FieldError` → 422 `{field, error}`. Error lain → 422 `{error}` dengan pesan umum.
  - `GET /daftar/bayar/:ref` → render `public/DaftarBayar`.
  - `POST /daftar/bayar/:ref/bukti` (`DaftarUploadRateLimit`, `BodyLimit` 3 MB): multipart `bukti`.
  - Semua respons: `Cache-Control: no-store`. Route di luar `/app` (tanpa CSRF), seperti `/hadir`.
- Service baru di task ini: `func (s *RegistrationService) CountClick(ctx context.Context, linkID int64, ip, ua string) error`. Klik di-dedup per `linkID+IP+UA` per hari WIB dengan set in-memory (mutex + map, dibersihkan saat ganti hari), lalu `BumpRegistrationStat(klik=1)`. Test service `TestCountClickDedup`: dua panggilan sama → klik 1; UA beda → 2.

UI `Daftar.svelte` (mobile-first, satu kolom, maks lebar 480px, logo di atas seperti `HadirProgram.svelte`): judul + deskripsi, field sesuai mode, varian sebagai pilihan kartu dengan harga, gender sebelum jadwal, dan daftar jadwal disaring oleh gender terpilih via `$derived` (jadwal sisa 0 tampil disabled "Penuh"). Honeypot input tersembunyi `website`. Tombol kirim disable saat mengirim. Error field ditampilkan di bawah field. Sukses `wa` → layar sukses + tombol "Lanjut ke WhatsApp" + auto `window.location` setelah 1,5 detik. Sukses `bayar` → `window.location = bayar_url`. `DaftarBayar.svelte`: ringkasan, nominal besar, info rekening (tombol salin), upload bukti dengan preview gambar, status sesuai `BayarInfo`.

- [ ] **Step 1: Tulis test handler** (pakai pola `app/handlers/*_test.go` yang ada dengan Fiber app test): `TestDaftarModeWARedirects` (302, Location `https://wa.me/`), `TestDaftarInactiveLink404`, `TestDaftarSubmitFieldError422` (jadwal penuh → 422 `field=jadwal_kode`), `TestDaftarSubmitOK`, `TestDaftarRouteNoCSRF` (POST tanpa token CSRF tidak 403).
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi handler, route, wiring, kedua halaman Svelte.
- [ ] **Step 4:** `go test ./app/handlers -run Daftar` → PASS; `npm run build` → sukses; `go build ./...` → sukses.
- [ ] **Step 5:** Commit `feat(daftar): halaman pendaftaran publik dan bayar manual`.

### Task 8: Admin link pendaftaran + panel lead

**Files:**
- Create: `app/handlers/registration_links.go`, `frontend/src/pages/app/RegistrationLinks.svelte`, `frontend/src/components/registration/LinkForm.svelte`, `frontend/src/components/leads/RegistrationPanel.svelte`, test `app/handlers/registration_links_test.go`
- Modify: `routes/web.go` (di `setupLeadRoutes`, role sama dengan `programRole`), menu navigasi Leads (cari tempat item "Program" didaftarkan di `AppLayout`/nav config) + item "Link Pendaftaran", `app/services/lead_crm.go` `Detail` → tambah `Registrasi []models.LeadRegistrationRow` dan `Payments []models.PaymentRow` ke `models.LeadDetail`, halaman detail lead render `RegistrationPanel`.
- Bukti manual: `GET /app/leads/:id/payments/:pid/bukti` (role `leadRole`) → `SendFile` dari `storage/payment_proofs`, path diambil dari DB (tidak dari input). `POST /app/leads/:id/payments/:pid/tolak` (role keuangan/super_admin, sama dengan role tombol Verifikasi).

**Interfaces:**
- Routes: `GET /app/leads/link-pendaftaran`, `POST /app/leads/link-pendaftaran`, `PUT /app/leads/link-pendaftaran/:lid`, `PUT /app/leads/link-pendaftaran/pengaturan`. Didaftarkan sebelum `/leads/:id` (pola route statis di `setupLeadRoutes`).
- Props halaman: `links` (Task 3 `ListLinks`), `programs`, `sources`, `settings`, `base_url`.

UI: tabel link (judul, slug + tombol salin URL lengkap, mode, program, nominal efektif, aktif, klik/submit/bayar 30 hari + konversi submit÷klik), tombol buat. `LinkForm` di drawer/kanan dengan pratinjau langsung yang memakai ulang tampilan field `Daftar.svelte` (ekstrak bagian field ke `components/registration/DaftarFields.svelte` yang dipakai keduanya). Kartu pengaturan: nomor WA CS, info rekening, masa berlaku tagihan (disimpan, dipakai fase 1b). `RegistrationPanel`: riwayat submit (waktu, link, mode), daftar pembayaran (metode, nominal, status, tombol lihat bukti, tombol tolak bila `menunggu_verifikasi`).

- [ ] **Step 1: Tulis test handler**: `TestRegistrationLinksRoleGuard` (role `guru` → 403), `TestBuktiNotPathTraversal` (pid milik lead lain → 404), `TestRegistrationLinkCreateRedirect` (303 ke daftar link).
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implementasi.
- [ ] **Step 4:** `go test ./...` → PASS; `npm run build` → sukses.
- [ ] **Step 5:** Commit `feat(daftar): admin link pendaftaran dan panel pembayaran di lead`.

### Task 9: Verifikasi akhir

- [ ] **Step 1:** `go vet ./... && go test ./...` → semua PASS.
- [ ] **Step 2:** `npm run build:all` → sukses.
- [ ] **Step 3:** E2E agent_browser (pola wiki `agent-browser-testing`, server dijalankan oleh user): buat produk + varian → tautkan ke program Reguler → buat link short/bayar → buka `/daftar/<slug>` di viewport 390px → isi → halaman bayar → upload PNG → lead tampil di CRM tahap `bukti_masuk` dengan panel pembayaran → Verifikasi → payment `paid`. Ulangi mode `wa` (redirect) dan full/`wa`. Screenshot tiap layar publik.
- [ ] **Step 4:** Pre-flight checklist `design-principles` §5 untuk tiga halaman baru.
- [ ] **Step 5:** Commit perbaikan bila ada, lalu push branch.

---

## Self-review

- Cakupan spec fase 1a: §5.0 → Task 2, 3; §5.1 → Task 1; §5.2 → Task 7; §5.3 submit → Task 5, manual → Task 6; §5.4 → Task 2, 8; §5.5 → Task 4, 6, 7; §5.7 → test di tiap task + Task 9. Gateway, webhook, penahanan kursi, CAPI Purchase saat lunas → fase 1b (sengaja tidak di sini; di fase 1a kuota tetap dihitung dari lead terverifikasi seperti sekarang).
- Nama konsisten: `RegistrationService`, `Submit`, `ResolveAmount`, `PublicLink`, `BayarInfo`, `SubmitManualProof`, `RejectManualPayment`, `FieldError`, `ErrLinkTidakAktif`.
