# Tasbih Digital — Design

Tanggal: 2026-11-16 · Status: draft, menunggu persetujuan

## 1. Intent

Alat dzikir pribadi untuk pemilik akun Ruang Sanad. Fokus: menghitung dzikir dengan nyaman (animasi, getar, target), tetap jalan saat koneksi putus, dan dashboard pribadi untuk melihat progres serta riwayat.

- **Pribadi.** Data hanya milik user itu sendiri. Tidak ada halaman admin untuk melihat data user lain dan tidak ada leaderboard.
- **Role `user`**: menu "Tasbih" dan "Dashboard Tasbih" muncul di sidebar, dan setelah login user langsung diarahkan ke `/app/tasbih` (menggantikan `/app/profile`).
- **Role lain** (super_admin, cs, guru, dll.): bisa membuka `/app/tasbih` lewat link dan mendapat tasbih pribadi yang sama persis. Tidak ada entri menu untuk mereka.

## 2. Goals / Non-goals / Anti-goals

**Goals (v1)**
1. Counter dengan target per dzikir, animasi, getar, suara opsional, undo, reset, dan layar tetap menyala (Wake Lock).
2. Paket wirid berurutan yang otomatis berpindah ke dzikir berikutnya.
3. Dzikir custom milik user (buat, ubah, arsipkan).
4. Target harian dan streak.
5. Gamifikasi penuh: badge dan level (tanpa leaderboard).
6. Offline-first: hitungan tidak hilang saat koneksi putus atau halaman di-reload. Data disinkronkan ke server.
7. Dashboard berisi grafik progres dan riwayat.

**Non-goals (v1)**
- Paket wirid custom buatan user (dicatat sebagai calon v2).
- Halaman admin untuk mengelola dzikir bawaan. Dzikir bawaan di-seed lewat migrasi.
- Cold-start offline, yaitu membuka aplikasi dari nol tanpa internet. Ini butuh service worker yang menyimpan halaman Inertia, dan itu di luar cakupan. Offline-first di v1 berarti **halaman yang sudah terbuka tetap berfungsi tanpa koneksi, dan antrean tetap ada setelah reload atau tab ditutup**.
- Notifikasi pengingat (push).
- Statistik agregat lintas user.

**Anti-goals** (gagal meski secara teknis jalan)
- Hitungan hilang atau tercatat ganda.
- Tap terasa lambat (>50 ms) atau animasi patah-patah di HP kelas menengah.
- Data dzikir satu user bisa terbaca oleh user lain.
- Badge atau teks yang menilai kualitas ibadah seseorang. Copy menggambarkan kebiasaan ("7 hari berturut-turut"), bukan nilai spiritual.

## 3. Constraints

- Arsitektur tiga lapis: Handler → Service → Query (sqlc) → SQLite. Satu file handler untuk fitur ini.
- Satu migrasi = satu tabel. Migrasi lama tidak diubah. Nomor berikutnya: `0109`.
- Svelte 5 + Inertia. `fetch()` ke `/app/*` wajib menyertakan header `X-XSRF-TOKEN` (`getCSRFToken()`).
- Tidak ada library chart. Grafik memakai SVG buatan sendiri, memperluas `frontend/src/components/charts/`.
- Warna dari token `@theme` di `frontend/src/app.css`, tanpa hex hardcode.
- Batas hari memakai WIB (konsisten dengan seluruh app, `wib` di services).
- Font Arab belum ada di project. Saya tambahkan satu font Arab (mis. Amiri / Noto Naskh Arabic, self-host di `public/fonts` atau Google Fonts). **Keputusan saya.**

## 4. Approaches (sinkronisasi offline)

