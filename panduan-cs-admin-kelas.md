# Panduan Pengelolaan Data Mahasantri

Dokumen ini menjelaskan pembagian tugas antara **CS** dan **Admin Kelas**, khususnya mengenai **Angkatan Pendaftaran**, **Angkatan Kelas**, ID Mahasantri, dan penempatan kelas.

## 1. Dua Jenis Angkatan

Sekarang sistem menggunakan dua data angkatan yang berbeda.

### Angkatan Pendaftaran

Angkatan Pendaftaran adalah angkatan ketika Mahasantri pertama kali mendaftar.

Data ini:

- Diisi oleh CS.
- Menjadi bagian dari ID Mahasantri.
- Menjadi identitas pendaftaran permanen.
- Tidak berubah walaupun Mahasantri pindah kelas.
- Digunakan untuk laporan berdasarkan periode pendaftaran.

Contoh:

> Mahasantri mendaftar melalui program Angkatan AKA38, maka Angkatan Pendaftarannya tetap **AKA38**.

### Angkatan Kelas

Angkatan Kelas adalah angkatan tempat Mahasantri benar-benar ditempatkan untuk belajar.

Data ini:

- Ditentukan oleh Admin Kelas.
- Boleh berbeda dari Angkatan Pendaftaran.
- Digunakan untuk pengelompokan dan penempatan kelas.
- Dapat berubah apabila Mahasantri dipindahkan ke kelas lain.

Contoh:

> Mahasantri mendaftar pada AKA38, tetapi baru mulai belajar bersama kelas AKA39.

Maka datanya menjadi:

| Data | Nilai |
|---|---|
| Angkatan Pendaftaran | AKA38 |
| Angkatan Kelas | AKA39 |

Hal ini valid dan bukan kesalahan data.

## 2. Alur Kerja Secara Umum

1. CS memasukkan data pendaftaran Mahasantri.
2. Sistem menerbitkan ID Mahasantri.
3. Angkatan Kelas masih kosong.
4. Data masuk ke halaman **Perlu Dilengkapi**.
5. Admin Kelas melengkapi data penempatan.
6. Sistem mencari atau membuat kelas yang sesuai.
7. Mahasantri otomatis masuk ke kelas.
8. Jika diperlukan, Admin Kelas dapat memindahkan Mahasantri ke kelas lain.

## Panduan untuk CS

### 3. Tanggung Jawab CS

CS bertanggung jawab atas data pendaftaran dan identitas dasar Mahasantri.

Data yang diisi CS meliputi:

- Kode Kelas atau program yang dipilih.
- Nama lengkap.
- Jenis kelamin.
- Nominal.
- Tanggal pendaftaran.
- Angkatan Pendaftaran.
- Usia.
- Domisili.
- Nomor WhatsApp.
- Email.

Pastikan **Tanggal Pendaftaran** dan **Angkatan Pendaftaran** benar sebelum data disimpan.

### 4. Penerbitan ID Mahasantri

Setelah data baru berhasil disimpan, sistem otomatis menerbitkan ID Mahasantri.

Format ID:

```text
MHS.[ANGKATAN PENDAFTARAN].[NOMOR URUT].[BULAN DAN TAHUN]
```

Contoh:

```text
MHS.AKA38.0317.072026
```

Artinya:

| Bagian | Arti |
|---|---|
| MHS | Identitas Mahasantri |
| AKA38 | Angkatan Pendaftaran |
| 0317 | Nomor urut Mahasantri |
| 072026 | Mendaftar pada Juli 2026 |

ID tersebut bukan nomor kelas. ID merupakan identitas pendaftaran Mahasantri.

### 5. Data yang Dikunci Setelah ID Terbit

Setelah ID Mahasantri terbit, data berikut dikunci:

- ID Mahasantri.
- Angkatan Pendaftaran.
- Tanggal Pendaftaran.

CS tidak dapat mengubah ketiga data tersebut dari form biasa.

Data tersebut dikunci agar:

- ID Mahasantri tidak berubah-ubah.
- Histori pendaftaran tetap konsisten.
- Laporan tidak berubah karena perpindahan kelas.
- Tidak terjadi ketidaksesuaian antara ID dan data pendaftaran.

### 6. Data yang Masih Bisa Diperbarui CS

Setelah ID terbit, CS masih dapat memperbarui:

- Kode Kelas atau program.
- Nama lengkap.
- Jenis kelamin.
- Nominal.
- Usia.
- Domisili.
- Nomor WhatsApp.
- Email.

#### Perhatian tentang Kode Kelas

Kode Kelas menentukan tipe dan frekuensi belajar, misalnya:

- Reguler.
- Private.
- Semi Private.
- 1x per pekan.
- 2x per pekan.
- Paket 4x atau 16x pertemuan.

Perubahan Kode Kelas pada Mahasantri yang sudah ditempatkan dapat memengaruhi pengelompokan kelasnya.

