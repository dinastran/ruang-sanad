package services

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// paidSession starts a gateway payment and marks the fake session COMPLETED.
func paidSession(t *testing.T) (*registrationFixture, *fakeGateway, int64, string) {
	t.Helper()
	f, gw, ref, leadID := gatewayLead(t)
	f.service.WithMetaCAPI(newTestCAPI(t, f.leadFixture, &fakeMeta{}, MetaCAPIConfig{DatasetID: "d", Token: "t"}))
	_, err := f.reg.StartGatewayPayment(t.Context(), ref)
	require.NoError(t, err)
	extID := gw.created[0].ReferenceID
	s := gw.sessions["ps-"+extID]
	s.Status, s.PaymentID = "COMPLETED", "py-1"
	gw.sessions[s.ID] = s
	return f, gw, leadID, extID
}

func webhookBody(event, extID string, amount int64) []byte {
	return []byte(fmt.Sprintf(`{"event":%q,"business_id":"b","created":"2026-10-24T05:00:00Z","data":{"payment_session_id":"ps-%s","reference_id":%q,"status":"COMPLETED","amount":%d,"currency":"IDR","payment_id":"py-1"}}`,
		event, extID, extID, amount))
}

func TestWebhookCompletedVerifiesOnce(t *testing.T) {
	f, _, leadID, extID := paidSession(t)
	for range 2 {
		require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.completed", extID, 150000)))
	}
	l, err := f.querier.GetLeadByID(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, LeadStageTerverifikasi, l.Stage)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM meta_capi_events WHERE lead_id = ? AND event_name = 'Purchase'`, leadID))
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM lead_events WHERE lead_id = ? AND type = 'payment.verified' AND actor = 'gateway:xendit'`, leadID))
	require.Equal(t, 1, f.count(t, `SELECT COALESCE(SUM(bayar),0) FROM registration_link_stats`))
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, "paid", pays[0].Status)
	require.Equal(t, "py-1", pays[0].Channel)
	require.Zero(t, pays[0].PerluTinjau)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND reference_id = ?`, f.keuanganID, leadID))
}

func TestWebhookNotCompletedAtXendit(t *testing.T) {
	f, gw, leadID, extID := paidSession(t)
	s := gw.sessions["ps-"+extID]
	s.Status = "ACTIVE"
	gw.sessions[s.ID] = s
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.completed", extID, 150000)))
	l, err := f.querier.GetLeadByID(t.Context(), leadID)
	require.NoError(t, err)
	require.NotEqual(t, LeadStageTerverifikasi, l.Stage, "webhook palsu tidak menandai lunas")
	pays, _ := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.Equal(t, "pending", pays[0].Status)
}

func TestWebhookAmountMismatch(t *testing.T) {
	f, gw, leadID, extID := paidSession(t)
	s := gw.sessions["ps-"+extID]
	s.Amount = 1000
	gw.sessions[s.ID] = s
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.completed", extID, 1000)))
	l, _ := f.querier.GetLeadByID(t.Context(), leadID)
	require.NotEqual(t, LeadStageTerverifikasi, l.Stage)
	pays, _ := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.Equal(t, "failed", pays[0].Status)
	require.EqualValues(t, 1, pays[0].PerluTinjau)
}

func TestWebhookLeadNoLongerOpen(t *testing.T) {
	f, _, leadID, extID := paidSession(t)
	_, err := f.db.Exec(`UPDATE leads SET status = 'lost', lost_reason = 'harga' WHERE id = ?`, leadID)
	require.NoError(t, err)
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.completed", extID, 150000)))
	l, _ := f.querier.GetLeadByID(t.Context(), leadID)
	require.Equal(t, LeadStatusLost, l.Status)
	require.NotEqual(t, LeadStageTerverifikasi, l.Stage)
	pays, _ := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.Equal(t, "paid", pays[0].Status)
	require.EqualValues(t, 1, pays[0].PerluTinjau)
}

func TestWebhookExpiredAndUnknown(t *testing.T) {
	f, _, leadID, extID := paidSession(t)
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), []byte(`{"event":"payment.capture","data":{}}`)))
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.completed", "rs-tidak-ada", 1)))
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.expired", extID, 150000)))
	pays, _ := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.Equal(t, "expired", pays[0].Status)
	require.Error(t, f.reg.HandleXenditWebhook(t.Context(), []byte(`not json`)))
}

func TestWebhookOverbookFlagsCoordinator(t *testing.T) {
	f, _, leadID, extID := paidSession(t)
	res, err := f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('ko@example.com', 'Koor', 'koordinator_guru')`)
	require.NoError(t, err)
	koorID, _ := res.LastInsertId()
	_, err = f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, jenis_kelamin, jadwal_kode, verified_at, created_at, updated_at)
		VALUES ('6289999999999@s.whatsapp.net', '6289999999999', 'P', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testJadwal)
	require.NoError(t, err)
	require.NoError(t, f.reg.HandleXenditWebhook(t.Context(), webhookBody("payment_session.completed", extID, 150000)))
	l, _ := f.querier.GetLeadByID(t.Context(), leadID)
	require.Equal(t, LeadStageTerverifikasi, l.Stage, "uang sudah masuk: tetap diterima")
	pays, _ := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.EqualValues(t, 1, pays[0].PerluTinjau)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM notifications WHERE user_id = ?`, koorID))
}
