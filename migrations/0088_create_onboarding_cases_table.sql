-- +goose Up
-- +goose StatementBegin
-- Only new registrations create cases; no backfill of existing santri.
CREATE TABLE onboarding_cases (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 santri_id INTEGER NOT NULL UNIQUE REFERENCES santri(id) ON DELETE CASCADE,
 stage TEXT NOT NULL DEFAULT 'onboarding_baru' CHECK (stage IN ('onboarding_baru','matching_kelas','waiting_kuota','siap_mulai','masuk_kelas','batal')),
 pic_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
 registered_at DATETIME NOT NULL,
 deadline_at DATETIME NOT NULL,
 actual_started_at DATETIME,
 cancelled_at DATETIME,
 cancellation_reason TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL,
 CHECK (deadline_at > registered_at)
);
CREATE INDEX idx_onboarding_cases_stage ON onboarding_cases(stage);
CREATE INDEX idx_onboarding_cases_deadline ON onboarding_cases(deadline_at);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE onboarding_cases;
-- +goose StatementEnd