Karena itu, jika Mahasantri sudah masuk kelas, perubahan Kode Kelas sebaiknya dikoordinasikan terlebih dahulu dengan Admin Kelas.

### 7. Jika Angkatan atau Tanggal Pendaftaran Salah

Jika kesalahan ditemukan setelah ID Mahasantri terbit:

1. Jangan mengganti Angkatan Kelas untuk memperbaiki kesalahan tersebut.
2. Jangan membuat data Mahasantri baru sebagai pengganti.
3. Laporkan kepada Super Admin.
4. Sertakan nama Mahasantri, ID Mahasantri, Angkatan Pendaftaran yang benar, Tanggal Pendaftaran yang benar, dan alasan koreksi.

Hanya **Super Admin** yang dapat melakukan koreksi identitas pendaftaran.

Saat dikoreksi, sistem akan menerbitkan ulang ID Mahasantri agar sesuai dengan Angkatan dan Tanggal Pendaftaran yang benar.

### 8. Yang Tidak Perlu Dilakukan CS

CS tidak perlu:

- Menentukan Angkatan Kelas.
- Menentukan level.
- Menentukan jadwal belajar.
- Menentukan guru.
- Memasukkan Mahasantri ke kelas tertentu.
- Mengubah ID Mahasantri secara manual.
- Menghapus Mahasantri secara permanen.

Penempatan kelas menjadi tanggung jawab Admin Kelas.

## Panduan untuk Admin Kelas

### 9. Tanggung Jawab Admin Kelas

Admin Kelas bertanggung jawab atas proses setelah data pendaftaran dibuat oleh CS.

Admin Kelas dapat melihat data pendaftaran Mahasantri, tetapi tidak dapat mengubah:

- ID Mahasantri.
- Angkatan Pendaftaran.
- Tanggal Pendaftaran.
- Nama dan data dasar milik CS.

Data tersebut hanya ditampilkan sebagai referensi.

### 10. Halaman Perlu Dilengkapi

Mahasantri baru akan muncul di halaman **Perlu Dilengkapi** karena Angkatan Kelas dan data penempatannya belum diisi.

Admin Kelas perlu melengkapi data seperti:

- FU.
- Tanggal VN.
- Hasil VN.
- Rekaman dan keterangan Voice Note.
- Status atau nama grup.
- Tanggal mulai belajar.
- Jumlah.
- Angkatan Kelas.
- Level.
- Jadwal.
- Guru.

#### Data utama untuk penempatan kelas

Agar Mahasantri dapat ditempatkan ke kelas, data berikut harus tersedia:

- Nama.
- Jenis kelamin.
- Angkatan Kelas.
- Level.
- Jadwal.

Nama dan jenis kelamin sudah berasal dari data CS. Admin Kelas terutama harus memastikan bahwa **Angkatan Kelas, Level, dan Jadwal** sudah benar.

### 11. Menentukan Angkatan Kelas

Angkatan Kelas tidak harus sama dengan Angkatan Pendaftaran.

Contoh:

| Data | Nilai |
|---|---|
| Angkatan Pendaftaran | AKA38 |
| Angkatan Kelas | AKA39 |
| Level | Dasar |
| Jadwal | Senin 19.00 |
| Guru | Ustadz Ahmad |

Dalam kasus tersebut:

- ID Mahasantri tetap menggunakan AKA38.
- Mahasantri ditempatkan di kelompok kelas AKA39.
- Tidak perlu mengubah Angkatan Pendaftaran.

#### Data lama

Pada beberapa data lama mungkin muncul pilihan dengan keterangan:

```text
(data lama)
```

Nilai tersebut masih boleh dipertahankan.

Namun, apabila Angkatan Kelas ingin diubah, pilih kode angkatan yang tersedia dalam pilihan sistem.

### 12. Cara Sistem Menentukan Kelas

Setelah Admin Kelas menyimpan data, sistem mengelompokkan Mahasantri berdasarkan:

- Kode Kelas atau program dari CS.
- Jenis kelamin.
- Level.
- Frekuensi.
- Jadwal.
- Angkatan Kelas.

Sistem kemudian melakukan salah satu dari dua hal:

1. Memasukkan Mahasantri ke kelas aktif yang sesuai dan masih memiliki kapasitas.
2. Membuat kelas baru jika belum ada kelas yang sesuai atau kelas sebelumnya sudah penuh.

Kapasitas awal kelas baru adalah **15 Mahasantri**.

Jika kapasitas kelas diubah, sistem akan menggunakan kapasitas terbaru yang tercatat pada kelas tersebut.

### 13. Pemilihan Guru

Guru dipilih pada data kelas.

Perlu diperhatikan bahwa guru merupakan bagian dari pengelolaan kelas, bukan hanya data pribadi satu Mahasantri.

