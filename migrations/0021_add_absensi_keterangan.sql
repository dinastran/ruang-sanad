-- +goose Up
-- +goose StatementBegin
ALTER TABLE pembinaan_absen ADD COLUMN jam_masuk TEXT NOT NULL DEFAULT '';
ALTER TABLE pembinaan_absen ADD COLUMN keterangan TEXT NOT NULL DEFAULT '';
ALTER TABLE rapat_absen ADD COLUMN jam_masuk TEXT NOT NULL DEFAULT '';
ALTER TABLE rapat_absen ADD COLUMN keterangan TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE rapat_absen DROP COLUMN keterangan;
ALTER TABLE rapat_absen DROP COLUMN jam_masuk;
ALTER TABLE pembinaan_absen DROP COLUMN keterangan;
ALTER TABLE pembinaan_absen DROP COLUMN jam_masuk;
-- +goose StatementEnd
