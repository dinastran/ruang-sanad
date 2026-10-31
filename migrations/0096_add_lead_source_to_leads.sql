-- +goose Up
-- +goose StatementBegin
-- Sumber lead yang tersimpan permanen. lead_source_by: '' = belum dicek,
-- 'auto' = dicocokkan dari pesan pertama (id NULL = tidak ada yang cocok),
-- 'manual' = diisi CS (tidak ditimpa pencocokan ulang).
ALTER TABLE leads ADD COLUMN lead_source_id INTEGER REFERENCES lead_sources(id) ON DELETE SET NULL;
ALTER TABLE leads ADD COLUMN lead_source_by TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_leads_lead_source ON leads(lead_source_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_leads_lead_source;
ALTER TABLE leads DROP COLUMN lead_source_by;
ALTER TABLE leads DROP COLUMN lead_source_id;
-- +goose StatementEnd
