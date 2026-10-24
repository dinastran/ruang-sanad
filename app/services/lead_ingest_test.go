package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

type leadFixture struct {
	db          *sql.DB
	querier     *queries.Querier
	service     *LeadService
	csID        int64
	keuanganID  int64
	adminKelas1 int64
	adminKelas2 int64
	clock       time.Time
}

func setupLeadFixture(t *testing.T) *leadFixture {
	t.Helper()
	db, querier := setupSantriDatabase(t)
	insertUser := func(email, role string) int64 {
		res, err := db.Exec(`INSERT INTO users (email, name, role) VALUES (?, ?, ?)`, email, "User "+role, role)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	_, err := db.Exec(`INSERT INTO angkatan (kode, keterangan, is_aktif) VALUES ('AKA38', 'Angkatan 38', 1), ('AKA39', 'Angkatan 39', 1)`)
	require.NoError(t, err)
	f := &leadFixture{
		db:          db,
		querier:     querier,
		csID:        insertUser("cs@example.com", "cs"),
		keuanganID:  insertUser("keu@example.com", "keuangan"),
		adminKelas1: insertUser("ak1@example.com", "admin_kelas"),
		adminKelas2: insertUser("ak2@example.com", "admin_kelas"),
		clock:       time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC),
	}
	santri := NewSantriService(querier, NewKelasEngineService(querier))
	f.service = NewLeadService(querier, santri)
	f.service.now = func() time.Time { return f.clock }
	return f
}

const testChatID = "6281234567890@s.whatsapp.net"

func leadEvent(id, typ, occurred string, data any) models.LeadIngestEvent {
	raw, _ := json.Marshal(data)
	return models.LeadIngestEvent{
		EventID: id, Type: typ, OccurredAt: occurred, ChatID: testChatID,
		Phone: "6281234567890", WaName: "Fulanah", Data: raw,
	}
}

func (f *leadFixture) ingest(t *testing.T, events ...models.LeadIngestEvent) models.LeadIngestResult {
	t.Helper()
	res, err := f.service.IngestEvents(t.Context(), events)
	require.NoError(t, err)
	return res
}

func (f *leadFixture) lead(t *testing.T) queries.Lead {
	t.Helper()
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	return l
}

func TestLeadIngestCreatesLeadAndMessage(t *testing.T) {
	f := setupLeadFixture(t)
	res := f.ingest(t, leadEvent("evt_a1", "message.in", "2026-10-24T03:12:45Z",
		map[string]string{"wa_message_id": "3EB0A1", "body": "Assalamu'alaikum, berapa biaya kelas tahsin?"}))
	require.Equal(t, 1, res.Accepted)
	require.Empty(t, res.Rejected)

	l := f.lead(t)
	require.Equal(t, LeadStageBaru, l.Stage)
	require.Equal(t, LeadStatusOpen, l.Status)
	require.Equal(t, "6281234567890", l.Phone)
	require.Equal(t, "Fulanah", l.WaName)
	require.True(t, l.LastInboundAt.Valid)

	msgs, err := f.querier.ListLeadMessagesLatest(t.Context(), queries.ListLeadMessagesLatestParams{LeadID: l.ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "in", msgs[0].Direction)
	require.Equal(t, "lead", msgs[0].Sender)

	// Same event again: acknowledged as duplicate, nothing new is written.
	res = f.ingest(t, leadEvent("evt_a1", "message.in", "2026-10-24T03:12:45Z",
		map[string]string{"wa_message_id": "3EB0A1", "body": "Assalamu'alaikum, berapa biaya kelas tahsin?"}))
	require.Equal(t, 0, res.Accepted)
	require.Equal(t, 1, res.Duplicates)
	n, err := f.querier.CountLeadMessages(t.Context(), l.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	var events int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM lead_events`).Scan(&events))
	require.Equal(t, 1, events)
}

func TestLeadIngestFullPipelineToBuktiMasuk(t *testing.T) {
	f := setupLeadFixture(t)
	res := f.ingest(t,
		leadEvent("evt_a1", "message.in", "2026-10-24T03:12:45Z", map[string]string{"wa_message_id": "A1", "body": "Assalamu'alaikum"}),
		leadEvent("evt_a2", "message.out", "2026-10-24T03:13:02Z", map[string]string{"wa_message_id": "A2", "body": "Wa'alaikumussalam", "sender": "ai"}),
		leadEvent("evt_a3", "form.sent", "2026-10-24T03:20:10Z", map[string]string{}),
		leadEvent("evt_a3b", "message.in", "2026-10-24T03:25:00Z", map[string]string{"wa_message_id": "A3", "body": leadFormTemplateFilled}),
		leadEvent("evt_a4", "payment_info.sent", "2026-10-24T03:31:00Z", map[string]string{"paket_kode": "1x", "nominal_text": "Rp149.000"}),
		leadEvent("evt_a5", "proof.received", "2026-10-24T04:02:00Z", map[string]string{"media_type": "gambar"}),
	)
	require.Equal(t, 6, res.Accepted, res.Rejected)

	l := f.lead(t)
	require.Equal(t, LeadStageBuktiMasuk, l.Stage)
	require.Equal(t, int64(149000), l.PaymentAmount)
	require.Equal(t, "1x", l.PaketKode)
	require.Equal(t, "R 1X", l.KelasKode)
	require.Equal(t, "Fulanah binti Fulan", l.Nama)
	require.Equal(t, "P", l.JenisKelamin)
	require.Equal(t, int64(27), l.Usia.Int64)
	require.Equal(t, int64(0), l.NeedsReview)
	require.True(t, l.PaymentInfoSentAt.Valid)
	require.True(t, l.ProofReceivedAt.Valid)
	require.True(t, l.LastOutboundAt.Valid)

	// Keuangan is notified about the transfer proof.
	var notif int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = 'lead_bukti_transfer'`, f.keuanganID).Scan(&notif))
	require.Equal(t, 1, notif)
}

