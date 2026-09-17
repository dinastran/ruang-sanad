# TODO — Eksekusi Fitur Tagihan Keuangan

Referensi: [`fiturtagihankeungan.md`](./fiturtagihankeungan.md)
Status legenda: `[ ]` belum · `[~]` proses · `[x]` selesai

---

## Fase 0 — Keputusan & Persiapan

- [x] Putuskan 5 pertanyaan terbuka (PRD §12):
  - [x] Bulan ke-1 dibuat sebagai baris `lunas` (histori) atau tidak? (default: tidak)
  - [x] Jatuh tempo = berapa hari setelah `tanggal_tagih`? (usul: 7)
  - [x] Metode bayar: pilihan tetap (transfer/tunai) atau bebas teks? teks saja.
  - [x] Pertemuan dihapus → tagihan belum lunas ikut dibatalkan? iya
  - [x] Konfirmasi nominal diambil dari `santri.nominal` per individu? iya
- [x] Konfirmasi teks template pesan WhatsApp final. iya bisa di custom+buatkan beberapa default nya
- [x] Konfirmasi label/menu keuangan di sidebar (`AppLayout.svelte`). iya

---

## Fase 1 — Database & Migrasi

- [x] Buat `migrations/0020_create_tagihan_table.sql` (tabel `tagihan` + index) — PRD §5.1
  - [x] Kolom lengkap: santri_id, kelas_id, pertemuan_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, jatuh_tempo, status, tanggal_bayar, metode, catatan, dicatat_oleh, fu_terakhir, fu_count, timestamps
  - [x] `UNIQUE(santri_id, bulan_ke)` untuk idempotensi
  - [x] Index: santri, kelas, status, tanggal
  - [x] Blok `+goose Down` (drop table + index)
- [x] (Opsional/edge case) `ALTER TABLE santri ADD COLUMN pertemuan_awal` — PRD §5.2
- [x] Jalankan migrasi & verifikasi skema terbentuk

---

## Fase 2 — Query (sqlc)

- [~] Buat `queries/tagihan.sql` — PRD §6
  - [x] `CreateTagihan` (`:execresult`, idempotent `ON CONFLICT(santri_id,bulan_ke) DO NOTHING`)
  - [x] `GetTagihanByID` (`:one`)
  - [x] `ListTagihan` (`:many`, join santri+kelas+guru, filter dinamis)
  - [x] `MarkTagihanLunas` (`:exec`)
  - [x] `BatalkanTagihan` (`:exec`)
  - [x] `TouchFollowUp` (`:exec`)
  - [x] `RingkasanTagihanPeriode` (`:one`)
  - [x] `ListTagihanBelumBayar` / `ListTagihanLunas` (`:many`)
  - [ ] `RekapPerKelas` (`:many`)
- [x] Generate `app/queries/tagihan.sql.go` via sqlc & pastikan build lolos

---

## Fase 3 — Service (business logic)

- [x] Buat `app/services/tagihan.go` — PRD §7
  - [x] `frekuensiKePertemuan(frekuensi) -> n` (4 / 8)
  - [x] `GenerateForPertemuan(pertemuanID)` — logika trigger `pertemuan_ke % n == 0 && >= 2n`, hitung `bulan_ke`, snapshot nominal, tanggal dari pertemuan
  - [x] Penanganan santri gabung tengah (anchor `pertemuan_awal`) — PRD §4.6
  - [x] `MarkLunas(...)`, `Batal(...)`
  - [x] `RingkasanPeriode(...)`
  - [x] `TouchFollowUp(...)`
  - [x] Builder nomor WA (normalisasi `08` → `628`) + render template pesan — PRD §9
- [x] Hook di `app/services/pertemuan.go`: panggil `GenerateForPertemuan` setelah pertemuan tercatat/selesai
- [x] Backfill: command/handler "Sinkronkan Tagihan" untuk pertemuan lama

---

## Fase 4 — Handler & Route

- [x] DTO di `app/models/tagihan.go` (request/response + tipe filter)
- [x] Buat `app/handlers/tagihan.go`:
  - [x] `Dashboard` (ringkasan + daftar sudah/belum bayar)
  - [x] `ListTagihan` (dengan filter query params)
  - [x] `MarkLunas`
  - [x] `Batal`
  - [x] `FollowUp` (return URL wa.me + touch FU)
  - [x] `Sync` (backfill)
- [x] Registrasi route di `routes/web.go` + middleware role `keuangan`/`super_admin`

---

## Fase 5 — Frontend (Inertia + Svelte)

- [x] Tambah tipe di `frontend/src/lib/types.ts` (Tagihan, RingkasanKeuangan, filter)
- [x] `frontend/src/pages/keuangan/Dashboard.svelte` — kartu ringkasan + tab sudah/belum bayar — PRD §8.1
- [x] `frontend/src/pages/keuangan/Tagihan.svelte` — tabel + filter + aksi baris — PRD §8.2
  - [x] Badge status (belum_bayar/lunas/terlambat/batal)
  - [x] Aksi: Tandai Lunas (modal), Detail, FU WhatsApp, Batalkan
  - [x] Tombol FU nonaktif bila `no_wa` kosong
- [x] (Opsional) komponen Detail Tagihan + log follow-up
- [x] Tambah menu "Keuangan" di `AppLayout.svelte` (khusus role keuangan/super_admin)

---

## Fase 6 — Pengujian

- [x] Unit test `app/services/tagihan_test.go` (pola seperti `tsi_test.go`):
  - [~] Trigger 1x/pekan muncul di pertemuan 8,12,16 (bukan 4) — tercakup 4/8/12; pertemuan 16 belum diuji eksplisit
  - [~] Trigger 2x/pekan muncul di pertemuan 16,24 (bukan 8/12) — tercakup 8/16; pertemuan 24 belum diuji eksplisit
  - [x] Idempotensi (generate 2x tidak duplikat)
  - [x] Snapshot nominal tidak berubah surut
  - [x] Edge case santri gabung tengah
  - [~] Normalisasi nomor WA + render template — normalisasi diuji; render template belum diuji eksplisit
- [ ] Uji manual end-to-end sesuai Acceptance Criteria (PRD §13)
- [x] `go build` / `go test ./...` + build frontend lolos

---

## Fase 7 — Rilis

- [x] Review kode (self-review + reviewer terpisah)
- [ ] Commit atomik per fase
- [ ] Update dokumentasi bila perlu
- [ ] Deploy & smoke test di lingkungan target
