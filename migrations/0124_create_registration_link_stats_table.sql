-- +goose Up
-- +goose StatementBegin
-- Agregat harian per link (tanggal WIB, YYYY-MM-DD). Angka klik indikatif.
CREATE TABLE IF NOT EXISTS registration_link_stats (
    link_id INTEGER NOT NULL REFERENCES registration_links(id) ON DELETE CASCADE,
    tanggal TEXT NOT NULL,
    klik INTEGER NOT NULL DEFAULT 0,
    submit INTEGER NOT NULL DEFAULT 0,
    bayar INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (link_id, tanggal)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS registration_link_stats;
-- +goose StatementEnd
