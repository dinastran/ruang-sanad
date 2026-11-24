-- +goose Up
-- +goose StatementBegin
-- Event Meta Pixel/CAPI per trigger untuk link pendaftaran (JSON:
-- {"wa":[...],"langkah1":[...],"submit":[...],"bukti":[...]}).
-- Kosong = bawaan (sama seperti sebelum kolom ini ada).
ALTER TABLE registration_links ADD COLUMN meta_events TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE registration_links DROP COLUMN meta_events;
-- +goose StatementEnd
