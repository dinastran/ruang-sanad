# Form Pendaftaran Dinamis + Pembayaran Online — Design

Tanggal: 2026-11-19 · Status: draft, menunggu persetujuan
Proyek 1 dari 3 (1 = form + pembayaran, 2 = tagihan SPP via gateway, 3 = affiliate).

## 1. Intent

Saat ini semua calon mahasantri masuk lewat chat WhatsApp CS. Data pendaftar
(nama, usia, gender, paket, jadwal, dst.) dikumpulkan CS lewat chat, tahap lead
digeser manual (`form_dikirim` → `data_terisi` → `info_bayar_dikirim` →
`bukti_masuk` → `terverifikasi`), dan pembayaran dicek manual dari mutasi.

Yang diinginkan:
- Pendaftaran bisa lewat **tiga jalur** yang dipilih per link: WA langsung,
  short form, atau full form.
- Setelah form, pendaftar bisa **lanjut ke WA CS** atau **langsung bayar**
  (dipilih per link).
- Pembayaran online lewat **payment gateway** (QRIS, e-wallet, VA) dengan
  **verifikasi otomatis**, plus transfer manual sebagai cadangan.

Kenapa sekarang:
- Menambah santri baru dari kanal di luar Meta Ads.
- Menjadi fondasi atribusi affiliate (proyek 3): link affiliate → form → lead
  terkunci ke affiliate, dan komisi dipicu oleh pembayaran terverifikasi
  yang datang dari data gateway, bukan input manual.
- Mengurangi kerja CS: input data dan cek mutasi.

## 2. Goals / Non-goals / Anti-goals

**Goals**
- Admin bisa membuat banyak "link pendaftaran", masing-masing dengan mode,
  program, aksi setelah submit, nominal, judul, dan teks sendiri.
- Pendaftar menyelesaikan short form < 1 menit di HP.
- Lead dari form dan lead dari chat WA dengan nomor yang sama = **satu lead**.
- Pembayaran via gateway otomatis membuat lead `terverifikasi` tanpa sentuhan CS.
- Statistik per link: klik, submit, bayar.

**Non-goals (proyek ini)**
- Form builder bebas (pertanyaan custom). Field ditentukan sistem, admin hanya
  atur tampil/wajib.
- Konversi otomatis lead → santri. Tetap manual oleh staf (`Convert`), karena
  butuh penempatan kelas.
- Tagihan SPP bulanan via gateway (proyek 2), affiliate (proyek 3).
- Kartu kredit, paylater, cicilan.

**Anti-goals (gagal walau "berfungsi")**
- Data lead ganda per nomor WA.
- Webhook palsu/terkirim ulang yang menandai lunas tanpa uang masuk, atau
  memverifikasi dua kali.
- Spam submit yang membanjiri CRM dan mengirim event konversi palsu ke Meta CAPI.
- Pendaftar memilih jadwal yang kuotanya sudah penuh.
- Halaman publik berat/lambat di HP murah.

## 3. Constraints (dari kode yang ada)

- Three-tier rule: Handler → Service → Query (sqlc) → DB. Handler terpisah per
  fitur. Migrasi satu file = satu tabel; jangan edit migrasi lama.
- `leads.wa_chat_id` UNIQUE, berformat `62xxxxxxxxxx@s.whatsapp.net`
  (`lead_ingest.go`, `lead_parser.go`). Dedup lead dari form memakai nomor yang
  dinormalisasi ke format yang sama.
- Perubahan tahap lewat `LeadService.mutateLeadAs(ctx, id, actor, prefix, fn)`
  (`lead_crm.go:411`), yang juga menulis `lead_events` dan menjadwalkan event
  Meta CAPI (`meta_capi_events`, INSERT OR IGNORE → idempoten per event_id).
- Tahap `data_terisi`, `info_bayar_dikirim`, `bukti_masuk` memicu event Meta
  CAPI. `terverifikasi` hanya bisa dicapai lewat `VerifyPayment`
  (`lead_closing.go:31`), bukan `ChangeStage`.
- Lead sudah punya `program_id` (→ `lead_programs` → `produk.harga`),
  `jadwal_kode` (→ `jadwal_penawaran`, kuota per gender), `payment_amount`,
  `proof_received_at`, `verified_at`.
- `jadwal_penawaran`: `is_buka`, `kuota_per_gender`, `level`, `hari`, jam.
- Upload yang ada (`upload.go`) adalah TUS untuk user login. Upload bukti di
  form publik butuh endpoint sendiri yang dibatasi.
- Frontend Svelte 5 + Inertia. Token warna di `frontend/src/app.css` `@theme`.
  Ikuti design principles wiki sebelum menulis halaman.
