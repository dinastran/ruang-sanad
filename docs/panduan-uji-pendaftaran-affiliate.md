# Panduan Uji — Link Pendaftaran, Pembayaran, dan Affiliate

Untuk diuji di production (`dashboard.ruangsanad.com`) dengan akun **super admin**.
Gunakan nomor WA dan nama uji (mis. "Uji Daftar 1") supaya mudah dibersihkan.

## Status pembayaran online (Xendit)

**Belum aktif.** Tanpa akun Xendit, variabel `XENDIT_SECRET_KEY` dan
`XENDIT_CALLBACK_TOKEN` kosong, sehingga aplikasi otomatis mematikan
pembayaran online:

- Halaman bayar hanya menampilkan **transfer manual + unggah bukti**.
- Tombol "Bayar sekarang (QRIS/e-wallet/VA)" tidak muncul.
- Semua fitur lain (link, form, bukti manual, verifikasi, affiliate) tetap jalan.

Untuk mengaktifkan nanti: daftar akun Xendit atas nama yayasan, buat Secret
API key (mulai dari mode **test**), daftarkan webhook Payment Session ke
`https://dashboard.ruangsanad.com/webhooks/xendit`, isi `.env`, restart.
Langkah lengkap: `PANDUAN-DEPLOY.md` → "Pembayaran online (Xendit)".

## 0. Persiapan (sekali)

1. Pastikan versi terbaru sudah dideploy dan migrasi sampai `0137` sudah jalan.
2. **Produk CRM** → buat produk "Tahsin Reguler" (kategori *Program*, harga
   mis. 150000) → di kartu produk, **Tambah varian**: "1x / pekan" (paket `1x`)
   dan "2x / pekan" (paket `2x`) dengan harga masing-masing.
3. **Leads → Program → Tahsin Reguler → Ubah** → pilih produk "Tahsin
   Reguler" di "Produk harga".
4. **Leads → Sumber** → pastikan ada sumber untuk link ini (mis. "Link Pendaftaran Bio IG").
5. **Leads → Link Pendaftaran → Pengaturan** → isi nomor WA CS, info
   rekening (mis. `BSI 123456 a.n. Yayasan …`), simpan.

Hasil yang diharapkan: tidak ada pesan error; pengaturan tersimpan.

## 1. Buat link pendaftaran

**Leads → Link Pendaftaran → Link baru**

| Uji | Isian | Hasil yang diharapkan |
|---|---|---|
| 1a | Judul "Daftar Tahsin Uji", slug terisi otomatis, program Tahsin Reguler, sumber dipilih, mode **Short form**, setelah submit **Halaman bayar**, Harga khusus kosong | Link tampil di tabel, harga "Rp…–Rp…" sesuai varian |
| 1b | Sama, tapi isi **Harga khusus** 99000 (slug lain) | Tersimpan, harga Rp99.000 (bug lama: tombol diam — sudah diperbaiki) |
| 1c | Tanpa memilih sumber lead | Pesan merah di dalam form "Pilih sumber lead." |
| 1d | Program yang belum terhubung ke produk | Opsi abu-abu / pesan merah di form |
| 1e | Mode **WA langsung** (slug lain) | Tersimpan; pilihan "Setelah submit" nonaktif |

Pratinjau di kanan (layar lebar) berubah sesuai mode dan pertanyaan tambahan.

## 2. Isi form sebagai pendaftar (pakai HP, jendela penyamaran)

Buka `https://dashboard.ruangsanad.com/daftar/<slug 1a>`.

1. Coba langsung tekan kirim < 3 detik setelah halaman terbuka → pesan
   "terlalu cepat" (anti-spam). Tunggu sebentar lalu kirim lagi.
2. Isi nama, nomor WA uji, pilih **Akhwat/Ikhwan** → daftar jadwal muncul
   sesuai gender; jadwal penuh tampil "Penuh" dan tidak bisa dipilih.
3. Pilih paket (varian) → Kirim → pindah ke **halaman bayar** dengan nominal
   sesuai varian.

Cek di dashboard: **Leads** → lead baru dengan tahap **Info bayar dikirim**,
nominal sesuai, panel "Pendaftaran & pembayaran" berisi riwayat isi form.

