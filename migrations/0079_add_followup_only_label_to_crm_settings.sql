-- +goose Up
-- +goose StatementBegin
-- Rollout bertahap follow-up (PRD 15.2): bila diisi nama label (mis.
-- "uji-followup"), worker hanya mengirim follow-up ke lead berlabel itu.
-- Kosong = semua lead.
ALTER TABLE crm_settings ADD COLUMN followup_only_label TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crm_settings DROP COLUMN followup_only_label;
-- +goose StatementEnd
