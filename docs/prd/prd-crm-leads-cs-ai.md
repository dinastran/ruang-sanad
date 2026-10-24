# PRD — CRM Leads (Lead → Closing) terintegrasi CS AI WhatsApp

| | |
|---|---|
| Status | Siap diimplementasikan (bertahap, 4 fase) |
| Repo target | `dinastran/ruang-sanad` (Dashboard). Lampiran B berisi perubahan pendamping di repo GOWA `ruangsanadindonesia-art/go-whatsapp-web-multidevice` (bridge CS AI) |
| Pemilik produk | Ruang Sanad |
| Pembaca utama | Claude Code / developer yang mengimplementasikan |
| Tanggal | Oktober 2026 |

> **Cara memakai dokumen ini (untuk Claude Code):**
> 1. Baca `AGENTS.md` repo ini dulu, lalu ikuti semua aturannya: three-tier Handler → Service → Query, satu file handler per fitur, satu migrasi = satu tabel, query lewat `queries/*.sql` + `sqlc generate`, Svelte 5 rules, CSRF untuk `fetch()`, dan jangan menjalankan dev server.
> 2. Kerjakan **per fase** (bagian 12). Satu fase = satu PR. Jangan mulai fase berikutnya sebelum *acceptance criteria* fase sebelumnya lulus.
> 3. Bagian bertanda **🔒 WAJIB** adalah aturan keamanan atau bisnis yang tidak boleh dilonggarkan.
> 4. Kalau ada yang ambigu, pilih yang paling aman dan tuliskan asumsinya di deskripsi PR. Jangan menebak data bisnis (harga, rekening, jadwal).

---

## 1. Latar belakang

Ruang Sanad sudah punya sistem-sistem berikut:
- **GOWA** (`wa.ruangsanad.com`): gateway WhatsApp multi-nomor (device): `cs-ai`, dan nanti `keuangan`, `admin-kelas`, `koord-guru`.
- **Bridge CS AI** (`hermes-cs-bridge`, satu VPS dengan GOWA): menerima pesan nomor CS dan meminta balasan ke Hermes Agent (profile `ruang-sanad`). Alur closing-nya:
  - discovery → rekomendasi paket
  - → `[[FORM]]` (form data resmi)
  - → rangkuman data → `[[REKENING:1x|2x]]` (info rekening + nominal)
  - → bukti transfer diteruskan ke admin (handoff).
- **Dashboard** (repo ini): data santri, kelas, tagihan, role (`cs`, `keuangan`, `admin_kelas`, `super_admin`, …), notifikasi, dan MCP baca-saja untuk AI agent.

Masalahnya, lead (calon santri) hanya hidup di chat WhatsApp dan di notifikasi WA ke admin. Akibatnya:
- tidak ada pipeline dan tidak ada pencatatan tahap;
- data form diketik ulang manual saat membuat santri;
- tidak ada laporan konversi;
- follow-up untuk yang belum closing bergantung ingatan CS;
- AI tidak bisa menganalisis lead secara terstruktur.

## 2. Tujuan dan non-tujuan

**Tujuan**
1. Setiap orang yang chat ke nomor CS otomatis menjadi **lead** di Dashboard, lengkap dengan transkrip percakapan.
2. **Pipeline lead → closing** dengan tahap yang terisi otomatis dari kejadian pasti di bridge (form dikirim, form diisi, info bayar dikirim, bukti transfer masuk), plus tindakan manual.
3. **Verifikasi pembayaran** dan **"Jadikan Santri"** sekali klik. Data form terisi otomatis ke formulir santri, lalu Admin Kelas diberi notifikasi.
4. **Analisis AI** per lead (persona, kebutuhan, keberatan, skor minat, saran tindak lanjut) dan **laporan funnel**, lewat MCP Dashboard.
5. **Follow-up terjadwal** (Hari 1/3/7/14/30 sesuai Script 30 Hari): disusun AI, **disetujui admin**, dikirim lewat nomor CS dengan aturan berhenti yang ketat.

**Non-tujuan (fase ini)**
- Broadcast massal dan kampanye marketing.
- CRM untuk santri yang sudah aktif. Setelah menjadi santri, datanya dikelola modul santri yang sudah ada.
- Lead dari nomor selain CS, atau dari kanal non-WhatsApp. Kolom `source` disiapkan untuk nanti.
- Verifikasi pembayaran otomatis (cek mutasi bank).
- Mengirim tagihan bulanan lewat WA. Fitur ini memakai pondasi `wa_outbox` yang sama, tapi di luar cakupan PRD ini.

## 3. Keputusan yang sudah diambil

| # | Keputusan |
|---|---|
| D1 | CRM berfokus pada **lead → closing**. Setelah closing, lead menjadi santri di modul yang sudah ada |
| D2 | Verifikasi pembayaran dan "Jadikan Santri" boleh dilakukan **CS dan Keuangan** (plus `super_admin`) |
| D3 | Setelah santri dibuat, **Admin Kelas mendapat notifikasi Dashboard** (tabel `notifications`), dan santri masuk alur "Perlu Dilengkapi" yang sudah ada |
| D4 | Follow-up: **AI menyusun draft, admin menyetujui**, lalu terkirim sesuai jadwal |
| D5 | Dashboard → GOWA memakai akun Basic Auth `dashboard` yang **dikunci ke device-nya** lewat `APP_BASIC_AUTH_DEVICES` di GOWA (fitur ini sudah ada di GOWA) |
| D6 | **Semua pesan keluar dari nomor CS lewat bridge**, termasuk follow-up (lihat 4.3). Dashboard tidak mengirim langsung dari device `cs-ai` |
| D7 | Agen AI (Hermes/Claude) menganalisis lewat **MCP Dashboard**, bukan MCP GOWA. MCP GOWA punya tool kirim dan hapus pesan di semua nomor |

## 4. Arsitektur

### 4.1 Gambaran

```
Calon santri ⇄ WhatsApp ⇄ GOWA (device cs-ai)
                              │ webhook
                              ▼
                   hermes-cs-bridge ── Hermes (balasan CS AI)
                     │  ▲
     (A) event lead  │  │ (D) kirim follow-up yang disetujui
                     ▼  │
           Dashboard Ruang Sanad ─────────────── (E) REST, device keuangan/admin-kelas ──► GOWA
             • leads, lead_messages, lead_events
             • analisis, follow-up, wa_outbox
             • UI CRM, verifikasi, Jadikan Santri
                     ▲
                     │ (C) MCP Dashboard (baca + tulis terbatas)
              Hermes (job harian) / Claude (laporan)
```

### 4.2 Topologi server (fakta saat ini)

- GOWA berjalan di Docker dan listen di `127.0.0.1:3000`. Bridge memakai `network_mode: host` dan listen di `172.17.0.1:8650`. Hermes API ada di `127.0.0.1:8642`. Ketiganya di VPS yang sama dengan Dashboard.
- Dashboard berjalan sebagai service systemd (binary Go) di host, sehingga bisa menjangkau GOWA di `http://127.0.0.1:3000`.
- Bridge berjalan dengan host network, sehingga bisa menjangkau Dashboard di `http://127.0.0.1:<APP_PORT>`. Dashboard menjangkau bridge di `http://172.17.0.1:8650`.

### 4.3 Kenapa follow-up dikirim lewat bridge (D6)

Pesan yang dikirim lewat API GOWA **tidak memicu webhook** ke bridge. Kalau Dashboard mengirim follow-up langsung dari `cs-ai`, akan terjadi tiga hal:
1. Bridge dan Hermes tidak tahu follow-up itu ada. Saat calon santri membalas, CS AI kehilangan konteks.
2. Akun Dashboard perlu akses ke `cs-ai`, sehingga isolasi nomor melemah.
3. Ada dua pengirim di nomor yang sama. Jeda dan anti-blokir sulit dikendalikan.

