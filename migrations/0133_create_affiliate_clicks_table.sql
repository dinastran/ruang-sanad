-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS affiliate_clicks (
    affiliate_id INTEGER NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    tanggal TEXT NOT NULL,
    klik INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (affiliate_id, tanggal)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliate_clicks;
-- +goose StatementEnd
