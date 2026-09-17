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
	IsAktif                  bool   `json:"is_aktif"`
	CreatedAt                string `json:"created_at"`
}

type SetPertemuanTerakhirRequest struct {
	PertemuanTerakhir int64 `json:"pertemuan_terakhir"`
}
