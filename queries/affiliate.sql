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

-- name: ListCommissionCandidatesPendaftaran :many
-- Lead ber-affiliate yang terverifikasi setelah terkunci (atau dikunci manual)
-- dan belum punya komisi pendaftaran.
SELECT l.id AS lead_id, la.affiliate_id,
       CAST(COALESCE(l.program_id, (SELECT s.program_id FROM lead_sources s WHERE s.id = l.lead_source_id), 1) AS INTEGER) AS program_id,
       l.payment_amount, l.santri_id
FROM lead_affiliates la
JOIN leads l ON l.id = la.lead_id
JOIN affiliates a ON a.id = la.affiliate_id
WHERE l.verified_at IS NOT NULL AND a.status = 'aktif'
  AND (la.sumber = 'manual' OR l.verified_at >= la.dikunci_at)
  AND NOT EXISTS (SELECT 1 FROM affiliate_commissions c WHERE c.sumber = 'pendaftaran' AND c.sumber_id = l.id);

-- name: ListReactivatableCommissions :many
SELECT c.id, c.lead_id, l.payment_amount FROM affiliate_commissions c JOIN leads l ON l.id = c.lead_id
WHERE c.sumber = 'pendaftaran' AND c.status = 'batal' AND c.alasan_batal = 'verifikasi dibatalkan' AND l.verified_at IS NOT NULL;

-- name: ReactivateCommission :exec
UPDATE affiliate_commissions SET status = 'pending', basis = ?, jumlah = ?, alasan_batal = '', created_at = ?, updated_at = ? WHERE id = ?;

-- name: CancelCommissionsVerificationUndone :execrows
UPDATE affiliate_commissions SET status = 'batal', alasan_batal = 'verifikasi dibatalkan', updated_at = sqlc.arg(now)
WHERE status = 'pending' AND sumber = 'pendaftaran'
  AND lead_id IN (SELECT id FROM leads WHERE verified_at IS NULL);

-- name: CancelCommissionsSantriStopped :execrows
-- Hanya bila santri berhenti SETELAH komisi dibuat (status lama santri yang
-- kembali mendaftar tidak membatalkan). Waktu dibandingkan pada 19 karakter
-- pertama karena nilai dari Go dan CURRENT_TIMESTAMP berbeda format.
UPDATE affiliate_commissions SET status = 'batal', alasan_batal = 'santri berhenti dalam masa tahan', updated_at = sqlc.arg(now)
WHERE status = 'pending'
  AND EXISTS (SELECT 1 FROM leads l JOIN santri s ON s.id = l.santri_id
      JOIN santri_status_log sl ON sl.santri_id = s.id
      WHERE l.id = affiliate_commissions.lead_id AND s.status IN ('nonaktif', 'tidak_lanjut')
        AND sl.status_baru IN ('nonaktif', 'tidak_lanjut')
        AND substr(sl.created_at, 1, 19) >= substr(affiliate_commissions.created_at, 1, 19));

-- name: MatureCommissions :execrows
UPDATE affiliate_commissions SET status = 'siap', siap_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE status = 'pending' AND created_at <= sqlc.arg(cutoff);

-- name: ListCommissionCandidatesSPP :many
-- Tagihan lunas setelah konversi, untuk lead yang komisi pendaftarannya berpola bulanan.
SELECT t.id AS tagihan_id, t.santri_id, t.nominal, t.created_at AS tagihan_created, l.id AS lead_id, l.converted_at,
       c.affiliate_id, c.jenis, c.nilai, c.maks_bulan
FROM tagihan t
JOIN leads l ON l.santri_id = t.santri_id
JOIN affiliate_commissions c ON c.sumber = 'pendaftaran' AND c.lead_id = l.id AND c.status != 'batal' AND c.pola = 'bulanan'
JOIN affiliates a ON a.id = c.affiliate_id
WHERE t.status = 'lunas' AND l.converted_at IS NOT NULL AND t.created_at >= l.converted_at AND a.status = 'aktif'
  AND NOT EXISTS (SELECT 1 FROM affiliate_commissions x WHERE x.sumber = 'spp' AND x.sumber_id = t.id);

-- name: CountTagihanSinceConversion :one
-- Urutan tagihan sejak konversi (termasuk yang dimaksud), tagihan batal tidak dihitung.
SELECT COUNT(*) FROM tagihan
WHERE santri_id = sqlc.arg(santri_id) AND status != 'batal'
  AND substr(created_at, 1, 19) >= substr(sqlc.arg(converted_at), 1, 19)
  AND (substr(created_at, 1, 19) < substr(sqlc.arg(created_at), 1, 19)
       OR (substr(created_at, 1, 19) = substr(sqlc.arg(created_at), 1, 19) AND id <= sqlc.arg(id)));

-- name: GetAffiliateCommission :one
SELECT * FROM affiliate_commissions WHERE id = ?;

-- name: ListSiapGuruCommissionTotals :many
-- Komisi siap milik affiliate guru yang matang sebelum batas (akhir bulan ujroh).
SELECT a.guru_id, g.nama AS guru_nama, g.status AS guru_status,
       CAST(SUM(c.jumlah) AS INTEGER) AS total, COUNT(*) AS jumlah_komisi
