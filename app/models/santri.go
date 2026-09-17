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
	NoWA          string `json:"no_wa"`
	Email         string `json:"email"`
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
	NoWA          string `json:"no_wa"`
	Email         string `json:"email"`
}

type UpdateSantriAdminKelasRequest struct {
	Fu            string `json:"fu"`
	TanggalVn     string `json:"tanggal_vn"`
	HasilVn       string `json:"hasil_vn"`
	MasukGrup     string `json:"masuk_grup"`
	MulaiBelajar  string `json:"mulai_belajar"`
	Jumlah        int64  `json:"jumlah"`
	AngkatanKelas string `json:"angkatan_kelas"`
	Level         string `json:"level"`
	Jadwal        string `json:"jadwal"`
	Guru          string `json:"guru"`
	GuruID        int64  `json:"guru_id"`
}

type CorrectSantriRegistrationRequest struct {
	Angkatan      string `json:"angkatan"`
	TanggalDaftar string `json:"tanggal_daftar"`
	Konfirmasi    bool   `json:"konfirmasi"`
}

type UpdateSantriKeuanganRequest struct {
	InfaqTerakhir         string `json:"infaq_terakhir"`
	KeteranganTidakLanjut string `json:"keterangan_tidak_lanjut"`
}

type SantriListParams struct {
	AngkatanPendaftaran string
	AngkatanKelas       string
	Level               string
	Tipe                string
	Jadwal              string
	Gender              string
	Status              string
	KelasID             int64
	Lengkap             int64 // -1 = any, 0 = belum, 1 = lengkap
	IDBermasalah        int64 // 0 = semua, 1 = ID perlu ditinjau
	Search              string
	Offset              int64
	Limit               int64
}

type SantriResponse struct {
	ID                     int64   `json:"id"`
	IDMahasantri           string  `json:"id_mahasantri"`
	IDMahasantriBermasalah bool    `json:"id_mahasantri_bermasalah"`
	IDMahasantriTerbit     bool    `json:"id_mahasantri_terbit"`
	KelasKode              string  `json:"kelas_kode"`
	Nama                   string  `json:"nama"`
	JenisKelamin           string  `json:"jenis_kelamin"`
	Nominal                int64   `json:"nominal"`
	TanggalDaftar          string  `json:"tanggal_daftar"`
	Angkatan               string  `json:"angkatan"`
	AngkatanKelas          string  `json:"angkatan_kelas"`
	Usia                   int64   `json:"usia"`
	Domisili               string  `json:"domisili"`
	NoWA                   string  `json:"no_wa"`
	Email                  string  `json:"email"`
	Fu                     string  `json:"fu"`
	TanggalVn              string  `json:"tanggal_vn"`
	HasilVn                string  `json:"hasil_vn"`
	VoiceNoteURL           string  `json:"voice_note_url"`
	KeteranganVn           string  `json:"keterangan_vn"`
	MasukGrup              string  `json:"masuk_grup"`
	MulaiBelajar           string  `json:"mulai_belajar"`
	Jumlah                 int64   `json:"jumlah"`
	Level                  string  `json:"level"`
	Jadwal                 string  `json:"jadwal"`
	Guru                   string  `json:"guru"`
	JadwalCatatan          string  `json:"jadwal_catatan"`
	InfaqTerakhir          string  `json:"infaq_terakhir"`
	KeteranganTidakLanjut  string  `json:"keterangan_tidak_lanjut"`
	Tipe                   string  `json:"tipe"`
	Frekuensi              string  `json:"frekuensi"`
	IsLengkap              bool    `json:"is_lengkap"`
	KelasID                *int64  `json:"kelas_id"`
	Status                 string  `json:"status"`
	TotalHadir             int64   `json:"total_hadir"`
	TotalIzin              int64   `json:"total_izin"`
	TotalSakit             int64   `json:"total_sakit"`
	TotalAlpa              int64   `json:"total_alpa"`
	TotalTelat             int64   `json:"total_telat"`
	PersenHadir            float64 `json:"persen_hadir"`
	CreatedBy              *int64  `json:"created_by"`
	CreatedAt              string  `json:"created_at"`
	UpdatedAt              string  `json:"updated_at"`
}

type SantriListResponse struct {
	Data  []SantriResponse `json:"data"`
	Total int64            `json:"total"`
}
