package models

type Product struct {
	ID        int64          `json:"id"`
	Nama      string         `json:"nama"`
	Kategori  string         `json:"kategori"`
	TrackStok bool           `json:"track_stok"`
	IsAktif   bool           `json:"is_aktif"`
	Stok         int64          `json:"stok"`
	MinimumStock int64          `json:"minimum_stock"`
	Batches      []ProductBatch `json:"batches"`
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

type ProductCRMDashboardFilters struct {
	DateFrom  string `json:"date_from"`
	DateTo    string `json:"date_to"`
	ProductID int64  `json:"product_id"`
	BatchID   int64  `json:"batch_id"`
	Category  string `json:"category"`
	Angkatan  string `json:"angkatan"`
	Status    string `json:"status"`
}

type ProductCRMDashboardPoint struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Total int64  `json:"total"`
}

type ProductCRMTrendPoint struct {
	Period string `json:"period"`
	Label  string `json:"label"`
	Total  int64  `json:"total"`
}

type ProductCRMStockMovement struct {
	Period string `json:"period"`
	Label  string `json:"label"`
	Masuk  int64  `json:"masuk"`
	Keluar int64  `json:"keluar"`
}

type ProductCRMInventoryHealth struct {
	ProductID    int64  `json:"product_id"`
	ProductName  string `json:"product_name"`
	Stock        int64  `json:"stock"`
	MinimumStock int64  `json:"minimum_stock"`
	Status       string `json:"status"`
}

type ProductCRMDashboardSummary struct {
	ActiveProducts       int64   `json:"active_products"`
	SantriTotal          int64   `json:"santri_total"`
	SantriWithProduct    int64   `json:"santri_with_product"`
	CoverageRate         float64 `json:"coverage_rate"`
	ActiveOwnerships     int64   `json:"active_ownerships"`
	TotalStock           int64   `json:"total_stock"`
	LowStockProducts     int64   `json:"low_stock_products"`
	PeriodAssignments    int64   `json:"period_assignments"`
	PreviousAssignments  int64   `json:"previous_assignments"`
	AssignmentChangeRate float64 `json:"assignment_change_rate"`
}

type ProductCRMDashboard struct {
	Summary             ProductCRMDashboardSummary  `json:"summary"`
	DistributionTrend   []ProductCRMTrendPoint      `json:"distribution_trend"`
	TopProducts         []ProductCRMDashboardPoint  `json:"top_products"`
	CategoryComposition []ProductCRMDashboardPoint  `json:"category_composition"`
	CoverageByAngkatan  []ProductCRMDashboardPoint  `json:"coverage_by_angkatan"`
	StockMovement       []ProductCRMStockMovement   `json:"stock_movement"`
	InventoryHealth     []ProductCRMInventoryHealth `json:"inventory_health"`
}

type ProductCRMDrilldownOwned struct {
	ProductID int64 `json:"product_id"`
	BatchID   int64 `json:"batch_id"`
}

type ProductCRMDrilldownRow struct {
	ID            int64                      `json:"id"`
	IDMahasantri  string                     `json:"id_mahasantri"`
	Nama          string                     `json:"nama"`
	NoWA          string                     `json:"no_wa"`
	Angkatan      string                     `json:"angkatan"`
	AngkatanKelas string                     `json:"angkatan_kelas"`
	Level         string                     `json:"level"`
	Status        string                     `json:"status"`
	Owned         []ProductCRMDrilldownOwned `json:"owned"`
}

type ProductCRMDrilldown struct {
	Data   []ProductCRMDrilldownRow `json:"data"`
	Total  int64                    `json:"total"`
	Page   int64                    `json:"page"`
	Limit  int64                    `json:"limit"`
	Mode   string                   `json:"mode"`
	Period string                   `json:"period"`
}

type CreateProductRequest struct {
	Nama         string `json:"nama"`
	Kategori     string `json:"kategori"`
	TrackStok    bool   `json:"track_stok"`
	MinimumStock int64  `json:"minimum_stock"`
}

type UpdateProductRequest struct {
	Nama         string `json:"nama"`
	Kategori     string `json:"kategori"`
	TrackStok    bool   `json:"track_stok"`
	IsAktif      bool   `json:"is_aktif"`
	MinimumStock int64  `json:"minimum_stock"`
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
