package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// SetProofDir sets where manual payment proofs are stored.
func (s *RegistrationService) SetProofDir(dir string) { s.proofDir = dir }

// ProofDir is where manual payment proofs are stored.
func (s *RegistrationService) ProofDir() string { return s.proofDir }

// SetNow overrides the clock (tests).
func (s *RegistrationService) SetNow(now func() time.Time) { s.now = now }

// CountClick records one visit of a link, deduplicated per IP + user agent
// per WIB day in memory (indicative, resets on restart).
func (s *RegistrationService) CountClick(ctx context.Context, linkID int64, ip, ua string) error {
	day := s.now().In(wib).Format(time.DateOnly)
	key := strings.Join([]string{itoa64(linkID), ip, ua}, "|")
	s.mu.Lock()
	if s.clickDay != day {
		s.clickDay, s.clickSeen = day, map[string]struct{}{}
	}
	_, seen := s.clickSeen[key]
	s.clickSeen[key] = struct{}{}
	s.mu.Unlock()
	if seen {
		return nil
	}
	return s.querier.BumpRegistrationStat(ctx, queries.BumpRegistrationStatParams{LinkID: linkID, Tanggal: day, Klik: 1})
}

// LinkWaURL is the wa.me redirect for a mode-wa link (no form data yet).
func (s *RegistrationService) LinkWaURL(ctx context.Context, slug string) string {
	l, prog, _, err := s.linkContext(ctx, slug)
	if err != nil {
		return s.CSWaURL(ctx, "")
	}
	tpl := l.WaTemplate
	if strings.TrimSpace(tpl) == "" {
		tpl = "Assalamu'alaikum, saya ingin mendaftar {program}."
	}
	return s.CSWaURL(ctx, strings.TrimSpace(fillWaTemplate(tpl, "", prog.Nama, "", "")))
}

// PublicJadwal lists open schedules with seats per gender for the form.
func (s *RegistrationService) PublicJadwal(ctx context.Context) ([]models.JadwalPenawaranResponse, error) {
	return s.jadwal.Open(ctx, "")
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }
