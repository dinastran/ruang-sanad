# PRD — Fitur Tagihan Keuangan (SPP by Pertemuan)

**Status:** Draft v1
**Tanggal:** 2026-07-30
**Penulis:** Tim Ruang Sanad
**Modul terkait:** Keuangan, Santri, Kelas, Pertemuan, Absensi

---

## 1. Ringkasan

Sistem SPP di Ruang Sanad bersifat **per pertemuan** (bukan kalender bulanan tetap). Satu bulan pembelajaran = 1 blok pertemuan sesuai frekuensi kelas:

- **Reguler (1x/pekan)** → 4 pertemuan / bulan.
- **Intensif (2x/pekan)** → 8 pertemuan / bulan.

Bulan pertama sudah dibayar di muka saat pendaftaran. Tagihan berikutnya **muncul otomatis** setiap kelipatan blok pertemuan berikutnya. Fitur ini menambahkan: (a) generator tagihan otomatis berbasis pertemuan kelas, (b) halaman/menu keuangan untuk memantau & menandai pelunasan, (c) tombol Follow-Up WhatsApp per santri, dan (d) dashboard + filter data keuangan.

---

## 2. Tujuan & Sasaran

| Tujuan | Ukuran keberhasilan |
|---|---|
| Otomasi pembuatan tagihan SPP berbasis pertemuan | Tagihan muncul tanpa input manual saat pertemuan pemicu tercatat |
| Visibilitas kolektibilitas | Dashboard menampilkan sudah bayar vs belum bayar per periode |
| Mempercepat penagihan | Tombol WA FU membuka pesan siap kirim ke santri |
| Data keuangan dapat difilter & ditelusuri | Filter kelas/angkatan/guru/status/tanggal berfungsi |

**Non-tujuan (out of scope v1):** payment gateway/otomatis rekonsiliasi bank, faktur pajak, cicilan/partial payment, refund, laporan akuntansi (jurnal/GL), pengiriman WA otomatis via API.

---

## 3. Istilah

- **Blok** = 1 bulan pembelajaran = `n` pertemuan (`n` = 4 untuk 1x/pekan, `n` = 8 untuk 2x/pekan).
- **Pertemuan pemicu (trigger)** = pertemuan yang saat tercatat memunculkan tagihan baru.
- **Periode/Bulan ke-k** = blok pembelajaran ke-k (k = 1 dibayar saat daftar).
- **Kolektibilitas** = rasio tagihan lunas terhadap total tagihan pada periode.

---

## 4. Aturan Bisnis Tagihan (INTI)

### 4.1 Parameter
- `n` (pertemuan per blok) diturunkan dari `santri.frekuensi`:
  - `"1x/pekan"` / `"Reguler"` → `n = 4`
  - `"2x/pekan"` → `n = 8`
- `nominal` tagihan = `santri.nominal` (SPP 1 bulan / 1 blok).
- Dasar hitung: **absolut per kelas** — memakai `pertemuan.pertemuan_ke` apa adanya.
- **Semua pertemuan yang diadakan dihitung**, tanpa memandang status absensi santri (model langganan). Izin/sakit/alpa tetap menambah counter.

### 4.2 Titik pemicu tagihan
Bulan pertama (pertemuan 1..n) dibayar saat pendaftaran → **tidak** menghasilkan tagihan sistem.

Tagihan ke-`(k-1)` untuk **bulan ke-`k`** dibuat saat kelas mencapai:

```
pertemuan_ke = n × k      untuk k = 2, 3, 4, ...
```

Artinya tagihan muncul di pertemuan **kelipatan `n`, mulai dari `2n`**.

| Frekuensi | n | Bulan 1 (dibayar daftar) | Tagihan muncul di pertemuan ke- |
|---|---|---|---|
| 1x/pekan | 4 | pertemuan 1–4 | **8, 12, 16, 20, 24, …** |
| 2x/pekan | 8 | pertemuan 1–8 | **16, 24, 32, 40, …** |

> Contoh 1x/pekan: santri bayar saat daftar (bulan 1). Saat kelas menyelesaikan pertemuan ke-8, muncul tagihan **bulan ke-2**. Pertemuan ke-12 → **bulan ke-3**, dst.

