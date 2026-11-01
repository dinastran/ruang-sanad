package services

import (
 "context"
 "errors"
 "fmt"
 "strings"
 "time"

 "github.com/maulanashalihin/laju-go/app/models"
 "github.com/maulanashalihin/laju-go/app/queries"
)

// dashboardRange requires a valid inclusive WIB interval, with a 7-day default.
// Unlike the legacy report, malformed inputs are not silently replaced.
func (s *LeadService) dashboardRange(from, to string) (time.Time, time.Time, string, string, error) {
 today := s.now().In(wib)
 endDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, wib)
 startDay := endDay.AddDate(0, 0, -6)
 if from != "" || to != "" {
  var ok bool
  if from == "" || to == "" { return time.Time{}, time.Time{}, "", "", errors.New("tanggal mulai dan akhir wajib diisi") }
  startDay, ok = wibDayStart(from)
  if !ok { return time.Time{}, time.Time{}, "", "", errors.New("tanggal mulai tidak valid") }
  endDay, ok = wibDayStart(to)
  if !ok { return time.Time{}, time.Time{}, "", "", errors.New("tanggal akhir tidak valid") }
 }
 exclusiveEnd := endDay.AddDate(0, 0, 1)
 if startDay.After(endDay) || exclusiveEnd.After(startDay.AddDate(0, 0, 366)) {
  return time.Time{}, time.Time{}, "", "", errors.New("rentang tanggal harus 1 sampai 366 hari")
 }
 return startDay, exclusiveEnd, startDay.Format(time.DateOnly), endDay.Format(time.DateOnly), nil
}

// Dashboard combines cohort metrics with event-date closings and the current CS queue.
func (s *LeadService) Dashboard(ctx context.Context, from, to string) (*models.LeadDashboard, error) {
 start, end, fromStr, toStr, err := s.dashboardRange(from, to)
 if err != nil { return nil, err }
 report, err := s.Report(ctx, fromStr, toStr)
 if err != nil { return nil, err }
 created, err := s.querier.ListLeadReportRows(ctx, start, end)
 if err != nil { return nil, err }
 closed, err := s.querier.ListLeadDashboardClosings(ctx, start, end)
 if err != nil { return nil, err }
 days := int(end.Sub(start).Hours() / 24)
 prevStart := start.AddDate(0, 0, -days)
 prevLeads, err := s.querier.LeadDashboardCreatedCount(ctx, prevStart, start)
 if err != nil { return nil, err }
 prevClosed, err := s.querier.ListLeadDashboardClosings(ctx, prevStart, start)
 if err != nil { return nil, err }
 reviews, err := s.querier.LeadDashboardReviewCount(ctx, start, end)
 if err != nil { return nil, err }
 dailyMap := map[string]*models.LeadDashboardDay{}
 daily := make([]models.LeadDashboardDay, 0, days)
 for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
  key := d.Format(time.DateOnly)
  dailyMap[key] = &models.LeadDashboardDay{Day:key}
 }
 for _, l := range created { dailyMap[l.CreatedAt.In(wib).Format(time.DateOnly)].Leads++ }
 for _, c := range closed { dailyMap[c.In(wib).Format(time.DateOnly)].Closing++ }
 for d := start; d.Before(end); d = d.AddDate(0, 0, 1) { daily = append(daily, *dailyMap[d.Format(time.DateOnly)]) }
 queue, err := s.querier.LeadDashboardQueue(ctx)
 if err != nil { return nil, err }
 priorities, err := s.querier.LeadDashboardPriorities(ctx)
 if err != nil { return nil, err }
 eligible, replied, err := s.querier.LeadDashboardMaturedFollowups(ctx, start, end, s.nowUTC().Add(-72*time.Hour))
 if err != nil { return nil, err }
 cohort, err := s.querier.LeadSourceCohort(ctx, queries.LeadSourceCohortParams{CreatedAt: start.UTC(), CreatedAt_2: end.UTC()})
 if err != nil { return nil, err }
 sources, err := s.sourceStats(ctx, cohort)
 if err != nil { return nil, err }
 res := &models.LeadDashboard{
  Sources:sources,
  From:fromStr, To:toStr, Report:report, ClosingPeriod:int64(len(closed)),
  PreviousLeads:prevLeads, PreviousClosing:int64(len(prevClosed)), ReviewCount:reviews,
  Daily:daily, Priorities:make([]models.LeadDashboardPriority, 0, len(priorities)),
  FollowupEligible:eligible, FollowupReplied:replied,
  Insights:make([]string, 0, 5),
 }
 for _, item := range queue {
  switch item.Status {
  case "pending_approval": res.FollowupPending = item.Count
  case "approved": res.FollowupApproved = item.Count
  case "failed": res.FollowupFailed = item.Count
  }
 }
 for _, p := range priorities {
  reason := dashboardPriorityReason(p)
  activity := p.CreatedAt
  if p.LastInbound.Valid { activity = p.LastInbound.Time }
  res.Priorities = append(res.Priorities, models.LeadDashboardPriority{
   ID:p.ID, Name:p.Nama, Stage:p.Stage, Reason:reason, ActivityAt:activity.In(wib).Format(time.RFC3339),
  })
 }
 res.Insights = append(res.Insights, fmt.Sprintf("%d lead masuk pada periode ini, dibanding %d pada periode sebelumnya.", report.TotalLeads, prevLeads))
 res.Insights = append(res.Insights, fmt.Sprintf("%d closing terjadi pada periode ini, dibanding %d pada periode sebelumnya.", len(closed), len(prevClosed)))
 if len(report.Funnel) > 1 {
  var largest int64
  var label string
  for i := 1; i < len(report.Funnel); i++ {
   loss := report.Funnel[i-1].Reached - report.Funnel[i].Reached
   if loss > largest { largest = loss; label = report.Funnel[i].Stage }
  }
  if largest > 0 { res.Insights = append(res.Insights, fmt.Sprintf("Penurunan pipeline terbesar menuju tahap %s: %d lead. Ini estimasi dari tahap terkini.", label, largest)) }
 }
 if len(report.LostReasons) > 0 { res.Insights = append(res.Insights, fmt.Sprintf("Alasan lost paling banyak pada cohort ini: %s (%d lead).", report.LostReasons[0].Key, report.LostReasons[0].Count)) }
 if reviews > 0 { res.Insights = append(res.Insights, fmt.Sprintf("%d lead yang masuk periode ini masih memerlukan peninjauan.", reviews)) }
 return res, nil
}

