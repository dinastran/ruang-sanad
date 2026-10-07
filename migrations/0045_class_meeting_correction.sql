-- +goose Up
-- +goose StatementBegin
-- Audit reason for class meeting-number corrections.
ALTER TABLE kelas_perubahan ADD COLUMN alasan TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE kelas_perubahan DROP COLUMN alasan;
-- +goose StatementEnd
