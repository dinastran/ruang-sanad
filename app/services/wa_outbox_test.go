package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

type fakeBridge struct {
	sent []OutboundRequest
	err  error
	n    int
}

func (b *fakeBridge) SendOutbound(_ context.Context, req OutboundRequest) (string, error) {
	if b.err != nil {
		return "", b.err
	}
	b.n++
	b.sent = append(b.sent, req)
	return "WAOUT" + string(rune('0'+b.n)), nil
}

type outboxFixture struct {
	*crmAgentFixture
	bridge *fakeBridge
	worker *WAOutboxService
}

func setupOutbox(t *testing.T) *outboxFixture {
	t.Helper()
	c := setupCRMAgent(t)
	b := &fakeBridge{}
	w := NewWAOutboxService(c.querier, b)
	w.now = func() time.Time { return c.clock }
	_, err := c.db.Exec(`UPDATE crm_settings SET followup_enabled = 1`)
	require.NoError(t, err)
	return &outboxFixture{crmAgentFixture: c, bridge: b, worker: w}
}

func (o *outboxFixture) tick(t *testing.T) TickResult {
	t.Helper()
	res, err := o.worker.Tick(context.Background())
	require.NoError(t, err)
	return res
}

func (o *outboxFixture) at(t time.Time) { o.clock = t }

// approvedD1 creates the standard conversation plus an approved d1 at
// 2026-10-25 04:00 UTC (11:00 WIB) for chat chatID.
func (o *outboxFixture) approvedD1(t *testing.T, chatID string) (int64, int64) {
	t.Helper()
	for _, ev := range []models.LeadIngestEvent{
		leadEvent(chatID+"_1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": chatID + "M1", "body": "Mau tanya kelas"}),
		leadEvent(chatID+"_2", "message.out", "2026-10-24T03:01:00Z", map[string]string{"wa_message_id": chatID + "M2", "body": "Ada kelas 1x dan 2x Kak", "sender": "ai"}),
	} {
		ev.ChatID = chatID + "@s.whatsapp.net"
		ev.Phone = chatID
		o.ingest(t, ev)
	}
	l, err := o.querier.GetLeadByChatID(context.Background(), chatID+"@s.whatsapp.net")
	require.NoError(t, err)
	saved := o.clock
	o.at(time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC))
	fid, err := o.service.ProposeFollowup(context.Background(), l.ID, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: d1Draft}, "ai", o.tokenID)
	require.NoError(t, err)
	require.NoError(t, o.service.ApproveFollowup(context.Background(), fid, o.csID, models.LeadFollowupRequest{}))
	o.at(saved)
	return l.ID, fid
}

func (o *outboxFixture) followup(t *testing.T, id int64) queries.LeadFollowup {
	t.Helper()
	f, err := o.querier.GetLeadFollowupByID(context.Background(), id)
	require.NoError(t, err)
	return f
}

func TestOutboxSendsApprovedFollowupWhenDue(t *testing.T) {
	o := setupOutbox(t)
	leadID, fid := o.approvedD1(t, "6281200000001")

	o.at(fuAt(25, 3)) // before scheduled_at
	require.Zero(t, o.tick(t).Queued)
	require.Empty(t, o.bridge.sent)

	o.at(fuAt(25, 4))
	res := o.tick(t)
	require.EqualValues(t, 1, res.Queued)
	require.Equal(t, 1, res.Sent)
	require.Len(t, o.bridge.sent, 1)
	req := o.bridge.sent[0]
	require.Equal(t, "6281200000001@s.whatsapp.net", req.ChatID)
	require.Equal(t, d1Draft, req.Text)
	require.Equal(t, "followup", req.Purpose)
	require.Equal(t, fid, req.ReferenceID)

	f := o.followup(t, fid)
	require.Equal(t, "sent", f.Status)
	require.Equal(t, "WAOUT1", f.WaMessageID)

	// Nothing is sent twice.
	o.at(fuAt(25, 6))
	o.tick(t)
	require.Len(t, o.bridge.sent, 1)

	// The bridge then reports the message; it lands in the transcript as followup.
	o.ingest(t, func() models.LeadIngestEvent {
		ev := leadEvent("evt_fu_out", "message.out", "2026-10-25T04:00:05Z", map[string]any{
			"wa_message_id": "WAOUT1", "body": d1Draft, "sender": "followup", "reference_id": fid, "purpose": "followup"})
		ev.ChatID, ev.Phone = "6281200000001@s.whatsapp.net", "6281200000001"
		return ev
	}())
	msgs, err := o.querier.ListLeadMessagesLatest(context.Background(), queries.ListLeadMessagesLatestParams{LeadID: leadID, Limit: 1})
	require.NoError(t, err)
	require.Equal(t, "followup", msgs[0].Sender)
}

func TestOutboxRespectsSwitchAndSendWindow(t *testing.T) {
	o := setupOutbox(t)
	_, fid := o.approvedD1(t, "6281200000002")

	_, err := o.db.Exec(`UPDATE crm_settings SET followup_enabled = 0`)
	require.NoError(t, err)
	o.at(fuAt(25, 4))
	require.Zero(t, o.tick(t).Queued)
	require.Empty(t, o.bridge.sent)

	// Window 12:00–20:00 WIB: 11:00 WIB waits, 12:00 WIB sends.
	_, err = o.db.Exec(`UPDATE crm_settings SET followup_enabled = 1, followup_send_start = '12:00'`)
	require.NoError(t, err)
	res := o.tick(t)
	require.EqualValues(t, 1, res.Queued)
	require.Zero(t, res.Sent)
	require.Equal(t, "di luar jam kirim", res.Note)
	o.at(fuAt(25, 13)) // 20:00 WIB: window end is exclusive
	require.Zero(t, o.tick(t).Sent)
	o.at(fuAt(25, 5)) // 12:00 WIB
	require.Equal(t, 1, o.tick(t).Sent)
	require.Equal(t, "sent", o.followup(t, fid).Status)
}

func TestOutboxDailyCapAndGap(t *testing.T) {
	o := setupOutbox(t)
	_, f1 := o.approvedD1(t, "6281200000003")
	_, f2 := o.approvedD1(t, "6281200000004")
	_, f3 := o.approvedD1(t, "6281200000005")
	_, err := o.db.Exec(`UPDATE crm_settings SET followup_daily_cap = 2, followup_min_gap_seconds = 300`)
	require.NoError(t, err)

	o.at(fuAt(25, 4))
	require.Equal(t, 1, o.tick(t).Sent)
	o.at(fuAt(25, 4).Add(time.Minute)) // gap 5 min not over
	res := o.tick(t)
	require.Zero(t, res.Sent)
	require.Equal(t, "menunggu jeda minimum", res.Note)
	o.at(fuAt(25, 4).Add(5 * time.Minute))
	require.Equal(t, 1, o.tick(t).Sent)
	o.at(fuAt(25, 5)) // cap 2 reached today
	res = o.tick(t)
	require.Zero(t, res.Sent)
	require.Equal(t, "batas harian tercapai", res.Note)
	o.at(fuAt(26, 2)) // next WIB day, 09:00
	require.Equal(t, 1, o.tick(t).Sent)

	sent := 0
	for _, id := range []int64{f1, f2, f3} {
		if o.followup(t, id).Status == "sent" {
			sent++
		}
	}
	require.Equal(t, 3, sent)
}

func TestOutboxBridgeDownFailsAfterThreeAttemptsThenRetry(t *testing.T) {
	o := setupOutbox(t)
	leadID, fid := o.approvedD1(t, "6281200000006")
	o.bridge.err = &OutboundError{Msg: "bridge tidak bisa dihubungi"}

	o.at(fuAt(25, 4))
	require.Equal(t, 1, o.tick(t).Failed)
	o.at(fuAt(25, 4).Add(30 * time.Second)) // backoff 1 min
	o.tick(t)
	o.at(fuAt(25, 4).Add(time.Minute))
	require.Equal(t, 1, o.tick(t).Failed)
	o.at(fuAt(25, 4).Add(3 * time.Minute)) // backoff 5 min
	require.Zero(t, o.tick(t).Failed)
	o.at(fuAt(25, 4).Add(6 * time.Minute))
	require.Equal(t, 1, o.tick(t).Failed)

	f := o.followup(t, fid)
	require.Equal(t, "approved", f.Status, "follow-up stays approved")
	require.Contains(t, f.Error, "Gagal kirim")
	failed, err := o.querier.ListFailedWaOutbox(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, failed, 1)
	require.EqualValues(t, 3, failed[0].Attempts)
	require.Equal(t, leadID, failed[0].LeadID)

	// Nothing more happens by itself.
	o.at(fuAt(25, 6))
	o.tick(t)
	require.Empty(t, o.bridge.sent)

	// "Coba lagi" after the bridge is back.
	o.bridge.err = nil
	require.NoError(t, o.worker.Retry(context.Background(), failed[0].ID, o.csID))
	require.Equal(t, 1, o.tick(t).Sent)
	f = o.followup(t, fid)
	require.Equal(t, "sent", f.Status)
	require.Empty(t, f.Error)
	require.Error(t, o.worker.Retry(context.Background(), failed[0].ID, o.csID), "only failed sends can be retried")
}

