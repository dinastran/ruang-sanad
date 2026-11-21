-- +goose Up
-- +goose StatementBegin
-- hanya_paket = 1: dzikir bawaan yang hanya dibaca lewat paket (mis. Al-Ma'tsurat),
-- tidak ditampilkan di daftar "Dzikir satuan".
ALTER TABLE tasbih_dzikir ADD COLUMN hanya_paket INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasbih_dzikir DROP COLUMN hanya_paket;
-- +goose StatementEnd
