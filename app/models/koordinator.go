package models

// ============ Dashboard ============

type GuruRingkas struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
	NoWa string `json:"no_wa"`
}

type KoordinatorDashboardResponse struct {
	TotalGuru      int64         `json:"total_guru"`
	GuruTetap      int64         `json:"guru_tetap"`
	GuruPartTime   int64         `json:"guru_part_time"`
	TotalKelas     int64         `json:"total_kelas"`
	TotalSantri    int64         `json:"total_santri"`
	GuruBelumAbsen []GuruRingkas `json:"guru_belum_absen"`
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
	JamMasuk   string `json:"jam_masuk"`
	Keterangan string `json:"keterangan"`
	Alasan     string `json:"alasan"`
}

type AbsenInput struct {
	GuruID     int64  `json:"guru_id"`
	Hadir      bool   `json:"hadir"`
	JamMasuk   string `json:"jam_masuk"`
	Keterangan string `json:"keterangan"`
	Alasan     string `json:"alasan"`
}

type RiwayatAbsensiGuru struct {
	GuruID     int64  `json:"guru_id"`
	GuruNama   string `json:"guru_nama"`
	Kegiatan   string `json:"kegiatan"`
	KegiatanID int64  `json:"kegiatan_id"`
	Tanggal    string `json:"tanggal"`
	Judul      string `json:"judul"`
	Hadir      bool   `json:"hadir"`
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

// ============ Kunjungan Kelas ============

type KunjunganRequest struct {
	GuruID        int64  `json:"guru_id"`
	KelasID       int64  `json:"kelas_id"`
	TargetMulai   string `json:"target_mulai"`
	TargetSelesai string `json:"target_selesai"`
	Tanggal       string `json:"tanggal"`
	Jam           string `json:"jam"`
	Status        string `json:"status"`
	Catatan       string `json:"catatan"`
}

type KunjunganResponse struct {
	ID            int64  `json:"id"`
	GuruID        int64  `json:"guru_id"`
	GuruNama      string `json:"guru_nama"`
	KelasID       *int64 `json:"kelas_id,omitempty"`
	KelasNama     string `json:"kelas_nama"`
	TargetMulai   string `json:"target_mulai"`
	TargetSelesai string `json:"target_selesai"`
	Tanggal       string `json:"tanggal"`
	Jam           string `json:"jam"`
	Status        string `json:"status"`
	Catatan       string `json:"catatan"`
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
