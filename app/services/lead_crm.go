package services

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	leadListPageSize     = 30
	leadMessagesPageSize = 50
	leadTimelineSize     = 100
)

var leadUntukSiapa = map[string]bool{"": true, "diri_sendiri": true, "anak": true, "keluarga": true}

func userActor(userID int64) string { return "user:" + strconv.FormatInt(userID, 10) }

func formatLeadTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(wib).Format(time.RFC3339)
}

func formatLeadNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return formatLeadTime(t.Time)
}

func labelResponses(labels []queries.LeadLabel) []models.LeadLabelResponse {
	out := make([]models.LeadLabelResponse, 0, len(labels))
	for _, l := range labels {
		out = append(out, models.LeadLabelResponse{ID: l.ID, Nama: l.Nama, Warna: l.Warna})
	}
	return out
}

// --- Daftar ---

func encodeLeadCursor(r queries.LeadListRow) string {
	raw := fmt.Sprintf("%d|%d|%s", r.Prio, r.ID, r.ActivityRaw)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeLeadCursor(cursor string, f *queries.LeadListFilter) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return
	}
	parts := strings.SplitN(string(raw), "|", 3)
	if len(parts) != 3 {
		return
	}
	prio, err1 := strconv.ParseInt(parts[0], 10, 64)
	id, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return
	}
	f.AfterPrio, f.AfterID, f.AfterAct, f.HasCursor = prio, id, parts[2], true
}

// wibDayStart parses YYYY-MM-DD as the start of that day in WIB.
func wibDayStart(date string) (time.Time, bool) {
	t, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(date), wib)
	return t, err == nil
}

