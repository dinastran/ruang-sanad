package models

// Kunjungan kelas oleh Koordinator Guru: penilaian rubrik 1-4 per aspek,
// pengiriman hasil ke guru, tanggapan guru, dan tindak lanjut.

type KunjunganRequest struct {
	GuruID            int64  `json:"guru_id"`
	KelasID           int64  `json:"kelas_id"`
	TargetMulai       string `json:"target_mulai"`
	TargetSelesai     string `json:"target_selesai"`
	Tanggal           string `json:"tanggal"`
	Jam               string `json:"jam"`
	Status            string `json:"status"`
	Catatan           string `json:"catatan"`
	NilaiKedisiplinan int64  `json:"nilai_kedisiplinan"`
	NilaiMateri       int64  `json:"nilai_materi"`
	NilaiMetode       int64  `json:"nilai_metode"`
	NilaiInteraksi    int64  `json:"nilai_interaksi"`
}

// KunjunganNilaiAspek is one rubric row. Nilai 0 means not scored yet.
type KunjunganNilaiAspek struct {
	Kode     string `json:"kode"`
	Label    string `json:"label"`
	Nilai    int64  `json:"nilai"`
	Predikat string `json:"predikat"`
}

type KunjunganResponse struct {
	ID                int64                  `json:"id"`
	GuruID            int64                  `json:"guru_id"`
	GuruNama          string                 `json:"guru_nama"`
	KelasID           *int64                 `json:"kelas_id,omitempty"`
	KelasNama         string                 `json:"kelas_nama"`
	TargetMulai       string                 `json:"target_mulai"`
	TargetSelesai     string                 `json:"target_selesai"`
	Tanggal           string                 `json:"tanggal"`
	Jam               string                 `json:"jam"`
	Status            string                 `json:"status"`
	Catatan           string                 `json:"catatan"`
	NilaiKedisiplinan int64                  `json:"nilai_kedisiplinan"`
	NilaiMateri       int64                  `json:"nilai_materi"`
	NilaiMetode       int64                  `json:"nilai_metode"`
	NilaiInteraksi    int64                  `json:"nilai_interaksi"`
	Aspek             []KunjunganNilaiAspek  `json:"aspek"`
	NilaiLengkap      bool                   `json:"nilai_lengkap"`
	NilaiRataRata     float64                `json:"nilai_rata_rata"`
	Predikat          string                 `json:"predikat"`
	StatusKirim       string                 `json:"status_kirim"` // draft/terkirim/dibaca/ditanggapi
	DikirimAt         string                 `json:"dikirim_at"`
	DibacaAt          string                 `json:"dibaca_at"`
	TanggapanGuru     string                 `json:"tanggapan_guru"`
	TanggapanAt       string                 `json:"tanggapan_at"`
	WALink            string                 `json:"wa_link"`
	TindakLanjut      []TindakLanjutResponse `json:"tindak_lanjut"`
}

type TindakLanjutRequest struct {
	Jenis         string `json:"jenis"`
	Catatan       string `json:"catatan"`
	TargetTanggal string `json:"target_tanggal"`
}

type TindakLanjutStatusRequest struct {
	Status string `json:"status"`
}

type TindakLanjutResponse struct {
	ID                    int64  `json:"id"`
	KunjunganID           int64  `json:"kunjungan_id"`
	Jenis                 string `json:"jenis"`
	JenisLabel            string `json:"jenis_label"`
	Internal              bool   `json:"internal"`
	Catatan               string `json:"catatan"`
	TargetTanggal         string `json:"target_tanggal"`
	Status                string `json:"status"`
	SelesaiAt             string `json:"selesai_at"`
	Terlambat             bool   `json:"terlambat"`
	KunjunganBerikutnyaID *int64 `json:"kunjungan_berikutnya_id,omitempty"`
	GuruID                int64  `json:"guru_id,omitempty"`
	GuruNama              string `json:"guru_nama,omitempty"`
	KelasNama             string `json:"kelas_nama,omitempty"`
	KunjunganTanggal      string `json:"kunjungan_tanggal,omitempty"`
}

type KunjunganAspekRataRata struct {
	Kode     string  `json:"kode"`
	Label    string  `json:"label"`
	RataRata float64 `json:"rata_rata"`
}

// KunjunganRekapGuru summarises only results already sent to the guru.
type KunjunganRekapGuru struct {
	GuruID              int64                    `json:"guru_id"`
	GuruNama            string                   `json:"guru_nama"`
	JumlahKunjungan     int                      `json:"jumlah_kunjungan"`
	TanggalTerakhir     string                   `json:"tanggal_terakhir"`
	NilaiTerakhir       float64                  `json:"nilai_terakhir"`
	PredikatTerakhir    string                   `json:"predikat_terakhir"`
	NilaiSebelumnya     float64                  `json:"nilai_sebelumnya"`
	Tren                string                   `json:"tren"` // naik/turun/tetap/"" (belum ada pembanding)
	RataRataAspek       []KunjunganAspekRataRata `json:"rata_rata_aspek"`
	TindakLanjutTerbuka int                      `json:"tindak_lanjut_terbuka"`
}

type KunjunganDashboard struct {
	Kunjungan           []KunjunganResponse    `json:"kunjungan"`
	Rekap               []KunjunganRekapGuru   `json:"rekap"`
	TindakLanjutTerbuka []TindakLanjutResponse `json:"tindak_lanjut_terbuka"`
	BelumDitanggapi     []KunjunganResponse    `json:"belum_ditanggapi"`
}

type TanggapanKunjunganRequest struct {
	Tanggapan string `json:"tanggapan"`
}
