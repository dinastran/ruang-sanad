package models

// LaporanKeuanganFilter selects the report period (YYYY-MM-DD, inclusive) and
// narrows the per-santri table.
type LaporanKeuanganFilter struct {
	Dari          string `json:"dari"`
	Sampai        string `json:"sampai"`
	Status        string `json:"status"`
	AngkatanKelas string `json:"angkatan_kelas"`
}

type LaporanKeuanganRingkasan struct {
	PemasukanSpp      int64 `json:"pemasukan_spp"`
	JumlahBayarSpp    int64 `json:"jumlah_bayar_spp"`
	PemasukanProduk   int64 `json:"pemasukan_produk"`
	JumlahOrderProduk int64 `json:"jumlah_order_produk"`
	TotalPemasukan    int64 `json:"total_pemasukan"`
	PengeluaranUjroh  int64 `json:"pengeluaran_ujroh"`
	JumlahUjroh       int64 `json:"jumlah_ujroh"`
	PengeluaranKasbon int64 `json:"pengeluaran_kasbon"`
	JumlahKasbon      int64 `json:"jumlah_kasbon"`
	TotalPengeluaran  int64 `json:"total_pengeluaran"`
	Saldo             int64 `json:"saldo"`
}

type LaporanKelompok struct {
	Nama    string `json:"nama"`
	Jumlah  int64  `json:"jumlah"`
	Nominal int64  `json:"nominal"`
}

type LaporanTagihanTerbit struct {
	Jumlah       int64   `json:"jumlah"`
	Nominal      int64   `json:"nominal"`
	JumlahLunas  int64   `json:"jumlah_lunas"`
	NominalLunas int64   `json:"nominal_lunas"`
	JumlahBatal  int64   `json:"jumlah_batal"`
	PersenLunas  float64 `json:"persen_lunas"`
}

// LaporanPiutang is outstanding SPP as of today across all santri.
type LaporanPiutang struct {
	BelumBayar      int64 `json:"belum_bayar"`
	JumlahTagihan   int64 `json:"jumlah_tagihan"`
	Terlambat       int64 `json:"terlambat"`
	JumlahTerlambat int64 `json:"jumlah_terlambat"`
	JumlahSantri    int64 `json:"jumlah_santri"`
}

type LaporanPembayaran struct {
	TagihanID     int64  `json:"tagihan_id"`
	TanggalBayar  string `json:"tanggal_bayar"`
	SantriID      int64  `json:"santri_id"`
	IDMahasantri  string `json:"id_mahasantri"`
	SantriNama    string `json:"santri_nama"`
	AngkatanKelas string `json:"angkatan_kelas"`
	KelasNama     string `json:"kelas_nama"`
	BulanKe       int64  `json:"bulan_ke"`
	Nominal       int64  `json:"nominal"`
	Metode        string `json:"metode"`
	Catatan       string `json:"catatan"`
	DicatatOleh   string `json:"dicatat_oleh"`
}

type LaporanUjrohDibayar struct {
	Bulan     string `json:"bulan"`
	GuruNama  string `json:"guru_nama"`
	Pokok     int64  `json:"pokok"`
	Komponen  int64  `json:"komponen"`
	Kasbon    int64  `json:"kasbon"`
	Diterima  int64  `json:"diterima"`
	DibayarAt string `json:"dibayar_at"`
	Catatan   string `json:"catatan"`
}

type LaporanKasbon struct {
	Tanggal    string `json:"tanggal"`
	GuruNama   string `json:"guru_nama"`
	Nominal    int64  `json:"nominal"`
	Keterangan string `json:"keterangan"`
}

// LaporanPembayaranTerakhir is a santri's latest paid tagihan.
type LaporanPembayaranTerakhir struct {
	Tanggal string `json:"tanggal"`
	BulanKe int64  `json:"bulan_ke"`
	Nominal int64  `json:"nominal"`
	Metode  string `json:"metode"`
}

type LaporanSantri struct {
	SantriID              int64                      `json:"santri_id"`
	IDMahasantri          string                     `json:"id_mahasantri"`
	Nama                  string                     `json:"nama"`
	Angkatan              string                     `json:"angkatan"`
	AngkatanKelas         string                     `json:"angkatan_kelas"`
	KelasNama             string                     `json:"kelas_nama"`
	Status                string                     `json:"status"`
	NominalSpp            int64                      `json:"nominal_spp"`
	KeteranganTidakLanjut string                     `json:"keterangan_tidak_lanjut"`
	TerbitPeriode         int64                      `json:"terbit_periode"`
	DibayarPeriode        int64                      `json:"dibayar_periode"`
	BelumBayar            int64                      `json:"belum_bayar"`
	JumlahBelumBayar      int64                      `json:"jumlah_belum_bayar"`
	Terlambat             int64                      `json:"terlambat"`
	JumlahTerlambat       int64                      `json:"jumlah_terlambat"`
	PembayaranTerakhir    *LaporanPembayaranTerakhir `json:"pembayaran_terakhir"`
	// CatatanInfaqLama is the old manual "infaq terakhir" note, shown only
	// when the santri has no paid tagihan yet.
	CatatanInfaqLama string `json:"catatan_infaq_lama"`
}

type LaporanKeuangan struct {
	Filter         LaporanKeuanganFilter    `json:"filter"`
	Ringkasan      LaporanKeuanganRingkasan `json:"ringkasan"`
	SppPerMetode   []LaporanKelompok        `json:"spp_per_metode"`
	SppPerAngkatan []LaporanKelompok        `json:"spp_per_angkatan"`
	Tagihan        LaporanTagihanTerbit     `json:"tagihan"`
	Piutang        LaporanPiutang           `json:"piutang"`
	Pembayaran     []LaporanPembayaran      `json:"pembayaran"`
	Ujroh          []LaporanUjrohDibayar    `json:"ujroh"`
	Kasbon         []LaporanKasbon          `json:"kasbon"`
	Santri         []LaporanSantri          `json:"santri"`
	TidakLanjut    []LaporanSantri          `json:"tidak_lanjut"`
	OpsiAngkatan   []string                 `json:"opsi_angkatan"`
}