| | Ringkas | Trade-off |
|---|---|---|
| **A. Log per sesi (DIPILIH)** | Setiap sesi disimpan di IndexedDB dengan UUID, lalu dikirim batch ke server. Server melakukan insert idempoten. | Data paling kaya (jam, durasi, paket), aman di multi-perangkat. Kodenya lebih banyak (antrean, dedup). |
| B. Agregat harian | Perangkat hanya mengirim total per dzikir per hari. | Sederhana, tetapi data bisa saling menimpa di 2 perangkat dan tidak ada analisis jam atau durasi. |
| C. Hanya lokal | Tanpa server. | Ditolak: riwayat hilang saat ganti HP atau cache dibersihkan. |

User memilih A.

## 5. Konten dzikir bawaan

User meminta "dzikir default yang masyhur, rekomendasi kamu". Teks Arab perlu dicek ulang dengan mushaf/kitab saat implementasi. Sebaiknya ustadz Ruang Sanad membaca sekilas sebelum rilis. Ini bukan blocker.

| slug | Latin | Target default | Rujukan |
|---|---|---|---|
| subhanallah | Subhanallah | 33 | Muslim 597 |
| alhamdulillah | Alhamdulillah | 33 | Muslim 597 |
| allahu-akbar | Allahu Akbar | 33 | Muslim 597 |
| tahlil | Laa ilaaha illallah | 100 | Tirmidzi 3383 |
| tahlil-lengkap | Laa ilaaha illallahu wahdahu laa syariika lah, lahul mulku wa lahul hamdu wa huwa 'alaa kulli syai-in qadiir | 100 | Bukhari 6403, Muslim 2691 |
| istighfar | Astaghfirullah | 100 | Muslim 2702 |
| istighfar-taubat | Astaghfirullaha wa atuubu ilaih | 100 | Bukhari 6307 |
| tasbih-tahmid | Subhanallahi wa bihamdihi | 100 | Bukhari 6405, Muslim 2691 |
| tasbih-azhim | Subhanallahi wa bihamdihi, subhanallahil 'azhiim | 33 | Bukhari 6406 |
| hauqalah | Laa haula wa laa quwwata illaa billaah | 33 | Bukhari 6384 |
| shalawat | Allahumma shalli 'alaa Muhammad | 100 | Muslim 408 (keutamaan) |
| hasbunallah | Hasbunallaahu wa ni'mal wakiil | 33 | Bukhari 4563 |

Setiap dzikir menyimpan: teks Arab (berharakat), latin, arti (Bahasa Indonesia), rujukan, dan target default.

**Paket wirid bawaan**
1. **Dzikir Ba'da Shalat**: subhanallah 33 → alhamdulillah 33 → allahu-akbar 33 → tahlil-lengkap 1 (Muslim 597).
2. **Tasbih Fatimah (sebelum tidur)**: subhanallah 33 → alhamdulillah 33 → allahu-akbar 34 (Bukhari 3705).
3. **Istighfar 100**: istighfar 100 (Muslim 2702).

## 6. Data model (SQLite, satu tabel per migrasi)

