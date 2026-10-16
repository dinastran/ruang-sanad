-- +goose Up
-- +goose StatementBegin
-- Kondisi tagihan sebelum dibatalkan massal, dipakai untuk memulihkan batch
-- persis ke status dan catatan semula.
CREATE TABLE IF NOT EXISTS tagihan_batal_batch_item (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id INTEGER NOT NULL REFERENCES tagihan_batal_batch(id) ON DELETE CASCADE,
    tagihan_id INTEGER NOT NULL REFERENCES tagihan(id) ON DELETE CASCADE,
    status_lama TEXT NOT NULL,
    catatan_lama TEXT NOT NULL DEFAULT '',
    UNIQUE(batch_id, tagihan_id)
);
CREATE INDEX IF NOT EXISTS idx_tagihan_batal_batch_item_tagihan ON tagihan_batal_batch_item(tagihan_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tagihan_batal_batch_item;
-- +goose StatementEnd
