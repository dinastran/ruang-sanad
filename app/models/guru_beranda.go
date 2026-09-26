package models

// Status slot mengajar di dashboard guru.
const (
	SlotBelum       = "belum"
	SlotBerlangsung = "berlangsung"
	SlotSelesai     = "selesai"
	SlotDibadalkan  = "dibadalkan" // diajar guru pengganti
)

// GuruSlotKelas adalah satu jadwal mengajar pada tanggal tertentu, dari
// jadwal rutin kelas atau dari menu Jadwal Pertemuan (reschedule/badal).
type GuruSlotKelas struct {
	KelasID      int64  `json:"kelas_id"`
	NamaKelas    string `json:"nama_kelas"`
	Level        string `json:"level"`
	Tanggal      string `json:"tanggal"`
	Jam          string `json:"jam"`
	JumlahSantri int64  `json:"jumlah_santri"`
	Sumber       string `json:"sumber"` // rutin / jadwal
	JadwalID     int64  `json:"jadwal_id"`
	JadwalStatus string `json:"jadwal_status"`
	Status       string `json:"status"`
	PertemuanID  int64  `json:"pertemuan_id"`
	Keterangan   string `json:"keterangan"`
	SebagaiBadal bool   `json:"sebagai_badal"`
}

type GuruKelasJadwalTakTerbaca struct {
	KelasID   int64  `json:"kelas_id"`
	NamaKelas string `json:"nama_kelas"`
	Jadwal    string `json:"jadwal"`
}

type GuruKinerja struct {
	TilawahStreak   int      `json:"tilawah_streak"`
	TilawahBulanIni int64    `json:"tilawah_bulan_ini"`
	SudahTilawah    bool     `json:"sudah_tilawah"`
	TsiBulan        string   `json:"tsi_bulan"`
	TsiTotal        *float64 `json:"tsi_total"`
	TsiPredikat     string   `json:"tsi_predikat"`
	PembinaanHadir  int64    `json:"pembinaan_hadir"`
	PembinaanTotal  int64    `json:"pembinaan_total"`
	RapatHadir      int64    `json:"rapat_hadir"`
}

type GuruAgendaItem struct {
	Jenis   string `json:"jenis"` // pembinaan / rapat / kalam
	Tanggal string `json:"tanggal"`
	Judul   string `json:"judul"`
}

// GuruBeranda adalah isi dashboard guru.
type GuruBeranda struct {
	Tanggal          string                      `json:"tanggal"`
	HariIni          []GuruSlotKelas             `json:"hari_ini"`
	Tertunda         []GuruSlotKelas             `json:"tertunda"`
	JadwalTakTerbaca []GuruKelasJadwalTakTerbaca `json:"jadwal_tak_terbaca"`
	Kinerja          *GuruKinerja                `json:"kinerja"`
	Agenda           []GuruAgendaItem            `json:"agenda"`
}