```
0109 tasbih_dzikir
  id INTEGER PK
  user_id INTEGER NULL REFERENCES users(id) ON DELETE CASCADE  -- NULL = bawaan
  slug TEXT NULL UNIQUE                                         -- hanya bawaan
  teks_arab TEXT NOT NULL, latin TEXT NOT NULL, arti TEXT NOT NULL DEFAULT ''
  rujukan TEXT NOT NULL DEFAULT ''
  target_default INTEGER NOT NULL CHECK (target_default BETWEEN 1 AND 100000)
  urutan INTEGER NOT NULL DEFAULT 0
  diarsipkan_at DATETIME NULL
  created_at, updated_at DATETIME
  INDEX (user_id)
  + seed 12 dzikir bawaan

0110 tasbih_paket
  id PK, slug TEXT UNIQUE, nama, deskripsi, rujukan, urutan, created_at
  + seed 3 paket

0111 tasbih_paket_item
  paket_id FK, urutan INTEGER, dzikir_id FK, target INTEGER
  PRIMARY KEY (paket_id, urutan)
  + seed

0112 tasbih_sesi
  id TEXT PK                     -- UUID dari klien (kunci idempotensi)
  user_id INTEGER NOT NULL FK ON DELETE CASCADE
  dzikir_id INTEGER NOT NULL FK
  paket_id INTEGER NULL FK
  paket_run_id TEXT NULL         -- UUID yang sama untuk semua sesi dalam satu putaran paket
  jumlah INTEGER NOT NULL CHECK (jumlah BETWEEN 1 AND 100000)
  target INTEGER NOT NULL
  selesai_target INTEGER NOT NULL DEFAULT 0   -- 1 bila jumlah >= target
  mulai_at DATETIME NOT NULL, selesai_at DATETIME NOT NULL   -- UTC
  tanggal_wib TEXT NOT NULL      -- 'YYYY-MM-DD', dihitung server dari mulai_at
  jam_wib INTEGER NOT NULL       -- 0..23
  created_at DATETIME NOT NULL
  INDEX (user_id, tanggal_wib), INDEX (user_id, mulai_at)

0113 tasbih_pengaturan
  user_id INTEGER PK FK
  target_harian INTEGER NOT NULL DEFAULT 100 CHECK (target_harian BETWEEN 1 AND 100000)
  updated_at DATETIME

0114 tasbih_badge_user
  user_id FK, badge_kode TEXT, diraih_at DATETIME
  PRIMARY KEY (user_id, badge_kode)

0115 tasbih_target_user          -- override target per dzikir, per akun
  user_id FK ON DELETE CASCADE, dzikir_id FK, target INTEGER CHECK (1..100000)
  updated_at DATETIME
  PRIMARY KEY (user_id, dzikir_id)
```

Target efektif untuk sebuah dzikir (di luar paket) = override user, atau `target_default` bila tidak ada override. User bisa mengubah target dzikir bawaan maupun custom dari counter (`PUT /app/tasbih/dzikir/:id/target`). **Keputusan saya**: target tersimpan per akun, bukan per perangkat. Target item di dalam paket tetap, tidak bisa di-override.

Definisi badge dan level ditulis sebagai konstanta Go di service, bukan tabel. Pengaturan perangkat (getar, suara, layar menyala) disimpan di `localStorage` per perangkat. **Keputusan saya**: ini preferensi perangkat, bukan data akun.

Mengapa `tanggal_wib` dan `jam_wib` disimpan: semua grafik melakukan GROUP BY pada kolom ini, jadi tidak perlu konversi zona waktu di SQL.

## 7. Definisi sesi (klien)

Satu sesi = hitungan berturut-turut untuk satu dzikir. Sesi ditutup dan masuk antrean ketika:
- target tercapai (dalam paket, otomatis lanjut ke item berikutnya sebagai sesi baru),
- user mengganti dzikir atau paket,
- user menekan "Selesai" atau reset,
- tidak ada tap selama 5 menit, atau
- halaman tersembunyi lebih dari 5 menit (`visibilitychange`).

Kalau user tetap menghitung setelah target tercapai (di luar paket), sesi yang sama berlanjut. Target hanya memicu perayaan.

Sesi yang masih berjalan ditulis ke IndexedDB (debounce sekitar 300 ms) supaya reload tidak menghilangkan hitungan. Sesi dengan `jumlah = 0` dibuang.

**Putaran paket.** State putaran (paket_id, paket_run_id, indeks item, hitungan item berjalan) disimpan di IndexedDB `aktif`, sehingga reload atau tab ditutup akan **melanjutkan** putaran. Jeda idle lebih dari 5 menit menutup sesi item yang sedang berjalan (kalau jumlah > 0), tetapi putaran tetap bisa dilanjutkan. Item yang tersisa akan memulai sesi baru dengan `paket_run_id` yang sama. Mengganti dzikir/paket atau menekan "Selesai" berarti **meninggalkan** putaran. Sesi yang sudah terjadi tetap dicatat, tetapi putaran tidak terhitung selesai.

Keyboard: abaikan `keydown` dengan `event.repeat`, supaya menahan Spasi tidak menambah hitungan.

