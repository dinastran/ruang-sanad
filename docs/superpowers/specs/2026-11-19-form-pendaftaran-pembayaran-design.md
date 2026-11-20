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
- Tahap lain digeser oleh worker `MetaCAPIService.reconcile` (meta_capi.go),
  yang membaca `lead_events` perubahan tahap dan menjadwalkan event
  CompleteRegistration / InitiateCheckout / AddPaymentInfo. `mutateLeadAs`
  sendiri tidak menulis ke `meta_capi_events`.
- **Purchase** saat ini hanya di-enqueue saat konversi (`Convert`
  lead_closing.go, `DaftarkanPeserta` lead_program.go) lewat `enqueuePurchase`,
  bukan saat `VerifyPayment`.
- `VerifyPayment` hanya menerima lead `open` di tahap `bukti_masuk`
  (atau `info_bayar_dikirim` + `NoProof`), dan mengisi `VerifiedBy` dengan
  user id + tanggal yang diketik staf. Belum bisa dipanggil oleh sistem.
- Lead sudah punya `program_id` (→ `lead_programs.produk_id`, **boleh NULL**;
  program bawaan Tahsin Reguler id 1 belum punya produk), `paket_kode`
  (`1x`/`2x` → `kelas_kode` via `crm_settings.paket_ke_kelas_kode`),
  `jadwal_kode` (→ `jadwal_penawaran`, kuota per gender), `payment_amount`,
  `proof_received_at`, `verified_at`. Nominal Reguler hari ini diketik CS di
  chat (`ParseRupiah` di lead_ingest.go).
- `produk` (0036, 0056): `nama`, `kategori` CHECK IN (`buku`, `program`),
  satu `harga`, punya `produk_batch`. CRUD sudah ada di `/app/produk-crm`.
- `jadwal_penawaran`: `is_buka`, `kuota_per_gender`, `level`, `hari`, jam.
  Kursi terpakai = lead dengan `verified_at` terisi dan status open/won
  (`ListVerifiedLeadSeats`). Tidak ada penahanan kursi.
- Chat WA format `@lid` tidak membawa nomor HP (`phoneFromChatID` → "").
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

### 5.0 Produk & varian harga (dinamis)

Harga pendaftaran berasal dari **produk**, bukan diketik. Admin bisa membuat
produk apa saja (Tahsin Reguler, Tahfidz, program pendek, kelas anak, dll.)
lewat menu Produk CRM yang sudah ada, ditambah **varian harga**:

- Produk tanpa varian aktif → harga = `produk.harga`.
- Produk dengan ≥ 1 varian aktif → pendaftar memilih varian di form, harga =
  `produk_varian.harga`.
- Varian boleh membawa `paket_kode` (mis. `1x`/`2x`) yang disalin ke
  `leads.paket_kode`, sehingga mapping ke `kelas_kode` lama tetap jalan.
- Program (`lead_programs`) yang dipakai link pendaftaran **wajib** punya
  `produk_id`. Admin menautkan Tahsin Reguler ke produk "Tahsin Reguler"
  lewat UI program yang ada. Tidak ada seed harga (harga diisi admin).
- Kategori produk tetap `buku`/`program`. Produk pendaftaran memakai
  `program`, jadi tidak perlu rebuild tabel untuk CHECK.

### 5.1 Data model (migrasi baru, satu tabel per file)

**`produk_varian`**
- `id`, `produk_id` → `produk` (CASCADE), `nama` (mis. "1x / pekan"),
  `harga` INTEGER ≥ 0, `paket_kode` TEXT default '', `urutan`, `is_aktif`,
  `created_at`, `updated_at`

