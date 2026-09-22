package models

type GuruDetailResponse struct {
	ID            int64  `json:"id"`
	Nama          string `json:"nama"`
	Gelar         string `json:"gelar"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Status        string `json:"status"`
	NoWa          string `json:"no_wa"`
	Email         string `json:"email"`
	TanggalGabung string `json:"tanggal_gabung"`
	Foto          string `json:"foto"`
	IsAktif       bool   `json:"is_aktif"`
	UserID        *int64 `json:"user_id,omitempty"`
	TotalKelas    int64  `json:"total_kelas"`
	TotalSantri   int64  `json:"total_santri"`
}

type GuruDirectoryResponse struct {
	ID            int64  `json:"id"`
	Nama          string `json:"nama"`
	Gelar         string `json:"gelar"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Status        string `json:"status"`
	NoWa          string `json:"no_wa"`
	Email         string `json:"email"`
	TanggalGabung string `json:"tanggal_gabung"`
	Foto          string `json:"foto"`
	IsAktif       bool   `json:"is_aktif"`
	UserID        *int64 `json:"user_id,omitempty"`
	TotalKelas    int64  `json:"total_kelas"`
	TotalSantri   int64  `json:"total_santri"`
}

type UpdateGuruDirectoryRequest struct {
	Nama          string `json:"nama"`
	Gelar         string `json:"gelar"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Status        string `json:"status"`
	NoWa          string `json:"no_wa"`
	Email         string `json:"email"`
	TanggalGabung string `json:"tanggal_gabung"`
	Foto          string `json:"foto"`
	IsAktif       bool   `json:"is_aktif"`
}

// CreateGuruDirectoryRequest carries the full profile when Koordinator/Super Admin
// registers a new master guru record.
type CreateGuruDirectoryRequest struct {
	Nama          string `json:"nama"`
	Gelar         string `json:"gelar"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Status        string `json:"status"`
	NoWa          string `json:"no_wa"`
	Email         string `json:"email"`
	TanggalGabung string `json:"tanggal_gabung"`
	Foto          string `json:"foto"`
	IsAktif       bool   `json:"is_aktif"`
}

// LinkGuruUserRequest links a login account to a master guru record.
type LinkGuruUserRequest struct {
	UserID int64 `json:"user_id"`
}

// LinkableUserResponse is an account eligible to be linked to a guru record.
type LinkableUserResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type KelasGuruResponse struct {
	ID                int64  `json:"id"`
	GuruID            *int64 `json:"guru_id,omitempty"`
	GuruNama          string `json:"guru_nama"`
	NamaKelas         string `json:"nama_kelas"`
	Level             string `json:"level"`
	Tipe              string `json:"tipe"`
	Frekuensi         string `json:"frekuensi"`
	Jadwal            string `json:"jadwal"`
	JenisKelamin      string `json:"jenis_kelamin"`
	Kapasitas         int64  `json:"kapasitas"`
	JumlahSantri      int64  `json:"jumlah_santri"`
	PertemuanTerakhir string `json:"pertemuan_terakhir"`
	TanggalTerakhir   string `json:"tanggal_terakhir"`
	MateriTerakhir    string `json:"materi_terakhir"`
	MateriIndividual  bool   `json:"materi_individual"`
}

type SantriGuruResponse struct {
	ID                   int64   `json:"id"`
	IDMahasantri         string  `json:"id_mahasantri"`
	Nama                 string  `json:"nama"`
	Usia                 *int64  `json:"usia,omitempty"`
	Domisili             string  `json:"domisili"`
	NoWa                 string  `json:"no_wa"`
	TanggalMulai         string  `json:"tanggal_mulai"`
	PersenHadir          float64 `json:"persen_hadir"`
	TotalHadir           int64   `json:"total_hadir"`
	TotalIzin            int64   `json:"total_izin"`
	TotalSakit           int64   `json:"total_sakit"`
	TotalAlpa            int64   `json:"total_alpa"`
	TotalTelat           int64   `json:"total_telat"`
	TanggalHadirTerakhir string  `json:"tanggal_hadir_terakhir"`
	BatasMateriTerakhir  string  `json:"batas_materi_terakhir"`
}

