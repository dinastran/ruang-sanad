-- name: ListTasbihDzikirForUser :many
SELECT d.id, d.user_id, d.teks_arab, d.latin, d.arti, d.rujukan, d.target_default,
       COALESCE(t.target, d.target_default) AS target_efektif
FROM tasbih_dzikir d
LEFT JOIN tasbih_target_user t ON t.dzikir_id = d.id AND t.user_id = sqlc.arg(user_id)
WHERE (d.user_id IS NULL OR d.user_id = sqlc.arg(user_id)) AND d.diarsipkan_at IS NULL
ORDER BY (d.user_id IS NOT NULL), d.urutan, d.id;

-- name: ListTasbihPaket :many
SELECT id, slug, nama, deskripsi, rujukan FROM tasbih_paket ORDER BY urutan, id;

-- name: ListTasbihPaketItem :many
SELECT paket_id, urutan, dzikir_id, target FROM tasbih_paket_item ORDER BY paket_id, urutan;

-- name: CountTasbihDzikirCustomAktif :one
SELECT COUNT(*) FROM tasbih_dzikir WHERE user_id = ? AND diarsipkan_at IS NULL;

-- name: CreateTasbihDzikirCustom :one
INSERT INTO tasbih_dzikir (user_id, teks_arab, latin, arti, rujukan, target_default, urutan, created_at, updated_at)
VALUES (?, ?, ?, ?, '', ?, 0, ?, ?)
RETURNING id;

-- name: UpdateTasbihDzikirCustom :execrows
UPDATE tasbih_dzikir
SET teks_arab = ?, latin = ?, arti = ?, target_default = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND diarsipkan_at IS NULL;

-- name: ArchiveTasbihDzikirCustom :execrows
UPDATE tasbih_dzikir SET diarsipkan_at = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND diarsipkan_at IS NULL;

-- name: CountTasbihDzikirAksesUser :one
SELECT COUNT(*) FROM tasbih_dzikir
WHERE id = sqlc.arg(id) AND (user_id IS NULL OR user_id = sqlc.arg(user_id));

-- name: UpsertTasbihTargetUser :exec
INSERT INTO tasbih_target_user (user_id, dzikir_id, target, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, dzikir_id) DO UPDATE SET target = excluded.target, updated_at = excluded.updated_at;

-- name: GetTasbihTargetHarian :one
SELECT target_harian FROM tasbih_pengaturan WHERE user_id = ?;

-- name: UpsertTasbihTargetHarian :exec
INSERT INTO tasbih_pengaturan (user_id, target_harian, updated_at) VALUES (?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET target_harian = excluded.target_harian, updated_at = excluded.updated_at;

-- name: InsertTasbihSesi :execrows
INSERT INTO tasbih_sesi (id, user_id, dzikir_id, paket_id, paket_run_id, jumlah, target, selesai_target, mulai_at, selesai_at, tanggal_wib, jam_wib, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO NOTHING;

-- name: GetTasbihSesiOwner :one
SELECT user_id FROM tasbih_sesi WHERE id = ?;

-- name: CountTasbihPaket :one
SELECT COUNT(*) FROM tasbih_paket WHERE id = ?;

-- name: ListTasbihPaketItemTarget :many
SELECT target FROM tasbih_paket_item WHERE paket_id = ? AND dzikir_id = ?;
