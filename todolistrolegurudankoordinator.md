# Todo Implementasi Role Guru dan Koordinator Guru

Checklist ini disusun dari `fitur-role-guru-dan-koordinator.md` dan kondisi proyek saat ini: Go Fiber, Svelte 5 + Inertia, SQLite, Goose, dan sqlc. Tidak mencakup implementasi WhatsApp Business API, aplikasi mobile native, payroll TSI, kelas video, import TSI historis, atau kurikulum lengkap sebagaimana ditunda dalam PRD.

## Kondisi Existing yang Menjadi Acuan

- ✅ Pertahankan tabel `guru` sebagai master guru; tabel ini sudah dipakai oleh `kelas.guru_id`.
- ✅ Tambahkan relasi eksplisit antara akun `users` dan `guru` (`guru.user_id` + unique index).
- ✅ Perluas enum role pada `app/models/user.go`, validasi role, seed, halaman `/admin/users`, dashboard, dan navigasi untuk `guru` serta `koordinator_guru`.
- ✅ Pertahankan middleware otorisasi per-route di `routes/web.go`; per-route pattern tidak berubah.
- ✅ Buat handler baru per domain fitur (`GuruHandler`, `PertemuanHandler`, `RiayahHandler`, `KoordinatorGuruHandler`).
- ✅ Semua akses database dibuat di `queries/*.sql`, digenerate melalui `sqlc`, lalu dipanggil oleh service.
- ✅ Buat migrasi Goose baru (`0015_guru_extend.sql`); migration `0007`–`0013` tidak diubah.

## Keputusan yang Harus Dikunci Sebelum Mulai

- ✅ Mekanisme `guru.user_id` UNIQUE untuk menghubungkan akun login dengan master guru.
- ✅ Super Admin yang membuat/mengaktifkan/menghubungkan akun Guru (via `/admin/users`).
- ✅ Status kehadiran: `hadir`, `izin`, `sakit`, `alpa`, `telat`.
- ❌ Batas keterlambatan mulai kelas & batas waktu pengisian absensi — butuh dikunci owner.
- ❌ Aturan kuota kelas Guru Tetap vs Part Time — butuh dikunci.
- ✅ Badal/reschedule: langsung berlaku, tercatat di audit trail.
- ✅ Periode TSI: bulan kalender.
- ❌ Ambang alert — butuh dikunci.
- ❌ Predikat huruf TSI — butuh dikunci.
- ❌ Cakupan kuesioner rilis pertama — butuh dikunci.

## Tahap 0: Fondasi Data, Role, dan Otorisasi

- ✅ Migration `0015_guru_extend.sql`: memperluas `guru` (status, no_wa, email, tanggal_gabung, foto, user_id).
- ✅ `santri.no_wa` sudah tersedia dari migration `0013`, dipakai untuk URL `wa.me`.
- ✅ Migration `0015` juga membuat `pertemuan`, `absensi`, `catatan_riayah` dengan badal & reschedule di `pertemuan`.
- ✅ Foreign key, UNIQUE(kelas_id, pertemuan_ke), UNIQUE(pertemuan_id, santri_id), index kelas/tanggal/guru, unique index `guru.user_id`.
- ✅ `RoleGuru` dan `RoleKoordinatorGuru` ditambahkan di `app/models/user.go` + `ValidRoles()`.
- ✅ Validasi role di admin handler — existing code handles it generically via `ROLE_OPTIONS`.
- ✅ Pilihan, label, dan warna role baru di `frontend/src/pages/admin/Users.svelte`.
- ✅ Akun seed Guru (`guru@ruangsanad.com`) dan Koordinator Guru (`koordinator@ruangsanad.com`) di `cmd/seed/main.go`, lengkap dengan relasi ke master `guru`.
- ✅ Middleware role terpisah di `routes/web.go` untuk Guru (`guruRole`), Koordinator Guru (`koordinatorRole`), dan kombinasi dengan Super Admin.
- ❌ Audit seluruh route lama — perlu review manual.

## Tahap 1: Role Guru dan Pertemuan/Absensi

