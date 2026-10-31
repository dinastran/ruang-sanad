-- +goose Up
-- +goose StatementBegin
-- Data performa iklan Meta per iklan per hari (Insights API, level=ad,
-- time_increment=1). Disinkron ulang 28 hari terakhir setiap hari karena
-- Meta masih bisa merevisi angkanya. spend dalam rupiah (mata uang akun).
CREATE TABLE IF NOT EXISTS meta_ad_insights_daily (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tanggal TEXT NOT NULL,
    ad_account_id TEXT NOT NULL,
    campaign_id TEXT NOT NULL,
    campaign_name TEXT NOT NULL DEFAULT '',
    adset_id TEXT NOT NULL DEFAULT '',
    adset_name TEXT NOT NULL DEFAULT '',
    ad_id TEXT NOT NULL,
    ad_name TEXT NOT NULL DEFAULT '',
    spend INTEGER NOT NULL DEFAULT 0,
    impressions INTEGER NOT NULL DEFAULT 0,
    reach INTEGER NOT NULL DEFAULT 0,
    clicks INTEGER NOT NULL DEFAULT 0,
    link_clicks INTEGER NOT NULL DEFAULT 0,
    messaging_started INTEGER NOT NULL DEFAULT 0,
    synced_at DATETIME NOT NULL,
    UNIQUE (tanggal, ad_id)
);
CREATE INDEX IF NOT EXISTS idx_meta_ad_insights_tanggal ON meta_ad_insights_daily(tanggal);
CREATE INDEX IF NOT EXISTS idx_meta_ad_insights_campaign ON meta_ad_insights_daily(campaign_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS meta_ad_insights_daily;
-- +goose StatementEnd
