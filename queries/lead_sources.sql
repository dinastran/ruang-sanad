-- name: ListLeadSources :many
SELECT * FROM lead_sources ORDER BY is_aktif DESC, nama;

-- name: GetLeadSource :one
SELECT * FROM lead_sources WHERE id = ?;

-- name: CreateLeadSource :one
INSERT INTO lead_sources (nama, keterangan, url, is_aktif, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateLeadSource :exec
UPDATE lead_sources SET nama = ?, keterangan = ?, url = ?, is_aktif = ?, updated_at = ? WHERE id = ?;

-- name: DeleteLeadSource :execrows
DELETE FROM lead_sources WHERE id = ?;

-- name: ListLeadSourcePatterns :many
SELECT * FROM lead_source_patterns ORDER BY source_id, id;

-- name: ListActiveLeadSourcePatterns :many
-- Teks pemicu dari sumber aktif saja; dipakai untuk pencocokan.
SELECT p.source_id, p.teks_normal FROM lead_source_patterns p
JOIN lead_sources s ON s.id = p.source_id
WHERE s.is_aktif = 1;

-- name: GetLeadSourcePatternByNormal :one
SELECT * FROM lead_source_patterns WHERE teks_normal = ?;

-- name: CreateLeadSourcePattern :one
INSERT INTO lead_source_patterns (source_id, teks, teks_normal, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: DeleteLeadSourcePattern :execrows
DELETE FROM lead_source_patterns WHERE id = ?;

-- name: CountLeadsBySource :many
SELECT lead_source_id, COUNT(*) AS total FROM leads
WHERE lead_source_id IS NOT NULL GROUP BY lead_source_id;

-- name: CountLeadsOfSource :one
SELECT COUNT(*) FROM leads WHERE lead_source_id = ?;

-- name: SetLeadSource :exec
UPDATE leads SET lead_source_id = ?, lead_source_by = ?, updated_at = ? WHERE id = ?;

-- name: CountLeadInboundMessages :one
SELECT COUNT(*) FROM lead_messages WHERE lead_id = ? AND direction = 'in';

-- name: ListLeadsForSourceRematch :many
-- Lead tanpa sumber yang tidak diisi manual, dengan pesan masuk pertamanya.
SELECT l.id, CAST(COALESCE((SELECT m.body FROM lead_messages m
    WHERE m.lead_id = l.id AND m.direction = 'in' ORDER BY m.sent_at, m.id LIMIT 1), '') AS TEXT) AS first_body
FROM leads l
WHERE l.lead_source_id IS NULL AND l.lead_source_by != 'manual';

-- name: LeadSourceCohort :many
-- Lead yang masuk dalam periode, dikelompokkan per sumber (0 = tanpa sumber).
SELECT CAST(COALESCE(lead_source_id, 0) AS INTEGER) AS source_id,
    COUNT(*) AS total,
    CAST(COALESCE(SUM(CASE WHEN status = 'won' THEN 1 ELSE 0 END), 0) AS INTEGER) AS won,
    CAST(COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END), 0) AS INTEGER) AS open
FROM leads WHERE created_at >= ? AND created_at < ?
GROUP BY 1;
