package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrTasbihValidasi       = errors.New("data tasbih tidak valid")
	ErrTasbihTidakDitemukan = errors.New("dzikir tidak ditemukan")
)

const (
	tasbihTargetMin        = 1
	tasbihTargetMax        = 100000
	tasbihTargetHarianAwal = 100
	tasbihMaksDzikirCustom = 50
	tasbihMaksLatin        = 300
	tasbihMaksArab         = 500
	tasbihMaksArti         = 500
)

// TasbihService: tasbih digital pribadi (katalog dzikir, sesi, statistik, badge).
type TasbihService struct {
	querier *queries.Querier
}

func NewTasbihService(querier *queries.Querier) *TasbihService {
	return &TasbihService{querier: querier}
}

func validTasbihTarget(v int64) bool { return v >= tasbihTargetMin && v <= tasbihTargetMax }

func nullUser(userID int64) sql.NullInt64 { return sql.NullInt64{Int64: userID, Valid: true} }

// Katalog mengembalikan dzikir bawaan + custom aktif milik user, dan paket wirid.
func (s *TasbihService) Katalog(ctx context.Context, userID int64) ([]models.TasbihDzikir, []models.TasbihPaket, error) {
	rows, err := s.querier.ListTasbihDzikirForUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	dzikir := make([]models.TasbihDzikir, 0, len(rows))
	for _, r := range rows {
		dzikir = append(dzikir, models.TasbihDzikir{
			ID: r.ID, Custom: r.UserID.Valid, Arab: r.TeksArab, Latin: r.Latin, Arti: r.Arti,
			Rujukan: r.Rujukan, Target: r.TargetEfektif, TargetDefault: r.TargetDefault,
		})
	}
	paket, err := s.daftarPaket(ctx)
	if err != nil {
		return nil, nil, err
	}
	return dzikir, paket, nil
}

func (s *TasbihService) daftarPaket(ctx context.Context) ([]models.TasbihPaket, error) {
	rows, err := s.querier.ListTasbihPaket(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.querier.ListTasbihPaketItem(ctx)
	if err != nil {
		return nil, err
	}
	byPaket := map[int64][]models.TasbihPaketItem{}
	for _, it := range items {
		byPaket[it.PaketID] = append(byPaket[it.PaketID], models.TasbihPaketItem{DzikirID: it.DzikirID, Target: it.Target})
	}
	out := make([]models.TasbihPaket, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.TasbihPaket{ID: r.ID, Slug: r.Slug, Nama: r.Nama, Deskripsi: r.Deskripsi, Rujukan: r.Rujukan, Items: byPaket[r.ID]})
	}
	return out, nil
}

func normalisasiDzikirInput(in models.TasbihDzikirInput) (models.TasbihDzikirInput, error) {
	in.Latin = strings.TrimSpace(in.Latin)
	in.Arab = strings.TrimSpace(in.Arab)
	in.Arti = strings.TrimSpace(in.Arti)
	switch {
	case in.Latin == "":
		return in, fmt.Errorf("%w: teks latin wajib diisi", ErrTasbihValidasi)
	case utf8.RuneCountInString(in.Latin) > tasbihMaksLatin:
		return in, fmt.Errorf("%w: teks latin maksimal %d karakter", ErrTasbihValidasi, tasbihMaksLatin)
	case utf8.RuneCountInString(in.Arab) > tasbihMaksArab:
		return in, fmt.Errorf("%w: teks Arab maksimal %d karakter", ErrTasbihValidasi, tasbihMaksArab)
	case utf8.RuneCountInString(in.Arti) > tasbihMaksArti:
		return in, fmt.Errorf("%w: arti maksimal %d karakter", ErrTasbihValidasi, tasbihMaksArti)
	case !validTasbihTarget(in.Target):
		return in, fmt.Errorf("%w: target harus 1 sampai %d", ErrTasbihValidasi, tasbihTargetMax)
	}
	return in, nil
}

func (s *TasbihService) BuatDzikir(ctx context.Context, userID int64, in models.TasbihDzikirInput) (int64, error) {
	in, err := normalisasiDzikirInput(in)
	if err != nil {
		return 0, err
	}
	n, err := s.querier.CountTasbihDzikirCustomAktif(ctx, nullUser(userID))
	if err != nil {
		return 0, err
	}
	if n >= tasbihMaksDzikirCustom {
		return 0, fmt.Errorf("%w: maksimal %d dzikir custom", ErrTasbihValidasi, tasbihMaksDzikirCustom)
	}
	now := time.Now()
	return s.querier.CreateTasbihDzikirCustom(ctx, queries.CreateTasbihDzikirCustomParams{
		UserID: nullUser(userID), TeksArab: in.Arab, Latin: in.Latin, Arti: in.Arti,
		TargetDefault: in.Target, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *TasbihService) UbahDzikir(ctx context.Context, userID, dzikirID int64, in models.TasbihDzikirInput) error {
	in, err := normalisasiDzikirInput(in)
	if err != nil {
		return err
	}
	n, err := s.querier.UpdateTasbihDzikirCustom(ctx, queries.UpdateTasbihDzikirCustomParams{
		TeksArab: in.Arab, Latin: in.Latin, Arti: in.Arti, TargetDefault: in.Target,
		UpdatedAt: time.Now(), ID: dzikirID, UserID: nullUser(userID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTasbihTidakDitemukan
	}
	return nil
}

func (s *TasbihService) ArsipkanDzikir(ctx context.Context, userID, dzikirID int64) error {
	now := time.Now()
	n, err := s.querier.ArchiveTasbihDzikirCustom(ctx, queries.ArchiveTasbihDzikirCustomParams{
		DiarsipkanAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: now, ID: dzikirID, UserID: nullUser(userID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTasbihTidakDitemukan
	}
	return nil
}

// SetTargetDzikir menyimpan override target untuk dzikir bawaan atau custom milik user.
func (s *TasbihService) SetTargetDzikir(ctx context.Context, userID, dzikirID, target int64) error {
	if !validTasbihTarget(target) {
		return fmt.Errorf("%w: target harus 1 sampai %d", ErrTasbihValidasi, tasbihTargetMax)
	}
	ok, err := s.dzikirBolehDipakai(ctx, userID, dzikirID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrTasbihTidakDitemukan
	}
	return s.querier.UpsertTasbihTargetUser(ctx, queries.UpsertTasbihTargetUserParams{
		UserID: userID, DzikirID: dzikirID, Target: target, UpdatedAt: time.Now(),
	})
}

func (s *TasbihService) dzikirBolehDipakai(ctx context.Context, userID, dzikirID int64) (bool, error) {
	n, err := s.querier.CountTasbihDzikirAksesUser(ctx, queries.CountTasbihDzikirAksesUserParams{ID: dzikirID, UserID: nullUser(userID)})
	return n > 0, err
}

func (s *TasbihService) TargetHarian(ctx context.Context, userID int64) (int64, error) {
	v, err := s.querier.GetTasbihTargetHarian(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return tasbihTargetHarianAwal, nil
	}
	return v, err
}

func (s *TasbihService) SetTargetHarian(ctx context.Context, userID, target int64) error {
	if !validTasbihTarget(target) {
		return fmt.Errorf("%w: target harian harus 1 sampai %d", ErrTasbihValidasi, tasbihTargetMax)
	}
	return s.querier.UpsertTasbihTargetHarian(ctx, queries.UpsertTasbihTargetHarianParams{UserID: userID, TargetHarian: target, UpdatedAt: time.Now()})
}
