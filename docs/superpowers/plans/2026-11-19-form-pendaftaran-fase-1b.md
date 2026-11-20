# Form Pendaftaran — Fase 1b (Xendit) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Pendaftar bisa membayar online (QRIS/e-wallet/VA) lewat Xendit Payment Session; lunas → lead terverifikasi otomatis + CAPI Purchase; kursi jadwal ditahan selama sesi bayar aktif.

**Architecture:** Interface `PaymentGateway` (CreateSession, GetSession) dengan implementasi `XenditGateway` (net/http, basic auth secret key). `RegistrationService` membuat sesi dan menangani webhook. Inti verifikasi lead dipecah dari `VerifyPayment` agar bisa dipanggil oleh staf maupun gateway.

**Tech Stack:** Go Fiber, sqlc, SQLite, Svelte 5.

**Spec:** `docs/superpowers/specs/2026-11-19-form-pendaftaran-pembayaran-design.md` (§5.3, §5.5, §10).

## Global Constraints

- Semua Global Constraints fase 1a berlaku.
- Env: `XENDIT_SECRET_KEY`, `XENDIT_CALLBACK_TOKEN`, `XENDIT_API_URL` (default `https://api.xendit.co`). Gateway aktif hanya bila dua yang pertama terisi dan `APP_URL` HTTPS.
- Request sesi: `reference_id` (≤64), `session_type: PAY`, `mode: PAYMENT_LINK`, `amount`, `currency: IDR`, `country: ID`, `customer{reference_id: "lead-<id>", type: INDIVIDUAL, mobile_number: "+62…", individual_detail{given_names}}`, `expires_at` = now + `payment_expiry_hours`, `description`, `success_return_url`/`cancel_return_url` = `APP_URL/daftar/bayar/<ref>`, `locale: id`.
- Webhook `POST /webhooks/xendit`: token `x-callback-token` constant-time, body ≤ 64 KB, selalu 200 untuk event tak dikenal, 401 untuk token salah.
- Kursi ditahan oleh `payments` gateway `pending` dengan `expires_at > now`.
- Tidak ada panggilan jaringan di test (gateway di-stub / httptest).

## Review Focus

1. Webhook dikirim dua kali / bersamaan → satu verifikasi, satu Purchase, satu increment `bayar` → Task 4.
2. Webhook `completed` palsu (token benar tapi sesi di Xendit belum COMPLETED / amount beda) → tidak lunas → Task 4.
3. Pendaftar klik "Bayar" berkali-kali → satu sesi aktif dipakai ulang → Task 3.
4. Lead dipindah lost/terverifikasi oleh staf sebelum webhook datang → payment paid + perlu tinjau, lead tidak diubah → Task 4.
5. Sesi kedaluwarsa tanpa webhook → kursi lepas, halaman bayar bisa membuat sesi baru → Task 3.

---

### Task 1: Gateway client Xendit
- Create `app/services/payment_gateway.go`: `type PaymentGateway interface { CreateSession(ctx, GatewaySessionRequest) (GatewaySession, error); GetSession(ctx, id string) (GatewaySession, error) }`, `GatewaySession{ID, ReferenceID, Status, Amount int64, URL string, ExpiresAt time.Time, PaymentID string}`, `XenditGateway{baseURL, secretKey, client}`.
- Test `payment_gateway_test.go` dengan `httptest.Server`: basic auth = secretKey + ":", body JSON memuat field di Global Constraints, parse response `payment_session_id`/`payment_link_url`/`status`/`expires_at`; status non-2xx → error berisi `error_code`.
- Commit `feat(bayar): klien Xendit Payment Session`.

### Task 2: Inti verifikasi bersama
- Refactor `lead_closing.go`: `verifyLeadTx(ctx, q, l *queries.Lead, v leadVerification, now) (map[string]any, error)` dipakai `VerifyPayment` (staf; aturan tahap lama tetap) dan gateway (`AllowAnyOpenStage`). `verified_by` NULL untuk sistem.
- Test regresi: test VerifyPayment yang ada tetap hijau; test baru `TestVerifyLeadTxGateway` (lead di `info_bayar_dikirim` → terverifikasi, verified_by NULL, event `payment.verified` dengan amount).
- Commit `refactor(lead): inti verifikasi pembayaran bersama`.

### Task 3: Buat sesi bayar + penahanan kursi
- Migration `0129_add_session_fields_index_to_payments.sql`: index `payments(metode, status, expires_at)` (kolom sudah ada).
- Query: `ListActiveSeatHolds(now)`, `GetActiveGatewayPaymentByLead(lead_id, now)`, `GetPaymentByExternalID`, `UpdatePaymentGatewaySession`, `MarkPaymentPaid`.
- `JadwalPenawaranService.seats` menambah kursi dari `ListActiveSeatHolds` (kecuali lead yang sudah terverifikasi agar tidak dobel).
- `RegistrationService.WithGateway(gw, appURL)`, `GatewayEnabled()`, `StartGatewayPayment(ctx, ref) (url string, err error)`.
- Tests: reuse sesi aktif; kursi penuh → FieldError jadwal; pending menahan kursi di `Open`; kedaluwarsa melepas kursi; gateway error → payment `failed` dan kursi lepas; lead won/verified → `ErrSudahDiproses`.
- Commit `feat(bayar): sesi bayar Xendit dan penahanan kursi`.

### Task 4: Webhook
- `RegistrationService.HandleXenditWebhook(ctx, body []byte) error` + handler `app/handlers/xendit_webhook.go` (token check di handler, service tidak tahu header).
- `completed`: GetSession konfirmasi (status COMPLETED, reference_id & amount cocok) → dalam satu tx: payment `paid` (idempoten: sudah paid → no-op), `channel` = payment_id, `BumpRegistrationStat(bayar)`; lead open & belum terverifikasi → `verifyLeadTx` (actor `gateway:xendit`) + `enqueuePurchase`; selain itu → `perlu_tinjau`. Re-cek kursi: kelebihan → `perlu_tinjau` + notifikasi koordinator. Notifikasi keuangan "Pembayaran online diterima".
- `expired`: pending → `expired`.
- Tests: token salah 401 (handler); replay → satu event Purchase & bayar=1; GetSession bilang ACTIVE → tidak lunas; amount beda → `failed`+tinjau; lead lost → paid + tinjau; event tak dikenal → nil.
- Commit `feat(bayar): webhook Xendit verifikasi otomatis`.

### Task 5: UI + wiring
- Config + main wiring, route `POST /daftar/bayar/:ref/gateway` (rate limit upload), `GET /daftar/bayar/:ref/status` (JSON status), `POST /webhooks/xendit`.
- `BayarInfo` + `gateway_aktif`; `DaftarBayar.svelte`: tombol utama "Bayar sekarang" (QRIS/e-wallet/VA), transfer manual jadi opsi sekunder, polling status tiap 5 detik ≤ 10 menit saat ada sesi aktif / kembali dari Xendit.
- `.env.example` (bila ada) + catatan di wiki deploy: URL webhook untuk didaftarkan di dashboard Xendit.
- Tests handler: gateway nonaktif → 404/422; status JSON.
- Commit `feat(bayar): tombol bayar online dan status`.

### Task 6: Verifikasi + review akhir
- `go vet ./...`, `go test ./...`, `npm run build:all`; review subagent; perbaikan.
