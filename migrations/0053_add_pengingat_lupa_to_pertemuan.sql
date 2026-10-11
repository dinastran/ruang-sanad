-- +goose Up
-- +goose StatementBegin
-- Waktu pengingat "lupa ditutup" dikirim untuk pertemuan yang terlalu lama
-- berstatus berlangsung, agar notifikasi tidak dikirim berulang.
ALTER TABLE pertemuan ADD COLUMN pengingat_lupa_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_pertemuan_status ON pertemuan(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pertemuan_status;
ALTER TABLE pertemuan DROP COLUMN pengingat_lupa_at;
-- +goose StatementEnd
