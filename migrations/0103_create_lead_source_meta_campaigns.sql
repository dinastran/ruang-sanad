-- +goose Up
-- +goose StatementBegin
-- Pemetaan kampanye Meta ke sumber lead (LP/iklan) agar biaya iklan bisa
-- dibandingkan dengan lead dan santri dari sumber itu. Satu kampanye hanya
-- ke satu sumber; satu sumber boleh punya banyak kampanye.
CREATE TABLE IF NOT EXISTS lead_source_meta_campaigns (
    campaign_id TEXT PRIMARY KEY,
    lead_source_id INTEGER NOT NULL REFERENCES lead_sources(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lead_source_meta_campaigns_source ON lead_source_meta_campaigns(lead_source_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_source_meta_campaigns;
-- +goose StatementEnd
