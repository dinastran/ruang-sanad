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

// KoreksiLevelState describes the class's current level segment for the
// "koreksi nomor di level saat ini" tool. Only global meeting numbers after
// BatasBawah may change level label.
type KoreksiLevelState struct {
	LevelNama string `json:"level_nama"`
	// TerakhirGlobal is the last global meeting number (incl. legacy anchor).
	TerakhirGlobal int64 `json:"terakhir_global"`
	// LevelAwal is the global number right before the current level begins.
	LevelAwal int64 `json:"level_awal"`
	// TerakhirSekarang is the current per-level number of the last meeting.
	TerakhirSekarang int64 `json:"terakhir_sekarang"`
	// MaksTerakhir is the largest per-level number the admin may set.
	MaksTerakhir int64 `json:"maks_terakhir"`
	// LevelSebelum / OffsetSebelum label meetings moved back to the previous
	// level when the level actually started later than recorded.
	LevelSebelum            string                  `json:"level_sebelum"`
	OffsetSebelum           int64                   `json:"offset_sebelum"`
	Pertemuan               []KoreksiLevelPertemuan `json:"pertemuan"`
	AdaPertemuanBerlangsung bool                    `json:"ada_pertemuan_berlangsung"`
}

type KoreksiLevelPertemuan struct {
	PertemuanKe int64  `json:"pertemuan_ke"`
	Tanggal     string `json:"tanggal"`
	LevelNama   string `json:"level_nama"`
	LevelKe     int64  `json:"level_ke"`
}

type KoreksiLevelRequest struct {
	TerakhirLevel int64  `json:"terakhir_level"`
	Alasan        string `json:"alasan"`
}

type KoreksiPertemuanRequest struct {
	PertemuanTerakhir int64  `json:"pertemuan_terakhir"`
	Alasan            string `json:"alasan"`
}

type KoreksiPertemuanState struct {
	AnchorSebelumSistem     int64 `json:"anchor_sebelum_sistem"`
	JumlahPertemuan         int64 `json:"jumlah_pertemuan"`
	PertemuanGlobalTerakhir int64 `json:"pertemuan_global_terakhir"`
	AdaPertemuanBerlangsung bool  `json:"ada_pertemuan_berlangsung"`
}

type SetKelasMateriIndividualRequest struct {
	MateriIndividual bool `json:"materi_individual"`
}

type SetKelasKapasitasRequest struct {
	Kapasitas int64 `json:"kapasitas"`
}

type GantiLevelKelasRequest struct {
	Level string `json:"level"`
}

type GantiJadwalKelasRequest struct {
	Jadwal  string   `json:"jadwal"`
	Jadwals []string `json:"jadwals"`
}

type JadwalRutinSlotRequest struct {
	Hari     int64  `json:"hari"`
	JamMulai string `json:"jam_mulai"`
}

type SetJadwalRutinRequest struct {
	Slots []JadwalRutinSlotRequest `json:"slots"`
}

type JadwalRutinResponse struct {
	ID           int64  `json:"id"`
	KelasID      int64  `json:"kelas_id"`
	Hari         int64  `json:"hari"`
	JamMulai     string `json:"jam_mulai"`
	BerlakuMulai string `json:"berlaku_mulai"`
	IsAktif      bool   `json:"is_aktif"`
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
	Alasan          string `json:"alasan"`
	SantriNama      string `json:"santri_nama"`
	KelasAsalID     *int64 `json:"kelas_asal_id"`
	KelasAsalNama   string `json:"kelas_asal_nama"`
	KelasTujuanID   *int64 `json:"kelas_tujuan_id"`
	KelasTujuanNama string `json:"kelas_tujuan_nama"`
	DibuatOlehNama  string `json:"dibuat_oleh_nama"`
	CreatedAt       string `json:"created_at"`
}

// PertemuanBelumDitutup is a meeting still marked berlangsung. Lupa is true
// once it has been running longer than the lupa-ditutup threshold.
type PertemuanBelumDitutup struct {
	ID                int64  `json:"id"`
	KelasID           int64  `json:"kelas_id"`
	KelasNama         string `json:"kelas_nama"`
	GuruNama          string `json:"guru_nama"`
	GuruPenggantiNama string `json:"guru_pengganti_nama"`
	PertemuanKe       int64  `json:"pertemuan_ke"`
	Tanggal           string `json:"tanggal"`
	JamMulai          string `json:"jam_mulai"`
	DurasiJam         int64  `json:"durasi_jam"`
	Lupa              bool   `json:"lupa"`
}

type BatalkanPertemuanRequest struct {
	Alasan string `json:"alasan"`
}
