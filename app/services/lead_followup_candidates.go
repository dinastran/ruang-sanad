package services

import (
 "context"
 "fmt"
 "strconv"
 "time"

 "github.com/maulanashalihin/laju-go/app/models"
 "github.com/maulanashalihin/laju-go/app/queries"
)

const (
 pilotFollowupLabel = "uji-followup"
 pilotCandidateDefault = int64(20)
 pilotCandidateMax = int64(50)
)

// nextPilotFollowupStep chooses one due step per lead. Overdue pilot leads
// restart at D1 rather than receiving every historical step at once.
// Gaps between sends also prevent catch-up messages on consecutive days.
func nextPilotFollowupStep(r queries.DormantFollowupCandidate, now time.Time) (string, time.Time, bool) {
 step := FollowupStepD1
 earliest := r.Anchor.AddDate(0, 0, 1)
 if r.LastStep.Valid {
  switch r.LastStep.String {
  case FollowupStepD1:
   step = FollowupStepD3
   earliest = r.Anchor.AddDate(0, 0, 3)
   if r.LastSentAt.Valid && earliest.Before(r.LastSentAt.Time.AddDate(0, 0, 2)) {
    earliest = r.LastSentAt.Time.AddDate(0, 0, 2)
   }
  case FollowupStepD3:
   // D7 is optional: without a real program update, the agent can skip it.
   // Once D14 is due, use the permission request instead of waiting on D7.
   step = FollowupStepD7
   earliest = r.Anchor.AddDate(0, 0, 7)
   if r.LastSentAt.Valid && earliest.Before(r.LastSentAt.Time.AddDate(0, 0, 4)) {
    earliest = r.LastSentAt.Time.AddDate(0, 0, 4)
   }
   closeAt := r.Anchor.AddDate(0, 0, 14)
   if r.LastSentAt.Valid && closeAt.Before(r.LastSentAt.Time.AddDate(0, 0, 7)) {
    closeAt = r.LastSentAt.Time.AddDate(0, 0, 7)
   }
   if !now.Before(closeAt) {
    step, earliest = FollowupStepD14, closeAt
   }
  case FollowupStepD7:
   step = FollowupStepD14
   earliest = r.Anchor.AddDate(0, 0, 14)
   if r.LastSentAt.Valid && earliest.Before(r.LastSentAt.Time.AddDate(0, 0, 7)) {
    earliest = r.LastSentAt.Time.AddDate(0, 0, 7)
   }
  case FollowupStepD14:
   if r.Permission != "yes" { return "", time.Time{}, false }
   step = FollowupStepD30
   earliest = r.Anchor.AddDate(0, 0, 30)
   if r.LastSentAt.Valid && earliest.Before(r.LastSentAt.Time.AddDate(0, 0, 16)) {
    earliest = r.LastSentAt.Time.AddDate(0, 0, 16)
   }
  default:
   return "", time.Time{}, false
  }
 }
 return step, earliest, !now.Before(earliest)
}

// pilotSuggestedTime is a future slot for human approval. This does not
// promise a send; the existing worker re-checks the configured time window.
func pilotSuggestedTime(now time.Time) string {
 at := now.In(wib).Add(10 * time.Minute)
 dayStart, _ := wibDayBounds(at)
 tenAM := dayStart.Add(10 * time.Hour)
 if at.Before(tenAM) {
  at = tenAM
 } else if at.Hour() >= 20 {
  at = dayStart.AddDate(0, 0, 1).Add(10 * time.Hour)
 }
 return at.Format("2006-01-02 15:04")
}

// toolDaftarKandidatFollowup is read-only. Hermes still uses detail_lead
// and the unchanged usulkan_followup tool for reviewable proposals.
func (s *AIAgentService) toolDaftarKandidatFollowup(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
 limit, err := a.int("limit", pilotCandidateDefault, 1, pilotCandidateMax)
 if err != nil { return nil, err }
 var beforeID int64
 if cursor := a.str("cursor"); cursor != "" {
  beforeID, err = strconv.ParseInt(cursor, 10, 64)
  if err != nil || beforeID <= 0 {
   return nil, aiArgErr("cursor tidak valid; gunakan cursor_berikut dari hasil tool")
  }
 }
 now := s.leads.nowUTC()
 dayStart, dayEnd := wibDayBounds(now)
 rows, err := s.querier.ListPilotDormantFollowups(ctx, beforeID, now, dayStart, dayEnd, limit + 1)
 if err != nil { return nil, err }
 next := ""
 if int64(len(rows)) > limit {
  rows = rows[:limit]
  next = strconv.FormatInt(rows[len(rows)-1].ID, 10)
 }
 items := make([]map[string]any, 0, len(rows))
 for _, r := range rows {
  step, earliest, eligible := nextPilotFollowupStep(r, now)
  if !eligible { continue }
  guidance := "Tinjau konteks; satu CTA, tanpa harga/rekening/link. Usulkan hanya bila relevan."
  if step == FollowupStepD7 || step == FollowupStepD30 {
   guidance = "Wajib ada update program yang nyata dan terverifikasi dalam reason; tanpa update, jangan usulkan."
  } else if step == FollowupStepD14 {
   guidance = "Pesan penutupan yang sopan dan wajib memberi opsi balas 'boleh kabari'."
  }
  items = append(items, map[string]any{
   "lead_ref": aiLeadRef(r.ID),
   "nama_depan": aiFirstName(r.Nama, r.WaName),
   "tahap": r.Stage,
   "step_disarankan": step,
   "hari_sejak_pesan_terakhir_cs": int(now.Sub(r.Anchor).Hours() / 24),
   "pesan_terakhir_cs": aiWIB(r.Anchor),
   "sudah_boleh_sejak": aiWIB(earliest),
   "jadwal_disarankan": pilotSuggestedTime(now),
   "panduan": guidance,
   "link": s.leadLink(p, r.ID),
  })
 }
 return map[string]any{
  "data_per": aiWIB(now), "label_pilot": pilotFollowupLabel,
  "jumlah": len(items), "kandidat": items, "cursor_berikut": next,
  "catatan": fmt.Sprintf("Hanya label %s. Baca detail_lead dahulu, lalu usulkan_followup dengan izin tulis. Bukan pengiriman otomatis; CS tetap wajib approve. Jangan melakukan follow-up jika tidak relevan atau tidak ada update nyata untuk D7/D30.", pilotFollowupLabel),
 }, nil
}