### Backend

- ✅ `GuruHandler` — Dashboard, KelasSaya, DetailKelas, RiwayatSantri.
- ✅ `PertemuanHandler` — Mulai, FormMulai, Selesai, FormSelesai, Detail, Reschedule, Badal, RekapAbsensi, EditAbsensi.
- ✅ `RiayahHandler` — List, Create, Update, Delete, WALink, BroadcastWA.
- ✅ Service `GuruService.EnsureOwnsClass` — filter kelas berdasarkan `kelas.guru_id` dari login user.
- ✅ Query sqlc: `ListKelasByGuruID`, `GetJadwalMengajarHariIni`, `GetLastPertemuanByKelas`, `GetNextPertemuanKe`.
- ✅ Query roster santri per kelas — hanya identitas, domisili, WA, statistik kehadiran (tanpa nominal/keuangan).
- ✅ `Mulai Pertemuan` — validasi kepemilikan kelas, `GetNextPertemuanKe` atomik, rekam jam mulai.
- ✅ `Selesai Pertemuan` — wajib materi + absensi per santri, simpan jam selesai + catatan.
- ✅ Cegah pertemuan tanpa absensi lengkap (validasi di handler & service).
- ✅ Batas edit absensi 48 jam — diimplementasikan di `PertemuanService.EditAbsensi`.
- ✅ `Reschedule` — alasan wajib, simpan jadwal semula + tanggal baru.
- ✅ `Badal` — alasan wajib, guru pengganti valid, penanda `is_badal`.
- ✅ Rekap matriks santri × pertemuan via `GetRekapAbsensiKelas` + `GetPertemuanByKelasForRekap`.
- ✅ CRUD catatan riayah — `RiayahService` dengan target santri, penulis, waktu.
- ✅ Generator URL `wa.me` dan broadcast links (daftar tautan satu-per-satu) — `RiayahService.GetLinkWA`, `GetBroadcastLinks`.
- ✅ Query dashboard Guru: `GetGuruStats`, `GetJadwalMengajarHariIni`, `GetKelasBelumAbsen`.

### Frontend

- ✅ Dashboard Guru — `frontend/src/pages/guru/Dashboard.svelte` (stat cards, jadwal hari ini, kelas belum diabsen).
- ✅ Kelas Saya — `frontend/src/pages/guru/KelasList.svelte` (kartu per kelas).
- ✅ Detail Kelas — `frontend/src/pages/guru/KelasDetail.svelte` (roster, kehadiran, riayah modal, Chat WA, riwayat).
- ✅ Flow mulai-selesai pertemuan mobile-first — `PertemuanMulai.svelte` + `PertemuanSelesai.svelte`.
- ✅ `router.post()`/`router.put()` dari `@inertiajs/svelte` untuk form.
- ✅ Menu role Guru di `AppLayout.svelte` (Dashboard Guru + Kelas Saya).
- ✅ Route-to-group mapping di `AppLayout.svelte` (`guru-dashboard`, `guru-kelas`).

### Verifikasi Tahap 1

- ✅ `go test ./...` lulus.
- ✅ `npm run build` lulus (frontend build sukses).
- ❌ Uji Guru A vs Guru B — perlu dijalankan manual atau via test.
- ❌ Uji pertemuan/absensi — perlu dijalankan manual atau via test.
- ❌ Uji 48h edit, reschedule, badal, santri tidak lanjut, WA kosong — perlu dijalankan.

## Tahap 2: Role Koordinator dan Bukti Penilaian TSI

### Data dan Backend

