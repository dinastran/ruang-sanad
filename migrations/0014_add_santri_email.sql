-- +goose Up
-- +goose StatementBegin
ALTER TABLE santri ADD COLUMN email TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE santri DROP COLUMN email;
-- +goose StatementEnd
