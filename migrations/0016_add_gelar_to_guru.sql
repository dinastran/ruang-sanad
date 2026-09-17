-- +goose Up
-- +goose StatementBegin
ALTER TABLE guru ADD COLUMN gelar TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE guru DROP COLUMN gelar;
-- +goose StatementEnd
