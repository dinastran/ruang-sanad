package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrGuruAvailabilityInvalid  = errors.New("jadwal available guru tidak valid")
	ErrGuruAvailabilityConflict = errors.New("jadwal available guru bertabrakan")
	ErrGuruAvailabilityNotFound = errors.New("jadwal available guru tidak ditemukan")
)

type GuruAvailabilityInput struct {
	Hari          int64  `json:"hari"`
	JamMulai      string `json:"jam_mulai"`
	JamSelesai    string `json:"jam_selesai"`
	BerlakuMulai  string `json:"berlaku_mulai"`
	BerlakuSampai string `json:"berlaku_sampai"`
}

type GuruAvailabilityService struct {
	querier *queries.Querier
}

func NewGuruAvailabilityService(q *queries.Querier) *GuruAvailabilityService {
	return &GuruAvailabilityService{querier: q}
}

func validateGuruAvailabilityInput(req *GuruAvailabilityInput) error {
	req.JamMulai = strings.TrimSpace(req.JamMulai)
	req.JamSelesai = strings.TrimSpace(req.JamSelesai)
	req.BerlakuMulai = strings.TrimSpace(req.BerlakuMulai)
	req.BerlakuSampai = strings.TrimSpace(req.BerlakuSampai)
	if req.Hari < 1 || req.Hari > 7 {
		return fmt.Errorf("%w: pilih hari Senin sampai Ahad", ErrGuruAvailabilityInvalid)
	}
	start, ok := planningMinutes(req.JamMulai)
	if !ok {
		return fmt.Errorf("%w: jam mulai harus HH:MM", ErrGuruAvailabilityInvalid)
	}
	end, ok := planningMinutes(req.JamSelesai)
	if !ok || end <= start || end-start < planningLessonMinutes {
		return fmt.Errorf("%w: rentang waktu minimal 90 menit dalam hari yang sama", ErrGuruAvailabilityInvalid)
	}
	from, err := time.Parse(time.DateOnly, req.BerlakuMulai)
	if err != nil || from.Format(time.DateOnly) != req.BerlakuMulai {
		return fmt.Errorf("%w: tanggal mulai harus YYYY-MM-DD", ErrGuruAvailabilityInvalid)
	}
	if req.BerlakuSampai != "" {
		until, err := time.Parse(time.DateOnly, req.BerlakuSampai)
		if err != nil || until.Format(time.DateOnly) != req.BerlakuSampai || until.Before(from) {
			return fmt.Errorf("%w: tanggal akhir tidak valid", ErrGuruAvailabilityInvalid)
		}
	}
	return nil
}

func (s *GuruAvailabilityService) List(ctx context.Context, guruID int64) ([]queries.GuruAvailabilityRow, error) {
	if guruID <= 0 {
		return nil, ErrGuruAvailabilityNotFound
	}
	if _, err := s.querier.GuruGetByID(ctx, guruID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGuruAvailabilityNotFound
		}
		return nil, err
	}
	return s.querier.ListGuruAvailability(ctx, guruID)
}

func (s *GuruAvailabilityService) save(ctx context.Context, guruID, slotID, actorID int64, req GuruAvailabilityInput) error {
	if guruID <= 0 || actorID <= 0 || slotID < 0 {
		return ErrGuruAvailabilityInvalid
	}
	if err := validateGuruAvailabilityInput(&req); err != nil {
		return err
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	if _, err := q.GuruGetByID(ctx, guruID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGuruAvailabilityNotFound
		}
		return err
	}
	if slotID > 0 {
		if _, err := q.GetGuruAvailability(ctx, guruID, slotID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrGuruAvailabilityNotFound
			}
			return err
		}
	}
	overlaps, err := q.CountGuruAvailabilityOverlap(ctx, guruID, slotID,
		req.Hari, req.JamMulai, req.JamSelesai, req.BerlakuMulai, req.BerlakuSampai)
	if err != nil {
		return err
	}
	if overlaps > 0 {
		return fmt.Errorf("%w: hari, jam, dan periode berlaku sudah beririsan dengan slot lain", ErrGuruAvailabilityConflict)
	}
	row := queries.GuruAvailabilityRow{
		ID: slotID, GuruID: guruID, Hari: req.Hari,
		JamMulai: req.JamMulai, JamSelesai: req.JamSelesai,
		BerlakuMulai: req.BerlakuMulai, BerlakuSampai: req.BerlakuSampai,
	}
	if slotID == 0 {
		_, err = q.CreateGuruAvailability(ctx, row, actorID)
	} else {
		var affected int64
		affected, err = q.UpdateGuruAvailability(ctx, row, actorID)
		if err == nil && affected == 0 {
			return ErrGuruAvailabilityNotFound
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *GuruAvailabilityService) Create(ctx context.Context, guruID, actorID int64, req GuruAvailabilityInput) error {
	return s.save(ctx, guruID, 0, actorID, req)
}

func (s *GuruAvailabilityService) Update(ctx context.Context, guruID, slotID, actorID int64, req GuruAvailabilityInput) error {
	if slotID <= 0 {
		return ErrGuruAvailabilityInvalid
	}
	return s.save(ctx, guruID, slotID, actorID, req)
}

func (s *GuruAvailabilityService) Delete(ctx context.Context, guruID, slotID, actorID int64) error {
	if guruID <= 0 || slotID <= 0 || actorID <= 0 {
		return ErrGuruAvailabilityInvalid
	}
	affected, err := s.querier.DeleteGuruAvailability(ctx, guruID, slotID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrGuruAvailabilityNotFound
	}
	return nil
}
