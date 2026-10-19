package models

import "time"

// AIToken is a personal access token an AI agent (MCP client) uses to read
// dashboard data on behalf of one user. The plaintext is never stored.
type AIToken struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	Nama       string     `json:"nama"`
	Prefix     string     `json:"prefix"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// AITokenAuth is the token row joined with its owner, used to authenticate
// an MCP request. Role is read from users on every request so a role change
// or demotion takes effect immediately.
type AITokenAuth struct {
	TokenID    int64
	UserID     int64
	UserName   string
	Role       string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}

// AIPrincipal is the authenticated caller of an AI agent tool.
type AIPrincipal struct {
	UserID   int64
	TokenID  int64
	UserName string
	Role     string
	// BaseURL is the dashboard origin used to build links in tool results.
	BaseURL string
}

type AIAuditEntry struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name"`
	TokenID   int64     `json:"token_id"`
	Tool      string    `json:"tool"`
	Arguments string    `json:"arguments"`
	Status    string    `json:"status"`
	Pesan     string    `json:"pesan"`
	DurasiMs  int64     `json:"durasi_ms"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Rows returned by AI agent analytic queries -------------------------
// These rows deliberately have no contact columns (no_wa, email, voice note,
// domisili). Free text is still redacted by the service before it leaves.

type AITunggakanSantri struct {
	SantriID      int64  `json:"santri_id"`
	JumlahTagihan int64  `json:"jumlah_tagihan"`
	Nominal       int64  `json:"nominal"`
	HariTelatMaks int64  `json:"hari_telat_maks"`
	FUTerakhir    string `json:"fu_terakhir"`
	FUCount       int64  `json:"fu_count"`
}

type AISantriRingkas struct {
	ID            int64  `json:"id"`
	IDMahasantri  string `json:"id_mahasantri"`
	Nama          string `json:"nama"`
	Status        string `json:"status"`
	Angkatan      string `json:"angkatan"`
	AngkatanKelas string `json:"angkatan_kelas"`
	Level         string `json:"level"`
	Tipe          string `json:"tipe"`
	Frekuensi     string `json:"frekuensi"`
	KelasID       int64  `json:"kelas_id"`
	NamaKelas     string `json:"nama_kelas"`
	GuruNama      string `json:"guru_nama"`
}

type AISantriProfil struct {
	AISantriRingkas
	JenisKelamin  string `json:"jenis_kelamin"`
	Usia          *int64 `json:"usia"`
	TanggalDaftar string `json:"tanggal_daftar"`
	MulaiBelajar  string `json:"mulai_belajar"`
	StatusAlasan  string `json:"status_alasan"`
	CutiMulai     string `json:"cuti_mulai"`
	CutiSelesai   string `json:"cuti_selesai"`
	// Nil for roles that cannot open the santri page (koordinator_guru).
	Nominal        *int64 `json:"nominal_spp,omitempty"`
	IsLengkap      bool   `json:"data_lengkap"`
	KeteranganTL   string `json:"keterangan_tidak_lanjut"`
	JadwalCatatan  string `json:"jadwal_catatan"`
	PertemuanAwal  int64  `json:"pertemuan_awal"`
	DibuatPada     string `json:"dibuat_pada"`
	TerakhirDiubah string `json:"terakhir_diubah"`
}

type AIStatusLog struct {
	Tanggal    string `json:"tanggal"`
	StatusLama string `json:"status_lama"`
	StatusBaru string `json:"status_baru"`
	Alasan     string `json:"alasan"`
}

type AITagihanSantri struct {
	ID           int64  `json:"id"`
	BulanKe      int64  `json:"bulan_ke"`
	Nominal      int64  `json:"nominal"`
	TanggalTagih string `json:"tanggal_tagih"`
	JatuhTempo   string `json:"jatuh_tempo"`
	Status       string `json:"status"`
	TanggalBayar string `json:"tanggal_bayar"`
	HariTelat    int64  `json:"hari_telat"`
	FUCount      int64  `json:"fu_count"`
	FUTerakhir   string `json:"fu_terakhir"`
}

type AICatatan struct {
	Sumber   string `json:"sumber"`
	ID       int64  `json:"id"`
	Tanggal  string `json:"tanggal"`
	Judul    string `json:"judul"`
	Isi      string `json:"isi"`
	Penulis  string `json:"penulis"`
	SantriID int64  `json:"santri_id,omitempty"`
	KelasID  int64  `json:"kelas_id,omitempty"`
	GuruID   int64  `json:"guru_id,omitempty"`
}

// AIProgramBulan is one (dimension value, month) cell of the program trend.
type AIProgramBulan struct {
	Nilai string `json:"nilai"`
	Bulan string `json:"bulan"`
	Angka int64  `json:"angka"`
	// Second measure, e.g. hadir out of Angka meetings, or omzet for products.
	Angka2 int64 `json:"angka2"`
}

type AIProgramSnapshot struct {
	Nilai    string `json:"nilai"`
	Aktif    int64  `json:"aktif"`
	Cuti     int64  `json:"cuti"`
	Berhenti int64  `json:"berhenti"`
	Total    int64  `json:"total"`
}

type AIKelasMetrik struct {
	ID                  int64    `json:"id"`
	NamaKelas           string   `json:"nama_kelas"`
	Angkatan            string   `json:"angkatan"`
	Level               string   `json:"level"`
	Tipe                string   `json:"tipe"`
	Frekuensi           string   `json:"frekuensi"`
	Jadwal              string   `json:"jadwal"`
	GuruID              int64    `json:"guru_id"`
	GuruNama            string   `json:"guru_nama"`
	GuruAktif           bool     `json:"guru_aktif"`
	Kapasitas           int64    `json:"kapasitas"`
	SantriAktif         int64    `json:"santri_aktif"`
	SantriCuti          int64    `json:"santri_cuti"`
	PertemuanSelesai30  int64    `json:"pertemuan_selesai_30_hari"`
	PertemuanBadal30    int64    `json:"pertemuan_badal_30_hari"`
	PertemuanTerakhir   string   `json:"pertemuan_terakhir"`
	PertemuanBelumTutup int64    `json:"pertemuan_belum_ditutup"`
	Absensi30           int64    `json:"absensi_30_hari"`
	Hadir30             int64    `json:"hadir_30_hari"`
	PersenHadir30       *float64 `json:"persen_hadir_30_hari"`
}

type AIPertemuanRingkas struct {
	ID          int64  `json:"id"`
	PertemuanKe int64  `json:"pertemuan_ke"`
	Tanggal     string `json:"tanggal"`
	Status      string `json:"status"`
	Materi      string `json:"materi"`
	Catatan     string `json:"catatan"`
	Badal       bool   `json:"badal"`
	Reschedule  bool   `json:"reschedule"`
	Hadir       int64  `json:"hadir"`
	TotalAbsen  int64  `json:"total_absensi"`
}

type AIGuruMetrik struct {
	ID                  int64  `json:"id"`
	Nama                string `json:"nama"`
	Gelar               string `json:"gelar"`
	Status              string `json:"status"`
	Aktif               bool   `json:"aktif"`
	TanggalGabung       string `json:"tanggal_gabung"`
	KelasAktif          int64  `json:"kelas_aktif"`
	SantriAktif         int64  `json:"santri_aktif"`
	PertemuanSelesai30  int64  `json:"pertemuan_selesai_30_hari"`
	PertemuanBelumTutup int64  `json:"pertemuan_belum_ditutup"`
	TindakLanjutTerbuka int64  `json:"tindak_lanjut_kunjungan_terbuka"`
	TindakLanjutLewat   int64  `json:"tindak_lanjut_lewat_target"`
	KunjunganTerakhir   string `json:"kunjungan_terakhir"`
	PembinaanHadir      int64  `json:"pembinaan_hadir_90_hari"`
	PembinaanTotal      int64  `json:"pembinaan_tercatat_90_hari"`
	RapatHadir          int64  `json:"rapat_hadir_90_hari"`
	RapatTotal          int64  `json:"rapat_tercatat_90_hari"`
}

type AIKunjunganRingkas struct {
	ID             int64  `json:"id"`
	Tanggal        string `json:"tanggal"`
	Status         string `json:"status"`
	KelasID        int64  `json:"kelas_id"`
	NamaKelas      string `json:"nama_kelas"`
	NilaiDisiplin  *int64 `json:"nilai_kedisiplinan"`
	NilaiMateri    *int64 `json:"nilai_materi"`
	NilaiMetode    *int64 `json:"nilai_metode"`
	NilaiInteraksi *int64 `json:"nilai_interaksi"`
	Catatan        string `json:"catatan"`
	TanggapanGuru  string `json:"tanggapan_guru"`
	TLTerbuka      int64  `json:"tindak_lanjut_terbuka"`
}

// AISantriDaftarFilter drives the daftar_santri tool. Zero values mean "no
// filter". TanpaKelasAktif and BelumLengkap match the dashboard overview
// definitions exactly.
type AISantriDaftarFilter struct {
	Status          string
	KelasID         int64
	GuruID          int64
	Level           string
	Tipe            string
	Frekuensi       string
	Angkatan        string
	AngkatanKelas   string
	JenisKelamin    string
	Kata            string
	DaftarDari      string
	DaftarSampai    string
	TanpaKelasAktif bool
	BelumLengkap    bool
	Limit           int64
	Offset          int64
}

type AISantriDaftar struct {
	AISantriRingkas
	JenisKelamin  string `json:"jenis_kelamin"`
	TanggalDaftar string `json:"tanggal_daftar"`
	MulaiBelajar  string `json:"mulai_belajar"`
	KelasAktif    bool   `json:"kelas_aktif"`
	DataLengkap   bool   `json:"data_lengkap"`
}

type AITagihanDaftarFilter struct {
	Status        string
	Terlambat     bool
	SantriID      int64
	KelasID       int64
	AngkatanKelas string
	TagihDari     string
	TagihSampai   string
	HariIni       string
	Limit         int64
	Offset        int64
}

type AITagihanDaftar struct {
	AITagihanSantri
	SantriID     int64  `json:"santri_id"`
	SantriNama   string `json:"santri_nama"`
	IDMahasantri string `json:"id_mahasantri"`
	SantriStatus string `json:"santri_status"`
	NamaKelas    string `json:"kelas"`
	GuruNama     string `json:"guru"`
	Angkatan     string `json:"angkatan_kelas"`
}