**`registration_links`**
- `id`, `slug` TEXT UNIQUE (huruf kecil, angka, `-`), `judul`, `deskripsi`
- `program_id` → `lead_programs` (wajib; program harus punya `produk_id`)
- `lead_source_id` → `lead_sources` (wajib; dipakai laporan iklan & atribusi)
- `mode` TEXT CHECK IN (`wa`, `short`, `full`)
- `after_submit` TEXT CHECK IN (`wa`, `bayar`) — diabaikan bila mode = `wa`
- `nominal_override` INTEGER NULL. NULL = harga varian/produk (§5.0).
  Override berlaku untuk semua varian (mis. harga promo flat).
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
- `id`, `lead_id`, `link_id` NULL, `produk_varian_id` NULL,
  `metode` TEXT (`gateway` | `manual`)
- `provider` (`xendit`), `external_id` UNIQUE (dibuat kita), `provider_ref`
  (invoice id), `checkout_url`
- `amount`, `status` CHECK IN (`pending`, `paid`, `expired`, `failed`,
  `menunggu_verifikasi`, `ditolak`)
- `paid_at`, `expires_at`, `channel` (QRIS/OVO/BCA_VA…, dari webhook)
- `jadwal_kode`, `jenis_kelamin` (snapshot untuk penahanan kursi, §5.3)
- `perlu_tinjau` INTEGER + `catatan_sistem` (lunas tapi lead tidak bisa
  diverifikasi otomatis, atau kursi kelebihan)
- `bukti_path` (manual), `raw_webhook` JSON terakhir
- `created_at`, `updated_at`
- Dirancang generik (bukan khusus lead), supaya proyek 2 bisa menambahkan
  kolom `tagihan_id` lewat migrasi baru.

**`crm_settings`** (migrasi tambah kolom)
- `cs_wa_number` (nomor tujuan wa.me), `payment_expiry_hours` (default **2**,
  pendek karena kursi ditahan selama tagihan aktif),
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

**Short form:** nama, no WA, gender, usia, domisili, pilihan varian (bila
produk punya >1 varian aktif), jadwal (**hanya untuk program jenis `reguler`**;
program jenis `program` memakai sesi/batch, sehingga tidak ada pilihan jadwal).
**Full form:** short + riwayat belajar, belajar di mana, untuk siapa, waktu
belajar. Field opsional mengikuti `field_config`.

**Pilihan jadwal:** hanya `jadwal_penawaran` dengan `is_buka = 1` yang sisa
kuotanya untuk gender terpilih > 0. Sisa kuota = kuota − (kursi terverifikasi
menurut definisi yang sudah ada) − (kursi **ditahan**: `payments` status
`pending` yang belum kedaluwarsa dengan `jadwal_kode` + `jenis_kelamin` sama).
Builder memperluas fungsi okupansi yang ada, bukan menulis rumus kedua. Daftar jadwal dimuat ulang setelah gender
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
1. Validasi input + honeypot + rate limit. Normalisasi nomor dengan
   `NormalizeLeadPhone` (`08…`/`+62…`/`62…` → `62…`).
2. Dalam satu transaksi: cari lead by `wa_chat_id = <nomor>@s.whatsapp.net`,
   lalu fallback by `leads.phone` (perlu index baru pada `phone`).
   - Tidak ada → buat lead baru dengan `wa_chat_id = <nomor>@s.whatsapp.net`,
     `source = form`, `lead_source_id` dari link.
   - Ada, status `open` → update field yang **kosong** saja (data dari CS/AI
     tidak ditimpa), lalu set `program_id`/`paket_kode`/`jadwal_kode` bila kosong.
   - Ada, status `lost` → buka kembali (`open`) + event `lead.reopened` dengan
     actor form, lalu lanjut seperti `open`.
   - Ada, status `won` (sudah santri) atau `do_not_contact` → tidak membuat
     lead baru, tidak mengubah tahap. Submit dicatat di `lead_registrations`,
     lead diberi `needs_review = 1`, dan pendaftar diarahkan ke WA CS
     (bukan halaman bayar).
   - Batas yang diterima: bila orang yang sama chat dari akun WA `@lid`
     (tanpa nomor), bridge bisa membuat lead kedua. Tidak diselesaikan di
     proyek ini. Pesan pembuka WA dari form membawa `L-<id>`, sehingga CS/AI
     bisa menautkannya. Kasus ini dicatat sebagai risiko.
