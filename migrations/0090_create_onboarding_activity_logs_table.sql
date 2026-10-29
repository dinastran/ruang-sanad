-- +goose Up
-- +goose StatementBegin
CREATE TABLE onboarding_activity_logs (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 case_id INTEGER NOT NULL REFERENCES onboarding_cases(id) ON DELETE CASCADE,
 actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
 event_type TEXT NOT NULL,
 from_stage TEXT NOT NULL DEFAULT '',
 to_stage TEXT NOT NULL DEFAULT '',
 note TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL
);
CREATE INDEX idx_onboarding_activity_logs_case ON onboarding_activity_logs(case_id, created_at DESC);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE onboarding_activity_logs;
-- +goose StatementEnd
