package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	PerubahanLevelKelas  = "level_kelas"
	PerubahanJadwalKelas = "jadwal_kelas"
	PerubahanLevelSantri = "level_santri"
)

// KelasPerubahanService handles Admin Kelas changes to a class's level or
// schedule. Both are part of kunci_kelas, so the class identity (kunci,
// sub_index, nama) is rebuilt while the class row, guru, and santri stay put.
type KelasPerubahanService struct {
	querier *queries.Querier
	engine  *KelasEngineService
}

func NewKelasPerubahanService(querier *queries.Querier, engine *KelasEngineService) *KelasPerubahanService {
	return &KelasPerubahanService{querier: querier, engine: engine}
}

// GantiLevelKelas moves a whole class to another level. Meeting numbering
// restarts at 1 for the new level (pertemuan_level_ke) while the internal
// pertemuan_ke keeps counting so billing periods are unaffected.
func (s *KelasPerubahanService) GantiLevelKelas(kelasID int64, levelKode string, userID int64) error {
	levelKode = strings.TrimSpace(levelKode)
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
	levelBaru, err := q.GetLevelByKode(ctx, levelKode)
	if err != nil {
		return fmt.Errorf("level tidak ditemukan di master data")
	}
	if kelas.Level == levelBaru.Kode {
		return fmt.Errorf("level baru sama dengan level kelas saat ini")
	}
	if _, err := q.GetActivePertemuanByKelas(ctx, kelasID); err == nil {
		return fmt.Errorf("masih ada pertemuan yang berlangsung; selesaikan dulu sebelum ganti level")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	next, err := q.GetNextPertemuanKe(ctx, kelasID)
	if err != nil {
		return err
	}
	levelPertemuanAwal := next - 1

	if err := s.updateIdentitas(ctx, q, kelas, levelBaru.Kode, kelas.Jadwal, levelPertemuanAwal); err != nil {
		return err
	}
	levelLama := s.engine.WithQuerier(q).resolveLevelNama(ctx, kelas.Level)
	levelBaruNama := s.engine.WithQuerier(q).resolveLevelNama(ctx, levelBaru.Kode)
	if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
		KelasID:     sql.NullInt64{Int64: kelasID, Valid: true},
		Jenis:       PerubahanLevelKelas,
		NilaiLama:   levelLama,
		NilaiBaru:   levelBaruNama,
		PertemuanKe: levelPertemuanAwal,
		DibuatOleh:  nullUserID(userID),
	}); err != nil {
		return err
	}
	if err := notifyGuruKelas(ctx, q, kelas.GuruID, kelasID, userID, "Level kelas berubah",
		fmt.Sprintf("Level kelas %s diganti dari %s ke %s. Pertemuan berikutnya menjadi pertemuan ke-1 di level %s.", kelas.NamaKelas, levelLama, levelBaruNama, levelBaruNama)); err != nil {
		return err
	}
	return tx.Commit()
}

// GantiJadwalKelas permanently replaces the class's routine schedule. Already
// scheduled sessions are left as-is but flagged so guru/admin can reschedule.
func (s *KelasPerubahanService) GantiJadwalKelas(kelasID int64, jadwal string, userID int64) error {
	jadwal = strings.TrimSpace(jadwal)
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
	if _, err := q.GetJadwalByNama(ctx, jadwal); err != nil {
		return fmt.Errorf("jadwal tidak ditemukan di master data")
	}
	if kelas.Jadwal == jadwal {
		return fmt.Errorf("jadwal baru sama dengan jadwal kelas saat ini")
	}
	state, err := q.GetKelasPerubahanState(ctx, kelasID)
	if err != nil {
		return err
	}
	if err := s.updateIdentitas(ctx, q, kelas, kelas.Level, jadwal, state.LevelPertemuanAwal); err != nil {
		return err
	}
	if err := q.MarkJadwalPertemuanKelasBerubah(ctx, kelasID); err != nil {
		return err
	}
	if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
		KelasID:    sql.NullInt64{Int64: kelasID, Valid: true},
		Jenis:      PerubahanJadwalKelas,
		NilaiLama:  kelas.Jadwal,
		NilaiBaru:  jadwal,
		DibuatOleh: nullUserID(userID),
	}); err != nil {
		return err
	}
	if err := notifyGuruKelas(ctx, q, kelas.GuruID, kelasID, userID, "Jadwal kelas berubah",
		fmt.Sprintf("Jadwal rutin kelas %s diganti dari %s menjadi %s. Periksa kembali sesi yang sudah terjadwal.", kelas.NamaKelas, kelas.Jadwal, jadwal)); err != nil {
		return err
	}
	return tx.Commit()
}

