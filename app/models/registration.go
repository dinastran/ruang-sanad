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

// FieldRule controls one optional form field of a registration link.
type FieldRule struct {
	Tampil bool `json:"tampil"`
	Wajib  bool `json:"wajib"`
}

type RegistrationLinkRequest struct {
	Slug            string               `json:"slug"`
	Judul           string               `json:"judul"`
	Deskripsi       string               `json:"deskripsi"`
	ProgramID       int64                `json:"program_id"`
	LeadSourceID    int64                `json:"lead_source_id"`
	Mode            string               `json:"mode"`
	AfterSubmit     string               `json:"after_submit"`
	NominalOverride *int64               `json:"nominal_override"`
	FieldConfig     map[string]FieldRule `json:"field_config"`
	WaTemplate      string               `json:"wa_template"`
	IzinkanManual   bool                 `json:"izinkan_manual"`
	IsAktif         bool                 `json:"is_aktif"`
}

// RegistrationLinkRow is one link on the admin page, with 30-day stats.
type RegistrationLinkRow struct {
	RegistrationLinkRequest
	ID          int64  `json:"id"`
	ProgramNama string `json:"program_nama"`
	SumberNama  string `json:"sumber_nama"`
	HargaLabel  string `json:"harga_label"`
	Klik        int64  `json:"klik"`
	Submit      int64  `json:"submit"`
	Bayar       int64  `json:"bayar"`
}

type PublicVarian struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Harga int64  `json:"harga"`
}

// PublicRegistrationLink is what the public /daftar/:slug page renders.
type PublicRegistrationLink struct {
	ID            int64                `json:"id"`
	Slug          string               `json:"slug"`
	Judul         string               `json:"judul"`
	Deskripsi     string               `json:"deskripsi"`
	Mode          string               `json:"mode"`
	AfterSubmit   string               `json:"after_submit"`
	ProgramNama   string               `json:"program_nama"`
	ButuhJadwal   bool                 `json:"butuh_jadwal"`
	Harga         int64                `json:"harga"`
	Varian        []PublicVarian       `json:"varian"`
	FieldConfig   map[string]FieldRule `json:"field_config"`
	IzinkanManual bool                 `json:"izinkan_manual"`
}

type RegistrationSettings struct {
	CsWaNumber         string `json:"cs_wa_number"`
	PaymentExpiryHours int64  `json:"payment_expiry_hours"`
	BankManualInfo     string `json:"bank_manual_info"`
}

type RegistrationSubmit struct {
	Nama           string `json:"nama"`
	NoWA           string `json:"no_wa"`
	JenisKelamin   string `json:"jenis_kelamin"`
	Usia           int64  `json:"usia"`
	Domisili       string `json:"domisili"`
	VarianID       int64  `json:"varian_id"`
	JadwalKode     string `json:"jadwal_kode"`
	RiwayatBelajar string `json:"riwayat_belajar"`
	BelajarDimana  string `json:"belajar_dimana"`
	UntukSiapa     string `json:"untuk_siapa"`
	WaktuBelajar   string `json:"waktu_belajar"`
	Honeypot       string `json:"website"`
	FormToken      string `json:"form_token"`
}

type RequestMeta struct {
	IP        string
	UserAgent string
}

type RegistrationResult struct {
	LeadRef  string `json:"lead_ref"`
	Next     string `json:"next"`
	WaURL    string `json:"wa_url"`
	BayarRef string `json:"bayar_ref,omitempty"`
}

// BayarInfo is what the public payment page shows for one registration ref.
type BayarInfo struct {
	LeadRef        string `json:"lead_ref"`
	NamaDepan      string `json:"nama_depan"`
	ProgramNama    string `json:"program_nama"`
	Nominal        int64  `json:"nominal"`
	BankManualInfo string `json:"bank_manual_info"`
	IzinkanManual  bool   `json:"izinkan_manual"`
	// Status: belum | menunggu_verifikasi | ditolak | lunas
	Status       string `json:"status"`
	CatatanTolak string `json:"catatan_tolak,omitempty"`
	WaURL        string `json:"wa_url"`
}

type LeadRegistrationRow struct {
	ID        int64  `json:"id"`
	LinkJudul string `json:"link_judul"`
	LinkSlug  string `json:"link_slug"`
	LinkMode  string `json:"link_mode"`
	CreatedAt string `json:"created_at"`
}

type PaymentRow struct {
	ID            int64  `json:"id"`
	Metode        string `json:"metode"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	Channel       string `json:"channel"`
	AdaBukti      bool   `json:"ada_bukti"`
	CatatanSistem string `json:"catatan_sistem"`
	PerluTinjau   bool   `json:"perlu_tinjau"`
	CreatedAt     string `json:"created_at"`
	PaidAt        string `json:"paid_at"`
}

// LeadRegistrationPanel is the "Pendaftaran & Pembayaran" panel on a lead.
type LeadRegistrationPanel struct {
	Registrasi []LeadRegistrationRow `json:"registrasi"`
	Payments   []PaymentRow          `json:"payments"`
}

type Option struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Info  string `json:"info,omitempty"`
	Aktif bool   `json:"aktif"`
}
