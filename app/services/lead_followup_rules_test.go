package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Conversation of seedConversation: last CS message 2026-10-24 03:01 UTC.
func fuAt(day int, hourUTC int) time.Time {
	return time.Date(2026, 10, day, hourUTC, 0, 0, 0, time.UTC)
}

func proposeAI(t *testing.T, c *crmAgentFixture, leadID int64, in LeadFollowupInput) (int64, error) {
	t.Helper()
	return c.service.ProposeFollowup(context.Background(), leadID, in, "ai", c.tokenID)
}

func requireBlock(t *testing.T, err error, kind, contains string) {
	t.Helper()
	var b *FollowupBlock
	require.True(t, errors.As(err, &b), "want FollowupBlock, got %v", err)
	require.Equal(t, kind, b.Kind, b.Msg)
	require.Contains(t, b.Msg, contains)
}

const d1Draft = "Assalamu'alaikum Kak 😊 Izin memastikan info kelas kemarin sudah terbaca. Mau saya bantu cek jadwal yang cocok?"

func TestFollowupTimingRelativeToLastCSMessage(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)

	_, err := proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 2), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockReject, "terlalu cepat")
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d3", ScheduledAt: fuAt(26, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockReject, "terlalu cepat")

	fid, err := proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: d1Draft})
	require.NoError(t, err)
	f, err := c.querier.ListLeadFollowups(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, f, 1)
	require.Equal(t, fid, f[0].ID)
	require.Equal(t, "pending_approval", f[0].Status)
	require.Equal(t, "ai", f[0].CreatedByKind)

	// Same step already waiting → rejected.
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(26, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockReject, "sudah menunggu persetujuan")
	// One touch per WIB day (manual from CS on the same day).
	_, err = c.service.ProposeFollowup(context.Background(), id, LeadFollowupInput{Step: "manual", ScheduledAt: fuAt(25, 9), DraftText: d1Draft}, "user", c.csID)
	requireBlock(t, err, FollowupBlockReject, "satu sentuhan per hari")
	_, err = c.service.ProposeFollowup(context.Background(), id, LeadFollowupInput{Step: "manual", ScheduledAt: fuAt(26, 9), DraftText: d1Draft}, "user", c.csID)
	require.NoError(t, err)
}

func TestFollowupBlockedWhenNoCSMessageOrLeadReplied(t *testing.T) {
	c := setupCRMAgent(t)
	c.ingest(t, leadEvent("evt_x1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "X1", "body": "Halo"}))
	id := c.lead(t).ID
	_, err := proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(26, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockReject, "belum ada pesan keluar")

	c.ingest(t,
		leadEvent("evt_x2", "message.out", "2026-10-24T03:01:00Z", map[string]string{"wa_message_id": "X2", "body": "Halo Kak", "sender": "ai"}),
		leadEvent("evt_x3", "message.in", "2026-10-24T03:30:00Z", map[string]string{"wa_message_id": "X3", "body": "Oke kak"}),
	)
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(26, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockReject, "sudah membalas")
}

func TestFollowupStopAndStatusRules(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)

	_, err := c.db.Exec(`UPDATE leads SET followup_permission = 'no' WHERE id = ?`, id)
	require.NoError(t, err)
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockCancel, "berhenti")

	_, err = c.db.Exec(`UPDATE leads SET followup_permission = 'unknown', status = 'lost', lost_reason = 'harga' WHERE id = ?`, id)
	require.NoError(t, err)
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockSkip, "status sekarang: lost")

	_, err = c.db.Exec(`UPDATE leads SET status = 'do_not_contact' WHERE id = ?`, id)
	require.NoError(t, err)
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockCancel, "berhenti")
}