Karena itu, bridge menyediakan endpoint `POST /api/outbound` (Lampiran B). Bridge mengirim pesan, mencatatnya di konteks chat, lalu mengembalikan event `message.out` ke Dashboard.

## 5. Model data

Konvensi: SQLite, satu migrasi = satu tabel. Waktu disimpan dalam `DATETIME` UTC dan ditampilkan dalam WIB (`Asia/Jakarta`). Nomor HP dinormalisasi menjadi digit internasional (`62…`).

### 5.1 `leads`

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | INTEGER PK | |
| wa_chat_id | TEXT NOT NULL UNIQUE | JID chat dari bridge (`628…@s.whatsapp.net` atau `…@lid`) |
| phone | TEXT NOT NULL DEFAULT '' | Digit `62…` bila diketahui. 🔒 Data kontak |
| wa_name | TEXT NOT NULL DEFAULT '' | Nama tampilan WA |
| nama | TEXT NOT NULL DEFAULT '' | Dari form |
| usia | INTEGER NULL | Dari form |
| domisili | TEXT NOT NULL DEFAULT '' | Dari form. 🔒 Data kontak/lokasi |
| jenis_kelamin | TEXT NOT NULL DEFAULT '' | `L` / `P` / `''` (Ikhwan→L, Akhwat→P) |
| paket_kode | TEXT NOT NULL DEFAULT '' | Kode paket bridge (`1x`, `2x`) |
| kelas_kode | TEXT NOT NULL DEFAULT '' | Kode kelas Dashboard (`R 1X`, `R 2X`), hasil pemetaan dari paket (5.9) |
| waktu_belajar | TEXT NOT NULL DEFAULT '' | `pagi` / `siang` / `malam` / teks bebas |
| riwayat_belajar | TEXT NOT NULL DEFAULT '' | "Pernah belajar tahsin?" |
| belajar_dimana | TEXT NOT NULL DEFAULT '' | |
| untuk_siapa | TEXT NOT NULL DEFAULT '' | `diri_sendiri` / `anak` / `keluarga` / `''` |
| stage | TEXT NOT NULL DEFAULT 'baru' | Lihat 6.1 |
| status | TEXT NOT NULL DEFAULT 'open' | `open` / `won` / `lost` / `do_not_contact` |
| lost_reason | TEXT NOT NULL DEFAULT '' | `harga`, `jadwal`, `belum_prioritas`, `keluarga`, `tidak_cocok`, `tidak_merespons`, `lainnya` |
| followup_permission | TEXT NOT NULL DEFAULT 'unknown' | `unknown` / `yes` / `no` (Script: izin follow-up) |
| source | TEXT NOT NULL DEFAULT 'wa_cs' | Untuk kanal lain nanti |
| assigned_user_id | INTEGER NULL FK users ON DELETE SET NULL | CS penanggung jawab |
| payment_amount | INTEGER NOT NULL DEFAULT 0 | Rupiah, dari event `payment_info.sent` |
| payment_info_sent_at | DATETIME NULL | |
| proof_received_at | DATETIME NULL | |
| verified_at / verified_by | DATETIME NULL / INTEGER NULL FK users | |
| converted_at / converted_by | DATETIME NULL / INTEGER NULL FK users | |
| santri_id | INTEGER NULL UNIQUE FK santri ON DELETE SET NULL | 🔒 UNIQUE untuk mencegah satu lead menjadi dua santri |
| needs_review | INTEGER NOT NULL DEFAULT 0 | 1 bila parser form ragu (7.2) |
| last_inbound_at / last_outbound_at | DATETIME NULL | Untuk daftar dan follow-up |
| last_analyzed_message_id | INTEGER NOT NULL DEFAULT 0 | ID `lead_messages` terakhir yang sudah dianalisis |
| created_at / updated_at | DATETIME | |

Index: `(status, stage, updated_at DESC)`, `(assigned_user_id)`, `(last_inbound_at)`, `(phone)`.

### 5.2 `lead_messages` (transkrip chat CS)

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | INTEGER PK | |
| lead_id | INTEGER NOT NULL FK leads ON DELETE CASCADE | |
| wa_message_id | TEXT NOT NULL DEFAULT '' | ID pesan WA. Bisa kosong untuk pesan sistem |
| direction | TEXT NOT NULL | `in` / `out` |
| sender | TEXT NOT NULL | `lead` / `ai` / `admin` (admin membalas dari HP) / `followup` |
| body | TEXT NOT NULL DEFAULT '' | Teks persis seperti yang dikirim/diterima |
| media_type | TEXT NOT NULL DEFAULT '' | `gambar`, `dokumen`, `audio`, … (file media tidak disimpan) |
| sent_at | DATETIME NOT NULL | Waktu pesan dari WA/bridge |
| created_at | DATETIME | |

Unique `(lead_id, wa_message_id)` bila `wa_message_id <> ''` (partial unique index). Index `(lead_id, sent_at)`.

### 5.3 `lead_events` (timeline dan audit)

| Kolom | Keterangan |
|---|---|
| id, lead_id FK CASCADE | |
| event_id TEXT NOT NULL UNIQUE | Idempotensi. Dari bridge (`evt_…`), atau UUID untuk event manual |
| type TEXT | Lihat 7.1 dan event manual: `stage.changed`, `payment.verified`, `lead.converted`, `lead.lost`, `lead.assigned`, `label.added`, `followup.approved`, … |
| actor TEXT | `bridge` / `user:<id>` / `ai:<token_id>` / `system` |
| payload TEXT (JSON) | Isi ringkas kejadian. 🔒 Jangan simpan secret |
| created_at | |

### 5.4 `lead_analyses` (hasil analisis AI, riwayat)

| Kolom | Keterangan |
|---|---|
| id, lead_id FK CASCADE | |
| ai_token_id INTEGER NULL FK ai_tokens | Siapa yang menulis |
| summary TEXT | Ringkasan ≤ 600 karakter |
| persona TEXT (JSON) | Skema tetap, lihat 9.3 |
| interest_score INTEGER | 0–100 |
| main_objection TEXT | Salah satu kategori keberatan Script |
| suggested_stage TEXT | **Saran** tahap. Tidak mengubah `leads.stage` (🔒 9.4) |
| suggested_next_action TEXT | ≤ 300 karakter |
| confidence TEXT | `rendah` / `sedang` / `tinggi` |
| analyzed_until_message_id INTEGER | |
| created_at | |

UI menampilkan analisis terbaru. Riwayat lama dipertahankan untuk evaluasi kualitas.

### 5.5 `lead_labels` dan 5.6 `lead_label_links`

- `lead_labels`: `id`, `nama UNIQUE`, `warna` (token warna yang sudah ada), `created_at`.
- `lead_label_links`: `lead_id`, `label_id`, `created_by`, `created_at`, dengan PK `(lead_id, label_id)`.

Contoh label awal (seed opsional): `prioritas`, `butuh telepon`, `wali/anak`, `minta jadwal khusus`.

### 5.7 `lead_followups`

