package models

type Product struct {
	ID        int64          `json:"id"`
	Nama      string         `json:"nama"`
	Kategori  string         `json:"kategori"`
	TrackStok bool           `json:"track_stok"`
	IsAktif   bool           `json:"is_aktif"`
	Stok      int64          `json:"stok"`
	Batches   []ProductBatch `json:"batches"`
}

type ProductBatch struct {
	ID             int64  `json:"id"`
	ProdukID       int64  `json:"produk_id"`
	Nama           string `json:"nama"`
	TanggalMulai   string `json:"tanggal_mulai"`
	TanggalSelesai string `json:"tanggal_selesai"`
	IsAktif        bool   `json:"is_aktif"`
}

type MahasantriProduct struct {
	ID             int64  `json:"id"`
	SantriID       int64  `json:"santri_id"`
	ProdukID       int64  `json:"produk_id"`
	ProdukBatchID  int64  `json:"produk_batch_id"`
	ProdukNama     string `json:"produk_nama"`
	BatchNama      string `json:"batch_nama"`
	Kategori       string `json:"kategori"`
	TrackStok      bool   `json:"track_stok"`
	Tanggal        string `json:"tanggal"`
	Status         string `json:"status"`
	Catatan        string `json:"catatan"`
	DicatatOleh    string `json:"dicatat_oleh"`
}

type ProductCRMRow struct {
	ID                 int64                `json:"id"`
	IDMahasantri       string               `json:"id_mahasantri"`
	Nama               string               `json:"nama"`
	Angkatan           string               `json:"angkatan"`
	AngkatanKelas      string               `json:"angkatan_kelas"`
	Level              string               `json:"level"`
	Status             string               `json:"status"`
	Produk             []MahasantriProduct  `json:"produk"`
}

type StockMutation struct {
	ID            int64  `json:"id"`
	ProdukID      int64  `json:"produk_id"`
	ProdukBatchID int64  `json:"produk_batch_id"`
	ProdukNama    string `json:"produk_nama"`
	BatchNama     string `json:"batch_nama"`
	Tipe          string `json:"tipe"`
	Qty           int64  `json:"qty"`
	Catatan       string `json:"catatan"`
	DicatatOleh   string `json:"dicatat_oleh"`
	CreatedAt     string `json:"created_at"`
}

type ProductCRMFilters struct {
	Search           string
	Angkatan         string
	Status           string
	HasProductID     int64
	MissingProductID int64
	BatchID          int64
	Page             int64
	Limit            int64
}

type ProductCRMList struct {
	Data  []ProductCRMRow `json:"data"`
	Total int64           `json:"total"`
}

type CreateProductRequest struct {
	Nama      string `json:"nama"`
	Kategori  string `json:"kategori"`
	TrackStok bool   `json:"track_stok"`
}

type UpdateProductRequest struct {
	Nama      string `json:"nama"`
	Kategori  string `json:"kategori"`
	TrackStok bool   `json:"track_stok"`
	IsAktif   bool   `json:"is_aktif"`
}

type CreateProductBatchRequest struct {
	Nama           string `json:"nama"`
	TanggalMulai   string `json:"tanggal_mulai"`
	TanggalSelesai string `json:"tanggal_selesai"`
}

type UpdateProductBatchRequest struct {
	Nama           string `json:"nama"`
	TanggalMulai   string `json:"tanggal_mulai"`
	TanggalSelesai string `json:"tanggal_selesai"`
	IsAktif        bool   `json:"is_aktif"`
}

type StockEntryRequest struct {
	ProdukID      int64  `json:"produk_id"`
	ProdukBatchID int64  `json:"produk_batch_id"`
	Tipe          string `json:"tipe"`
	Qty           int64  `json:"qty"`
	Catatan       string `json:"catatan"`
}

type StockOpnameRequest struct {
	ProdukID      int64  `json:"produk_id"`
	ProdukBatchID int64  `json:"produk_batch_id"`
	StokFisik     int64  `json:"stok_fisik"`
	Catatan       string `json:"catatan"`
}

type AssignProductRequest struct {
	ProdukID      int64  `json:"produk_id"`
	ProdukBatchID int64  `json:"produk_batch_id"`
	Tanggal       string `json:"tanggal"`
	Catatan       string `json:"catatan"`
}
