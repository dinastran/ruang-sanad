-- +goose Up
-- +goose StatementBegin
-- Lead dari sumber ini otomatis masuk program tersebut (NULL = Tahsin Reguler).
ALTER TABLE lead_sources ADD COLUMN program_id INTEGER REFERENCES lead_programs(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE lead_sources DROP COLUMN program_id;
-- +goose StatementEnd