| Kolom | Keterangan |
|---|---|
| id, lead_id FK CASCADE | |
| step | `d1` / `d3` / `d7` / `d14` / `d30` / `manual` |
| scheduled_at DATETIME | Waktu kirim yang direncanakan (WIB, dalam jam kirim) |
| draft_text TEXT | ≤ 1000 karakter |
| reason TEXT | Alasan relevansi. 🔒 Wajib untuk `d7` dan `d30` ("update nyata") |
| status | `pending_approval` / `approved` / `sent` / `cancelled` / `skipped` / `failed` |
| created_by_kind | `ai` / `user` |
| created_by | user id atau ai_token id |
| approved_by / approved_at | |
| sent_at / wa_message_id / error | |
| created_at / updated_at | |

Index `(status, scheduled_at)` dan `(lead_id, step)`. 🔒 Satu `step` aktif per lead: partial unique `(lead_id, step)` WHERE `status IN ('pending_approval','approved')`.

### 5.8 `wa_outbox` (antrean kirim WA, dipakai bersama)

| Kolom | Keterangan |
|---|---|
| id | |
| purpose | `followup` (nanti: `tagihan`, `reminder_kelas`) |
| channel | `bridge_cs` (follow-up, lewat bridge) / `gowa:<device>` (nanti, untuk tagihan) |
| to_chat_id | JID tujuan |
| body | |
| reference_type / reference_id | Mis. `lead_followup` / id |
| not_before | DATETIME. Jangan kirim sebelum waktu ini |
| status | `queued` / `sending` / `sent` / `failed` / `cancelled` |
| attempts, last_error, sent_at, wa_message_id | |
| created_at, updated_at | |

Index `(status, not_before)`.

### 5.9 Pengaturan

**Secret di `.env` Dashboard**, tidak pernah ditampilkan di UI:

| Variabel | Fungsi |
|---|---|
| `CS_BRIDGE_INGEST_TOKEN` | Token yang dipakai bridge saat mengirim event ke Dashboard (min. 32 karakter acak) |
| `CS_BRIDGE_URL` | Mis. `http://172.17.0.1:8650` |
| `CS_BRIDGE_OUTBOUND_TOKEN` | Token yang dipakai Dashboard saat memanggil `POST /api/outbound` di bridge |
| `GOWA_URL`, `GOWA_USER`, `GOWA_PASS` | Akun `dashboard` di GOWA, untuk tes koneksi dan fitur tagihan nanti |

**Non-secret di tabel `crm_settings`** (satu baris, migrasi sendiri), bisa diubah `super_admin`:

| Kunci | Default | Keterangan |
|---|---|---|
| paket_ke_kelas_kode | `{"1x":"R 1X","2x":"R 2X"}` | Pemetaan paket bridge → `kode_kelas` |
| followup_enabled | `false` | Saklar global follow-up |
| followup_send_window | `08:00–20:00` WIB | Di luar jam ini follow-up ditunda |
| followup_min_gap_seconds | `45` | Jeda minimum antar kirim (global) |
| followup_daily_cap | `40` | Maks follow-up terkirim per hari (global) |
| transcript_retention_days_lost | `180` | Transkrip lead `lost`/`do_not_contact` dihapus setelah N hari |
| gowa_devices | `{"keuangan":"keuangan","admin_kelas":"admin-kelas","koordinator":"koord-guru"}` | Untuk fitur kirim per peran nanti |

## 6. Pipeline dan aturan bisnis

### 6.1 Tahap (`stage`) untuk lead `status=open`

| Stage | Label UI | Masuk lewat |
|---|---|---|
| `baru` | Baru | Event pertama dari nomor baru |
| `discovery` | Discovery | Manual (CS), atau saran AI yang diterapkan CS |
| `rekomendasi` | Rekomendasi | Manual / saran AI yang diterapkan |
| `form_dikirim` | Form dikirim | Event `form.sent` |
| `data_terisi` | Data terisi | Event `message.in` yang lolos parser form (7.2) |
| `info_bayar_dikirim` | Info bayar dikirim | Event `payment_info.sent` |
| `bukti_masuk` | Bukti transfer masuk | Event `proof.received` |
| `terverifikasi` | Terverifikasi | Tombol **Verifikasi pembayaran** (CS/Keuangan) |

Status akhir:
- `won`: setelah **Jadikan Santri** berhasil.
- `lost`: manual, dengan `lost_reason` wajib.
- `do_not_contact`: manual, atau otomatis ketika lead berkata "stop/jangan hubungi" (7.3).

### 6.2 🔒 Aturan transisi

1. Event otomatis **hanya boleh memajukan** stage, tidak pernah memundurkan. Event `form.sent` untuk lead yang sudah `info_bayar_dikirim` tetap dicatat di timeline, tapi stage tidak berubah.
2. **Manual boleh maju atau mundur**, kecuali ke `terverifikasi`. Tahap itu hanya bisa dicapai lewat aksi Verifikasi, yang mencatat siapa dan kapan.
3. `terverifikasi` → `won` hanya lewat **Jadikan Santri**.
4. Lead `won` bersifat **read-only** untuk kolom data. Lead `lost` dan `do_not_contact` bisa dibuka kembali secara manual dengan event `lead.reopened`.
5. Pesan masuk baru dari lead `lost` membuka kembali lead itu (`status=open`, stage tetap, event `lead.reopened` actor `bridge`). Pesan masuk dari lead `do_not_contact` **tidak** membuka kembali lead itu, tapi tetap ditandai di UI ("menghubungi lagi").
6. Setiap perubahan stage atau status menulis `lead_events` dengan `from` dan `to`.

### 6.3 Verifikasi pembayaran

- Role: `cs`, `keuangan`, `super_admin`.
- Syarat: stage `info_bayar_dikirim` atau `bukti_masuk`. Bila masih `info_bayar_dikirim`, minta konfirmasi tambahan "bukti belum tercatat dari WA, tetap verifikasi?".
- Input: nominal diterima (default `payment_amount`), tanggal terima (default hari ini), catatan opsional.
- 🔒 Kalau nominal berbeda dari `payment_amount`, tampilkan peringatan dan wajibkan catatan.
- Efek:
  - stage menjadi `terverifikasi`, `verified_at`/`verified_by` terisi, event `payment.verified`;
  - **opsional** (checkbox, default tidak dicentang): kirim pesan konfirmasi ke calon santri lewat `wa_outbox` → bridge. Teksnya dari template `crm_settings.payment_verified_template`.

### 6.4 Jadikan Santri

- Role: `cs`, `keuangan`, `super_admin`. Syarat: stage `terverifikasi` dan `santri_id IS NULL`.
- Dialog berisi data yang sudah terisi dan bisa diedit:

| Field santri | Sumber |
|---|---|
| nama | `leads.nama` (wajib) |
| jenis_kelamin | `leads.jenis_kelamin` (wajib L/P) |
| usia, domisili | `leads.usia`, `leads.domisili` |
| no_wa | `leads.phone` |
| kelas_kode | `leads.kelas_kode` (dari pemetaan paket). Pilihan dari `kode_kelas` |
| nominal | nominal yang diverifikasi |
| tanggal_daftar | tanggal verifikasi (YYYY-MM-DD) |
| angkatan | **wajib dipilih** dari angkatan `is_aktif=1`. Default: angkatan aktif terbaru |

- Implementasi:
  - **Pakai validasi dan aturan `SantriService.Create` yang sudah ada.** Tambahkan varian yang menerima `*queries.Querier` transaksi (mis. `CreateWithQuerier`) supaya pembuatan santri dan update lead terjadi dalam **satu transaksi**: buat santri, isi `leads.santri_id`, `status=won`, `converted_at`/`converted_by`, lalu tulis event `lead.converted`.
  - 🔒 Kalau terjadi race (dua klik, dua user), constraint UNIQUE `santri_id` dan pengecekan `santri_id IS NULL` di dalam transaksi menjamin hanya satu santri yang dibuat.
