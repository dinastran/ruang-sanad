-- +goose Up
-- +goose StatementBegin
ALTER TABLE registration_links ADD COLUMN tampil_kode_referal INTEGER NOT NULL DEFAULT 1 CHECK (tampil_kode_referal IN (0, 1));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE registration_links DROP COLUMN tampil_kode_referal;
-- +goose StatementEnd