Undo hanya mengurangi hitungan sesi yang sedang berjalan (minimal 0). Sesi yang sudah dikirim tidak bisa diubah di v1.

## 8. Sinkronisasi

**Klien** (`frontend/src/lib/tasbih/`):
- `db.ts`: wrapper IndexedDB dengan store `antrean` (sesi tertutup) dan `aktif` (sesi berjalan).
- `sync.ts`: kirim antrean ketika: sesi ditutup, event `online`, halaman dibuka, dan setiap 60 detik selama ada antrean. Batch berisi maksimal 100 sesi. Hapus dari antrean hanya ID yang dikonfirmasi server. Backoff saat gagal.
- Indikator status di UI: "Tersimpan" / "Menunggu sinkron (n)" / "Offline".

**Server**: `POST /app/tasbih/sync` (JSON, wajib CSRF)
```
req:  { sesi: [{ id, dzikir_id, paket_id?, paket_run_id?, jumlah, target, mulai_at, selesai_at }] }
resp: { diterima: [id...], ditolak: [{id, alasan}], badge_baru: [...], ringkasan_hari_ini: {...} }
```
Validasi di service:
- UUID valid; 1 ≤ jumlah ≤ 100000; selesai_at ≥ mulai_at.
- `mulai_at` tidak lebih dari 5 menit di masa depan dan tidak lebih tua dari 400 hari.
- Dzikir harus bawaan atau milik user (dzikir yang diarsipkan tetap diterima).
- Paket harus ada.
- Tidak ada batas laju hitungan. **Keputusan saya**: ini alat pribadi tanpa leaderboard, sehingga tidak ada pihak lain yang dirugikan oleh angka yang tidak wajar, sedangkan batas laju berisiko membuang hitungan asli (anti-goal). Pencegahannya cukup di klien: `event.repeat` diabaikan.
- Untuk sesi dengan `paket_id`, `target` harus sama dengan target item paket untuk dzikir tersebut. Kalau tidak sama, sesi tetap diterima, tetapi `paket_id` dan `paket_run_id` dikosongkan (diperlakukan sebagai sesi biasa).
- `INSERT ... ON CONFLICT(id) DO NOTHING`. ID yang sudah ada milik user sama dianggap diterima. ID milik user lain dianggap ditolak, tanpa membocorkan apa pun.
- Sesi yang ditolak juga dihapus dari antrean klien dan dicatat di console, supaya antrean tidak macet.
- **Putaran paket selesai** = setiap `tasbih_paket_item` dari paket itu punya minimal satu sesi dengan `paket_run_id` yang sama dan `selesai_target=1` untuk dzikir item tersebut. Penghitungan dilakukan dengan query (GROUP BY `paket_run_id`). Putaran yang tidak lengkap tidak dihitung.
- Setelah insert, service mengevaluasi badge dan mengembalikan badge yang baru diraih. Klien menampilkan animasi badge.

## 9. Routes (`app/handlers/tasbih.go`, tanpa middleware role, semua scoped ke `user_id` sesi)

| Method | Path | Fungsi |
|---|---|---|
| GET | `/app/tasbih` | Halaman counter (Inertia `app/tasbih/Index`) |
| GET | `/app/tasbih/dashboard?rentang=7\|30\|90\|365` | Dashboard (Inertia `app/tasbih/Dashboard`) |
| GET | `/app/tasbih/riwayat?halaman=n&dzikir_id=` | JSON riwayat sesi berpaginasi (20/halaman) |
| POST | `/app/tasbih/sync` | JSON, lihat §8 |
| POST | `/app/tasbih/dzikir` | Buat dzikir custom (router.post → redirect back) |
| PUT | `/app/tasbih/dzikir/:id` | Ubah dzikir custom milik sendiri |
| DELETE | `/app/tasbih/dzikir/:id` | Arsipkan dzikir custom (soft) |
| PUT | `/app/tasbih/target` | Ubah target harian |
| PUT | `/app/tasbih/dzikir/:id/target` | Override target dzikir (bawaan/custom) untuk user ini |