### 4.3 Tanggal tagihan
`tagihan.tanggal_tagih` = `pertemuan.tanggal` dari pertemuan pemicu (tanggal tagihan berbasis pertemuan tersebut, sesuai permintaan). `jatuh_tempo` opsional = `tanggal_tagih + X hari` (default konfigurasi, mis. 7 hari) — dipakai untuk penandaan status "terlambat".

### 4.4 Nomor periode
`tagihan.bulan_ke = pertemuan_ke / n` (mis. pertemuan 8 & n=4 → bulan ke-2; pertemuan 16 & n=8 → bulan ke-2).

### 4.5 Idempotensi & anti-duplikat
- Kombinasi **unik** `(santri_id, bulan_ke)` — satu santri hanya punya satu tagihan per periode.
- Generator harus aman dijalankan ulang (re-run tidak menggandakan tagihan).

### 4.6 Penanganan santri gabung di tengah (edge case)
Karena dasar hitung "absolut per kelas", santri yang masuk saat kelas sudah berjalan berpotensi ter-generate tagihan periode lampau. Aturan:
- Simpan **anchor** per santri: `pertemuan_awal` = `pertemuan_ke` kelas saat santri mulai (default `0` bila ikut dari awal). Sumber: pertemuan pertama yang tercatat untuk santri, atau diisi manual saat penempatan kelas.
- Tagihan **hanya dibuat untuk periode ≥ periode saat santri bergabung**. Periode sebelum `pertemuan_awal` tidak ditagih (dianggap belum jadi tanggungan santri tersebut).
- Bila `pertemuan_awal = 0`, perilaku = persis tabel di 4.2.

### 4.7 Status tagihan
`belum_bayar` (default) → `lunas`. Status turunan **`terlambat`** dihitung saat `belum_bayar` dan `today > jatuh_tempo`. Status `batal` untuk pembatalan manual (mis. santri berhenti). Tidak ada partial/cicilan di v1.

### 4.8 Pelunasan
Ditandai **manual** oleh staf `keuangan`/`super_admin`:
- Set `status = lunas`, `tanggal_bayar`, `metode` (opsional: transfer/tunai), `catatan`, `dicatat_oleh`.
- Opsional sinkronisasi ke `santri.infaq_terakhir` (field existing) sebagai ringkasan pembayaran terakhir.

---

## 5. Model Data

Mengikuti pola existing: **goose** untuk migrasi (`migrations/`), **sqlc** untuk query (`queries/*.sql` → `app/queries/*.sql.go`).

### 5.1 Migrasi baru: `migrations/0020_create_tagihan_table.sql`

```sql
CREATE TABLE IF NOT EXISTS tagihan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    pertemuan_id INTEGER REFERENCES pertemuan(id) ON DELETE SET NULL,  -- pertemuan pemicu
    bulan_ke INTEGER NOT NULL,               -- periode: pertemuan_ke / n
    pertemuan_ke INTEGER NOT NULL,           -- pertemuan_ke pemicu (8,12,16,...)
    nominal INTEGER NOT NULL DEFAULT 0,      -- snapshot santri.nominal saat generate
    tanggal_tagih DATE NOT NULL,             -- = pertemuan.tanggal pemicu
    jatuh_tempo DATE,                        -- tanggal_tagih + X hari (opsional)
    status TEXT NOT NULL DEFAULT 'belum_bayar', -- belum_bayar | lunas | batal
    tanggal_bayar DATE,
    metode TEXT NOT NULL DEFAULT '',         -- transfer | tunai | ''
    catatan TEXT NOT NULL DEFAULT '',
    dicatat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    fu_terakhir DATETIME,                    -- kapan terakhir di-FU via WA
    fu_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(santri_id, bulan_ke)              -- anti-duplikat
);

CREATE INDEX idx_tagihan_santri ON tagihan(santri_id);
CREATE INDEX idx_tagihan_kelas ON tagihan(kelas_id);
CREATE INDEX idx_tagihan_status ON tagihan(status);
CREATE INDEX idx_tagihan_tanggal ON tagihan(tanggal_tagih);
```

