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
	// TampilKodeReferal shows the optional "Kode referal" field (default on).
	TampilKodeReferal *bool `json:"tampil_kode_referal"`
	// MetaTracking loads the Meta Pixel on the public page and sends the
	// form events through CAPI too (default on).
	MetaTracking *bool `json:"meta_tracking"`
	// MetaPixelExtra: extra Pixel IDs (comma separated), browser only.
	MetaPixelExtra string `json:"meta_pixel_extra"`
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
	KodeReferal   bool                 `json:"kode_referal"`
	// Meta is nil when the link has no Pixel to load.
	Meta *PublicMeta `json:"meta,omitempty"`
}

// PublicMeta tells the public page which Pixels to load. ViewEventID is
// shared by the browser ViewContent and its CAPI copy so Meta dedupes them.
type PublicMeta struct {
	PixelIDs    []string `json:"pixel_ids"`
	ViewEventID string   `json:"view_event_id"`
}

// MetaBrowserEvent is one Pixel event the browser fires after a submit,
// with the same event_id the server sends through CAPI.
type MetaBrowserEvent struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Value    int64  `json:"value"`
	Currency string `json:"currency"`
}

// MetaBrowserTracking is returned after a submit: events to fire and the
// hashed advanced-matching data (raw personal data never goes to the page).
type MetaBrowserTracking struct {
	Events []MetaBrowserEvent `json:"events"`
	User   map[string]string  `json:"user"`
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
	KodeReferal    string `json:"kode_referal"`
	Honeypot       string `json:"website"`
	FormToken      string `json:"form_token"`
	// PageURL is the full page URL (with UTM/fbclid) for event_source_url.
	PageURL string `json:"page_url"`
}

type RequestMeta struct {
	IP        string
	UserAgent string
	AffCookie string // raw rs_aff cookie value, if any
	Fbp       string // _fbp cookie (Meta browser id)
	Fbc       string // _fbc cookie (Meta click id)
	PageURL   string // page the visitor is on (event_source_url)
}

type RegistrationResult struct {
	LeadRef  string `json:"lead_ref"`
	Next     string `json:"next"`
	WaURL    string `json:"wa_url"`
	BayarRef string `json:"bayar_ref,omitempty"`
	// Meta: Pixel events for the browser; nil when tracking is off.
	Meta *MetaBrowserTracking `json:"meta,omitempty"`
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
	GatewayAktif bool   `json:"gateway_aktif"`
	// SesiAktif: an online payment session is open (checkout started).
	SesiAktif bool `json:"sesi_aktif"`
	// Meta: Pixel for the payment page; nil when tracking is off.
	Meta *BayarMeta `json:"meta,omitempty"`
}

