# Penambahan Role GURU & KOORDINATOR GURU
## Dashboard Pengelolaan Mahasantri — Ruang Sanad

Dokumen ini menjelaskan fitur baru yang ditambahkan ke sistem yang sudah jalan, berdasarkan analisis dua spreadsheet:
- **TSI Guru** (penilaian guru, 42 sheet) — `1dDbdlcZKFk2iLmG_ueQpnBEiPDBrJ8kTO894Q0vWSZw`
- **Todo Koordinator Guru** (riayah guru, 11 sheet) — `1QDNP9OzTyezrGMdxeSUFSCzQd1A0i8tyMEIVJcDJ0og`

**Keputusan yang sudah dikonfirmasi owner:**
1. Dua role baru: **Guru** dan **Koordinator Guru** (terpisah, bukan digabung).
2. Absensi diinput **guru, per pertemuan, per santri**.
3. Skor TSI **sebagian otomatis** dari log sistem, sisanya manual oleh Koordinator Guru.
4. No WA santri sudah ditambahkan sendiri oleh owner ke sistem.

---

# BAGIAN 1 — TEMUAN PENTING DARI SPREADSHEET

Sebelum masuk fitur, ini kondisi nyata data yang harus dipahami — karena beberapa hal akan berubah signifikan saat dipindah ke aplikasi.

### 1.1 Struktur TSI Guru saat ini
- **42 sheet**: 41 tab per-guru (1 guru = 1 tab) + 1 sheet rekap.
- **Hanya 1 dari 41 tab yang benar-benar terisi lengkap** (Dina Riska Lestari). 1 tab terisi separuh (Ai Muliati). **39 sisanya template kosong.** Artinya: sistem penilaian ini secara praktis belum jalan — beban input manualnya terlalu berat. Ini justru alasan terkuat kenapa harus diaplikasikan.
- Penilaian bersifat **bulanan**: `Akhir Juni`, `Akhir Juli`, `Akhir Agustus`.
- Setiap tab menyimpan: identitas guru, beban mengajar (jumlah kelas reguler/privat + jumlah santri), detail tiap kelas (`Nama Kelas`, `Jumlah Santri`, `Pertemuan ke-`, `Batas Materi`, `Kesesuaian batas materi`), lalu blok nilai TSI.

### 1.2 Masalah kualitas data yang harus diperbaiki di aplikasi
Ini masalah nyata, bukan teoritis:

| Masalah | Bukti di file | Dampak | Solusi di aplikasi |
|---|---|---|---|
| Tab guru duplikat | `Kasmawati` dan `Ridwan Husnaeni Prikhtiarullah` masing-masing punya 2 tab | Nilai bisa dobel/hilang | Tabel `guru` dengan `id` sebagai primary key |
| Guru ada di rekap tapi tak punya tab | `Muhammad Yahya Ayasy S. Pd`, `Pasha Perdana Al-Ravin.,S.T.,C.SQ`, `Ramdani`, `Zhafirah Haya` | 4 guru tak bisa dinilai | Auto-generate periode penilaian untuk semua guru aktif |
| Nama sebagai kunci relasi | `Lufi Vidiasari S.Ak` vs `Lufi Vidiasari S.ak`; `Irni Sania Putri` vs `Irni Sania Putri, S.Pd., B.A., M.TM` | Join gagal, rekap salah | Relasi pakai `guru_id`, bukan string nama |
| Jadwal & santri privat tertanam di teks | `Private Rabu 06.00 (Bu Hendria)`, `Ptivate Senin & Jumat 18.30 (Bu Juni)` (typo di sumber) | Tak bisa difilter/dihitung | Kelas sudah terstruktur di tabel `kelas` sistem |
| Guru muncul di 1 sheet saja | `Mohamad Irvan Fahmi` (hanya di matriks kompetensi) | Data tidak konsisten | Satu master guru |
| Desimal koma Indonesia | `92,13%`, `36,80%` | Parsing error saat import | Simpan sebagai angka (REAL), format saat tampil |

### 1.3 Struktur Todo Koordinator Guru
Bukan todo-list biasa. Ada **dua model** yang bercampur:

