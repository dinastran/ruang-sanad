package services

import (
 "fmt"
 "testing"
 "time"

 "github.com/maulanashalihin/laju-go/app/models"
 "github.com/stretchr/testify/require"
)

func TestDashboardRangeWIBAndValidation(t *testing.T) {
 f := setupLeadFixture(t)
 dash := f.service
 dash.now = func() time.Time { return time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC) }
 start, end, from, to, err := dash.dashboardRange("", "")
 require.NoError(t, err)
 require.Equal(t, "2026-10-19", from)
 require.Equal(t, "2026-10-25", to)
 require.Equal(t, 7*24*time.Hour, end.Sub(start))
 for _, args := range [][2]string{
  {"2026-10-25", "2026-10-24"}, {"2026-10-25", ""},
  {"2026-10-xyz", "2026-10-25"}, {"2025-01-01", "2026-10-25"},
 } {
  _, _, _, _, err := dash.dashboardRange(args[0], args[1])
  require.Error(t, err, args)
 }
}

func TestLeadDashboardClosingUsesConversionDateAndDrilldown(t *testing.T) {
 f := setupLeadFixture(t)
 f.service.now = func() time.Time { return time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC) }
 seed := func(chat, created, converted, stage, status string, review int64) {
  t.Helper()
  var convertedValue any
  if converted != "" { v, err := time.Parse(time.RFC3339, converted); require.NoError(t, err); convertedValue = v }
  createdValue, err := time.Parse(time.RFC3339, created)
  require.NoError(t, err)
  _, err = f.db.Exec(`INSERT INTO leads (wa_chat_id, created_at, updated_at, converted_at, stage, status, needs_review) VALUES (?, ?, ?, ?, ?, ?, ?)`,
   chat, createdValue, createdValue, convertedValue, stage, status, review)
  require.NoError(t, err)
 }
 seed("older", "2026-10-10T02:00:00Z", "2026-10-25T03:00:00Z", "terverifikasi", "won", 0)
 seed("this-cohort-won", "2026-10-24T02:00:00Z", "2026-10-25T05:00:00Z", "terverifikasi", "won", 0)
 seed("this-cohort-open", "2026-10-25T02:00:00Z", "", "baru", "open", 1)
 seed("previous", "2026-10-17T02:00:00Z", "", "baru", "lost", 0)

 d, err := f.service.Dashboard(t.Context(), "2026-10-19", "2026-10-25")
 require.NoError(t, err)
 require.Equal(t, int64(2), d.Report.TotalLeads)
 require.Equal(t, int64(1), d.Report.Won)
 require.Equal(t, int64(2), d.ClosingPeriod)
 require.Equal(t, int64(1), d.PreviousLeads)
 require.Equal(t, int64(1), d.ReviewCount)
 require.Len(t, d.Daily, 7)
 require.Equal(t, "2026-10-25", d.Daily[6].Day)
 require.Equal(t, int64(2), d.Daily[6].Closing)
 require.Equal(t, int64(1), d.Daily[6].Leads)
 require.NotEmpty(t, d.Priorities)

 list, err := f.service.List(t.Context(), models.LeadListParams{
  Status: "won", DateField: "converted", From:"2026-10-25", To:"2026-10-25",
 })
 require.NoError(t, err)
 require.Len(t, list.Items, 2)

 reached, err := f.service.List(t.Context(), models.LeadListParams{
  Status: "all", ReachedStage: "discovery", From:"2026-10-19", To:"2026-10-25",
 })
 require.NoError(t, err)
 require.Len(t, reached.Items, 1, "won lead from the selected cohort reaches all funnel stages")
 require.Equal(t, d.Report.Funnel[1].Reached, reached.Total, "list total matches the funnel figure")
 require.Equal(t, map[string]int64{"won": 1}, reached.Counts)
}

func TestLeadListTotalAndColumnCountsCoverAllPages(t *testing.T) {
 f := setupLeadFixture(t)
 created := time.Date(2026, 10, 24, 2, 0, 0, 0, time.UTC)
 for i := 0; i < leadListPageSize+5; i++ {
  _, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, created_at, updated_at, stage, status) VALUES (?, ?, ?, 'discovery', 'open')`, fmt.Sprint("open-", i), created, created)
  require.NoError(t, err)
 }
 for chat, status := range map[string]string{"won": "won", "lost": "lost"} {
  _, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, created_at, updated_at, stage, status) VALUES (?, ?, ?, 'rekomendasi', ?)`, chat, created, created, status)
  require.NoError(t, err)
 }

 all, err := f.service.List(t.Context(), models.LeadListParams{Status: "all", From: "2026-10-24", To: "2026-10-24"})
 require.NoError(t, err)
 require.Len(t, all.Items, leadListPageSize)
 require.Equal(t, int64(leadListPageSize+7), all.Total)
 require.Equal(t, map[string]int64{"discovery": int64(leadListPageSize + 5), "won": 1, "lost": 1}, all.Counts)

 next, err := f.service.List(t.Context(), models.LeadListParams{Status: "all", From: "2026-10-24", To: "2026-10-24", Cursor: all.NextCursor})
 require.NoError(t, err)
 require.Len(t, next.Items, 7)
 require.Equal(t, all.Total, next.Total, "total ignores the cursor")

 open, err := f.service.List(t.Context(), models.LeadListParams{})
 require.NoError(t, err)
 require.Equal(t, int64(leadListPageSize+5), open.Total)
}

func TestPriorityPageFiltersAndOrdering(t *testing.T) {
 f := setupLeadFixture(t)
 created := time.Date(2026, 10, 25, 2, 0, 0, 0, time.UTC)
 for _, row := range []struct{
  chat,name,stage,status string
  review int
  inbound bool
 }{
  {"verify","Zahra","bukti_masuk","open",0,false},
  {"convert","Salman","terverifikasi","open",0,false},
  {"review","Aminah","baru","open",1,false},
  {"reply","Hasan","discovery","open",0,true},
  {"passive","Lukman","baru","open",0,false},
  {"won","Hanan","terverifikasi","won",0,false},
 } {
  var inbound any
  if row.inbound { inbound = created }
  _,err := f.db.Exec(`INSERT INTO leads
   (wa_chat_id,nama,stage,status,needs_review,last_inbound_at,created_at,updated_at)
   VALUES (?,?,?,?,?,?,?,?)`,
   row.chat,row.name,row.stage,row.status,row.review,inbound,created,created)
  require.NoError(t,err)
 }
 all,err := f.service.PriorityPage(t.Context(),"","",1)
 require.NoError(t,err)
 require.Equal(t,int64(4),all.Total)
 require.Len(t,all.Items,4)
 require.Equal(t,[]string{"Zahra","Salman","Aminah","Hasan"},[]string{
  all.Items[0].Name,all.Items[1].Name,all.Items[2].Name,all.Items[3].Name,
 })
 replies,err := f.service.PriorityPage(t.Context(),"has","reply",1)
 require.NoError(t,err)
 require.Equal(t,int64(1),replies.Total)
 require.Equal(t,"Hasan",replies.Items[0].Name)
 absent,err := f.service.PriorityPage(t.Context(),"lukman","",1)
 require.NoError(t,err)
 require.Zero(t,absent.Total)
 require.Empty(t,absent.Items)
}