func (s *LeadService) List(ctx context.Context, p models.LeadListParams) (*models.LeadListResponse, error) {
	f := queries.LeadListFilter{Limit: leadListPageSize + 1}
	if p.DateField == "converted" {
		f.DateField = "converted"
	}
	switch p.Status {
	case "", LeadStatusOpen:
		f.Status = LeadStatusOpen
	case "all":
	case LeadStatusWon, LeadStatusLost, LeadStatusDoNotContact:
		f.Status = p.Status
	}
	if isValidLeadStage(p.Stage) {
		f.Stage = p.Stage
	}
	if isValidLeadStage(p.ReachedStage) {
		f.ReachedStage = p.ReachedStage
	}
	for _, reason := range LeadLostReasons {
		if p.LostReason == reason {
			f.LostReason = reason
			break
		}
	}
	f.LabelID = p.LabelID
	if p.Mine {
		f.AssignedTo = p.UserID
	}
	f.NeedsReview = p.NeedsReview
	if q := strings.TrimSpace(p.Search); q != "" {
		f.Search = truncateRunes(q, 80)
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, q)
		if len(digits) >= 4 {
			f.PhoneSearch = strings.TrimPrefix(digits, "0")
		}
	}
	if t, ok := wibDayStart(p.From); ok {
		f.CreatedFrom = t
	}
	if t, ok := wibDayStart(p.To); ok {
		f.CreatedTo = t.AddDate(0, 0, 1)
	}
	if p.Cursor != "" {
		decodeLeadCursor(p.Cursor, &f)
	}
	rows, err := s.querier.ListLeads(ctx, f)
	if err != nil {
		return nil, err
	}
	resp := &models.LeadListResponse{Items: make([]models.LeadListItem, 0, len(rows))}
	if len(rows) > leadListPageSize {
		rows = rows[:leadListPageSize]
		resp.NextCursor = encodeLeadCursor(rows[len(rows)-1])
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	labels, err := s.querier.ListLabelsForLeads(ctx, ids)
	if err != nil {
		return nil, err
	}
	pauses, err := s.leadPauseStates(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		item := models.LeadListItem{
			ID: r.ID, Nama: r.Nama, WaName: r.WaName, Phone: r.Phone, Stage: r.Stage, Status: r.Status,
			PaketKode: r.PaketKode, NeedsReview: r.NeedsReview == 1, AIPaused: pauses[r.ID].paused,
			AssignedUserID: r.AssignedUserID.Int64, AssignedName: r.AssignedName,
			LastInboundAt: formatLeadNullTime(r.LastInboundAt), Labels: labelResponses(labels[r.ID]),
		}
		if t, ok := queries.ParseStoredTime(r.ActivityRaw); ok {
			item.LastActivityAt = formatLeadTime(t)
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

func (s *LeadService) Labels(ctx context.Context) ([]models.LeadLabelResponse, error) {
	labels, err := s.querier.ListLeadLabels(ctx)
	if err != nil {
		return nil, err
	}
	return labelResponses(labels), nil
}

// AssignableUsers lists the users a lead can be assigned to.
func (s *LeadService) AssignableUsers(ctx context.Context) ([]models.LeadUserOption, error) {
	rows, err := s.querier.ListUsersByRoles(ctx, []string{"cs", "keuangan", "super_admin"})
	if err != nil {
		return nil, err
	}
	out := make([]models.LeadUserOption, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.LeadUserOption{ID: r.ID, Name: r.Name, Role: r.Role})
	}
	return out, nil
}

// --- Detail ---

func messageResponses(msgs []queries.LeadMessage) []models.LeadMessageResponse {
	out := make([]models.LeadMessageResponse, 0, len(msgs))
	// Queries return newest first; the chat view reads oldest first.
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		out = append(out, models.LeadMessageResponse{
			ID: m.ID, Direction: m.Direction, Sender: m.Sender, Body: m.Body,
			MediaType: m.MediaType, SentAt: formatLeadTime(m.SentAt),
		})
	}
	return out
}

func (s *LeadService) actorNames(ctx context.Context, events []queries.LeadEvent) map[string]string {
	names := map[string]string{}
	for _, e := range events {
		if _, done := names[e.Actor]; done || !strings.HasPrefix(e.Actor, "user:") {
			continue
		}
		id, err := strconv.ParseInt(strings.TrimPrefix(e.Actor, "user:"), 10, 64)
		if err != nil {
			continue
		}
		if u, err := s.querier.GetUserByID(ctx, id); err == nil {
			names[e.Actor] = u.Name
		} else {
			names[e.Actor] = "User dihapus"
		}
	}
	return names
}

func (s *LeadService) Detail(ctx context.Context, id int64) (*models.LeadDetail, error) {
	l, err := loadLead(ctx, s.querier, id)
	if err != nil {
		return nil, err
	}
	d := &models.LeadDetail{
		ID: l.ID, WaChatID: l.WaChatID, Phone: l.Phone, WaName: l.WaName, Nama: l.Nama, Usia: l.Usia.Int64,
		Domisili: l.Domisili, JenisKelamin: l.JenisKelamin, PaketKode: l.PaketKode, KelasKode: l.KelasKode,
		WaktuBelajar: l.WaktuBelajar, RiwayatBelajar: l.RiwayatBelajar, BelajarDimana: l.BelajarDimana,
		UntukSiapa: l.UntukSiapa, Stage: l.Stage, Status: l.Status, LostReason: l.LostReason,
		FollowupPermission: l.FollowupPermission, AssignedUserID: l.AssignedUserID.Int64,
		PaymentAmount: l.PaymentAmount, PaymentInfoSentAt: formatLeadNullTime(l.PaymentInfoSentAt),
		ProofReceivedAt: formatLeadNullTime(l.ProofReceivedAt), VerifiedAt: formatLeadNullTime(l.VerifiedAt),
		ConvertedAt: formatLeadNullTime(l.ConvertedAt), SantriID: l.SantriID.Int64, NeedsReview: l.NeedsReview == 1,
		LastInboundAt: formatLeadNullTime(l.LastInboundAt), LastOutboundAt: formatLeadNullTime(l.LastOutboundAt),
		CreatedAt: formatLeadTime(l.CreatedAt), ReviewFields: []string{},
	}
	if l.AssignedUserID.Valid {
		if u, err := s.querier.GetUserByID(ctx, l.AssignedUserID.Int64); err == nil {
			d.AssignedName = u.Name
		}
	}
	labels, err := s.querier.ListLabelsForLead(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Labels = labelResponses(labels)

	msgs, err := s.querier.ListLeadMessagesLatest(ctx, queries.ListLeadMessagesLatestParams{LeadID: id, Limit: leadMessagesPageSize + 1})
	if err != nil {
		return nil, err
	}
	if len(msgs) > leadMessagesPageSize {
		msgs = msgs[:leadMessagesPageSize]
		d.HasOlderMessages = true
	}
	d.Messages = messageResponses(msgs)

	events, err := s.querier.ListLeadEvents(ctx, queries.ListLeadEventsParams{LeadID: id, Limit: leadTimelineSize})
	if err != nil {
		return nil, err
	}
	names := s.actorNames(ctx, events)
	d.Events = make([]models.LeadEventResponse, 0, len(events))
	pauses, err := s.leadPauseStates(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	d.AIPaused, d.AIPausedUntil = pauses[id].paused, pauses[id].until
	seenForm, seenVerify, seenDNC := false, false, false
	for _, e := range events {
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(e.Payload), &payload)
		d.Events = append(d.Events, models.LeadEventResponse{
			ID: e.ID, Type: e.Type, Actor: e.Actor, ActorName: names[e.Actor], Payload: payload, CreatedAt: formatLeadTime(e.CreatedAt),
		})
		switch {
		case !seenForm && e.Type == "form.parsed":
			seenForm = true
			if fields, ok := payload["review_fields"].([]any); ok && l.NeedsReview == 1 {
				for _, f := range fields {
					if name, ok := f.(string); ok {
						d.ReviewFields = append(d.ReviewFields, name)
					}
				}
			}
		case !seenVerify && e.Type == "payment.verified":
			seenVerify = true
			if v, ok := payload["amount"].(float64); ok {
				d.VerifiedAmount = int64(v)
			}
			d.VerifiedDate, _ = payload["date"].(string)
		}
		// "Menghubungi lagi": a message arrived after the lead was marked do_not_contact.
		if l.Status == LeadStatusDoNotContact && !seenDNC {
			if e.Type == "lead.do_not_contact" {
				seenDNC = true
			} else if again, _ := payload["contacted_again"].(bool); again {
				d.ContactedAgain = true
			}
		}
	}
	if d.VerifiedAmount == 0 {
		d.VerifiedAmount = l.PaymentAmount
	}

	if a, err := s.querier.GetLatestLeadAnalysis(ctx, id); err == nil {
		persona := map[string]any{}
		_ = json.Unmarshal([]byte(a.Persona), &persona)
		d.Analysis = &models.LeadAnalysisResponse{
			ID: a.ID, Summary: a.Summary, Persona: persona, InterestScore: a.InterestScore, MainObjection: a.MainObjection,
			SuggestedStage: a.SuggestedStage, SuggestedNextAction: a.SuggestedNextAction, Confidence: a.Confidence,
			CreatedAt: formatLeadTime(a.CreatedAt),
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	followups, err := s.querier.ListLeadFollowups(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Followups = make([]models.LeadFollowupResponse, 0, len(followups))
	for _, f := range followups {
		d.Followups = append(d.Followups, models.LeadFollowupResponse{
			ID: f.ID, Step: f.Step, ScheduledAt: formatLeadTime(f.ScheduledAt), DraftText: f.DraftText, Reason: f.Reason,
			Status: f.Status, SentAt: formatLeadNullTime(f.SentAt), Error: f.Error, CreatedByKind: f.CreatedByKind,
			Warnings: FollowupDraftWarnings(f.Step, f.DraftText, f.Reason, l.FollowupPermission),
		})
	}
	settings, err := s.querier.GetCRMSettings(ctx)
	if err != nil {
		return nil, err
	}
	d.ConfirmationReady = strings.TrimSpace(settings.PaymentVerifiedTemplate) != ""
	return d, nil
}

// OlderMessages loads the page of messages before message beforeID.
func (s *LeadService) OlderMessages(ctx context.Context, leadID, beforeID int64) ([]models.LeadMessageResponse, bool, error) {
	ref, err := s.querier.GetLeadMessageByID(ctx, queries.GetLeadMessageByIDParams{ID: beforeID, LeadID: leadID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, ErrLeadNotFound
	}
	if err != nil {
		return nil, false, err
	}
	msgs, err := s.querier.ListLeadMessagesBefore(ctx, queries.ListLeadMessagesBeforeParams{
		LeadID: leadID, BeforeSentAt: ref.SentAt, BeforeID: ref.ID, RowLimit: leadMessagesPageSize + 1,
	})
	if err != nil {
		return nil, false, err
	}
	more := len(msgs) > leadMessagesPageSize
	if more {
		msgs = msgs[:leadMessagesPageSize]
	}
	return messageResponses(msgs), more, nil
}

// --- Aksi manual ---

// mutateLead loads a lead in a transaction, applies fn, saves it and writes
// the timeline event fn returns (type "" = no event).
func (s *LeadService) mutateLead(ctx context.Context, id, actorID int64, fn func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error)) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, id)
		if err != nil {
			return err
		}
		now := s.nowUTC()
		typ, payload, err := fn(q, &l, now)
		if err != nil {
			return err
		}
		if err := saveLead(ctx, q, l, now); err != nil {
			return err
		}
		if typ == "" {
			return nil
		}
		return writeLeadEvent(ctx, q, l.ID, newLeadEventID("usr"), typ, userActor(actorID), payload, now)
	})
}

func (s *LeadService) UpdateData(ctx context.Context, id, actorID int64, req models.LeadUpdateRequest) error {
	return s.mutateLead(ctx, id, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		if l.Status == LeadStatusWon {
			return "", nil, ErrLeadReadOnly
		}
		jk := strings.ToUpper(strings.TrimSpace(req.JenisKelamin))
		if jk != "" && jk != "L" && jk != "P" {
			return "", nil, errors.New("jenis kelamin harus L atau P")
		}
		if req.Usia != 0 && (req.Usia < 5 || req.Usia > 99) {
			return "", nil, errors.New("usia harus 5–99")
		}
		untuk := strings.TrimSpace(req.UntukSiapa)
		if !leadUntukSiapa[untuk] {
			return "", nil, errors.New("pilihan 'untuk siapa' tidak valid")
		}
		paket := strings.ToLower(strings.TrimSpace(req.PaketKode))
		if paket != "" && paket != "1x" && paket != "2x" {
			return "", nil, errors.New("paket harus 1x atau 2x")
		}
		kelasKode := strings.TrimSpace(req.KelasKode)
		if kelasKode != "" {
			if _, err := q.GetKodeKelasByKode(ctx, kelasKode); err != nil {
				return "", nil, fmt.Errorf("kode kelas %q tidak terdaftar", kelasKode)
			}
		}
		before := *l
		l.Nama = truncateRunes(req.Nama, 200)
		l.Usia = sql.NullInt64{Int64: req.Usia, Valid: req.Usia > 0}
		l.Domisili = truncateRunes(req.Domisili, 200)
		l.JenisKelamin = jk
		l.PaketKode = paket
		l.KelasKode = kelasKode
		l.WaktuBelajar = truncateRunes(req.WaktuBelajar, 200)
		l.RiwayatBelajar = truncateRunes(req.RiwayatBelajar, 200)
		l.BelajarDimana = truncateRunes(req.BelajarDimana, 200)
		l.UntukSiapa = untuk
		if req.MarkReviewed {
			l.NeedsReview = 0
		}
		// Only field names go to the timeline, never the (personal) values.
		changed := []string{}
		check := func(name string, a, b any) {
			if a != b {
				changed = append(changed, name)
			}
		}
		check("nama", before.Nama, l.Nama)
		check("usia", before.Usia, l.Usia)
		check("domisili", before.Domisili, l.Domisili)
		check("jenis_kelamin", before.JenisKelamin, l.JenisKelamin)
		check("paket_kode", before.PaketKode, l.PaketKode)
		check("kelas_kode", before.KelasKode, l.KelasKode)
		check("waktu_belajar", before.WaktuBelajar, l.WaktuBelajar)
		check("riwayat_belajar", before.RiwayatBelajar, l.RiwayatBelajar)
		check("belajar_dimana", before.BelajarDimana, l.BelajarDimana)
		check("untuk_siapa", before.UntukSiapa, l.UntukSiapa)
		if len(changed) == 0 && before.NeedsReview == l.NeedsReview {
			return "", nil, nil
		}
		return "lead.updated", map[string]any{"fields": changed, "reviewed": req.MarkReviewed}, nil
	})
}