- Payment gateway: Ruang Sanad sudah berbadan hukum (yayasan/PT/CV), sehingga
  Xendit/Midtrans bisa aktif penuh. Nominal tipikal Rp100–500 rb, dengan MDR
  QRIS sektor pendidikan 0,6% (kebijakan BI, berlaku 1 Okt 2026).

## 4. Approaches

Dipertimbangkan:

| | Pendekatan | Trade-off |
|-|-|-|
| **A (dipilih)** | Halaman publik Svelte/Inertia + tabel `registration_links` | Satu stack, komponen & validasi konsisten, pratinjau admin mudah. Bundle JS sedikit lebih berat, ditekan dengan code-splitting per halaman. |
| B | Halaman publik templ (server-rendered, minim JS) | Paling ringan, tapi dua gaya UI, dan interaksi (jadwal per gender, upload + preview, status bayar) harus ditulis ulang tanpa Svelte. |
| C | Google Form + webhook | Ditolak: tidak bisa dedup per nomor, tidak membaca kuota jadwal, tidak bisa mengunci affiliate. |

Dinamis = **preset per link** (dipilih user), bukan form builder bebas.

Gateway:

| | Opsi | Trade-off |
|-|-|-|
| **Xendit (dipilih)** | Invoice API (hosted checkout) | QRIS, e-wallet, dan VA dalam satu halaman bayar. Punya **disbursement** untuk membayar affiliate eksternal di proyek 3. |
| Midtrans | Snap | Setara untuk menerima pembayaran. Dipilih bila lembaga sudah punya akun Midtrans. |
| Aggregator kecil (Tripay/Duitku) | — | Syarat lebih longgar, tapi tidak diperlukan karena lembaga sudah berbadan hukum. |

Integrasi via **hosted checkout**, bukan API per metode. Kode memanggil satu
interface `PaymentGateway` (buat invoice, verifikasi webhook), sehingga ganti
vendor = satu implementasi baru.

## 5. Design

### 5.1 Data model (migrasi baru, satu tabel per file)

**`registration_links`**
- `id`, `slug` TEXT UNIQUE (huruf kecil, angka, `-`), `judul`, `deskripsi`
- `program_id` → `lead_programs` (wajib)
- `mode` TEXT CHECK IN (`wa`, `short`, `full`)
- `after_submit` TEXT CHECK IN (`wa`, `bayar`) — diabaikan bila mode = `wa`
- `nominal_override` INTEGER NULL. NULL = pakai `produk.harga` program.
- `field_config` TEXT JSON: per field opsional `{"tampil":bool,"wajib":bool}`.
  Field inti (nama, no WA) selalu tampil & wajib.
- `wa_template` TEXT: template pesan pembuka ke CS, dengan placeholder
  `{nama}`, `{program}`, `{jadwal}`, `{ref}`
- `izinkan_manual` INTEGER (tampilkan opsi transfer manual di halaman bayar)
- `is_aktif`, `created_by`, `created_at`, `updated_at`

**`registration_link_stats`** (agregat harian, ringan)
- `link_id`, `tanggal` (WIB), `klik`, `submit`, `bayar`. PK (`link_id`, `tanggal`)
- Klik dihitung per kunjungan halaman/redirect. Dedup sederhana per IP+UA per
  hari via cache in-memory. Angka ini indikatif, bukan audit.

**`lead_registrations`** (jejak submit form; satu lead bisa submit berkali-kali)
- `id`, `lead_id`, `link_id`, `payload` JSON (jawaban mentah), `ip`, `user_agent`,
  `created_at`

**`payments`**
- `id`, `lead_id`, `link_id` NULL, `metode` TEXT (`gateway` | `manual`)
- `provider` (`xendit`), `external_id` UNIQUE (dibuat kita), `provider_ref`
  (invoice id), `checkout_url`
- `amount`, `status` CHECK IN (`pending`, `paid`, `expired`, `failed`,
  `menunggu_verifikasi`, `ditolak`)
- `paid_at`, `expires_at`, `channel` (QRIS/OVO/BCA_VA…, dari webhook)
- `bukti_path` (manual), `raw_webhook` JSON terakhir
- `created_at`, `updated_at`
- Dirancang generik (bukan khusus lead), supaya proyek 2 bisa menambahkan
  kolom `tagihan_id` lewat migrasi baru.

**`crm_settings`** (migrasi tambah kolom)
- `cs_wa_number` (nomor tujuan wa.me), `payment_expiry_hours` (default 24),
  `bank_manual_info` (teks rekening untuk transfer manual).
- Kredensial Xendit (secret key, webhook token) di **env/config**, bukan DB.

### 5.2 Halaman publik

