package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Ambang penanda riayah (disepakati dengan pengguna).
const (
	riayahJendelaHari        = 30
	riayahMinPertemuan       = 3
	riayahBatasPersenHadir   = 70.0
	riayahAlpaBerturut       = 2
	riayahProgresMacetJumlah = 4
)

var ErrRiayahAksesDitolak = errors.New("anda tidak memiliki akses ke santri ini")

type RiayahSantriService struct {
	querier *queries.Querier
}

func NewRiayahSantriService(querier *queries.Querier) *RiayahSantriService {
	return &RiayahSantriService{querier: querier}
}

func guruScope(guruID *int64) interface{} {
	if guruID == nil {
		return nil
	}
	return *guruID
}

// ListPerhatian mengembalikan santri dalam cakupan viewer, diurutkan dari yang
// paling perlu diperhatikan.
func (s *RiayahSantriService) ListPerhatian(viewer models.RiayahViewer, today time.Time) ([]models.RiayahSantriItem, models.RiayahRingkasan, error) {
	ctx := context.Background()
	scope := guruScope(viewer.GuruID)

	santriRows, err := s.querier.ListSantriRiayahScope(ctx, scope)
	if err != nil {
		return nil, models.RiayahRingkasan{}, err
	}
	absensiRows, err := s.querier.ListAbsensiTerakhirRiayah(ctx, scope)
	if err != nil {
		return nil, models.RiayahRingkasan{}, err
	}
	sejak := today.AddDate(0, 0, -riayahJendelaHari).Format("2006-01-02")
	rekapRows, err := s.querier.RekapKehadiranRiayahSejak(ctx, queries.RekapKehadiranRiayahSejakParams{Sejak: sejak, GuruID: scope})
	if err != nil {
		return nil, models.RiayahRingkasan{}, err
	}

	absensiBySantri := make(map[int64][]queries.ListAbsensiTerakhirRiayahRow, len(santriRows))
	for _, row := range absensiRows {
		absensiBySantri[row.SantriID] = append(absensiBySantri[row.SantriID], row)
	}
	type rekap struct{ total, hadir int64 }
	rekapBySantri := make(map[int64]rekap, len(rekapRows))
	for _, row := range rekapRows {
		hadir := int64(0)
		if row.TotalHadir.Valid {
			hadir = int64(row.TotalHadir.Float64)
		}
		rekapBySantri[row.SantriID] = rekap{total: row.Total, hadir: hadir}
	}

	items := make([]models.RiayahSantriItem, 0, len(santriRows))
	ringkasan := models.RiayahRingkasan{TotalSantri: len(santriRows)}
	for _, row := range santriRows {
		absen := absensiBySantri[row.ID]
		r := rekapBySantri[row.ID]
		item := models.RiayahSantriItem{
			ID:               row.ID,
			Nama:             row.Nama,
			IDMahasantri:     row.IDMahasantri,
			NoWa:             row.NoWa,
			KelasID:          row.KelasID,
			NamaKelas:        row.NamaKelas,
			Level:            row.Level,
			Jadwal:           row.Jadwal,
			GuruNama:         row.GuruNama,
			TotalPertemuan30: r.total,
		}
		if r.total > 0 {
			persen := float64(r.hadir) / float64(r.total) * 100
			item.PersenHadir30 = &persen
		}
		for _, a := range absen {
			if a.BatasMateri != "" {
				item.BatasMateriTerakhir = a.BatasMateri
				break
			}
		}
		item.Penanda = hitungPenandaRiayah(absen, r.total, r.hadir)

		if len(item.Penanda) > 0 {
			ringkasan.PerluPerhatian++
		}
		for _, p := range item.Penanda {
			switch p.Kode {
			case models.PenandaKehadiran:
				ringkasan.Kehadiran++
			case models.PenandaKontak:
				ringkasan.Kontak++
			case models.PenandaProgres:
				ringkasan.Progres++
			}
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		si, sj := skorPenanda(items[i].Penanda), skorPenanda(items[j].Penanda)
		if si != sj {
			return si > sj
		}
		return strings.ToLower(items[i].Nama) < strings.ToLower(items[j].Nama)
	})
	return items, ringkasan, nil
}

// Ringkasan menghitung jumlah santri per penanda untuk dashboard guru.
func (s *RiayahSantriService) Ringkasan(viewer models.RiayahViewer, today time.Time) (models.RiayahRingkasan, error) {
	_, ringkasan, err := s.ListPerhatian(viewer, today)
	return ringkasan, err
}