- **Model A — tabel perencanaan** (Sheet 1): tiap indikator TSI dipetakan ke cara pengumpulan buktinya. Kolom: `Unsur Penilaian`, `Teknis Penilaian`, `Kebutuhan`, `Timing`, `Link Pendukung`, `Pelaksanaan`. Kolom `Link Pendukung` dan `Pelaksanaan` **kosong total** — perencanaan tidak pernah ditutup eksekusinya.
- **Model B — grid kepatuhan** (Sheet 3,4,6,7,9,11): matriks guru × periode berisi checkbox `TRUE`/`FALSE`. Tidak ada status "On Progress"/"Pending" — hanya sudah/belum.

Sheet 1 pada dasarnya adalah **spesifikasi operasional untuk penilaian TSI**. Ini yang membuat otomasi memungkinkan: koordinator sudah mendefinisikan sendiri "bukti" tiap indikator, dan sebagian besar bukti itu bisa dihasilkan sistem.

---

# BAGIAN 2 — ROLE GURU

Guru login ke aplikasi dan **hanya melihat kelas yang ia ampu**. Tidak bisa melihat data santri kelas lain, tidak bisa melihat nominal/keuangan, tidak bisa melihat nilai TSI guru lain.

## 2.1 Dashboard Guru (halaman utama)
Ringkasan pribadi:
- Total kelas diampu (dipecah Reguler / Privat), total santri.
- **Jadwal mengajar hari ini & pekan ini** — kartu besar, karena ini yang dibuka tiap hari.
- Kelas yang **belum diabsen** (pertemuan sudah lewat tapi absensi belum diisi) — badge merah. Ini langsung menyentuh indikator TSI `Membuat absen di kelas setiap selesai mengajar`.
- Santri yang perlu diriayah (absen berturut-turut / lama tidak hadir).
- Ringkasan nilai TSI pribadi bulan berjalan + tren 3 bulan (guru boleh lihat nilainya sendiri, tidak boleh lihat guru lain).
- Pengingat: pembinaan pekan ini, rapat guru bulan ini, tilawah harian hari ini.

## 2.2 Daftar Kelas Saya
Kartu per kelas berisi: nama kelas, level, frekuensi, jadwal, gender santri, jumlah santri / kapasitas, **pertemuan ke-berapa**, **batas materi terakhir**, tanggal pertemuan terakhir & berikutnya.

Kolom `Pertemuan ke-` dan `Batas Materi` di spreadsheet TSI diisi manual tiap bulan. Di aplikasi keduanya **dihitung otomatis** dari riwayat absensi — koordinator tidak perlu menagih lagi.

## 2.3 Detail Kelas & Roster Santri
Tabel santri satu kelas: nama, ID Mahasantri, usia, domisili, no WA, tanggal mulai belajar, **% kehadiran**, jumlah hadir/izin/sakit/alpa, tanggal hadir terakhir.

Aksi per santri:
- **Tombol "Chat WA"** → buka `wa.me/<no_wa>` di tab baru, dengan **template pesan siap pakai** (lihat 2.6).
- **Catatan riayah** → guru menulis catatan personal tentang santri (progres bacaan, kendala, hasil japri). Tersimpan dengan tanggal & penulis, jadi jejak riayah bisa ditelusuri.
- **Riwayat kehadiran santri** → kapan saja ia hadir/absen.

Aksi per kelas:
- **Broadcast WA ke seluruh santri kelas** → generate daftar link `wa.me` per santri (satu per satu, karena WA pribadi tidak bisa blast massal tanpa API). Kalau nanti pakai WhatsApp Business API, ini tinggal disambungkan.
- Salin daftar nomor untuk buat grup WA kelas.

## 2.4 Pertemuan & Absensi — fitur inti
Alur yang dipakai guru tiap kali mengajar:

