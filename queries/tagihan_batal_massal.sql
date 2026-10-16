-- name: CreateTagihanBatalBatch :one
INSERT INTO tagihan_batal_batch (alasan, filter_json, dibuat_oleh)
VALUES (?, ?, ?) RETURNING id;

-- name: FinalizeTagihanBatalBatch :exec
UPDATE tagihan_batal_batch SET jumlah = ?, total_nominal = ? WHERE id = ?;

-- name: BatalkanTagihanBelumBayar :execrows
UPDATE tagihan
SET status = 'batal', catatan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'belum_bayar';

-- name: CreateTagihanBatalBatchItem :exec
INSERT INTO tagihan_batal_batch_item (batch_id, tagihan_id, status_lama, catatan_lama)
VALUES (?, ?, ?, ?);

-- name: GetTagihanBatalBatch :one
SELECT * FROM tagihan_batal_batch WHERE id = ?;

-- name: ListTagihanBatalBatchItems :many
SELECT * FROM tagihan_batal_batch_item WHERE batch_id = ? ORDER BY id;

-- name: PulihkanTagihanBatal :execrows
UPDATE tagihan
SET status = ?, catatan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'batal';

-- name: MarkTagihanBatalBatchDipulihkan :execrows
UPDATE tagihan_batal_batch
SET dipulihkan_at = CURRENT_TIMESTAMP, dipulihkan_oleh = ?, jumlah_dipulihkan = ?
WHERE id = ? AND dipulihkan_at IS NULL;

-- name: ListTagihanBatalBatch :many
SELECT b.*, COALESCE(u.name, '') AS dibuat_oleh_nama, COALESCE(p.name, '') AS dipulihkan_oleh_nama
FROM tagihan_batal_batch b
LEFT JOIN users u ON u.id = b.dibuat_oleh
LEFT JOIN users p ON p.id = b.dipulihkan_oleh
ORDER BY b.id DESC
LIMIT ?;