// hitungPenandaRiayah menerima absensi terbaru lebih dulu.
func hitungPenandaRiayah(absen []queries.ListAbsensiTerakhirRiayahRow, total30, hadir30 int64) []models.RiayahPenanda {
	penanda := []models.RiayahPenanda{}

	alpaBerturut := 0
	for _, a := range absen {
		if a.Status != "alpa" {
			break
		}
		alpaBerturut++
	}
	switch {
	case alpaBerturut >= riayahAlpaBerturut:
		penanda = append(penanda, models.RiayahPenanda{
			Kode:   models.PenandaKehadiran,
			Level:  models.LevelMerah,
			Alasan: fmt.Sprintf("Alpa %d pertemuan terakhir berturut-turut", alpaBerturut),
		})
	case total30 >= riayahMinPertemuan && float64(hadir30)/float64(total30)*100 < riayahBatasPersenHadir:
		penanda = append(penanda, models.RiayahPenanda{
			Kode:   models.PenandaKehadiran,
			Level:  models.LevelMerah,
			Alasan: fmt.Sprintf("Hadir %d dari %d pertemuan dalam %d hari terakhir", hadir30, total30, riayahJendelaHari),
		})
	}

	batas := make([]string, 0, riayahProgresMacetJumlah)
	for _, a := range absen {
		if (a.Status != "hadir" && a.Status != "telat") || strings.TrimSpace(a.BatasMateri) == "" {
			continue
		}
		batas = append(batas, a.BatasMateri)
		if len(batas) == riayahProgresMacetJumlah {
			break
		}
	}
	if len(batas) == riayahProgresMacetJumlah {
		sama := true
		for _, b := range batas[1:] {
			if normalisasiBatas(b) != normalisasiBatas(batas[0]) {
				sama = false
				break
			}
		}
		if sama {
			penanda = append(penanda, models.RiayahPenanda{
				Kode:   models.PenandaProgres,
				Level:  models.LevelKuning,
				Alasan: fmt.Sprintf("Batas materi \"%s\" tidak berubah dalam %d pertemuan terakhir", strings.TrimSpace(batas[0]), riayahProgresMacetJumlah),
			})
		}
	}
	return penanda
}

func normalisasiBatas(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func skorPenanda(penanda []models.RiayahPenanda) int {
	skor := 0
	for _, p := range penanda {
		switch p.Level {
		case models.LevelMerah:
			skor += 100
		case models.LevelOranye:
			skor += 10
		case models.LevelKuning:
			skor++
		}
	}
	return skor
}

// EnsureCanAccessSantri memastikan guru hanya membuka santri di kelasnya.
func (s *RiayahSantriService) EnsureCanAccessSantri(viewer models.RiayahViewer, santriID int64) (queries.Santri, error) {
	santri, err := s.querier.GetSantriByID(context.Background(), santriID)
	if err != nil {
		return queries.Santri{}, fmt.Errorf("santri tidak ditemukan")
	}
	if viewer.GuruID == nil {
		return santri, nil
	}
	if !santri.KelasID.Valid {
		return queries.Santri{}, ErrRiayahAksesDitolak
	}
	kelas, err := s.querier.GetKelasByID(context.Background(), santri.KelasID.Int64)
	if err != nil || !kelas.GuruID.Valid || kelas.GuruID.Int64 != *viewer.GuruID {
		return queries.Santri{}, ErrRiayahAksesDitolak
	}
	return santri, nil
}

// GetProfil menyusun profil santri beserta timeline riayah.
func (s *RiayahSantriService) GetProfil(viewer models.RiayahViewer, santriID int64, today time.Time) (*models.RiayahSantriProfil, error) {
	ctx := context.Background()
	santri, err := s.EnsureCanAccessSantri(viewer, santriID)
	if err != nil {
		return nil, err
	}

	item := models.RiayahSantriItem{
		ID:           santri.ID,
		Nama:         santri.Nama,
		IDMahasantri: santri.IDMahasantri,
		NoWa:         santri.NoWa,
		Penanda:      []models.RiayahPenanda{},
	}
	var kelasGuruID *int64
	if santri.KelasID.Valid {
		if kelas, err := s.querier.GetKelasByID(ctx, santri.KelasID.Int64); err == nil {
			item.KelasID = kelas.ID
			item.NamaKelas = kelas.NamaKelas
			item.Level = kelas.Level
			item.Jadwal = kelas.Jadwal
			if kelas.GuruID.Valid {
				kelasGuruID = &kelas.GuruID.Int64
				if guru, err := s.querier.GuruGetByID(ctx, kelas.GuruID.Int64); err == nil {
					item.GuruNama = guru.Nama
				}
			}
		}
	}
	// Penanda dihitung dari daftar yang sama agar konsisten dengan halaman riayah,
	// dipersempit ke guru kelas santri supaya tidak memindai semua santri.
	if santri.Status == "aktif" && kelasGuruID != nil {
		scope := viewer
		scope.GuruID = kelasGuruID
		if list, _, err := s.ListPerhatian(scope, today); err == nil {
			for _, it := range list {
				if it.ID == santri.ID {
					item.Penanda = it.Penanda
					item.PersenHadir30 = it.PersenHadir30
					item.TotalPertemuan30 = it.TotalPertemuan30
					item.KontakTerakhir = it.KontakTerakhir
					break
				}
			}
		}
	}

	absensi, err := s.querier.ListTimelineAbsensiSantri(ctx, santriID)
	if err != nil {
		return nil, err
	}
	profil := &models.RiayahSantriProfil{
		Domisili: santri.Domisili,
		Timeline: make([]models.RiayahTimelineItem, 0, len(absensi)),
	}
	if santri.Usia.Valid {
		profil.Usia = &santri.Usia.Int64
	}
	if santri.MulaiBelajar.Valid {
		profil.Mulai = santri.MulaiBelajar.Time.Format("2006-01-02")
	} else if santri.TanggalDaftar.Valid {
		profil.Mulai = santri.TanggalDaftar.Time.Format("2006-01-02")
	}

	for _, a := range absensi {
		profil.Rekap.Total++
		switch a.Status {
		case "hadir":
			profil.Rekap.Hadir++
		case "telat":
			profil.Rekap.Telat++
		case "izin":
			profil.Rekap.Izin++
		case "sakit":
			profil.Rekap.Sakit++
		case "alpa":
			profil.Rekap.Alpa++
		}
		if item.BatasMateriTerakhir == "" && a.BatasMateri != "" {
			item.BatasMateriTerakhir = a.BatasMateri
		}
		judul := a.NamaKelas
		if a.PertemuanLevelKe > 0 {
			judul = fmt.Sprintf("Pertemuan %d · %s", a.PertemuanLevelKe, a.NamaKelas)
		}
		profil.Timeline = append(profil.Timeline, models.RiayahTimelineItem{
			Jenis:   "pertemuan",
			ID:      a.ID,
			Tanggal: a.Tanggal.Format("2006-01-02"),
			Judul:   judul,
			Status:  a.Status,
			Isi:     a.Catatan,
			Materi:  a.Materi,
			Batas:   a.BatasMateri,
		})
	}

	catatan, err := s.querier.ListRiayahByTarget(ctx, queries.ListRiayahByTargetParams{TargetType: "santri", TargetID: santriID})
	if err != nil {
		return nil, err
	}
	for _, c := range catatan {
		profil.Timeline = append(profil.Timeline, models.RiayahTimelineItem{
			Jenis:     "catatan",
			ID:        c.ID,
			Tanggal:   c.CreatedAt.Format("2006-01-02"),
			Waktu:     c.CreatedAt.Format("15:04"),
			Judul:     "Catatan riayah",
			Isi:       c.Catatan,
			Penulis:   c.PenulisNama,
			BisaHapus: viewer.CanWrite && bisaKelolaCatatan(viewer, c.GuruID),
		})
	}

	urutkanTimeline(profil.Timeline)
	profil.Santri = item
	return profil, nil
}

// bisaKelolaCatatan mengikuti aturan lama: guru hanya catatannya sendiri,
// admin (tanpa profil guru) boleh semua catatan.
func bisaKelolaCatatan(viewer models.RiayahViewer, guruID sql.NullInt64) bool {
	if viewer.GuruID == nil {
		return true
	}
	return guruID.Valid && guruID.Int64 == *viewer.GuruID
}

func urutkanTimeline(items []models.RiayahTimelineItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Tanggal != items[j].Tanggal {
			return items[i].Tanggal > items[j].Tanggal
		}
		return items[i].Waktu > items[j].Waktu
	})
}

