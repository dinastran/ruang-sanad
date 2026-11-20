# Affiliate / Referral — Design

Tanggal: 2026-11-19 · Status: siap disetujui (builder check selesai)
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
- `guru.user_id` → `users` (unik). Satu role per user (`users.role`), daftar
  role di `models.ValidRoles()` (app/models/dto.go) dan halaman awal per role
  di `roleHome` (app/handlers/auth.go). Role baru `affiliate` menyentuh
  keduanya. Guru bisa ber-role `guru`, `koordinator_guru`, atau
  `super_admin`; sebagian guru belum punya `user_id`.
- Lead → santri: `leads.santri_id`. Pembayaran pertama terverifikasi:
  `leads.verified_at` (staf atau gateway, `verifyLeadTx`).
- SPP: `tagihan` (`santri_id`, `bulan_ke`, `nominal`, `status`
  belum_bayar/lunas/batal, `tanggal_bayar`, `tanggal_tagih`). Satu-satunya
  jalur lunas: `TagihanService.MarkLunas`. Lunas tidak bisa jadi batal.
  **Bulan 1 dibayar di muka saat pendaftaran dan tidak punya baris
  tagihan**; tagihan pertama = `bulan_ke` 2. `bulan_ke` berlanjut dari
  riwayat santri lama (lead yang ditautkan ke santri yang sudah ada).
- Lead won tidak bisa menjadi lost. Memundurkan tahap dari `terverifikasi`
  mengosongkan `verified_at` (lead_crm.go). Program pendek tidak membuat
  santri.
- Password reset: token 1 jam, email lewat SMTP (opsional, bisa kosong).
  WA keluar lewat `wa_outbox` (CS bridge).
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
yang lewat 14 hari, dan membatalkan yang sumbernya hilang. Alasan:
verifikasi datang dari staf maupun gateway, dan verifikasi bisa dibatalkan
dengan memundurkan tahap; rekonsiliasi idempoten dari data menangani semua
itu tanpa mengubah alur keuangan yang ada.

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
`lead_ingest` mendeteksi pola `ref: KODE` di **pesan masuk mana pun selama
lead belum punya affiliate** (lead lama/lost yang kembali ikut terdeteksi)
→ kunci (`kode_wa`). Pola ini tidak memengaruhi pencocokan sumber lead.

**Manual**: di halaman lead, CS/super_admin bisa set/ganti affiliate.

Lead yang sudah won sebelum fitur ini hidup tidak diatribusikan mundur.

### 5.3 Komisi (worker rekonsiliasi, tiap 10 menit + tombol "Hitung ulang")

1. **Pendaftaran:** lead dengan `lead_affiliates`, `verified_at` terisi
   (setelah `dikunci_at`), belum ada komisi `pendaftaran` aktif → cari aturan
   (affiliate > program lead > global) → buat komisi `bulan_ke=1`, basis =
   nominal `payment.verified` terakhir (fallback `payment_amount`). Berlaku
   untuk semua program (reguler & pendek); admin bisa membuat aturan program
   bernilai 0 untuk mengecualikan program tertentu. Lead yang ditautkan ke
   santri lama tetap dihitung sebagai pendaftaran baru (keputusan user).
2. **SPP (pola bulanan):** tagihan `lunas` milik santri dari lead ber-affiliate,
   dengan `tanggal_tagih` **setelah** `leads.converted_at` (tagihan lama tidak
   pernah ikut). Urutan bulan dihitung dari konversi: tagihan pertama setelah
   konversi = bulan ke-2, dst. — bukan `tagihan.bulan_ke` (santri lama bisa
   berlanjut dari bulan 14). Dibuat selama urutan ≤ `maks_bulan` (bila
   diisi). Aturan = snapshot di komisi pendaftaran (tidak berlaku mundur).
3. **Matang & batal (hanya selama `pending`):**
   - `verified_at` lead kosong lagi (verifikasi dibatalkan) → komisi
     pendaftaran `batal`. Bila lead diverifikasi ulang, baris yang sama
     diaktifkan kembali (`pending`, `created_at` baru, basis baru) — tidak
     membuat baris kedua.
   - Santri `nonaktif`/`tidak_lanjut` dalam masa tahan → `batal`.
   - Lewat `masa_tahan_hari` tanpa pembatalan → `siap`.
   Komisi `siap`/`dibayar` tidak pernah dibatalkan otomatis; koreksi oleh
   admin (batal manual dengan alasan, hanya sebelum dibayar).
