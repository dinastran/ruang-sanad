package services

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

var ErrBuktiTidakAda = errors.New("bukti transfer tidak ditemukan")

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// LeadPanel lists a lead's form submissions and payments.
func (s *RegistrationService) LeadPanel(ctx context.Context, leadID int64) (models.LeadRegistrationPanel, error) {
	out := models.LeadRegistrationPanel{Registrasi: []models.LeadRegistrationRow{}, Payments: []models.PaymentRow{}}
	regs, err := s.querier.ListLeadRegistrationsByLead(ctx, leadID)
	if err != nil {
		return out, err
	}
	for _, r := range regs {
		out.Registrasi = append(out.Registrasi, models.LeadRegistrationRow{
			ID: r.ID, LinkJudul: r.LinkJudul, LinkSlug: r.LinkSlug, LinkMode: r.LinkMode, CreatedAt: fmtTime(r.CreatedAt),
		})
	}
	pays, err := s.querier.ListPaymentsByLead(ctx, leadID)
	if err != nil {
		return out, err
	}
	for _, p := range pays {
		row := models.PaymentRow{
			ID: p.ID, Metode: p.Metode, Amount: p.Amount, Status: p.Status, Channel: p.Channel,
			AdaBukti: p.BuktiPath != "", CatatanSistem: p.CatatanSistem, PerluTinjau: p.PerluTinjau == 1,
			CreatedAt: fmtTime(p.CreatedAt),
		}
		if p.PaidAt.Valid {
			row.PaidAt = fmtTime(p.PaidAt.Time)
		}
		out.Payments = append(out.Payments, row)
	}
	return out, nil
}

// ProofPath returns the stored proof file of a payment that belongs to the
// lead. The file name is reduced to its base so a stored path can never
// point outside the proof directory.
func (s *RegistrationService) ProofPath(ctx context.Context, leadID, paymentID int64) (string, error) {
	p, err := s.querier.GetPayment(ctx, paymentID)
	if err != nil || p.LeadID != leadID || p.BuktiPath == "" {
		return "", ErrBuktiTidakAda
	}
	name := filepath.Base(p.BuktiPath)
	if name == "." || name == ".." || name == string(filepath.Separator) {
		return "", ErrBuktiTidakAda
	}
	return filepath.Join(s.proofDir, name), nil
}

// LinkFormOptions lists the programs and lead sources an admin can pick.
func (s *RegistrationService) LinkFormOptions(ctx context.Context) (programs, sources []models.Option, err error) {
	progs, err := s.querier.ListLeadPrograms(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range progs {
		info := ""
		if p.ProdukID <= 0 {
			info = "belum terhubung ke produk"
		} else {
			info = p.ProdukNama
		}
		programs = append(programs, models.Option{ID: p.ID, Nama: p.Nama, Info: info, Aktif: p.IsAktif})
	}
	srcs, err := s.querier.ListLeadSources(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, src := range srcs {
		sources = append(sources, models.Option{ID: src.ID, Nama: src.Nama, Aktif: src.IsAktif == 1})
	}
	return programs, sources, nil
}