// GantiLevelSantri moves selected santri from kelasAsalID to a class of a
// different level, chosen by the admin or newly created.
func (s *KelasPerubahanService) GantiLevelSantri(kelasAsalID int64, req models.GantiLevelSantriRequest, userID int64) (int64, error) {
	req.SantriIDs = uniqueIDs(req.SantriIDs)
	if len(req.SantriIDs) == 0 {
		return 0, fmt.Errorf("pilih minimal satu santri")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)
	engine := s.engine.WithQuerier(q)

	kelasAsal, err := q.GetKelasByID(ctx, kelasAsalID)
	if err != nil {
		return 0, fmt.Errorf("kelas asal tidak ditemukan")
	}
	if err := ensureTanpaPertemuanBerlangsung(ctx, q, sql.NullInt64{Int64: kelasAsalID, Valid: true}); err != nil {
		return 0, err
	}
	for _, santriID := range req.SantriIDs {
		santri, err := q.GetSantriByID(ctx, santriID)
		if err != nil || !santri.KelasID.Valid || santri.KelasID.Int64 != kelasAsalID {
			return 0, fmt.Errorf("santri tidak terdaftar di kelas ini")
		}
	}

	var kelasTujuanID int64
	if req.BuatKelasBaru {
		kelasTujuanID, err = s.buatKelasLevelBaru(ctx, q, kelasAsal, strings.TrimSpace(req.Level), int64(len(req.SantriIDs)))
		if err != nil {
			return 0, err
		}
	} else {
		if req.KelasTujuanID == 0 || req.KelasTujuanID == kelasAsalID {
			return 0, fmt.Errorf("pilih kelas tujuan")
		}
		kelasTujuanID = req.KelasTujuanID
		if err := ensureTanpaPertemuanBerlangsung(ctx, q, sql.NullInt64{Int64: kelasTujuanID, Valid: true}); err != nil {
			return 0, fmt.Errorf("kelas tujuan: %w", err)
		}
	}

	kelasTujuan, err := q.GetKelasByID(ctx, kelasTujuanID)
	if err != nil {
		return 0, fmt.Errorf("kelas tujuan tidak ditemukan")
	}
	if kelasTujuan.Level == kelasAsal.Level {
		return 0, fmt.Errorf("level kelas tujuan sama dengan level saat ini; gunakan Pindahkan untuk pindah kelas biasa")
	}
	if !req.BuatKelasBaru && kelasTujuan.JumlahSantri+int64(len(req.SantriIDs)) > kelasTujuan.Kapasitas {
		return 0, fmt.Errorf("kelas tujuan tidak cukup (sisa %d kursi)", max(kelasTujuan.Kapasitas-kelasTujuan.JumlahSantri, 0))
	}

	levelLama := engine.resolveLevelNama(ctx, kelasAsal.Level)
	levelBaru := engine.resolveLevelNama(ctx, kelasTujuan.Level)
	nama := make([]string, 0, len(req.SantriIDs))
	for _, santriID := range req.SantriIDs {
		santri, err := q.GetSantriByID(ctx, santriID)
		if err != nil {
			return 0, err
		}
		if err := engine.PindahkanSantri(ctx, santriID, kelasTujuanID); err != nil {
			return 0, fmt.Errorf("%s: %w", santri.Nama, err)
		}
		if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
			KelasID:       sql.NullInt64{Int64: kelasAsalID, Valid: true},
			SantriID:      sql.NullInt64{Int64: santriID, Valid: true},
			KelasTujuanID: sql.NullInt64{Int64: kelasTujuanID, Valid: true},
			Jenis:         PerubahanLevelSantri,
			NilaiLama:     levelLama,
			NilaiBaru:     levelBaru,
			DibuatOleh:    nullUserID(userID),
		}); err != nil {
			return 0, err
		}
		nama = append(nama, santri.Nama)
	}

	daftar := strings.Join(nama, ", ")
	if err := notifyGuruKelas(ctx, q, kelasAsal.GuruID, kelasAsalID, userID, "Santri naik level",
		fmt.Sprintf("%s dipindahkan dari kelas %s ke kelas level %s (%s).", daftar, kelasAsal.NamaKelas, levelBaru, kelasTujuan.NamaKelas)); err != nil {
		return 0, err
	}
	if kelasTujuan.GuruID.Valid && (!kelasAsal.GuruID.Valid || kelasTujuan.GuruID.Int64 != kelasAsal.GuruID.Int64) {
		if err := notifyGuruKelas(ctx, q, kelasTujuan.GuruID, kelasTujuanID, userID, "Santri baru di kelas Anda",
			fmt.Sprintf("%s masuk ke kelas %s setelah naik level dari %s.", daftar, kelasTujuan.NamaKelas, levelLama)); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return kelasTujuanID, nil
}

