package models

type SantriAnalyticsFilters struct {
	DateFrom              string `json:"date_from"`
	DateTo                string `json:"date_to"`
	AngkatanPendaftaran   string `json:"angkatan_pendaftaran"`
	AngkatanKelas         string `json:"angkatan_kelas"`
	Level                 string `json:"level"`
	Tipe                  string `json:"tipe"`
	SegmentAgeBucket      string `json:"segment_age_bucket"`
	SegmentGender         string `json:"segment_gender"`
	SegmentDomisili       string `json:"segment_domisili"`
	SegmentStatus         string `json:"segment_status"`
	SegmentRegistrationMo string `json:"segment_registration_month"`
	Page                  int64  `json:"page"`
	Limit                 int64  `json:"limit"`
}

type SantriAnalyticsSummary struct {
	Total        int64   `json:"total"`
	Aktif        int64   `json:"aktif"`
	Cuti         int64   `json:"cuti"`
	Nonaktif     int64   `json:"nonaktif"`
	TidakLanjut  int64   `json:"tidak_lanjut"`
	ActiveRate   float64 `json:"active_rate"`
	RataRataUsia float64 `json:"rata_rata_usia"`
}

type SantriAnalyticsPoint struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Total int64  `json:"total"`
}

type SantriAnalyticsGrowthPoint struct {
	Month string `json:"month"`
	Total int64  `json:"total"`
}

type SantriAnalyticsCohortStatus struct {
	Angkatan    string `json:"angkatan"`
	Aktif       int64  `json:"aktif"`
	Cuti        int64  `json:"cuti"`
	Nonaktif    int64  `json:"nonaktif"`
	TidakLanjut int64  `json:"tidak_lanjut"`
	Lainnya     int64  `json:"lainnya"`
	Total       int64  `json:"total"`
}

type SantriAnalyticsDataQuality struct {
	Total               int64 `json:"total"`
	UsiaKosong          int64 `json:"usia_kosong"`
	DomisiliKosong      int64 `json:"domisili_kosong"`
	TanggalDaftarKosong int64 `json:"tanggal_daftar_kosong"`
	JenisKelaminKosong  int64 `json:"jenis_kelamin_kosong"`
}

type SantriAnalyticsRow struct {
	ID            int64  `json:"id"`
	IDMahasantri  string `json:"id_mahasantri"`
	Nama          string `json:"nama"`
	Usia          int64  `json:"usia"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Domisili      string `json:"domisili"`
	TanggalDaftar string `json:"tanggal_daftar"`
	Angkatan      string `json:"angkatan"`
	AngkatanKelas string `json:"angkatan_kelas"`
	Level         string `json:"level"`
	Tipe          string `json:"tipe"`
	Status        string `json:"status"`
}

type SantriAnalyticsData struct {
	Summary              SantriAnalyticsSummary        `json:"summary"`
	Growth               []SantriAnalyticsGrowthPoint  `json:"growth"`
	AgeDistribution      []SantriAnalyticsPoint        `json:"age_distribution"`
	GenderDistribution   []SantriAnalyticsPoint        `json:"gender_distribution"`
	DomisiliDistribution []SantriAnalyticsPoint        `json:"domisili_distribution"`
	StatusDistribution   []SantriAnalyticsPoint        `json:"status_distribution"`
	StatusByAngkatan     []SantriAnalyticsCohortStatus `json:"status_by_angkatan"`
	LevelDistribution    []SantriAnalyticsPoint        `json:"level_distribution"`
	TipeDistribution     []SantriAnalyticsPoint        `json:"tipe_distribution"`
	DataQuality          SantriAnalyticsDataQuality    `json:"data_quality"`
	Rows                 []SantriAnalyticsRow          `json:"rows"`
	RowTotal             int64                         `json:"row_total"`
}
