package models

// MetaAdsReport feeds the "Performa Iklan" dashboard. It carries raw daily
// rows (per ad, and CRM leads per source) for the selected period and the
// period of the same length before it; the page filters and aggregates them,
// so every filter moves every number, chart and table at once.
type MetaAdsReport struct {
	From        string           `json:"from"`
	To          string           `json:"to"`
	PrevFrom    string           `json:"prev_from"`
	PrevTo      string           `json:"prev_to"`
	Enabled     bool             `json:"enabled"`
	Accounts    []string         `json:"accounts"` // masked
	LastRun     *MetaAdsSyncInfo `json:"last_run"`
	LastSuccess *MetaAdsSyncInfo `json:"last_success"`

	Rows      []MetaAdsRow          `json:"rows"`
	Ads       []MetaAdsAdInfo       `json:"ads"`
	Campaigns []MetaAdsCampaignInfo `json:"campaigns"` // every campaign ever synced
	LeadDays  []MetaAdsLeadDay      `json:"lead_days"`
	Sources   []LeadSourceOption    `json:"sources"`
}

// MetaAdsRow is one ad on one day. Short JSON keys keep the payload small.
type MetaAdsRow struct {
	Day         string `json:"d"`
	AdID        string `json:"ad"`
	AdsetID     string `json:"as"`
	CampaignID  string `json:"c"`
	Spend       int64  `json:"sp"`
	Impressions int64  `json:"im"`
	Clicks      int64  `json:"cl"`
	LinkClicks  int64  `json:"lk"`
	Messaging   int64  `json:"ms"`
}

type MetaAdsAdInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AdsetID    string `json:"adset_id"`
	AdsetName  string `json:"adset_name"`
	CampaignID string `json:"campaign_id"`
}

type MetaAdsCampaignInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LastDate string `json:"last_date"`
	SourceID int64  `json:"source_id"` // 0 = not mapped
}

// MetaAdsLeadDay counts CRM leads that came in on a day from one source
// (0 = no source), and how many of them are santri now.
type MetaAdsLeadDay struct {
	Day      string `json:"d"`
	SourceID int64  `json:"s"`
	Leads    int64  `json:"l"`
	Won      int64  `json:"w"`
}

type MetaAdsSyncInfo struct {
	Trigger    string `json:"trigger"`
	Status     string `json:"status"`
	Rows       int64  `json:"rows"`
	Error      string `json:"error"`
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
}
