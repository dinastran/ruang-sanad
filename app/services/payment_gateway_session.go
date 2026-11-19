package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

var ErrGatewayNonaktif = errors.New("pembayaran online belum tersedia")

// WithGateway enables online payment. appURL is the public HTTPS base URL
// used for the hosted checkout return links.
func (s *RegistrationService) WithGateway(gw PaymentGateway, appURL string) *RegistrationService {
	s.gateway, s.appURL = gw, appURL
	return s
}

func (s *RegistrationService) GatewayEnabled() bool { return s.gateway != nil }

func newExternalID() string { return "rs-" + newRegistrationRef()[:24] }

// StartGatewayPayment returns the hosted checkout URL for a registration:
// the active session when there is one, otherwise a new session that holds
// the lead's schedule seat until it expires.
func (s *RegistrationService) StartGatewayPayment(ctx context.Context, ref string) (string, error) {
	if s.gateway == nil {
		return "", ErrGatewayNonaktif
	}
	reg, err := s.querier.GetLeadRegistrationByRef(ctx, ref)
	if err != nil {
		return "", ErrRefTidakAda
	}
	l, err := s.querier.GetLeadByID(ctx, reg.LeadID)
	if err != nil {
		return "", ErrRefTidakAda
	}
	if l.Status != LeadStatusOpen || l.Stage == LeadStageTerverifikasi || l.VerifiedAt.Valid {
		return "", ErrSudahDiproses
	}
	if l.PaymentAmount <= 0 {
		return "", errors.New("nominal pembayaran belum ditentukan, silakan hubungi CS")
	}
	now := s.nowUTC()
	if p, err := s.querier.GetActiveGatewayPaymentByLead(ctx, queries.GetActiveGatewayPaymentByLeadParams{LeadID: l.ID, Now: sql.NullTime{Time: now, Valid: true}}); err == nil {
		return p.CheckoutUrl, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if l.JadwalKode != "" {
		if err := s.checkSeat(ctx, l.JadwalKode, l.JenisKelamin); err != nil {
			return "", err
		}
	}
	settings, err := s.Settings(ctx)
	if err != nil {
		return "", err
	}
	expires := now.Add(time.Duration(settings.PaymentExpiryHours) * time.Hour)
	extID := newExternalID()
	pay, err := s.querier.CreateGatewayPayment(ctx, queries.CreateGatewayPaymentParams{
		LeadID: l.ID, LinkID: sql.NullInt64{Int64: reg.LinkID, Valid: true}, Provider: "xendit",
		ExternalID: sql.NullString{String: extID, Valid: true}, Amount: l.PaymentAmount,
		ExpiresAt: sql.NullTime{Time: expires, Valid: true}, JadwalKode: l.JadwalKode, JenisKelamin: l.JenisKelamin,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return "", err
	}
	desc := "Pendaftaran Ruang Sanad " + leadRef(l.ID)
	if link, err := s.querier.GetRegistrationLink(ctx, reg.LinkID); err == nil {
		if prog, err := s.querier.GetLeadProgram(ctx, link.ProgramID); err == nil && prog != nil {
			desc = prog.Nama + " · " + leadRef(l.ID)
		}
	}
	name := leadDisplayName(l)
	if name == "" {
		name = "Pendaftar"
	}
	sess, err := s.gateway.CreateSession(ctx, GatewaySessionRequest{
		ReferenceID: extID, Amount: l.PaymentAmount, ExpiresAt: expires, Description: truncateRunes(desc, 200),
		CustomerRef: "lead-" + itoa64(l.ID), CustomerName: truncateRunes(name, 50), CustomerPhone: l.Phone,
		ReturnURL: s.appURL + "/daftar/bayar/" + ref,
	})
	if err != nil {
		slog.Error("xendit create session failed", "lead_id", l.ID, "error", err)
		_, _ = s.querier.SetPaymentStatusIfPending(ctx, queries.SetPaymentStatusIfPendingParams{
			Status: "failed", CatatanSistem: truncateRunes(err.Error(), 300), UpdatedAt: s.nowUTC(), ID: pay.ID,
		})
		return "", errors.New("pembayaran online sedang bermasalah, silakan coba lagi atau gunakan transfer manual")
	}
	exp := sess.ExpiresAt
	if exp.IsZero() {
		exp = expires
	}
	if err := s.querier.UpdatePaymentGatewaySession(ctx, queries.UpdatePaymentGatewaySessionParams{
		ProviderRef: sess.ID, CheckoutUrl: sess.URL, ExpiresAt: sql.NullTime{Time: exp.UTC(), Valid: true}, UpdatedAt: s.nowUTC(), ID: pay.ID,
	}); err != nil {
		return "", err
	}
	return sess.URL, nil
}

// checkSeat fails when the schedule has no seat left for the gender.
func (s *RegistrationService) checkSeat(ctx context.Context, kode, gender string) error {
	open, err := s.jadwal.Open(ctx, gender)
	if err != nil {
		return err
	}
	for _, j := range open {
		if j.Kode != kode {
			continue
		}
		sisa := j.SisaIkhwan
		if gender == "P" {
			sisa = j.SisaAkhwat
		}
		if sisa > 0 {
			return nil
		}
	}
	return &FieldError{Field: "jadwal_kode", Pesan: "Kursi di jadwal ini baru saja penuh. Silakan hubungi CS untuk memilih jadwal lain."}
}
