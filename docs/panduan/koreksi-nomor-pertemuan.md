# Panduan Admin Kelas: Mengatur dan Mengoreksi Nomor Pertemuan

Panduan ini menjelaskan bagian **"Pertemuan terakhir sebelum sistem"** dan tombol **Koreksi nomor** di halaman detail kelas (`Kelas → pilih kelas`).

Hak akses: **Admin Kelas** dan **Super Admin**.

---

## 1. Kapan fitur ini dipakai

| Situasi | Yang dipakai |
|---|---|
| Kelas lama baru mulai memakai aplikasi, dan **belum ada satu pun pertemuan** tercatat di aplikasi | Isi kolom **Pertemuan terakhir sebelum sistem**, lalu klik **Simpan** |
| Kelas **sudah punya pertemuan** di aplikasi, tetapi nomor pertemuannya ternyata tidak sesuai rekap riil | Klik **Koreksi nomor** |

Setelah pertemuan pertama dicatat guru, kolom isian terkunci dan tombol **Simpan** berganti menjadi **Koreksi nomor**.

---

## 2. Istilah yang perlu dipahami

| Istilah | Arti |
|---|---|
| **Anchor** (Pertemuan terakhir sebelum sistem) | Nomor pertemuan riil **terakhir sebelum** kelas memakai aplikasi. Contoh: rekap manual berhenti di pertemuan 226, maka anchor = 226. |
| **Nomor global** | Nomor urut pertemuan kelas sejak awal, dipakai sistem secara internal. Pertemuan pertama di aplikasi = anchor + 1. |
| **Nomor level** | Nomor yang **tampil** di kartu "Pertemuan Terakhir" (contoh "Ke-237 · Nomor dihitung sejak awal level TALAQQI"). Bila kelas belum pernah ganti level di aplikasi, nomor level sama dengan nomor global. Bila kelas ganti level, nomor level mulai lagi dari 1 di level baru. |
| **Pertemuan terdampak** | Jumlah pertemuan yang sudah tercatat di aplikasi untuk kelas ini. Semuanya ikut digeser saat koreksi. |

> **Penting.** Kolom di jendela Koreksi meminta **anchor**, yaitu nomor terakhir **sebelum sistem**. Kolom ini **bukan** tempat mengisi nomor pertemuan terakhir yang tampil di kartu. Kesalahan ini paling sering terjadi. Lihat Bagian 7.

---

## 3. Rumus

**Pertemuan berikutnya** (nomor global):

```
pertemuan berikutnya = nomor global terakhir + 1
(bila belum ada pertemuan di aplikasi: anchor + 1)
```

**Koreksi nomor**:

```
selisih      = nomor terakhir yang BENAR − nomor terakhir yang TAMPIL SEKARANG
anchor baru  = anchor lama + selisih
```

Semua pertemuan yang sudah tercatat bergeser sebesar **selisih**. Urutan dan jaraknya tetap sama.

---

## 4. Mode A: Kelas belum punya pertemuan di aplikasi

**Langkah:**

1. Buka halaman detail kelas.
2. Cari kotak biru **Pertemuan terakhir sebelum sistem**.
3. Isi nomor pertemuan riil terakhir dari rekap manual.
4. Pastikan teks "Pertemuan berikutnya akan menjadi …" sudah benar.
5. Klik **Simpan**.

**Contoh 1: Kelas pindah dari rekap manual**

- Rekap manual kelas berhenti di pertemuan ke-226.
- Isi **226**, lalu klik **Simpan**.
- Teks berubah menjadi "Pertemuan berikutnya akan menjadi 227".
- Saat guru mencatat pertemuan pertama di aplikasi, nomornya **Ke-227**.

**Contoh 2: Kelas baru dari nol**

- Isi **0** (atau biarkan 0).
- Pertemuan pertama di aplikasi menjadi **Ke-1**.

Catatan:
- Nilai ini masih bisa diubah dan disimpan ulang selama belum ada pertemuan tercatat.
- Bila kelas sudah **ganti level** sebelum ada pertemuan di aplikasi, level baru dianggap mulai setelah anchor. Pertemuan pertama tampil sebagai **Ke-1** di level baru, sedangkan nomor globalnya tetap anchor + 1.

---

## 5. Mode B: Koreksi nomor (kelas sudah punya pertemuan)

### Syarat

- Sudah ada minimal satu pertemuan di aplikasi.
- **Tidak ada pertemuan yang sedang berlangsung.** Bila guru sedang mengajar, tombol tidak aktif dan muncul pesan "Koreksi sementara diblokir karena ada pertemuan yang sedang berlangsung". Tunggu guru menyelesaikan pertemuan.
- Alasan koreksi wajib diisi, maksimal 500 karakter.

### Langkah