- Setelah commit:
  - buat `notifications` untuk **setiap user role `admin_kelas`**: type `santri_baru_dari_lead`, title "Santri baru perlu penempatan kelas", action_url `/app/perlu-dilengkapi`, reference `santri`/id;
  - redirect ke `/app/santri/:id`.
- Duplikat: sebelum membuat, cari santri dengan `no_wa` sama. Kalau ada, tampilkan peringatan "nomor ini sudah terdaftar sebagai santri X". Pengguna memilih **tautkan ke santri itu** (isi `santri_id` tanpa membuat baru) atau **tetap buat baru**.

### 6.5 Penugasan

- Lead baru tidak ditugaskan ke siapa pun secara default. CS bisa klik "Ambil" (assign ke diri sendiri). Super admin bisa assign ke CS mana pun.
- Filter "Lead saya" di daftar.

## 7. Integrasi bridge → Dashboard (Fase A)

### 7.1 Endpoint ingest

`POST /integrations/cs-ai/events`. Endpoint ini di luar grup `/app`, jadi tidak memakai session/CSRF. Handler-nya file sendiri: `app/handlers/cs_ai_ingest.go`.

🔒 Keamanan:
- Header `Authorization: Bearer <CS_BRIDGE_INGEST_TOKEN>`, dibandingkan *constant-time*. Kalau token di `.env` kosong, endpoint mengembalikan 404 (fitur mati).
- Batasi IP sumber ke loopback/docker (`127.0.0.1`, `::1`, `172.16.0.0/12`). Daftar bisa dikonfigurasi lewat `CS_BRIDGE_ALLOWED_CIDRS`. Pertahanan berlapis, karena Dashboard publik di belakang Nginx. Perhatikan `X-Forwarded-For`: pakai IP koneksi langsung, kecuali proxy tepercaya sudah diset.
- Body maksimal 256 KB. Rate limit 600 request/menit.
- Idempotensi: `event_id` sudah ada → balas `200 {"status":"duplicate"}` tanpa efek.

Format request (batch, maksimal 100 event):

```json
{
  "events": [
    {
      "event_id": "evt_01JB8K9Z3F2N…",
      "type": "message.in",
      "occurred_at": "2026-10-24T03:12:45Z",
      "chat_id": "6281234567890@s.whatsapp.net",
      "phone": "6281234567890",
      "wa_name": "Fulanah",
      "data": { "wa_message_id": "3EB0C1…", "body": "Assalamu'alaikum, mau tanya kelas tahsin", "media_type": "" }
    }
  ]
}
```

| `type` | `data` | Efek di Dashboard |
|---|---|---|
| `message.in` | `wa_message_id`, `body`, `media_type` | Buat lead bila belum ada (`stage=baru`). Simpan `lead_messages` (`in`, `lead`). Update `last_inbound_at`. Jalankan parser form bila stage `form_dikirim` (7.2). Jalankan deteksi stop (7.3). Batalkan follow-up `pending_approval`/`approved` milik lead itu (`status=cancelled`, alasan "lead membalas") |
| `message.out` | `wa_message_id`, `body`, `sender` (`ai`/`admin`/`followup`) | Simpan `lead_messages` (`out`). Update `last_outbound_at`. `sender=followup` → tandai `lead_followups.status=sent` lewat `reference_id` |
| `form.sent` | `{}` | Stage maju ke `form_dikirim` |
| `payment_info.sent` | `paket_kode`, `nominal_text` (mis. `"Rp149.000"`) | Simpan `paket_kode`, petakan ke `kelas_kode`, parse nominal ke integer (`payment_amount`). Stage maju ke `info_bayar_dikirim`. Isi `payment_info_sent_at` |
| `proof.received` | `media_type` | Stage maju ke `bukti_masuk`. Isi `proof_received_at`. Notifikasi Dashboard ke user role `keuangan` dan CS yang di-assign: "Bukti transfer masuk dari <nama/wa_name>" |
| `handoff` | `reason`, `summary` | Event timeline. Bila `reason` mengandung "bukti transfer", perlakukan seperti `proof.received` (kalau belum) |
| `ai.paused` / `ai.resumed` | `until`, `reason` | Timeline saja (badge "AI dijeda" di detail lead) |

Respons: `200 {"accepted": n, "duplicates": m}`. Kalau satu event rusak, tolak event itu saja (`"rejected":[{"event_id":…,"error":…}]`) dan proses sisanya. Bridge **tidak** mengirim ulang event yang ditolak karena skema.

### 7.2 Parser form (deterministik)

Bridge mengirim form dengan label tetap (lihat pengaturan "Form data pendaftaran" di `/cs-ai`). Parser berjalan pada pesan `message.in` saat stage `form_dikirim`, atau `data_terisi` bila lead mengirim revisi.

- Pisahkan per baris. Tiap baris dicocokkan dengan `^[\s*•\-\d.)]*([^:]+):\s*(.*)$`.
- Label dinormalisasi: lowercase, tanpa tanda baca, spasi tunggal. Peta sinonim:

| Field | Label yang diterima |
|---|---|
| nama | `nama`, `nama lengkap` |
| usia | `usia`, `umur` → angka pertama 5–99 |
| domisili | `domisili`, `alamat`, `kota` |
| jenis_kelamin | `ikhwan/akhwat`, `ikhwan akhwat`, `jenis kelamin`, `gender` → `ikhwan`/`laki`/`pria`/`L` = **L**, `akhwat`/`perempuan`/`wanita`/`P` = **P** |
| paket_kode | `pilihan kelas`, `paket`, `kelas` → mengandung `1` = `1x`, `2` = `2x` |
| waktu_belajar | `waktu belajar`, `waktu` |
| riwayat_belajar | `apakah sebelumnya pernah belajar…`, `pernah belajar` |
| belajar_dimana | `jika iya/pernah belajar dimana`, `belajar dimana` |

- Pesan dianggap **form terisi** bila ≥ 3 field dikenali dan `nama` tidak kosong. Stage maju ke `data_terisi`, kolom lead diisi (yang kosong tidak menimpa yang sudah ada), dan event `form.parsed` dicatat beserta daftar field.
- 🔒 Field yang gagal dinormalisasi (mis. jenis kelamin "-") **tidak ditebak**. Biarkan kosong dan set `needs_review=1`. UI menampilkan badge "Perlu dicek".
- Unit test wajib memakai contoh nyata: form persis template, form dengan baris dihapus, semua isian dalam satu baris (gagal → `needs_review`), angka usia dengan teks ("25 th"), dan variasi huruf besar-kecil.

### 7.3 Deteksi permintaan berhenti

Pada `message.in`, kalau body (lowercase) **cocok persis atau diawali** salah satu frasa (`stop`, `berhenti`, `jangan hubungi`, `jangan chat lagi`, `unsubscribe`, `tidak usah dihubungi`, `jangan kirim pesan lagi`):
- set `followup_permission=no`;
- batalkan semua follow-up aktif;
- tambahkan event `optout.detected`;
- **jangan** otomatis set `do_not_contact`. CS yang mengonfirmasi lewat tombol, karena frasa seperti "stop dulu ya, nanti saya kabari" ambigu.

Frasa "belum dulu"/"nanti saja" **tidak** di sini. Itu ditangani AI (penutupan sopan) dan analisis (`suggested_stage`/`lost_reason`).

## 8. UI CRM (Fase B)

Svelte 5 + Inertia, mengikuti design principles dan token warna di `frontend/src/app.css`. **Mobile-first**, karena CS banyak bekerja dari HP. Menu baru **"Leads"** terlihat untuk role `cs`, `keuangan`, `super_admin`.

