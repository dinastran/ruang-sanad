-- +goose Up
-- +goose StatementBegin
-- One in-app notification per recipient, subject and escalation milestone.
-- Existing notification types are unaffected; no historical notifications are backfilled.
CREATE UNIQUE INDEX idx_onboarding_notifications_dedup
 ON notifications(user_id, type, reference_type, reference_id)
 WHERE type IN (
 'onboarding_vacancy_due', 'onboarding_quota_push', 'onboarding_quota_resolve',
 'onboarding_sla_warning', 'onboarding_sla_overdue'
 ) AND reference_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_onboarding_notifications_dedup;
-- +goose StatementEnd
