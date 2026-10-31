package models

// MetaAdsReport is the "Performa Iklan" page: Meta Ads spend and delivery
// next to CRM leads and santri for the same period. CRM numbers only count
// leads whose source is mapped to at least one Meta campaign.
type MetaAdsReport struct {
	From        string           `json:"from"`
	To          string           `json:"to"`
	Enabled     bool             `json:"enabled"`
	Accounts    []string         `json:"accounts"` // masked
	LastRun     *MetaAdsSyncInfo `json:"last_run"`
	LastSuccess *MetaAdsSyncInfo `json:"last_success"`
	HasMapping  bool             `json:"has_mapping"`

	Totals    MetaAdsTotals       `json:"totals"`
	Daily     []MetaAdsDay        `json:"daily"`
	Campaigns []MetaAdsCampaign   `json:"campaigns"`
	Ads       []MetaAdsAd         `json:"ads"`
	Sources   []MetaAdsSourceStat `json:"sources"`
	// Every campaign ever synced, for the mapping select (not only this period).
	CampaignOptions []MetaAdsCampaignOption `json:"campaign_options"`
	SourceOptions   []LeadSourceOption      `json:"source_options"`
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

type MetaAdsTotals struct {
	Spend       int64 `json:"spend"`
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	LinkClicks  int64 `json:"link_clicks"`
	Messaging   int64 `json:"messaging"` // percakapan WA dimulai menurut Meta
	LeadsAll    int64 `json:"leads_all"` // semua lead CRM yang masuk
	LeadsAds    int64 `json:"leads_ads"` // lead CRM dari sumber yang terhubung ke kampanye
	WonAds      int64 `json:"won_ads"`   // dari LeadsAds yang kini jadi santri
}

type MetaAdsDay struct {
	Day       string `json:"day"`
	Spend     int64  `json:"spend"`
	Messaging int64  `json:"messaging"`
	LeadsAds  int64  `json:"leads_ads"`
}

type MetaAdsCampaign struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Spend       int64  `json:"spend"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	LinkClicks  int64  `json:"link_clicks"`
	Messaging   int64  `json:"messaging"`
	Ads         int    `json:"ads"`
	SourceID    int64  `json:"source_id"`
	SourceNama  string `json:"source_nama"`
}

type MetaAdsAd struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	Spend        int64  `json:"spend"`
	Impressions  int64  `json:"impressions"`
	Clicks       int64  `json:"clicks"`
	LinkClicks   int64  `json:"link_clicks"`
	Messaging    int64  `json:"messaging"`
}

// MetaAdsSourceStat compares a lead source's mapped campaigns with its leads.
type MetaAdsSourceStat struct {
	ID        int64  `json:"id"`
	Nama      string `json:"nama"`
	Campaigns int    `json:"campaigns"`
	Spend     int64  `json:"spend"`
	Leads     int64  `json:"leads"`
	Won       int64  `json:"won"`
}

type MetaAdsCampaignOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LastDate string `json:"last_date"`
	SourceID int64  `json:"source_id"`
}
