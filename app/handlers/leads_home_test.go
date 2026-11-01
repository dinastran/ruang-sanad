package handlers

import (
 "net/http/httptest"
 "testing"

 "github.com/gofiber/fiber/v2"
 "github.com/stretchr/testify/require"
)

// URLs already used by CRM filters must continue opening the list after
// /app/leads becomes the new default dashboard.
func TestLeadOverviewRoutingCompatibility(t *testing.T) {
 app := fiber.New()
 app.Get("/app/leads",func(c *fiber.Ctx) error {
  if leadOverviewRequested(c) { return c.SendString("overview") }
  return c.SendString("list")
 })
 cases := []struct{ url, want string }{
  {"/app/leads","overview"},
  {"/app/leads?summary_days=7","overview"},
  {"/app/leads?summary_days=30","overview"},
  {"/app/leads?view=list","list"},
  {"/app/leads?status=all","list"},
  {"/app/leads?status=won&date_field=converted&from=2026-10-01&to=2026-10-31","list"},
  {"/app/leads?q=Fulan","list"},
  {"/app/leads?cursor=abc","list"},
  {"/app/leads?view=list&summary_days=14","list"},
 }
 for _,tc := range cases {
  t.Run(tc.url,func(t *testing.T) {
   req := httptest.NewRequest("GET",tc.url,nil)
   resp,err := app.Test(req)
   require.NoError(t,err)
   defer resp.Body.Close()
   require.Equal(t,200,resp.StatusCode)
   body := make([]byte,16)
   n,err:=resp.Body.Read(body)
   require.NoError(t,err)
   require.Equal(t,tc.want,string(body[:n]))
  })
 }
}