1. Guru buka kelas → klik **"Mulai Pertemuan"**.
2. Sistem otomatis mengisi: **pertemuan ke-** (lanjut dari terakhir), **tanggal**, **jam mulai**.
3. Guru mengisi:
   - **Materi / batas materi** (mis. `Q.S. An Nazi'at`, `Halaman 156`) — bebas teks, dengan saran otomatis dari materi pertemuan sebelumnya.
   - **Absensi per santri**: `Hadir` / `Izin` / `Sakit` / `Alpa` / `Telat`.
   - **Catatan pertemuan** (opsional): kondisi kelas, kendala, santri yang perlu perhatian.
4. Klik **"Selesai"** → sistem mencatat **jam selesai**.

Yang dicatat sistem secara diam-diam (untuk TSI, bukan untuk dilihat guru):
- Selisih **jam mulai aktual vs jadwal seharusnya** → indikator `Memulai dan mengakhiri kelas sesuai waktu (tidak telat)`.
- **Jeda antara jam selesai kelas dan waktu absensi disimpan** → indikator `Membuat absen di kelas setiap selesai mengajar` dan `Riayah grup dengan update absen setelah pertemuan`.
- Pertemuan yang **dijadwal ulang** (reschedule) → indikator `Tidak reschedule kelas`.
- Pertemuan yang **diampu guru pengganti (badal)** → indikator `Tidak sering dibadal`.

**Fitur pendukung:**
- **Badal / guru pengganti**: guru mengajukan badal, pilih guru pengganti, isi alasan. Pertemuan tetap tercatat atas nama kelas, tapi ditandai `dibadal` + siapa penggantinya. Koordinator dapat notifikasi.
- **Reschedule**: guru mengubah tanggal pertemuan, wajib isi alasan. Tercatat sebagai reschedule.
- **Edit absensi terbatas**: absensi bisa diedit maksimal 48 jam setelah dibuat. Lewat itu perlu izin Koordinator — supaya angka kedisiplinan tidak dimanipulasi belakangan.
- **Rekap absensi kelas**: matriks santri × pertemuan (mirip grid di spreadsheet), bisa diekspor.

## 2.5 Kewajiban Rutin Guru (checklist pribadi)
Diambil dari sheet-sheet grid di Todo Koordinator, tapi dibalik: kalau di spreadsheet koordinator yang mencentang manual, di aplikasi **guru yang menandai sendiri** dan koordinator tinggal memantau.

- **Tilawah harian** — checkbox per hari (spreadsheet punya 60 kolom untuk Juli–Agustus). Di aplikasi: satu tombol di dashboard, otomatis tercatat tanggalnya. Muncul streak/rekap bulanan.
- **Absen pembinaan** — hadir/tidak per sesi pembinaan mingguan (Selasa), + alasan bila tidak hadir.
- **Absen rapat guru** — per rapat bulanan.
- **Share info Kalam Bersanad** — konfirmasi sudah share + upload screenshot (opsional).
- **Posting SS mengajar** — upload screenshot + tandai sudah mention koordinator.
- **Sapa kelas di luar jam mengajar** — guru mencatat aktivitas menyapa grup (quotes, kuis, PR). Ini indikator TSI berbobot yang selama ini paling sulit dibuktikan.
- **Isi kuesioner** — bila koordinator menyebarkan kuesioner.

Semua ini muncul sebagai **daftar tugas hari ini** di dashboard guru, bukan menu terpisah yang harus dicari.

## 2.6 Template Pesan WA (untuk riayah santri)
Tombol "Chat WA" tidak sekadar membuka chat kosong — guru memilih template, sistem mengisi variabel:

| Template | Kapan dipakai |
|---|---|
| Sapa santri baru | Santri baru masuk kelas |
| Pengingat pertemuan | H-1 / beberapa jam sebelum kelas |
| Follow-up santri tidak hadir | Setelah ditandai Alpa |
| Follow-up santri absen berturut-turut | Auto-trigger 3x alpa |
| Apresiasi progres | Santri menyelesaikan target materi |
| Info perubahan jadwal | Saat reschedule/badal |
| Pemberitahuan PR/tugas | Setelah pertemuan |

Variabel otomatis: `{nama_santri}`, `{nama_kelas}`, `{jadwal}`, `{pertemuan_ke}`, `{materi_terakhir}`, `{nama_guru}`. Template dikelola Koordinator Guru / Super Admin supaya bahasanya seragam dan sesuai adab.

