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

// BatasLupaDitutup is how long a meeting may stay berlangsung before it is
// treated as forgotten: flagged in the UI and the guru is reminded once.
const BatasLupaDitutup = 3 * time.Hour

const PerubahanBatalPertemuan = "batal_pertemuan"

// ListBelumDitutup returns meetings still berlangsung. guruID nil means all
// classes; otherwise classes taught by the guru or meetings they substitute.
func (s *PertemuanService) ListBelumDitutup(guruID *int64, now time.Time) ([]models.PertemuanBelumDitutup, error) {
	rows, err := s.querier.ListPertemuanBerlangsung(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.PertemuanBelumDitutup, 0, len(rows))
	for _, r := range rows {
		if guruID != nil {
			utama := r.GuruID.Valid && r.GuruID.Int64 == *guruID
			badal := r.GuruPenggantiID.Valid && r.GuruPenggantiID.Int64 == *guruID
			if !utama && !badal {
				continue
			}
		}
		durasi := now.Sub(r.CreatedAt)
		out = append(out, models.PertemuanBelumDitutup{
			ID:                r.ID,
			KelasID:           r.KelasID,
			KelasNama:         r.NamaKelas,
			GuruNama:          r.GuruNama,
			GuruPenggantiNama: r.GuruPenggantiNama,
			PertemuanKe:       r.PertemuanLevelKe,
			Tanggal:           r.Tanggal.Format("2006-01-02"),
			JamMulai:          r.JamMulai,
			DurasiJam:         int64(durasi.Hours()),
			Lupa:              durasi >= BatasLupaDitutup,
		})
	}
	return out, nil
}

// BatalkanPertemuan removes a meeting that was started by mistake. Only a
// meeting still berlangsung can be cancelled: it has no attendance or billing
// yet. A schedule that started it becomes startable again, and the
// cancellation is kept in the class change history.
func (s *PertemuanService) BatalkanPertemuan(pertemuanID, kelasID, userID int64, alasan string) error {
	alasan = strings.TrimSpace(alasan)
	if alasan == "" {
		return fmt.Errorf("alasan pembatalan wajib diisi")
	}
	if len([]rune(alasan)) > 500 {
		return fmt.Errorf("alasan pembatalan maksimal 500 karakter")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	p, err := q.GetPertemuanByID(ctx, pertemuanID)
	if err != nil || p.KelasID != kelasID {
		return fmt.Errorf("pertemuan tidak ditemukan")
	}
	if p.Status != "berlangsung" {
		return fmt.Errorf("hanya pertemuan yang sedang berlangsung yang bisa dibatalkan")
	}
	ref := sql.NullInt64{Int64: pertemuanID, Valid: true}
	jadwalID, jadwalErr := q.GetJadwalIDByPertemuan(ctx, ref)
	if jadwalErr != nil && !errors.Is(jadwalErr, sql.ErrNoRows) {
		return jadwalErr
	}
	if err := q.ResetJadwalPertemuanDariPertemuan(ctx, ref); err != nil {
		return err
	}
	deleted, err := q.DeletePertemuanBerlangsung(ctx, queries.DeletePertemuanBerlangsungParams{ID: pertemuanID, KelasID: kelasID})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("pertemuan gagal dibatalkan karena status telah berubah")
	}
	nilaiLama := fmt.Sprintf("Pertemuan ke-%d · %s %s", p.PertemuanLevelKe, p.Tanggal.Format("2006-01-02"), p.JamMulai)
	if err := q.CreateKelasPerubahan(ctx, queries.CreateKelasPerubahanParams{
		KelasID:     sql.NullInt64{Int64: kelasID, Valid: true},
		Jenis:       PerubahanBatalPertemuan,
		NilaiLama:   nilaiLama,
		NilaiBaru:   "Dibatalkan",
		PertemuanKe: p.PertemuanKe,
		Alasan:      alasan,
		DibuatOleh:  nullUserID(userID),
	}); err != nil {
		return err
	}
	if jadwalErr == nil {
		if err := q.CreateScheduleActivityLog(ctx, queries.CreateScheduleActivityLogParams{
			JadwalPertemuanID: jadwalID,
			Action:            "pertemuan_dibatalkan",
			Details:           alasan,
			CreatedBy:         nullUserID(userID),
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// KirimPengingatLupaDitutup notifies the guru (and substitute guru) once for
// each meeting berlangsung longer than BatasLupaDitutup. It returns how many
// meetings were reminded.
func (s *PertemuanService) KirimPengingatLupaDitutup(now time.Time) (int, error) {
	ctx := context.Background()
	rows, err := s.querier.ListPertemuanLupaBelumDiingatkan(ctx)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, r := range rows {
		if now.Sub(r.CreatedAt) < BatasLupaDitutup {
			continue
		}
		pesan := fmt.Sprintf("Pertemuan kelas %s yang dimulai %s pukul %s belum ditutup. Selesaikan dengan mengisi absensi, atau batalkan bila tidak jadi berlangsung. Kelas ini tidak bisa memulai pertemuan baru sebelum ditutup.",
			r.NamaKelas, r.Tanggal.Format("2006-01-02"), r.JamMulai)
		for _, uid := range uniqueIDs([]int64{r.GuruUserID.Int64, r.GuruPenggantiUserID.Int64}) {
			if _, err := s.querier.CreateNotification(ctx, queries.CreateNotificationParams{
				UserID:        uid,
				Type:          "pertemuan_lupa_ditutup",
				Title:         "Pertemuan belum ditutup",
				Message:       pesan,
				ActionUrl:     "/app/guru/jadwal-pertemuan",
				ReferenceType: "pertemuan",
				ReferenceID:   sql.NullInt64{Int64: r.ID, Valid: true},
			}); err != nil {
				return sent, err
			}
		}
		if err := s.querier.MarkPengingatLupaPertemuan(ctx, queries.MarkPengingatLupaPertemuanParams{
			PengingatLupaAt: sql.NullTime{Time: now, Valid: true}, ID: r.ID,
		}); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
