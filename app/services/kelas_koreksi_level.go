package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Koreksi nomor di level saat ini.
//
// A class has two numberings: the global pertemuan_ke (continues across level
// changes; drives billing periods and santri start meetings) and the per-level
// pertemuan_level_ke shown to guru/admin, which restarts at 1 after a level
// change. kelas.level_pertemuan_awal is the global number right before the
// current level begins (per-level = global - level_pertemuan_awal).
//
// When Admin Kelas changes the level late, the first meetings of the new level
// were recorded under the old level label. This correction moves the start of
// the current level so the per-level number of the last meeting becomes the
// real one: meetings in between are relabelled to the current level (or back
// to the previous level when the level actually started later). Global
// numbers, tagihan, absensi, and santri start meetings are not changed.

type koreksiLevelKonteks struct {
	kelas          queries.GetKelasByIDRow
	levelNama      string
	levelAwal      int64
	terakhirGlobal int64
	batasBawah     int64 // previous level change boundary, or 0
	perubahanID    int64 // latest level_kelas record at levelAwal, or 0
	levelSebelum   string
	offsetSebelum  int64
	pertemuan      []queries.ListPertemuanKoreksiLevelRow
}

var errKoreksiLevelTidakTersedia = errors.New("kelas ini belum pernah ganti level; gunakan Koreksi nomor total bila nomornya salah")

func (s *KelasPerubahanService) koreksiLevelKonteks(ctx context.Context, q *queries.Querier, kelasID int64) (*koreksiLevelKonteks, error) {
	kelas, err := q.GetKelasByID(ctx, kelasID)
	if err != nil {
		return nil, fmt.Errorf("kelas tidak ditemukan")
	}
	state, err := q.GetKelasPerubahanState(ctx, kelasID)
	if err != nil {
		return nil, err
	}
	if state.LevelPertemuanAwal <= 0 {
		return nil, errKoreksiLevelTidakTersedia
	}
	next, err := q.GetNextPertemuanKe(ctx, kelasID)
	if err != nil {
		return nil, err
	}
	k := &koreksiLevelKonteks{
		kelas:          kelas,
		levelNama:      s.engine.WithQuerier(q).resolveLevelNama(ctx, kelas.Level),
		levelAwal:      state.LevelPertemuanAwal,
		terakhirGlobal: next - 1,
	}
	if k.terakhirGlobal < k.levelAwal {
		k.terakhirGlobal = k.levelAwal
	}
	batas, err := q.ListLevelChangeBoundaries(ctx, kelasID)
	if err != nil {
		return nil, err
	}
	for i, b := range batas {
		if b.PertemuanKe == k.levelAwal && k.perubahanID == 0 {
			k.perubahanID = b.ID
			k.levelSebelum = b.NilaiLama
			if i+1 < len(batas) {
				k.batasBawah = batas[i+1].PertemuanKe
			}
		}
	}
	if k.perubahanID == 0 {
		// No matching record (e.g. a class split off with a new level):
		// only boundaries below the current start limit the correction.
		for _, b := range batas {
			if b.PertemuanKe < k.levelAwal {
				k.batasBawah = b.PertemuanKe
				break
			}
		}
	}
	k.pertemuan, err = q.ListPertemuanKoreksiLevel(ctx, kelasID)
	if err != nil {
		return nil, err
	}
	// Meetings moved back to the previous level continue its numbering: use
	// the last meeting recorded before the current level, else its boundary.
	k.offsetSebelum = k.batasBawah
	for _, p := range k.pertemuan {
		if p.PertemuanKe <= k.levelAwal && p.PertemuanKe > k.batasBawah {
			k.offsetSebelum = p.PertemuanKe - p.PertemuanLevelKe
			if strings.TrimSpace(k.levelSebelum) == "" {
				k.levelSebelum = p.LevelNama
			}
		}
	}
	return k, nil
}

// maksTerakhir is the largest per-level number allowed: the current level may
// start at the earliest right after the previous level change (keeping at
// least one meeting for that level), or at global 0 when there is none.
func (k *koreksiLevelKonteks) maksTerakhir() int64 {
	if k.batasBawah > 0 {
		return k.terakhirGlobal - k.batasBawah - 1
	}
	return k.terakhirGlobal
}

