-- name: DeleteMetaAdInsightsRange :exec
-- Hapus data satu akun dalam rentang yang baru disinkron ulang, agar iklan
-- yang tidak muncul lagi di Meta ikut hilang.
DELETE FROM meta_ad_insights_daily WHERE ad_account_id = ? AND tanggal >= ? AND tanggal <= ?;

-- name: UpsertMetaAdInsight :exec
INSERT INTO meta_ad_insights_daily (tanggal, ad_account_id, campaign_id, campaign_name, adset_id, adset_name,
    ad_id, ad_name, spend, impressions, reach, clicks, link_clicks, messaging_started, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (tanggal, ad_id) DO UPDATE SET
    ad_account_id = excluded.ad_account_id, campaign_id = excluded.campaign_id, campaign_name = excluded.campaign_name,
    adset_id = excluded.adset_id, adset_name = excluded.adset_name, ad_name = excluded.ad_name,
    spend = excluded.spend, impressions = excluded.impressions, reach = excluded.reach, clicks = excluded.clicks,
    link_clicks = excluded.link_clicks, messaging_started = excluded.messaging_started, synced_at = excluded.synced_at;

-- name: CreateMetaAdsSyncRun :one
INSERT INTO meta_ads_sync_runs (trigger, date_from, date_to, status, started_at)
VALUES (?, ?, ?, 'running', ?)
RETURNING id;

-- name: FinishMetaAdsSyncRun :exec
UPDATE meta_ads_sync_runs SET status = ?, rows = ?, error = ?, finished_at = ? WHERE id = ?;

-- name: GetLastMetaAdsSyncRun :one
SELECT * FROM meta_ads_sync_runs ORDER BY id DESC LIMIT 1;

-- name: GetLastSuccessfulMetaAdsSyncRun :one
SELECT * FROM meta_ads_sync_runs WHERE status = 'success' ORDER BY id DESC LIMIT 1;

-- name: ListMetaCampaigns :many
-- Semua kampanye yang pernah tersinkron (untuk pemetaan ke sumber lead).
SELECT campaign_id, CAST(MAX(campaign_name) AS TEXT) AS campaign_name, CAST(MAX(tanggal) AS TEXT) AS last_date
FROM meta_ad_insights_daily GROUP BY campaign_id ORDER BY 3 DESC, 2;

-- name: ListLeadSourceMetaCampaigns :many
SELECT * FROM lead_source_meta_campaigns;

-- name: SetLeadSourceMetaCampaign :exec
INSERT INTO lead_source_meta_campaigns (campaign_id, lead_source_id, created_at) VALUES (?, ?, ?)
ON CONFLICT (campaign_id) DO UPDATE SET lead_source_id = excluded.lead_source_id;

-- name: DeleteLeadSourceMetaCampaign :exec
DELETE FROM lead_source_meta_campaigns WHERE campaign_id = ?;

-- name: ListLeadSourceRows :many
-- Lead yang masuk dalam rentang, untuk menghitung lead/santri per sumber dan per hari.
SELECT id, lead_source_id, status, created_at FROM leads WHERE created_at >= ? AND created_at < ?;

-- name: MetaAdsRows :many
-- Baris harian per iklan untuk dashboard (filter dan agregasi di browser).
SELECT tanggal, ad_id, adset_id, campaign_id, spend, impressions, clicks, link_clicks, messaging_started
FROM meta_ad_insights_daily WHERE tanggal >= ? AND tanggal <= ?
ORDER BY tanggal, ad_id;

-- name: MetaAdNames :many
-- Nama terbaru setiap iklan beserta ad set dan kampanyenya.
SELECT ad_id, adset_id, campaign_id,
    CAST(MAX(ad_name) AS TEXT) AS ad_name, CAST(MAX(adset_name) AS TEXT) AS adset_name
FROM meta_ad_insights_daily WHERE tanggal >= ? AND tanggal <= ?
GROUP BY ad_id, adset_id, campaign_id;
