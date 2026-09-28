package models

type ProductOrder struct {
	ID              int64              `json:"id"`
	OrderNo         string             `json:"order_no"`
	Tanggal         string             `json:"tanggal"`
	NamaPembeli     string             `json:"nama_pembeli"`
	NoWA            string             `json:"no_wa"`
	Sumber          string             `json:"sumber"`
	Status          string             `json:"status"`
	Catatan         string             `json:"catatan"`
	DicatatOleh     string             `json:"dicatat_oleh"`
	DibatalkanOleh  string             `json:"dibatalkan_oleh"`
	CreatedAt       string             `json:"created_at"`
	Items           []ProductOrderItem `json:"items"`
}

type ProductOrderItem struct {
	ID             int64  `json:"id"`
	OrderID        int64  `json:"order_id"`
	ProdukID       int64  `json:"produk_id"`
	ProdukBatchID  int64  `json:"produk_batch_id"`
	ProdukNama     string `json:"produk_nama"`
	BatchNama      string `json:"batch_nama"`
	Kategori       string `json:"kategori"`
	TrackStok      bool   `json:"track_stok"`
	Qty            int64  `json:"qty"`
}

type CreateProductOrderItemRequest struct {
	ProdukID      int64 `json:"produk_id"`
	ProdukBatchID int64 `json:"produk_batch_id"`
	Qty           int64 `json:"qty"`
}

type CreateProductOrderRequest struct {
	Tanggal     string                          `json:"tanggal"`
	NamaPembeli string                          `json:"nama_pembeli"`
	NoWA        string                          `json:"no_wa"`
	Sumber      string                          `json:"sumber"`
	Catatan     string                          `json:"catatan"`
	Items       []CreateProductOrderItemRequest `json:"items"`
}
