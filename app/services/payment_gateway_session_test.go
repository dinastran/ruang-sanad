package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"


	"github.com/stretchr/testify/require"
)

// fakeGateway records sessions and lets tests set GetSession answers.
type fakeGateway struct {
	created  []GatewaySessionRequest
	failNext bool
	sessions map[string]GatewaySession
}

func (g *fakeGateway) CreateSession(_ context.Context, r GatewaySessionRequest) (GatewaySession, error) {
	if g.failNext {
		g.failNext = false
		return GatewaySession{}, errors.New("xendit 500")
	}
	g.created = append(g.created, r)
	s := GatewaySession{ID: "ps-" + r.ReferenceID, ReferenceID: r.ReferenceID, Status: "ACTIVE", Amount: r.Amount,
		URL: "https://dev.xen.to/" + r.ReferenceID, ExpiresAt: r.ExpiresAt}
	if g.sessions == nil {
		g.sessions = map[string]GatewaySession{}
	}
	g.sessions[s.ID] = s
	return s, nil
}

func (g *fakeGateway) GetSession(_ context.Context, id string) (GatewaySession, error) {
	s, ok := g.sessions[id]
	if !ok {
		return s, errors.New("not found")
	}
	return s, nil
}

func gatewayLead(t *testing.T) (*registrationFixture, *fakeGateway, string, int64) {
	t.Helper()
	f, ref, leadID := bayarLead(t)
	gw := &fakeGateway{}
	f.reg.WithGateway(gw, "https://app.test")
	return f, gw, ref, leadID
}

func TestStartGatewayPaymentReusesActiveSession(t *testing.T) {
	f, gw, ref, leadID := gatewayLead(t)
	url1, err := f.reg.StartGatewayPayment(t.Context(), ref)
	require.NoError(t, err)
	url2, err := f.reg.StartGatewayPayment(t.Context(), ref)
	require.NoError(t, err)
	require.Equal(t, url1, url2)
	require.Len(t, gw.created, 1)
	r := gw.created[0]
	require.EqualValues(t, 150000, r.Amount)
	require.Equal(t, "6281234567890", r.CustomerPhone)
	require.Equal(t, "https://app.test/daftar/bayar/"+ref, r.ReturnURL)
	require.True(t, r.ExpiresAt.Equal(f.reg.nowUTC().Add(2*time.Hour)))
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Len(t, pays, 1)
	require.Equal(t, "pending", pays[0].Status)
	require.Equal(t, testJadwal, pays[0].JadwalKode)

	// Expired session: a new one is created.
	f.clock = f.clock.Add(3 * time.Hour)
	_, err = f.reg.StartGatewayPayment(t.Context(), ref)
	require.NoError(t, err)
	require.Len(t, gw.created, 2)
}

func TestPendingSessionHoldsSeat(t *testing.T) {
	f, _, ref, _ := gatewayLead(t)
	_, err := f.reg.StartGatewayPayment(t.Context(), ref)
	require.NoError(t, err)
	open, err := f.reg.jadwal.Open(t.Context(), "P")
	require.NoError(t, err)
	for _, j := range open {
		if j.Kode == testJadwal {
			require.Zero(t, j.SisaAkhwat, "kuota 1 ditahan sesi aktif")
		}
	}
	f.clock = f.clock.Add(3 * time.Hour)
	open, err = f.reg.jadwal.Open(t.Context(), "P")
	require.NoError(t, err)
	for _, j := range open {
		if j.Kode == testJadwal {
			require.EqualValues(t, 1, j.SisaAkhwat, "sesi kedaluwarsa melepas kursi")
		}
	}
}

func TestStartGatewayPaymentFullSeat(t *testing.T) {
	f, gw, ref, _ := gatewayLead(t)
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, jenis_kelamin, jadwal_kode, verified_at, created_at, updated_at)
		VALUES ('6289999999999@s.whatsapp.net', '6289999999999', 'P', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testJadwal)
	require.NoError(t, err)
	_, err = f.reg.StartGatewayPayment(t.Context(), ref)
	var fe *FieldError
	require.True(t, errors.As(err, &fe))
	require.Equal(t, "jadwal_kode", fe.Field)
	require.Empty(t, gw.created)
}

func TestStartGatewayPaymentFailureReleasesSeat(t *testing.T) {
	f, gw, ref, leadID := gatewayLead(t)
	gw.failNext = true
	_, err := f.reg.StartGatewayPayment(t.Context(), ref)
	require.Error(t, err)
	pays, err := f.querier.ListPaymentsByLead(t.Context(), leadID)
	require.NoError(t, err)
	require.Equal(t, "failed", pays[0].Status)
	holds, err := f.querier.ListActiveSeatHolds(t.Context(), sql.NullTime{Time: f.reg.nowUTC(), Valid: true})
	require.NoError(t, err)
	require.Empty(t, holds)
}

func TestStartGatewayPaymentRefusals(t *testing.T) {
	f, ref, leadID := bayarLead(t)
	_, err := f.reg.StartGatewayPayment(t.Context(), ref)
	require.True(t, errors.Is(err, ErrGatewayNonaktif))
	f.reg.WithGateway(&fakeGateway{}, "https://app.test")
	_, err = f.db.Exec(`UPDATE leads SET stage = 'terverifikasi', verified_at = CURRENT_TIMESTAMP WHERE id = ?`, leadID)
	require.NoError(t, err)
	_, err = f.reg.StartGatewayPayment(t.Context(), ref)
	require.True(t, errors.Is(err, ErrSudahDiproses))
}
