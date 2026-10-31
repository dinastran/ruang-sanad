-- name: ListMetaCAPICandidates :many
-- Lead yang berubah dalam jendela waktu CAPI (Meta menolak event > 7 hari).
-- Lead "jangan hubungi" dan tanpa nomor tidak dikirim. verified_amount =
-- nominal yang diverifikasi CS (event payment.verified terakhir).
SELECT l.id, l.phone, l.nama, l.domisili, l.jenis_kelamin, l.stage, l.status, l.payment_amount,
    l.created_at, l.payment_info_sent_at, l.proof_received_at, l.converted_at,
    CAST(COALESCE((SELECT json_extract(e.payload, '$.amount') FROM lead_events e
        WHERE e.lead_id = l.id AND e.type = 'payment.verified' ORDER BY e.id DESC LIMIT 1), 0) AS INTEGER) AS verified_amount
FROM leads l
WHERE (l.created_at >= sqlc.arg(since) OR l.updated_at >= sqlc.arg(since))
  AND l.status != 'do_not_contact' AND l.phone != '';

-- name: ListLeadStageReachedTimes :many
-- Waktu pertama setiap lead pindah ke suatu tahap (dari riwayat event).
SELECT lead_id, CAST(json_extract(payload, '$.stage_to') AS TEXT) AS stage_to,
    CAST(MIN(created_at) AS TEXT) AS reached_at
FROM lead_events
WHERE created_at >= sqlc.arg(since) AND json_extract(payload, '$.stage_to') IS NOT NULL
GROUP BY lead_id, stage_to;

-- name: InsertMetaCAPIEvent :execrows
INSERT OR IGNORE INTO meta_capi_events (lead_id, event_name, event_id, event_time, value, test_mode, next_attempt_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListDueMetaCAPIEvents :many
SELECT * FROM meta_capi_events
WHERE status = 'pending' AND next_attempt_at <= sqlc.arg(now)
ORDER BY id LIMIT sqlc.arg(lim);

-- name: MarkMetaCAPIEventSent :exec
UPDATE meta_capi_events SET status = 'sent', attempts = attempts + 1, last_error = '',
    fbtrace_id = ?, sent_at = ? WHERE id = ?;

-- name: MarkMetaCAPIEventRetry :exec
UPDATE meta_capi_events SET attempts = attempts + 1, last_error = ?, next_attempt_at = ? WHERE id = ?;

-- name: MarkMetaCAPIEventDone :exec
-- Akhir tanpa terkirim: status 'failed' atau 'skipped'.
UPDATE meta_capi_events SET status = ?, attempts = attempts + ?, last_error = ? WHERE id = ?;

-- name: RetryFailedMetaCAPIEvents :execrows
UPDATE meta_capi_events SET status = 'pending', attempts = 0, next_attempt_at = sqlc.arg(now)
WHERE status = 'failed' AND event_time >= sqlc.arg(since);

-- name: CountMetaCAPIEventsByStatus :many
SELECT status, COUNT(*) AS total FROM meta_capi_events GROUP BY status;

-- name: ListRecentMetaCAPIEvents :many
SELECT e.id, e.lead_id, e.event_name, e.event_time, e.value, e.status, e.attempts,
    e.last_error, e.test_mode, e.sent_at, e.created_at,
    CAST(COALESCE(NULLIF(l.nama, ''), NULLIF(l.wa_name, ''), 'Lead ' || l.id) AS TEXT) AS lead_nama
FROM meta_capi_events e JOIN leads l ON l.id = e.lead_id
ORDER BY e.id DESC LIMIT ?;

-- name: CountResendableTestMetaCAPIEvents :one
-- Event yang terkirim/gagal saat mode uji dan masih bisa diterima Meta.
SELECT COUNT(*) FROM meta_capi_events
WHERE test_mode = 1 AND status IN ('sent', 'failed') AND event_time >= ?;

-- name: ResendTestMetaCAPIEvents :execrows
-- Antre ulang event mode uji agar terkirim ke produksi (event_id tetap).
UPDATE meta_capi_events SET status = 'pending', test_mode = 0, attempts = 0, last_error = '',
    fbtrace_id = '', sent_at = NULL, next_attempt_at = sqlc.arg(now)
WHERE test_mode = 1 AND status IN ('sent', 'failed') AND event_time >= sqlc.arg(since);
