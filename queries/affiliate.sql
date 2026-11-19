-- name: CreateAffiliate :one
INSERT INTO affiliates (kode, tipe, guru_id, user_id, nama, no_wa, email, status, bank_nama, bank_rekening, bank_atas_nama, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetAffiliate :one
SELECT * FROM affiliates WHERE id = ?;

-- name: GetAffiliateByKode :one
SELECT * FROM affiliates WHERE kode = ?;

-- name: GetAffiliateByUser :one
SELECT * FROM affiliates WHERE user_id = ?;

-- name: GetAffiliateByGuru :one
SELECT * FROM affiliates WHERE guru_id = ?;

-- name: ListAffiliates :many
SELECT * FROM affiliates ORDER BY CASE status WHEN 'menunggu' THEN 0 WHEN 'aktif' THEN 1 ELSE 2 END, nama;

-- name: UpdateAffiliate :exec
UPDATE affiliates SET kode = ?, nama = ?, no_wa = ?, email = ?, status = ?, bank_nama = ?, bank_rekening = ?, bank_atas_nama = ?,
    catatan_admin = ?, user_id = ?, disetujui_oleh = ?, disetujui_at = ?, updated_at = ?
WHERE id = ?;

-- name: ListGuruWithoutAffiliate :many
SELECT g.id, g.nama, g.user_id FROM guru g
WHERE g.user_id IS NOT NULL AND g.is_aktif = 1 AND NOT EXISTS (SELECT 1 FROM affiliates a WHERE a.guru_id = g.id);

-- name: GetAffiliateSettings :one
SELECT * FROM affiliate_settings WHERE id = 1;

-- name: UpdateAffiliateSettings :exec
UPDATE affiliate_settings SET masa_berlaku_hari = ?, masa_tahan_hari = ?, link_tujuan_default_id = ?, template_bagikan = ?, updated_by = ?, updated_at = ?
WHERE id = 1;

-- name: UpsertAffiliateRule :one
INSERT INTO affiliate_rules (scope, target_id, jenis, nilai, pola, maks_bulan, is_aktif, updated_by, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (scope, target_id) DO UPDATE SET jenis = excluded.jenis, nilai = excluded.nilai, pola = excluded.pola,
    maks_bulan = excluded.maks_bulan, is_aktif = excluded.is_aktif, updated_by = excluded.updated_by, updated_at = excluded.updated_at
RETURNING *;

-- name: ListAffiliateRules :many
SELECT * FROM affiliate_rules ORDER BY CASE scope WHEN 'global' THEN 0 WHEN 'program' THEN 1 ELSE 2 END, target_id;

-- name: GetAffiliateRule :one
SELECT * FROM affiliate_rules WHERE scope = ? AND target_id = ? AND is_aktif = 1;

-- name: DeleteAffiliateRule :exec
DELETE FROM affiliate_rules WHERE id = ? AND scope != 'global';

-- name: BumpAffiliateClick :exec
INSERT INTO affiliate_clicks (affiliate_id, tanggal, klik) VALUES (?, ?, 1)
ON CONFLICT (affiliate_id, tanggal) DO UPDATE SET klik = klik + 1;

-- name: SumAffiliateClicksSince :many
SELECT affiliate_id, CAST(SUM(klik) AS INTEGER) AS klik FROM affiliate_clicks WHERE tanggal >= ? GROUP BY affiliate_id;

-- name: InsertLeadAffiliate :execrows
INSERT OR IGNORE INTO lead_affiliates (lead_id, affiliate_id, sumber, diklik_at, dikunci_at, diubah_oleh)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ReplaceLeadAffiliate :exec
INSERT INTO lead_affiliates (lead_id, affiliate_id, sumber, diklik_at, dikunci_at, diubah_oleh)
VALUES (?, ?, 'manual', NULL, ?, ?)
ON CONFLICT (lead_id) DO UPDATE SET affiliate_id = excluded.affiliate_id, sumber = 'manual', diklik_at = NULL,
    dikunci_at = excluded.dikunci_at, diubah_oleh = excluded.diubah_oleh;

-- name: DeleteLeadAffiliate :exec
DELETE FROM lead_affiliates WHERE lead_id = ?;

-- name: GetLeadAffiliate :one
SELECT la.*, a.kode, a.nama, a.tipe FROM lead_affiliates la JOIN affiliates a ON a.id = la.affiliate_id WHERE la.lead_id = ?;

-- name: InsertAffiliateCommission :execrows
INSERT OR IGNORE INTO affiliate_commissions (affiliate_id, lead_id, santri_id, sumber, sumber_id, bulan_ke, basis, jenis, nilai, pola,
    maks_bulan, jumlah, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateAffiliateInvite :one
INSERT INTO affiliate_invites (affiliate_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?) RETURNING *;

-- name: GetAffiliateInviteByHash :one
SELECT * FROM affiliate_invites WHERE token_hash = ?;

-- name: UseAffiliateInvite :execrows
UPDATE affiliate_invites SET used_at = ? WHERE id = ? AND used_at IS NULL AND expires_at > ?;
