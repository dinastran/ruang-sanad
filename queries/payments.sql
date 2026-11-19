-- name: CreatePayment :one
INSERT INTO payments (lead_id, link_id, produk_varian_id, metode, amount, status, bukti_path, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPayment :one
SELECT * FROM payments WHERE id = ?;

-- name: ListPaymentsByLead :many
SELECT * FROM payments WHERE lead_id = ? ORDER BY created_at DESC, id DESC;

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = ?, paid_at = ?, catatan_sistem = ?, updated_at = ? WHERE id = ?;

-- name: UpdatePaymentProof :exec
UPDATE payments SET bukti_path = ?, amount = ?, updated_at = ? WHERE id = ?;

-- name: ListPendingManualPaymentsByLead :many
SELECT * FROM payments WHERE lead_id = ? AND metode = 'manual' AND status = 'menunggu_verifikasi'
ORDER BY id;

-- name: CreateGatewayPayment :one
INSERT INTO payments (lead_id, link_id, metode, provider, external_id, amount, status, expires_at,
    jadwal_kode, jenis_kelamin, created_at, updated_at)
VALUES (?, ?, 'gateway', ?, ?, ?, 'pending', ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePaymentGatewaySession :exec
UPDATE payments SET provider_ref = ?, checkout_url = ?, expires_at = ?, updated_at = ? WHERE id = ?;

-- name: GetActiveGatewayPaymentByLead :one
SELECT * FROM payments
WHERE lead_id = ? AND metode = 'gateway' AND status = 'pending' AND checkout_url != '' AND expires_at > sqlc.arg(now)
ORDER BY id DESC LIMIT 1;

-- name: GetPaymentByExternalID :one
SELECT * FROM payments WHERE external_id = ?;

-- name: SetPaymentStatusIfPending :execrows
UPDATE payments SET status = ?, catatan_sistem = ?, updated_at = ? WHERE id = ? AND status = 'pending';

-- name: MarkPaymentPaid :execrows
UPDATE payments SET status = 'paid', paid_at = ?, channel = ?, perlu_tinjau = ?, catatan_sistem = ?, raw_webhook = ?, updated_at = ?
WHERE id = ? AND status != 'paid';

-- name: ListActiveSeatHolds :many
-- Kursi yang ditahan sesi bayar online aktif (lead belum terverifikasi).
SELECT p.lead_id, p.jadwal_kode, p.jenis_kelamin FROM payments p JOIN leads l ON l.id = p.lead_id
WHERE p.metode = 'gateway' AND p.status = 'pending' AND p.expires_at > sqlc.arg(now)
  AND p.jadwal_kode != '' AND l.verified_at IS NULL;

-- name: FailPaymentForReview :execrows
UPDATE payments SET status = 'failed', perlu_tinjau = 1, catatan_sistem = ?, raw_webhook = ?, updated_at = ?
WHERE id = ? AND status = 'pending';

-- name: FlagPaymentReview :exec
UPDATE payments SET perlu_tinjau = 1, catatan_sistem = ?, updated_at = ? WHERE id = ?;
