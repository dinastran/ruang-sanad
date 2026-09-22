-- +goose Up
-- +goose StatementBegin
ALTER TABLE kelas ADD COLUMN materi_individual INTEGER NOT NULL DEFAULT 0;
ALTER TABLE absensi ADD COLUMN batas_materi TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE absensi DROP COLUMN batas_materi;
ALTER TABLE kelas DROP COLUMN materi_individual;
-- +goose StatementEnd
