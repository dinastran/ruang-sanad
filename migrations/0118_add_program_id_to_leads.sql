-- +goose Up
-- +goose StatementBegin
-- Program yang dipilih manual untuk lead ini. NULL = ikut program sumbernya,
-- atau Tahsin Reguler bila sumbernya tidak berprogram.
ALTER TABLE leads ADD COLUMN program_id INTEGER REFERENCES lead_programs(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE leads DROP COLUMN program_id;
-- +goose StatementEnd
