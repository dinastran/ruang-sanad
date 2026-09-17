-- +goose Up
-- +goose StatementBegin
ALTER TABLE kelas ADD COLUMN pertemuan_terakhir INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE kelas DROP COLUMN pertemuan_terakhir;
-- +goose StatementEnd