## 2.7 Nilai TSI Saya
Guru melihat **nilainya sendiri**: skor 4 kategori, total, tren antar bulan, dan rincian per indikator. Yang terpenting: indikator otomatis menampilkan **angka mentahnya** ("absen dibuat tepat waktu 11 dari 12 pertemuan"), sehingga guru tahu persis apa yang harus diperbaiki — bukan sekadar menerima angka.

Guru **tidak bisa** melihat nilai guru lain, tidak bisa mengubah nilainya sendiri.

---

# BAGIAN 3 — ROLE KOORDINATOR GURU

Koordinator Guru mengelola guru, bukan santri. Ia tidak bisa input data pendaftaran atau keuangan.

## 3.1 Dashboard Koordinator
- Total guru aktif (Guru Tetap vs Part Time), total kelas & santri yang tercover.
- **Guru yang belum mengisi absensi** pekan ini — daftar nama, langsung bisa di-WA.
- **Kesesuaian Jumlah Kelas–Status**: sistem mengecek apakah beban kelas seorang guru sesuai statusnya (Tetap vs Part Time). Di spreadsheet ini diisi manual (`SESUAI`); di aplikasi dihitung otomatis dari aturan minimum/maksimum kelas per status.
- **Kesesuaian batas materi**: apakah progres materi tiap kelas sesuai kurikulum pada pertemuan ke-sekian. Bisa otomatis kalau kurikulum per level didefinisikan; kalau belum, tetap manual.
- Progres pengisian TSI bulan berjalan (berapa guru sudah dinilai dari total).
- Ringkasan kepatuhan: % kehadiran pembinaan, rapat, tilawah, share Kalam.
- Alert: guru dengan skor TSI turun tajam, guru sering dibadal, kelas yang hampir kosong.

## 3.2 Master Data Guru
Menggantikan 41 tab yang tersebar. Satu tabel, satu sumber kebenaran:
`nama`, `gelar`, `status` (Guru Tetap / Part Time), `jenis_kelamin`, `no_wa`, `email`, `tanggal_gabung`, `is_aktif`, foto.

Ditambah **matriks kompetensi** (dari Sheet 10 Todo Koordinator, saat ini kosong total):
`Hafalan Qur'an`, `Hafalan Tuhfah`, `Hafalan Jazariy`, `Hafalan Khaqaniy`, `Hafalan Syakhawiy`, `Sanad Qiro'ah`, `Bahasa Arab Pasif`, `Bahasa Arab Aktif`.

Kegunaan praktis matriks ini: saat menempatkan guru ke kelas level tertentu, sistem bisa merekomendasikan guru yang kompetensinya cocok — bukan sekadar yang jadwalnya kosong.

## 3.3 Penilaian TSI — semi-otomatis
Inilah perubahan terbesar dari spreadsheet.

**Struktur yang dipertahankan persis:** 4 kategori, bobot 40/30/20/10, 18 indikator, skala persen, rumus:
```
Skor kategori = bobot × rata-rata indikator yang terisi
Total TSI     = jumlah 4 skor kategori
```
Catatan penting dari analisis: indikator yang **kosong tidak dihitung nol** — ia dikeluarkan dari rata-rata. Contoh Dina Riska: Kompetensi punya 6 indikator, indikator ke-6 kosong, rata-rata dihitung dari 5 indikator saja → 92% × 40% = 36,80%. Aturan ini **wajib dipertahankan** di aplikasi, kalau tidak semua nilai akan berbeda dari spreadsheet.

### Pemetaan 18 indikator: mana yang bisa otomatis

**Kompetensi mengajar (40%)**