4. Isi form yang sama sekali lagi dengan format nomor berbeda (`+62 812…`) →
   tetap **satu lead** (tidak dobel), riwayat isi form bertambah.

## 3. Bayar manual

Di halaman bayar (HP):

1. Unggah file bukan gambar/PDF atau > 2 MB → ditolak dengan pesan.
2. Unggah foto bukti → status "sedang diperiksa".

Dashboard:
- Lead pindah ke **Bukti transfer masuk**; user role keuangan mendapat notifikasi.
- Panel pembayaran → **Lihat bukti** membuka file.
- **Tolak bukti** (isi alasan) → halaman bayar pendaftar menampilkan alasan dan
  bisa unggah ulang; lead kembali ke "Info bayar dikirim".
- Unggah ulang → **Verifikasi pembayaran** (tombol lama) → lead
  **Terverifikasi**, pembayaran berstatus Lunas, halaman bayar pendaftar
  menampilkan "Pembayaran sudah terverifikasi".

## 4. Mode WA langsung dan link nonaktif

1. Buka `/daftar/<slug 1e>` → langsung terbuka WhatsApp CS dengan pesan terisi.
2. Nonaktifkan link 1a → buka lagi → halaman "Link pendaftaran ini sudah
   tidak aktif" + tombol Chat CS.
3. Statistik di tabel Link Pendaftaran: klik/daftar bertambah (klik dihitung
   sekali per perangkat per hari).

## 5. Affiliate (super admin saja untuk sementara)

Menu affiliate saat ini hanya tampil untuk super admin (guru & keuangan belum).

1. **Leads → Affiliate → Aturan komisi** → buat aturan default, mis. 10% sekali.
2. **Pengaturan** → masa berlaku 30/60/90 hari, masa tahan (mis. 0 hari agar
   cepat terlihat saat uji), link tujuan default = link 1a.
3. Daftarkan affiliate eksternal uji di `/affiliate/daftar` (email yang belum
   punya akun) → muncul di "Menunggu persetujuan" → **Setujui** → salin link
   aktivasi → buka, buat password → login dengan email tsb → masuk halaman
   **Affiliate Saya** (menu lain tidak bisa dibuka).
4. Salin link affiliate (`/r/KODE`) → buka di HP (jendela penyamaran baru) →
   isi form → bayar manual → verifikasi di dashboard.
5. **Leads → Affiliate → Hitung ulang komisi** → komisi muncul di Buku
   komisi (status tertahan → siap cair setelah masa tahan).
6. Detail lead → panel **Affiliate** menunjukkan affiliate dan sumbernya.
7. Tab **Buku komisi** → centang komisi siap cair → **Tandai sudah ditransfer**.
8. Uji guru (opsional): guru yang punya akun login otomatis punya affiliate
   (diperbarui tiap 10 menit). Komisinya masuk ujroh sebagai baris
   "Komisi Affiliate" dan tercatat dibayar saat bulan ujroh **dikunci**.

## 6. Bersihkan data uji

Tandai lead uji sebagai hilang / hapus (super admin), nonaktifkan link dan
affiliate uji. Komisi uji bisa dibatalkan di Buku komisi.

## Belum termasuk / keterbatasan

- **Pembayaran online Xendit**: kode siap, belum aktif (butuh akun & key).
- **Tagihan SPP bulanan via Xendit** (proyek 2): belum dibuat; SPP tetap manual.
- **Pencairan otomatis ke rekening affiliate**: belum; keuangan transfer manual.
- Menu affiliate untuk guru & keuangan: disembunyikan sementara (halaman
  masih bisa dibuka lewat URL).
- Pesan WA otomatis ke pendaftar/affiliate tidak dikirim sistem; tautan dibagikan
  manual (tombol WhatsApp).
- Lead bisa dobel bila orang yang sama chat dari akun WA berformat `@lid`
  setelah mengisi form.
- Affiliate: kode referal tetap diterima walau kolomnya disembunyikan;
  rujukan oleh guru ke dirinya sendiri tidak terdeteksi; mengganti kode
  affiliate memutus link lama; daftar rujukan/buku komisi dipotong 200/300 baris.
