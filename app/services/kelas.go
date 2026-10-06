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

type KelasService struct {
	querier *queries.Querier
}

func NewKelasService(querier *queries.Querier) *KelasService {
	return &KelasService{querier: querier}
}

func (s *KelasService) ListByAngkatan(angkatan string) ([]models.KelasResponse, error) {
	list, err := s.querier.ListKelasByAngkatan(context.Background(), angkatan)
	if err != nil {
		return nil, err
	}
	result := make([]models.KelasResponse, len(list))
	for i, k := range list {
		result[i] = k.ToResponse()
	}
	return result, nil
}

func (s *KelasService) ListAll() ([]models.KelasResponse, error) {
	list, err := s.querier.ListKelasAll(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]models.KelasResponse, len(list))
	for i, k := range list {
		result[i] = k.ToResponse()
	}
	return result, nil
}

func (s *KelasService) GetByID(id int64) (*models.KelasResponse, error) {
	k, err := s.querier.GetKelasByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	resp := k.ToResponse()
	return &resp, nil
}

func (s *KelasService) GetSantriByKelasID(kelasID int64) ([]models.SantriResponse, error) {
	list, err := s.querier.GetSantriByKelasID(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]models.SantriResponse, len(list))
	for i, s := range list {
		result[i] = s.ToResponse()
	}
	return result, nil
}

func (s *KelasService) GetDetail(kelasID int64) (*models.KelasResponse, []models.SantriResponse, error) {
	kelas, err := s.GetByID(kelasID)
	if err != nil {
		return nil, nil, err
	}

	if pertemuan, err := s.querier.GetLastPertemuanByKelas(context.Background(), kelasID); err == nil {
		kelas.PertemuanTerakhir = pertemuan.PertemuanLevelKe
		kelas.TanggalPertemuanTerakhir = pertemuan.Tanggal.Format("2006-01-02")
		kelas.MateriTerakhir = pertemuan.Materi
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	// Detail Kelas lists every status (aktif, cuti, nonaktif, tidak_lanjut);
	// the page groups them into roster and history sections.
	rows, err := s.querier.GetSantriRosterByKelasID(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, nil, err
	}
	santri := make([]models.SantriResponse, len(rows))
	for i, row := range rows {
		santri[i] = row.ToResponse()
	}

	stats, err := s.querier.CountAbsensiSantriByKelas(context.Background(), kelasID)
	if err != nil {
		return nil, nil, err
	}
	statsBySantri := make(map[int64]queries.CountAbsensiSantriByKelasRow, len(stats))
	for _, stat := range stats {
		statsBySantri[stat.SantriID] = stat
	}
	for i := range santri {
		stat, ok := statsBySantri[santri[i].ID]
		if !ok {
			continue
		}
		santri[i].TotalHadir = int64(stat.TotalHadir.Float64)
		santri[i].TotalIzin = int64(stat.TotalIzin.Float64)
		santri[i].TotalSakit = int64(stat.TotalSakit.Float64)
		santri[i].TotalAlpa = int64(stat.TotalAlpa.Float64)
		santri[i].TotalTelat = int64(stat.TotalTelat.Float64)
		if stat.Total > 0 {
			santri[i].PersenHadir = float64(santri[i].TotalHadir) / float64(stat.Total) * 100
		}
	}

	return kelas, santri, nil
}

func (s *KelasService) AssignGuru(kelasID, guruID int64) error {
	return s.querier.AssignGuru(context.Background(), queries.AssignGuruParams{
		GuruID: sql.NullInt64{Int64: guruID, Valid: true},
		ID:     kelasID,
	})
}

// SetPertemuanTerakhir sets the real last meeting before this class starts
// recording meetings in the application. It cannot change once records exist.
func (s *KelasService) SetPertemuanTerakhir(kelasID, pertemuanTerakhir int64) error {
	if pertemuanTerakhir < 0 {
		return fmt.Errorf("pertemuan terakhir tidak boleh negatif")
	}
	result, err := s.querier.UpdateKelasPertemuanTerakhir(context.Background(), queries.UpdateKelasPertemuanTerakhirParams{PertemuanTerakhir: pertemuanTerakhir, ID: kelasID})
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		if _, err := s.querier.GetKelasByID(context.Background(), kelasID); err != nil {
			return fmt.Errorf("kelas tidak ditemukan")
		}
		return fmt.Errorf("nomor awal tidak dapat diubah setelah pertemuan tercatat")
	}
	return nil
}

func (s *KelasService) HasPertemuan(kelasID int64) (bool, error) {
	count, err := s.querier.CountPertemuanByKelas(context.Background(), kelasID)
	return count > 0, err
}

func (s *KelasService) SetAktif(kelasID int64, aktif bool) error {
	var v int64
	if aktif {
		v = 1
	}
	return s.querier.SetKelasAktif(context.Background(), queries.SetKelasAktifParams{
		IsAktif: v,
		ID:      kelasID,
	})
}

func (s *KelasService) SetMateriIndividual(kelasID int64, enabled bool) error {
	var value int64
	if enabled {
		value = 1
	}
	return s.querier.SetKelasMateriIndividual(context.Background(), queries.SetKelasMateriIndividualParams{
		MateriIndividual: value,
		ID:               kelasID,
	})
}

