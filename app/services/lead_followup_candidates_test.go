package services

import (
 "fmt"
 "strings"
 "database/sql"
 "encoding/json"
 "testing"
 "time"

 "github.com/maulanashalihin/laju-go/app/queries"
 "github.com/stretchr/testify/require"
)

func enablePilotLead(t *testing.T, c *crmAgentFixture, id int64) {
 t.Helper()
 _, err := c.db.Exec(`INSERT INTO lead_label_links (lead_id, label_id)
 SELECT ?, id FROM lead_labels WHERE nama = 'uji-followup'`, id)
 require.NoError(t, err)
}

func pilotCandidateList(t *testing.T, c *crmAgentFixture, args map[string]any) map[string]any {
 t.Helper()
 if args == nil { args = map[string]any{} }
 txt, isErr := c.call(t, c.writer(), "daftar_kandidat_followup", args)
 require.False(t, isErr, txt)
 var value map[string]any
 require.NoError(t, jsonUnmarshalPilot([]byte(txt), &value))
 return value
}

// The MCP result is JSON; use encoding/json so the tests assert the public contract.
func jsonUnmarshalPilot(b []byte, dst any) error {
 return json.Unmarshal(b, dst)
}

func TestHermesPilotCandidateRequiresLabelAndNoReply(t *testing.T) {
 c := setupCRMAgent(t)
 id := c.seedConversation(t)
 c.clock = time.Date(2026, 10, 25, 5, 0, 0, 0, time.UTC)
 require.Contains(t, namesForPilot(c.agent.ToolsFor("cs", false)), "daftar_kandidat_followup")
 require.NotContains(t, namesForPilot(c.agent.ToolsFor("admin_kelas", true)), "daftar_kandidat_followup")
 blank := pilotCandidateList(t, c, nil)
 require.Equal(t, float64(0), blank["jumlah"])
 enablePilotLead(t, c, id)
 data := pilotCandidateList(t, c, nil)
 require.Equal(t, float64(1), data["jumlah"])
 require.NotContains(t, fmt.Sprint(data), "6281234567890", "no phone in MCP discovery")
 items := data["kandidat"].([]any)
 item := items[0].(map[string]any)
 require.Equal(t, fmt.Sprintf("L-%d", id), item["lead_ref"])
 require.Equal(t, "d1", item["step_disarankan"])

 _, err := c.db.Exec(`UPDATE leads SET last_inbound_at = ? WHERE id = ?`,
  time.Date(2026, 10, 25, 4, 0, 0, 0, time.UTC), id)
 require.NoError(t, err)
 require.Equal(t, float64(0), pilotCandidateList(t, c, nil)["jumlah"])
}

func namesForPilot(ts []AITool) []string {
 out := []string{}
 for _, tool := range ts { out = append(out, tool.Name) }
 return out
}

func TestHermesPilotPreservesProposalAndDeduplicates(t *testing.T) {
 c := setupCRMAgent(t)
 id := c.seedConversation(t)
 enablePilotLead(t, c, id)
 c.clock = time.Date(2026, 10, 25, 5, 0, 0, 0, time.UTC)
 require.Equal(t, float64(1), pilotCandidateList(t, c, nil)["jumlah"])
 text, isErr := c.call(t, c.writer(), "usulkan_followup", map[string]any{
  "lead_ref": fmt.Sprintf("L-%d", id), "step": "d1",
  "scheduled_at": "2026-10-25 13:00", "draft_text": d1Draft,
 })
 require.False(t, isErr, text)
 require.Contains(t, text, "pending_approval")
 require.Equal(t, float64(0), pilotCandidateList(t, c, nil)["jumlah"], "existing pending proposal suppresses scan")
 text, isErr = c.call(t, c.writer(), "usulkan_followup", map[string]any{
  "lead_ref": fmt.Sprintf("L-%d", id), "step": "d1",
  "scheduled_at": "2026-10-25 13:30", "draft_text": d1Draft,
 })
 require.True(t, isErr)
 require.True(t, strings.Contains(text, "menunggu persetujuan") || strings.Contains(text, "satu sentuhan"))

 _, err := c.db.Exec(`UPDATE lead_followups SET status = 'sent', sent_at = ? WHERE lead_id = ?`,
  time.Date(2026, 10, 25, 6, 0, 0, 0, time.UTC), id)
 require.NoError(t, err)
 c.clock = time.Date(2026, 10, 27, 7, 0, 0, 0, time.UTC)
 rows := pilotCandidateList(t, c, nil)["kandidat"].([]any)
 require.Len(t, rows, 1)
 require.Equal(t, "d3", rows[0].(map[string]any)["step_disarankan"])
}