4. Affiliate `nonaktif`: komisi baru tidak dibuat; yang sudah ada tetap.

Persen: `jumlah = basis * nilai / 10000`. Flat: `jumlah = nilai`.

### 5.4 Pencairan

- **Guru (keputusan user: otomatis saat bulan dikunci)** — ujroh dihitung
  live. Komisi `siap` milik affiliate guru tampil di ujroh bulan berjalan
  sebagai baris sistem "Komisi Affiliate" (`ujroh_komponen_bulan`, komponen
  khusus, **tidak bisa diubah/dihapus** lewat `SetKomponenBulan` /
  `HapusKomponenBulan`). Saat keuangan `Kunci` bulan itu: komisi terkait →
  `dibayar` (`metode_bayar=ujroh`, `ujroh_bulan`). Saat `Buka` bulan itu:
  komisi → kembali `siap` dan baris dihitung ulang. Komisi yang matang
  setelah bulan dikunci masuk bulan berikutnya.
- **Eksternal** — halaman Keuangan → Komisi Affiliate: daftar `siap` per
  affiliate + rekening, tombol "Tandai sudah ditransfer" (bisa pilih
  beberapa), opsional catatan referensi.

### 5.5 Halaman

**Publik**
- `GET /affiliate/daftar` — form daftar eksternal (nama, WA, email,
  rekening, setuju syarat). Status `menunggu`, notifikasi super_admin.
- Setelah disetujui: akun `users` role `affiliate` dibuat dan link "buat
  password" (token 72 jam, sekali pakai) dikirim lewat WhatsApp
  (`wa_outbox`) dan email bila SMTP aktif. Admin juga bisa menyalin link.
  Email sudah dipakai user lain: bila role `user` → diubah ke `affiliate`;
  bila role staf → persetujuan ditolak dengan pesan ke admin.

**Affiliate Saya** (`/app/affiliate`) — akses ditentukan oleh **punya baris
`affiliates` aktif** yang terhubung ke user (guru apa pun rolenya, atau role
`affiliate`), bukan oleh daftar role. Guru otomatis dibuatkan baris
affiliate saat punya `user_id`.
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
- Role `affiliate`: penjaga khusus (deny-guard) di grup `/app` yang hanya
  mengizinkan `/app/affiliate*`, profil, logout, dan aset; rute `/app`
  lain yang terbuka untuk semua user login (dashboard, tasbih, ai-akses,
  upload) ditolak untuk role ini. Halaman awal role ini `/app/affiliate`.
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
- Guru dibayar lewat ujroh, otomatis tercatat dibayar saat bulan dikunci;
  eksternal transfer manual.
- Santri lama yang kembali lewat affiliate = pendaftaran baru (komisi
  dihitung dari tagihan setelah daftar ulang).
- Masa berlaku 30 / 60 / 90 hari, dinamis.

## 7. Keputusan Claude (bisa dikoreksi)

- Komisi dihitung oleh worker rekonsiliasi, bukan hook di tiap tombol.
- Basis komisi pendaftaran = nominal pembayaran pertama terverifikasi;
  bulanan = nominal tagihan SPP lunas.
- Aturan di-snapshot saat komisi pendaftaran dibuat (tidak berlaku mundur).
- Komisi bulanan dihitung dari tagihan setelah konversi, urutan bulan sejak
  konversi (bukan `tagihan.bulan_ke`).
- Pembatalan verifikasi → komisi batal; verifikasi ulang mengaktifkan baris
  yang sama.
- Komisi berlaku untuk semua program; pengecualian lewat aturan program 0.
- Akses "Affiliate Saya" berdasarkan baris affiliate, bukan role.
- Ref WA berlaku di pesan mana pun selama lead belum ber-affiliate.
- Akun eksternal: link buat-password 72 jam via WA + email (bila SMTP aktif).
- Baris "Komisi Affiliate" di ujroh dikelola sistem, tidak bisa diedit.
- Tanpa aturan aktif = tanpa komisi (admin wajib membuat aturan global).
- Self-referral (nomor WA sama) tidak dikunci.
- Lead lama tidak diatribusikan mundur.

## 8. Risiko yang diterima

- Cookie hilang saat ganti perangkat/browser → ditutup sebagian oleh kode
  di form, ref WA, dan set manual CS.
- Komisi yang sudah `siap`/`dibayar` tidak ditarik otomatis bila santri
  berhenti setelah masa tahan.