- ✅ Migration `0017_koordinator_features.sql`: `guru_kompetensi`, `pembinaan`, `pembinaan_absen`, `rapat_guru`, `rapat_absen`, `tilawah_harian`, `kunjungan_kelas`, `kalam_bersanad`, `kalam_share_log`, `wa_template`.
- ⚠️ Log sapa kelas & posting SS — **ditunda ke Tahap 3** (dibutuhkan indikator TSI semi-otomatis); share Kalam sudah via `kalam_share_log`.
- ❌ Penyimpanan file bukti (upload SS) — ditunda; belum kritikal untuk alur inti.
- ✅ `KoordinatorGuruHandler` + `KoordinatorGuruDirectoryHandler` — dashboard + Direktori Guru lengkap.
- ✅ `MasterService` dan query sqlc untuk master guru lengkap (CreateGuru, UpdateGuruFull, ListGuruFull, GetGuruFull).
- ✅ CRUD master guru via Direktori Guru dengan field profil, status, kontak, gelar, foto, tanggal bergabung.
- ✅ **Buat master guru baru** dari Direktori Guru (`CreateGuruFull` + `GuruService.CreateDirectory`, form modal di `GuruList.svelte`).
- ✅ **Hubungkan/lepas akun login** ke record guru (`GuruService.LinkUser`/`UnlinkUser`, route `PUT/DELETE /koordinator-guru/guru/:id/link` — **Super Admin only**, panel di `GuruDetail.svelte`). Validasi UNIQUE `guru.user_id` + pesan ramah.
- ✅ Alur lengkap tambah guru: buat master → hubungkan akun (Super Admin) → set role Guru di `/admin/users` → guru login.
- ✅ CRUD matriks kompetensi 8 bidang (`guru_kompetensi`, upsert, panel di GuruDetail).
- ✅ Pembinaan dan rapat dengan absensi (grid guru × sesi, `KoordinatorFeaturesHandler`).
- ✅ Tilawah harian (self check-in guru di dashboard + rekap bulan berjalan).
- ✅ Kalam Bersanad + pelacakan share per guru.
- ✅ Kunjungan kelas (jadwal target + pelaksanaan + status).
- ✅ Dashboard Koordinator (statistik guru/kelas/santri + guru belum absen pekan ini).
- ✅ Tampilan riayah per Guru (catatan_riayah target_type=guru di GuruDetail).
- ✅ Template WA dasar (CRUD `wa_template`).

### Frontend dan Akses

- ✅ Dashboard Koordinator Guru — real data (`koordinator/Dashboard.svelte`).
- ✅ Halaman master Guru untuk atribut diperluas (direktori Koordinator: profil, status, kontak, gelar, foto, dan tanggal bergabung).
- ✅ Batas akses Admin Kelas ke master Guru (hanya baca; perubahan melalui Direktori Koordinator).
- ✅ Halaman pembinaan/rapat (+ halaman absensi grid masing-masing).
- ✅ Halaman kunjungan kelas.
- ✅ Halaman Kalam Bersanad (+ halaman pelacakan share).
- ✅ Halaman Template WA.
- ✅ Menu Koordinator Guru di `AppLayout.svelte` + route-to-group (7 menu).

### Verifikasi Tahap 2

- ❌ Uji Guru vs Koordinator akses.
- ❌ Uji Super Admin akses penuh.
- ❌ Uji check-in tilawah, absensi per sesi.
- ❌ Uji unggahan bukti.

## Tahap 3: Mesin dan Rapor TSI

### Master dan Perhitungan