### 8.1 `/app/leads`: daftar dan pipeline
- Dua tampilan dengan toggle:
  - **Daftar**: default di HP. Kartu berisi nama/wa_name, stage badge, waktu chat terakhir, paket, label, ikon "AI dijeda", dan "Perlu dicek".
  - **Kanban** per stage: desktop. Pindah kartu = ubah stage manual, mengikuti aturan 6.2.
- Filter: stage, status (default `open`), label, "Lead saya", "Perlu dicek", "Bukti masuk", rentang tanggal. Pencarian nama/wa_name/nomor.
- Urutan default: `bukti_masuk` di atas, lalu `last_inbound_at` terbaru.
- Paginasi berbasis cursor, 30 per halaman.

### 8.2 `/app/leads/:id`: detail lead
- **Header**: nama, nomor (tombol salin + link `wa.me`), stage, status, assigned, label, dan tombol aksi sesuai stage: Ambil, Ubah tahap, **Verifikasi pembayaran**, **Jadikan Santri**, Tandai hilang, Jangan hubungi, Buka kembali.
- **Data calon santri**: field dari form, bisa diedit (event `lead.updated`), dengan badge "Perlu dicek" per field.
- **Percakapan**: transkrip `lead_messages` gaya chat. Gelembung kiri untuk lead, kanan untuk AI/admin/follow-up, diberi label pengirim. Muat 50 pesan terakhir, scroll ke atas memuat lebih banyak. Media hanya sebagai label ("📷 gambar"). 🔒 Teks ditampilkan sebagai teks, tidak pernah sebagai HTML.
- **Analisis AI**: kartu analisis terbaru berisi ringkasan, persona, skor minat, keberatan, saran tahap dan tindakan, keyakinan, serta "dianalisis X jam lalu". Tombol **Terapkan saran tahap**. 🔒 Label tetap "Dugaan AI — periksa sebelum dipakai".
- **Follow-up**: daftar follow-up lead ini dengan statusnya, plus tombol "Buat follow-up manual".
- **Timeline**: `lead_events` ringkas.

### 8.3 `/app/leads/followups`: antrean persetujuan
- Daftar `pending_approval`, urut `scheduled_at`. Tiap item menampilkan lead, step, jadwal, alasan, draft (bisa diedit), dan potongan 3 pesan terakhir.
- Aksi: **Setujui**, **Edit & setujui**, **Ubah jadwal**, **Batalkan**. Aksi massal "Setujui terpilih" hanya untuk item tanpa peringatan.
- Peringatan merah otomatis:
  - draft memuat angka rupiah atau nomor rekening;
  - draft memuat link;
  - step `d7`/`d30` tanpa `reason`;
  - lead `followup_permission=no`, yang **diblok**, bukan hanya diberi peringatan.

### 8.4 `/app/leads/laporan`
- Funnel per stage (jumlah dan konversi antar tahap) untuk rentang tanggal.
- Waktu median dari `baru` ke `won`.
- Distribusi `lost_reason` dan `main_objection` dari analisis.
- Lead per minggu dan konversi per minggu.
- Kinerja follow-up: terkirim, dibalas dalam 72 jam, berlanjut ke `won`.
- Ekspor CSV, tanpa nomor HP kecuali untuk `super_admin`.

### 8.5 `/app/integrasi-whatsapp` (super_admin)
- Status koneksi: GOWA (`GET /devices/:id/status` per device yang dipetakan, memakai akun `dashboard`) dan bridge (`GET /healthz`).
- Pengaturan `crm_settings`: pemetaan paket → kode kelas, saklar dan aturan follow-up, retensi.
- Info: event terakhir dari bridge dan jumlah event hari ini, untuk memantau kalau ingest berhenti.

### 8.6 Matriks akses

| Aksi | cs | keuangan | super_admin | admin_kelas | lainnya |
|---|---|---|---|---|---|
| Lihat daftar/detail lead | ✅ | ✅ | ✅ | ❌ | ❌ |
| Ubah data/stage/label/assign | ✅ | ✅ | ✅ | ❌ | ❌ |
| Verifikasi pembayaran | ✅ | ✅ | ✅ | ❌ | ❌ |
| Jadikan Santri | ✅ | ✅ | ✅ | ❌ | ❌ |
| Setujui follow-up | ✅ | ❌ | ✅ | ❌ | ❌ |
| Laporan | ✅ | ✅ | ✅ | ❌ | ❌ |
| Integrasi WhatsApp | ❌ | ❌ | ✅ | ❌ | ❌ |
| Notifikasi santri baru | — | — | — | ✅ | — |

🔒 Guard role ada di route (`middlewares.RoleRequired`), bukan hanya disembunyikan di UI. Pola ini sama dengan route lain di `routes/web.go`.

## 9. Analisis AI lewat MCP (Fase C)

### 9.1 Prinsip
- Pakai **framework MCP yang sudah ada** (`app/handlers/mcp.go`, `app/services/ai_agent*.go`, audit `ai_audit_log`, token `ai_tokens`).
- 🔒 **Data kontak tetap tidak pernah keluar lewat MCP**: `phone`, `domisili`, dan `wa_chat_id` tidak ada di DTO MCP. Lead diidentifikasi dengan `lead_ref` = `L-<id>`. Teks transkrip **wajib** melewati `ai_redact.go` sebelum dikembalikan, untuk menyensor nomor, email, dan link WA yang diketik di chat.
- 🔒 Tool **tulis** hanya untuk token yang punya izin tulis. Tambahkan kolom `can_write INTEGER NOT NULL DEFAULT 0` di `ai_tokens` lewat migrasi baru. Tidak bisa diaktifkan dari halaman Akses AI biasa; hanya `super_admin` yang bisa membuat token dengan centang "boleh menulis analisis CRM". Role pemilik token tetap harus `cs`/`keuangan`/`super_admin`.

### 9.2 Tool baru

| Tool | Akses | Input | Output |
|---|---|---|---|
| `daftar_lead` | baca | `status`, `stage[]`, `updated_since`, `butuh_analisis` (bool: ada pesan baru setelah `last_analyzed_message_id`), `limit` ≤ 50, `cursor` | `lead_ref`, nama depan (tanpa nama lengkap), stage, status, paket, hari sejak chat terakhir, jumlah pesan baru |
| `detail_lead` | baca | `lead_ref`, `pesan_sejak_id` opsional, `max_pesan` ≤ 200 | Field non-kontak (usia, jenis_kelamin, paket, waktu_belajar, riwayat), stage, label, analisis terakhir, transkrip ter-redaksi (`id`, `arah`, `pengirim`, `waktu`, `teks`) |
| `laporan_funnel` | baca | `dari`, `sampai` | Angka funnel seperti 8.4, tanpa data pribadi |
| `simpan_analisis_lead` | **tulis** | `lead_ref`, `summary`, `persona` (9.3), `interest_score`, `main_objection`, `suggested_stage`, `suggested_next_action`, `confidence`, `analyzed_until_message_id` | Insert `lead_analyses`, update `leads.last_analyzed_message_id`, event `analysis.saved` |
| `usulkan_followup` | **tulis** | `lead_ref`, `step`, `scheduled_at`, `draft_text`, `reason` | Insert `lead_followups` status `pending_approval` **setelah lolos aturan 10.2**. Kalau ditolak, kembalikan alasannya ke AI (mis. "lead minta berhenti") |
| `tambah_label_lead` | **tulis** | `lead_ref`, `label` (hanya label yang sudah ada) | Link label |