// ChangeStage is the manual stage move (PRD 6.2 rule 2): forward or back,
// but never into terverifikasi, which only the verify action reaches.
func (s *LeadService) ChangeStage(ctx context.Context, id, actorID int64, stage string, eventType string) error {
	if !isValidLeadStage(stage) {
		return ErrLeadInvalidStage
	}
	if stage == LeadStageTerverifikasi {
		return errors.New("tahap Terverifikasi hanya bisa dicapai lewat Verifikasi pembayaran")
	}
	if eventType == "" {
		eventType = "stage.changed"
	}
	return s.mutateLead(ctx, id, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		if l.Status != LeadStatusOpen {
			return "", nil, errors.New("tahap hanya bisa diubah untuk lead yang masih berjalan")
		}
		if l.Stage == stage {
			return "", nil, nil
		}
		from := l.Stage
		l.Stage = stage
		if from == LeadStageTerverifikasi {
			// Moving back out of terverifikasi voids the verification.
			l.VerifiedAt, l.VerifiedBy = sql.NullTime{}, sql.NullInt64{}
		}
		return eventType, map[string]any{"stage_from": from, "stage_to": stage}, nil
	})
}

func isValidLostReason(reason string) bool {
	for _, r := range LeadLostReasons {
		if r == reason {
			return true
		}
	}
	return false
}

