package models

// LeadProgram is a program marketed through the WhatsApp CRM. Jenis "reguler"
// closes into a santri; jenis "program" (short program such as Sepekan
// Bersama Al-Fatihah) closes into a participant = a Produk CRM order.
type LeadProgram struct {
	ID            int64  `json:"id"`
	Nama          string `json:"nama"`
	Jenis         string `json:"jenis"`
	ProdukID      int64  `json:"produk_id"`
	ProdukNama    string `json:"produk_nama"`
	ProdukHarga   int64  `json:"produk_harga"`
	ProdukBatchID int64  `json:"produk_batch_id"`
	BatchNama     string `json:"batch_nama"`
	Keterangan    string `json:"keterangan"`
	IsAktif       bool   `json:"is_aktif"`
	// Filled by the service for list pages.
	SumberIDs     []int64 `json:"sumber_ids"`
	LeadOpen      int64   `json:"lead_open"`
	LeadWon       int64   `json:"lead_won"`
	JumlahSesi    int64   `json:"jumlah_sesi"`
	JumlahPeserta int64   `json:"jumlah_peserta"`
}

type LeadProgramRequest struct {
	Nama          string `json:"nama"`
	Jenis         string `json:"jenis"`
	ProdukID      int64  `json:"produk_id"`
	ProdukBatchID int64  `json:"produk_batch_id"`
	Keterangan    string `json:"keterangan"`
	IsAktif       bool   `json:"is_aktif"`
}

// LeadProgramOption is a program shown in selects (lead list, lead detail).
type LeadProgramOption struct {
	ID      int64  `json:"id"`
	Nama    string `json:"nama"`
	Jenis   string `json:"jenis"`
	IsAktif bool   `json:"is_aktif"`
}

type ProgramSesi struct {
	ID        int64  `json:"id"`
	ProgramID int64  `json:"program_id"`
	Ke        int64  `json:"ke"`
	Judul     string `json:"judul"`
	Tanggal   string `json:"tanggal"`
	JamMulai  string `json:"jam_mulai"`
	Token     string `json:"token"`
	Hadir     int64  `json:"hadir"`
}

type ProgramSesiRequest struct {
	Ke       int64  `json:"ke"`
	Judul    string `json:"judul"`
	Tanggal  string `json:"tanggal"`
	JamMulai string `json:"jam_mulai"`
}

// ProgramPeserta is one participant: an active Produk CRM order for the
// program's product (and batch, when the program has one).
type ProgramPeserta struct {
	OrderID    int64   `json:"order_id"`
	OrderNo    string  `json:"order_no"`
	Nama       string  `json:"nama"`
	NoWA       string  `json:"no_wa"`
	Tanggal    string  `json:"tanggal"`
	LeadID     int64   `json:"lead_id"`
	LeadStatus string  `json:"lead_status"`
	Hadir      []int64 `json:"hadir"` // sesi ids attended
}

type ProgramKehadiran struct {
	SesiID  int64
	OrderID int64
	Cara    string
}

// LeadProgramOrder is the participant order created when a lead closed into
// a short program.
type LeadProgramOrder struct {
	OrderID int64  `json:"order_id"`
	OrderNo string `json:"order_no"`
	Status  string `json:"status"`
}

type LeadPesertaRequest struct {
	Nama    string `json:"nama"`
	NoWA    string `json:"no_wa"`
	Nominal int64  `json:"nominal"`
	Tanggal string `json:"tanggal"`
}
