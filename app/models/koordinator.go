package models

// ============ Dashboard ============

type GuruRingkas struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
	NoWa string `json:"no_wa"`
}

// KoordinatorDashboardResponse holds the daily overview for koordinator guru.
// Kelas hari ini dan pertemuan belum ditutup diambil dari service monitoring
// dan pertemuan, bukan dari sini.
type KoordinatorDashboardResponse struct {
	Tanggal       string                  `json:"tanggal"`
	TotalGuru     int64                   `json:"total_guru"`
	GuruTetap     int64                   `json:"guru_tetap"`
	GuruPartTime  int64                   `json:"guru_part_time"`
	TotalKelas    int64                   `json:"total_kelas"`
	TotalSantri   int64                   `json:"total_santri"`
	TindakLanjut  []DashboardTindakLanjut `json:"tindak_lanjut"`
	Todo          []DashboardTodo         `json:"todo"`
	Agenda        []DashboardAgenda       `json:"agenda"`
	TsiBulan      string                  `json:"tsi_bulan"`
	TsiBelumFinal int64                   `json:"tsi_belum_final"`
}

// DashboardTindakLanjut is an open kunjungan follow-up due today or earlier.
type DashboardTindakLanjut struct {
	ID            int64  `json:"id"`
	KunjunganID   int64  `json:"kunjungan_id"`
	Jenis         string `json:"jenis"`
	Catatan       string `json:"catatan"`
	GuruNama      string `json:"guru_nama"`
	TargetTanggal string `json:"target_tanggal"`
	Terlambat     bool   `json:"terlambat"`
}

// DashboardTodo is an unfinished koordinator to-do due today or earlier.
type DashboardTodo struct {
	ID        int64  `json:"id"`
	Judul     string `json:"judul"`
	Deadline  string `json:"deadline"`
	Pic       string `json:"pic"`
	Status    string `json:"status"`
	Terlambat bool   `json:"terlambat"`
}

// DashboardAgenda is a scheduled pembinaan, rapat, or kunjungan kelas.
type DashboardAgenda struct {
	Jenis   string `json:"jenis"` // pembinaan/rapat/kunjungan
	ID      int64  `json:"id"`
	Tanggal string `json:"tanggal"`
	Jam     string `json:"jam"`
	Judul   string `json:"judul"`
	Href    string `json:"href"`
}

// ============ Kompetensi ============

type KompetensiRequest struct {
	HafalanQuran     int64 `json:"hafalan_quran"`
	HafalanTuhfah    int64 `json:"hafalan_tuhfah"`
	HafalanJazariy   int64 `json:"hafalan_jazariy"`
	HafalanKhaqaniy  int64 `json:"hafalan_khaqaniy"`
	HafalanSyakhawiy int64 `json:"hafalan_syakhawiy"`
	SanadQiroah      int64 `json:"sanad_qiroah"`
	BahasaArabPasif  int64 `json:"bahasa_arab_pasif"`
	BahasaArabAktif  int64 `json:"bahasa_arab_aktif"`
}

type KompetensiResponse struct {
	HafalanQuran     int64 `json:"hafalan_quran"`
	HafalanTuhfah    int64 `json:"hafalan_tuhfah"`
	HafalanJazariy   int64 `json:"hafalan_jazariy"`
	HafalanKhaqaniy  int64 `json:"hafalan_khaqaniy"`
	HafalanSyakhawiy int64 `json:"hafalan_syakhawiy"`
	SanadQiroah      int64 `json:"sanad_qiroah"`
	BahasaArabPasif  int64 `json:"bahasa_arab_pasif"`
	BahasaArabAktif  int64 `json:"bahasa_arab_aktif"`
}

// ============ Absensi umum (pembinaan & rapat) ============

type AbsenRow struct {
	GuruID     int64  `json:"guru_id"`
	Nama       string `json:"nama"`
	Status     string `json:"status"`
	Hadir      bool   `json:"hadir"`
	Kehadiran  string `json:"kehadiran"` // hadir/izin/sakit/alfa; kosong = belum diisi
	JamMasuk   string `json:"jam_masuk"`
	Keterangan string `json:"keterangan"`
	Alasan     string `json:"alasan"`
}

// Status kehadiran absensi pembinaan & rapat.
const (
	KehadiranHadir = "hadir"
	KehadiranIzin  = "izin"
	KehadiranSakit = "sakit"
	KehadiranAlfa  = "alfa"
)

type AbsenInput struct {
	GuruID     int64  `json:"guru_id"`
	Hadir      bool   `json:"hadir"`
	Kehadiran  string `json:"kehadiran"`
	JamMasuk   string `json:"jam_masuk"`
	Keterangan string `json:"keterangan"`
	Alasan     string `json:"alasan"`
}

type LaporanAbsensi struct {
	JenisKegiatan string     `json:"jenis_kegiatan"`
	Judul         string     `json:"judul"`
	Tanggal       string     `json:"tanggal"`
	Keterangan    string     `json:"keterangan"`
	Total         int        `json:"total"`
	Hadir         int        `json:"hadir"`
	TidakHadir    int        `json:"tidak_hadir"`
	Izin          int        `json:"izin"`
	Sakit         int        `json:"sakit"`
	Alfa          int        `json:"alfa"`
	Absen         []AbsenRow `json:"absen"`
}

type RiwayatAbsensiGuru struct {
	GuruID     int64  `json:"guru_id"`
	GuruNama   string `json:"guru_nama"`
	Kegiatan   string `json:"kegiatan"`
	KegiatanID int64  `json:"kegiatan_id"`
	Tanggal    string `json:"tanggal"`
	Judul      string `json:"judul"`
	Hadir      bool   `json:"hadir"`
	Kehadiran  string `json:"kehadiran"`
	JamMasuk   string `json:"jam_masuk"`
	Keterangan string `json:"keterangan"`
	Alasan     string `json:"alasan"`
}

