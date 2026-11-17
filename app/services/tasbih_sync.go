package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	tasbihMaksBatch     = 100
	tasbihJumlahMaks    = 100000
	tasbihToleransiMasa = 5 * time.Minute
	tasbihUmurMaks      = 400 * 24 * time.Hour
)

var tasbihUUIDPola = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Sync menyimpan sesi dari klien secara idempoten. Sesi yang tidak valid
// dilaporkan di Ditolak tanpa menggagalkan sesi lain di batch yang sama.
func (s *TasbihService) Sync(ctx context.Context, userID int64, sesi []models.TasbihSesiInput, now time.Time) (models.TasbihSyncHasil, error) {
	hasil := models.TasbihSyncHasil{Diterima: []string{}, Ditolak: []models.TasbihSesiDitolak{}, BadgeBaru: []models.TasbihBadge{}}
	if len(sesi) > tasbihMaksBatch {
		return hasil, fmt.Errorf("%w: maksimal %d sesi per sync", ErrTasbihValidasi, tasbihMaksBatch)
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return hasil, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	for _, in := range sesi {
		alasan, err := s.simpanSesi(ctx, q, userID, in, now)
		if err != nil {
			return hasil, err
		}
		if alasan != "" {
			hasil.Ditolak = append(hasil.Ditolak, models.TasbihSesiDitolak{ID: in.ID, Alasan: alasan})
			continue
		}
		hasil.Diterima = append(hasil.Diterima, in.ID)
	}
	if err := tx.Commit(); err != nil {
		return hasil, err
	}
	hariIni, err := s.HariIni(ctx, userID, now)
	if err != nil {
		return hasil, err
	}
	hasil.HariIni = hariIni
	if len(hasil.Diterima) > 0 {
		baru, err := s.EvaluasiBadge(ctx, userID, now)
		if err != nil {
			return hasil, err
		}
		hasil.BadgeBaru = baru
	}
	return hasil, nil
}

// simpanSesi mengembalikan alasan penolakan ("" bila diterima).
func (s *TasbihService) simpanSesi(ctx context.Context, q *queries.Querier, userID int64, in models.TasbihSesiInput, now time.Time) (string, error) {
	switch {
	case !tasbihUUIDPola.MatchString(in.ID):
		return "id sesi tidak valid", nil
	case in.Jumlah < 1 || in.Jumlah > tasbihJumlahMaks:
		return "jumlah di luar batas", nil
	case !validTasbihTarget(in.Target):
		return "target di luar batas", nil
	case in.MulaiAt.IsZero() || in.SelesaiAt.Before(in.MulaiAt):
		return "waktu sesi tidak valid", nil
	case in.MulaiAt.After(now.Add(tasbihToleransiMasa)):
		return "waktu sesi di masa depan", nil
	case in.MulaiAt.Before(now.Add(-tasbihUmurMaks)):
		return "sesi terlalu lama", nil
	}
	n, err := q.CountTasbihDzikirAksesUser(ctx, queries.CountTasbihDzikirAksesUserParams{ID: in.DzikirID, UserID: nullUser(userID)})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "dzikir tidak ditemukan", nil
	}

	paketID := sql.NullInt64{}
	runID := sql.NullString{}
	if in.PaketID != nil {
		ada, err := q.CountTasbihPaket(ctx, *in.PaketID)
		if err != nil {
			return "", err
		}
		if ada == 0 {
			return "paket tidak ditemukan", nil
		}
		targets, err := q.ListTasbihPaketItemTarget(ctx, queries.ListTasbihPaketItemTargetParams{PaketID: *in.PaketID, DzikirID: in.DzikirID})
		if err != nil {
			return "", err
		}
		// Target harus sama dengan target item paket; bila tidak, simpan sebagai sesi biasa.
		for _, t := range targets {
			if t == in.Target && in.PaketRunID != nil && tasbihUUIDPola.MatchString(*in.PaketRunID) {
				paketID = sql.NullInt64{Int64: *in.PaketID, Valid: true}
				runID = sql.NullString{String: *in.PaketRunID, Valid: true}
				break
			}
		}
	}

	mulaiWIB := in.MulaiAt.In(wib)
	selesai := int64(0)
	if in.Jumlah >= in.Target {
		selesai = 1
	}
	rows, err := q.InsertTasbihSesi(ctx, queries.InsertTasbihSesiParams{
		ID: in.ID, UserID: userID, DzikirID: in.DzikirID, PaketID: paketID, PaketRunID: runID,
		Jumlah: in.Jumlah, Target: in.Target, SelesaiTarget: selesai,
		MulaiAt: in.MulaiAt.UTC(), SelesaiAt: in.SelesaiAt.UTC(),
		TanggalWib: mulaiWIB.Format("2006-01-02"), JamWib: int64(mulaiWIB.Hour()), CreatedAt: now.UTC(),
	})
	if err != nil {
		return "", err
	}
	if rows == 0 {
		owner, err := q.GetTasbihSesiOwner(ctx, in.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		if owner != userID {
			return "id sesi tidak valid", nil
		}
	}
	return "", nil
}