type PertemuanResponse struct {
	ID               int64  `json:"id"`
	KelasID          int64  `json:"kelas_id"`
	PertemuanKe      int64  `json:"pertemuan_ke"`
	PertemuanLevelKe int64  `json:"pertemuan_level_ke"`
	LevelNama        string `json:"level_nama"`
	Tanggal          string `json:"tanggal"`
	JamMulai         string `json:"jam_mulai"`
	JamSelesai       string `json:"jam_selesai"`
	Materi           string `json:"materi"`
	Catatan          string `json:"catatan"`
	IsReschedule     bool   `json:"is_reschedule"`
	JadwalSemula     string `json:"jadwal_semula"`
	AlasanReschedule string `json:"alasan_reschedule"`
	IsBadal          bool   `json:"is_badal"`
	GuruPenggantiID  *int64 `json:"guru_pengganti_id,omitempty"`
	AlasanBadal      string `json:"alasan_badal"`
	Status           string `json:"status"`
}

type JadwalPertemuanResponse struct {
	ID                 int64  `json:"id"`
	KelasID            int64  `json:"kelas_id"`
	KelasNama          string `json:"kelas_nama"`
	GuruUtamaID        *int64 `json:"guru_utama_id,omitempty"`
	GuruUtamaNama      string `json:"guru_utama_nama"`
	Tanggal            string `json:"tanggal"`
	JamMulai           string `json:"jam_mulai"`
	Catatan            string `json:"catatan"`
	IsReschedule       bool   `json:"is_reschedule"`
	JadwalSemula       string `json:"jadwal_semula"`
	AlasanReschedule   string `json:"alasan_reschedule"`
	GuruPenggantiID    *int64 `json:"guru_pengganti_id,omitempty"`
	GuruPenggantiNama  string `json:"guru_pengganti_nama"`
	AlasanBadal        string `json:"alasan_badal"`
	JadwalKelasBerubah bool   `json:"jadwal_kelas_berubah"`
	Status             string `json:"status"`
	PertemuanID        *int64 `json:"pertemuan_id,omitempty"`
	CanManage          bool   `json:"can_manage"`
	CanStart           bool   `json:"can_start"`
}

type AbsensiResponse struct {
	ID          int64  `json:"id"`
	PertemuanID int64  `json:"pertemuan_id"`
	SantriID    int64  `json:"santri_id"`
	SantriNama  string `json:"santri_nama"`
	Status      string `json:"status"`
	Catatan     string `json:"catatan"`
	BatasMateri string `json:"batas_materi"`
}

type RiayahResponse struct {
	ID          int64  `json:"id"`
	TargetType  string `json:"target_type"`
	TargetID    int64  `json:"target_id"`
	TargetNama  string `json:"target_nama"`
	Catatan     string `json:"catatan"`
	PenulisNama string `json:"penulis_nama"`
	CreatedAt   string `json:"created_at"`
}

type GuruDashboardResponse struct {
	TotalKelas      int64               `json:"total_kelas"`
	TotalSantri     int64               `json:"total_santri"`
	SantriAktif     int64               `json:"santri_aktif"`
	TotalPertemuan  int64               `json:"total_pertemuan"`
	JadwalHariIni   []KelasGuruResponse `json:"jadwal_hari_ini"`
	KelasBelumAbsen []KelasGuruResponse `json:"kelas_belum_absen"`
}

type MulaiPertemuanRequest struct {
	JadwalID    int64  `json:"jadwal_id"`
	JamMulai    string `json:"jam_mulai"`
	Catatan     string `json:"catatan"`
	BatasMateri string `json:"batas_materi"`
}

type BuatJadwalPertemuanRequest struct {
	KelasID  int64  `json:"kelas_id"`
	Tanggal  string `json:"tanggal"`
	JamMulai string `json:"jam_mulai"`
	Catatan  string `json:"catatan"`
}

type SelesaiPertemuanRequest struct {
	Materi  string         `json:"materi"`
	Catatan string         `json:"catatan"`
	Absensi []AbsensiInput `json:"absensi"`
}

type AbsensiInput struct {
	SantriID    int64  `json:"santri_id"`
	Status      string `json:"status"`
	Catatan     string `json:"catatan"`
	BatasMateri string `json:"batas_materi"`
}

type RescheduleRequest struct {
	TanggalBaru string `json:"tanggal_baru"`
	JamBaru     string `json:"jam_baru"`
	Alasan      string `json:"alasan"`
}

type BadalRequest struct {
	GuruPenggantiID int64  `json:"guru_pengganti_id"`
	Alasan          string `json:"alasan"`
}

type RiayahInput struct {
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	Catatan    string `json:"catatan"`
}
