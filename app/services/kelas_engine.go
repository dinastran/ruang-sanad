package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

const DefaultKapasitas = 10

type KelasEngineService struct {
	querier *queries.Querier
}

func NewKelasEngineService(querier *queries.Querier) *KelasEngineService {
	return &KelasEngineService{querier: querier}
}

func (s *KelasEngineService) HitungTipe(kelasKode string) string {
	kode := strings.TrimSpace(strings.ToUpper(kelasKode))
	switch {
	case strings.HasPrefix(kode, "SP"):
		return "Semi Private"
	case strings.HasPrefix(kode, "P"):
		return "Private"
	case strings.HasPrefix(kode, "R"):
		return "Reguler"
	}
	return ""
}

func (s *KelasEngineService) HitungFrekuensi(kelasKode string) string {
	kode := strings.ToUpper(kelasKode)
	switch {
	case strings.Contains(kode, "16X"):
		return "16x pertemuan"
	case strings.Contains(kode, "4X"):
		return "4x pertemuan"
	case strings.Contains(kode, "2X"):
		return "2x/pekan"
	case strings.Contains(kode, "1X"):
		return "1x/pekan"
	}
	return ""
}

// HitungIsLengkap decides whether a santri has enough kelas-placement data to be
// assigned to a class. It intentionally does NOT require tipe/frekuensi: those are
// derived from kelas_kode (a CS field) and many kode formats don't encode them, so
// requiring them would leave santri stuck in "Perlu Dilengkapi" forever even after
// Admin Kelas fills level & jadwal. The fields below are exactly what forms a kelas.
func (s *KelasEngineService) HitungIsLengkap(nama, angkatan, level, jadwal, jenisKelamin string) bool {
	return nama != "" && angkatan != "" && level != "" && jadwal != "" && jenisKelamin != ""
}

// joinSegments joins the non-empty parts with " | " so class keys/names stay clean
// even when optional segments (tipe, frekuensi) are absent.
func joinSegments(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, " | ")
}

// BuatKunciKelas builds the grouping key. It keys on kelas_kode (which encodes
// tipe + frekuensi) and the level KODE (stable), so two santri with the same
// placement land in the same class bucket.
func (s *KelasEngineService) BuatKunciKelas(kelasKode, jenisKelamin, levelKode, frekuensi, jadwal, angkatan string) string {
	return joinSegments(kelasKode, jenisKelamin, levelKode, frekuensi, jadwal, angkatan)
}

// namaKelas renders the human-readable class name using the level NAMA (e.g.
// "TQ A") rather than its kode: JK | KodeKelas | Level | Frekuensi | Jadwal | Angkatan.
func namaKelas(jenisKelamin, kelasKode, levelNama, frekuensi, jadwal, angkatan string) string {
	return joinSegments(jenisKelamin, kelasKode, levelNama, frekuensi, jadwal, angkatan)
}

// resolveLevelNama maps a level kode to its display nama, falling back to the
// kode itself when the level isn't found in master data.
func (s *KelasEngineService) resolveLevelNama(ctx context.Context, levelKode string) string {
	if levelKode == "" {
		return ""
	}
	if lv, err := s.querier.GetLevelByKode(ctx, levelKode); err == nil && lv.Nama != "" {
		return lv.Nama
	}
	return levelKode
}