3. Geser tahap ke `data_terisi` bila tahap saat ini lebih awal (jangan pernah
   mundur), lewat `mutateLeadAs` dengan actor `form:<link_id>`. `lead_events`
   tertulis, dan worker CAPI `reconcile` mengirim CompleteRegistration seperti
   tahap manual. Submit yang ditolak anti-spam tidak pernah menyentuh lead,
   sehingga tidak ada event CAPI palsu.
4. Tulis `lead_registrations`, increment `submit` di stats.
5. Bila `after_submit = bayar`: set tahap `info_bayar_dikirim` dan
   `payment_amount`.

`PaymentService`:
- `CreateGatewayInvoice(leadID)`: bila ada payment `pending` yang belum
  kedaluwarsa → kembalikan `checkout_url`-nya (jangan buat ganda). Kalau tidak,
  dalam satu transaksi: cek ulang sisa kursi (jadwal + gender). Penuh → tolak
  dan minta pilih jadwal lain. Ada → buat `payments` row (snapshot
  `jadwal_kode`, `jenis_kelamin`; ini yang **menahan kursi**). Lalu panggil
  `PaymentGateway.CreateInvoice(external_id, amount, deskripsi, expiry,
  success/failure redirect URL)`. Bila panggilan gateway gagal → payment
  `failed`, kursi lepas.
- Kursi dilepas otomatis saat `expires_at` lewat (query okupansi mengabaikan
  pending yang kedaluwarsa, sehingga tidak bergantung pada webhook EXPIRED).
- `HandleWebhook(headers, body)`:
  1. Verifikasi `x-callback-token` == token config (constant-time compare).
     Gagal → 401, tidak ada efek.
  2. Cari payment by `external_id`. Tidak ada → 200 + log (jangan retry loop).
  3. Idempoten: status sudah `paid` → 200 tanpa efek.
  4. Cocokkan `amount` dengan record. Beda → tandai `failed` + notifikasi admin,
     **jangan** verifikasi.
  5. Status `PAID`/`SETTLED` → `payments.status = paid`, increment `bayar`.
     Lalu verifikasi lead lewat **inti verifikasi bersama** (refactor dari
     `VerifyPayment`): fungsi yang menerima actor string (`user:<id>` atau
     `gateway:xendit`), nominal, dan tanggal, dipakai oleh tombol staf maupun
     webhook. Untuk actor sistem, `verified_by` NULL dan actor tercatat di
     `lead_events`.
     - Lead `open` di tahap mana pun sebelum `terverifikasi` → `terverifikasi`.
       Pembayaran gateway adalah bukti, sehingga syarat tahap `bukti_masuk`
       hanya berlaku untuk jalur staf.
     - Lead tidak `open`, atau sudah terverifikasi → **jangan** ubah lead;
       payment tetap `paid` + `perlu_tinjau = 1` + notifikasi keuangan.
     - **Purchase dikirim saat uang masuk** (keputusan user):
       `enqueuePurchase` dipanggil di sini dengan nominal dari payment.
       Event id Purchase sudah deterministik per lead
       (`metaEventID(id, MetaEventPurchase)`, lead_closing.go) dan insert-nya
       INSERT OR IGNORE, sehingga `Convert` / `DaftarkanPeserta` yang memanggil
       `enqueuePurchase` lagi tidak membuat event ganda.
     - Re-cek kursi: bila jadwal ternyata penuh (mis. kuota diturunkan admin),
       pembayaran tetap diterima, lead tetap terverifikasi, payment
       `perlu_tinjau = 1` + notifikasi koordinator ("kursi kelebihan").
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
- Produk CRM: tambah kelola **varian harga** per produk (nama, harga,
  paket_kode, urutan, aktif).
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