### 5.2 Kolom tambahan pada `santri` (opsional, untuk edge case gabung di tengah)
```sql
ALTER TABLE santri ADD COLUMN pertemuan_awal INTEGER NOT NULL DEFAULT 0;
```

> Catatan: pembayaran bulan pertama saat pendaftaran **tidak** dibuatkan baris `tagihan` (bukan tanggungan tertunggak). Bila diinginkan sebagai riwayat, dapat dibuat baris tagihan `bulan_ke=1` berstatus `lunas` — **keputusan terbuka (lihat §12)**.

---

## 6. Query sqlc (garis besar) — `queries/tagihan.sql`

- `CreateTagihan` (`:execresult`) — INSERT dengan `INSERT OR IGNORE`/`ON CONFLICT(santri_id,bulan_ke) DO NOTHING` untuk idempotensi.
- `GetTagihanByID` (`:one`).
- `ListTagihan` (`:many`) — join `santri` + `kelas` + `guru`, filter dinamis (status, kelas_id, angkatan, guru_id, frekuensi, rentang tanggal, gender, level, keyword nama).
- `MarkTagihanLunas` (`:exec`) — set status/tanggal_bayar/metode/catatan/dicatat_oleh.
- `BatalkanTagihan` (`:exec`).
- `TouchFollowUp` (`:exec`) — set `fu_terakhir = now`, `fu_count = fu_count + 1`.
- **Ringkasan dashboard:**
  - `RingkasanTagihanPeriode` (`:one`) — total tagihan, total lunas, total belum, jumlah rupiah masing-masing, dalam rentang tanggal.
  - `ListTagihanBelumBayar` / `ListTagihanLunas` (`:many`).
  - `RekapPerKelas` (`:many`) — agregasi per kelas untuk breakdown.

---

## 7. Alur Sistem

### 7.1 Generate tagihan (otomatis)
Dipicu **saat pertemuan tercatat** (setelah `CreatePertemuan` / saat status pertemuan → `selesai`):

```
1. Ambil pertemuan (kelas_id, pertemuan_ke, tanggal).
2. Ambil semua santri AKTIF di kelas tsb.
3. Untuk tiap santri:
   n = frekuensiKePertemuan(santri.frekuensi)      // 4 atau 8
   if pertemuan_ke % n == 0 AND pertemuan_ke >= 2n:
       bulan_ke = pertemuan_ke / n
       if bulan_ke > periodeGabung(santri):        // §4.6
           CreateTagihan(santri, bulan_ke, pertemuan_ke,
                         nominal=santri.nominal,
                         tanggal_tagih=pertemuan.tanggal,
                         jatuh_tempo=+X hari)        // idempotent
```

- Hook di `app/services/pertemuan.go` (service pertemuan) memanggil `TagihanService.GenerateForPertemuan(pertemuanID)`.
- **Backfill:** sediakan command/endpoint admin untuk men-generate tagihan atas pertemuan yang sudah tercatat sebelum fitur aktif (mis. `cmd/seed` atau tombol "Sinkronkan Tagihan" di menu keuangan).

### 7.2 Tandai lunas
Staf keuangan buka daftar tagihan → klik "Tandai Lunas" → isi tanggal bayar, metode, catatan → simpan (`MarkTagihanLunas`).

### 7.3 Follow-Up WhatsApp (§9).

---

## 8. Antarmuka (Frontend — Inertia + Svelte)

Halaman baru di `frontend/src/pages/keuangan/` dengan route di `routes/web.go`, dijaga middleware role `keuangan`/`super_admin`.

### 8.1 Dashboard Keuangan (`/keuangan` atau `/keuangan/dashboard`)
Kartu ringkasan periode berjalan:
- Total tagihan (jumlah & Rp), **Sudah Bayar** (jumlah & Rp), **Belum Bayar** (jumlah & Rp), **Terlambat**, **% Kolektibilitas**.
- Dua daftar/tab: **"Sudah Lanjut Bayar"** dan **"Belum Bayar"** untuk bulan berjalan.
- Grafik sederhana (opsional): tren kolektibilitas per bulan, breakdown per kelas.