1. Siapkan **nomor pertemuan terakhir yang benar** dari rekap riil (absensi manual, catatan guru, atau rekap admin sebelumnya).
2. Lihat kartu **Pertemuan Terakhir** di halaman kelas. Catat nomor yang tampil, misalnya "Ke-237".
3. Hitung `selisih = nomor benar − nomor tampil`.
4. Klik **Koreksi nomor**.
5. Di kolom **Pertemuan terakhir sebelum sistem yang benar**, isi `anchor lama + selisih`. Nilai anchor lama sudah terisi otomatis di kolom ini.
6. Periksa ringkasan di jendela koreksi:
   - **Anchor**: lama → baru
   - **Pertemuan terdampak**: jumlah pertemuan yang ikut digeser
   - **Nomor global terakhir**: lama → baru. Angka baru ini harus sama dengan nomor benar dari rekap riil, bila kelas belum pernah ganti level di aplikasi.
   - **Pertemuan berikutnya**: nomor global pertemuan selanjutnya
7. Isi **Alasan koreksi** secara spesifik: sumber data dan nomor yang benar.
8. Klik **Terapkan koreksi**.

---

## 6. Contoh kasus Mode B

### Contoh 3: Nomor di aplikasi terlalu kecil (kasus di kelas TALAQQI)

**Kondisi sekarang:**

- Anchor: **226**
- Sudah ada **11 pertemuan** di aplikasi (227 sampai 237)
- Kartu Pertemuan Terakhir: **Ke-237** · level TALAQQI

Rekap riil guru ternyata menunjukkan pertemuan terakhir seharusnya **Ke-240**.

**Hitungan:**

```
selisih     = 240 − 237 = +3
anchor baru = 226 + 3   = 229
```

**Isi kolom:** `229`

**Ringkasan di jendela koreksi:**

| Kolom | Nilai |
|---|---|
| Anchor | 226 → 229 |
| Pertemuan terdampak | 11 |
| Nomor global terakhir | 237 → 240 |
| Pertemuan berikutnya | Ke-241 |

**Contoh alasan:** "Rekap absensi manual Ustadz X: sebelum pakai aplikasi kelas sudah sampai pertemuan 229, bukan 226."

**Hasil:**

- Pertemuan di aplikasi menjadi 230 sampai 240.
- Kartu Pertemuan Terakhir menampilkan Ke-240.
- Pertemuan berikutnya Ke-241.

---

### Contoh 4: Nomor di aplikasi terlalu besar

**Kondisi sekarang:** sama dengan Contoh 3 (anchor 226, terakhir Ke-237).

Rekap riil ternyata menunjukkan pertemuan terakhir seharusnya **Ke-235**.

**Hitungan:**

```
selisih     = 235 − 237 = −2
anchor baru = 226 − 2   = 224
```

**Isi kolom:** `224`

**Hasil:**

| Kolom | Nilai |
|---|---|
| Anchor | 226 → 224 |
| Nomor global terakhir | 237 → 235 |
| Pertemuan berikutnya | Ke-236 |

---

### Contoh 5: Kelas sudah ganti level setelah memakai aplikasi

**Kondisi sekarang:**

- Anchor: **10**
- 2 pertemuan di level lama: global 11 dan 12 (tampil Ke-11, Ke-12)
- Kelas ganti level
- 2 pertemuan di level baru: global 13 dan 14 (tampil **Ke-1**, **Ke-2**)

Arsip kelas menunjukkan anchor yang benar adalah **15**.

**Isi kolom:** `15` (selisih +5)

**Hasil:**

| Pertemuan | Global sebelum → sesudah | Nomor tampil sebelum → sesudah |
|---|---|---|
| Level lama #1 | 11 → 16 | Ke-11 → Ke-16 |
| Level lama #2 | 12 → 17 | Ke-12 → Ke-17 |
| Level baru #1 | 13 → 18 | Ke-1 → **Ke-1** (tetap) |
| Level baru #2 | 14 → 19 | Ke-2 → **Ke-2** (tetap) |

- Nomor level baru **tidak berubah**, karena level baru memang mulai dari 1.
- Pertemuan berikutnya tampil **Ke-3** di level baru. Di jendela koreksi tertulis **Ke-20**, karena jendela itu memakai nomor global.

> Pada kelas yang pernah ganti level, angka "Nomor global terakhir" di jendela koreksi **tidak sama** dengan nomor yang tampil di kartu Pertemuan Terakhir. Jangan menghitung selisih dari nomor level baru. Hitung dari nomor global, atau langsung tentukan anchor yang benar dari arsip.

---

### Contoh 6: Kelas yang sejak awal di aplikasi sudah memulai level baru

Contohnya kelas hasil naik level sebelum ada pertemuan di aplikasi:

- Anchor: 12
- Level baru dimulai setelah pertemuan 12
- 2 pertemuan di aplikasi: global 13 dan 14, tampil **Ke-1** dan **Ke-2**

Koreksi anchor menjadi **17**:

- Nomor global berubah: 13 → 18, 14 → 19.
- Nomor tampil tetap **Ke-1** dan **Ke-2**.
- Pertemuan berikutnya tetap tampil **Ke-3**.

Artinya: bila yang salah hanya nomor tampil di level berjalan, koreksi anchor **tidak** mengubah nomor tampil tersebut.

---

## 7. Kesalahan yang sering terjadi

### Salah 1: Mengisi nomor terakhir yang benar langsung ke kolom anchor

