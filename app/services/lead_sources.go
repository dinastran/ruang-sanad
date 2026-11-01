package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Lead sources tell which landing page or ad a lead came from. Ads open
// WhatsApp with a prefilled text, so the first inbound message is matched
// against each source's trigger texts. The result is stored on the lead and
// never recomputed on its own: editing triggers does not reshuffle old leads.
// "Cocokkan ulang" fills only leads that still have no source and were not
// set by hand.

const (
	LeadSourceByAuto   = "auto"
	LeadSourceByManual = "manual"
	// minPatternRunes keeps triggers specific: "halo" would match every lead.
	minPatternRunes = 12
)

var ErrLeadSourceNotFound = errors.New("sumber lead tidak ditemukan")

// NormalizeLeadSourceText lowercases and keeps only letters and digits,
// separated by single spaces: "Assalamu'alaikum,  Kak!" → "assalamu alaikum kak".
func NormalizeLeadSourceText(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		} else {
			space = true
		}
	}
	return b.String()
}

// matchLeadSource returns the source whose trigger appears as whole words in
// body. The longest (most specific) trigger wins; 0 when none matches.
func matchLeadSource(patterns []queries.ListActiveLeadSourcePatternsRow, body string) int64 {
	text := " " + NormalizeLeadSourceText(body) + " "
	var best int64
	bestLen := 0
	for _, p := range patterns {
		if p.TeksNormal == "" || len(p.TeksNormal) <= bestLen {
			continue
		}
		if strings.Contains(text, " "+p.TeksNormal+" ") {
			best, bestLen = p.SourceID, len(p.TeksNormal)
		}
	}
	return best
}

// detectLeadSource runs once per lead, on its first inbound message. A lead
// with no match is marked checked (auto, no source) so later messages such
// as "halo" never assign one.
func detectLeadSource(ctx context.Context, q *queries.Querier, lead *queries.Lead, body string, now time.Time, payload map[string]any) error {
	if lead.LeadSourceID.Valid || lead.LeadSourceBy != "" {
		return nil
	}
	n, err := q.CountLeadInboundMessages(ctx, lead.ID)
	if err != nil || n != 1 {
		return err
	}
	patterns, err := q.ListActiveLeadSourcePatterns(ctx)
	if err != nil {
		return err
	}
	id := matchLeadSource(patterns, body)
	lead.LeadSourceID = sql.NullInt64{Int64: id, Valid: id > 0}
	lead.LeadSourceBy = LeadSourceByAuto
	if id > 0 {
		payload["lead_source_id"] = id
	}
	return q.SetLeadSource(ctx, queries.SetLeadSourceParams{
		LeadSourceID: lead.LeadSourceID, LeadSourceBy: lead.LeadSourceBy, UpdatedAt: now, ID: lead.ID,
	})
}

// Sources lists every source with its trigger texts and all-time lead count.
func (s *LeadService) Sources(ctx context.Context) ([]models.LeadSourceResponse, error) {
	rows, err := s.querier.ListLeadSources(ctx)
	if err != nil {
		return nil, err
	}
	patterns, err := s.querier.ListLeadSourcePatterns(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.querier.CountLeadsBySource(ctx)
	if err != nil {
		return nil, err
	}
	bySource := map[int64][]models.LeadSourcePatternResponse{}
	for _, p := range patterns {
		bySource[p.SourceID] = append(bySource[p.SourceID], models.LeadSourcePatternResponse{ID: p.ID, Teks: p.Teks})
	}
	total := map[int64]int64{}
	for _, c := range counts {
		total[c.LeadSourceID.Int64] = c.Total
	}
	out := make([]models.LeadSourceResponse, 0, len(rows))
	for _, r := range rows {
		p := bySource[r.ID]
		if p == nil {
			p = []models.LeadSourcePatternResponse{}
		}
		out = append(out, models.LeadSourceResponse{
			ID: r.ID, Nama: r.Nama, Keterangan: r.Keterangan, URL: r.Url, IsAktif: r.IsAktif == 1, Patterns: p, TotalLeads: total[r.ID],
		})
	}
	return out, nil
}

// SourceOptions lists sources for selects (lead page, list filter).
func (s *LeadService) SourceOptions(ctx context.Context) ([]models.LeadSourceOption, error) {
	rows, err := s.querier.ListLeadSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.LeadSourceOption, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.LeadSourceOption{ID: r.ID, Nama: r.Nama, IsAktif: r.IsAktif == 1})
	}
	return out, nil
}

