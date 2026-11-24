-- +goose Up
-- +goose StatementBegin
-- Rekening transfer manual terstruktur (JSON: [{"bank","no_rekening","atas_nama"}]),
-- agar halaman bayar bisa menyalin nomor rekening saja. bank_manual_info
-- tetap dipakai sebagai catatan tambahan.
ALTER TABLE crm_settings ADD COLUMN rekening_bank TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crm_settings DROP COLUMN rekening_bank;
-- +goose StatementEnd
