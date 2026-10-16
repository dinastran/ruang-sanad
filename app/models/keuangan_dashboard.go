package models

// KeuanganBulan is one month of cash flow and collection figures.
// Kas masuk/keluar are cash basis (tanggal bayar); Terbit* and Kolektibilitas
// are billing basis (tagihan yang terbit di bulan itu).
type KeuanganBulan struct {
	Bulan          string  `json:"bulan"`
	Label          string  `json:"label"`
	KasMasukSpp    int64   `json:"kas_masuk_spp"`
	JumlahBayarSpp int64   `json:"jumlah_bayar_spp"`
	KasMasukProduk int64   `json:"kas_masuk_produk"`
	KasMasuk       int64   `json:"kas_masuk"`
	KasKeluar      int64   `json:"kas_keluar"`
	JumlahUjroh    int64   `json:"jumlah_ujroh_dibayar"`
	KasBersih      int64   `json:"kas_bersih"`
	TerbitJumlah   int64   `json:"terbit_jumlah"`
	TerbitNominal  int64   `json:"terbit_nominal"`
	TerbitLunas    int64   `json:"terbit_lunas_nominal"`
	Kolektibilitas float64 `json:"kolektibilitas"`
}

type KeuanganAgingBucket struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Jumlah  int64  `json:"jumlah"`
	Nominal int64  `json:"nominal"`
}

type KeuanganPrioritasFU struct {
	TagihanResponse
	HariTelat int64 `json:"hari_telat"`
}

type KeuanganPiutangGroup struct {
	Label         string `json:"label"`
	Href          string `json:"href"`
	JumlahSantri  int64  `json:"jumlah_santri"`
	JumlahTagihan int64  `json:"jumlah_tagihan"`
	Nominal       int64  `json:"nominal"`
}

type KeuanganUjrohPanel struct {
	Bulan        string `json:"bulan"`
	Terkunci     bool   `json:"terkunci"`
	TotalUjroh   int64  `json:"total_ujroh"`
	TotalDibayar int64  `json:"total_dibayar"`
	TotalBelum   int64  `json:"total_belum"`
	JumlahGuru   int64  `json:"jumlah_guru"`
	GuruDibayar  int64  `json:"guru_dibayar"`
	// Kewajiban: locked months not yet paid, plus the running month's live
	// estimate while it is still unlocked.
	KewajibanTerkunci     int64 `json:"kewajiban_terkunci"`
	KewajibanTerkunciGuru int64 `json:"kewajiban_terkunci_guru"`
	EstimasiBerjalan      int64 `json:"estimasi_berjalan"`
}

type KeuanganProdukPanel struct {
	JumlahOrder int64 `json:"jumlah_order"`
	JumlahQty   int64 `json:"jumlah_qty"`
	Omzet       int64 `json:"omzet"`
}

type KeuanganDashboard struct {
	Bulan           string                 `json:"bulan"`
	BulanLabel      string                 `json:"bulan_label"`
	BulanSebelum    string                 `json:"bulan_sebelum"`
	BulanBerikut    string                 `json:"bulan_berikut"`
	HariIni         string                 `json:"hari_ini"`
	Sekarang        KeuanganBulan          `json:"sekarang"`
	Sebelumnya      KeuanganBulan          `json:"sebelumnya"`
	Tren            []KeuanganBulan        `json:"tren"`
	Aging           []KeuanganAgingBucket  `json:"aging"`
	TotalPiutang    int64                  `json:"total_piutang"`
	JumlahPiutang   int64                  `json:"jumlah_piutang"`
	JumlahTerlambat int64                  `json:"jumlah_terlambat"`
	Prioritas       []KeuanganPrioritasFU  `json:"prioritas"`
	PiutangKelas    []KeuanganPiutangGroup `json:"piutang_kelas"`
	PiutangAngkatan []KeuanganPiutangGroup `json:"piutang_angkatan"`
	Ujroh           KeuanganUjrohPanel     `json:"ujroh"`
	Produk          KeuanganProdukPanel    `json:"produk"`
}
