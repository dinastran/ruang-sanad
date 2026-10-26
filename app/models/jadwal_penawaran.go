package models

// JadwalPenawaranRequest creates or updates a schedule offered to calon santri.
type JadwalPenawaranRequest struct {
	Kode           string `json:"kode"` // only used on create
	Hari           string `json:"hari"` // "3,6" (1=Senin .. 7=Ahad)
	JamMulai       string `json:"jam_mulai"`
	JamSelesai     string `json:"jam_selesai"`
	Level          string `json:"level"` // "" = semua level
	KuotaPerGender int64  `json:"kuota_per_gender"`
	IsBuka         bool   `json:"is_buka"`
	Urutan         int64  `json:"urutan"`
	Catatan        string `json:"catatan"`
}

// JadwalPenawaranResponse is one schedule with its seat counts. It is also
// what the CS AI bridge reads from GET /integrations/cs-ai/jadwal.
type JadwalPenawaranResponse struct {
	ID             int64  `json:"id"`
	Kode           string `json:"kode"`
	Hari           string `json:"hari"`
	HariList       []int  `json:"hari_list"`
	Frekuensi      int    `json:"frekuensi"` // kali per pekan
	JamMulai       string `json:"jam_mulai"`
	JamSelesai     string `json:"jam_selesai"`
	Label          string `json:"label"`
	Level          string `json:"level"`
	KuotaPerGender int64  `json:"kuota_per_gender"`
	IsBuka         bool   `json:"is_buka"`
	Urutan         int64  `json:"urutan"`
	Catatan        string `json:"catatan"`
	PeriodeMulai   string `json:"periode_mulai"`
	TerisiIkhwan   int64  `json:"terisi_ikhwan"`
	TerisiAkhwat   int64  `json:"terisi_akhwat"`
	SisaIkhwan     int64  `json:"sisa_ikhwan"`
	SisaAkhwat     int64  `json:"sisa_akhwat"`
	// TanpaGender counts verified leads whose gender is empty (not in quota).
	TanpaGender int64 `json:"tanpa_gender"`
}