func TestLeadIngestNeverMovesStageBackward(t *testing.T) {
	f := setupLeadFixture(t)
	res := f.ingest(t,
		leadEvent("evt_1", "payment_info.sent", "2026-10-24T03:31:00Z", map[string]string{"paket_kode": "2x", "nominal_text": "Rp298.000"}),
		leadEvent("evt_2", "form.sent", "2026-10-24T03:40:00Z", map[string]string{}),
	)
	require.Equal(t, 2, res.Accepted)
	l := f.lead(t)
	require.Equal(t, LeadStageInfoBayarDikirim, l.Stage)
	require.Equal(t, "R 2X", l.KelasKode)

	var formEvents int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM lead_events WHERE type = 'form.sent'`).Scan(&formEvents))
	require.Equal(t, 1, formEvents, "event is still on the timeline")
}

func TestLeadIngestRejectsBadEventButKeepsOthers(t *testing.T) {
	f := setupLeadFixture(t)
	bad := leadEvent("evt_bad", "message.unknown", "2026-10-24T03:31:00Z", map[string]string{})
	noTime := leadEvent("evt_bad2", "message.in", "kemarin", map[string]string{})
	good := leadEvent("evt_ok", "message.in", "2026-10-24T03:31:00Z", map[string]string{"body": "halo"})
	res := f.ingest(t, bad, noTime, good)
	require.Equal(t, 1, res.Accepted)
	require.Len(t, res.Rejected, 2)
	require.Equal(t, "evt_bad", res.Rejected[0].EventID)
}

func TestLeadIngestFormUnreadableNeedsReview(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t,
		leadEvent("evt_1", "form.sent", "2026-10-24T03:00:00Z", map[string]string{}),
		leadEvent("evt_2", "message.in", "2026-10-24T03:05:00Z", map[string]string{"body": "Nama: Ahmad Usia: 30 Domisili: Depok"}),
	)
	l := f.lead(t)
	require.Equal(t, LeadStageFormDikirim, l.Stage)
	require.Equal(t, int64(1), l.NeedsReview)
	require.Empty(t, l.Nama)
}

func TestLeadIngestStopRequestCancelsFollowupsWithoutDNC(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("evt_1", "message.out", "2026-10-24T03:00:00Z", map[string]string{"body": "info kelas", "sender": "ai"}))
	l := f.lead(t)
	_, err := f.db.Exec(`INSERT INTO lead_followups (lead_id, step, scheduled_at, draft_text, status) VALUES (?, 'd1', ?, 'halo', 'approved')`, l.ID, f.clock)
	require.NoError(t, err)

	f.ingest(t, leadEvent("evt_2", "message.in", "2026-10-25T03:00:00Z", map[string]string{"body": "Stop ya kak"}))
	l = f.lead(t)
	require.Equal(t, "no", l.FollowupPermission)
	require.Equal(t, LeadStatusOpen, l.Status, "never auto do_not_contact")
	var status string
	require.NoError(t, f.db.QueryRow(`SELECT status FROM lead_followups WHERE lead_id = ?`, l.ID).Scan(&status))
	require.Equal(t, "cancelled", status)
	var optout int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM lead_events WHERE type = 'optout.detected'`).Scan(&optout))
	require.Equal(t, 1, optout)
}

