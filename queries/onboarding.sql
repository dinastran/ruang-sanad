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