Route publik (tanpa login, di luar `/app`):
- `GET /daftar/:slug` — halaman sesuai mode link. Link nonaktif/tidak ada →
  halaman "link tidak aktif" + tombol WA CS.
- `POST /daftar/:slug` — submit form.
- `GET /daftar/:slug/bayar/:ref` — halaman status/pilihan bayar.
  `ref` = token acak (bukan id berurutan).
- `POST /daftar/:slug/bayar/:ref/gateway` — buat/ambil invoice aktif → redirect.
- `POST /daftar/:slug/bayar/:ref/manual` — upload bukti transfer.
- `POST /webhooks/xendit` — callback gateway (CSRF dikecualikan, token diverifikasi).

**Mode `wa`:** `GET /daftar/:slug` mencatat klik lalu redirect 302 ke
`https://wa.me/<cs_wa_number>?text=<wa_template>`. Tidak ada form.

**Short form:** nama, no WA, gender, usia, domisili, pilihan paket (bila program
punya >1), jadwal.
**Full form:** short + riwayat belajar, belajar di mana, untuk siapa, waktu
belajar. Field opsional mengikuti `field_config`.

**Pilihan jadwal:** hanya `jadwal_penawaran` dengan `is_buka = 1` yang sisa
kuotanya untuk gender terpilih > 0. Sisa kuota dihitung dengan definisi
okupansi yang sudah dipakai fitur jadwal penawaran (builder memakai ulang
fungsinya, bukan menulis rumus baru). Daftar jadwal dimuat ulang setelah gender
dipilih. Kuota dicek ulang di server saat submit; bila penuh → error ramah +
minta pilih jadwal lain.

**Setelah submit:**
- `after_submit = wa`: halaman sukses + tombol besar "Lanjut ke WhatsApp"
  (wa.me dengan ringkasan data + `ref` lead `L-<id>`), dan auto-redirect.
- `after_submit = bayar`: redirect ke halaman bayar.

**Halaman bayar:** ringkasan (nama, program, jadwal, nominal). Tombol utama
"Bayar sekarang (QRIS / e-wallet / VA)" → invoice Xendit. Bila `izinkan_manual`,
tampil link sekunder "Transfer manual" → info rekening + upload bukti. Halaman
ini juga menampilkan status (menunggu / lunas / kedaluwarsa → tombol buat
tagihan baru) supaya pendaftar bisa kembali lewat link yang sama.

### 5.3 Alur lead (service)

`RegistrationService.Submit(slug, input, meta)`:
1. Validasi input + honeypot + rate limit. Normalisasi nomor
   (`08…`/`+62…`/`62…` → `62…`), lalu `wa_chat_id = <nomor>@s.whatsapp.net`.
2. Dalam satu transaksi: cari lead by `wa_chat_id`.
   - Tidak ada → buat lead baru (`source = form`).
   - Ada, status `open` → update field yang **kosong** saja (data dari CS/AI
     tidak ditimpa), lalu set `program_id`/`jadwal_kode` bila kosong.
   - Ada, status `won`/`lost`/`do_not_contact` → lihat §8 (keputusan builder).
3. Geser tahap ke `data_terisi` bila tahap saat ini lebih awal (jangan pernah
   mundur), lewat `mutateLeadAs` dengan actor `form:<link_id>`, sehingga
   `lead_events` + Meta CAPI ikut seperti biasa.
4. Tulis `lead_registrations`, increment `submit` di stats.
5. Bila `after_submit = bayar`: set tahap `info_bayar_dikirim` dan
   `payment_amount`.

`PaymentService`:
- `CreateGatewayInvoice(leadID)`: bila ada payment `pending` yang belum
  kedaluwarsa → kembalikan `checkout_url`-nya (jangan buat ganda). Kalau tidak,
  buat `payments` row + panggil `PaymentGateway.CreateInvoice(external_id,
  amount, deskripsi, expiry, success/failure redirect URL)`.
- `HandleWebhook(headers, body)`:
  1. Verifikasi `x-callback-token` == token config (constant-time compare).
     Gagal → 401, tidak ada efek.
  2. Cari payment by `external_id`. Tidak ada → 200 + log (jangan retry loop).
  3. Idempoten: status sudah `paid` → 200 tanpa efek.
  4. Cocokkan `amount` dengan record. Beda → tandai `failed` + notifikasi admin,
     **jangan** verifikasi.
  5. Status `PAID`/`SETTLED` → `payments.status = paid`, lalu panggil jalur
     verifikasi lead yang sama dengan `VerifyPayment` (refactor inti
     verifikasinya menjadi fungsi bersama yang bisa dipanggil dengan actor
     sistem `gateway:xendit`). Lead → `terverifikasi`, CAPI Purchase ikut.
     Increment `bayar` di stats.
  6. `EXPIRED` → `payments.status = expired`.
  Semua langkah dalam satu transaksi DB.