### 8.2 Daftar Tagihan (`/keuangan/tagihan`)
Tabel kolom: Nama santri, ID mahasantri, Kelas, Guru, Frekuensi, Bulan ke-, Pertemuan ke-, Nominal, Tanggal tagih, Jatuh tempo, Status (badge), Aksi.

**Aksi per baris:** Tandai Lunas • Detail • **FU WhatsApp** • Batalkan.

**Filter (query params):**
- Status: semua / belum_bayar / lunas / terlambat / batal
- Kelas, Angkatan, Guru, Frekuensi, Level, Jenis kelamin
- Rentang `tanggal_tagih` (dari–sampai)
- Bulan ke-
- Pencarian nama / ID mahasantri

**Bulk action (opsional v1.1):** tandai lunas massal, export CSV.

### 8.3 Detail Tagihan
Riwayat status, info santri + kontak WA, log follow-up (`fu_terakhir`, `fu_count`), tombol aksi.

---

## 9. Follow-Up WhatsApp (wa.me)

Mekanisme: **link `wa.me`** — tanpa API/biaya.

- Format nomor: normalisasi `santri.no_wa` ke internasional tanpa `+`/spasi (mis. `08xx…` → `628xx…`).
- URL: `https://wa.me/<nomor>?text=<pesan_url_encoded>`.
- Klik tombol → buka WA (web/app) dengan pesan template terisi → admin tinggal kirim.
- Setelah membuka, panggil `TouchFollowUp(tagihan_id)` (update `fu_terakhir`, `fu_count`) untuk pelacakan.

**Template pesan (default, dapat dikonfigurasi):**
```
Assalamu'alaikum {nama}, 🙏
Kami informasikan tagihan SPP Ruang Sanad bulan ke-{bulan_ke}
sebesar Rp{nominal} telah terbit (tanggal {tanggal_tagih}).
Mohon konfirmasi pembayarannya. Jazaakumullah khairan. 🌿
```
Placeholder: `{nama}`, `{bulan_ke}`, `{nominal}` (format ribuan), `{tanggal_tagih}`, `{kelas}`.

**Validasi:** jika `no_wa` kosong/tidak valid → tombol dinonaktifkan dengan tooltip "Nomor WA belum diisi".

---

## 10. Peran & Hak Akses

| Aksi | keuangan | super_admin | admin | lainnya |
|---|---|---|---|---|
| Lihat dashboard & daftar tagihan | ✅ | ✅ | ✅ (read) | ❌ |
| Tandai lunas / batal | ✅ | ✅ | ❌ | ❌ |
| Follow-Up WA | ✅ | ✅ | ✅ | ❌ |
| Sinkronkan/backfill tagihan | ✅ | ✅ | ❌ | ❌ |

Generate otomatis berjalan di level sistem (tidak butuh peran khusus).

---

## 11. Edge Cases & Aturan

1. **Santri non-aktif/berhenti** → tidak di-generate tagihan baru; tagihan tertunggak boleh di-`batal` manual.
2. **Reschedule/badal pertemuan** → tidak menambah `pertemuan_ke` baru (nomor tetap), jadi tidak menggandakan tagihan.
3. **`nominal` berubah** setelah tagihan terbit → tagihan lama pakai snapshot (`tagihan.nominal`), tidak berubah surut.
4. **`frekuensi` berubah** di tengah → hanya memengaruhi periode berikutnya (mengubah `n` untuk perhitungan ke depan).
5. **`no_wa` kosong** → FU dinonaktifkan.
6. **Re-run generator** → idempotent via `UNIQUE(santri_id, bulan_ke)`.
7. **Santri gabung di tengah** → §4.6 (anchor `pertemuan_awal`).
8. **Pertemuan dihapus** → kebijakan: tagihan terkait yang belum lunas ikut dibatalkan (butuh konfirmasi, lihat §12).

---

## 12. Pertanyaan Terbuka / Keputusan Menunggu