| # | Indikator (verbatim) | Sumber nilai |
|---|---|---|
| 1 | Kesesuaian dengan teknik mengajar metode AQU | **Manual** — hasil kunjungan kelas |
| 2 | Kesesuaian dengan kurikulum Ruang Sanad | **Semi-otomatis** — sistem bandingkan `pertemuan ke-` vs `batas materi` terhadap kurikulum level; koordinator konfirmasi |
| 3 | Tampilan mengajar sesuai standar R Sanad | **Manual** — hasil kunjungan kelas |
| 4 | Riayah grup dengan update absen setelah pertemuan | **OTOMATIS** — % pertemuan yang absensinya diisi ≤ X jam setelah kelas selesai |
| 5 | Menyapa grup di luar waktu mengajar (quotes, SS, kuis, PR, jawab pertanyaan) | **Semi-otomatis** — dihitung dari log aktivitas "sapa kelas" yang dicatat guru |
| 6 | Ketepatan pelafalan | **Manual** — hasil cek bacaan oleh Ust. Arif |

**Kepuasan Murid (30%)**

| # | Indikator | Sumber nilai |
|---|---|---|
| 1 | Tidak banyak maha santri yang mundur | **OTOMATIS** — rasio santri berstatus `tidak_lanjut` di kelas guru tsb |
| 2 | Kelas tidak dimerger | **OTOMATIS** — riwayat merge kelas di sistem |
| 3 | Tidak ada komplain dari maha santri di grup ataupun via admin kelas | **Semi-otomatis** — modul komplain; skor turun tiap komplain tercatat |
| 4 | Maha santri mengapresiasi guru dan berkenan terlibat mensosialisasikan ruang sanad | **Manual / survei** — dari kuesioner santri |

**Kedisiplinan (20%) — kategori paling bisa diotomasi**

| # | Indikator | Sumber nilai |
|---|---|---|
| 1 | Membuat absen di kelas setiap selesai mengajar | **OTOMATIS** — % pertemuan yang ada absensinya |
| 2 | Tidak reschedule kelas | **OTOMATIS** — jumlah reschedule ÷ total pertemuan |
| 3 | Tidak sering dibadal | **OTOMATIS** — jumlah badal ÷ total pertemuan |
| 4 | Memulai dan mengakhiri kelas sesuai waktu (tidak telat) | **OTOMATIS** — selisih jam mulai aktual vs jadwal |
| 5 | Mengikuti pembinaan | **OTOMATIS** — dari absen pembinaan |
| 6 | Menghadiri rapat guru | **OTOMATIS** — dari absen rapat |

**Kontribusi program (10%)**

| # | Indikator | Sumber nilai |
|---|---|---|
| 1 | Share info Kalam Bersanad | **OTOMATIS** — dari checklist share per pekan |
| 2 | Posting SS mengajar | **OTOMATIS** — dari log upload SS |

**Ringkasan: 11 dari 18 indikator bisa dihitung penuh oleh sistem, 3 semi-otomatis, 4 tetap manual.** Beban koordinator turun drastis — dari mengisi 18 angka × 43 guru tiap bulan (774 input), jadi hanya 4–7 angka per guru yang benar-benar butuh penilaian manusia.

### Cara kerja halaman penilaian
1. Koordinator pilih guru + periode (bulan).
2. Sistem menampilkan indikator otomatis **sudah terisi**, lengkap dengan angka mentahnya — mis. "Absen dibuat: 11/12 pertemuan (91,7%)". Transparan, bisa ditelusuri.
3. Koordinator mengisi indikator manual, boleh menambahkan catatan.
4. Koordinator boleh **override** nilai otomatis, tapi **wajib isi alasan** (tercatat sebagai audit trail). Ini penting — otomasi tidak boleh jadi kotak hitam yang tak bisa dikoreksi, tapi juga tidak boleh diubah diam-diam.
5. Total dihitung otomatis, periode bisa dikunci (`final`) agar tidak berubah lagi.

**Yang perlu diputuskan owner:** spreadsheet tidak punya kategori huruf/predikat (A/B/C atau Baik/Cukup) — hanya angka persen. Kalau ingin ada predikat, ambang batasnya harus ditentukan (mis. ≥90% Sangat Baik, 80–89% Baik, dst).

