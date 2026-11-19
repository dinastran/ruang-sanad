# Affiliate / Referral — Design

Tanggal: 2026-11-19 · Status: draft, menunggu persetujuan
Proyek 3. Bergantung pada proyek 1 (link pendaftaran, form, pembayaran).

## 1. Intent

Guru Ruang Sanad dan pihak eksternal (alumni, santri, umum) mendapat komisi
dari setiap mahasantri yang mendaftar lewat link affiliate mereka.

Tujuan bisnis (dari user):
- Menambah santri baru dari kanal di luar Meta Ads.
- Penghasilan tambahan untuk guru.

## 2. Goals / Non-goals / Anti-goals

**Goals**
- Setiap guru otomatis punya kode & link affiliate di dashboardnya.
- Eksternal mendaftar sendiri lewat form publik → disetujui admin → login
  terbatas ke halaman "Affiliate Saya".
- Aturan komisi dinamis: persen atau flat; sekali atau berkelanjutan per
  bulan (SPP), dengan batas bulan; berlaku global, per program, dan per
  affiliate (yang paling spesifik menang).
- Masa berlaku atribusi (cookie) bisa diatur 30 / 60 / 90 hari.
- Atribusi first-click: affiliate pertama yang masih berlaku menang.
- Komisi tahan 14 hari sebelum siap cair; batal bila santri berhenti /
  pembayaran dibatalkan dalam masa tahan.
- Pencairan: guru → komponen ujroh bulan berjalan; eksternal → keuangan
  menandai "sudah ditransfer".
- Dashboard affiliate: klik → daftar → santri → komisi, tombol salin/bagikan.

**Non-goals**
- Pencairan otomatis ke rekening (Xendit disbursement) — nanti.
- Multi-level (komisi dari rekrutan affiliate).
- Kupon diskon untuk pendaftar.

**Anti-goals**
- Komisi ganda untuk satu santri / satu tagihan.
- Komisi untuk pembayaran yang belum benar-benar masuk.
- Affiliate "menyerobot" calon santri yang sudah terkunci ke affiliate lain.
- Affiliate eksternal melihat data pribadi pendaftar selain nama depan.

## 3. Constraints (dari kode)

- Three-tier rule, sqlc, satu tabel per migrasi (AGENTS.md).
- `guru.user_id` → `users` (unik). Role login di `users.role` (teks bebas;
  daftar role di `routes/web.go`). Role baru: `affiliate`.
- Lead → santri: `leads.santri_id`. Pembayaran pertama terverifikasi:
  `leads.verified_at` (staf atau gateway, `verifyLeadTx`).
- SPP: `tagihan` (`santri_id`, `bulan_ke`, `nominal`, `status`
  belum_bayar/lunas/batal, `tanggal_bayar`). Lunas lewat
  `TagihanService.MarkLunas`.
- Status santri: aktif, cuti, nonaktif, tidak_lanjut.
- Ujroh: `ujroh_komponen` (jenis tetap/insidental), input per bulan
  `ujroh_komponen_bulan` (bulan, guru, komponen, jumlah, tarif), bulan bisa
  dikunci (`ujroh_bulan.dikunci_at`).
- Lead masuk lewat: form `/daftar/:slug` (`RegistrationService.Submit`), chat
  WA (`lead_ingest`), atau CS manual.
- `session.ClientIP` untuk IP di balik proxy.

## 4. Approaches

| | Pendekatan | Trade-off |
|-|-|-|
| **A (dipilih)** | Link `/r/KODE` + cookie first-party, + kode di pesan WA, + kolom "kode referal" opsional di form, + set manual oleh CS | Alur pendaftar tidak berubah; data klik ada; cookie hilang di ganti perangkat (ditutup jalur cadangan) |
| B | Kode referal saja di form | Simpel, tanpa data klik, bergantung pendaftar ingat kode |
| C | Link pendaftaran unik per affiliate | Akurat tanpa cookie, link meledak, masa berlaku 30/60/90 tak berlaku |

Mesin komisi: **rekonsiliasi berkala dari data** (bukan hook di setiap
tombol). Satu worker tiap 10 menit membaca lead terverifikasi & tagihan
lunas, membuat baris komisi yang belum ada (unik per sumber), mematangkan
yang lewat 14 hari, dan membatalkan yang sumbernya batal. Alasan: SPP bisa
lunas lewat beberapa jalur (tombol, massal, impor), verifikasi lewat staf
atau gateway; satu rekonsiliasi idempoten menutup semuanya dan tidak
menambah risiko ke alur keuangan yang ada.

## 5. Design

### 5.1 Data model (migrasi baru, satu tabel per file)

**`affiliates`**
- `id`, `kode` TEXT UNIQUE (huruf besar/angka/`-`, 3–20, mis. `UST-AHMAD`)
- `tipe` CHECK IN (`guru`, `eksternal`), `guru_id` NULL UNIQUE → guru,
  `user_id` NULL UNIQUE → users (login)
- `nama`, `no_wa`, `email`, `status` CHECK IN (`menunggu`, `aktif`,
  `ditolak`, `nonaktif`)