- ✅ Migration `0018_tsi_engine.sql`: `tsi_kriteria`, `tsi_periode`, `tsi_nilai`, `tsi_audit`.
- ✅ Seed 18 indikator TSI (verbatim) di migration.
- ✅ Service kalkulasi TSI (`TSIService`).
- ✅ Aturan indikator kosong (dikecualikan dari rata-rata) — `kategoriSkor`.
- ✅ Rumus bobot 40/30/20/10 (skor kategori = bobot × rata-rata, total = jumlah).
- ✅ Kalkulasi 8 indikator otomatis (absen dibuat/riayah, no-reschedule, no-badal, retensi, pembinaan, rapat, share Kalam). Sisanya manual/semi (data belum tersedia — jujur).
- ✅ Semi/manual dengan input koordinator.
- ✅ Override beralasan + audit trail (`tsi_audit`).
- ✅ Finalisasi & buka kembali periode.
- ⚠️ Regenerasi draft: nilai otomatis dihitung ulang setiap kali halaman dibuka (tidak disnapshot sampai difinalisasi).
- ✅ Tunda otomasi kurikulum (indikator #2 kompetensi = semi/manual).
- ✅ Tunda komplain/survei (indikator terkait = manual).

### Halaman dan Laporan

- ✅ Halaman penilaian TSI Koordinator (`koordinator/TSIPenilaian.svelte`).
- ✅ Halaman TSI Saya — real, read-only dengan angka mentah (`guru/TSISaya.svelte`).
- ✅ Rekap matriks guru × kategori (`koordinator/TSIRekap.svelte`, dipisah Tetap/Part Time).
- ❌ Tren dan peringkat — **ditunda** (butuh data lintas bulan).
- ❌ Rapor PDF — **ditunda**.
- ❌ Export Excel — **ditunda**.

### Verifikasi Tahap 3

- ✅ Unit test rumus (`app/services/tsi_test.go`).
- ✅ Fixture Dina Riska (Kompetensi 5 dari 6 indikator × 92% = 36,80).
- ✅ Migration up/down tervalidasi pada DB sementara.
- ⚠️ Uji indikator otomatis end-to-end & isolasi antar guru — perlu uji manual/integration.

## Tahap 4: Todo Koordinator, Kuesioner, Template, dan Reminder

- ✅ **Todo Koordinator** — migration `0019_todo_koordinator.sql`, CRUD + status (belum/proses/selesai/batal) + penanda berulang (harian/mingguan/bulanan/4-bulanan), field teknis/kebutuhan/deadline/PIC/link/catatan. Halaman `koordinator/Todo.svelte`.
- ✅ **Template WA** — sudah dibuat di Tahap 2 (`wa_template` CRUD).
- ⚠️ **Reminder** — sebagian: dashboard koordinator sudah menyoroti "guru belum absen pekan ini" + tombol WA. Reminder terjadwal (H-1 pembinaan/rapat via notifikasi) **ditunda** — butuh sistem cron/notifikasi.
- ❌ **Kuesioner & survei** — **ditunda**: butuh permukaan baru (halaman isian publik tanpa login + agregasi hasil) yang layak dikerjakan sebagai pass tersendiri. Belum kritikal untuk alur inti.
- ⚠️ **Todo berulang otomatis** — penanda `recurring` tersimpan; generator instance otomatis per periode ditunda (butuh cron).

## Pengamanan, Kualitas Data, dan Observabilitas

- ✅ Validasi ID relasi di server + cek kepemilikan/role di service (`EnsureOwnsClass`).
- ✅ Hindari nama Guru sebagai key relasi; gunakan `guru.id` secara konsisten.
- ✅ Simpan nilai numerik sebagai angka; format `id-ID` hanya saat ditampilkan.
- ⚠️ Audit trail — TSI override sudah beraudit (`tsi_audit`); domain lain belum.
- ❌ Pagination/filter (list masih tanpa paging).
- ✅ Index sudah dibuat di migration 0015/0017/0018/0019.
- ❌ Keamanan file upload (fitur upload bukti belum dibuat).
- ❌ Logging terstruktur khusus domain guru/koordinator.

## Uji Rilis dan Migrasi Operasional

- ✅ Migration up/down tervalidasi pada DB sementara (0001–0019, seed 18 indikator TSI).
- ⚠️ Prosedur awal akun Guru — alur tersedia di UI (buat master → hubungkan akun → set role); SOP tertulis belum dibuat.
- ✅ Mulai TSI dari periode berjalan (periode dibuat on-demand per bulan).
- ❌ Skenario E2E otomatis.
- ⚠️ Uji mobile — layout mobile-first dipakai; belum diuji perangkat nyata.
- ⚠️ Uji pemisahan data — `EnsureOwnsClass` + role middleware ada; unit/integration test isolasi belum.
- ✅ `go test ./...` (termasuk unit test rumus TSI) dan `npm run build` lulus.
- ❌ Review keamanan menyeluruh.
- ❌ SOP Koordinator tertulis.