// ChangeStatus marks a lead lost / do_not_contact, or reopens it.
func (s *LeadService) ChangeStatus(ctx context.Context, id, actorID int64, req models.LeadStatusRequest) error {
	return s.mutateLead(ctx, id, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		from := l.Status
		payload := map[string]any{"status_from": from, "status_to": req.Status}
		switch req.Status {
		case LeadStatusLost:
			if from != LeadStatusOpen {
				return "", nil, errors.New("hanya lead yang masih berjalan yang bisa ditandai hilang")
			}
			if !isValidLostReason(req.LostReason) {
				return "", nil, errors.New("alasan hilang wajib dipilih")
			}
			l.Status, l.LostReason = LeadStatusLost, req.LostReason
			payload["reason"] = req.LostReason
			if _, err := cancelActiveFollowups(ctx, q, l.ID, "lead ditandai hilang", now); err != nil {
				return "", nil, err
			}
			return "lead.lost", payload, nil
		case LeadStatusDoNotContact:
			if from != LeadStatusOpen && from != LeadStatusLost {
				return "", nil, errors.New("status lead ini tidak bisa diubah ke Jangan hubungi")
			}
			l.Status, l.FollowupPermission = LeadStatusDoNotContact, "no"
			if _, err := cancelActiveFollowups(ctx, q, l.ID, "jangan hubungi", now); err != nil {
				return "", nil, err
			}
			return "lead.do_not_contact", payload, nil
		case LeadStatusOpen:
			if from != LeadStatusLost && from != LeadStatusDoNotContact {
				return "", nil, errors.New("hanya lead hilang atau jangan-hubungi yang bisa dibuka kembali")
			}
			l.Status, l.LostReason = LeadStatusOpen, ""
			return "lead.reopened", payload, nil
		}
		return "", nil, errors.New("status tidak valid")
	})
}