Kondisi seperti Contoh 3 (anchor 226, terakhir Ke-237, seharusnya Ke-240). Admin langsung mengisi **240** ke kolom.

```
selisih = 240 − 226 = +14
```

Hasilnya, nomor global terakhir menjadi **251**, bukan 240.

**Cara mencegah:** sebelum klik Terapkan, pastikan "Nomor global terakhir → (baru)" sama dengan nomor benar dari rekap riil.

### Salah 2: Membaca teks "Pertemuan berikutnya akan menjadi …" di kotak biru

Setelah ada pertemuan di aplikasi, teks di kotak biru tetap menghitung `anchor + 1`. Contohnya tetap tertulis "227" padahal pertemuan terakhir sudah Ke-237. Teks ini **hanya berlaku sebelum ada pertemuan**. Untuk Mode B, lihat angka di jendela Koreksi.

### Salah 3: Koreksi saat guru sedang mengajar

Tombol tidak aktif. Minta guru menyelesaikan pertemuan, lalu muat ulang halaman.

### Salah 4: Alasan terlalu umum

"Salah nomor" tidak cukup untuk audit. Tulis sumbernya dan angka yang benar. Contoh: "Rekap Google Sheet admin lama: pertemuan terakhir sebelum aplikasi = 229."

---

## 8. Apa yang berubah dan tidak berubah

| Data | Berubah? | Keterangan |
|---|---|---|
| Anchor (pertemuan terakhir sebelum sistem) | Ya | Menjadi nilai baru |
| Nomor global semua pertemuan di aplikasi | Ya | Bergeser sebesar selisih |
| Nomor tampil (per level) | Sebagian | Ikut bergeser bila masih mengikuti anchor. Nomor di level yang dimulai dari 1 tetap. Lihat Contoh 5 dan 6. |
| Batas pergantian level di riwayat | Ya | Ikut bergeser |
| Titik awal santri yang bergabung di tengah | Ya | Ikut bergeser |
| Nomor pertemuan yang tercantum di tagihan | Ya | Disinkronkan dengan nomor baru |
| Tanggal, jam, materi, catatan pertemuan | Tidak | |
| Absensi santri | Tidak | |
| Tagihan: bulan ke-, nominal, status bayar | Tidak | Tidak ada tagihan yang dibuat atau dihapus |
| Guru, santri, jadwal kelas | Tidak | |

---

## 9. Pesan error dan artinya

| Pesan | Penyebab | Yang dilakukan |
|---|---|---|
| koreksi tidak dapat dilakukan saat ada pertemuan yang sedang berlangsung | Guru belum menutup pertemuan | Tunggu pertemuan selesai |
| kelas belum memiliki pertemuan di aplikasi; gunakan pengaturan pertemuan terakhir sebelum sistem | Belum ada pertemuan di aplikasi | Pakai Mode A (isi kolom, klik Simpan) |
| nomor pertemuan belum berubah | Anchor baru sama dengan anchor lama | Periksa hitungan selisih |
| koreksi menghasilkan nomor pertemuan tidak valid | Hasil koreksi membuat nomor pertemuan kurang dari 1 | Anchor baru minimal 0. Periksa angka. |
| alasan koreksi wajib diisi | Kolom alasan kosong | Isi alasan |
| alasan koreksi maksimal 500 karakter | Alasan terlalu panjang | Ringkas |
| nomor awal tidak dapat diubah setelah pertemuan tercatat | Mencoba Simpan Mode A padahal sudah ada pertemuan | Muat ulang halaman, pakai Koreksi nomor |

Tombol **Terapkan koreksi** tetap nonaktif bila anchor baru sama dengan anchor lama, angkanya bukan bilangan bulat ≥ 0, atau alasan kosong.

---

## 10. Setelah koreksi: cara memeriksa

1. Kartu **Pertemuan Terakhir** menampilkan nomor yang benar.
2. Bagian **Riwayat perubahan** memuat entri **"Koreksi nomor pertemuan"** berisi nilai lama → baru, alasan, dan nama admin.
3. Bila ada tagihan yang terhubung ke pertemuan kelas ini, nomor pertemuan di detail tagihan sudah mengikuti nomor baru.

### Membatalkan koreksi

Tidak ada tombol undo. Lakukan koreksi balik dengan mengembalikan anchor ke nilai sebelumnya, dan tulis alasan "Membatalkan koreksi tanggal … karena …". Kedua koreksi tetap tercatat di Riwayat perubahan.

---

## 11. Checklist sebelum klik "Terapkan koreksi"

- [ ] Nomor benar berasal dari sumber yang bisa ditunjukkan (rekap, absensi manual, konfirmasi guru)
- [ ] Tidak ada pertemuan yang sedang berlangsung
- [ ] Selisih dihitung dari **nomor global**, bukan nomor level baru (penting untuk kelas yang pernah ganti level)
- [ ] Kolom diisi **anchor baru**, bukan nomor terakhir
- [ ] "Nomor global terakhir → baru" di jendela koreksi sudah sesuai rekap riil
- [ ] Alasan menyebut sumber data dan angka yang benar
