package models

type CreateSantriRequest struct {
	KelasKode     string `json:"kelas_kode"`
	Nama          string `json:"nama"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Nominal       int64  `json:"nominal"`
	TanggalDaftar string `json:"tanggal_daftar"`
	Angkatan      string `json:"angkatan"`
	Usia          int64  `json:"usia"`
	Domisili      string `json:"domisili"`
}

type UpdateSantriCSRequest struct {
	KelasKode     string `json:"kelas_kode"`
	Nama          string `json:"nama"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Nominal       int64  `json:"nominal"`
	TanggalDaftar string `json:"tanggal_daftar"`
	Angkatan      string `json:"angkatan"`
	Usia          int64  `json:"usia"`
	Domisili      string `json:"domisili"`
}

type UpdateSantriAdminKelasRequest struct {
	Fu           string `json:"fu"`
	TanggalVn    string `json:"tanggal_vn"`
	HasilVn      string `json:"hasil_vn"`
	MasukGrup    string `json:"masuk_grup"`
	MulaiBelajar string `json:"mulai_belajar"`
	Jumlah       int64  `json:"jumlah"`
	Level        string `json:"level"`
	Jadwal       string `json:"jadwal"`
	Guru         string `json:"guru"`
}

type UpdateSantriKeuanganRequest struct {
	InfaqTerakhir         string `json:"infaq_terakhir"`
	KeteranganTidakLanjut string `json:"keterangan_tidak_lanjut"`
}

type SantriListParams struct {
	Angkatan string
	Level    string
	Tipe     string
	Jadwal   string
	Gender   string
	Status   string
	KelasID  int64
	Lengkap  int64 // -1 = any, 0 = belum, 1 = lengkap
	Search   string
	Offset   int64
	Limit    int64
}

type SantriResponse struct {
	ID                    int64  `json:"id"`
	IDMahasantri          string `json:"id_mahasantri"`
	KelasKode             string `json:"kelas_kode"`
	Nama                  string `json:"nama"`
	JenisKelamin          string `json:"jenis_kelamin"`
	Nominal               int64  `json:"nominal"`
	TanggalDaftar         string `json:"tanggal_daftar"`
	Angkatan              string `json:"angkatan"`
	Usia                  int64  `json:"usia"`
	Domisili              string `json:"domisili"`
	Fu                    string `json:"fu"`
	TanggalVn             string `json:"tanggal_vn"`
	HasilVn               string `json:"hasil_vn"`
	MasukGrup             string `json:"masuk_grup"`
	MulaiBelajar          string `json:"mulai_belajar"`
	Jumlah                int64  `json:"jumlah"`
	Level                 string `json:"level"`
	Jadwal                string `json:"jadwal"`
	Guru                  string `json:"guru"`
	JadwalCatatan         string `json:"jadwal_catatan"`
	InfaqTerakhir         string `json:"infaq_terakhir"`
	KeteranganTidakLanjut string `json:"keterangan_tidak_lanjut"`
	Tipe                  string `json:"tipe"`
	Frekuensi             string `json:"frekuensi"`
	IsLengkap             bool   `json:"is_lengkap"`
	KelasID               *int64 `json:"kelas_id"`
	Status                string `json:"status"`
	CreatedBy             *int64 `json:"created_by"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type SantriListResponse struct {
	Data  []SantriResponse `json:"data"`
	Total int64            `json:"total"`
}