## 3.4 Rekap & Rapor TSI
- **Matriks rekap** guru × bulan (persis sheet rekap, tapi terisi otomatis): kolom per bulan berisi 4 skor kategori + total.
- Dipecah blok **Guru Tetap** dan **Guru Part Time** seperti aslinya.
- **Tren per guru** — grafik garis lintas bulan, langsung terlihat siapa yang menurun.
- **Peringkat guru** per periode (tidak ada di spreadsheet, tapi berguna untuk apresiasi & pembinaan terarah).
- **Rapor guru (PDF)** — bisa dikirim ke masing-masing guru.
- **Export Excel** dengan format sama seperti spreadsheet lama, supaya transisi tidak memutus kebiasaan.

## 3.5 Kunjungan Kelas
Dari Sheet 7 (`KUNJUNGAN KELAS`) — kolom `Target Waktu Kunjungan`, `Tanggal`, `Jam`, `Catatan Kunjungan`. Di spreadsheet baru terisi targetnya saja (`3-8 Agustus 2026` untuk Guru Tetap, `9-15 Agustus` untuk Part Time); pelaksanaan kosong semua.

Di aplikasi:
- Jadwalkan kunjungan per guru (target rentang tanggal, lalu tanggal & jam aktual).
- Saat kunjungan, koordinator mengisi **form penilaian kunjungan** yang langsung mengisi indikator manual Kompetensi #1, #3 (dan #6 bila cek bacaan dilakukan). Jadi kunjungan tidak berhenti jadi catatan — langsung jadi nilai.
- Status: `Dijadwalkan` / `Terlaksana` / `Ditunda` / `Batal` + catatan.
- Reminder otomatis mendekati target.

## 3.6 Pembinaan & Rapat Guru
Menggantikan Sheet 3, 5, 8.

- **Timeline pembinaan** (Sheet 8: `BULAN`, `PEKAN KE-`, `TANGGAL`, `TOPIK`, `KETERANGAN`) → jadwal program pembinaan Juni–Oktober. Contoh topik nyata: *"Al Arba'un Al Quraniyyah part 1 bersama Syaikh Bakr Suleiman Ibrahim Alzamli"*, *"Tausiyah/pembekalan bersama U Arif"*, *"Mabadi Kitab At Tibyan"*.
- **Absensi pembinaan** — grid guru × sesi, ganti checkbox manual dengan self-check guru + verifikasi koordinator. Status per sesi: `Terlaksana` / `Libur` + alasan (spreadsheet mencatat `Libur idul adha`, `libur U Arif ada Keg lain`).
- **Absensi rapat guru** — bulanan.
- **Reminder otomatis** — di Sheet 1 kolom `Pelaksanaan` ada satu catatan: *"buat sistem reminder"*. Itu permintaan koordinator sendiri, dan di aplikasi jadi gratis: reminder H-1 pembinaan/rapat via notifikasi + link WA.

## 3.7 Kalam Bersanad
Dari Sheet 4 — program kajian pekanan (Sabtu) dengan topik & kitab. Contoh: *"Adab-Adab Shalat (Bagian 2) (Kitab Bidayatul Hidayah)"*, *"Muhkam dan Mutasyabih (Kitab Mawaridul Bayan fii Ulumil Qur'an)"*.

