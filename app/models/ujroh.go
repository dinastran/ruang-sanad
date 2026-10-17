package models

// UjrohPertemuan is one completed meeting that earns ujroh.
type UjrohPertemuan struct {
	PertemuanID int64  `json:"pertemuan_id"`
	KelasID     int64  `json:"kelas_id"`
	KelasNama   string `json:"kelas_nama"`
	Tanggal     string `json:"tanggal"`
	PertemuanKe int64  `json:"pertemuan_ke"`
	IsBadal     bool   `json:"is_badal"`
	Tarif       int64  `json:"tarif"`
}

// UjrohGuru is one guru's ujroh for a month. When the month is locked the
// numbers come from the snapshot and LiveTotal/LiveJumlah expose the current
// calculation so differences can be flagged without changing paid figures.
type UjrohGuru struct {
	GuruID          int64  `json:"guru_id"`
	GuruNama        string `json:"guru_nama"`
	GuruStatus      string `json:"guru_status"`
	Tarif           int64  `json:"tarif"`
	TarifKhusus     bool   `json:"tarif_khusus"`
	JumlahPertemuan int64  `json:"jumlah_pertemuan"`
	JumlahBadal     int64  `json:"jumlah_badal"`
	// Total is the pokok from meetings. Diterima = Total + TotalKomponen - TotalKasbon.
	Total         int64               `json:"total"`
	TotalKomponen int64               `json:"total_komponen"`
	TotalKasbon   int64               `json:"total_kasbon"`
	Diterima      int64               `json:"diterima"`
	Dibayar       bool                `json:"dibayar"`
	DibayarAt     string              `json:"dibayar_at"`
	DibayarOleh   string              `json:"dibayar_oleh"`
	CatatanBayar  string              `json:"catatan_bayar"`
	Selisih       bool                `json:"selisih"`
	LiveJumlah    int64               `json:"live_jumlah"`
	LiveTotal     int64               `json:"live_total"`
	Pertemuan     []UjrohPertemuan    `json:"pertemuan"`
	Komponen      []UjrohKomponenLine `json:"komponen"`
	Kasbon        []UjrohKasbonLine   `json:"kasbon"`
}

// UjrohKomponenLine is one component paid to a guru in a month.
type UjrohKomponenLine struct {
	KomponenID   int64  `json:"komponen_id"`
	KomponenNama string `json:"komponen_nama"`
	Jenis        string `json:"jenis"`
	Jumlah       int64  `json:"jumlah"`
	Tarif        int64  `json:"tarif"`
	Total        int64  `json:"total"`
	Catatan      string `json:"catatan"`
	// DiubahBulan is true when this month has its own entry (an insidental
	// entry, or a tetap amount adjusted for this month only).
	DiubahBulan  bool  `json:"diubah_bulan"`
	NominalTetap int64 `json:"nominal_tetap"`
}

// UjrohKasbonLine is one kasbon deduction from a guru's ujroh in a month.
type UjrohKasbonLine struct {
	KasbonID    int64  `json:"kasbon_id"`
	Tanggal     string `json:"tanggal"`
	Keterangan  string `json:"keterangan"`
	TotalKasbon int64  `json:"total_kasbon"`
	SisaSebelum int64  `json:"sisa_sebelum"`
	Potongan    int64  `json:"potongan"`
	SisaSetelah int64  `json:"sisa_setelah"`
}

type UjrohRingkasan struct {
	// TotalUjroh is the net amount to transfer (sum of diterima).
	TotalUjroh      int64 `json:"total_ujroh"`
	TotalPokok      int64 `json:"total_pokok"`
	TotalKomponen   int64 `json:"total_komponen"`
	TotalKasbon     int64 `json:"total_kasbon"`
	TotalDibayar    int64 `json:"total_dibayar"`
	TotalBelum      int64 `json:"total_belum"`
	JumlahGuru      int64 `json:"jumlah_guru"`
	JumlahPertemuan int64 `json:"jumlah_pertemuan"`
}

