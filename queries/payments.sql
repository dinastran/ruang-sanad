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