func TestLeadIngestReopensLostButNotDoNotContact(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("evt_1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"body": "halo"}))
	l := f.lead(t)

	_, err := f.db.Exec(`UPDATE leads SET status = 'lost', lost_reason = 'harga' WHERE id = ?`, l.ID)
	require.NoError(t, err)
	f.ingest(t, leadEvent("evt_2", "message.in", "2026-10-26T03:00:00Z", map[string]string{"body": "kak masih ada kelas?"}))
	l = f.lead(t)
	require.Equal(t, LeadStatusOpen, l.Status)
	require.Empty(t, l.LostReason)

	_, err = f.db.Exec(`UPDATE leads SET status = 'do_not_contact' WHERE id = ?`, l.ID)
	require.NoError(t, err)
	f.ingest(t, leadEvent("evt_3", "message.in", "2026-10-27T03:00:00Z", map[string]string{"body": "halo lagi"}))
	l = f.lead(t)
	require.Equal(t, LeadStatusDoNotContact, l.Status)
	var payload string
	require.NoError(t, f.db.QueryRow(`SELECT payload FROM lead_events WHERE event_id = 'evt_3'`).Scan(&payload))
	require.Contains(t, payload, "contacted_again")
}

func TestLeadIngestFollowupSentMarksFollowup(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("evt_1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"body": "halo"}))
	l := f.lead(t)
	res, err := f.db.Exec(`INSERT INTO lead_followups (lead_id, step, scheduled_at, draft_text, status) VALUES (?, 'd1', ?, 'halo', 'approved')`, l.ID, f.clock)
	require.NoError(t, err)
	fid, _ := res.LastInsertId()

	f.ingest(t, leadEvent("evt_2", "message.out", "2026-10-25T03:00:00Z", map[string]any{"wa_message_id": "F1", "body": "halo", "sender": "followup", "reference_id": fid}))
	var status, waID string
	require.NoError(t, f.db.QueryRow(`SELECT status, wa_message_id FROM lead_followups WHERE id = ?`, fid).Scan(&status, &waID))
	require.Equal(t, "sent", status)
	require.Equal(t, "F1", waID)
}

func TestLeadIngestHandoffWithProofActsAsProof(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t,
		leadEvent("evt_1", "payment_info.sent", "2026-10-24T03:00:00Z", map[string]string{"paket_kode": "1x", "nominal_text": "Rp149.000"}),
		leadEvent("evt_2", "handoff", "2026-10-24T03:30:00Z", map[string]string{"reason": "Bukti transfer diterima", "summary": "lead kirim bukti"}),
	)
	l := f.lead(t)
	require.Equal(t, LeadStageBuktiMasuk, l.Stage)
	require.True(t, l.ProofReceivedAt.Valid)
}

func TestLeadIngestOutOfOrderKeepsLatestTimes(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t,
		leadEvent("evt_2", "message.in", "2026-10-24T05:00:00Z", map[string]string{"body": "kedua"}),
		leadEvent("evt_1", "message.in", "2026-10-24T04:00:00Z", map[string]string{"body": "pertama"}),
	)
	l := f.lead(t)
	require.Equal(t, time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC), l.LastInboundAt.Time.UTC())
	msgs, err := f.querier.ListLeadMessagesLatest(t.Context(), queries.ListLeadMessagesLatestParams{LeadID: l.ID, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, "kedua", msgs[0].Body, fmt.Sprintf("%+v", msgs))
}
