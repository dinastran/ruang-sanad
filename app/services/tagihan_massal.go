package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrBatalMassalTanggalWajib  = errors.New("isi rentang tanggal tagih (dari dan sampai) lalu terapkan filter sebelum membatalkan massal")
	ErrBatalMassalTanggalTidak  = errors.New("rentang tanggal tagih tidak valid")
	ErrBatalMassalAlasanWajib   = errors.New("alasan pembatalan wajib diisi")
	ErrBatalMassalKosong        = errors.New("tidak ada tagihan belum bayar yang sesuai filter")
	ErrBatalMassalBerubah       = errors.New("jumlah tagihan sudah berubah sejak preview, silakan buka preview ulang")
	ErrBatalBatchTidakDitemukan = errors.New("riwayat pembatalan massal tidak ditemukan")
	ErrBatalBatchSudahPulih     = errors.New("batch ini sudah pernah dipulihkan")
)

// TagihanFilterFromRequest converts the page's string filter values into the
// filter used by List, so bulk actions match exactly what the table shows.
func TagihanFilterFromRequest(r models.TagihanFilterRequest) models.TagihanFilter {
	parse := func(v string) int64 { n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64); return n }
	return models.TagihanFilter{
		Status: r.Status, KelasID: parse(r.KelasID), AngkatanKelas: r.AngkatanKelas, GuruID: parse(r.GuruID),
		Frekuensi: r.Frekuensi, Level: r.Level, Gender: r.Gender, BulanKe: parse(r.BulanKe),
		TanggalDari: strings.TrimSpace(r.TanggalDari), TanggalSampai: strings.TrimSpace(r.TanggalSampai), Search: r.Search,
	}
}

func validateBatalMassalFilter(f models.TagihanFilter) error {
	if f.TanggalDari == "" || f.TanggalSampai == "" {
		return ErrBatalMassalTanggalWajib
	}
	dari, err1 := time.Parse("2006-01-02", f.TanggalDari)
	sampai, err2 := time.Parse("2006-01-02", f.TanggalSampai)
	if err1 != nil || err2 != nil || sampai.Before(dari) {
		return ErrBatalMassalTanggalTidak
	}
	return nil
}

func batalMassalEligible(t models.TagihanResponse) bool {
	// List reports overdue belum_bayar rows as "terlambat".
	return t.Status == "belum_bayar" || t.Status == "terlambat"
}

func (s *TagihanService) PreviewBatalMassal(filter models.TagihanFilter) (*models.BatalMassalPreview, error) {
	if err := validateBatalMassalFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.List(filter)
	if err != nil {
		return nil, err
	}
	out := &models.BatalMassalPreview{}
	for _, t := range rows {
		switch {
		case batalMassalEligible(t):
			out.Jumlah++
			out.TotalNominal += t.Nominal
		case t.Status == "lunas":
			out.DilewatiLunas++
		case t.Status == "batal":
			out.DilewatiBatal++
		}
	}
	return out, nil
}

