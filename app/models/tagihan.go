package models

type TagihanFilter struct {
	Status        string
	KelasID       int64
	AngkatanKelas string
	GuruID        int64
	Frekuensi     string
	Level         string
	Gender        string
	BulanKe       int64
	TanggalDari   string
	TanggalSampai string
	Search        string
}

type TagihanResponse struct {
	ID            int64  `json:"id"`
	SantriID      int64  `json:"santri_id"`
	SantriNama    string `json:"santri_nama"`
	IDMahasantri  string `json:"id_mahasantri"`
	NoWA          string `json:"no_wa"`
	KelasID       *int64 `json:"kelas_id"`
	KelasNama     string `json:"kelas_nama"`
	GuruNama      string `json:"guru_nama"`
	Angkatan      string `json:"angkatan"`
	AngkatanKelas string `json:"angkatan_kelas"`
	Frekuensi     string `json:"frekuensi"`
	BulanKe       int64  `json:"bulan_ke"`
	PertemuanKe   int64  `json:"pertemuan_ke"`
	Nominal         int64 `json:"nominal"`
	NominalOverride bool  `json:"nominal_override"`
	TanggalTagih    string `json:"tanggal_tagih"`
	JatuhTempo    string `json:"jatuh_tempo"`
	Status        string `json:"status"`
	TanggalBayar  string `json:"tanggal_bayar"`
	Metode        string `json:"metode"`
	Catatan       string `json:"catatan"`
	FuTerakhir    string `json:"fu_terakhir"`
	FuCount       int64  `json:"fu_count"`
}

type TagihanRingkasan struct {
	TotalTagihan      int64   `json:"total_tagihan"`
	NominalTagihan    int64   `json:"nominal_tagihan"`
	TotalLunas        int64   `json:"total_lunas"`
	NominalLunas      int64   `json:"nominal_lunas"`
	TotalBelumBayar   int64   `json:"total_belum_bayar"`
	NominalBelumBayar int64   `json:"nominal_belum_bayar"`
	TotalTerlambat    int64   `json:"total_terlambat"`
	Kolektibilitas    float64 `json:"kolektibilitas"`
}

type MarkTagihanLunasRequest struct {
	TanggalBayar string `json:"tanggal_bayar"`
	Metode       string `json:"metode"`
	Catatan      string `json:"catatan"`
}

type BatalkanTagihanRequest struct {
	Catatan string `json:"catatan"`
}


type UpdateTagihanNominalRequest struct {
	Nominal int64 `json:"nominal"`
}

type FollowUpTagihanRequest struct {
	TemplateID int64  `json:"template_id"`
	Message    string `json:"message"`
}

type TagihanFollowUpLogResponse struct {
	ID           int64  `json:"id"`
	TemplateID   *int64 `json:"template_id,omitempty"`
	TemplateNama string `json:"template_nama"`
	MessageBody  string `json:"message_body"`
	PetugasNama  string `json:"petugas_nama"`
	CreatedAt    string `json:"created_at"`
}

// TagihanFilterRequest is the filter as the Tagihan page holds it (query-string
// values), sent in JSON bodies for bulk actions.
type TagihanFilterRequest struct {
	Status        string `json:"status"`
	Search        string `json:"search"`
	TanggalDari   string `json:"tanggal_dari"`
	TanggalSampai string `json:"tanggal_sampai"`
	KelasID       string `json:"kelas_id"`
	AngkatanKelas string `json:"angkatan_kelas"`
	GuruID        string `json:"guru_id"`
	Frekuensi     string `json:"frekuensi"`
	Level         string `json:"level"`
	Gender        string `json:"gender"`
	BulanKe       string `json:"bulan_ke"`
}

type BatalMassalPreviewRequest struct {
	Filter TagihanFilterRequest `json:"filter"`
}

type BatalMassalRequest struct {
	Filter TagihanFilterRequest `json:"filter"`
	Alasan string               `json:"alasan"`
	// Konfirmasi is the count the user typed from the preview; the request is
	// rejected when the eligible set no longer has exactly this many tagihan.
	Konfirmasi int64 `json:"konfirmasi"`
}

type BatalMassalPreview struct {
	Jumlah        int64 `json:"jumlah"`
	TotalNominal  int64 `json:"total_nominal"`
	DilewatiLunas int64 `json:"dilewati_lunas"`
	DilewatiBatal int64 `json:"dilewati_batal"`
}

type BatalMassalResult struct {
	BatchID      int64 `json:"batch_id"`
	Jumlah       int64 `json:"jumlah"`
	TotalNominal int64 `json:"total_nominal"`
}

type TagihanBatalBatchResponse struct {
	ID                 int64  `json:"id"`
	Alasan             string `json:"alasan"`
	FilterRingkas      string `json:"filter_ringkas"`
	Jumlah             int64  `json:"jumlah"`
	TotalNominal       int64  `json:"total_nominal"`
	DibuatOlehNama     string `json:"dibuat_oleh_nama"`
	CreatedAt          string `json:"created_at"`
	DipulihkanAt       string `json:"dipulihkan_at"`
	DipulihkanOlehNama string `json:"dipulihkan_oleh_nama"`
	JumlahDipulihkan   int64  `json:"jumlah_dipulihkan"`
}