func TestOutboxPermanentBridgeErrorFailsAtOnce(t *testing.T) {
	o := setupOutbox(t)
	o.approvedD1(t, "6281200000007")
	o.bridge.err = &OutboundError{Msg: "bridge menolak token", Permanent: true}
	o.at(fuAt(25, 4))
	require.Equal(t, 1, o.tick(t).Failed)
	failed, err := o.querier.ListFailedWaOutbox(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, failed, 1)
	require.EqualValues(t, 1, failed[0].Attempts)
}

func TestOutboxRulesRecheckedBeforeSend(t *testing.T) {
	cases := []struct {
		name, sql, status string
	}{
		{"lead minta berhenti", `UPDATE leads SET followup_permission = 'no' WHERE id = ?`, "cancelled"},
		{"lead jangan hubungi", `UPDATE leads SET status = 'do_not_contact' WHERE id = ?`, "cancelled"},
		{"lead hilang", `UPDATE leads SET status = 'lost', lost_reason = 'harga' WHERE id = ?`, "skipped"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := setupOutbox(t)
			leadID, fid := o.approvedD1(t, "6281200000008")
			_, err := o.db.Exec(tc.sql, leadID)
			require.NoError(t, err)
			o.at(fuAt(25, 4))
			o.tick(t)
			require.Empty(t, o.bridge.sent)
			require.Equal(t, tc.status, o.followup(t, fid).Status)
		})
	}

	t.Run("label uji bertahap", func(t *testing.T) {
		o := setupOutbox(t)
		_, fid := o.approvedD1(t, "6281200000009")
		_, err := o.db.Exec(`UPDATE crm_settings SET followup_only_label = 'uji-followup'`)
		require.NoError(t, err)
		o.at(fuAt(25, 4))
		o.tick(t)
		require.Empty(t, o.bridge.sent)
		f := o.followup(t, fid)
		require.Equal(t, "skipped", f.Status)
		require.Contains(t, f.Error, "uji-followup")

		leadID, fid2 := o.approvedD1(t, "6281200000010")
		var labelID int64
		require.NoError(t, o.db.QueryRow(`SELECT id FROM lead_labels WHERE nama = 'uji-followup'`).Scan(&labelID))
		require.NoError(t, o.service.SetLabel(context.Background(), leadID, o.csID, labelID, true))
		o.at(fuAt(25, 5))
		o.tick(t)
		require.Equal(t, "sent", o.followup(t, fid2).Status)
	})

	t.Run("satu sentuhan per hari ditunda", func(t *testing.T) {
		o := setupOutbox(t)
		leadID, fid := o.approvedD1(t, "6281200000011")
		// A manual follow-up was already sent to this lead today.
		_, err := o.db.Exec(`INSERT INTO lead_followups (lead_id, step, scheduled_at, draft_text, status, sent_at) VALUES (?, 'manual', ?, 'x', 'sent', ?)`,
			leadID, fuAt(25, 2), fuAt(25, 2))
		require.NoError(t, err)
		o.at(fuAt(25, 4))
		o.tick(t)
		require.Empty(t, o.bridge.sent)
		require.Equal(t, "approved", o.followup(t, fid).Status)
		o.at(fuAt(26, 1)) // 08:00 WIB next day
		o.tick(t)
		require.Len(t, o.bridge.sent, 1)
	})
}

func TestOutboxLeadReplyCancelsQueuedFollowup(t *testing.T) {
	o := setupOutbox(t)
	_, err := o.db.Exec(`UPDATE crm_settings SET followup_send_start = '12:00'`)
	require.NoError(t, err)
	_, fid := o.approvedD1(t, "6281200000012")
	o.at(fuAt(25, 4))
	require.EqualValues(t, 1, o.tick(t).Queued) // queued, waiting for the window

	ev := leadEvent("evt_reply", "message.in", "2026-10-25T04:30:00Z", map[string]string{"wa_message_id": "R1", "body": "Iya kak, jadwal malam ada?"})
	ev.ChatID, ev.Phone = "6281200000012@s.whatsapp.net", "6281200000012"
	o.ingest(t, ev)
	require.Equal(t, "cancelled", o.followup(t, fid).Status)

	o.at(fuAt(25, 6))
	o.tick(t)
	require.Empty(t, o.bridge.sent)
}