🔒 **Tidak ada** tool untuk: mengubah `leads.stage`/`status`, verifikasi, konversi, mengirim pesan, menyetujui follow-up, atau membaca nomor. Semua itu wewenang manusia.

### 9.3 Skema `persona` (validasi ketat, field di luar skema ditolak)

```json
{
  "untuk_siapa": "diri_sendiri | anak | keluarga | tidak_diketahui",
  "level": "pemula | pernah_belajar | sedang_belajar | tidak_diketahui",
  "tujuan": ["makhraj", "tajwid", "percaya_diri", "rutinitas", "lainnya"],
  "preferensi_waktu": "pagi | siang | malam | fleksibel | tidak_diketahui",
  "paket_diminati": "1x | 2x | belum_jelas",
  "hambatan": ["harga", "jadwal", "konsistensi", "keluarga", "ragu_online", "percaya_program", "belum_prioritas"],
  "gaya_komunikasi": "singkat | detail | formal | santai",
  "catatan": "≤ 300 karakter"
}
```

🔒 Persona **tidak boleh** memuat penilaian tingkat keagamaan, kondisi ekonomi, kesehatan, etnis, atau politik. Validator menolak `catatan` yang memuat kata kunci kategori sensitif dari daftar yang bisa dikonfigurasi. Instruksi agen (Lampiran C) juga melarangnya.

### 9.4 🔒 AI memberi saran, manusia memutuskan
`suggested_stage` hanya disimpan di analisis. UI menampilkan tombol **Terapkan**, dan penerapannya dicatat sebagai event manual oleh user yang mengklik.

### 9.5 Job harian Hermes
- Jadwal lewat cron Hermes di profile `ruang-sanad`, misalnya setiap hari pukul 21:00 WIB. Hermes memakai token MCP Dashboard yang punya `can_write`.
- Alur:
  1. `daftar_lead(butuh_analisis=true, status=open)`
  2. untuk tiap lead: `detail_lead(pesan_sejak_id=…)`, lalu `simpan_analisis_lead`
  3. bila layak, `usulkan_followup`
  4. ringkasan ke admin (mis. via Telegram Hermes): jumlah lead dianalisis, follow-up yang menunggu persetujuan, dan lead panas
- Prompt lengkap ada di Lampiran C.
- 🔒 Biaya: hanya lead dengan pesan baru yang dianalisis, dengan batas `limit` per run (default 50). Lead `won`/`lost`/`do_not_contact` tidak dianalisis.

## 10. Follow-up terjadwal (Fase D)

### 10.1 Alur
```
AI (job harian) atau CS → lead_followups(pending_approval)
→ CS menyetujui di /app/leads/followups → approved
→ worker Dashboard (tiap 1 menit) mengambil follow-up approved yang scheduled_at ≤ sekarang
  & berada di jam kirim & lolos cek ulang 10.2 → wa_outbox(queued, channel=bridge_cs)
→ worker outbox (jeda ≥ followup_min_gap_seconds, cap harian) → POST bridge /api/outbound
→ bridge mengirim lewat GOWA, mengembalikan wa_message_id, mengirim event message.out(sender=followup)
→ lead_followups.status = sent
```
Worker mengikuti pola `time.NewTicker` yang sudah ada di `cmd/laju-go/main.go`. Ia berjalan di satu proses, dengan transaksi `UPDATE … WHERE status='approved'` sebagai *claim* supaya tidak ada kiriman ganda.

### 10.2 🔒 Aturan follow-up (Script 30 Hari), dicek saat diusulkan DAN tepat sebelum dikirim

| Aturan | Implementasi |
|---|---|
| Hanya lead `status=open` | Lainnya → `skipped` |
| Berhenti bila menolak | `followup_permission=no` atau `do_not_contact` → ditolak dan dibatalkan |
| Lead membalas → urutan berhenti | `message.in` membatalkan semua follow-up aktif lead itu (7.1) |
| Satu sentuhan per hari per lead | Tolak kalau ada follow-up `sent` atau `approved` di tanggal WIB yang sama |
| d1/d3/d7/d14/d30 relatif ke pesan keluar bermakna terakhir tanpa balasan | `scheduled_at ≥ last_outbound_at + N hari` dan `last_inbound_at < last_outbound_at` |
| d7 hanya dengan nilai baru | `reason` wajib dan ≥ 20 karakter, disetujui manusia |
| d14 = permission reset | Template wajib memuat opsi "boleh kabari". Balasan "boleh kabari" (deteksi frasa) → `followup_permission=yes` |
| d30 hanya untuk yang berizin | Wajib `followup_permission=yes` |
| Jam wajar | Hanya dalam `followup_send_window`. Di luar itu ditunda ke awal jendela berikutnya |
| Anti-blokir | Jeda global `followup_min_gap_seconds`, cap harian `followup_daily_cap`, dan saklar global `followup_enabled` |
| Isi aman | Draft tidak boleh memuat nomor rekening atau nominal rupiah (pembayaran hanya lewat alur CS AI). Tolak saat diusulkan, peringatan merah saat persetujuan |

### 10.3 Pemicu otomatis kandidat follow-up (oleh job AI)
Job AI hanya **mengusulkan**. Kandidat adalah lead `open` dengan `last_outbound_at` > 24 jam dan tanpa balasan sejak itu. Step dipilih berdasarkan follow-up yang sudah terkirim (d1 → d3 → d7 → d14 → d30). Draft mengikuti template Script 30 Hari (Lampiran D) dan dipersonalisasi dari transkrip.

## 11. Keamanan, privasi, dan keandalan

### 11.1 🔒 Keamanan
- Token ingest dan outbound berbeda arah dan berbeda nilai, minimal 32 byte acak, dibandingkan *constant-time*, dan hanya ada di `.env`.
- Endpoint ingest: token, allowlist IP, batas ukuran, idempotensi, dan rate limit.
- Akun GOWA `dashboard` dikunci di GOWA: `APP_BASIC_AUTH_DEVICES=…,dashboard:keuangan|admin-kelas|koord-guru`. Akun ini **tidak** mendapat `cs-ai`, karena follow-up lewat bridge.
- Transkrip ditampilkan sebagai teks (escape). Tidak ada `{@html}` di Svelte untuk konten chat.
- Semua aksi manusia dan AI tercatat di `lead_events`. Tool MCP juga tercatat di `ai_audit_log`.

### 11.2 Privasi (UU PDP)
- Tujuan data: proses pendaftaran. Jangan pakai data lead untuk keperluan lain tanpa dasar.
- **Retensi**: transkrip lead `lost`/`do_not_contact` dihapus otomatis setelah `transcript_retention_days_lost` hari oleh job harian. Profil ringkas tetap ada untuk laporan agregat. Transkrip lead `won` mengikuti kebijakan data santri.
- **Hak hapus**: tombol "Hapus data lead" (super_admin) menghapus lead, pesan, analisis, follow-up, dan event (CASCADE). Catat di log aplikasi (`slog`) siapa menghapus `lead_id` berapa dan kapan, **tanpa** isi data lead.
- MCP tidak pernah mengirim data kontak (9.1).

### 11.3 Keandalan dan kegagalan

