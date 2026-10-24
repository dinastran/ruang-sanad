package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

// leadReportRange parses from/to (YYYY-MM-DD, WIB, inclusive) with a default
// of the last 30 days. It returns UTC-comparable [start, end) bounds.
func (s *LeadService) leadReportRange(from, to string) (time.Time, time.Time, string, string) {
	today := s.now().In(wib)
	end, ok := wibDayStart(to)
	if !ok {
		end = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, wib)
	}
	start, ok := wibDayStart(from)
	if !ok || start.After(end) {
		start = end.AddDate(0, 0, -29)
	}
	if end.Sub(start) > 366*24*time.Hour {
		start = end.AddDate(-1, 0, 0)
	}
	return start, end.AddDate(0, 0, 1), start.Format(time.DateOnly), end.Format(time.DateOnly)
}

func countByKey(m map[string]int64) []models.LeadCountByKey {
	out := make([]models.LeadCountByKey, 0, len(m))
	for k, v := range m {
		out = append(out, models.LeadCountByKey{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// Report builds the funnel report for leads created in the range (PRD 8.4).
// A lead "reached" a stage when its current stage is at or past it (stage is
// kept when a lead is lost), and won leads reached every stage.
func (s *LeadService) Report(ctx context.Context, from, to string) (*models.LeadReport, error) {
	start, end, fromStr, toStr := s.leadReportRange(from, to)
	rows, err := s.querier.ListLeadReportRows(ctx, start, end)
	if err != nil {
		return nil, err
	}
	rep := &models.LeadReport{From: fromStr, To: toStr, TotalLeads: int64(len(rows))}
	reached := make([]int64, len(LeadStages))
	lost := map[string]int64{}
	objections := map[string]int64{}
	weekly := map[string]*models.LeadWeekRow{}
	var hoursToWon []float64
	for _, r := range rows {
		switch r.Status {
		case LeadStatusWon:
			rep.Won++
		case LeadStatusLost:
			rep.Lost++
			lost[r.LostReason]++
		case LeadStatusDoNotContact:
			rep.DoNotContact++
		default:
			rep.Open++
		}
		idx := leadStageIndex(r.Stage)
		if r.Status == LeadStatusWon {
			idx = len(LeadStages) - 1
		}
		for i := 0; i <= idx; i++ {
			reached[i]++
		}
		if r.Objection != "" {
			objections[r.Objection]++
		}
		year, week := r.CreatedAt.In(wib).ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		if weekly[key] == nil {
			weekly[key] = &models.LeadWeekRow{Week: key}
		}
		weekly[key].Leads++
		if r.Status == LeadStatusWon {
			weekly[key].Won++
			if r.ConvertedAt.Valid {
				hoursToWon = append(hoursToWon, r.ConvertedAt.Time.Sub(r.CreatedAt).Hours())
			}
		}
	}
	for i, stage := range LeadStages {
		row := models.LeadFunnelStage{Stage: stage, Reached: reached[i]}
		if i > 0 && reached[i-1] > 0 {
			row.Conversion = float64(reached[i]) / float64(reached[i-1])
		}
		rep.Funnel = append(rep.Funnel, row)
	}
	rep.Funnel = append(rep.Funnel, models.LeadFunnelStage{Stage: "won", Reached: rep.Won, Conversion: ratio(rep.Won, reached[len(reached)-1])})
	if len(hoursToWon) > 0 {
		sort.Float64s(hoursToWon)
		mid := len(hoursToWon) / 2
		if len(hoursToWon)%2 == 1 {
			rep.MedianHoursToWon = hoursToWon[mid]
		} else {
			rep.MedianHoursToWon = (hoursToWon[mid-1] + hoursToWon[mid]) / 2
		}
	}
	rep.LostReasons = countByKey(lost)
	rep.Objections = countByKey(objections)
	rep.Weekly = make([]models.LeadWeekRow, 0, len(weekly))
	for _, w := range weekly {
		rep.Weekly = append(rep.Weekly, *w)
	}
	sort.Slice(rep.Weekly, func(i, j int) bool { return rep.Weekly[i].Week < rep.Weekly[j].Week })

	sent, err := s.querier.ListSentLeadFollowups(ctx, start, end)
	if err != nil {
		return nil, err
	}
	wonLeads := map[int64]bool{}
	for _, f := range sent {
		rep.FollowupSent++
		replied, err := s.querier.LeadHasInboundBetween(ctx, f.LeadID, f.SentAt, f.SentAt.Add(72*time.Hour))
		if err != nil {
			return nil, err
		}
		if replied {
			rep.FollowupReplied++
		}
		if f.LeadStatus == LeadStatusWon && !wonLeads[f.LeadID] {
			wonLeads[f.LeadID] = true
			rep.FollowupWon++
		}
	}
	return rep, nil
}

func ratio(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// ExportCSV writes the leads of the range as CSV. Phone numbers are included
// only for super_admin (PRD 8.4).
func (s *LeadService) ExportCSV(ctx context.Context, w io.Writer, from, to string, includePhone bool) error {
	start, end, _, _ := s.leadReportRange(from, to)
	rows, err := s.querier.ListLeadExportRows(ctx, start, end)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	header := []string{"lead_ref", "nama", "nama_wa", "tahap", "status", "alasan_hilang", "paket", "dibuat_wib"}
	if includePhone {
		header = append(header, "nomor")
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		line := []string{
			"L-" + strconv.FormatInt(r.ID, 10), csvSafe(r.Nama), csvSafe(r.WaName), csvSafe(r.Stage), csvSafe(r.Status), csvSafe(r.LostReason), csvSafe(r.PaketKode),
			r.CreatedAt.In(wib).Format("2006-01-02 15:04"),
		}
		if includePhone {
			line = append(line, r.Phone)
		}
		if err := cw.Write(line); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvSafe neutralizes spreadsheet formula injection in text cells.
func csvSafe(v string) string {
	trimmed := strings.TrimLeft(v, " \t\r\n")
	if v != "" && (strings.ContainsAny(v[:1], "\t\r") || (trimmed != "" && strings.ContainsAny(trimmed[:1], "=+-@"))) {
		return "'" + v
	}
	return v
}