Batas dzikir custom: 50 per user. Validasi: latin wajib, maksimal 300 karakter; arab maksimal 500 karakter dan opsional untuk dzikir custom.

Perubahan lain:
- `app/handlers/dashboard.go` (case `"user"`) dan `auth.go:roleHome` (`RoleUser`) → `/app/tasbih`.
- `AppLayout.svelte`: `homeHref` untuk role user → `/app/tasbih`. Tambah menu "Tasbih" dan "Dashboard Tasbih" dengan `show: role === "user"`. Menu Profil tetap ada.

## 10. Halaman counter (`/app/tasbih`)

Mobile-first, satu tangan.
```
┌──────────────────────────────┐
│ [Dzikir ▾]  [Paket ▾]   ● sync│
│        سُبْحَانَ اللهِ         │
│   Subhanallah · Maha Suci Allah│
│       ╭─────────────╮         │
│      │   ⟳  27 / 33  │  ← ring progres
│       ╰─────────────╯         │
│  ◦ ◦ ◦ ● ◦ ◦   ← butir tasbih bergeser
│                              │
│   (area tap besar ≈ 60% layar)│
│ [↶ Undo] [Reset] [Selesai]    │
│ Hari ini 240/300 ▓▓▓▓▓░  🔥 12 │
└──────────────────────────────┘
```
- Tap di mana saja pada area tap (pointerdown, bukan click, supaya tidak ada delay). Dukung juga keyboard: Spasi/Enter untuk tambah, Backspace untuk undo.
- Animasi: ring progres (SVG stroke-dashoffset), butir tasbih bergeser satu posisi per tap (transform, GPU), angka "pop". Saat target tercapai: getar pola panjang dan efek perayaan ringan. Saat paket selesai: perayaan penuh dan ringkasan paket.
- Getar: `navigator.vibrate(10)` per tap, pola khusus di kelipatan 33 dan target. Pada iOS Safari tidak tersedia, jadi diam-diam diabaikan.
- Suara klik opsional (Web Audio, default mati).
- Wake Lock API saat counter aktif (default nyala, bisa dimatikan).
- `prefers-reduced-motion`: animasi diganti fade sederhana.
- Mode fokus/layar penuh (sembunyikan chrome AppLayout).
- Sheet "Kelola dzikir": daftar bawaan + custom, form tambah/ubah/arsip.

## 11. Dashboard (`/app/tasbih/dashboard`)

Filter rentang: 7 / 30 / 90 / 365 hari.

1. **Baris KPI**: Hari ini vs target (ring) · Streak saat ini & terpanjang · Total sepanjang masa + level (bar progres ke level berikutnya) · Rata-rata harian dalam rentang.
2. **Tren harian**: bar per hari, dengan garis target harian dan perbandingan dengan rentang sebelumnya (mis. "+18% vs 30 hari sebelumnya").
3. **Heatmap kalender**: 365 hari ala kontribusi (intensitas = total/hari). Tooltip berisi tanggal dan total.
4. **Komposisi dzikir**: donut + legenda (top 6, sisanya digabung jadi "Lainnya").
5. **Pola waktu**: bar 24 jam (WIB). Penanda waktu shalat tidak ada di v1.
6. **Paket**: jumlah putaran selesai per paket.
7. **Badge**: grid badge yang sudah diraih (dengan tanggal) dan yang terkunci (dengan syarat).
8. **Riwayat sesi**: daftar berpaginasi (tanggal, jam, dzikir, jumlah/target, durasi, paket). Filter per dzikir.
9. Antrean lokal yang belum tersinkron **hanya** digabungkan ke: KPI "Hari ini" (dan status streak hari ini), serta daftar riwayat (baris berlabel "belum tersinkron" di atas). Grafik lain (tren, heatmap, komposisi, jam, level, badge) murni dari server, disertai catatan kecil "n sesi menunggu sinkron" bila antrean tidak kosong. **Keputusan saya**: supaya logika agregasi tidak diduplikasi di TypeScript. Halaman counter juga mencoba sync terlebih dahulu saat dashboard dibuka.

