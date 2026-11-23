-- +goose Up
-- +goose StatementBegin
-- Sinyal browser terakhir dari form pendaftaran publik, dipakai Meta CAPI
-- untuk mencocokkan event (fbp/fbc, IP, user agent, URL halaman). Satu baris
-- per lead; ditimpa pada submit berikutnya.
CREATE TABLE IF NOT EXISTS lead_web_signals (
    lead_id INTEGER PRIMARY KEY REFERENCES leads(id) ON DELETE CASCADE,
    fbp TEXT NOT NULL DEFAULT '',
    fbc TEXT NOT NULL DEFAULT '',
    client_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_web_signals;
-- +goose StatementEnd