// TambahCatatan menulis catatan riayah untuk santri.
func (s *RiayahSantriService) TambahCatatan(viewer models.RiayahViewer, santriID int64, catatan string) error {
	if !viewer.CanWrite {
		return ErrRiayahAksesDitolak
	}
	catatan = strings.TrimSpace(catatan)
	if catatan == "" {
		return fmt.Errorf("catatan wajib diisi")
	}
	if _, err := s.EnsureCanAccessSantri(viewer, santriID); err != nil {
		return err
	}
	ctx := context.Background()
	author := sql.NullInt64{Int64: viewer.UserID, Valid: true}
	if viewer.GuruID == nil {
		_, err := s.querier.CreateUserRiayah(ctx, queries.CreateUserRiayahParams{AuthorUserID: author, TargetType: "santri", TargetID: santriID, Catatan: catatan})
		return err
	}
	_, err := s.querier.CreateRiayah(ctx, queries.CreateRiayahParams{
		GuruID:       sql.NullInt64{Int64: *viewer.GuruID, Valid: true},
		AuthorUserID: author,
		TargetType:   "santri",
		TargetID:     santriID,
		Catatan:      catatan,
	})
	return err
}

// HapusCatatan menghapus catatan riayah milik viewer.
func (s *RiayahSantriService) HapusCatatan(viewer models.RiayahViewer, santriID, catatanID int64) error {
	if !viewer.CanWrite {
		return ErrRiayahAksesDitolak
	}
	if _, err := s.EnsureCanAccessSantri(viewer, santriID); err != nil {
		return err
	}
	ctx := context.Background()
	c, err := s.querier.GetRiayahByID(ctx, catatanID)
	if err != nil || c.TargetType != "santri" || c.TargetID != santriID {
		return fmt.Errorf("catatan riayah tidak ditemukan")
	}
	if !bisaKelolaCatatan(viewer, c.GuruID) {
		return ErrRiayahAksesDitolak
	}
	return s.querier.DeleteRiayahByID(ctx, catatanID)
}
