package services

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// ReconcileResult counts what one reconciliation pass changed.
type ReconcileResult struct {
	Dibuat, Diaktifkan, Batal, Siap int
}

func commissionAmount(jenis string, nilai, basis int64) int64 {
	if jenis == "persen" {
		return basis * nilai / 10000
	}
	return nilai
}

// purchaseBasis is the first verified payment: the last payment.verified
// amount, falling back to the lead's expected amount.
func purchaseBasis(ctx context.Context, q *queries.Querier, leadID, fallback int64) (int64, error) {
	v, err := q.GetMetaCAPIPurchaseAmount(ctx, leadID)
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		v = fallback
	}
	return v, nil
}

// Reconcile builds the commission ledger from data, idempotently (spec §5.3):
// new registration and monthly SPP commissions, re-activation after a
// re-verification, cancellations during the hold, and maturing to siap.
func (s *AffiliateService) Reconcile(ctx context.Context) (ReconcileResult, error) {
	var res ReconcileResult
	err := s.leads.withTx(ctx, func(q *queries.Querier) error {
		now := s.nowUTC()
		st, err := q.GetAffiliateSettings(ctx)
		if err != nil {
			return err
		}

		// 1. Cancel pending registration commissions whose verification was undone.
		n, err := q.CancelCommissionsVerificationUndone(ctx, now)
		if err != nil {
			return err
		}
		res.Batal += int(n)

		// 2. Re-activate those whose lead is verified again (same row).
		re, err := q.ListReactivatableCommissions(ctx)
		if err != nil {
			return err
		}
		for _, r := range re {
			c, err := q.GetAffiliateCommission(ctx, r.ID)
			if err != nil {
				return err
			}
			basis, err := purchaseBasis(ctx, q, r.LeadID, r.PaymentAmount)
			if err != nil {
				return err
			}
			if err := q.ReactivateCommission(ctx, queries.ReactivateCommissionParams{
				Basis: basis, Jumlah: commissionAmount(c.Jenis, c.Nilai, basis), CreatedAt: now, UpdatedAt: now, ID: r.ID,
			}); err != nil {
				return err
			}
			res.Diaktifkan++
		}

		// 3. New registration commissions.
		cands, err := q.ListCommissionCandidatesPendaftaran(ctx)
		if err != nil {
			return err
		}
		for _, c := range cands {
			rule, err := ruleLookup(ctx, q, c.AffiliateID, c.ProgramID)
			if err != nil {
				return err
			}
			if rule == nil {
				continue
			}
			basis, err := purchaseBasis(ctx, q, c.LeadID, c.PaymentAmount)
			if err != nil {
				return err
			}
			n, err := q.InsertAffiliateCommission(ctx, queries.InsertAffiliateCommissionParams{
				AffiliateID: c.AffiliateID, LeadID: c.LeadID, SantriID: c.SantriID, Sumber: "pendaftaran", SumberID: c.LeadID,
				BulanKe: 1, Basis: basis, Jenis: rule.Jenis, Nilai: rule.Nilai, Pola: rule.Pola, MaksBulan: rule.MaksBulan,
				Jumlah: commissionAmount(rule.Jenis, rule.Nilai, basis), Status: "pending", CreatedAt: now, UpdatedAt: now,
			})
			if err != nil {
				return err
			}
			res.Dibuat += int(n)
		}

		// 4. Monthly SPP commissions (rule snapshot from the registration row).
		spp, err := q.ListCommissionCandidatesSPP(ctx)
		if err != nil {
			return err
		}
		for _, c := range spp {
			urutan, err := q.CountTagihanSinceConversion(ctx, queries.CountTagihanSinceConversionParams{
				SantriID: c.SantriID, ConvertedAt: c.ConvertedAt.Time, CreatedAt: c.TagihanCreated, ID: c.TagihanID,
			})
			if err != nil {
				return err
			}
			bulanKe := urutan + 1 // bulan 1 = pembayaran pendaftaran
			if c.MaksBulan.Valid && bulanKe > c.MaksBulan.Int64 {
				continue
			}
			n, err := q.InsertAffiliateCommission(ctx, queries.InsertAffiliateCommissionParams{
				AffiliateID: c.AffiliateID, LeadID: c.LeadID, SantriID: sql.NullInt64{Int64: c.SantriID, Valid: true},
				Sumber: "spp", SumberID: c.TagihanID, BulanKe: bulanKe, Basis: c.Nominal, Jenis: c.Jenis, Nilai: c.Nilai,
				Pola: "bulanan", MaksBulan: c.MaksBulan, Jumlah: commissionAmount(c.Jenis, c.Nilai, c.Nominal),
				Status: "pending", CreatedAt: now, UpdatedAt: now,
			})
			if err != nil {
				return err
			}
			res.Dibuat += int(n)
		}

		// 5. Santri stopped during the hold → cancel; then mature the rest.
		n, err = q.CancelCommissionsSantriStopped(ctx, now)
		if err != nil {
			return err
		}
		res.Batal += int(n)
		n, err = q.MatureCommissions(ctx, queries.MatureCommissionsParams{
			Now: sql.NullTime{Time: now, Valid: true}, Cutoff: now.Add(-time.Duration(st.MasaTahanHari) * 24 * time.Hour),
		})
		if err != nil {
			return err
		}
		res.Siap += int(n)
		return nil
	})
	return res, err
}

// StartReconcileWorker runs Reconcile every interval until ctx ends.
func (s *AffiliateService) StartReconcileWorker(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			if _, err := s.EnsureGuruAffiliates(ctx); err != nil {
				slog.Error("affiliate: sync guru gagal", "error", err)
			}
			if r, err := s.Reconcile(ctx); err != nil {
				slog.Error("affiliate: rekonsiliasi komisi gagal", "error", err)
			} else if r != (ReconcileResult{}) {
				slog.Info("affiliate: rekonsiliasi komisi", "dibuat", r.Dibuat, "aktif_lagi", r.Diaktifkan, "batal", r.Batal, "siap", r.Siap)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}
