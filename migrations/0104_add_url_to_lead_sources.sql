-- +goose Up
-- +goose StatementBegin
-- URL landing page sumber lead. Dikirim sebagai event_source_url pada event
-- Meta Conversions API; Meta memblokir event tanpa URL untuk bisnis dengan
-- kategori terbatas.
ALTER TABLE lead_sources ADD COLUMN url TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE lead_sources DROP COLUMN url;
-- +goose StatementEnd