- Service test (SQLite in-memory): dedup per nomor (berbagai format, match
  via `phone`), tidak menimpa data CS, tahap tidak mundur, lead won/DNC tidak
  disentuh, lead lost dibuka kembali, kuota penuh ditolak, kursi ditahan oleh
  pending dan lepas setelah kedaluwarsa, harga: override > varian > produk.
- Webhook: token salah → 401; replay → tanpa efek ganda; amount beda → tidak
  verifikasi; PAID → lead terverifikasi + tepat satu event CAPI Purchase,
  dan Convert sesudahnya tidak menambah Purchase kedua; PAID untuk lead lost →
  payment perlu_tinjau, lead tidak berubah.
- Regresi: tombol Verifikasi staf tetap berperilaku sama setelah refactor.
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
- Harga dinamis lewat **produk** yang bisa dibuat admin (bukan hanya Tahsin),
  dengan varian harga.
- Lunas via gateway → lead `terverifikasi` otomatis, konversi ke santri tetap
  manual, **CAPI Purchase dikirim saat uang masuk**.
- Kursi **ditahan** selama tagihan aktif, masa tagihan 2 jam. Kelebihan tetap
  diterima + tanda untuk koordinator.

## 7. Keputusan Claude (bisa dikoreksi)

- Harga: `nominal_override` link > `produk_varian.harga` > `produk.harga`.
- Varian harga sebagai tabel baru `produk_varian`, bukan banyak produk.
- Lead won/DNC yang submit ulang: tidak diubah, ditandai tinjau, diarahkan ke WA.
  Lead lost: dibuka kembali.
- Dedup juga via `leads.phone`. Kasus chat `@lid` diterima sebagai risiko.
- Pembayaran gateway untuk lead yang tidak open → dicatat lunas + perlu tinjau.
- Program pendek tidak menampilkan jadwal.
- Setiap link wajib punya `lead_source_id`.
- Transfer manual + upload bukti tetap tersedia sebagai cadangan (per link).
- Form tidak menimpa field lead yang sudah terisi.
- Kredensial gateway di env, bukan DB.
- Turnstile opsional, mati secara default.

## 8. Diserahkan ke builder

- Struktur komponen Svelte, nama route admin, detail UI mengikuti
  design principles + token `app.css`.
- Mekanisme cache dedup klik.
- Detail mapping status Xendit → `payments.status`.

## 9. Risiko yang diterima

- Lead ganda bila orang yang sama chat dari akun WA `@lid` setelah mengisi form.
- Angka klik indikatif (dedup in-memory hilang saat restart).
- Settlement gateway tidak instan; dana mengendap di saldo Xendit.

## 10. Amandemen fase 1b (2026-11-19)

- **Xendit Payment Session menggantikan Invoice.** Dokumentasi Xendit kini
  menandai Invoice (`/v2/invoices`) sebagai *legacy* dan menyarankan
  Payment Session (`POST /sessions`, `session_type: PAY`,
  `mode: PAYMENT_LINK`). Alur pengguna sama (hosted checkout). Pemetaan:
  `external_id` → `reference_id`, status `PAID` → `COMPLETED`, webhook
  `payment_session.completed` / `payment_session.expired`, header
  `x-callback-token`, sesi memakai `expires_at` (ISO 8601).
- **Konfirmasi ulang ke API.** Payload webhook hanya memuat jumlah sesi,
  bukan jumlah dibayar. Setiap webhook `completed` dikonfirmasi lewat
  `GET /sessions/{id}` (status `COMPLETED`, `amount`, `reference_id` cocok)
  sebelum lead diverifikasi. Webhook palsu tidak bisa menandai lunas walau
  token bocor.
- `payments.external_id` = `reference_id` sesi; `provider_ref` =
  `payment_session_id`; `channel` diisi `payment_id` Xendit (channel QRIS/
  e-wallet tidak ada di payload sesi).
