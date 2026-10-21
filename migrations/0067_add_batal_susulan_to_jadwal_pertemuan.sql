-- +goose Up
-- +goose StatementBegin
-- alasan_batal: why the guru/koordinator cancelled the session.
-- is_susulan/alasan_susulan: attendance filled in after the scheduled date;
-- the meeting keeps the scheduled date, the label tells the koordinator it
-- was entered late.
ALTER TABLE jadwal_pertemuan ADD COLUMN alasan_batal TEXT NOT NULL DEFAULT '';
ALTER TABLE jadwal_pertemuan ADD COLUMN is_susulan INTEGER NOT NULL DEFAULT 0;
ALTER TABLE jadwal_pertemuan ADD COLUMN alasan_susulan TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jadwal_pertemuan DROP COLUMN alasan_susulan;
ALTER TABLE jadwal_pertemuan DROP COLUMN is_susulan;
ALTER TABLE jadwal_pertemuan DROP COLUMN alasan_batal;
-- +goose StatementEnd