- `bank_nama`, `bank_rekening`, `bank_atas_nama` (eksternal)
- `catatan_admin`, `disetujui_oleh`, `disetujui_at`, `created_at`, `updated_at`

**`affiliate_rules`** (aturan komisi)
- `id`, `scope` CHECK IN (`global`, `program`, `affiliate`), `program_id`
  NULL, `affiliate_id` NULL (unik per scope+target)
- `jenis` CHECK IN (`persen`, `flat`), `nilai` INTEGER (persen: basis poin
  0–10000 = 0–100%; flat: rupiah)
- `pola` CHECK IN (`sekali`, `bulanan`), `maks_bulan` INTEGER NULL
  (NULL = selama aktif; default 6 saat dibuat)
- `is_aktif`, `updated_by`, `updated_at`
- Pencarian: affiliate+program? → **tidak** (dua dimensi cukup):
  `affiliate` > `program` > `global`. Bila tidak ada aturan aktif → tidak
  ada komisi.

**`affiliate_settings`** (baris tunggal)
- `masa_berlaku_hari` CHECK IN (30, 60, 90) default 60
- `masa_tahan_hari` default 14
- `link_tujuan_default_id` → `registration_links` (ke mana `/r/KODE` tanpa
  tujuan mengarah)
- `komponen_ujroh_id` → `ujroh_komponen` ("Komisi Affiliate", insidental,
  dibuat oleh migrasi seed)

**`affiliate_clicks`** (agregat harian, seperti `registration_link_stats`)
- `affiliate_id`, `tanggal`, `klik` — PK (affiliate_id, tanggal)

**`lead_affiliates`** (atribusi, satu per lead)
- `lead_id` UNIQUE, `affiliate_id`, `sumber` CHECK IN (`cookie`, `kode_form`,
  `kode_wa`, `manual`), `diklik_at` (waktu klik pertama bila dari cookie),
  `dikunci_at`, `diubah_oleh` NULL
- First-click: baris dibuat sekali; jalur otomatis tidak pernah menimpa.
  Hanya admin/CS (`manual`) yang boleh mengganti, dengan event di
  `lead_events`.

**`affiliate_commissions`** (buku komisi)
- `id`, `affiliate_id`, `lead_id`, `santri_id` NULL
- `sumber` CHECK IN (`pendaftaran`, `spp`), `sumber_id` (lead_id untuk
  pendaftaran, tagihan.id untuk spp) — UNIQUE (`sumber`, `sumber_id`)
- `bulan_ke` (1 = pendaftaran), `basis` (nominal pembayaran), `jenis`,
  `nilai`, `jumlah` (rupiah, dibulatkan ke bawah) — snapshot aturan saat dibuat
- `status` CHECK IN (`pending`, `siap`, `dibayar`, `batal`), `siap_at`,
  `dibayar_at`, `dibayar_oleh`, `metode_bayar` (`ujroh`/`transfer`),
  `ujroh_bulan` NULL, `alasan_batal`, `created_at`, `updated_at`

### 5.2 Atribusi

**Link affiliate** `GET /r/:kode[?ke=<slug>]`:
1. Affiliate tidak ada / tidak aktif → redirect ke tujuan tanpa cookie.
2. Hitung klik (dedup per IP+UA per hari, seperti link pendaftaran).
3. Cookie `rs_aff` = `kode|unix_ts|hmac` (HttpOnly, SameSite=Lax, Secure di
   https, Max-Age = masa berlaku). **First-click:** bila cookie sah yang
   belum kedaluwarsa sudah ada untuk affiliate lain, jangan ditimpa.
4. Redirect 302 ke `/daftar/<ke>` atau `link_tujuan_default`.

**Saat submit form** (`RegistrationService.Submit`): bila lead belum punya
`lead_affiliates`, kunci dengan urutan: cookie sah & belum kedaluwarsa →
kolom "Kode referal" (opsional, tampil bila diaktifkan per link) → tidak ada.
Affiliate tidak boleh mereferensikan nomor WA miliknya sendiri
(`affiliates.no_wa` == nomor pendaftar → tidak dikunci, dicatat).

**Mode WA**: `/r/KODE?ke=<slug-mode-wa>` → teks WA ditambah `(ref: KODE)`.
`lead_ingest` mendeteksi pola `ref: KODE` di pesan pertama → kunci
(`kode_wa`). Pola ini dihapus dari kecocokan sumber lead.

**Manual**: di halaman lead, CS/super_admin bisa set/ganti affiliate.

Lead yang sudah won sebelum fitur ini hidup tidak diatribusikan mundur.

### 5.3 Komisi (worker rekonsiliasi, tiap 10 menit + tombol "Hitung ulang")

1. **Pendaftaran:** lead dengan `lead_affiliates`, `verified_at` terisi,
   belum ada komisi `pendaftaran` → cari aturan (affiliate > program lead >
   global). Bila aturan `sekali` atau `bulanan` → buat komisi `bulan_ke=1`,
   basis = nominal `payment.verified` terakhir (fallback `payment_amount`).
   Hanya untuk `verified_at` setelah lead terkunci ke affiliate.
