-- +goose Up
-- +goose StatementBegin
-- Maksimal peserta per kelas program pendek (kelas dipisah ikhwan/akhwat).
ALTER TABLE lead_programs ADD COLUMN kapasitas_kelas INTEGER NOT NULL DEFAULT 30 CHECK (kapasitas_kelas BETWEEN 1 AND 500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE lead_programs DROP COLUMN kapasitas_kelas;
-- +goose StatementEnd
