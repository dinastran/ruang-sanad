package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrTeachingLevelInvalid  = errors.New("persetujuan level guru tidak valid")
	ErrTeachingLevelNotFound = errors.New("guru tidak ditemukan")
)

type GuruTeachingLevelsService struct {
	q *queries.Querier
}

func NewGuruTeachingLevelsService(q *queries.Querier) *GuruTeachingLevelsService {
	return &GuruTeachingLevelsService{q: q}
}

type GuruTeachingLevelData struct {
	Available []models.LevelResponse         `json:"available"`
	Approved  []queries.GuruTeachingLevelRow `json:"approved"`
}

func (s *GuruTeachingLevelsService) Get(ctx context.Context, guruID int64) (GuruTeachingLevelData, error) {
	out := GuruTeachingLevelData{Available: make([]models.LevelResponse, 0), Approved: make([]queries.GuruTeachingLevelRow, 0)}
	if guruID <= 0 {
		return out, ErrTeachingLevelInvalid
	}
	if _, err := s.q.GuruGetByID(ctx, guruID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, ErrTeachingLevelNotFound
		}
		return out, err
	}
	rows, err := s.q.ListLevel(ctx)
	if err != nil {
		return out, err
	}
	for _, row := range rows {
		out.Available = append(out.Available, models.LevelResponse{
			ID: row.ID, Kode: row.Kode, Nama: row.Nama, Urutan: row.Urutan,
		})
	}
	out.Approved, err = s.q.ListGuruTeachingLevels(ctx, guruID)
	return out, err
}

func normalizeApprovedLevels(selected []string, master map[string]bool) ([]string, error) {
	if len(selected) > 100 {
		return nil, fmt.Errorf("%w: terlalu banyak level", ErrTeachingLevelInvalid)
	}
	out := make([]string, 0, len(selected))
	seen := make(map[string]bool, len(selected))
	for _, raw := range selected {
		code := strings.TrimSpace(raw)
		if !master[code] || code == "" {
			return nil, fmt.Errorf("%w: kode level tidak ada di master", ErrTeachingLevelInvalid)
		}
		if seen[code] {
			return nil, fmt.Errorf("%w: level berulang", ErrTeachingLevelInvalid)
		}
		seen[code] = true
		out = append(out, code)
	}
	sort.Strings(out)
	return out, nil
}

// Replace is a complete coordinator-approved set. Empty list revokes all
// permissions; the separate existing proficiency matrix is unchanged.
func (s *GuruTeachingLevelsService) Replace(ctx context.Context, guruID, actorID int64, selected []string) error {
	if guruID <= 0 || actorID <= 0 {
		return ErrTeachingLevelInvalid
	}
	tx, err := s.q.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	if _, err := q.GuruGetByID(ctx, guruID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTeachingLevelNotFound
		}
		return err
	}
	masterRows, err := q.ListLevel(ctx)
	if err != nil {
		return err
	}
	master := make(map[string]bool, len(masterRows))
	for _, row := range masterRows {
		master[row.Kode] = true
	}
	codes, err := normalizeApprovedLevels(selected, master)
	if err != nil {
		return err
	}
	// Preserve the original approver/timestamp for an unchanged set.
	existing, err := q.ListGuruTeachingLevels(ctx, guruID)
	if err != nil {
		return err
	}
	unchanged := len(existing) == len(codes)
	if unchanged {
		for i, row := range existing {
			if row.LevelKode != codes[i] {
				unchanged = false
				break
			}
		}
	}
	if unchanged {
		return nil
	}
	if err := q.ReplaceTeachingLevelsTx(ctx, guruID, actorID, codes); err != nil {
		return err
	}
	return tx.Commit()
}