func normalizeLeadSourceRequest(req *models.LeadSourceRequest) error {
	req.Nama = strings.Join(strings.Fields(req.Nama), " ")
	if n := len([]rune(req.Nama)); n < 2 || n > 80 {
		return errors.New("nama sumber 2–80 karakter")
	}
	req.Keterangan = truncateRunes(strings.TrimSpace(req.Keterangan), 300)
	req.URL = strings.TrimSpace(req.URL)
	if req.URL != "" {
		u, err := url.Parse(req.URL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(req.URL) > 500 {
			return errors.New("URL LP harus lengkap, mis. https://ruangsanad.com/tahsin")
		}
	}
	return nil
}

func duplicateNameErr(err error, nama string) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("sumber %q sudah ada", nama)
	}
	return err
}

func (s *LeadService) CreateSource(ctx context.Context, req models.LeadSourceRequest) (int64, error) {
	if err := normalizeLeadSourceRequest(&req); err != nil {
		return 0, err
	}
	now := s.nowUTC()
	row, err := s.querier.CreateLeadSource(ctx, queries.CreateLeadSourceParams{
		Nama: req.Nama, Keterangan: req.Keterangan, Url: req.URL, IsAktif: boolInt(req.IsAktif), CreatedAt: now, UpdatedAt: now,
	})
	return row.ID, duplicateNameErr(err, req.Nama)
}

func (s *LeadService) UpdateSource(ctx context.Context, id int64, req models.LeadSourceRequest) error {
	if _, err := s.getSource(ctx, id); err != nil {
		return err
	}
	if err := normalizeLeadSourceRequest(&req); err != nil {
		return err
	}
	return duplicateNameErr(s.querier.UpdateLeadSource(ctx, queries.UpdateLeadSourceParams{
		Nama: req.Nama, Keterangan: req.Keterangan, Url: req.URL, IsAktif: boolInt(req.IsAktif), UpdatedAt: s.nowUTC(), ID: id,
	}), req.Nama)
}

// DeleteSource removes a source no lead uses; otherwise deactivate it so the
// report keeps its history.
func (s *LeadService) DeleteSource(ctx context.Context, id int64) error {
	if _, err := s.getSource(ctx, id); err != nil {
		return err
	}
	n, err := s.querier.CountLeadsOfSource(ctx, sql.NullInt64{Int64: id, Valid: true})
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("sumber ini dipakai %d lead; nonaktifkan saja agar laporannya tetap terbaca", n)
	}
	_, err = s.querier.DeleteLeadSource(ctx, id)
	return err
}