func (s *KelasPerubahanService) ListRiwayat(kelasID int64) ([]models.KelasPerubahanResponse, error) {
	rows, err := s.querier.ListKelasPerubahanByKelas(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]models.KelasPerubahanResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.KelasPerubahanResponse{
			ID:              r.ID,
			Jenis:           r.Jenis,
			NilaiLama:       r.NilaiLama,
			NilaiBaru:       r.NilaiBaru,
			PertemuanKe:     r.PertemuanKe,
			SantriNama:      r.SantriNama,
			KelasAsalID:     nullInt64Ptr(r.KelasID),
			KelasAsalNama:   r.KelasAsalNama,
			KelasTujuanID:   nullInt64Ptr(r.KelasTujuanID),
			KelasTujuanNama: r.KelasTujuanNama,
			DibuatOlehNama:  r.DibuatOlehNama,
			CreatedAt:       r.CreatedAt.In(wib).Format("2006-01-02 15:04"),
		})
	}
	return out, nil
}

// updateIdentitas rebuilds kunci/sub_index/nama for a class with a new level
// and/or jadwal and syncs the santri placement fields so the kelas engine keeps
// recognising them as members of this class. When the new kunci is already used
// by another class, this class stays separate under the next free sub_index.
func (s *KelasPerubahanService) updateIdentitas(ctx context.Context, q *queries.Querier, kelas queries.GetKelasByIDRow, levelKode, jadwal string, levelPertemuanAwal int64) error {
	engine := s.engine.WithQuerier(q)
	kelasKode, err := kelasKodeKelas(ctx, q, kelas)
	if err != nil {
		return err
	}
	kunci := engine.BuatKunciKelas(kelasKode, kelas.JenisKelamin, levelKode, kelas.Frekuensi, jadwal, kelas.Angkatan)
	subIndex := kelas.SubIndex
	if kunci != kelas.KunciKelas {
		next, err := q.GetNextKelasSubIndexByKunci(ctx, kunci)
		if err != nil {
			return err
		}
		subIndex = next
	}
	nama := namaKelas(kelas.JenisKelamin, kelasKode, engine.resolveLevelNama(ctx, levelKode), kelas.Frekuensi, jadwal, kelas.Angkatan)
	if subIndex > 1 {
		nama = fmt.Sprintf("%s — Kelas %d", nama, subIndex)
	}
	if err := q.UpdateKelasIdentitas(ctx, queries.UpdateKelasIdentitasParams{
		Level:              levelKode,
		Jadwal:             jadwal,
		KunciKelas:         kunci,
		SubIndex:           subIndex,
		NamaKelas:          nama,
		LevelPertemuanAwal: levelPertemuanAwal,
		ID:                 kelas.ID,
	}); err != nil {
		return err
	}
	return q.SyncSantriLevelJadwalByKelas(ctx, queries.SyncSantriLevelJadwalByKelasParams{
		Level:     levelKode,
		Jadwal:    jadwal,
		UpdatedAt: time.Now(),
		KelasID:   sql.NullInt64{Int64: kelas.ID, Valid: true},
	})
}