// BatalMassal cancels every belum_bayar tagihan matching filter in one
// transaction and records the previous state so the batch can be restored.
// konfirmasi must equal the eligible count, guarding against data that changed
// after the preview was shown.
func (s *TagihanService) BatalMassal(filter models.TagihanFilter, alasan string, userID, konfirmasi int64) (*models.BatalMassalResult, error) {
	if err := validateBatalMassalFilter(filter); err != nil {
		return nil, err
	}
	alasan = strings.TrimSpace(alasan)
	if alasan == "" {
		return nil, ErrBatalMassalAlasanWajib
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	rows, err := s.listWithQuerier(ctx, q, filter)
	if err != nil {
		return nil, err
	}
	eligible := make([]models.TagihanResponse, 0, len(rows))
	for _, t := range rows {
		if batalMassalEligible(t) {
			eligible = append(eligible, t)
		}
	}
	if len(eligible) == 0 {
		return nil, ErrBatalMassalKosong
	}
	if int64(len(eligible)) != konfirmasi {
		return nil, ErrBatalMassalBerubah
	}

	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}
	batchID, err := q.CreateTagihanBatalBatch(ctx, queries.CreateTagihanBatalBatchParams{
		Alasan: alasan, FilterJson: string(filterJSON), DibuatOleh: sql.NullInt64{Int64: userID, Valid: userID > 0},
	})
	if err != nil {
		return nil, err
	}
	result := &models.BatalMassalResult{BatchID: batchID}
	for _, t := range eligible {
		n, err := q.BatalkanTagihanBelumBayar(ctx, queries.BatalkanTagihanBelumBayarParams{Catatan: alasan, ID: t.ID})
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		if err := q.CreateTagihanBatalBatchItem(ctx, queries.CreateTagihanBatalBatchItemParams{
			BatchID: batchID, TagihanID: t.ID, StatusLama: "belum_bayar", CatatanLama: t.Catatan,
		}); err != nil {
			return nil, err
		}
		result.Jumlah++
		result.TotalNominal += t.Nominal
	}
	if err := q.FinalizeTagihanBatalBatch(ctx, queries.FinalizeTagihanBatalBatchParams{Jumlah: result.Jumlah, TotalNominal: result.TotalNominal, ID: batchID}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// PulihkanBatalMassal restores a batch to each tagihan's previous status and
// catatan. Tagihan that are no longer "batal" are left untouched. It returns the
// number of tagihan restored.
func (s *TagihanService) PulihkanBatalMassal(batchID, userID int64) (int64, error) {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	batch, err := q.GetTagihanBatalBatch(ctx, batchID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrBatalBatchTidakDitemukan
	}
	if err != nil {
		return 0, err
	}
	if batch.DipulihkanAt.Valid {
		return 0, ErrBatalBatchSudahPulih
	}
	items, err := q.ListTagihanBatalBatchItems(ctx, batchID)
	if err != nil {
		return 0, err
	}
	var restored int64
	for _, item := range items {
		n, err := q.PulihkanTagihanBatal(ctx, queries.PulihkanTagihanBatalParams{Status: item.StatusLama, Catatan: item.CatatanLama, ID: item.TagihanID})
		if err != nil {
			return 0, err
		}
		restored += n
	}
	n, err := q.MarkTagihanBatalBatchDipulihkan(ctx, queries.MarkTagihanBatalBatchDipulihkanParams{
		DipulihkanOleh: sql.NullInt64{Int64: userID, Valid: userID > 0}, JumlahDipulihkan: restored, ID: batchID,
	})
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, ErrBatalBatchSudahPulih
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return restored, nil
}

func (s *TagihanService) ListBatalMassal(limit int64) ([]models.TagihanBatalBatchResponse, error) {
	rows, err := s.querier.ListTagihanBatalBatch(context.Background(), limit)
	if err != nil {
		return nil, err
	}
	out := make([]models.TagihanBatalBatchResponse, 0, len(rows))
	for _, r := range rows {
		item := models.TagihanBatalBatchResponse{
			ID: r.ID, Alasan: r.Alasan, FilterRingkas: ringkasFilterBatalMassal(r.FilterJson),
			Jumlah: r.Jumlah, TotalNominal: r.TotalNominal, DibuatOlehNama: r.DibuatOlehNama,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04"), DipulihkanOlehNama: r.DipulihkanOlehNama,
			JumlahDipulihkan: r.JumlahDipulihkan,
		}
		if r.DipulihkanAt.Valid {
			item.DipulihkanAt = r.DipulihkanAt.Time.Format("2006-01-02 15:04")
		}
		out = append(out, item)
	}
	return out, nil
}

func ringkasFilterBatalMassal(raw string) string {
	var f models.TagihanFilter
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return ""
	}
	parts := []string{fmt.Sprintf("Tagih %s s/d %s", f.TanggalDari, f.TanggalSampai)}
	add := func(label, v string) {
		if v != "" {
			parts = append(parts, label+" "+v)
		}
	}
	addID := func(format string, v int64) {
		if v > 0 {
			parts = append(parts, fmt.Sprintf(format, v))
		}
	}
	add("status", strings.ReplaceAll(f.Status, "_", " "))
	addID("kelas #%d", f.KelasID)
	addID("guru #%d", f.GuruID)
	add("angkatan", f.AngkatanKelas)
	add("frekuensi", f.Frekuensi)
	add("level", f.Level)
	add("gender", f.Gender)
	addID("bulan ke-%d", f.BulanKe)
	if f.Search != "" {
		parts = append(parts, fmt.Sprintf("cari %q", f.Search))
	}
	return strings.Join(parts, " · ")
}
