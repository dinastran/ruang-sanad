package models

// Penanda perhatian riayah. Level menentukan urutan prioritas di daftar.
const (
	PenandaKehadiran = "kehadiran" // merah: alpa berturut / kehadiran rendah
	PenandaKontak    = "kontak"    // oranye: lama tidak dikontak
	PenandaProgres   = "progres"   // kuning: batas materi tidak bergerak

	LevelMerah  = "merah"
	LevelOranye = "oranye"
	LevelKuning = "kuning"
)

type RiayahPenanda struct {
	Kode   string `json:"kode"`
	Level  string `json:"level"`
	Alasan string `json:"alasan"`
}

// RiayahSantriItem adalah satu baris di halaman Riayah Santri.
type RiayahSantriItem struct {
	ID                  int64           `json:"id"`
	Nama                string          `json:"nama"`
	IDMahasantri        string          `json:"id_mahasantri"`
	NoWa                string          `json:"no_wa"`
	KelasID             int64           `json:"kelas_id"`
	NamaKelas           string          `json:"nama_kelas"`
	Level               string          `json:"level"`
	Jadwal              string          `json:"jadwal"`
	GuruNama            string          `json:"guru_nama"`
	PersenHadir30       *float64        `json:"persen_hadir_30"`
	TotalPertemuan30    int64           `json:"total_pertemuan_30"`
	BatasMateriTerakhir string          `json:"batas_materi_terakhir"`
	KontakTerakhir      string          `json:"kontak_terakhir"`
	RaporTerkirim       bool            `json:"rapor_terkirim"`
	Penanda             []RiayahPenanda `json:"penanda"`
}

type RiayahRingkasan struct {
	TotalSantri    int `json:"total_santri"`
	PerluPerhatian int `json:"perlu_perhatian"`
	Kehadiran      int `json:"kehadiran"`
	Kontak         int `json:"kontak"`
	Progres        int `json:"progres"`
	// Rapor bulan lalu yang sudah dikirim, untuk memantau pengiriman rapor.
	RaporPeriode  string `json:"rapor_periode"`
	RaporTerkirim int    `json:"rapor_terkirim"`
}

// RiayahTimelineItem menggabungkan absensi, catatan riayah, dan log kontak
// dalam satu urutan waktu.
type RiayahTimelineItem struct {
	Jenis     string `json:"jenis"` // pertemuan / catatan / kontak
	ID        int64  `json:"id"`
	Tanggal   string `json:"tanggal"`
	Waktu     string `json:"waktu"`
	Judul     string `json:"judul"`
	Status    string `json:"status,omitempty"`
	Isi       string `json:"isi,omitempty"`
	Materi    string `json:"materi,omitempty"`
	Batas     string `json:"batas_materi,omitempty"`
	Media     string `json:"media,omitempty"`
	Penulis   string `json:"penulis,omitempty"`
	BisaHapus bool   `json:"bisa_hapus"`
}

type RiayahSantriProfil struct {
	Santri   RiayahSantriItem     `json:"santri"`
	Domisili string               `json:"domisili"`
	Usia     *int64               `json:"usia"`
	Mulai    string               `json:"mulai_belajar"`
	Rekap    RiayahRekapKehadiran `json:"rekap"`
	Timeline []RiayahTimelineItem `json:"timeline"`
}

type RiayahRekapKehadiran struct {
	Total int64 `json:"total"`
	Hadir int64 `json:"hadir"`
	Telat int64 `json:"telat"`
	Izin  int64 `json:"izin"`
	Sakit int64 `json:"sakit"`
	Alpa  int64 `json:"alpa"`
}

// RiayahViewer menentukan cakupan data dan hak tulis.
// GuruID nil berarti boleh melihat semua santri.
type RiayahViewer struct {
	UserID   int64
	GuruID   *int64
	CanWrite bool
}

type RiayahKontakInput struct {
	Tanggal string `json:"tanggal" form:"tanggal"`
	Media   string `json:"media" form:"media"`
	Jenis   string `json:"jenis" form:"jenis"`
	Periode string `json:"periode" form:"periode"`
	Catatan string `json:"catatan" form:"catatan"`
}

type RiayahWATemplate struct {
	Nama string `json:"nama"`
	Body string `json:"body"`
}

// RiayahRapor adalah bahan rapor bulanan satu santri.
type RiayahRapor struct {
	Periode      string               `json:"periode"`
	PeriodeOpsi  []string             `json:"periode_opsi"`
	Santri       RiayahSantriItem     `json:"santri"`
	Rekap        RiayahRekapKehadiran `json:"rekap"`
	BatasAwal    string               `json:"batas_awal"`
	BatasAkhir   string               `json:"batas_akhir"`
	Pertemuan    []RiayahTimelineItem `json:"pertemuan"`
	TerkirimPada []string             `json:"terkirim_pada"`
	PesanMinimal int                  `json:"pesan_minimal"`
}