Jika Mahasantri ditempatkan ke kelas yang sudah ada, perubahan guru dapat memengaruhi guru yang tercatat pada kelas tersebut.

Sebelum mengganti guru, pastikan pilihan tersebut memang berlaku untuk kelas terkait.

### 14. Jadwal Lebih dari Satu Sesi

Untuk program dengan beberapa pertemuan dalam satu pekan, Admin Kelas dapat menambahkan lebih dari satu jadwal.

Contoh:

```text
Senin 19.00
Kamis 19.00
```

Pastikan seluruh jadwal dipilih dengan benar karena kombinasi jadwal digunakan oleh sistem untuk menentukan kelompok kelas.

### 15. Memindahkan Mahasantri ke Kelas Lain

Admin Kelas dapat memindahkan Mahasantri melalui detail kelas atau detail Mahasantri.

Saat pemindahan dilakukan, sistem akan memeriksa:

- Kelas tujuan masih aktif.
- Jenis kelamin sesuai.
- Kelas tujuan belum penuh.
- Kelas tujuan tersedia.

Jika valid, sistem akan menyesuaikan data penempatan Mahasantri berdasarkan kelas tujuan:

- Angkatan Kelas.
- Kode Kelas.
- Level.
- Jadwal.
- Tipe.
- Frekuensi.

Data berikut tidak berubah:

- ID Mahasantri.
- Angkatan Pendaftaran.
- Tanggal Pendaftaran.

Contoh:

| Data | Sebelum Pindah | Setelah Pindah |
|---|---|---|
| ID Mahasantri | MHS.AKA38.0317.072026 | MHS.AKA38.0317.072026 |
| Angkatan Pendaftaran | AKA38 | AKA38 |
| Angkatan Kelas | AKA39 | AKA40 |

### 16. Kondisi Pemindahan yang Ditolak

Sistem akan menolak pemindahan jika:

- Kelas tujuan tidak aktif.
- Jenis kelamin tidak sesuai.
- Kelas tujuan sudah penuh.
- Kelas tujuan tidak ditemukan.

Jika pemindahan ditolak, data Mahasantri tetap berada di kelas sebelumnya.

### 17. Jika Kelas Dihapus

Jika sebuah kelas dihapus:

- Mahasantri dalam kelas tersebut akan dikeluarkan dari penempatan kelas.
- Angkatan Kelas Mahasantri dikosongkan.
- Mahasantri kembali masuk ke status **Perlu Dilengkapi**.
- Admin Kelas perlu menentukan penempatan baru.

Menghapus kelas sebaiknya dilakukan dengan hati-hati.

Pastikan seluruh Mahasantri sudah dipindahkan jika kelas tersebut masih memiliki anggota aktif.

Histori tagihan yang sudah diterbitkan tetap menyimpan Angkatan Kelas pada saat tagihan dibuat.

## 18. Pencarian dan Filter

Pada daftar Mahasantri tersedia dua filter berbeda.

### Filter Angkatan Pendaftaran

Gunakan filter ini jika ingin mencari Mahasantri berdasarkan periode saat pertama kali mendaftar.

### Filter Angkatan Kelas

Gunakan filter ini jika ingin mencari Mahasantri berdasarkan penempatan kelas saat ini.

Jangan tertukar karena hasilnya dapat berbeda.

## 19. Ringkasan Pembagian Tugas

| Aktivitas | CS | Admin Kelas | Super Admin |
|---|:---:|:---:|:---:|
| Membuat data Mahasantri | Ya | Tidak | Ya |
| Mengisi Angkatan Pendaftaran | Ya, saat pendaftaran | Tidak | Ya |
| Mengisi Tanggal Pendaftaran | Ya, saat pendaftaran | Tidak | Ya |
| Mengubah identitas setelah ID terbit | Tidak | Tidak | Ya |
| Mengedit data dasar Mahasantri | Ya | Hanya lihat | Ya |
| Mengisi Angkatan Kelas | Tidak | Ya | Ya |
| Mengisi level dan jadwal | Tidak | Ya | Ya |
| Menentukan guru | Tidak | Ya | Ya |
| Memindahkan kelas | Tidak | Ya | Ya |
| Menghapus Mahasantri permanen | Tidak | Tidak | Ya |

## 20. Prinsip Utama

### Untuk CS

> Pastikan Angkatan Pendaftaran dan Tanggal Pendaftaran benar sebelum menyimpan data karena keduanya menjadi bagian dari ID Mahasantri.

### Untuk Admin Kelas

> Gunakan Angkatan Kelas untuk menentukan penempatan belajar. Jangan mengubah atau meminta perubahan Angkatan Pendaftaran hanya karena Mahasantri belajar bersama angkatan yang berbeda.

### Untuk Semua Tim

> Angkatan Pendaftaran adalah identitas awal. Angkatan Kelas adalah posisi belajar saat ini. Keduanya boleh berbeda.