func TestHermesPilotStatusAndOptOutGuards(t *testing.T) {
 for _, tc := range []struct {
  name string
  sql string
 }{
  {"do_not_contact", "UPDATE leads SET status = 'do_not_contact' WHERE id = ?"},
  {"won", "UPDATE leads SET status = 'won' WHERE id = ?"},
  {"lost", "UPDATE leads SET status = 'lost' WHERE id = ?"},
  {"optout", "UPDATE leads SET followup_permission = 'no' WHERE id = ?"},
  {"proof_processing", "UPDATE leads SET stage = 'bukti_masuk' WHERE id = ?"},
 } {
  t.Run(tc.name, func(t *testing.T) {
   c := setupCRMAgent(t)
   id := c.seedConversation(t)
   enablePilotLead(t, c, id)
   c.clock = time.Date(2026, 10, 25, 5, 0, 0, 0, time.UTC)
   _, err := c.db.Exec(tc.sql, id)
   require.NoError(t, err)
   require.Equal(t, float64(0), pilotCandidateList(t, c, nil)["jumlah"])
  })
 }
}

func TestNextPilotFollowupStepSpacingAndSkipD7(t *testing.T) {
 anchor := time.Date(2026, 10, 24, 3, 0, 0, 0, time.UTC)
 r := queries.DormantFollowupCandidate{Anchor: anchor, Permission: "unknown"}
 step, _, eligible := nextPilotFollowupStep(r, anchor.Add(25*time.Hour))
 require.True(t, eligible)
 require.Equal(t, "d1", step)
 r.LastStep = sql.NullString{String: "d3", Valid:true}
 r.LastSentAt = sql.NullTime{Time: anchor.AddDate(0, 0, 3), Valid:true}
 step, _, eligible = nextPilotFollowupStep(r, anchor.AddDate(0, 0, 7))
 require.True(t, eligible)
 require.Equal(t, "d7", step)
 step, _, eligible = nextPilotFollowupStep(r, anchor.AddDate(0, 0, 14))
 require.True(t, eligible)
 require.Equal(t, "d14", step)
 r.LastStep = sql.NullString{String:"d14", Valid:true}
 r.LastSentAt = sql.NullTime{Time:anchor.AddDate(0, 0, 14),Valid:true}
 step, _, eligible = nextPilotFollowupStep(r, anchor.AddDate(0, 0, 31))
 require.False(t, eligible, "d30 must have explicit permission")
 r.Permission = "yes"
 step, _, eligible = nextPilotFollowupStep(r, anchor.AddDate(0, 0, 31))
 require.True(t, eligible)
 require.Equal(t, "d30", step)

 // D14 consent reply stops the previous sequence; a subsequent CS response
 // establishes a new anchor, but the recent consent history permits D30.
 r.LastStep = sql.NullString{}
 r.LastSentAt = sql.NullTime{}
 r.Anchor = anchor.AddDate(0, 0, 16)
 r.PreviousD14At = sql.NullTime{Time: anchor.AddDate(0, 0, 14), Valid:true}
 step, _, eligible = nextPilotFollowupStep(r, anchor.AddDate(0, 0, 46))
 require.True(t, eligible)
 require.Equal(t, "d30", step)
}

func TestPilotSuggestedTimeRespectsWIBWindow(t *testing.T) {
 settings := queries.CrmSetting{FollowupSendStart:"08:00", FollowupSendEnd:"20:00"}
 require.Equal(t, "2026-10-25 08:00", pilotSuggestedTime(
  time.Date(2026, 10, 24, 23, 0, 0, 0, time.UTC), settings))
 require.Equal(t, "2026-10-26 08:00", pilotSuggestedTime(
  time.Date(2026, 10, 25, 13, 0, 0, 0, time.UTC), settings))
}
