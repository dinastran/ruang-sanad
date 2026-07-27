-- +goose Up
-- +goose StatementBegin
ALTER TABLE santri ADD COLUMN no_wa TEXT NOT NULL DEFAULT '';
ALTER TABLE santri ADD COLUMN voice_note_url TEXT NOT NULL DEFAULT '';
ALTER TABLE santri ADD COLUMN keterangan_vn TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE santri DROP COLUMN keterangan_vn;
ALTER TABLE santri DROP COLUMN voice_note_url;
ALTER TABLE santri DROP COLUMN no_wa;
-- +goose StatementEnd
