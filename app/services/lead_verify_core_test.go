package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestVerifyLeadTxGateway(t *testing.T) {
	f := setupLeadFixture(t)
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, stage, payment_amount, created_at, updated_at)
		VALUES (?, '6281234567890', 'info_bayar_dikirim', 150000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testChatID)
	require.NoError(t, err)
	l0, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)

	err = f.service.withTx(t.Context(), func(q *queries.Querier) error {
		l, err := loadLead(t.Context(), q, l0.ID)
		require.NoError(t, err)
		now := f.clock
		payload, err := verifyLeadTx(t.Context(), q, &l, leadVerification{Amount: 150000, Date: now, AnyOpenStage: true}, now)
		require.NoError(t, err)
		require.NoError(t, saveLead(t.Context(), q, l, now))
		return writeLeadEvent(t.Context(), q, l.ID, newLeadEventID("gw"), "payment.verified", "gateway:xendit", payload, now)
	})
	require.NoError(t, err)

	l, err := f.querier.GetLeadByID(t.Context(), l0.ID)
	require.NoError(t, err)
	require.Equal(t, LeadStageTerverifikasi, l.Stage)
	require.True(t, l.VerifiedAt.Valid)
	require.False(t, l.VerifiedBy.Valid, "sistem: verified_by kosong")
	amount, err := f.querier.GetMetaCAPIPurchaseAmount(t.Context(), l.ID)
	require.NoError(t, err)
	require.EqualValues(t, 150000, amount)

	// Staff path keeps its stricter stage rule.
	_, err = f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, stage, created_at, updated_at) VALUES ('x@s.whatsapp.net', '6280000000009', 'data_terisi', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	err = f.service.withTx(t.Context(), func(q *queries.Querier) error {
		l2, err := q.GetLeadByChatID(t.Context(), "x@s.whatsapp.net")
		require.NoError(t, err)
		_, err = verifyLeadTx(t.Context(), q, &l2, leadVerification{Amount: 1, Date: time.Now(), ActorUserID: f.keuanganID}, f.clock)
		return err
	})
	require.Error(t, err)
}