FROM affiliate_commissions c
JOIN affiliates a ON a.id = c.affiliate_id
JOIN guru g ON g.id = a.guru_id
WHERE c.status = 'siap' AND a.tipe = 'guru' AND a.guru_id IS NOT NULL AND c.siap_at < sqlc.arg(cutoff)
GROUP BY a.guru_id, g.nama, g.status;

-- name: MarkGuruCommissionsPaid :execrows
UPDATE affiliate_commissions SET status = 'dibayar', metode_bayar = 'ujroh', ujroh_bulan = sqlc.arg(bulan),
    dibayar_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE status = 'siap' AND siap_at < sqlc.arg(cutoff)
  AND affiliate_id IN (SELECT id FROM affiliates WHERE tipe = 'guru' AND guru_id IS NOT NULL);

-- name: ReopenGuruCommissions :execrows
UPDATE affiliate_commissions SET status = 'siap', metode_bayar = '', ujroh_bulan = '', dibayar_at = NULL, updated_at = sqlc.arg(now)
WHERE status = 'dibayar' AND metode_bayar = 'ujroh' AND ujroh_bulan = sqlc.arg(bulan);

-- name: MarkCommissionTransferred :execrows
UPDATE affiliate_commissions SET status = 'dibayar', metode_bayar = 'transfer', catatan = ?, dibayar_at = ?, dibayar_oleh = ?, updated_at = ?
WHERE affiliate_commissions.id = ? AND status = 'siap'
  AND affiliate_id IN (SELECT id FROM affiliates WHERE tipe = 'eksternal');

-- name: CancelCommission :execrows
UPDATE affiliate_commissions SET status = 'batal', alasan_batal = ?, updated_at = ? WHERE id = ? AND status IN ('pending', 'siap');

-- name: ListAffiliateReferrals :many
SELECT l.id AS lead_id, l.nama, l.wa_name, l.stage, l.status, l.verified_at, l.santri_id, la.dikunci_at, la.sumber,
       COALESCE(p.nama, '') AS program_nama,
       CAST(COALESCE((SELECT SUM(c.jumlah) FROM affiliate_commissions c WHERE c.lead_id = l.id AND c.status != 'batal'), 0) AS INTEGER) AS komisi
FROM lead_affiliates la
JOIN leads l ON l.id = la.lead_id
LEFT JOIN lead_programs p ON p.id = l.program_id
WHERE la.affiliate_id = ?
ORDER BY la.dikunci_at DESC LIMIT 200;

-- name: AffiliateCommissionSums :many
SELECT status, CAST(COALESCE(SUM(jumlah), 0) AS INTEGER) AS total, COUNT(*) AS jumlah
FROM affiliate_commissions WHERE affiliate_id = ? GROUP BY status;

-- name: AffiliateReferralCounts :many
-- Per affiliate: total rujukan, terverifikasi, dan rujukan sejak tanggal.
SELECT la.affiliate_id, COUNT(*) AS daftar,
       CAST(SUM(CASE WHEN l.verified_at IS NOT NULL THEN 1 ELSE 0 END) AS INTEGER) AS santri,
       CAST(SUM(CASE WHEN la.dikunci_at >= sqlc.arg(since) THEN 1 ELSE 0 END) AS INTEGER) AS daftar_baru
FROM lead_affiliates la JOIN leads l ON l.id = la.lead_id
GROUP BY la.affiliate_id;

-- name: AffiliateCommissionTotals :many
SELECT affiliate_id,
       CAST(COALESCE(SUM(CASE WHEN status = 'siap' THEN jumlah END), 0) AS INTEGER) AS siap,
       CAST(COALESCE(SUM(CASE WHEN status = 'pending' THEN jumlah END), 0) AS INTEGER) AS pending,
       CAST(COALESCE(SUM(CASE WHEN status = 'dibayar' THEN jumlah END), 0) AS INTEGER) AS dibayar
FROM affiliate_commissions GROUP BY affiliate_id;

-- name: ListCommissionLedger :many
SELECT c.*, a.nama AS affiliate_nama, a.kode AS affiliate_kode, a.tipe AS affiliate_tipe,
       a.bank_nama, a.bank_rekening, a.bank_atas_nama, l.nama AS lead_nama, l.wa_name AS lead_wa_name
FROM affiliate_commissions c
JOIN affiliates a ON a.id = c.affiliate_id
JOIN leads l ON l.id = c.lead_id
WHERE (sqlc.arg(status) = '' OR c.status = sqlc.arg(status))
ORDER BY c.created_at DESC, c.id DESC LIMIT 300;

-- name: RevokeAffiliateInvites :exec
UPDATE affiliate_invites SET used_at = ? WHERE affiliate_id = ? AND used_at IS NULL;

-- name: ListCommissionsByLead :many
SELECT * FROM affiliate_commissions WHERE lead_id = ? AND status != 'batal';

-- name: DeletePendingCommissionsByLead :execrows
DELETE FROM affiliate_commissions WHERE lead_id = ? AND status IN ('pending', 'batal');