// BayarMeta lets the payment page fire AddPaymentInfo after a proof upload,
// with the event_id of the CRM CAPI event (deduplicated by Meta).
type BayarMeta struct {
	PixelIDs         []string          `json:"pixel_ids"`
	AddPaymentInfoID string            `json:"add_payment_info_id"`
	Value            int64             `json:"value"`
	User             map[string]string `json:"user"`
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

// --- Affiliate ---

type AffiliateSignup struct {
	Nama         string `json:"nama"`
	NoWA         string `json:"no_wa"`
	Email        string `json:"email"`
	BankNama     string `json:"bank_nama"`
	BankRekening string `json:"bank_rekening"`
	BankAtasNama string `json:"bank_atas_nama"`
	Setuju       bool   `json:"setuju"`
	Honeypot     string `json:"website"`
}

type AffiliateRuleRequest struct {
	Scope     string `json:"scope"`
	TargetID  int64  `json:"target_id"`
	Jenis     string `json:"jenis"`
	Nilai     int64  `json:"nilai"`
	Pola      string `json:"pola"`
	MaksBulan *int64 `json:"maks_bulan"`
	Nonaktif  bool   `json:"nonaktif"`
}

type AffiliateSettings struct {
	MasaBerlakuHari     int64  `json:"masa_berlaku_hari"`
	MasaTahanHari       int64  `json:"masa_tahan_hari"`
	LinkTujuanDefaultID int64  `json:"link_tujuan_default_id"`
	TemplateBagikan     string `json:"template_bagikan"`
}

type AffiliateRow struct {
	ID           int64  `json:"id"`
	Kode         string `json:"kode"`
	Tipe         string `json:"tipe"`
	Nama         string `json:"nama"`
	NoWA         string `json:"no_wa"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	BankNama     string `json:"bank_nama"`
	BankRekening string `json:"bank_rekening"`
	BankAtasNama string `json:"bank_atas_nama"`
	PunyaAkun    bool   `json:"punya_akun"`
	Klik         int64  `json:"klik"`
	Daftar       int64  `json:"daftar"`
	Santri       int64  `json:"santri"`
	KomisiSiap   int64  `json:"komisi_siap"`
	KomisiTotal  int64  `json:"komisi_total"`
}

type AffiliateRujukan struct {
	NamaDepan string `json:"nama_depan"`
	Program   string `json:"program"`
	Tanggal   string `json:"tanggal"`
	Status    string `json:"status"` // daftar | terverifikasi | santri | batal
	Komisi    int64  `json:"komisi"`
}

type AffiliateRuleInfo struct {
	Label  string `json:"label"`  // "10%" / "Rp50.000"
	Pola   string `json:"pola"`   // sekali | bulanan
	Durasi string `json:"durasi"` // "6 bulan" / "selama aktif"
}

type AffiliateTujuan struct {
	Slug  string `json:"slug"`
	Judul string `json:"judul"`
}

type AffiliateDashboard struct {
	ID            int64              `json:"id"`
	Kode          string             `json:"kode"`
	Nama          string             `json:"nama"`
	Tipe          string             `json:"tipe"`
	BankNama      string             `json:"bank_nama"`
	BankRekening  string             `json:"bank_rekening"`
	BankAtasNama  string             `json:"bank_atas_nama"`
	Klik30        int64              `json:"klik_30"`
	Daftar30      int64              `json:"daftar_30"`
	Santri        int64              `json:"santri"`
	KomisiPending int64              `json:"komisi_pending"`
	KomisiSiap    int64              `json:"komisi_siap"`
	KomisiDibayar int64              `json:"komisi_dibayar"`
	MasaBerlaku   int64              `json:"masa_berlaku_hari"`
	MasaTahan     int64              `json:"masa_tahan_hari"`
	Aturan        *AffiliateRuleInfo `json:"aturan"`
	Rujukan       []AffiliateRujukan `json:"rujukan"`
	Tujuan        []AffiliateTujuan  `json:"tujuan"`
	TujuanDefault string             `json:"tujuan_default"`
	TemplateBagi  string             `json:"template_bagikan"`
}

type AffiliateLedgerRow struct {
	ID            int64  `json:"id"`
	AffiliateID   int64  `json:"affiliate_id"`
	AffiliateNama string `json:"affiliate_nama"`
	AffiliateKode string `json:"affiliate_kode"`
	AffiliateTipe string `json:"affiliate_tipe"`
	Rekening      string `json:"rekening"`
	LeadID        int64  `json:"lead_id"`
	LeadNama      string `json:"lead_nama"`
	Sumber        string `json:"sumber"`
	BulanKe       int64  `json:"bulan_ke"`
	Basis         int64  `json:"basis"`
	Jumlah        int64  `json:"jumlah"`
	Status        string `json:"status"`
	MetodeBayar   string `json:"metode_bayar"`
	UjrohBulan    string `json:"ujroh_bulan"`
	Catatan       string `json:"catatan"`
	AlasanBatal   string `json:"alasan_batal"`
	CreatedAt     string `json:"created_at"`
}

// MetaPixelInfo: the main Pixel (.env) and whether CAPI sending is on.
type MetaPixelInfo struct {
	PixelID   string `json:"pixel_id"`
	CAPIAktif bool   `json:"capi_aktif"`
}
