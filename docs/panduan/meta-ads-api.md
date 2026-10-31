# Panduan: Menghubungkan Meta Ads (Marketing API) ke Dashboard Ruang Sanad

Panduan ini menjelaskan langkah demi langkah agar Dashboard Ruang Sanad bisa **membaca** data iklan Meta (biaya, impresi, klik, percakapan WhatsApp, dll.) secara otomatis, lalu menggabungkannya dengan data lead dan santri di CRM.

> **Cakupan:** hanya **membaca** data performa iklan (read-only). Dashboard **tidak** membuat, mengubah, atau menghentikan iklan.

---

## Daftar isi

1. [Gambaran umum](#1-gambaran-umum)
2. [Yang perlu disiapkan](#2-yang-perlu-disiapkan)
3. [Langkah 1: Pastikan aplikasi terhubung ke Business Portfolio](#langkah-1-pastikan-aplikasi-terhubung-ke-business-portfolio)
4. [Langkah 2: Aktifkan Marketing API di aplikasi](#langkah-2-aktifkan-marketing-api-di-aplikasi)
5. [Langkah 3: Buat System User](#langkah-3-buat-system-user)
6. [Langkah 4: Beri System User akses ke akun iklan](#langkah-4-beri-system-user-akses-ke-akun-iklan)
7. [Langkah 5: Hubungkan aplikasi ke System User](#langkah-5-hubungkan-aplikasi-ke-system-user)
8. [Langkah 6: Buat token akses](#langkah-6-buat-token-akses)
9. [Langkah 7: Catat ID akun iklan, versi API, dan zona waktu](#langkah-7-catat-id-akun-iklan-versi-api-dan-zona-waktu)
10. [Langkah 8: Uji token](#langkah-8-uji-token)
11. [Langkah 9: Simpan rahasia di server Dashboard](#langkah-9-simpan-rahasia-di-server-dashboard)
12. [Langkah 10: Cara Dashboard menarik data (untuk developer)](#langkah-10-cara-dashboard-menarik-data-untuk-developer)
13. [Langkah 11: Menghubungkan iklan dengan sumber lead](#langkah-11-menghubungkan-iklan-dengan-sumber-lead)
14. [Keamanan](#keamanan)
15. [Masalah umum dan solusinya](#masalah-umum-dan-solusinya)
16. [Checklist akhir](#checklist-akhir)
17. [Referensi](#referensi)

---

## 1. Gambaran umum

```
Meta Ads Manager ──(Marketing API / Insights)──▶ Dashboard Ruang Sanad
   biaya, impresi, klik,                          sinkron harian ke database
   percakapan WhatsApp                                      │
                                                            ▼
                                  digabung dengan Lead CRM & Santri
                                  → biaya per lead, biaya per santri, per LP/iklan
```

Komponen yang dipakai:

| Komponen | Fungsi |
|---|---|
| **Aplikasi Meta** (sudah dimiliki Ruang Sanad) | "Identitas" Dashboard di mata Meta |
| **System User** | Akun khusus server (bukan akun pribadi) yang memegang akses ke akun iklan. Tidak ikut terputus jika ada staf yang keluar |
| **Token akses System User** | "Kunci" yang dipakai Dashboard untuk memanggil API |
| **ID akun iklan** (`act_…`) | Akun iklan yang datanya dibaca |
| **Insights API** | Endpoint Meta yang memberi angka performa iklan |

---

## 2. Yang perlu disiapkan

- [ ] Akun Facebook dengan peran **Admin penuh** di Business Portfolio (Meta Business Suite) Ruang Sanad.
- [ ] Aplikasi Meta milik Ruang Sanad (sudah ada) — Anda perlu akses sebagai **Admin** aplikasi di [developers.facebook.com/apps](https://developers.facebook.com/apps).
- [ ] Akun iklan yang dipakai untuk iklan Ruang Sanad sudah **dimiliki atau dikelola** oleh Business Portfolio yang sama.
- [ ] Akses ke server Dashboard (atau developer yang mengelolanya) untuk menyimpan token.

> **Catatan:** Meta cukup sering mengganti nama menu dan tata letak. Jika nama menu di layar Anda sedikit berbeda dari panduan ini, cari menu dengan maksud yang sama (mis. "Business Settings" kini bernama **Settings** di Meta Business Suite; "Business Manager" kini disebut **Business Portfolio**).

---

## Langkah 1: Pastikan aplikasi terhubung ke Business Portfolio

System User hanya bisa memakai aplikasi yang **dimiliki Business Portfolio** yang sama.

1. Buka **[business.facebook.com](https://business.facebook.com)** → pilih portfolio **Ruang Sanad**.
2. Klik **Settings** (ikon roda gigi) di kiri bawah.
3. Di sidebar, buka **Accounts → Apps**.
4. Pastikan aplikasi Ruang Sanad ada di daftar.
   - Jika **belum ada**: klik **Add** → **Connect an app ID** → masukkan **App ID** (lihat di [developers.facebook.com/apps](https://developers.facebook.com/apps), tertulis di bagian atas dashboard aplikasi) → konfirmasi.
   - Jika aplikasi dimiliki portfolio lain, pemilik portfolio itu harus memindahkan/membagikannya dulu.

✅ **Hasil:** aplikasi Ruang Sanad muncul di **Settings → Accounts → Apps**.

---

## Langkah 2: Aktifkan Marketing API di aplikasi

1. Buka **[developers.facebook.com/apps](https://developers.facebook.com/apps)** → klik aplikasi Ruang Sanad.
2. Cari bagian produk / use case:
   - Tampilan baru: **Use cases** → **Add use case** → pilih yang berkaitan dengan **Marketing API / mengelola dan mengukur iklan** (mis. "Create & manage ads with Marketing API" atau "Measure ad performance data").
   - Tampilan lama: **Add Product** → **Marketing API** → **Set up**.
3. Jika diminta, hubungkan aplikasi dengan Business Portfolio Ruang Sanad.
4. Buka **App settings → Basic**, catat:
   - **App ID**
   - **App Secret** (klik **Show**; simpan aman, jangan dibagikan di chat)
5. Pastikan **Privacy Policy URL** dan kategori aplikasi terisi (Meta kadang meminta ini sebelum aplikasi bisa dipakai penuh).

> **Mode aplikasi & tingkat akses:** untuk **membaca data akun iklan milik sendiri** lewat System User, umumnya cukup dengan akses standar (*Standard Access*) tanpa App Review. Batasannya adalah *rate limit* yang lebih rendah — cukup untuk sinkron harian Ruang Sanad. Jika suatu saat API menolak dengan pesan terkait akses/review, lihat bagian [Masalah umum](#masalah-umum-dan-solusinya).

✅ **Hasil:** aplikasi punya produk/use case Marketing API, dan App ID + App Secret tercatat.

---

## Langkah 3: Buat System User

Langkah ini sesuai artikel bantuan Meta *[Add system users to your business portfolio](https://www.facebook.com/business/help/503306463479099)*.

1. Buka **Meta Business Suite → Settings**.
2. Di sidebar, buka **Users → System users**.
3. Klik **Add** / **Add new system user**.
4. Isi:
   - **Name:** `dashboard-ruang-sanad` (nama bebas, mudah dikenali)
   - **Role:** pilih **Employee / Regular system user** (BUKAN Admin)
5. Klik **Create system user**.

> **Kenapa bukan Admin?** Meta menyarankan memakai *regular system user* untuk panggilan API sehari-hari: jika tokennya bocor, cakupannya terbatas hanya ke aset yang diberikan. Lihat *[About system users](https://www.facebook.com/business/help/327596604689624)*.

✅ **Hasil:** System User `dashboard-ruang-sanad` muncul di daftar.

---

## Langkah 4: Beri System User akses ke akun iklan

1. Masih di **Settings → Users → System users**, klik `dashboard-ruang-sanad`.
2. Klik **Assign assets** (atau **Add assets**).
3. Pilih jenis aset **Ad accounts**.
4. Centang akun iklan Ruang Sanad (ulangi jika ada lebih dari satu akun iklan).
5. Pada izin, pilih tingkat **paling rendah yang cukup untuk membaca performa**, mis. **View performance** / **Analyse performance**.
   - Jangan beri "Manage campaigns" / "Full control" — Dashboard hanya membaca.
6. Klik **Save changes / Assign**.

✅ **Hasil:** di halaman System User, bagian **Assigned assets** menampilkan akun iklan Ruang Sanad dengan izin baca performa.

---

## Langkah 5: Hubungkan aplikasi ke System User

1. Di halaman System User `dashboard-ruang-sanad`, klik **Assign assets** lagi.
2. Pilih jenis aset **Apps** → centang aplikasi Ruang Sanad.
3. Beri izin **Develop app / Manage app** (dibutuhkan agar System User bisa membuat token untuk aplikasi tersebut).
4. Simpan.

> Di beberapa tampilan, langkah ini tidak diperlukan karena aplikasi langsung bisa dipilih saat membuat token (Langkah 6). Jika aplikasi tidak muncul di Langkah 6, kembali ke langkah ini.

---

## Langkah 6: Buat token akses

1. Di halaman System User `dashboard-ruang-sanad`, klik **Generate new token**.
2. **Select app:** pilih aplikasi Ruang Sanad.
3. **Token expiration:** pilih **Never** (tidak kedaluwarsa), agar sinkron otomatis tidak putus tiba-tiba.
   - Jika kebijakan internal mengharuskan kedaluwarsa, pilih **60 days** dan buat pengingat untuk membuat token baru sebelum habis.
4. **Permissions:** centang **hanya**:
   - `ads_read` — **wajib** (membaca data dan performa iklan)
   - `business_management` — **opsional**, hanya jika nanti Dashboard perlu membaca daftar akun iklan milik portfolio. Untuk tahap awal, **tidak perlu**.
   - Jangan centang `ads_management` (izin mengubah iklan).
5. Klik **Generate token**.
6. **Salin token segera** dan simpan di tempat aman (password manager). Meta **hanya menampilkan token sekali** — jika hilang, buat token baru.

> ⚠️ **Jangan** kirim token lewat WhatsApp, email, atau chat, dan jangan menaruhnya di kode/GitHub. Lihat [Langkah 9](#langkah-9-simpan-rahasia-di-server-dashboard).

✅ **Hasil:** satu token akses System User dengan izin `ads_read`, tersimpan aman.

---

## Langkah 7: Catat ID akun iklan, versi API, dan zona waktu

### a. ID akun iklan

1. Buka **[Ads Manager](https://adsmanager.facebook.com)**.
2. Lihat menu pilih akun di kiri atas; ID akun berupa angka panjang, mis. `123456789012345`.
   - Atau: **Business Suite → Settings → Accounts → Ad accounts** → klik akun → ID tertera di bawah nama.
3. Untuk API, ID ditulis dengan awalan **`act_`**: `act_123456789012345`.

### b. Versi API

1. Buka dashboard aplikasi di **developers.facebook.com/apps** → **App settings → Advanced** (atau lihat di Graph API Explorer).
2. Catat versi Graph API terbaru yang tersedia, formatnya `vNN.0` (mis. `v23.0`).
3. Meta merilis versi baru beberapa kali setahun dan setiap versi didukung sekitar 2 tahun. Dashboard menyimpan versi di pengaturan (`META_API_VERSION`) agar mudah dinaikkan.

### c. Zona waktu & mata uang akun iklan

1. Di Ads Manager → **Settings** akun iklan (atau Business Suite → Ad accounts → detail akun), lihat **Time zone** dan **Currency**.
2. Data Insights per hari mengikuti **zona waktu akun iklan**. Idealnya **Asia/Jakarta (WIB)** dan **IDR**, sama dengan Dashboard. Jika berbeda, beri tahu developer — zona waktu akun iklan **tidak bisa diubah** setelah dibuat.

✅ **Hasil:** tercatat `act_…`, versi API `vNN.0`, zona waktu, dan mata uang.

---

## Langkah 8: Uji token

### Cara A — Access Token Debugger (tanpa coding)

1. Buka **[developers.facebook.com/tools/debug/accesstoken](https://developers.facebook.com/tools/debug/accesstoken/)**.
2. Tempel token → **Debug**.
3. Periksa:
   - **App** = aplikasi Ruang Sanad
   - **Type** = System User
   - **Expires** = Never (atau tanggal 60 hari jika memilih itu)
   - **Scopes** berisi `ads_read`
   - **Valid** = True

### Cara B — Graph API Explorer

1. Buka **[developers.facebook.com/tools/explorer](https://developers.facebook.com/tools/explorer/)**.
2. Pilih aplikasi Ruang Sanad, tempel token di kolom **Access Token**.
3. Jalankan:
   ```
   GET act_123456789012345?fields=name,currency,timezone_name,account_status
   ```
   Harus mengembalikan nama akun, mata uang, zona waktu. `account_status: 1` = aktif.
4. Lalu uji Insights (7 hari terakhir, per kampanye):
   ```
   GET act_123456789012345/insights?level=campaign&date_preset=last_7d&fields=campaign_name,spend,impressions,clicks
   ```

### Cara C — Terminal (developer)

```bash
# Ganti nilai di bawah; jangan simpan token di riwayat shell bersama (gunakan variabel lingkungan).
export META_ADS_TOKEN='...'
export META_AD_ACCOUNT_ID='act_123456789012345'
export META_API_VERSION='v23.0'   # sesuaikan dengan Langkah 7b

curl -sG "https://graph.facebook.com/$META_API_VERSION/$META_AD_ACCOUNT_ID/insights" \
  --data-urlencode "access_token=$META_ADS_TOKEN" \
  --data-urlencode "level=campaign" \
  --data-urlencode "date_preset=last_7d" \
  --data-urlencode "fields=campaign_name,spend,impressions,clicks"
```

✅ **Hasil:** API mengembalikan data (bukan error). Jika error, lihat [Masalah umum](#masalah-umum-dan-solusinya).

---

## Langkah 9: Simpan rahasia di server Dashboard

Token dan App Secret disimpan sebagai **variabel lingkungan** di server (file `.env` produksi, yang **tidak** ikut ke Git), bukan di kode dan bukan di database yang bisa diakses semua peran.

Tambahkan ke `.env` server produksi:

```dotenv
# Meta Ads (Marketing API) — read-only
META_ADS_TOKEN=EAAB...              # token System User dari Langkah 6
META_AD_ACCOUNT_IDS=act_123456789012345   # pisahkan koma jika lebih dari satu akun
META_API_VERSION=v23.0              # dari Langkah 7b
META_APP_SECRET=...                 # dari Langkah 2, untuk appsecret_proof
```

Lalu restart aplikasi Dashboard (sesuai `docs/deployment`).

Siapa yang memasukkan: **pemilik/admin server**, langsung di server. Developer atau AI asisten **tidak perlu** melihat nilai token.

> Tambahkan juga baris kosong (tanpa nilai) untuk keempat variabel di `.env.example` agar developer lain tahu variabel ini ada.

---

## Langkah 10: Cara Dashboard menarik data (untuk developer)

Bagian ini adalah spesifikasi implementasi yang disarankan, mengikuti arsitektur repo (Handler → Service → Query → DB).

### 10.1 Request Insights

```
GET https://graph.facebook.com/{META_API_VERSION}/{act_id}/insights
  ?level=ad
  &time_increment=1
  &time_range={"since":"2026-10-01","until":"2026-10-31"}
  &fields=campaign_id,campaign_name,adset_id,adset_name,ad_id,ad_name,
          spend,impressions,reach,clicks,inline_link_clicks,actions,cost_per_action_type
  &limit=500
  &access_token=...
  &appsecret_proof=...
```

- `level=ad` + `time_increment=1` → **satu baris per iklan per hari**; agregat kampanye/ad set dihitung di Dashboard.
- Hasil berhalaman: ikuti `paging.next` sampai habis.
- `spend` berupa string desimal dalam mata uang akun (IDR) → simpan sebagai integer rupiah.
- Metrik per tujuan ada di array `actions`, mis.:
  - `onsite_conversion.messaging_conversation_started_7d` → **percakapan pesan dimulai** (iklan Click-to-WhatsApp; dihitung bila orang memulai percakapan setelah minimal 7 hari tidak aktif)
  - `link_click` → klik tautan (iklan ke LP)
  - `landing_page_view` → tampilan LP (jika Pixel terpasang di LP)
- Untuk rentang panjang/akun besar, Meta menyediakan **laporan asinkron** (POST ke `/insights` lalu polling `report_run_id`). Untuk ukuran Ruang Sanad, request sinkron harian per 1–7 hari biasanya cukup.

### 10.2 Jadwal sinkron

- **Harian** (mis. 02.00 WIB): tarik ulang **28 hari terakhir** dan *upsert* (ganti) baris yang sama. Angka Meta (terutama konversi/atribusi) masih bisa berubah beberapa hari setelah tanggal kejadian.
- Tombol **"Sinkron sekarang"** di halaman Performa Iklan (super_admin) untuk tarik manual.
- Catat waktu sinkron terakhir + pesan error terakhir agar terlihat di Dashboard.

### 10.3 Tabel yang disarankan (satu tabel per migrasi)

| Tabel | Isi |
|---|---|
| `meta_ad_insights_daily` | `tanggal`, `ad_account_id`, `campaign_id/name`, `adset_id/name`, `ad_id/name`, `spend`, `impressions`, `reach`, `clicks`, `link_clicks`, `messaging_started`, `synced_at`; UNIQUE (`tanggal`, `ad_id`) |
| `meta_ads_sync_runs` | `started_at`, `finished_at`, `status`, `rows`, `error` |
| `lead_source_meta_campaigns` | pemetaan `lead_source_id` ↔ `campaign_id` (atau `ad_id`) — lihat Langkah 11 |

### 10.4 Batas pemakaian (rate limit)

- Akses standar punya kuota panggilan lebih kecil. Sinkron sekali sehari dengan `level=ad` + paginasi aman untuk volume Ruang Sanad.
- Jika API membalas error kode **17**, **80004**, atau pesan "User request limit reached": hentikan, tunggu, coba lagi nanti (backoff); jangan diulang terus-menerus.
- Header respons `x-business-use-case-usage` menunjukkan persentase kuota yang terpakai.

### 10.5 Yang ditampilkan di Dashboard (usulan)

Halaman **Performa Iklan** (`/app/leads/iklan`, peran super_admin & cs):

- Ringkasan periode: total biaya, lead CRM, **biaya per lead (CPL)**, santri dari iklan, **biaya per santri**, percakapan WA menurut Meta vs lead tercatat.
- Tabel per kampanye/iklan: biaya, impresi, klik, percakapan WA, lead CRM (dari pemetaan sumber), closing, CPL, biaya per santri.
- Tren harian biaya vs lead.

---

## Langkah 11: Menghubungkan iklan dengan sumber lead

Data Meta memberi **biaya per kampanye/iklan**; data CRM memberi **lead & santri per sumber** (fitur *Sumber lead* di `/app/leads/sumber`). Keduanya digabung lewat **pemetaan**:

1. **Tahap awal (paling sederhana):** di halaman Sumber lead, setiap sumber dipetakan ke satu atau beberapa **kampanye** Meta. Contoh:
   - Sumber "LP Kelas Reguler Tahsin" ↔ kampanye "Tahsin Reguler – Okt 2026"
   - CPL sumber = total biaya kampanye terpetakan ÷ lead dari sumber itu.
2. **Lebih akurat (per iklan):** beri setiap iklan/LP **teks pembuka WhatsApp yang unik** atau **kode `ref`**, lalu daftarkan teks itu sebagai teks pemicu sumber lead. Pakai parameter dinamis Meta di URL iklan ke LP agar UTM terisi otomatis:
   ```
   ?utm_source=meta&utm_medium=paid&utm_campaign={{campaign.name}}&utm_content={{ad.name}}&ad_id={{ad.id}}
   ```
3. **Iklan Click-to-WhatsApp dengan teks default Meta** ("Bisakah saya mendapatkan info selengkapnya tentang ini?"): biayanya tetap terbaca per kampanye, tetapi lead **tidak bisa dibedakan per iklan** kecuali teks pembukanya dibuat unik per iklan. Ganti teks default di setiap iklan CTWA.

---

## Keamanan

- ✅ Gunakan **regular/employee System User** + izin **`ads_read`** saja.
- ✅ Simpan token & App Secret hanya di `.env` server; **jangan** commit ke Git, jangan kirim lewat chat.
- ✅ Aktifkan **appsecret_proof**: di dashboard aplikasi **App settings → Advanced → Require App Secret** = Yes. Dashboard lalu mengirim `appsecret_proof` = HMAC-SHA256(token, App Secret) di setiap request, sehingga token yang bocor tidak bisa dipakai tanpa App Secret.
- ✅ Jika token diduga bocor: di **Settings → Users → System users → dashboard-ruang-sanad**, cabut token (**Revoke tokens**), buat token baru (Langkah 6), perbarui `.env`, restart.
- ✅ Tinjau berkala (mis. tiap 6 bulan) daftar System User dan aset yang diberikan; hapus yang tidak dipakai.
- ✅ Data yang ditarik adalah angka agregat iklan, **bukan** data pribadi calon santri.

---

## Masalah umum dan solusinya

| Gejala / kode error | Kemungkinan penyebab | Solusi |
|---|---|---|
| `(#190) Invalid OAuth access token` / `Error validating access token` | Token salah salin, kedaluwarsa, atau dicabut | Cek di Access Token Debugger; buat token baru (Langkah 6) |
| `(#200) Permissions error` / `(#10) ... does not have permission` | Token tanpa `ads_read`, atau System User belum diberi akses akun iklan | Ulangi Langkah 4 & 6; pastikan scope berisi `ads_read` |
| `(#100) Unsupported get request` / `Object with ID 'act_...' does not exist` | ID akun salah, lupa awalan `act_`, atau akun bukan milik portfolio | Periksa Langkah 7a dan Langkah 4 |
| `(#100) ... is not a valid field` | Nama field salah / sudah diganti di versi API | Cek versi API; sesuaikan nama field |
| `(#17)` / `(#80004)` / "User request limit reached" | Rate limit | Tunggu lalu coba lagi; kurangi frekuensi; pakai `limit` & paginasi |
| `API access blocked` / pesan tentang App Review atau Advanced Access | Aplikasi belum memenuhi syarat akses | Lengkapi data aplikasi (Privacy Policy, kategori), pastikan aplikasi milik portfolio yang sama; jika tetap diminta, ajukan akses lanjut `ads_read` di **App Review → Permissions and features** |
| `appsecret_proof` error | "Require App Secret" aktif tapi Dashboard belum mengirim proof, atau App Secret salah | Pastikan `META_APP_SECRET` benar |
| Aplikasi tidak muncul saat **Generate token** | Aplikasi belum milik portfolio / belum diberikan ke System User | Langkah 1 dan Langkah 5 |
| Angka biaya di Dashboard beda tipis dengan Ads Manager | Data hari-hari terakhir masih diperbarui Meta, atau beda zona waktu | Tunggu sinkron berikutnya (tarik ulang 28 hari); cek zona waktu akun (Langkah 7c) |
| Percakapan WA menurut Meta ≠ lead CRM | Definisi berbeda (Meta menghitung percakapan baru setelah ≥7 hari tidak aktif; CRM menghitung nomor baru) | Normal; gunakan CRM untuk lead/closing dan Meta untuk biaya |

---

## Checklist akhir

Untuk admin Business Portfolio:

- [ ] Aplikasi Ruang Sanad tercantum di **Settings → Accounts → Apps** (Langkah 1)
- [ ] Marketing API aktif di aplikasi; App ID & App Secret tercatat (Langkah 2)
- [ ] System User `dashboard-ruang-sanad` (Employee/Regular) dibuat (Langkah 3)
- [ ] System User punya akses **lihat performa** ke akun iklan (Langkah 4)
- [ ] Aplikasi diberikan ke System User (Langkah 5)
- [ ] Token dibuat dengan scope `ads_read`, kedaluwarsa **Never**, tersimpan aman (Langkah 6)
- [ ] `act_…`, versi API, zona waktu & mata uang tercatat (Langkah 7)
- [ ] Token lolos uji di Access Token Debugger / Graph API Explorer (Langkah 8)
- [ ] **Require App Secret** diaktifkan (Keamanan)

Untuk pengelola server:

- [ ] `META_ADS_TOKEN`, `META_AD_ACCOUNT_IDS`, `META_API_VERSION`, `META_APP_SECRET` diisi di `.env` produksi (Langkah 9)
- [ ] Aplikasi di-restart

Untuk developer (setelah fitur sinkron dibuat):

- [ ] Sinkron harian berjalan dan menulis `meta_ads_sync_runs` dengan status sukses
- [ ] Angka biaya 7 hari terakhir cocok dengan Ads Manager (selisih kecil wajar)
- [ ] Pemetaan sumber lead ↔ kampanye terisi (Langkah 11)

---

## Referensi

- [Add system users to your business portfolio](https://www.facebook.com/business/help/503306463479099) — langkah resmi membuat System User dan token
- [About system users in Meta Business Suite](https://www.facebook.com/business/help/327596604689624) — perbedaan Admin vs Regular System User
- Dokumentasi developer Meta (developers.facebook.com):
  - Marketing API → **System Users**
  - Marketing API → **Insights API** (endpoint `/insights`, parameter `level`, `time_range`, `time_increment`, `fields`)
  - Graph API → **Changelog** (versi API terbaru)
  - Graph API → **Securing Requests** (`appsecret_proof`)
  - Tools → **Access Token Debugger**, **Graph API Explorer**
- Panduan terkait di repo ini: `docs/panduan/crm-ai-agent.md` (CRM & CS AI), halaman **Sumber lead** (`/app/leads/sumber`)
