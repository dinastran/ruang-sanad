-- +goose Up
-- +goose StatementBegin
-- Locked totals: total (pokok from meetings) + total_komponen - total_kasbon
-- = diterima, the amount actually transferred to the guru.
ALTER TABLE ujroh_guru ADD COLUMN total_komponen INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ujroh_guru ADD COLUMN total_kasbon INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ujroh_guru ADD COLUMN diterima INTEGER NOT NULL DEFAULT 0;
UPDATE ujroh_guru SET diterima = total;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ujroh_guru DROP COLUMN diterima;
ALTER TABLE ujroh_guru DROP COLUMN total_kasbon;
ALTER TABLE ujroh_guru DROP COLUMN total_komponen;
-- +goose StatementEnd