// AddSourcePattern adds a trigger text. It must be specific enough and not
// already belong to any source.
func (s *LeadService) AddSourcePattern(ctx context.Context, sourceID int64, teks string) error {
	if _, err := s.getSource(ctx, sourceID); err != nil {
		return err
	}
	teks = truncateRunes(strings.Join(strings.Fields(teks), " "), 300)
	normal := NormalizeLeadSourceText(teks)
	if len([]rune(normal)) < minPatternRunes {
		return fmt.Errorf("teks pemicu terlalu pendek (minimal %d huruf); pakai kalimat khas dari LP/iklan", minPatternRunes)
	}
	if existing, err := s.querier.GetLeadSourcePatternByNormal(ctx, normal); err == nil {
		if existing.SourceID == sourceID {
			return errors.New("teks pemicu ini sudah ada di sumber ini")
		}
		src, _ := s.querier.GetLeadSource(ctx, existing.SourceID)
		return fmt.Errorf("teks pemicu ini sudah dipakai sumber %q", src.Nama)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err := s.querier.CreateLeadSourcePattern(ctx, queries.CreateLeadSourcePatternParams{
		SourceID: sourceID, Teks: teks, TeksNormal: normal, CreatedAt: s.nowUTC(),
	})
	return err
}

func (s *LeadService) DeleteSourcePattern(ctx context.Context, id int64) error {
	n, err := s.querier.DeleteLeadSourcePattern(ctx, id)
	if err == nil && n == 0 {
		return errors.New("teks pemicu tidak ditemukan")
	}
	return err
}

// RematchSources fills leads without a source (not set by hand) from their
// first inbound message, using the current triggers. Returns how many leads
// got a source.
func (s *LeadService) RematchSources(ctx context.Context) (int, error) {
	matched := 0
	err := s.withTx(ctx, func(q *queries.Querier) error {
		patterns, err := q.ListActiveLeadSourcePatterns(ctx)
		if err != nil {
			return err
		}
		leads, err := q.ListLeadsForSourceRematch(ctx)
		if err != nil {
			return err
		}
		now := s.nowUTC()
		for _, l := range leads {
			if l.FirstBody == "" {
				continue
			}
			id := matchLeadSource(patterns, l.FirstBody)
			if id == 0 {
				continue
			}
			if err := q.SetLeadSource(ctx, queries.SetLeadSourceParams{
				LeadSourceID: sql.NullInt64{Int64: id, Valid: true}, LeadSourceBy: LeadSourceByAuto, UpdatedAt: now, ID: l.ID,
			}); err != nil {
				return err
			}
			if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("src"), "source.detected", "dashboard",
				map[string]any{"lead_source_id": id, "rematch": true}, now); err != nil {
				return err
			}
			matched++
		}
		return nil
	})
	return matched, err
}

// SetLeadSourceManual lets CS correct a lead's source; 0 = tanpa sumber.
// A manual choice is never overwritten by matching.
func (s *LeadService) SetLeadSourceManual(ctx context.Context, leadID, userID, sourceID int64) error {
	if sourceID < 0 {
		return errors.New("sumber tidak valid")
	}
	if sourceID > 0 {
		if _, err := s.getSource(ctx, sourceID); err != nil {
			return err
		}
	}
	return s.withTx(ctx, func(q *queries.Querier) error {
		l, err := q.GetLeadByID(ctx, leadID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLeadNotFound
		}
		if err != nil {
			return err
		}
		now := s.nowUTC()
		if err := q.SetLeadSource(ctx, queries.SetLeadSourceParams{
			LeadSourceID: sql.NullInt64{Int64: sourceID, Valid: sourceID > 0}, LeadSourceBy: LeadSourceByManual, UpdatedAt: now, ID: leadID,
		}); err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("usr"), "source.changed", userActor(userID),
			map[string]any{"lead_source_from": l.LeadSourceID.Int64, "lead_source_to": sourceID}, now)
	})
}

// sourceStats returns the period cohort per source, largest first, with
// "Tanpa sumber" (id 0) last.
func (s *LeadService) sourceStats(ctx context.Context, rows []queries.LeadSourceCohortRow) ([]models.LeadSourceStat, error) {
	sources, err := s.querier.ListLeadSources(ctx)
	if err != nil {
		return nil, err
	}
	names := map[int64]string{}
	for _, src := range sources {
		names[src.ID] = src.Nama
	}
	out := make([]models.LeadSourceStat, 0, len(rows))
	var none *models.LeadSourceStat
	for _, r := range rows {
		st := models.LeadSourceStat{ID: r.SourceID, Nama: names[r.SourceID], Leads: r.Total, Won: r.Won, Open: r.Open}
		if r.SourceID == 0 || st.Nama == "" {
			if none == nil {
				none = &models.LeadSourceStat{Nama: "Tanpa sumber"}
			}
			none.Leads += st.Leads
			none.Won += st.Won
			none.Open += st.Open
			continue
		}
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Leads != out[j].Leads {
			return out[i].Leads > out[j].Leads
		}
		return out[i].Nama < out[j].Nama
	})
	if none != nil {
		out = append(out, *none)
	}
	return out, nil
}

func (s *LeadService) getSource(ctx context.Context, id int64) (queries.LeadSource, error) {
	src, err := s.querier.GetLeadSource(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return src, ErrLeadSourceNotFound
	}
	return src, err
}
