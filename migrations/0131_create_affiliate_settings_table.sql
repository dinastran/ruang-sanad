-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS affiliate_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    masa_berlaku_hari INTEGER NOT NULL DEFAULT 60 CHECK (masa_berlaku_hari IN (30, 60, 90)),
    masa_tahan_hari INTEGER NOT NULL DEFAULT 14 CHECK (masa_tahan_hari BETWEEN 0 AND 90),
    link_tujuan_default_id INTEGER REFERENCES registration_links(id) ON DELETE SET NULL,
    komponen_ujroh_id INTEGER REFERENCES ujroh_komponen(id) ON DELETE SET NULL,
    template_bagikan TEXT NOT NULL DEFAULT '',
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO affiliate_settings (id, komponen_ujroh_id)
VALUES (1, (SELECT id FROM ujroh_komponen WHERE nama = 'Komisi Affiliate'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliate_settings;
-- +goose StatementEnd
