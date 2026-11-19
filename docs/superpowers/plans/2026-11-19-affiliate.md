# Affiliate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. Steps use checkbox syntax.

**Goal:** Guru & eksternal mendapat komisi dari pendaftar lewat link affiliate, sesuai spec.
**Architecture:** `AffiliateService` (affiliate, aturan, pengaturan, atribusi, rekonsiliasi komisi, pencairan). Atribusi dipanggil dari `RegistrationService.Submit`, `lead_ingest`, dan halaman lead. Worker rekonsiliasi 10 menit. Ujroh membaca komisi lewat service.
**Tech Stack:** Go Fiber, sqlc, SQLite, Svelte 5.
**Spec:** `docs/superpowers/specs/2026-11-19-affiliate-design.md`

## Global Constraints
- Semua aturan AGENTS.md; migrasi mulai 0129; satu tabel per file.
- Persen disimpan basis poin (0–10000). Pembulatan komisi ke bawah.
- Cookie `rs_aff` = `KODE.unix.hmac` (HMAC-SHA256 secret sesi), HttpOnly, SameSite=Lax, Secure bila https.
- Kode affiliate: `^[A-Z0-9]+(-[A-Z0-9]+)*$`, 3–20 karakter, disimpan huruf besar.
- Role baru `affiliate` di `models.ValidRoles()` dan `roleHome` → `/app/affiliate`.
- Affiliate hanya melihat nama depan rujukan.

## Review Focus
1. Rekonsiliasi dijalankan dua kali / bersamaan → tidak ada komisi ganda.
2. Pendaftar klik link dua affiliate → affiliate pertama tetap.
3. Santri lama dengan tagihan lunas sebelum daftar ulang → tidak ada komisi untuk tagihan lama.
4. Bulan ujroh dikunci lalu dibuka → komisi kembali siap, tidak hilang/dobel.
5. Role affiliate mencoba membuka /app/dashboard atau /app/leads → ditolak.

### Task 1: Skema + query
Tabel: affiliates, affiliate_rules, affiliate_settings (seed 1 baris), affiliate_clicks, lead_affiliates, affiliate_commissions, affiliate_invites; seed ujroh_komponen "Komisi Affiliate" (insidental). Test round-trip.

### Task 2: Affiliate & aturan (service)
`AffiliateService`: `EnsureGuruAffiliates`, `Create/Update/Approve/Reject/SetStatus`, kode unik, `ResolveRule(affiliateID, programID)`, `Settings/SaveSettings`, `Rules/SaveRule`. Tests: resolusi affiliate>program>global, validasi persen/flat, kode unik, sinkron guru.

### Task 3: Atribusi
`TrackClick` + cookie `AffCookie/ParseAffCookie` (first-click tidak menimpa cookie sah lain); `LockLead(q, leadID, src)` first-click, self-referral ditolak; integrasi `RegistrationService.Submit` (cookie via `RequestMeta.AffCookie`, field `KodeReferal`); deteksi `ref: KODE` di ingest bila lead belum ber-affiliate; `SetLeadAffiliateManual`. Handler `/r/:kode`. Tests per jalur.

### Task 4: Rekonsiliasi komisi
`Reconcile(ctx)`: pendaftaran, SPP setelah konversi (urutan sejak konversi, ≤ maks_bulan), matang/batal, aktivasi ulang. Worker 10 menit di main. Tests: idempoten 2×, santri lama, batal verifikasi & re-verify, santri berhenti dalam tahan, maks_bulan, persen/flat, affiliate nonaktif.

### Task 5: Pencairan
Ujroh: baris sistem "Komisi Affiliate" ikut perhitungan live, tidak bisa di-set/hapus; `Kunci` → komisi dibayar; `Buka` → siap. Eksternal: `MarkTransferred(ids, actor, catatan)`. Tests.

### Task 6: Akun eksternal
Form `/affiliate/daftar` (+rate limit), approve → user role affiliate + invite token 72 jam (WA outbox + email bila SMTP) → halaman buat password `/affiliate/aktivasi/:token`. Deny-guard role affiliate. Tests.

### Task 7: UI
Affiliate Saya, admin (`/app/leads/affiliate`: affiliate, persetujuan, aturan, pengaturan, leaderboard, buku komisi & pencairan), panel lead, kolom kode referal di form (per link `tampil_kode_referal`), navigasi.

### Task 8: Verifikasi + review akhir
