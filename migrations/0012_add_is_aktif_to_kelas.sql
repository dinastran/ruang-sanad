-- +goose Up
-- +goose StatementBegin
ALTER TABLE kelas ADD COLUMN is_aktif INTEGER NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE kelas DROP COLUMN is_aktif;
-- +goose StatementEnd