func TestOutboxStuckSendingMarkedFailed(t *testing.T) {
	o := setupOutbox(t)
	o.approvedD1(t, "6281200000013")
	_, err := o.db.Exec(`UPDATE crm_settings SET followup_send_start = '12:00'`)
	require.NoError(t, err)
	o.at(fuAt(25, 4))
	o.tick(t)
	_, err = o.db.Exec(`UPDATE wa_outbox SET status = 'sending', updated_at = ?`, fuAt(25, 4))
	require.NoError(t, err)
	o.at(fuAt(25, 4).Add(11 * time.Minute))
	o.tick(t)
	failed, err := o.querier.ListFailedWaOutbox(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, failed, 1)
	require.Contains(t, failed[0].LastError, "tidak pasti")
	require.Empty(t, o.bridge.sent)
}

func TestPaymentConfirmationThroughOutbox(t *testing.T) {
	o := setupOutbox(t)
	_, err := o.db.Exec(`UPDATE crm_settings SET followup_enabled = 0`)
	require.NoError(t, err)
	o.ingest(t,
		leadEvent("evt_p1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "P1", "body": "Daftar kak"}),
		leadEvent("evt_p2", "form.sent", "2026-10-24T03:05:00Z", map[string]string{}),
		leadEvent("evt_p3", "message.in", "2026-10-24T03:10:00Z", map[string]string{"wa_message_id": "P3", "body": "Nama: Fulanah Zahra\nUsia: 25\nDomisili: Bekasi\nIkhwan/Akhwat: Akhwat\nPilihan kelas: 1x"}),
		leadEvent("evt_p4", "payment_info.sent", "2026-10-24T03:20:00Z", map[string]string{"paket_kode": "1x", "nominal_text": "Rp149.000"}),
		leadEvent("evt_p5", "proof.received", "2026-10-24T03:40:00Z", map[string]string{"media_type": "gambar"}),
	)
	l := o.lead(t)
	req := models.LeadVerifyRequest{Amount: 149000, Date: "2026-10-24", SendConfirmation: true}
	require.ErrorContains(t, o.service.VerifyPayment(context.Background(), l.ID, o.csID, req), "template konfirmasi belum diisi")
	require.Equal(t, LeadStageBuktiMasuk, o.lead(t).Stage, "nothing changed when the confirmation cannot be queued")

	_, err = o.db.Exec(`UPDATE crm_settings SET payment_verified_template = 'Terima kasih Kak [NAMA], pembayaran [NOMINAL] sudah kami terima.'`)
	require.NoError(t, err)
	require.NoError(t, o.service.VerifyPayment(context.Background(), l.ID, o.csID, req))

	// Confirmations ignore the follow-up switch and window, but go through the bridge.
	o.at(time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC)) // 22:00 WIB
	require.Equal(t, 1, o.tick(t).Sent)
	require.Len(t, o.bridge.sent, 1)
	sent := o.bridge.sent[0]
	require.Equal(t, "payment_verified", sent.Purpose)
	require.Zero(t, sent.ReferenceID)
	require.Equal(t, "Terima kasih Kak Fulanah, pembayaran Rp149.000 sudah kami terima.", sent.Text)

	// A message.out for another purpose never marks a follow-up as sent.
	fid, err := o.querier.CreateLeadFollowup(context.Background(), queries.CreateLeadFollowupParams{
		LeadID: l.ID, Step: "manual", ScheduledAt: fuAt(26, 4), DraftText: "x", CreatedByKind: "user", CreatedAt: o.clock, UpdatedAt: o.clock})
	require.NoError(t, err)
	o.ingest(t, leadEvent("evt_p6", "message.out", "2026-10-24T15:00:05Z", map[string]any{
		"wa_message_id": "WAOUT1", "body": sent.Text, "sender": "followup", "reference_id": fid, "purpose": "payment_verified"}))
	require.Equal(t, "pending_approval", o.followup(t, fid).Status)
}

