package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// MarkTransferred records manual transfers to external affiliates. Only
// ready (siap) commissions of external affiliates change; returns how many.
func (s *AffiliateService) MarkTransferred(ctx context.Context, ids []int64, actorID int64, catatan string) (int, error) {
	catatan = truncateRunes(strings.TrimSpace(catatan), 200)
	done := 0
	err := s.leads.withTx(ctx, func(q *queries.Querier) error {
		now := s.nowUTC()
		for _, id := range ids {
			n, err := q.MarkCommissionTransferred(ctx, queries.MarkCommissionTransferredParams{
				Catatan: catatan, DibayarAt: sql.NullTime{Time: now, Valid: true},
				DibayarOleh: sql.NullInt64{Int64: actorID, Valid: actorID > 0}, UpdatedAt: now, ID: id,
			})
			if err != nil {
				return err
			}
			done += int(n)
		}
		return nil
	})
	return done, err
}

// CancelCommission lets an admin void a commission that is not paid yet.
func (s *AffiliateService) CancelCommission(ctx context.Context, id, actorID int64, alasan string) error {
	alasan = truncateRunes(strings.TrimSpace(alasan), 200)
	if alasan == "" {
		return errors.New("alasan pembatalan wajib diisi")
	}
	n, err := s.querier.CancelCommission(ctx, queries.CancelCommissionParams{AlasanBatal: alasan + " (admin)", UpdatedAt: s.nowUTC(), ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("komisi tidak ditemukan atau sudah dibayar")
	}
	return nil
}

// ujrohKomisiCutoff is the end of an ujroh month (exclusive), UTC.
func ujrohKomisiCutoff(bulan string) (time.Time, error) {
	start, err := time.ParseInLocation("2006-01", bulan, wib)
	if err != nil {
		return time.Time{}, err
	}
	return start.AddDate(0, 1, 0).UTC(), nil
}