Empty state: user baru melihat ajakan "Mulai dzikir pertama" beserta tautannya.

Semua agregasi dilakukan oleh query sqlc (GROUP BY `tanggal_wib`, `jam_wib`, `dzikir_id`), lalu service merakit DTO. Rentang 365 hari untuk satu user hanya beberapa ribu baris, sehingga aman.

## 12. Gamifikasi

**Streak**: sebuah hari dihitung jika total hari itu ≥ `target_harian` saat ini. **Keputusan saya**: memakai target saat ini, bukan histori target, demi kesederhanaan. Kalau target diturunkan, streak lama bisa ikut "bertambah". Ini diterima. Streak saat ini tetap hidup sampai akhir hari ini (WIB): kalau hari ini belum tercapai, streak dihitung sampai kemarin.

**Level** (total hitungan sepanjang masa; ambang ±×2,5):

| Lv | Total | Lv | Total |
|---|---|---|---|
| 1 | 0 | 6 | 10.000 |
| 2 | 100 | 7 | 25.000 |
| 3 | 500 | 8 | 50.000 |
| 4 | 1.500 | 9 | 100.000 |
| 5 | 4.000 | 10 | 250.000 |

Nama level netral, berupa angka ("Level 4"). Tidak ada gelar spiritual.

**Badge** (kode → syarat):
- `langkah-pertama`: sesi pertama.
- `target-pertama`: pertama kali mencapai target harian.
- `paket-pertama`: menyelesaikan satu putaran paket.
- `total-1k`, `total-10k`, `total-100k`, `total-1m`: total kumulatif.
- `streak-3`, `streak-7`, `streak-30`, `streak-100`.
- `paket-ba'da-shalat-35`: 35 putaran Dzikir Ba'da Shalat dalam 7 hari terakhir (rata-rata 5x/hari).
- `subuh-7`: 7 hari berbeda dengan sesi yang dimulai pukul 03:00–06:00 WIB.
- `custom-pertama`: membuat dzikir custom pertama.
- `istighfar-1k`: total istighfar (kedua slug) 1.000.

Badge dievaluasi oleh satu fungsi service `EvaluasiBadge(userID)` yang dipanggil setelah: sync, membuat dzikir custom, dan mengubah target harian. Endpoint `router.*` (dzikir custom, target) menyimpan badge baru di flash session dan menampilkannya sebagai prop `badge_baru` pada render berikutnya. Halaman counter dan dashboard menampilkan animasi badge dari prop itu maupun dari respons sync. Badge dievaluasi saat sync, mengikuti kebijakan idempoten (`INSERT OR IGNORE`). Badge tidak pernah dicabut.

## 13. Testing

- Go service test (in-memory SQLite): idempotensi sync (kirim ulang batch yang sama), kepemilikan (ID atau dzikir milik user lain), validasi (batas laju, waktu masa depan, jumlah 0), batas hari WIB (sesi 23:59 vs 00:01 WIB), streak (bolong sehari, hari ini belum tercapai), level, setiap badge, arsip dzikir custom.
- Handler test: auth wajib, CSRF pada sync, role non-user tetap bisa akses.
- Unit TS untuk logika antrean/sesi tidak ada infrastruktur saat ini. Diverifikasi manual + E2E agent browser: hitung → matikan jaringan (`context.setOffline`) → reload → nyalakan → cek data di DB.

## 14. Diserahkan ke builder

- Detail visual (palet aksen di dalam token, bentuk butir, tipografi angka) mengikuti skill `ui-ux-pro-max` dan `interface-design` saat implementasi, dengan token dari `app.css`.
- Ekstensi komponen chart: boleh membuat komponen baru di `components/tasbih/` (Heatmap, Ring, HourBar) alih-alih mengubah chart lama yang dipakai halaman lain.
- Pemilihan font Arab dan cara memuatnya.