- `SubmitManualProof(ref, file)`: simpan file, `payments(metode=manual,
  status=menunggu_verifikasi)`, lead → `bukti_masuk`, notifikasi ke role
  keuangan. CS memverifikasi lewat tombol `VerifyPayment` yang sudah ada.
  Payment manual ikut ditandai `paid`/`ditolak`.

### 5.4 Admin

Handler terpisah `app/handlers/registration_links.go`. Menu di bawah **Leads**:
"Link Pendaftaran". Role: `super_admin` + role yang sudah boleh mengelola
sumber/program lead (ikuti `programRole`).
- Daftar link: judul, mode, program, nominal, status, klik/submit/bayar 30 hari
  + rasio konversi, tombol salin link & buka.
- Form buat/ubah link + **pratinjau langsung** di sisi kanan (desktop) /
  tab (mobile).
- Pengaturan di `crm_settings`: nomor WA CS, masa berlaku tagihan, info rekening.
- Di halaman detail lead: panel "Pendaftaran & Pembayaran" (submit terakhir via
  link apa, daftar payments + status + channel + bukti manual).

### 5.5 Keamanan & keandalan

- Anti-spam: field honeypot, waktu isi minimal (≥ 3 detik sejak render, via
  token bertanda tangan), rate limit per IP (mis. 5 submit / 10 menit) dan per
  nomor WA (mis. 3 / jam). Cloudflare Turnstile **opsional** via config, mati
  secara default.
- Upload bukti manual: hanya JPG/PNG/WebP/PDF, ≤ 2 MB, cek magic bytes (bukan
  hanya ekstensi), nama file acak, disimpan di luar folder publik dan hanya
  bisa dibuka staf yang login.
- Webhook: hanya POST, token wajib, body ≤ 64 KB, idempoten, transaksional.
  Handler selalu membalas cepat.
- `ref` halaman bayar: token acak ≥ 128-bit, bukan `lead_id`.
- Data pribadi di halaman publik: halaman bayar hanya menampilkan nama depan +
  program + nominal.
- Nominal dihitung di server (override link → harga produk). Tidak pernah
  diambil dari input pendaftar.

### 5.6 Rilis bertahap

- **Fase 1a:** registration_links, halaman publik (wa/short/full), dedup lead,
  stats, halaman bayar **manual** (upload bukti), admin. Bisa dipakai tanpa
  akun gateway.
- **Fase 1b:** Xendit (sandbox → live), webhook, verifikasi otomatis, status
  bayar realtime (polling ringan di halaman bayar).

Pendaftaran akun & KYC Xendit dikerjakan paralel oleh lembaga selama fase 1a.

### 5.7 Testing

- Service test (SQLite in-memory): dedup per nomor (berbagai format),
  tidak menimpa data CS, tahap tidak mundur, kuota penuh ditolak,
  nominal override vs harga produk.
- Webhook: token salah → 401; replay → tanpa efek ganda; amount beda → tidak
  verifikasi; PAID → lead terverifikasi + satu event CAPI Purchase.
- Gateway di-stub lewat interface `PaymentGateway` (tidak ada panggilan
  jaringan di test).
- E2E agent_browser: isi short form → lanjut WA; isi → bayar manual → upload.

## 6. Keputusan yang diambil user

- Pendekatan A (Svelte/Inertia + tabel link).
- Dinamis = preset per link (bukan form builder).
- Aksi setelah submit dipilih per link (WA atau bayar).
- Short form = nama + WA, gender + usia + domisili, program/paket + jadwal.
- Pakai payment gateway. Xendit dipilih, hosted checkout, QRIS default.
- Gateway untuk pendaftaran dulu; SPP menyusul (proyek 2).
- Urutan proyek: form + pembayaran → SPP gateway → affiliate.

## 7. Keputusan Claude (bisa dikoreksi)

- Nominal = `produk.harga` program, dengan `nominal_override` per link.
- Masa berlaku invoice 24 jam (diatur di `crm_settings`).
- Transfer manual + upload bukti tetap tersedia sebagai cadangan (per link).
- Form tidak menimpa field lead yang sudah terisi.
- Kredensial gateway di env, bukan DB.
- Turnstile opsional, mati secara default.

## 8. Diserahkan ke builder

- Perilaku bila nomor sudah jadi lead `won`/`lost`/`do_not_contact` (lihat
  pertanyaan terbuka di bawah).
- Struktur komponen Svelte, nama route admin, detail UI mengikuti
  design principles + token `app.css`.
- Mekanisme cache dedup klik.
- Detail mapping status Xendit → `payments.status`.

## 9. Pertanyaan terbuka

(diisi setelah builder check)
