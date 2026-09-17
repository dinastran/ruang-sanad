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
	Nominal       int64  `json:"nominal"`
	TanggalTagih  string `json:"tanggal_tagih"`
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