1. **Bulan ke-1 sebagai riwayat?** Buat baris `tagihan bulan_ke=1` berstatus `lunas` (untuk histori) atau tidak sama sekali? (Default PRD: tidak dibuat.)
2. **Jatuh tempo (X hari)** setelah tanggal tagih — berapa hari default? (Usulan: 7.)
3. **Metode pembayaran** — perlu daftar metode tetap (transfer/tunai/lainnya) atau bebas teks?
4. **Penghapusan pertemuan** — apakah otomatis membatalkan tagihan terkait?
5. **Nominal berbeda per santri sudah benar?** (Saat ini `santri.nominal` per-individu — dipakai apa adanya.)

---

## 13. Kriteria Penerimaan (Acceptance Criteria)

- [ ] Saat pertemuan ke-8 (1x/pekan) tercatat, setiap santri aktif di kelas mendapat tagihan `bulan_ke=2`, `tanggal_tagih = tanggal pertemuan`, `nominal = santri.nominal`, status `belum_bayar`.
- [ ] Saat pertemuan ke-16 (2x/pekan) tercatat, tagihan `bulan_ke=2` terbentuk (bukan di pertemuan 8/12).
- [ ] Menjalankan generator dua kali tidak menggandakan tagihan.
- [ ] Staf keuangan dapat menandai lunas → status berubah, tercatat tanggal/petugas.
- [ ] Dashboard menampilkan angka sudah bayar vs belum bayar bulan berjalan yang konsisten dengan data.
- [ ] Filter (kelas/angkatan/guru/status/tanggal) mengubah hasil daftar dengan benar.
- [ ] Tombol FU membuka `wa.me` berisi template terisi; `fu_count` bertambah.
- [ ] Santri tanpa `no_wa` → tombol FU nonaktif.

---

## 14. Rencana Implementasi (bertahap)

1. **DB & Query**
   - Migrasi `0020_create_tagihan_table.sql` (+ kolom `santri.pertemuan_awal` bila dipakai).
   - `queries/tagihan.sql` → generate `app/queries/tagihan.sql.go` via sqlc.
2. **Service**
   - `app/services/tagihan.go`: `frekuensiKePertemuan`, `GenerateForPertemuan`, `MarkLunas`, `Batal`, `RingkasanPeriode`, `TouchFollowUp`, builder nomor WA + template.
   - Hook di `app/services/pertemuan.go` memanggil generate setelah pertemuan tercatat.
   - Backfill command (mis. di `cmd/seed` atau handler "Sinkronkan Tagihan").
3. **Handler & Route**
   - `app/handlers/keuangan.go`: dashboard, list+filter, mark lunas, batal, FU, sync.
   - Registrasi route di `routes/web.go` + middleware role `keuangan`/`super_admin`.
   - DTO di `app/models/` (mis. `tagihan.go`).
4. **Frontend**
   - `frontend/src/pages/keuangan/Dashboard.svelte`, `Tagihan.svelte`, komponen filter & badge status.
   - Tipe di `frontend/src/lib/types.ts`; menu di `AppLayout.svelte` untuk role keuangan.
5. **Uji**
   - Unit test service (perhitungan trigger 1x & 2x, idempotensi, edge gabung tengah) — pola seperti `app/services/tsi_test.go`.
   - Uji manual alur end-to-end sesuai §13.

---

## 15. Lampiran — Referensi Skema Existing

- `santri`: `nominal`, `frekuensi` (`"1x/pekan"`/`"2x/pekan"`), `no_wa`, `tanggal_daftar`, `mulai_belajar`, `kelas_id`, `status`, `infaq_terakhir`.
- `kelas`: `frekuensi`, `guru_id`, `angkatan`, `level`, `nama_kelas`.
- `pertemuan`: `kelas_id`, `pertemuan_ke`, `tanggal`, `status` (`selesai`), `UNIQUE(kelas_id, pertemuan_ke)`.
- `absensi`: per santri per pertemuan (tidak dipakai untuk perhitungan tagihan di v1).
- Role existing: `keuangan`, `super_admin`, `admin`, `admin_kelas`, `cs`, `guru`, `koordinator_guru`.
```
