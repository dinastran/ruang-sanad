package services

import (
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
}
