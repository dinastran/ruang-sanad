-- +goose Up
-- +goose StatementBegin
-- Untuk siapa jadwal ini: '' = ikhwan & akhwat (kuota per gender),
-- 'L' = khusus ikhwan, 'P' = khusus akhwat. Jadwal lama tetap ''.
ALTER TABLE jadwal_penawaran ADD COLUMN gender TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jadwal_penawaran DROP COLUMN gender;
-- +goose StatementEnd