Fitur: kelola jadwal & materi kajian, lacak guru yang sudah share info (indikator Kontribusi #1), dan sediakan materi promosi siap share agar guru tinggal forward.

## 3.8 Todo & Riayah Koordinator
Sheet 1 diubah dari tabel rencana statis menjadi **todo aktif**:
- Field: `unsur/kegiatan`, `teknis pelaksanaan`, `kebutuhan`, `deadline`, `PIC`, `status`, `link pendukung`, `catatan`.
- Status: `Belum` / `Proses` / `Selesai` / `Batal` — spreadsheet belum punya kosakata status, ini perlu diperkenalkan.
- **Todo berulang**: harian (tilawah), mingguan (pembinaan, Kalam), bulanan (rapat, penilaian TSI), 4-bulanan (survei). Sistem generate otomatis, koordinator tidak perlu bikin manual tiap periode.
- **Riayah personal per guru**: halaman satu guru berisi seluruh jejak — nilai TSI, riwayat kunjungan, kehadiran pembinaan, catatan personal, komunikasi WA. Ini yang selama ini tersebar di 11 sheet.
- **Tombol WA ke guru** dengan template: undangan rapat, reminder pembinaan, teguran absensi belum diisi, apresiasi nilai naik, konfirmasi jadwal kunjungan.

## 3.9 Kuesioner & Survei
Dari Sheet 11 (`SHARE KUESIONER`) dan kebutuhan indikator Kepuasan Murid.
- Buat kuesioner (untuk santri atau guru), sebar via link, lacak siapa sudah mengisi.
- Hasil survei kepuasan santri **otomatis mengisi** indikator Kepuasan Murid #4.
- Kuesioner alasan santri mundur — spreadsheet menyebut kebutuhan ini eksplisit: *"kuesioner pengecekan alasan santri (alasan detail) yang mundur"*. Hasilnya berguna dua arah: menilai guru, dan memperbaiki program.

---

# BAGIAN 4 — TABEL DATABASE BARU

Ringkas, sebagai gambaran cakupan (detail teknis menyusul di prompt implementasi):

| Tabel | Isi |
|---|---|
| `guru` | Perluasan tabel yang sudah ada: status, no WA, gender, email, tanggal gabung, is_aktif |
| `guru_kompetensi` | 8 kompetensi per guru (hafalan, sanad, bahasa Arab) |
| `pertemuan` | 1 baris per pertemuan: kelas, pertemuan ke-, tanggal, jam mulai/selesai, materi, is_reschedule, is_badal, guru_pengganti, catatan |
| `absensi` | 1 baris per santri per pertemuan: status (hadir/izin/sakit/alpa/telat), catatan |
| `tsi_kriteria` | Master 18 indikator: kategori, bobot, urutan, sumber (auto/manual), nama verbatim |
| `tsi_periode` | Periode penilaian per guru per bulan + status (draft/final) |
| `tsi_nilai` | Nilai per indikator per periode + flag override + alasan override |
| `kunjungan_kelas` | Target, tanggal & jam aktual, status, catatan, hasil penilaian |
| `pembinaan` | Sesi pembinaan: tanggal, pekan, topik, keterangan, status |
| `pembinaan_absen` | Kehadiran guru per sesi |
| `rapat_guru` + `rapat_absen` | Rapat bulanan & kehadiran |
| `kalam_bersanad` | Jadwal kajian, topik, kitab + log share per guru |
| `tilawah_harian` | Checklist harian per guru |
| `todo_koordinator` | Todo + recurring rule |
| `catatan_riayah` | Catatan personal (target: guru atau santri), penulis, tanggal |
| `wa_template` | Template pesan WA + variabel |
| `kuesioner` + `kuesioner_jawaban` | Survei |

---

# BAGIAN 5 — MATRIKS AKSES 6 ROLE

| Fitur | Super Admin | CS | Admin Kelas | Keuangan | **Koord. Guru** | **Guru** |
|---|---|---|---|---|---|---|
| Data pendaftaran santri | ✅ | ✅ | 👁 | 👁 | 👁 | 👁 kelasnya |
| Data keuangan/infaq | ✅ | ✗ | ✗ | ✅ | ✗ | ✗ |
| Pembagian & manajemen kelas | ✅ | ✗ | ✅ | ✗ | 👁 | 👁 kelasnya |
| Pertemuan & absensi | ✅ | ✗ | 👁 | ✗ | 👁 | ✅ kelasnya |
| Catatan riayah santri | ✅ | ✗ | ✅ | ✗ | 👁 | ✅ kelasnya |
| Master data guru | ✅ | ✗ | 👁 | ✗ | ✅ | 👁 dirinya |
| Penilaian TSI | ✅ | ✗ | ✗ | ✗ | ✅ | 👁 dirinya |
| Kunjungan kelas | ✅ | ✗ | ✗ | ✗ | ✅ | 👁 dirinya |
| Pembinaan & rapat | ✅ | ✗ | ✗ | ✗ | ✅ | ✅ absen sendiri |
| Todo koordinator | ✅ | ✗ | ✗ | ✗ | ✅ | ✗ |
| Kuesioner | ✅ | ✗ | 👁 | ✗ | ✅ | 👁 |
| Kelola user & role | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ |

✅ input & edit · 👁 lihat saja · ✗ tidak ada akses

---

# BAGIAN 6 — CATATAN JUJUR & RISIKO

Beberapa hal yang perlu disadari sebelum membangun:

1. **Otomasi TSI hanya seakurat kedisiplinan input.** Kalau guru tidak mengisi absensi, indikator otomatis jadi kosong atau rendah — dan itu memang tujuannya. Tapi di masa transisi, ekspektasikan nilai turun bukan karena guru memburuk, melainkan karena datanya baru mulai terekam. Jangan bandingkan bulan pertama aplikasi dengan bulan terakhir spreadsheet.

2. **Indikator "sapa grup" dan "posting SS" tetap self-reported.** Sistem tidak bisa membaca grup WA. Yang bisa dilakukan hanya menyediakan tempat pencatatan + upload bukti. Ini bukan otomasi sungguhan, hanya pemindahan dari ingatan koordinator ke basis data.

3. **Bobot 40/30/20/10 menempatkan 40% pada hal yang paling sulit diukur** (kompetensi mengajar, yang bergantung kunjungan kelas). Kunjungan kelas di spreadsheet **belum pernah terlaksana satu pun**. Kalau pola ini berlanjut, 40% penilaian akan kosong terus. Saran: jadwalkan kunjungan sebagai todo berulang dengan reminder, atau pertimbangkan ulang bobotnya.

4. **`Kesesuaian batas materi` butuh kurikulum terstruktur.** Saat ini "kesesuaian" dinilai berdasarkan penilaian pribadi koordinator (`batasan masing-masing` untuk kelas talaqqi). Otomasi indikator ini baru mungkin kalau ada tabel kurikulum: level → pertemuan ke-N → materi yang seharusnya. Kalau belum ada, biarkan manual dulu — jangan dipaksakan otomatis.

5. **Data spreadsheet lama sebaiknya tidak diimpor.** Hanya 1 dari 41 guru punya data lengkap. Lebih bersih mulai dari periode berjalan, dan simpan spreadsheet lama sebagai arsip.

6. **Kelas privat menyimpan nama santri di judul kelas** (`Private Rabu 06.00 (Bu Hendria)`). Saat migrasi, ini harus dipecah jadi relasi kelas–santri yang benar, bukan dibawa apa adanya sebagai teks.

---

# BAGIAN 7 — YANG SAYA SARANKAN TIDAK DIBANGUN DULU

Agar cakupan tidak melebar dan rilis pertama cepat sampai:
- Integrasi WhatsApp Business API (cukup `wa.me` link dulu — 90% manfaat, 5% biaya).
- Aplikasi mobile native (web responsif sudah cukup untuk guru).
- Payroll / penggajian guru berbasis nilai TSI.
- Video call / kelas online di dalam aplikasi.
- Modul kurikulum lengkap per level (kecuali kalau otomasi `kesesuaian materi` jadi prioritas).
- Import data TSI lama.

---

# URUTAN PEMBANGUNAN YANG DISARANKAN

| Tahap | Isi | Alasan |
|---|---|---|
| **1** | Role Guru + login, Kelas Saya, Pertemuan & Absensi per santri, tombol WA | Ini fondasi. Tanpa data absensi, otomasi TSI tidak punya bahan. |
| **2** | Role Koordinator + master guru, absen pembinaan & rapat, tilawah, kunjungan kelas | Mengumpulkan sisa bahan untuk TSI. |
| **3** | Mesin TSI: 18 indikator, 11 otomatis, penilaian manual, rekap & rapor | Baru bermakna setelah tahap 1–2 berjalan ≥1 bulan. |
| **4** | Todo koordinator, kuesioner, template WA, reminder otomatis | Penyempurnaan alur kerja. |

Menjalankan tahap 3 lebih dulu adalah kesalahan umum — hasilnya sama seperti spreadsheet sekarang: form kosong yang tidak ada yang mengisi.