func TestFollowupStepSpecificRules(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)

	// d7 needs a real update as reason.
	_, err := proposeAI(t, c, id, LeadFollowupInput{Step: "d7", ScheduledAt: fuAt(31, 4), DraftText: d1Draft, Reason: "follow up"})
	requireBlock(t, err, FollowupBlockReject, "reason")
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d7", ScheduledAt: fuAt(31, 4), DraftText: d1Draft, Reason: "Jadwal kelas malam akhwat sudah terkonfirmasi Senin"})
	require.NoError(t, err)

	// d14 must offer "boleh kabari".
	d14At := time.Date(2026, 11, 7, 4, 0, 0, 0, time.UTC)
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d14", ScheduledAt: d14At, DraftText: "Saya izin menutup follow-up ya Kak."})
	requireBlock(t, err, FollowupBlockReject, "boleh kabari")
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d14", ScheduledAt: d14At,
		DraftText: "Saya izin menutup follow-up ya Kak 🙏🏻 Jika masih ingin dikabari, cukup balas *boleh kabari*."})
	require.NoError(t, err)

	// d30 only with permission yes (and a reason).
	d30At := time.Date(2026, 11, 23, 4, 0, 0, 0, time.UTC)
	reason := "Kelas akhwat malam baru dibuka bulan depan"
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d30", ScheduledAt: d30At, DraftText: d1Draft, Reason: reason})
	requireBlock(t, err, FollowupBlockReject, "boleh kabari")
	_, err = c.db.Exec(`UPDATE leads SET followup_permission = 'yes' WHERE id = ?`, id)
	require.NoError(t, err)
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d30", ScheduledAt: d30At, DraftText: d1Draft})
	requireBlock(t, err, FollowupBlockReject, "reason")
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d30", ScheduledAt: d30At, DraftText: d1Draft, Reason: reason})
	require.NoError(t, err)
}

func TestFollowupContentRules(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)
	for _, draft := range []string{
		"Biayanya hanya Rp149.000 per bulan Kak",
		"Cukup 149rb saja",
		"Investasinya 149.000",
		"Silakan transfer ke BSI 7123456789",
		"No rek kami ada di bio",
		"Isi [[FORM]] ya Kak",
	} {
		_, err := proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: draft})
		var b *FollowupBlock
		require.True(t, errors.As(err, &b), draft)
		require.Equal(t, FollowupBlockReject, b.Kind, draft)
	}
	_, err := proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(25, 4), DraftText: ""})
	requireBlock(t, err, FollowupBlockReject, "wajib")

	// Input errors: past time, AI cannot use "manual".
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "d1", ScheduledAt: fuAt(23, 4), DraftText: d1Draft})
	require.ErrorContains(t, err, "masa lalu")
	_, err = proposeAI(t, c, id, LeadFollowupInput{Step: "manual", ScheduledAt: fuAt(25, 4), DraftText: d1Draft})
	require.ErrorContains(t, err, "step harus")
}

func TestFollowupDraftWarnings(t *testing.T) {
	require.Empty(t, FollowupDraftWarnings("d1", d1Draft, "", "unknown"))
	require.Len(t, FollowupDraftWarnings("d1", "Cek https://ruangsanad.com ya", "", "unknown"), 1)
	require.Len(t, FollowupDraftWarnings("d7", "Biaya Rp100.000", "", "no"), 3)
}

func TestFollowupProposeViaMCP(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)
	text, isErr := c.call(t, c.writer(), "usulkan_followup", map[string]any{
		"lead_ref": "L-1", "step": "d1", "scheduled_at": "2026-10-25 11:00", "draft_text": d1Draft,
	})
	require.False(t, isErr, text)
	require.Contains(t, text, "pending_approval")
	f, err := c.querier.ListLeadFollowups(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, f, 1)
	require.Equal(t, fuAt(25, 4), f[0].ScheduledAt.UTC(), "11:00 WIB = 04:00 UTC")

	text, isErr = c.call(t, c.writer(), "usulkan_followup", map[string]any{
		"lead_ref": "L-1", "step": "d3", "scheduled_at": "2026-10-26 11:00", "draft_text": d1Draft,
	})
	require.True(t, isErr)
	require.Contains(t, text, "follow-up ditolak: terlalu cepat")
}
