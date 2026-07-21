package models

// Response DTOs — lowercase json keys consumed by the frontend (SantriForm
// dropdowns, MasterData management page, etc.). The generated sqlc structs
// have no json tags and would otherwise marshal with capitalized keys.

type AngkatanResponse struct {
	ID         int64  `json:"id"`
	Kode       string `json:"kode"`
	Keterangan string `json:"keterangan"`
	IsAktif    bool   `json:"is_aktif"`
}

type LevelResponse struct {
	ID     int64  `json:"id"`
	Kode   string `json:"kode"`
	Nama   string `json:"nama"`
	Urutan int64  `json:"urutan"`
}

type JadwalResponse struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
}

type GuruResponse struct {
	ID           int64  `json:"id"`
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
	IsAktif      bool   `json:"is_aktif"`
}

type KodeKelasResponse struct {
	ID        int64  `json:"id"`
	Kode      string `json:"kode"`
	Tipe      string `json:"tipe"`
	Frekuensi string `json:"frekuensi"`
	Urutan    int64  `json:"urutan"`
}

// Request DTOs for creating master data.

type CreateAngkatanRequest struct {
	Kode       string `json:"kode"`
	Keterangan string `json:"keterangan"`
}

type CreateLevelRequest struct {
	Kode   string `json:"kode"`
	Nama   string `json:"nama"`
	Urutan int64  `json:"urutan"`
}

type CreateJadwalRequest struct {
	Nama string `json:"nama"`
}

type CreateGuruRequest struct {
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
}

type CreateKodeKelasRequest struct {
	Kode      string `json:"kode"`
	Tipe      string `json:"tipe"`
	Frekuensi string `json:"frekuensi"`
	Urutan    int64  `json:"urutan"`
}

// Request DTOs for editing master data. Codes (angkatan.kode, level.kode) are the
// natural keys and are not editable — only the descriptive fields change.

type UpdateAngkatanRequest struct {
	Keterangan string `json:"keterangan"`
}

type UpdateLevelRequest struct {
	Nama   string `json:"nama"`
	Urutan int64  `json:"urutan"`
}

type UpdateJadwalRequest struct {
	Nama string `json:"nama"`
}

type UpdateGuruRequest struct {
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
}

// UpdateKodeKelasRequest edits the derived attributes of a kode kelas. The kode
// itself is the natural key and is not editable.
type UpdateKodeKelasRequest struct {
	Tipe      string `json:"tipe"`
	Frekuensi string `json:"frekuensi"`
	Urutan    int64  `json:"urutan"`
}
