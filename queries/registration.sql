-- name: CreateRegistrationLink :one
INSERT INTO registration_links (slug, judul, deskripsi, program_id, lead_source_id, mode, after_submit,
    nominal_override, field_config, wa_template, izinkan_manual, is_aktif, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateRegistrationLink :exec
UPDATE registration_links SET slug = ?, judul = ?, deskripsi = ?, program_id = ?, lead_source_id = ?, mode = ?,
    after_submit = ?, nominal_override = ?, field_config = ?, wa_template = ?, izinkan_manual = ?, is_aktif = ?, updated_at = ?
WHERE id = ?;

-- name: GetRegistrationLink :one
SELECT * FROM registration_links WHERE id = ?;

-- name: GetRegistrationLinkBySlug :one
SELECT * FROM registration_links WHERE slug = ?;

-- name: ListRegistrationLinks :many
SELECT * FROM registration_links ORDER BY is_aktif DESC, id DESC;

-- name: BumpRegistrationStat :exec
INSERT INTO registration_link_stats (link_id, tanggal, klik, submit, bayar)
VALUES (sqlc.arg(link_id), sqlc.arg(tanggal), sqlc.arg(klik), sqlc.arg(submit), sqlc.arg(bayar))
ON CONFLICT (link_id, tanggal) DO UPDATE SET
    klik = klik + excluded.klik, submit = submit + excluded.submit, bayar = bayar + excluded.bayar;

-- name: ListRegistrationStatsSince :many
SELECT link_id, CAST(SUM(klik) AS INTEGER) AS klik, CAST(SUM(submit) AS INTEGER) AS submit, CAST(SUM(bayar) AS INTEGER) AS bayar
FROM registration_link_stats WHERE tanggal >= ? GROUP BY link_id;

-- name: CreateLeadRegistration :one
INSERT INTO lead_registrations (lead_id, link_id, ref, payload, ip, user_agent, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetLeadRegistrationByRef :one
SELECT * FROM lead_registrations WHERE ref = ?;

-- name: ListLeadRegistrationsByLead :many
SELECT r.*, l.judul AS link_judul, l.slug AS link_slug, l.mode AS link_mode
FROM lead_registrations r JOIN registration_links l ON l.id = r.link_id
WHERE r.lead_id = ? ORDER BY r.created_at DESC, r.id DESC;

-- name: GetLeadByPhone :one
SELECT * FROM leads WHERE phone = ? ORDER BY updated_at DESC, id DESC LIMIT 1;

-- name: GetRegistrationSettings :one
SELECT cs_wa_number, payment_expiry_hours, bank_manual_info FROM crm_settings WHERE id = 1;

-- name: UpdateRegistrationSettings :exec
UPDATE crm_settings SET cs_wa_number = ?, payment_expiry_hours = ?, bank_manual_info = ?, updated_by = ?, updated_at = ?
WHERE id = 1;