func TestFollowupApprovalActions(t *testing.T) {
	o := setupOutbox(t)
	o.ingest(t,
		leadEvent("evt_a1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "A1", "body": "Halo"}),
		leadEvent("evt_a2", "message.out", "2026-10-24T03:01:00Z", map[string]string{"wa_message_id": "A2", "body": "Halo Kak", "sender": "ai"}),
	)
	l := o.lead(t)
	ctx := context.Background()
	f1, err := o.service.ProposeFollowup(ctx, l.ID, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: d1Draft}, "ai", o.tokenID)
	require.NoError(t, err)
	f3, err := o.service.ProposeFollowup(ctx, l.ID, LeadFollowupInput{Step: "d3", ScheduledAt: fuAt(27, 4), DraftText: "Info lengkap di https://ruangsanad.com Kak"}, "ai", o.tokenID)
	require.NoError(t, err)

	// Bulk approval skips drafts with a warning (link).
	n, skipped, err := o.service.BulkApprove(ctx, []int64{f1, f3}, o.csID)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Len(t, skipped, 1)
	require.Contains(t, skipped[0], "link")
	require.Equal(t, "approved", o.followup(t, f1).Status)
	require.Equal(t, "pending_approval", o.followup(t, f3).Status)

	// Edit & approve validates the edited draft.
	err = o.service.ApproveFollowup(ctx, f3, o.csID, models.LeadFollowupRequest{DraftText: "Biayanya Rp149.000 Kak"})
	require.Error(t, err)
	require.NoError(t, o.service.ApproveFollowup(ctx, f3, o.csID, models.LeadFollowupRequest{DraftText: "Kak, jadwal malam masih ada. Mau saya bantu pilih?", ScheduledAt: "2026-10-27T12:00"}))
	f := o.followup(t, f3)
	require.Equal(t, "approved", f.Status)
	require.Equal(t, time.Date(2026, 10, 27, 5, 0, 0, 0, time.UTC), f.ScheduledAt.UTC())

	// Reschedule of a queued follow-up cancels its queued send; it is re-queued at the new time.
	o.at(fuAt(25, 4))
	_, err = o.db.Exec(`UPDATE crm_settings SET followup_send_start = '12:00'`)
	require.NoError(t, err)
	require.EqualValues(t, 1, o.tick(t).Queued)
	require.NoError(t, o.service.RescheduleFollowup(ctx, f1, o.csID, "2026-10-26 13:00"))
	o.at(fuAt(25, 6))
	o.tick(t)
	require.Empty(t, o.bridge.sent, "old time no longer sends")
	o.at(fuAt(26, 6))
	o.tick(t)
	require.Len(t, o.bridge.sent, 1)
	require.Equal(t, f1, o.bridge.sent[0].ReferenceID)

	// Cancel.
	require.NoError(t, o.service.CancelFollowup(ctx, f3, o.csID))
	require.Equal(t, "cancelled", o.followup(t, f3).Status)
	require.Error(t, o.service.CancelFollowup(ctx, f3, o.csID))

	// Approval is blocked when the lead asked to stop.
	f7, err := o.service.ProposeFollowup(ctx, l.ID, LeadFollowupInput{Step: "d7", ScheduledAt: fuAt(31, 4), DraftText: d1Draft, Reason: "Jadwal kelas malam akhwat sudah dibuka"}, "ai", o.tokenID)
	require.NoError(t, err)
	_, err = o.db.Exec(`UPDATE leads SET followup_permission = 'no' WHERE id = ?`, l.ID)
	require.NoError(t, err)
	require.ErrorContains(t, o.service.ApproveFollowup(ctx, f7, o.csID, models.LeadFollowupRequest{}), "berhenti")

	page, err := o.service.FollowupQueue(ctx, "cs")
	require.NoError(t, err)
	require.Len(t, page.Pending, 1)
	require.True(t, page.Pending[0].Blocked)
	require.True(t, page.CanApprove)
	page, err = o.service.FollowupQueue(ctx, "keuangan")
	require.NoError(t, err)
	require.False(t, page.CanApprove)
}

func TestCSBridgeClient(t *testing.T) {
	var got OutboundRequest
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/outbound", r.URL.Path)
		require.Equal(t, "Bearer rahasia", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"wa_message_id":"3EB0XYZ","error":"paused"}`))
	}))
	defer srv.Close()
	c := NewCSBridgeClient(srv.URL+"/", "rahasia")
	id, err := c.SendOutbound(context.Background(), OutboundRequest{ChatID: "628@s.whatsapp.net", Text: "Halo", ReferenceID: 7, Purpose: "followup"})
	require.NoError(t, err)
	require.Equal(t, "3EB0XYZ", id)
	require.Equal(t, int64(7), got.ReferenceID)

	for code, permanent := range map[int]bool{http.StatusConflict: false, http.StatusBadGateway: false, http.StatusUnauthorized: true, http.StatusBadRequest: true} {
		status = code
		_, err := c.SendOutbound(context.Background(), OutboundRequest{ChatID: "x", Text: "y"})
		var oe *OutboundError
		require.ErrorAs(t, err, &oe)
		require.Equal(t, permanent, oe.Permanent, code)
	}
	_, err = NewCSBridgeClient("", "").SendOutbound(context.Background(), OutboundRequest{})
	require.ErrorIs(t, err, ErrBridgeNotConfigured)
}