func dashboardPriorityReason(p queries.LeadDashboardPriorityRow) string {
 switch {
 case p.Stage == LeadStageBuktiMasuk: return "Bukti masuk, perlu verifikasi"
 case p.Stage == LeadStageTerverifikasi: return "Terverifikasi, perlu dikonversi"
 case p.NeedsReview: return "Data perlu ditinjau"
 default: return "Pesan masuk belum dibalas"
 }
}

// PriorityPage is an operational queue across all dates, not a cohort report.
func (s *LeadService) PriorityPage(ctx context.Context, search, reason string, page int) (*models.LeadPriorityPage, error) {
 const size = 30
 search = strings.ToLower(strings.TrimSpace(search))
 chars := []rune(search)
 if len(chars) > 80 { search = string(chars[:80]) }
 switch reason {
 case "verify", "convert", "review", "reply":
 default: reason = ""
 }
 if page < 1 { page = 1 }
 if page > 100000 { page = 100000 }
 rows, total, err := s.querier.LeadDashboardPriorityPage(ctx, search, reason, size, (page-1)*size)
 if err != nil { return nil, err }
 out := &models.LeadPriorityPage{
  Items:make([]models.LeadDashboardPriority,0,len(rows)), Total:total,
  Page:page, PerPage:size, Query:search, Reason:reason,
 }
 for _, p := range rows {
  activity := p.CreatedAt
  if p.LastInbound.Valid { activity = p.LastInbound.Time }
  out.Items = append(out.Items, models.LeadDashboardPriority{
   ID:p.ID, Name:p.Nama, Stage:p.Stage, Reason:dashboardPriorityReason(p),
   ActivityAt:activity.In(wib).Format(time.RFC3339),
  })
 }
 return out,nil
}
