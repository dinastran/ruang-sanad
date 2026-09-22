package models

type KelasResponse struct {
	ID                       int64  `json:"id"`
	KunciKelas               string `json:"kunci_kelas"`
	Angkatan                 string `json:"angkatan"`
	Tipe                     string `json:"tipe"`
	JenisKelamin             string `json:"jenis_kelamin"`
	Level                    string `json:"level"`
	Frekuensi                string `json:"frekuensi"`
	Jadwal                   string `json:"jadwal"`
	SubIndex                 int64  `json:"sub_index"`
	NamaKelas                string `json:"nama_kelas"`
	GuruID                   *int64 `json:"guru_id"`
	GuruNama                 string `json:"guru_nama"`
	Kapasitas                int64  `json:"kapasitas"`
	JumlahSantri             int64  `json:"jumlah_santri"`
	PertemuanTerakhir        int64  `json:"pertemuan_terakhir"`
	TanggalPertemuanTerakhir string `json:"tanggal_pertemuan_terakhir"`
	MateriTerakhir           string `json:"materi_terakhir"`
	MateriIndividual         bool   `json:"materi_individual"`
	IsAktif                  bool   `json:"is_aktif"`
	CreatedAt                string `json:"created_at"`
}

type SetPertemuanTerakhirRequest struct {
	PertemuanTerakhir int64 `json:"pertemuan_terakhir"`
}

type SetKelasMateriIndividualRequest struct {
	MateriIndividual bool `json:"materi_individual"`
}

type GantiLevelKelasRequest struct {
	Level string `json:"level"`
}

type GantiJadwalKelasRequest struct {
	Jadwal string `json:"jadwal"`
}

// GantiLevelSantriRequest moves selected santri to a class of another level.
// Either KelasTujuanID is set, or BuatKelasBaru with Level to open a new class
// that copies the origin class's kode, gender, frekuensi, jadwal, and angkatan.
type GantiLevelSantriRequest struct {
	SantriIDs     []int64 `json:"santri_ids"`
	KelasTujuanID int64   `json:"kelas_tujuan_id"`
	BuatKelasBaru bool    `json:"buat_kelas_baru"`
	Level         string  `json:"level"`
}

type KelasPerubahanResponse struct {
	ID              int64  `json:"id"`
	Jenis           string `json:"jenis"`
	NilaiLama       string `json:"nilai_lama"`
	NilaiBaru       string `json:"nilai_baru"`
	PertemuanKe     int64  `json:"pertemuan_ke"`
	SantriNama      string `json:"santri_nama"`
	KelasAsalID     *int64 `json:"kelas_asal_id"`
	KelasAsalNama   string `json:"kelas_asal_nama"`
	KelasTujuanID   *int64 `json:"kelas_tujuan_id"`
	KelasTujuanNama string `json:"kelas_tujuan_nama"`
	DibuatOlehNama  string `json:"dibuat_oleh_nama"`
	CreatedAt       string `json:"created_at"`
}
