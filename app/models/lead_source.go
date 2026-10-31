package models

// LeadSourceRequest creates or updates a lead source (landing page / iklan).
type LeadSourceRequest struct {
	Nama       string `json:"nama"`
	Keterangan string `json:"keterangan"`
	IsAktif    bool   `json:"is_aktif"`
}

// LeadSourcePatternRequest adds a trigger text to a source.
type LeadSourcePatternRequest struct {
	Teks string `json:"teks"`
}

type LeadSourcePatternResponse struct {
	ID   int64  `json:"id"`
	Teks string `json:"teks"`
}

// LeadSourceResponse is one source with its trigger texts and lead count.
type LeadSourceResponse struct {
	ID         int64                       `json:"id"`
	Nama       string                      `json:"nama"`
	Keterangan string                      `json:"keterangan"`
	IsAktif    bool                        `json:"is_aktif"`
	Patterns   []LeadSourcePatternResponse `json:"patterns"`
	TotalLeads int64                       `json:"total_leads"`
}

// LeadSourceOption is a source for the select on the lead page.
type LeadSourceOption struct {
	ID      int64  `json:"id"`
	Nama    string `json:"nama"`
	IsAktif bool   `json:"is_aktif"`
}

// LeadSourceStat is one source's cohort in the leads dashboard period.
// ID 0 is "Tanpa sumber".
type LeadSourceStat struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Leads int64  `json:"leads"`
	Won   int64  `json:"won"`
	Open  int64  `json:"open"`
}

// LeadSourceSetRequest sets a lead's source by hand; 0 = tanpa sumber.
type LeadSourceSetRequest struct {
	SourceID int64 `json:"source_id"`
}
