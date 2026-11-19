package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

type xenditWebhook struct {
	Event string `json:"event"`
	Data  struct {
		PaymentSessionID string  `json:"payment_session_id"`
		ReferenceID      string  `json:"reference_id"`
		Status           string  `json:"status"`
		Amount           float64 `json:"amount"`
		PaymentID        string  `json:"payment_id"`
	} `json:"data"`
}

const gatewayActor = "gateway:xendit"

// ErrWebhookTidakTerbaca: the body is not JSON (answer 400, no retry).
var ErrWebhookTidakTerbaca = errors.New("webhook tidak terbaca")

// HandleXenditWebhook processes a payment session webhook whose callback
// token the handler already checked. A completed session is confirmed with
// GetSession before anything is marked paid, so a forged body cannot verify
// a lead. Unknown events and references are ignored (nil) so Xendit does
// not retry them forever.
func (s *RegistrationService) HandleXenditWebhook(ctx context.Context, body []byte) error {
	var wh xenditWebhook
	if err := json.Unmarshal(body, &wh); err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookTidakTerbaca, err)
	}
	if wh.Event != "payment_session.completed" && wh.Event != "payment_session.expired" {
		return nil
	}
	pay, err := s.querier.GetPaymentByExternalID(ctx, sql.NullString{String: wh.Data.ReferenceID, Valid: wh.Data.ReferenceID != ""})
	if errors.Is(err, sql.ErrNoRows) {
		slog.Warn("xendit webhook for unknown reference", "reference_id", wh.Data.ReferenceID)
		return nil
	}
	if err != nil {
		return err
	}
	now := s.nowUTC()
	if wh.Event == "payment_session.expired" {
		_, err := s.querier.SetPaymentStatusIfPending(ctx, queries.SetPaymentStatusIfPendingParams{Status: "expired", UpdatedAt: now, ID: pay.ID})
		return err
	}
	if pay.Status == "paid" {
		return nil
	}
	if s.gateway == nil {
		return ErrGatewayNonaktif
	}
	sess, err := s.gateway.GetSession(ctx, pay.ProviderRef)
	if err != nil {
		return err // let Xendit retry
	}
	raw := truncateRunes(string(body), 4000)
	if sess.Status != "COMPLETED" || sess.ReferenceID != wh.Data.ReferenceID {
		slog.Warn("xendit webhook not confirmed by API", "payment_id", pay.ID, "api_status", sess.Status)
		return nil
	}
	if sess.Amount != pay.Amount {
		_, err := s.querier.FailPaymentForReview(ctx, queries.FailPaymentForReviewParams{
			CatatanSistem: fmt.Sprintf("nominal di Xendit Rp%d berbeda dari tagihan Rp%d", sess.Amount, pay.Amount),
			RawWebhook:    raw, UpdatedAt: now, ID: pay.ID,
		})
		if err == nil {
			s.notifyRole(ctx, "keuangan", pay.LeadID, "Pembayaran online perlu ditinjau", "Nominal di Xendit berbeda dari tagihan untuk "+leadRef(pay.LeadID)+".")
		}
		return err
	}

	var verified bool
	var lead queries.Lead
	err = s.leads.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, pay.LeadID)
		if err != nil {
			return err
		}
		lead = l
		tinjau, catatan := int64(0), ""
		if l.Status != LeadStatusOpen || l.Stage == LeadStageTerverifikasi || l.VerifiedAt.Valid {
			tinjau, catatan = 1, "Pembayaran masuk tetapi lead sudah "+l.Status+"/"+l.Stage+"; periksa manual."
		}
		n, err := q.MarkPaymentPaid(ctx, queries.MarkPaymentPaidParams{
			PaidAt: sql.NullTime{Time: now, Valid: true}, Channel: sess.PaymentID, PerluTinjau: tinjau,
			CatatanSistem: catatan, RawWebhook: raw, UpdatedAt: now, ID: pay.ID,
		})
		if err != nil || n == 0 {
			return err // n == 0: already paid by a concurrent delivery
		}
		if pay.LinkID.Valid {
			if err := q.BumpRegistrationStat(ctx, queries.BumpRegistrationStatParams{
				LinkID: pay.LinkID.Int64, Tanggal: now.In(wib).Format(time.DateOnly), Bayar: 1,
			}); err != nil {
				return err
			}
		}
		if tinjau == 1 {
			return nil
		}
		payload, err := verifyLeadTx(ctx, q, &l, leadVerification{Amount: pay.Amount, Date: now, AnyOpenStage: true,
			Note: "Lunas otomatis via Xendit (" + sess.PaymentID + ")"}, now)
		if err != nil {
			return err
		}
		payload["payment_id"] = pay.ID
		if err := saveLead(ctx, q, l, now); err != nil {
			return err
		}
		if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("gw"), "payment.verified", gatewayActor, payload, now); err != nil {
			return err
		}
		verified = true
		lead = l
		return s.leads.enqueuePurchase(ctx, q, l.ID, l.Phone, pay.Amount, now)
	})
	if err != nil {
		return err
	}
	msg := leadDisplayName(lead) + " (" + leadRef(lead.ID) + ") membayar " + rupiahLabel(pay.Amount) + " lewat Xendit."
	if verified {
		s.notifyRole(ctx, "keuangan", lead.ID, "Pembayaran online diterima", msg+" Lead otomatis terverifikasi.")
		s.checkOverbook(ctx, pay, lead)
	} else {
		s.notifyRole(ctx, "keuangan", lead.ID, "Pembayaran online perlu ditinjau", msg+" Lead tidak sedang menunggu pembayaran.")
	}
	return nil
}

// checkOverbook flags a paid seat that pushed its schedule over quota.
func (s *RegistrationService) checkOverbook(ctx context.Context, pay queries.Payment, l queries.Lead) {
	if l.JadwalKode == "" {
		return
	}
	all, err := s.jadwal.List(ctx)
	if err != nil {
		return
	}
	for _, j := range all {
		if j.Kode != l.JadwalKode {
			continue
		}
		terisi := j.TerisiIkhwan
		if l.JenisKelamin == "P" {
			terisi = j.TerisiAkhwat
		}
		if terisi > j.KuotaPerGender {
			catatan := "Kursi kelebihan: jadwal " + j.Label + " terisi " + strconv.FormatInt(terisi, 10) + "/" + strconv.FormatInt(j.KuotaPerGender, 10)
			_ = s.querier.FlagPaymentReview(ctx, queries.FlagPaymentReviewParams{CatatanSistem: catatan, UpdatedAt: s.nowUTC(), ID: pay.ID})
			s.notifyRole(ctx, "koordinator_guru", l.ID, "Kursi jadwal kelebihan", catatan+" setelah "+leadRef(l.ID)+" membayar online.")
		}
	}
}

func (s *RegistrationService) notifyRole(ctx context.Context, role string, leadID int64, title, message string) {
	ids, err := s.querier.ListUserIDsByRole(ctx, role)
	if err != nil {
		return
	}
	for _, uid := range ids {
		_, _ = s.querier.CreateNotification(ctx, queries.CreateNotificationParams{
			UserID: uid, Type: "pembayaran_online", Title: title, Message: message,
			ActionUrl: "/app/leads/" + strconv.FormatInt(leadID, 10), ReferenceType: "lead",
			ReferenceID: sql.NullInt64{Int64: leadID, Valid: true},
		})
	}
}