// Delete removes a class. Santri linked to it have their kelas_id set to NULL
// via the ON DELETE SET NULL foreign key, so they become unassigned.
func (s *KelasService) Delete(kelasID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	querier := s.querier.WithTx(tx)
	if err := querier.ClearSantriKelasByKelasID(ctx, queries.ClearSantriKelasByKelasIDParams{UpdatedAt: time.Now(), KelasID: sql.NullInt64{Int64: kelasID, Valid: true}}); err != nil {
		return err
	}
	if err := querier.DeleteKelas(ctx, kelasID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *KelasService) ListJadwalRutin(kelasID int64) ([]models.JadwalRutinResponse, error) {
	rows, err := s.querier.ListRoutineSchedulesByClass(context.Background(), kelasID)
	if err != nil {
		return nil, err
	}
	out := make([]models.JadwalRutinResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.JadwalRutinResponse{
			ID:           row.ID,
			KelasID:      row.KelasID,
			Hari:         row.Hari,
			JamMulai:     row.JamMulai,
			BerlakuMulai: row.BerlakuMulai.Format("2006-01-02"),
			IsAktif:      row.IsAktif == 1,
		})
	}
	return out, nil
}

func (s *KelasService) SetJadwalRutin(kelasID, userID int64, req models.SetJadwalRutinRequest) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	kelas, err := q.GetKelasByID(ctx, kelasID)
	if err != nil {
		return fmt.Errorf("kelas tidak ditemukan")
	}
	existing, err := q.ListRoutineSchedulesByClass(ctx, kelasID)
	if err != nil {
		return err
	}

	normalized := make([]models.JadwalRutinSlotRequest, 0, len(req.Slots))
	seen := make(map[string]struct{}, len(req.Slots))
	for _, slot := range req.Slots {
		if slot.Hari < 1 || slot.Hari > 7 {
			return fmt.Errorf("hari jadwal rutin tidak valid")
		}
		jam := strings.TrimSpace(slot.JamMulai)
		if err := validateScheduleTime(jam); err != nil {
			return err
		}
		key := fmt.Sprintf("%d|%s", slot.Hari, jam)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("jadwal rutin %s pukul %s tercatat lebih dari sekali", routineDayName(slot.Hari), jam)
		}
		seen[key] = struct{}{}
		normalized = append(normalized, models.JadwalRutinSlotRequest{Hari: slot.Hari, JamMulai: jam})
	}

	oldSlots := make([]models.JadwalRutinSlotRequest, 0, len(existing))
	for _, row := range existing {
		oldSlots = append(oldSlots, models.JadwalRutinSlotRequest{Hari: row.Hari, JamMulai: row.JamMulai})
	}
	oldSummary := formatRoutineSlots(oldSlots)
	newSummary := formatRoutineSlots(normalized)
	if oldSummary == newSummary {
		return fmt.Errorf("jadwal rutin baru sama dengan jadwal saat ini")
	}

	today := startOfToday()
	effectiveStart := today
	if len(existing) > 0 {
		effectiveStart = today.AddDate(0, 0, 1)
		if err := q.CloseActiveRoutineSchedules(ctx, kelasID, today); err != nil {
			return err
		}
		if err := q.DeleteFutureRoutineOccurrences(ctx, kelasID, today); err != nil {
			return err
		}
	}
	for _, slot := range normalized {
		if _, err := q.CreateRoutineSchedule(ctx, kelasID, slot.Hari, slot.JamMulai, effectiveStart, sql.NullInt64{Int64: userID, Valid: userID > 0}); err != nil {
			return err
		}
	}

	if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
		KelasID:    sql.NullInt64{Int64: kelasID, Valid: true},
		Jenis:      PerubahanJadwalRutin,
		NilaiLama:  oldSummary,
		NilaiBaru:  newSummary,
		DibuatOleh: nullUserID(userID),
	}); err != nil {
		return err
	}
	if kelas.GuruID.Valid {
		message := fmt.Sprintf("Jadwal rutin otomatis kelas %s diubah dari %s menjadi %s.", kelas.NamaKelas, oldSummary, newSummary)
		if len(existing) > 0 {
			message += " Pola baru berlaku mulai besok; jadwal hari ini tidak diubah."
		}
		if err := notifyGuruKelas(ctx, q, kelas.GuruID, kelasID, userID, "Jadwal rutin otomatis berubah", message); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func routineDayName(day int64) string {
	return map[int64]string{1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Ahad"}[day]
}

func formatRoutineSlots(slots []models.JadwalRutinSlotRequest) string {
	if len(slots) == 0 {
		return "Nonaktif"
	}
	copySlots := append([]models.JadwalRutinSlotRequest(nil), slots...)
	sort.Slice(copySlots, func(i, j int) bool {
		if copySlots[i].Hari != copySlots[j].Hari {
			return copySlots[i].Hari < copySlots[j].Hari
		}
		return copySlots[i].JamMulai < copySlots[j].JamMulai
	})
	parts := make([]string, 0, len(copySlots))
	for _, slot := range copySlots {
		parts = append(parts, fmt.Sprintf("%s %s", routineDayName(slot.Hari), slot.JamMulai))
	}
	return strings.Join(parts, ", ")
}

func (s *KelasService) CountTotal() (int64, error) {
	return s.querier.CountKelasTotal(context.Background())
}

func (s *KelasService) CountByAngkatan() ([]queries.CountKelasByAngkatanRow, error) {
	return s.querier.CountKelasByAngkatan(context.Background())
}
