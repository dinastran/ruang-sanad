package models

type ImportResult struct {
	Total    int    `json:"total"`
	Berhasil int    `json:"berhasil"`
	Gagal    int    `json:"gagal"`
	Catatan  string `json:"catatan"`
}
