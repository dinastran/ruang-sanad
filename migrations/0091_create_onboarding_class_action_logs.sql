-- +goose Up
-- +goose StatementBegin
-- Class operations are auditable independently of individual onboarding cases.
-- No historical rows are backfilled. Existing class controls remain unchanged.
CREATE TABLE onboarding_class_action_logs (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 kelas_id INTEGER NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
 actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
 event_type TEXT NOT NULL CHECK(event_type IN (
  'quota_set','vacancy_sent','cs_followup','marketing_followup',
  'exception_approved','exception_revoked'
 )),
 note TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL
);
CREATE INDEX idx_onboarding_class_action_logs_class
 ON onboarding_class_action_logs(kelas_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE onboarding_class_action_logs;
-- +goose StatementEnd
