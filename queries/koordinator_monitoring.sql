-- name: ListClassMonitoringSchedules :many
SELECT
    jp.id AS schedule_id,
    jp.kelas_id,
    k.nama_kelas,
    k.angkatan,
    k.level,
    k.frekuensi,
    k.jadwal AS jadwal_kelas,
    k.materi_individual,
    jp.tanggal,
    jp.jam_mulai,
    jp.catatan AS schedule_note,
    jp.status AS schedule_status,
    jp.is_reschedule,
    jp.jadwal_semula,
    jp.alasan_reschedule,
    k.guru_id AS guru_utama_id,
    COALESCE(gu.nama, '') AS guru_utama_nama,
    jp.guru_pengganti_id,
    COALESCE(gb.nama, '') AS guru_pengganti_nama,
    jp.alasan_badal,
    ua.id AS assigned_user_id,
    jp.pertemuan_id,
    COALESCE(p.status, '') AS meeting_status,
    COALESCE(p.jam_mulai, '') AS actual_start_time,
    COALESCE(p.jam_selesai, '') AS actual_end_time,
    COALESCE(p.materi, '') AS materi,
    COALESCE(p.catatan, '') AS meeting_note,
    p.created_at AS teacher_checked_in_at,
    -- Once attendance is recorded, judge completeness against the roster at
    -- meeting time (the absensi rows), not today's roster, so later enrolment
    -- or status changes don't reopen finished meetings.
    CAST(CASE
        WHEN EXISTS (SELECT 1 FROM absensi a WHERE a.pertemuan_id = jp.pertemuan_id)
            THEN (SELECT COUNT(*) FROM absensi a WHERE a.pertemuan_id = jp.pertemuan_id)
        ELSE (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'aktif')
    END AS INTEGER) AS active_student_count,
    (SELECT COUNT(*) FROM absensi a WHERE a.pertemuan_id = jp.pertemuan_id) AS attendance_count
FROM jadwal_pertemuan jp
JOIN kelas k ON k.id = jp.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = jp.guru_pengganti_id
LEFT JOIN users ua ON ua.id = CASE WHEN jp.guru_pengganti_id IS NOT NULL THEN gb.user_id ELSE gu.user_id END
    AND ua.role IN ('guru', 'super_admin')
LEFT JOIN pertemuan p ON p.id = jp.pertemuan_id
WHERE substr(jp.tanggal, 1, 10) >= substr(sqlc.arg(start_date), 1, 10)
  AND substr(jp.tanggal, 1, 10) <= substr(sqlc.arg(end_date), 1, 10)
ORDER BY jp.tanggal, jp.jam_mulai, k.nama_kelas, jp.id;

-- name: ListUnscheduledMonitoringMeetings :many
-- Pertemuan yang dimulai langsung oleh guru tanpa jadwal_pertemuan terkait.
SELECT
    p.id AS pertemuan_id,
    p.kelas_id,
    k.nama_kelas,
    k.angkatan,
    k.level,
    k.frekuensi,
    k.jadwal AS jadwal_kelas,
    k.materi_individual,
    p.tanggal,
    p.jam_mulai,
    p.jam_selesai,
    p.status AS meeting_status,
    p.materi,
    p.catatan AS meeting_note,
    p.created_at AS teacher_checked_in_at,
    k.guru_id AS guru_utama_id,
    COALESCE(gu.nama, '') AS guru_utama_nama,
    p.guru_pengganti_id,
    COALESCE(gb.nama, '') AS guru_pengganti_nama,
    p.alasan_badal,
    CAST(CASE
        WHEN EXISTS (SELECT 1 FROM absensi a WHERE a.pertemuan_id = p.id)
            THEN (SELECT COUNT(*) FROM absensi a WHERE a.pertemuan_id = p.id)
        ELSE (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'aktif')
    END AS INTEGER) AS active_student_count,
    (SELECT COUNT(*) FROM absensi a WHERE a.pertemuan_id = p.id) AS attendance_count
