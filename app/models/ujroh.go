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
	GuruID          int64            `json:"guru_id"`
	GuruNama        string           `json:"guru_nama"`
	GuruStatus      string           `json:"guru_status"`
	Tarif           int64            `json:"tarif"`
	TarifKhusus     bool             `json:"tarif_khusus"`
	JumlahPertemuan int64            `json:"jumlah_pertemuan"`
	JumlahBadal     int64            `json:"jumlah_badal"`
	Total           int64            `json:"total"`
	Dibayar         bool             `json:"dibayar"`
	DibayarAt       string           `json:"dibayar_at"`
	DibayarOleh     string           `json:"dibayar_oleh"`
	CatatanBayar    string           `json:"catatan_bayar"`
	Selisih         bool             `json:"selisih"`
	LiveJumlah      int64            `json:"live_jumlah"`
	LiveTotal       int64            `json:"live_total"`
	Pertemuan       []UjrohPertemuan `json:"pertemuan"`
}

type UjrohRingkasan struct {
	TotalUjroh      int64 `json:"total_ujroh"`
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