// Assign sets the responsible user. A non-super-admin can only take a lead
// for themselves; userID 0 unassigns (super_admin only).
func (s *LeadService) Assign(ctx context.Context, id, actorID int64, actorRole string, userID int64) error {
	if actorRole != "super_admin" && userID != actorID {
		return errors.New("hanya super admin yang bisa menugaskan lead ke orang lain")
	}
	return s.mutateLead(ctx, id, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		if userID != 0 {
			u, err := q.GetUserByID(ctx, userID)
			if err != nil {
				return "", nil, errors.New("user tidak ditemukan")
			}
			if r := string(u.Role); r != "cs" && r != "keuangan" && r != "super_admin" {
				return "", nil, errors.New("lead hanya bisa ditugaskan ke CS, Keuangan, atau super admin")
			}
		}
		if l.AssignedUserID.Int64 == userID {
			return "", nil, nil
		}
		from := l.AssignedUserID.Int64
		l.AssignedUserID = sql.NullInt64{Int64: userID, Valid: userID != 0}
		return "lead.assigned", map[string]any{"from_user_id": from, "to_user_id": userID}, nil
	})
}

func (s *LeadService) SetLabel(ctx context.Context, id, actorID, labelID int64, add bool) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		if _, err := loadLead(ctx, q, id); err != nil {
			return err
		}
		label, err := q.GetLeadLabelByID(ctx, labelID)
		if err != nil {
			return errors.New("label tidak ditemukan")
		}
		now := s.nowUTC()
		var n int64
		typ := "label.added"
		if add {
			n, err = q.AddLeadLabelLink(ctx, queries.AddLeadLabelLinkParams{LeadID: id, LabelID: labelID, CreatedBy: sql.NullInt64{Int64: actorID, Valid: true}, CreatedAt: now})
		} else {
			typ = "label.removed"
			n, err = q.RemoveLeadLabelLink(ctx, queries.RemoveLeadLabelLinkParams{LeadID: id, LabelID: labelID})
		}
		if err != nil || n == 0 {
			return err
		}
		return writeLeadEvent(ctx, q, id, newLeadEventID("usr"), typ, userActor(actorID), map[string]any{"label": label.Nama}, now)
	})
}

// Delete permanently removes a lead and everything attached to it (PRD 11.2
// right to erasure). Only who/which id/when is logged, never lead data.
func (s *LeadService) Delete(ctx context.Context, id, actorID int64) error {
	n, err := s.querier.DeleteLead(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLeadNotFound
	}
	slog.Info("lead dihapus", "lead_id", id, "oleh_user_id", actorID, "waktu", s.nowUTC().Format(time.RFC3339))
	return nil
}