type UjrohRekap struct {
	Bulan       string           `json:"bulan"`
	Terkunci    bool             `json:"terkunci"`
	DikunciAt   string           `json:"dikunci_at"`
	DikunciOleh string           `json:"dikunci_oleh"`
	DibukaAt    string           `json:"dibuka_at"`
	DibukaOleh  string           `json:"dibuka_oleh"`
	Guru        []UjrohGuru      `json:"guru"`
	Ringkasan   UjrohRingkasan   `json:"ringkasan"`
	TanpaGuru   []UjrohPertemuan `json:"tanpa_guru"`
	Berlangsung int64            `json:"berlangsung"`
	// GuruBaruSelisih lists teachers who earn ujroh now but are missing from
	// the locked snapshot (e.g. a meeting completed after locking).
	GuruBaruSelisih []UjrohGuru `json:"guru_baru_selisih"`
}

type UjrohTarif struct {
	Tetap    int64 `json:"tetap"`
	PartTime int64 `json:"part_time"`
}

type UjrohTarifGuru struct {
	GuruID      int64  `json:"guru_id"`
	Nama        string `json:"nama"`
	Status      string `json:"status"`
	IsAktif     bool   `json:"is_aktif"`
	TarifKhusus *int64 `json:"tarif_khusus"`
}

type UpdateUjrohTarifRequest struct {
	Tetap    int64 `json:"tetap"`
	PartTime int64 `json:"part_time"`
}

type UpdateUjrohTarifGuruRequest struct {
	Nominal *int64 `json:"nominal"`
}

type TandaiUjrohDibayarRequest struct {
	Tanggal string `json:"tanggal"`
	Catatan string `json:"catatan"`
}

type UjrohKomponen struct {
	ID             int64  `json:"id"`
	Nama           string `json:"nama"`
	Jenis          string `json:"jenis"`
	NominalDefault int64  `json:"nominal_default"`
	IsAktif        bool   `json:"is_aktif"`
	// Penerima maps guru_id to the recurring amount (tetap components only).
	Penerima map[int64]int64 `json:"penerima"`
}

type SimpanUjrohKomponenRequest struct {
	Nama           string `json:"nama"`
	Jenis          string `json:"jenis"`
	NominalDefault int64  `json:"nominal_default"`
	IsAktif        *bool  `json:"is_aktif"`
}

type SetUjrohKomponenGuruRequest struct {
	Nominal *int64 `json:"nominal"`
}

type SetUjrohKomponenBulanRequest struct {
	Jumlah  int64  `json:"jumlah"`
	Tarif   int64  `json:"tarif"`
	Catatan string `json:"catatan"`
}

type Kasbon struct {
	ID         int64  `json:"id"`
	GuruID     int64  `json:"guru_id"`
	GuruNama   string `json:"guru_nama"`
	Tanggal    string `json:"tanggal"`
	Nominal    int64  `json:"nominal"`
	Cicilan    int64  `json:"cicilan"`
	MulaiBulan string `json:"mulai_bulan"`
	Keterangan string `json:"keterangan"`
	DibuatOleh string `json:"dibuat_oleh"`
	Terpotong  int64  `json:"terpotong"`
	Sisa       int64  `json:"sisa"`
	// Status: berjalan, lunas, dibatalkan.
	Status string `json:"status"`
}

type CreateKasbonRequest struct {
	GuruID     int64  `json:"guru_id"`
	Tanggal    string `json:"tanggal"`
	Nominal    int64  `json:"nominal"`
	Cicilan    int64  `json:"cicilan"`
	MulaiBulan string `json:"mulai_bulan"`
	Keterangan string `json:"keterangan"`
}

type UpdateKasbonRequest struct {
	Cicilan    int64  `json:"cicilan"`
	Keterangan string `json:"keterangan"`
}