func (s *KelasEngineService) ProcessSantri(ctx context.Context, santri *queries.Santri) error {
	tipe := s.HitungTipe(santri.KelasKode)
	frekuensi := s.HitungFrekuensi(santri.KelasKode)
	isLengkap := s.HitungIsLengkap(santri.Nama, santri.Angkatan, santri.Level, santri.Jadwal, santri.JenisKelamin)

	var kelasID sql.NullInt64

	if isLengkap {
		levelNama := s.resolveLevelNama(ctx, santri.Level)
		kunciKelas := s.BuatKunciKelas(santri.KelasKode, santri.JenisKelamin, santri.Level, frekuensi, santri.Jadwal, santri.Angkatan)
		assignedID, err := s.assignKeKelas(ctx, kunciKelas, santri.KelasKode, tipe, santri.JenisKelamin, santri.Level, levelNama, frekuensi, santri.Jadwal, santri.Angkatan)
		if err != nil {
			return fmt.Errorf("assign ke kelas: %w", err)
		}
		kelasID = sql.NullInt64{Int64: assignedID, Valid: true}
	}

	if santri.KelasID.Valid && (!kelasID.Valid || santri.KelasID.Int64 != kelasID.Int64) {
		_ = s.querier.DecrementJumlahSantri(ctx, santri.KelasID.Int64)
		// Remove the old class if it is now empty, so moving a santri (e.g. after a
		// gender change) doesn't leave dangling empty classes in the list.
		if old, err := s.querier.GetKelasByID(ctx, santri.KelasID.Int64); err == nil && old.JumlahSantri <= 0 {
			_ = s.querier.DeleteKelas(ctx, santri.KelasID.Int64)
		}
	}

	now := time.Now()
	return s.querier.UpdateSantriEngine(ctx, queries.UpdateSantriEngineParams{
		Tipe:      tipe,
		Frekuensi: frekuensi,
		IsLengkap: boolToInt64(isLengkap),
		KelasID:   kelasID,
		UpdatedAt: now,
		ID:        santri.ID,
	})
}

func (s *KelasEngineService) assignKeKelas(ctx context.Context, kunciKelas, kelasKode, tipe, jenisKelamin, levelKode, levelNama, frekuensi, jadwal, angkatan string) (int64, error) {
	kelasList, err := s.querier.FindKelasByKunci(ctx, kunciKelas)
	if err != nil {
		return 0, err
	}

	for _, k := range kelasList {
		if k.JumlahSantri < DefaultKapasitas {
			if err := s.querier.IncrementJumlahSantri(ctx, k.ID); err != nil {
				return 0, err
			}
			return k.ID, nil
		}
	}

	var subIndex int64 = 1
	if len(kelasList) > 0 {
		subIndex = kelasList[len(kelasList)-1].SubIndex + 1
	}

	nama := namaKelas(jenisKelamin, kelasKode, levelNama, frekuensi, jadwal, angkatan)
	if subIndex > 1 {
		nama = fmt.Sprintf("%s — Kelas %d", nama, subIndex)
	}

	result, err := s.querier.CreateKelas(ctx, queries.CreateKelasParams{
		KunciKelas:   kunciKelas,
		Angkatan:     angkatan,
		Tipe:         tipe,
		JenisKelamin: jenisKelamin,
		Level:        levelKode,
		Frekuensi:    frekuensi,
		Jadwal:       jadwal,
		SubIndex:     subIndex,
		NamaKelas:    nama,
		GuruID:       sql.NullInt64{Valid: false},
		Kapasitas:    DefaultKapasitas,
		JumlahSantri: 1,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *KelasEngineService) PindahkanSantri(ctx context.Context, santriID, kelasTujuanID int64) error {
	santri, err := s.querier.GetSantriByID(ctx, santriID)
	if err != nil {
		return err
	}

	kelasTujuan, err := s.querier.GetKelasByID(ctx, kelasTujuanID)
	if err != nil {
		return err
	}
	if kelasTujuan.JumlahSantri >= DefaultKapasitas {
		return fmt.Errorf("kelas tujuan sudah penuh (kapasitas %d)", DefaultKapasitas)
	}

	if santri.KelasID.Valid {
		if err := s.querier.DecrementJumlahSantri(ctx, santri.KelasID.Int64); err != nil {
			return err
		}
	}

	now := time.Now()
	if err := s.querier.UpdateSantriKelas(ctx, queries.UpdateSantriKelasParams{
		KelasID:   sql.NullInt64{Int64: kelasTujuanID, Valid: true},
		UpdatedAt: now,
		ID:        santriID,
	}); err != nil {
		return err
	}

	return s.querier.IncrementJumlahSantri(ctx, kelasTujuanID)
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