| Situasi | Perilaku |
|---|---|
| Dashboard mati saat bridge mengirim event | Bridge menyimpan event di antrean lokal (file) dan mengulang dengan backoff (Lampiran B). Idempotensi mencegah duplikat |
| Bridge mati saat worker follow-up mengirim | `wa_outbox` → `failed` setelah 3 percobaan (backoff 1/5/15 menit). Follow-up tetap `approved`, lalu muncul di UI "Gagal kirim" dengan tombol Coba lagi |
| Event datang tidak berurutan | Stage hanya maju (6.2). `lead_messages` diurutkan dengan `sent_at` |
| Lead mengganti nomor / LID | `wa_chat_id` unik. Bila `phone` sama dengan lead lain, tampilkan saran "gabungkan lead" (manual, fase berikutnya) |
| Parser salah baca | `needs_review=1`, ditinjau manusia sebelum Jadikan Santri. Dialog konversi selalu bisa diedit |

## 12. Rencana implementasi bertahap

> Tiap fase = 1 PR di repo ini. Jalankan `go test ./...`, `go vet ./...`, `sqlc generate` (bersih), dan `vite build` sebelum PR. Uji E2E dengan agent_browser untuk halaman baru (inject session lewat SQL, lihat wiki testing).

### Fase A: Data model + ingest (fondasi)
**Lingkup**
- Migrasi:
  - `leads`, `lead_messages`, `lead_events`, `lead_labels`, `lead_label_links`, `crm_settings`;
  - `lead_analyses`, `lead_followups`, `wa_outbox` (dibuat sekarang supaya skema stabil, dipakai di fase C/D).
- Query sqlc. `LeadService` (buat/temukan lead, simpan pesan, transisi stage maju, parser form, deteksi stop). `CsAiIngestHandler`.
- Config `.env` (5.9). Notifikasi `proof.received` ke keuangan dan CS yang di-assign.
- **Bridge (repo GOWA, Lampiran B bagian 1–2)**: pengirim event + antrean lokal.

**Acceptance**
- Event contoh 7.1 membuat lead dan pesan. Event duplikat tidak membuat apa pun.
- Token salah → 401. Token kosong di `.env` → 404. IP di luar allowlist → 403.
- Urutan `form.sent` → `message.in` (form) → `payment_info.sent` → `proof.received` membawa stage dari `baru` ke `bukti_masuk`, dengan field terisi dan `payment_amount=149000`.
- Event mundur tidak memundurkan stage.
- Test parser lulus untuk semua contoh 7.2.

### Fase B: UI CRM + verifikasi + Jadikan Santri
**Lingkup**
- Halaman 8.1, 8.2, 8.4, dan 8.5 (status + pengaturan pemetaan paket).
- Aksi stage, label, assign, lost, dan do-not-contact.
- Verifikasi (6.3) dan Jadikan Santri (6.4) dalam satu transaksi, plus notifikasi Admin Kelas.
- Menu dan guard role (8.6).

**Acceptance**
- CS dan Keuangan bisa verifikasi dan konversi. Admin Kelas tidak bisa membuka `/app/leads` (403/redirect).
- Konversi menghasilkan tepat satu santri. Klik ganda paralel tetap satu santri (test konkurensi).
- Semua admin_kelas menerima notifikasi. Santri muncul di "Perlu Dilengkapi".
- Duplikat `no_wa` memunculkan peringatan dan opsi tautkan.
- Tampilan HP (390 px) tanpa scroll horizontal. Kanban hanya di ≥ 1024 px.

### Fase C: MCP CRM + job Hermes + laporan AI
**Lingkup**
- Kolom `ai_tokens.can_write` dan UI super_admin untuk token tulis.
- Tool 9.2 dengan redaksi dan validasi persona 9.3.
- Kartu analisis di UI, tombol Terapkan saran.
- Dokumentasi `docs/panduan/crm-ai-agent.md` (setup token + cron Hermes, Lampiran C).

**Acceptance**
- Token tanpa `can_write` mendapat error di tool tulis. Tool tulis tercatat di `ai_audit_log`.
- `detail_lead` tidak pernah memuat nomor, email, domisili, atau `wa_chat_id`. Test memakai transkrip berisi nomor dan link WA, dan output harus tersensor.
- Persona dengan field asing atau kata sensitif ditolak.
- `suggested_stage` tidak mengubah `leads.stage`.

### Fase D: Follow-up terjadwal
**Lingkup**
- Halaman 8.3. Worker follow-up dan outbox (10.1). Semua aturan 10.2.
- Klien bridge `POST /api/outbound`.
- **Bridge (Lampiran B bagian 3)**: endpoint outbound + konteks follow-up untuk AI.
- Default `followup_enabled=false`. Super admin menyalakannya setelah uji.

**Acceptance**
- Test per aturan 10.2: masing-masing ada kasus diblok dan kasus lolos.
- Lead membalas → follow-up aktif dibatalkan.
- Di luar jam kirim → ditunda. Cap harian dan jeda dihormati (test dengan jam palsu).
- Bridge mati → `failed` setelah 3 percobaan, lalu tombol Coba lagi berfungsi.
- Follow-up terkirim muncul di transkrip sebagai `followup`, dan balasan lead ditangani CS AI dengan konteks follow-up.

## 13. Metrik keberhasilan

Ukur sebelum dan sesudah. Angka target adalah hipotesis, bukan janji.
- Persentase lead dengan stage tercatat: target ≥ 95% otomatis.
- Waktu dari bukti masuk ke santri dibuat: target median < 4 jam kerja.
- Konversi `form_dikirim` → `won` dan `info_bayar_dikirim` → `won`, per minggu.
- Follow-up: tingkat balasan 72 jam dan jumlah opt-out. 🔒 Kalau opt-out atau keluhan naik, turunkan cap dan evaluasi template.
- Kualitas analisis AI: setiap minggu, sampel 10 analisis dinilai CS (setuju/tidak). Target ≥ 80% setuju sebelum follow-up dibuat lebih otomatis.

## 14. Risiko dan mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Nomor CS diblokir WhatsApp karena follow-up | Tinggi | Persetujuan manusia, cap harian, jeda, jam kirim, aturan berhenti, saklar global default mati |
| Analisis AI salah (persona/stage) | Sedang | Label "dugaan AI", saran tidak otomatis diterapkan, review mingguan |
| Data lead bocor lewat MCP | Tinggi | DTO tanpa kontak + redaksi + token tulis terpisah + audit |
| Parser form salah | Sedang | `needs_review`, dialog konversi bisa diedit |
| Dua santri dari satu lead | Sedang | UNIQUE + transaksi + test konkurensi |
| Ingest terputus tanpa ketahuan | Sedang | Status "event terakhir" di Integrasi WhatsApp. Opsional: notifikasi super_admin bila tidak ada event > 24 jam padahal ada chat |

## 15. Rekomendasi tambahan

1. **Mulai dari Fase A + B saja dan pakai 2–3 minggu.** Data tahap yang akurat jauh lebih berharga daripada analisis AI di atas data yang belum rapi. Fase C dan D dikerjakan setelah pipeline terbukti terisi benar.
2. **Follow-up mati dulu** (`followup_enabled=false`). Nyalakan untuk sebagian kecil lead (mis. label `uji-followup`) sebelum dibuka untuk semua.
3. **Satukan sumber "harga dan paket".** Saat ini harga ada di pengaturan bridge (`/cs-ai`). Fase lanjutan: Dashboard menjadi sumber, dan bridge membacanya lewat endpoint, supaya tidak ada dua tempat yang bisa berbeda.
4. **Pertimbangkan WhatsApp Business API resmi untuk follow-up** bila volume naik (> ~200 follow-up/minggu). Templat resmi dan opt-in jauh lebih aman dari blokir. Arsitektur `wa_outbox.channel` sudah disiapkan untuk kanal lain.
5. **Jangan menyimpan file bukti transfer di Dashboard** pada fase ini. Admin melihatnya di HP nomor CS. Kalau nanti perlu, unduh lewat GOWA `/message/:id/download` dengan akun terbatas dan simpan dengan kontrol akses.
6. **Review mingguan 10 transkrip** memakai rubrik QA di Pedoman Closing. Hasilnya dipakai untuk memperbaiki instruksi CS di `/cs-ai`.