// ============ Pembinaan ============

type PembinaanRequest struct {
	Tanggal    string `json:"tanggal"`
	Bulan      string `json:"bulan"`
	PekanKe    int64  `json:"pekan_ke"`
	Topik      string `json:"topik"`
	Keterangan string `json:"keterangan"`
	Status     string `json:"status"`
}

type PembinaanResponse struct {
	ID         int64  `json:"id"`
	Tanggal    string `json:"tanggal"`
	Bulan      string `json:"bulan"`
	PekanKe    int64  `json:"pekan_ke"`
	Topik      string `json:"topik"`
	Keterangan string `json:"keterangan"`
	Status     string `json:"status"`
}

// ============ Rapat ============

type RapatRequest struct {
	Tanggal string `json:"tanggal"`
	Judul   string `json:"judul"`
	Catatan string `json:"catatan"`
	Status  string `json:"status"`
}

type RapatResponse struct {
	ID      int64  `json:"id"`
	Tanggal string `json:"tanggal"`
	Judul   string `json:"judul"`
	Catatan string `json:"catatan"`
	Status  string `json:"status"`
}

// ============ Kalam Bersanad ============

type KalamRequest struct {
	Tanggal    string `json:"tanggal"`
	Topik      string `json:"topik"`
	Kitab      string `json:"kitab"`
	Keterangan string `json:"keterangan"`
}

type KalamResponse struct {
	ID         int64  `json:"id"`
	Tanggal    string `json:"tanggal"`
	Topik      string `json:"topik"`
	Kitab      string `json:"kitab"`
	Keterangan string `json:"keterangan"`
	TotalShare int64  `json:"total_share"`
}

type KalamShareRow struct {
	GuruID     int64  `json:"guru_id"`
	Nama       string `json:"nama"`
	SudahShare bool   `json:"sudah_share"`
}

// ============ WA Template ============

type WaTemplateRequest struct {
	Nama       string `json:"nama"`
	TargetType string `json:"target_type"`
	Body       string `json:"body"`
	IsAktif    bool   `json:"is_aktif"`
}

type WaTemplateResponse struct {
	ID         int64  `json:"id"`
	Nama       string `json:"nama"`
	TargetType string `json:"target_type"`
	Body       string `json:"body"`
	IsAktif    bool   `json:"is_aktif"`
}

// ============ Tilawah ============

type TilawahStatusResponse struct {
	SudahHariIni bool     `json:"sudah_hari_ini"`
	BulanIni     int64    `json:"bulan_ini"`
	Tanggal      []string `json:"tanggal"`
}

// ============ Todo Koordinator ============

type TodoRequest struct {
	Judul         string `json:"judul"`
	Teknis        string `json:"teknis"`
	Kebutuhan     string `json:"kebutuhan"`
	Deadline      string `json:"deadline"`
	Pic           string `json:"pic"`
	Status        string `json:"status"`
	Recurring     string `json:"recurring"`
	LinkPendukung string `json:"link_pendukung"`
	Catatan       string `json:"catatan"`
}

type TodoResponse struct {
	ID            int64  `json:"id"`
	Judul         string `json:"judul"`
	Teknis        string `json:"teknis"`
	Kebutuhan     string `json:"kebutuhan"`
	Deadline      string `json:"deadline"`
	Pic           string `json:"pic"`
	Status        string `json:"status"`
	Recurring     string `json:"recurring"`
	LinkPendukung string `json:"link_pendukung"`
	Catatan       string `json:"catatan"`
}

// ============ Analisis alfa berturut-turut ============

// AnalisisKehadiranSesi adalah satu sesi terlaksana pada strip riwayat.
type AnalisisKehadiranSesi struct {
	ID      int64  `json:"id"`
	Tanggal string `json:"tanggal"`
	Judul   string `json:"judul"`
}

type AnalisisKehadiranGuru struct {
	GuruID       int64  `json:"guru_id"`
	Nama         string `json:"nama"`
	NoWa         string `json:"no_wa"`
	AlfaBeruntun int    `json:"alfa_beruntun"`
	Level        string `json:"level"` // merah / oranye / "" (aman)
	// Riwayat sejajar dengan AnalisisKehadiran.Sesi; "" = tidak ada catatan absen.
	Riwayat         []string `json:"riwayat"`
	HadirTerakhir   int      `json:"hadir_terakhir"`
	DicatatTerakhir int      `json:"dicatat_terakhir"`
	TerakhirHadir   string   `json:"terakhir_hadir"`
}

type AnalisisKehadiran struct {
	AmbangAlfa    int                     `json:"ambang_alfa"`
	AmbangWaspada int                     `json:"ambang_waspada"`
	Sesi          []AnalisisKehadiranSesi `json:"sesi"`
	Guru          []AnalisisKehadiranGuru `json:"guru"`
	Merah         int                     `json:"merah"`
	Waspada       int                     `json:"waspada"`
	// Persentase hadir dari seluruh catatan absen pada sesi di strip; nil bila belum ada catatan.
	PersenHadir *float64 `json:"persen_hadir"`
}

// RingkasanAlfaBeruntun dipakai kartu dashboard koordinator.
type RingkasanAlfaBeruntun struct {
	AmbangAlfa       int `json:"ambang_alfa"`
	PembinaanMerah   int `json:"pembinaan_merah"`
	PembinaanWaspada int `json:"pembinaan_waspada"`
	RapatMerah       int `json:"rapat_merah"`
	RapatWaspada     int `json:"rapat_waspada"`
}