// GetKoreksiLevelState returns the data for the correction dialog, or
// (nil, nil) when the class never changed level.
func (s *KelasPerubahanService) GetKoreksiLevelState(kelasID int64) (*models.KoreksiLevelState, error) {
	ctx := context.Background()
	k, err := s.koreksiLevelKonteks(ctx, s.querier, kelasID)
	if errors.Is(err, errKoreksiLevelTidakTersedia) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &models.KoreksiLevelState{
		LevelNama:        k.levelNama,
		TerakhirGlobal:   k.terakhirGlobal,
		LevelAwal:        k.levelAwal,
		TerakhirSekarang: k.terakhirGlobal - k.levelAwal,
		MaksTerakhir:     k.maksTerakhir(),
		LevelSebelum:     k.levelSebelum,
		OffsetSebelum:    k.offsetSebelum,
		Pertemuan:        []models.KoreksiLevelPertemuan{},
	}
	for _, p := range k.pertemuan {
		if p.PertemuanKe > k.batasBawah {
			out.Pertemuan = append(out.Pertemuan, models.KoreksiLevelPertemuan{
				PertemuanKe: p.PertemuanKe, Tanggal: p.Tanggal.Format("2006-01-02"), LevelNama: p.LevelNama, LevelKe: p.PertemuanLevelKe,
			})
		}
	}
	if _, err := s.querier.GetActivePertemuanByKelas(ctx, kelasID); err == nil {
		out.AdaPertemuanBerlangsung = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return out, nil
}

// KoreksiNomorLevel sets the real per-level number of the class's last
// meeting (terakhirLevel) by moving the start of the current level.
func (s *KelasPerubahanService) KoreksiNomorLevel(kelasID, terakhirLevel, userID int64, alasan string) error {
	alasan = strings.TrimSpace(alasan)
	if alasan == "" {
		return fmt.Errorf("alasan koreksi wajib diisi")
	}
	if len([]rune(alasan)) > 500 {
		return fmt.Errorf("alasan koreksi maksimal 500 karakter")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	if _, err := q.GetActivePertemuanByKelas(ctx, kelasID); err == nil {
		return fmt.Errorf("koreksi tidak dapat dilakukan saat ada pertemuan yang sedang berlangsung")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	k, err := s.koreksiLevelKonteks(ctx, q, kelasID)
	if err != nil {
		return err
	}
	sekarang := k.terakhirGlobal - k.levelAwal
	if terakhirLevel < 0 {
		return fmt.Errorf("nomor pertemuan tidak boleh negatif")
	}
	if terakhirLevel == sekarang {
		return fmt.Errorf("nomor pertemuan belum berubah")
	}
	if maks := k.maksTerakhir(); terakhirLevel > maks {
		if k.batasBawah == 0 {
			return fmt.Errorf("maksimal pertemuan ke-%d di level %s, yaitu seluruh pertemuan kelas ini", maks, k.levelNama)
		}
		return fmt.Errorf("maksimal pertemuan ke-%d di level %s: level sebelumnya harus tetap punya minimal satu pertemuan", maks, k.levelNama)
	}
	awalBaru := k.terakhirGlobal - terakhirLevel

	if awalBaru > k.levelAwal {
		// The level started later than recorded: meetings in between go
		// back to the previous level and continue its numbering.
		adaPertemuan := false
		for _, p := range k.pertemuan {
			if p.PertemuanKe > k.levelAwal && p.PertemuanKe <= awalBaru {
				adaPertemuan = true
				break
			}
		}
		if adaPertemuan && strings.TrimSpace(k.levelSebelum) == "" {
			return fmt.Errorf("level sebelum %s tidak diketahui, jadi pertemuan tidak bisa dikembalikan ke level sebelumnya", k.levelNama)
		}
		if err := q.RelabelPertemuanLevel(ctx, queries.RelabelPertemuanLevelParams{
			LevelNama: k.levelSebelum, OffsetLevel: k.offsetSebelum, KelasID: kelasID, Dari: k.levelAwal, Sampai: awalBaru,
		}); err != nil {
			return err
		}
	}
	// Every meeting after the new start belongs to the current level.
	if err := q.RelabelPertemuanLevel(ctx, queries.RelabelPertemuanLevelParams{
		LevelNama: k.levelNama, OffsetLevel: awalBaru, KelasID: kelasID, Dari: awalBaru, Sampai: k.terakhirGlobal,
	}); err != nil {
		return err
	}
	if err := q.SetKelasLevelPertemuanAwal(ctx, queries.SetKelasLevelPertemuanAwalParams{LevelPertemuanAwal: awalBaru, ID: kelasID}); err != nil {
		return err
	}
	if k.perubahanID > 0 {
		if err := q.SetKelasPerubahanPertemuanKe(ctx, queries.SetKelasPerubahanPertemuanKeParams{PertemuanKe: awalBaru, ID: k.perubahanID}); err != nil {
			return err
		}
	}
	if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
		KelasID:     sql.NullInt64{Int64: kelasID, Valid: true},
		Jenis:       PerubahanKoreksiLevel,
		NilaiLama:   fmt.Sprintf("%s ke-%d", k.levelNama, sekarang),
		NilaiBaru:   fmt.Sprintf("%s ke-%d", k.levelNama, terakhirLevel),
		PertemuanKe: k.terakhirGlobal,
		Alasan:      alasan,
		DibuatOleh:  nullUserID(userID),
	}); err != nil {
		return err
	}
	if err := notifyGuruKelas(ctx, q, k.kelas.GuruID, kelasID, userID, "Nomor pertemuan dikoreksi",
		fmt.Sprintf("Nomor pertemuan level %s di kelas %s dikoreksi dari ke-%d menjadi ke-%d. Pertemuan berikutnya adalah ke-%d.",
			k.levelNama, k.kelas.NamaKelas, sekarang, terakhirLevel, terakhirLevel+1)); err != nil {
		return err
	}
	return tx.Commit()
}