2. **SPP (pola bulanan):** tagihan `lunas` milik santri yang lead-nya
   punya affiliate, `bulan_ke` ≥ 2 dan ≤ `maks_bulan` (bila diisi), belum ada
   komisi `spp` untuk tagihan itu → buat komisi dengan aturan yang berlaku
   **saat komisi pendaftaran dibuat** (snapshot di komisi bulan_ke=1;
   perubahan aturan tidak berlaku mundur ke santri lama).
3. **Matang:** `pending` dengan `created_at + masa_tahan` ≤ sekarang →
   `siap`, kecuali santri `nonaktif`/`tidak_lanjut`, tagihan `batal`, atau
   lead `lost` → `batal` dengan alasan.
4. Affiliate `nonaktif`: komisi baru tidak dibuat; yang sudah ada tetap.

Persen: `jumlah = basis * nilai / 10000`. Flat: `jumlah = nilai`.

### 5.4 Pencairan

- **Guru** — saat bulan ujroh dibuat/dihitung (sebelum dikunci): semua
  komisi `siap` milik affiliate guru dijumlah → `ujroh_komponen_bulan`
  (komponen "Komisi Affiliate", jumlah = 1, tarif = total, catatan = jumlah
  santri), komisi → `dibayar` (`metode_bayar=ujroh`, `ujroh_bulan`). Bulan
  dikunci → masuk bulan berikutnya. Tombol di halaman ujroh: "Tarik komisi
  affiliate".
- **Eksternal** — halaman Keuangan → Komisi Affiliate: daftar `siap` per
  affiliate + rekening, tombol "Tandai sudah ditransfer" (bisa pilih
  beberapa), opsional catatan referensi.

### 5.5 Halaman

**Publik**
- `GET /affiliate/daftar` — form daftar eksternal (nama, WA, email,
  rekening, setuju syarat). Status `menunggu`, notifikasi super_admin.
- Setelah disetujui: akun `users` role `affiliate` dibuat, email berisi
  link set password (pakai alur reset password yang ada).

**Affiliate Saya** (`/app/affiliate`, role guru & affiliate)
- Link + kode, tombol salin & "Bagikan ke WhatsApp" (template pesan).
- Pilih tujuan link (dropdown link pendaftaran aktif).
- 4 angka 30 hari: klik, daftar, jadi santri, komisi (pending/siap/dibayar).
- Daftar rujukan: nama depan, program, tanggal, status komisi.
- Info aturan komisi yang berlaku untuknya.
- Eksternal: ubah data rekening.

**Admin** (`/app/leads/affiliate`, super_admin; keuangan baca + bayar)
- Daftar affiliate + persetujuan pendaftar eksternal.
- Aturan komisi: global, per program, per affiliate.
- Pengaturan: masa berlaku 30/60/90, masa tahan, link tujuan default.
- Leaderboard 30 hari.
- Buku komisi + filter + pencairan eksternal.
- Di detail lead: panel affiliate (siapa, sumber, ganti manual).

### 5.6 Keamanan & privasi

- Cookie ditandatangani HMAC (secret sesi); kode palsu/kedaluwarsa diabaikan.
- Affiliate hanya melihat nama depan & status rujukannya; tanpa nomor WA,
  tanpa chat.
- Role `affiliate` hanya boleh rute `/app/affiliate*` (+ logout, profil).
- Rate limit `/affiliate/daftar` dan `/r/:kode`.
- Perubahan atribusi manual & pencairan tercatat dengan pelaku.

### 5.7 Testing

Service test SQLite: urutan atribusi & first-click, cookie kedaluwarsa,
self-referral, aturan paling spesifik, persen/flat, sekali vs bulanan &
`maks_bulan`, idempoten rekonsiliasi (jalan 2× = sama), masa tahan & batal,
pencairan ke ujroh (bulan terkunci → bulan berikut), role guard.

## 6. Keputusan user

- Pendekatan A. Aturan di 3 level (global, program, affiliate).
- First-click. Masa tahan 14 hari. Batas bulanan diatur admin, default 6.
- Guru otomatis jadi affiliate; eksternal daftar → disetujui admin.
- Guru dibayar lewat ujroh; eksternal transfer manual.
- Masa berlaku 30 / 60 / 90 hari, dinamis.

## 7. Keputusan Claude (bisa dikoreksi)

- Komisi dihitung oleh worker rekonsiliasi, bukan hook di tiap tombol.
- Basis komisi pendaftaran = nominal pembayaran pertama terverifikasi;
  bulanan = nominal tagihan SPP lunas.
- Aturan di-snapshot saat komisi pendaftaran dibuat (tidak berlaku mundur).
- Komisi bulanan mulai `bulan_ke` 2 (bulan 1 = pembayaran pendaftaran).
- Tanpa aturan aktif = tanpa komisi (admin wajib membuat aturan global).
- Self-referral (nomor WA sama) tidak dikunci.
- Lead lama tidak diatribusikan mundur.

## 8. Pertanyaan terbuka

(diisi setelah builder check)