// buatKelasLevelBaru opens an empty class that mirrors the origin class but with
// a different level. It always creates a new row (next free sub_index) because
// the admin explicitly asked for a new class.
//
// The new class continues the origin's internal pertemuan_ke so the moved
// santri's billing periods (bulan_ke) keep advancing instead of colliding with
// periods already billed, while its per-level numbering starts at 1.
func (s *KelasPerubahanService) buatKelasLevelBaru(ctx context.Context, q *queries.Querier, asal queries.GetKelasByIDRow, levelKode string, jumlahSantri int64) (int64, error) {
	level, err := q.GetLevelByKode(ctx, levelKode)
	if err != nil {
		return 0, fmt.Errorf("level tidak ditemukan di master data")
	}
	if level.Kode == asal.Level {
		return 0, fmt.Errorf("level kelas baru sama dengan level saat ini")
	}
	engine := s.engine.WithQuerier(q)
	kelasKode, err := kelasKodeKelas(ctx, q, asal)
	if err != nil {
		return 0, err
	}
	kunci := engine.BuatKunciKelas(kelasKode, asal.JenisKelamin, level.Kode, asal.Frekuensi, asal.Jadwal, asal.Angkatan)
	subIndex, err := q.GetNextKelasSubIndexByKunci(ctx, kunci)
	if err != nil {
		return 0, err
	}
	nama := namaKelas(asal.JenisKelamin, kelasKode, engine.resolveLevelNama(ctx, level.Kode), asal.Frekuensi, asal.Jadwal, asal.Angkatan)
	if subIndex > 1 {
		nama = fmt.Sprintf("%s — Kelas %d", nama, subIndex)
	}
	result, err := q.CreateKelas(ctx, queries.CreateKelasParams{
		KunciKelas:   kunci,
		Angkatan:     asal.Angkatan,
		Tipe:         asal.Tipe,
		JenisKelamin: asal.JenisKelamin,
		Level:        level.Kode,
		Frekuensi:    asal.Frekuensi,
		Jadwal:       asal.Jadwal,
		SubIndex:     subIndex,
		NamaKelas:    nama,
		Kapasitas:    DefaultKapasitas,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	nextAsal, err := q.GetNextPertemuanKe(ctx, asal.ID)
	if err != nil {
		return 0, err
	}
	if err := q.SetKelasAwalPertemuan(ctx, queries.SetKelasAwalPertemuanParams{
		PertemuanTerakhir:  nextAsal - 1,
		LevelPertemuanAwal: nextAsal - 1,
		Kapasitas:          max(DefaultKapasitas, jumlahSantri),
		ID:                 id,
	}); err != nil {
		return 0, err
	}
	return id, nil
}

// kelasKodeDariKunci extracts the kelas_kode segment from a kunci_kelas. The
// first segment is the kode unless the kode was empty, in which case the first
// segment is the gender.
func kelasKodeDariKunci(kunci, jenisKelamin string) string {
	first, _, found := strings.Cut(kunci, " | ")
	if !found {
		return ""
	}
	first = strings.TrimSpace(first)
	if first == jenisKelamin {
		return ""
	}
	return first
}

// kelasKodeKelas returns the kelas_kode encoded in a class's kunci. Legacy keys
// without segments fall back to a member santri's kelas_kode so the rebuilt
// kunci still matches what the kelas engine computes for those santri.
func kelasKodeKelas(ctx context.Context, q *queries.Querier, kelas queries.GetKelasByIDRow) (string, error) {
	if strings.Contains(kelas.KunciKelas, " | ") {
		return kelasKodeDariKunci(kelas.KunciKelas, kelas.JenisKelamin), nil
	}
	santri, err := q.GetSantriByKelasID(ctx, sql.NullInt64{Int64: kelas.ID, Valid: true})
	if err != nil {
		return "", err
	}
	for _, st := range santri {
		if st.KelasKode != "" {
			return st.KelasKode, nil
		}
	}
	return "", nil
}

func uniqueIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func notifyGuruKelas(ctx context.Context, q *queries.Querier, guruID sql.NullInt64, kelasID, actorID int64, title, message string) error {
	if !guruID.Valid {
		return nil
	}
	guru, err := q.GetGuruByID(ctx, guruID.Int64)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !guru.UserID.Valid {
		return nil
	}
	_, err = q.CreateNotification(ctx, queries.CreateNotificationParams{
		UserID:        guru.UserID.Int64,
		Type:          "perubahan_kelas",
		Title:         title,
		Message:       message,
		ActionUrl:     fmt.Sprintf("/app/guru/kelas/%d", kelasID),
		ReferenceType: "kelas",
		ReferenceID:   sql.NullInt64{Int64: kelasID, Valid: true},
		CreatedBy:     nullUserID(actorID),
	})
	return err
}

func nullUserID(userID int64) sql.NullInt64 {
	return sql.NullInt64{Int64: userID, Valid: userID > 0}
}