---

## Lampiran A — Contoh event

```json
{"events":[
 {"event_id":"evt_a1","type":"message.in","occurred_at":"2026-10-24T03:12:45Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"wa_message_id":"3EB0A1","body":"Assalamu'alaikum, berapa biaya kelas tahsin?","media_type":""}},
 {"event_id":"evt_a2","type":"message.out","occurred_at":"2026-10-24T03:13:02Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"wa_message_id":"3EB0A2","body":"Wa'alaikumussalam, Kak 😊 …","sender":"ai"}},
 {"event_id":"evt_a3","type":"form.sent","occurred_at":"2026-10-24T03:20:10Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{}},
 {"event_id":"evt_a4","type":"payment_info.sent","occurred_at":"2026-10-24T03:31:00Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"paket_kode":"1x","nominal_text":"Rp149.000"}},
 {"event_id":"evt_a5","type":"proof.received","occurred_at":"2026-10-24T04:02:00Z","chat_id":"6281234567890@s.whatsapp.net","phone":"6281234567890","wa_name":"Fulanah","data":{"media_type":"gambar"}}
]}
```

## Lampiran B — Perubahan pendamping di bridge (repo GOWA)

Repo: `ruangsanadindonesia-art/go-whatsapp-web-multidevice`, folder `integrations/hermes-cs-bridge/`. Kerjakan sebagai PR terpisah di repo itu, bersamaan dengan Fase A dan Fase D.

**1. Pengirim event (Fase A)**
- Konfigurasi `.env` bridge: `DASHBOARD_URL` (mis. `http://127.0.0.1:8080`) dan `DASHBOARD_INGEST_TOKEN`. Kalau kosong, fitur mati.
- Titik emit event:

| Tempat di bridge | Event |
|---|---|
| `handleEvent` setelah lolos filter, untuk setiap pesan masuk non-admin | `message.in` (sertakan `media_type`) |
| `send()` / `sendImage()` ke chat lead (bukan ke nomor admin) | `message.out`: `sender=ai`, atau `followup` bila dari outbound |
| Pesan `is_from_me` dari HP CS | `message.out` dengan `sender=admin` |
| `renderReply` → `[[FORM]]` | `form.sent` |
| `renderReply` → `[[REKENING:x]]` | `payment_info.sent` dengan `paket_kode`, `nominal_text` dari settings |
| `mediaNote` dengan `paymentSent=true` | `proof.received` |
| `handoff()` | `handoff` |
| `SetPause` / `ClearPause` | `ai.paused` / `ai.resumed` |

- `event_id` unik (mis. `evt_<unix-nano>_<rand>`).

**2. Antrean lokal**
- File JSONL di `/data/outbox-dashboard/`. Worker mengirim batch ≤ 100 setiap 2 detik dengan backoff eksponensial sampai 10 menit. Batas antrean 50.000 event (event tertua dibuang dengan log peringatan).
- Event yang **ditolak karena skema** dibuang dan di-log. Event yang gagal karena jaringan atau 5xx diulang.

**3. Endpoint outbound (Fase D)**
- `POST /api/outbound` dengan `Authorization: Bearer <OUTBOUND_TOKEN>` (env bridge, sama dengan `CS_BRIDGE_OUTBOUND_TOKEN` Dashboard). Hanya listen di alamat bridge yang sudah ada.
- Body: `{"chat_id":"…","text":"…","reference_id":123,"purpose":"followup"}`.
- Perilaku:
  - 🔒 tolak bila chat dijeda (admin sedang menangani), dengan `409 {"error":"paused"}`;
  - 🔒 tolak teks yang memuat `[[` (tidak ada marker di follow-up);
  - kirim lewat GOWA device `cs-ai`;
  - simpan "follow-up terakhir" per chat di `state.json`, lalu sisipkan ke konteks prompt berikutnya: "Follow-up terakhir yang dikirim admin: … (tanggal)", supaya CS AI paham saat lead membalas;
  - emit `message.out(sender=followup)`;
  - balas `{"wa_message_id":"…"}`.
- Test: token salah → 401, chat dijeda → 409, marker → 400, sukses → event terkirim dan konteks tersisip.

## Lampiran C — Prompt job harian Hermes (analisis CRM)

```
Anda analis CRM Ruang Sanad. Gunakan HANYA tool MCP Dashboard berikut:
daftar_lead, detail_lead, simpan_analisis_lead, usulkan_followup, tambah_label_lead.

1. Panggil daftar_lead(status="open", butuh_analisis=true, limit=50).
2. Untuk setiap lead: detail_lead(lead_ref, pesan_sejak_id=<last_analyzed>).
   Analisis berdasarkan isi chat saja. Jangan menebak hal yang tidak tertulis.
3. simpan_analisis_lead dengan:
   - summary ≤ 600 karakter (apa yang dicari, di tahap mana, apa yang menahan)
   - persona sesuai skema; isi "tidak_diketahui" bila tidak jelas
   - interest_score 0–100, confidence rendah/sedang/tinggi
   - suggested_stage dan suggested_next_action yang konkret
   DILARANG menilai tingkat keagamaan, ekonomi, kesehatan, etnis, politik.
4. Usulkan follow-up HANYA bila: lead belum membalas pesan terakhir kita > 24 jam,
   tidak meminta berhenti, dan ada alasan relevan. Ikuti template Script 30 Hari
   (Lampiran D), personalisasi dari chat, satu CTA, ≤ 2 pertanyaan, tanpa nominal
   atau rekening, tanpa urgensi palsu, tanpa tekanan agama.
   Untuk d7 tulis reason berisi "nilai baru" yang nyata; jika tidak ada, jangan usulkan.
5. Akhiri dengan ringkasan: jumlah lead dianalisis, follow-up diusulkan, 3 lead paling
   panas (lead_ref + alasan singkat).
```

## Lampiran D — Template follow-up (dari Script 30 Hari)

| Step | Template dasar (dipersonalisasi AI, disetujui CS) |
|---|---|
| d1 | "Assalamu'alaikum, Kak [NAMA] 😊 Izin memastikan info kelas yang saya kirim kemarin sudah terbaca. Kalau berkenan, saya bantu dari satu hal dulu: cek jadwal yang cocok, jelaskan proses belajar, atau ringkas biaya. Yang mana, Kak?" |
| d3 | Pilih satu sesuai hambatan: jadwal / biaya / kecocokan-pemula / diskusi keluarga (teks di Script bagian Hari 3) |
| d7 | Hanya dengan update nyata: jadwal terkonfirmasi, jawaban pertanyaan tertunda, atau slot nyata. Tanpa update → jangan kirim |
| d14 | "Saya izin menutup follow-up pendaftaran agar tidak mengganggu, Kak 🙏🏻 Jika masih ingin dikabari ketika ada kelas [IKHWAN/AKHWAT] yang sesuai jadwal [PREFERENSI], cukup balas *boleh kabari*. Jika belum, tidak perlu membalas — kami tidak akan menghubungi lagi terkait pendaftaran ini." |
| d30 | Hanya bila `followup_permission=yes`: "Dulu Kakak sempat mencari kelas [..] untuk [KEBUTUHAN]. Saat ini ada opsi yang sesuai: [INFO TERKONFIRMASI]. Jika tidak, cukup balas *tidak dulu* dan kami tidak akan follow-up lagi." |
