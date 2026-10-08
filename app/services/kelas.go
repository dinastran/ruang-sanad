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

func (s *KelasService) GetKoreksiPertemuanState(kelasID int64) (models.KoreksiPertemuanState, error) {
	ctx := context.Background()
	kelas, err := s.querier.GetKelasByID(ctx, kelasID)
	if err != nil {
		return models.KoreksiPertemuanState{}, err
	}
	bounds, err := s.querier.GetPertemuanRebaseBounds(ctx, kelasID)
	if err != nil {
		return models.KoreksiPertemuanState{}, err
	}
	adaBerlangsung := false
	if _, err := s.querier.GetActivePertemuanByKelas(ctx, kelasID); err == nil {
		adaBerlangsung = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return models.KoreksiPertemuanState{}, err
	}
	return models.KoreksiPertemuanState{
		AnchorSebelumSistem:     kelas.PertemuanTerakhir,
		JumlahPertemuan:         bounds.Total,
		PertemuanGlobalTerakhir: bounds.MaxPertemuanKe,
		AdaPertemuanBerlangsung: adaBerlangsung,
	}, nil
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

func (s *KelasService) SetKapasitas(kelasID, kapasitas int64) error {
	if kapasitas < 1 {
		return fmt.Errorf("kapasitas minimal 1 peserta")
	}

	kelas, err := s.querier.GetKelasByID(context.Background(), kelasID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("kelas tidak ditemukan")
		}
		return err
	}
	if kapasitas < kelas.JumlahSantri {
		return fmt.Errorf("kapasitas tidak boleh lebih kecil dari jumlah santri aktif/cuti saat ini (%d)", kelas.JumlahSantri)
	}
	if kapasitas == kelas.Kapasitas {
		return nil
	}

	return s.querier.SetKelasKapasitas(context.Background(), queries.SetKelasKapasitasParams{
		Kapasitas: kapasitas,
		ID:        kelasID,
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
	hasil, err := terapkanJadwalRutin(ctx, q, kelasID, userID, req.Slots)
	if err != nil {
		return err
	}
	if !hasil.Berubah {
		return fmt.Errorf("jadwal rutin baru sama dengan jadwal saat ini")
	}
	if kelas.GuruID.Valid {
		message := fmt.Sprintf("Jadwal rutin otomatis kelas %s diubah dari %s menjadi %s.", kelas.NamaKelas, hasil.Lama, hasil.Baru)
		if hasil.MulaiBesok {
			message += " Pola baru berlaku mulai besok; jadwal hari ini tidak diubah."
		}
		if err := notifyGuruKelas(ctx, q, kelas.GuruID, kelasID, userID, "Jadwal rutin otomatis berubah", message); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type jadwalRutinHasil struct {
	Lama, Baru string
	Berubah    bool
	// MulaiBesok is true when an existing pattern was replaced, so the new
	// slots only take effect tomorrow.
	MulaiBesok bool
}

// terapkanJadwalRutin replaces a class's routine slots inside the caller's
// transaction and logs the change. An unchanged pattern is a no-op
// (Berubah=false) so callers decide whether that is an error.
func terapkanJadwalRutin(ctx context.Context, q *queries.Querier, kelasID, userID int64, slots []models.JadwalRutinSlotRequest) (jadwalRutinHasil, error) {
	existing, err := q.ListRoutineSchedulesByClass(ctx, kelasID)
	if err != nil {
		return jadwalRutinHasil{}, err
	}

	normalized := make([]models.JadwalRutinSlotRequest, 0, len(slots))
	seen := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		if slot.Hari < 1 || slot.Hari > 7 {
			return jadwalRutinHasil{}, fmt.Errorf("hari jadwal rutin tidak valid")
		}
		jam := strings.TrimSpace(slot.JamMulai)
		if err := validateScheduleTime(jam); err != nil {
			return jadwalRutinHasil{}, err
		}
		key := fmt.Sprintf("%d|%s", slot.Hari, jam)
		if _, ok := seen[key]; ok {
			return jadwalRutinHasil{}, fmt.Errorf("jadwal rutin %s pukul %s tercatat lebih dari sekali", routineDayName(slot.Hari), jam)
		}
		seen[key] = struct{}{}
		normalized = append(normalized, models.JadwalRutinSlotRequest{Hari: slot.Hari, JamMulai: jam})
	}

	oldSlots := make([]models.JadwalRutinSlotRequest, 0, len(existing))
	for _, row := range existing {
		oldSlots = append(oldSlots, models.JadwalRutinSlotRequest{Hari: row.Hari, JamMulai: row.JamMulai})
	}
	hasil := jadwalRutinHasil{Lama: formatRoutineSlots(oldSlots), Baru: formatRoutineSlots(normalized)}
	if hasil.Lama == hasil.Baru {
		return hasil, nil
	}
	hasil.Berubah = true

	today := startOfToday()
	effectiveStart := today
	if len(existing) > 0 {
		hasil.MulaiBesok = true
		effectiveStart = today.AddDate(0, 0, 1)
		if err := q.DeleteFutureRoutineOccurrences(ctx, kelasID, today); err != nil {
			return hasil, err
		}
		pendingFuture := true
		for _, row := range existing {
			if !routineDate(row.BerlakuMulai, today.Location()).After(today) {
				pendingFuture = false
				break
			}
		}
		if pendingFuture {
			if err := q.DeleteFutureRoutineSchedules(ctx, kelasID, today); err != nil {
				return hasil, err
			}
		} else if err := q.CloseActiveRoutineSchedules(ctx, kelasID, today); err != nil {
			return hasil, err
		}
	}
	for _, slot := range normalized {
		if _, err := q.CreateRoutineSchedule(ctx, kelasID, slot.Hari, slot.JamMulai, effectiveStart, sql.NullInt64{Int64: userID, Valid: userID > 0}); err != nil {
			return hasil, err
		}
	}

	if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
		KelasID:    sql.NullInt64{Int64: kelasID, Valid: true},
		Jenis:      PerubahanJadwalRutin,
		NilaiLama:  hasil.Lama,
		NilaiBaru:  hasil.Baru,
		DibuatOleh: nullUserID(userID),
	}); err != nil {
		return hasil, err
	}
	return hasil, nil
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