FROM pertemuan p
JOIN kelas k ON k.id = p.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = p.guru_pengganti_id
WHERE substr(p.tanggal, 1, 10) >= substr(sqlc.arg(start_date), 1, 10)
  AND substr(p.tanggal, 1, 10) <= substr(sqlc.arg(end_date), 1, 10)
  AND NOT EXISTS (SELECT 1 FROM jadwal_pertemuan jp WHERE jp.pertemuan_id = p.id)
ORDER BY p.tanggal, p.jam_mulai, k.nama_kelas, p.id;

-- name: GetMonitoringReminderTarget :one
SELECT
    jp.id AS schedule_id,
    jp.kelas_id,
    k.nama_kelas,
    jp.tanggal,
    jp.jam_mulai,
    jp.status,
    COALESCE(gb.nama, gu.nama, '') AS assigned_teacher_name,
    ua.id AS assigned_user_id
FROM jadwal_pertemuan jp
JOIN kelas k ON k.id = jp.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = jp.guru_pengganti_id
LEFT JOIN users ua ON ua.id = CASE WHEN jp.guru_pengganti_id IS NOT NULL THEN gb.user_id ELSE gu.user_id END
    AND ua.role IN ('guru', 'super_admin')
WHERE jp.id = ? AND jp.kelas_id = ?;

-- name: CountRecentScheduleReminders :one
SELECT COUNT(*)
FROM notifications
WHERE reference_type = 'jadwal_pertemuan'
  AND reference_id = sqlc.arg(schedule_id)
  AND type = sqlc.arg(kind)
  AND created_at >= datetime('now', '-5 minutes');

-- name: ListMonitoringAttendanceByMeeting :many
SELECT
    s.id AS santri_id,
    s.nama AS santri_nama,
    s.id_mahasantri,
    COALESCE(a.status, 'belum_diisi') AS attendance_status,
    COALESCE(a.catatan, '') AS attendance_note,
    COALESCE(a.batas_materi, '') AS batas_materi
FROM santri s
LEFT JOIN absensi a ON a.santri_id = s.id AND a.pertemuan_id = sqlc.arg(pertemuan_id)
-- Recorded attendance keeps the roster as it was at the meeting; before any
-- attendance exists, fall back to the current active roster.
WHERE a.id IS NOT NULL
   OR (
        s.kelas_id = sqlc.arg(kelas_id) AND s.status = 'aktif'
        AND NOT EXISTS (SELECT 1 FROM absensi x WHERE x.pertemuan_id = sqlc.arg(pertemuan_id))
   )
ORDER BY s.nama;

-- name: CreateClassMonitoringNote :one
INSERT INTO class_monitoring_notes (jadwal_pertemuan_id, note, created_by)
VALUES (?, ?, ?)
RETURNING id;

-- name: ListClassMonitoringNotes :many
SELECT
    n.id,
    n.jadwal_pertemuan_id,
    n.note,
    n.created_at,
    COALESCE(u.name, 'Sistem') AS author_name
FROM class_monitoring_notes n
LEFT JOIN users u ON u.id = n.created_by
WHERE n.jadwal_pertemuan_id = ?
ORDER BY n.created_at DESC, n.id DESC;

-- name: CreateScheduleActivityLog :exec
INSERT INTO schedule_activity_logs (jadwal_pertemuan_id, action, details, created_by)
VALUES (?, ?, ?, ?);

-- name: ListScheduleActivityLogs :many
SELECT
    l.id,
    l.jadwal_pertemuan_id,
    l.action,
    l.details,
    l.created_at,
    COALESCE(u.name, 'Sistem') AS actor_name
FROM schedule_activity_logs l
LEFT JOIN users u ON u.id = l.created_by
WHERE l.jadwal_pertemuan_id = ?
ORDER BY l.created_at DESC, l.id DESC;

-- name: CreateNotification :one
INSERT INTO notifications (
    user_id, type, title, message, action_url,
    reference_type, reference_id, created_by
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: ListNotificationsForUser :many
SELECT id, type, title, message, action_url, read_at, created_at
FROM notifications
WHERE user_id = ?
ORDER BY (read_at IS NULL) DESC, created_at DESC, id DESC
LIMIT 20;

-- name: MarkNotificationRead :execrows
UPDATE notifications
SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP)
WHERE id = ? AND user_id = ?;
