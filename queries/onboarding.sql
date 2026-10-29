-- name: CreateOnboardingCase :execresult
INSERT INTO onboarding_cases (santri_id, stage, registered_at, deadline_at, created_at, updated_at)
VALUES (?, 'onboarding_baru', ?, ?, ?, ?);

-- name: CreateOnboardingActivity :exec
INSERT INTO onboarding_activity_logs (case_id, actor_user_id, event_type, to_stage, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListOnboardingCases :many
SELECT oc.id, oc.santri_id, s.nama, s.id_mahasantri, oc.stage, oc.registered_at, oc.deadline_at
FROM onboarding_cases oc
JOIN santri s ON s.id = oc.santri_id
ORDER BY oc.registered_at DESC, oc.id DESC
LIMIT ?;


-- PR 2: Paginated Kanban read model. No parallel class roster:
-- santri.kelas_id remains the single source of truth.
-- name: ListOnboardingBoard :many
SELECT oc.id, oc.santri_id, s.nama, s.id_mahasantri, oc.stage,
       oc.pic_user_id, COALESCE(u.name, '') AS pic_name,
       oc.registered_at, oc.deadline_at, oc.actual_started_at,
       s.kelas_id, COALESCE(k.nama_kelas, '') AS kelas_nama
FROM onboarding_cases oc
JOIN santri s ON s.id = oc.santri_id
LEFT JOIN users u ON u.id = oc.pic_user_id
LEFT JOIN kelas k ON k.id = s.kelas_id
ORDER BY CASE WHEN oc.stage = 'batal' THEN 1 ELSE 0 END, oc.registered_at DESC, oc.id DESC
LIMIT ? OFFSET ?;

-- name: CountOnboardingBoard :one
SELECT COUNT(*) FROM onboarding_cases;

-- name: CountOnboardingStages :many
SELECT stage, COUNT(*) AS total FROM onboarding_cases GROUP BY stage;

-- name: ListOnboardingPICs :many
SELECT id, name FROM users WHERE role IN ('admin_kelas', 'super_admin') ORDER BY name, id;

-- name: GetOnboardingPICRole :one
SELECT role FROM users WHERE id = ?;

-- name: GetOnboardingCaseState :one
SELECT id, santri_id, stage, pic_user_id, registered_at, deadline_at
FROM onboarding_cases WHERE id = ?;

-- name: GetOnboardingQuotaTarget :one
SELECT target_kuota FROM onboarding_class_controls WHERE kelas_id = ?;

-- name: SetOnboardingPIC :execrows
UPDATE onboarding_cases
SET pic_user_id = sqlc.narg(pic_user_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stage NOT IN ('masuk_kelas','batal')
  AND pic_user_id IS sqlc.narg(expected_pic_id);

-- name: SetOnboardingStage :execrows
UPDATE onboarding_cases
SET stage = sqlc.arg(stage), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stage = sqlc.arg(expected_stage);

-- name: FinishOnboardingCase :execrows
UPDATE onboarding_cases
SET stage = 'masuk_kelas', actual_started_at = sqlc.arg(actual_started_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stage = 'siap_mulai';

-- name: CancelOnboardingCase :execrows
UPDATE onboarding_cases
SET stage = 'batal', cancelled_at = sqlc.arg(cancelled_at),
    cancellation_reason = sqlc.arg(cancellation_reason), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stage = sqlc.arg(expected_stage)
  AND stage NOT IN ('masuk_kelas', 'batal');

-- name: LogOnboardingActivity :exec
INSERT INTO onboarding_activity_logs
(case_id, actor_user_id, event_type, from_stage, to_stage, note, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListOnboardingActivity :many
SELECT l.id, l.event_type, l.from_stage, l.to_stage, l.note, l.created_at,
       COALESCE(u.name, 'Sistem') AS actor_name
FROM onboarding_activity_logs l
LEFT JOIN users u ON u.id = l.actor_user_id
WHERE l.case_id = ?
ORDER BY l.id DESC LIMIT 100;
