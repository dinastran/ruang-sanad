-- +goose Up
-- +goose StatementBegin
-- Operational target is distinct from the existing maximum kelas.kapasitas.
CREATE TABLE onboarding_class_controls (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 kelas_id INTEGER NOT NULL UNIQUE REFERENCES kelas(id) ON DELETE CASCADE,
 target_kuota INTEGER NOT NULL CHECK (target_kuota > 0),
 first_candidate_at DATETIME NOT NULL,
 vacancy_alert_sent_at DATETIME,
 exception_approved_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
 exception_approved_at DATETIME,
 exception_reason TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL
);
CREATE INDEX idx_onboarding_class_controls_first_candidate ON onboarding_class_controls(first_candidate_at);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE onboarding_class_controls;
-- +goose StatementEnd
