package models

// ProdukVarian is one price option of a produk (e.g. Tahsin Reguler 2x/pekan).
type ProdukVarian struct {
	ID        int64  `json:"id"`
	ProdukID  int64  `json:"produk_id"`
	Nama      string `json:"nama"`
	Harga     int64  `json:"harga"`
	PaketKode string `json:"paket_kode"`
	Urutan    int64  `json:"urutan"`
	IsAktif   bool   `json:"is_aktif"`
}

type ProdukVarianRequest struct {
	Nama      string `json:"nama"`
	Harga     int64  `json:"harga"`
	PaketKode string `json:"paket_kode"`
	Urutan    int64  `json:"urutan"`
	IsAktif   bool   `json:"is_aktif"`
}
