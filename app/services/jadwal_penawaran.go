package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// JadwalPenawaranService manages the class schedules offered to calon santri
// (/app/jadwal-penawaran) and what the CS AI bridge reads from them.
//
// A seat counts as taken when a lead with that schedule has a verified
// payment (verified_at) on or after the schedule's periode_mulai and the lead
// is not lost. Quota is per gender (L/P). "Mulai periode baru" moves
// periode_mulai to now, so the next intake starts from zero.
type JadwalPenawaranService struct {
	querier *queries.Querier
	now     func() time.Time
}

func NewJadwalPenawaranService(querier *queries.Querier) *JadwalPenawaranService {
	return &JadwalPenawaranService{querier: querier, now: time.Now}
}

var (
	ErrJadwalNotFound = errors.New("jadwal tidak ditemukan")
	jadwalKodeRe      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,39}$`)
	jamRe             = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

var hariNama = map[int]string{1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jum'at", 6: "Sabtu", 7: "Ahad"}

// parseHari turns "3,6" into [3 6] (sorted, unique, 1..7).
func parseHari(raw string) ([]int, error) {
	seen := map[int]bool{}
	var out []int
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		d, err := strconv.Atoi(p)
		if err != nil || d < 1 || d > 7 {
			return nil, fmt.Errorf("hari %q tidak valid", p)
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("pilih minimal satu hari")
	}
	sort.Ints(out)
	return out, nil
}

func joinHari(days []int) string {
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	return strings.Join(parts, ",")
}

// JadwalLabel is the human text, e.g. "Rabu & Sabtu 19.30–21.00 WIB".
func JadwalLabel(hari, mulai, selesai string) string {
	days, _ := parseHari(hari)
	names := make([]string, 0, len(days))
	for _, d := range days {
		names = append(names, hariNama[d])
	}
	var h string
	switch len(names) {
	case 0:
		h = "-"
	case 1:
		h = names[0]
	default:
		h = strings.Join(names[:len(names)-1], ", ") + " & " + names[len(names)-1]
	}
	return fmt.Sprintf("%s %s–%s WIB", h, strings.ReplaceAll(mulai, ":", "."), strings.ReplaceAll(selesai, ":", "."))
}

func normalizeJadwalInput(req *models.JadwalPenawaranRequest) error {
	days, err := parseHari(req.Hari)
	if err != nil {
		return err
	}
	req.Hari = joinHari(days)
	req.JamMulai, req.JamSelesai = strings.TrimSpace(req.JamMulai), strings.TrimSpace(req.JamSelesai)
	if !jamRe.MatchString(req.JamMulai) || !jamRe.MatchString(req.JamSelesai) {
		return errors.New("jam harus format HH:MM")
	}
	if req.JamSelesai <= req.JamMulai {
		return errors.New("jam selesai harus setelah jam mulai")
	}
	req.Level = strings.TrimSpace(req.Level)
	if req.Level != "" && req.Level != "1" && req.Level != "2" {
		return errors.New("level harus kosong (semua level), 1, atau 2")
	}
	req.Gender = strings.ToUpper(strings.TrimSpace(req.Gender))
	if req.Gender != "" && req.Gender != "L" && req.Gender != "P" {
		return errors.New("jadwal untuk harus ikhwan, akhwat, atau keduanya")
	}
	if req.KuotaPerGender < 1 || req.KuotaPerGender > 500 {
		return errors.New("kuota per gender harus 1–500")
	}
	req.Catatan = truncateRunes(req.Catatan, 300)
	return nil
}

type jadwalSeats map[string]map[string]int64 // kode → gender (L/P) → seats taken

func (s *JadwalPenawaranService) seats(ctx context.Context, rows []queries.JadwalPenawaran) (jadwalSeats, error) {
	start := make(map[string]time.Time, len(rows))
	for _, r := range rows {
		start[r.Kode] = r.PeriodeMulai
	}
	leads, err := s.querier.ListVerifiedLeadSeats(ctx)
	if err != nil {
		return nil, err
	}
	out := jadwalSeats{}
	for _, l := range leads {
		from, ok := start[l.JadwalKode]
		if !ok || !l.VerifiedAt.Valid || l.VerifiedAt.Time.Before(from) {
			continue
		}
		if out[l.JadwalKode] == nil {
			out[l.JadwalKode] = map[string]int64{}
		}
		out[l.JadwalKode][l.JenisKelamin]++
	}
	return out, nil
}

func toJadwalResponse(r queries.JadwalPenawaran, taken map[string]int64) models.JadwalPenawaranResponse {
	days, _ := parseHari(r.Hari)
	sisa := func(g string) int64 {
		if r.Gender != "" && r.Gender != g {
			return 0
		}
		return max(r.KuotaPerGender-taken[g], 0)
	}
	var tidakSesuai int64
	switch r.Gender {
	case "L":
		tidakSesuai = taken["P"]
	case "P":
		tidakSesuai = taken["L"]
	}
	return models.JadwalPenawaranResponse{
		ID: r.ID, Kode: r.Kode, Hari: r.Hari, HariList: days, Frekuensi: len(days),
		JamMulai: r.JamMulai, JamSelesai: r.JamSelesai, Label: JadwalLabel(r.Hari, r.JamMulai, r.JamSelesai),
		Level: r.Level, Gender: r.Gender, Untuk: jadwalUntuk(r.Gender), KuotaPerGender: r.KuotaPerGender, IsBuka: r.IsBuka == 1, Urutan: r.Urutan,
		Catatan: r.Catatan, PeriodeMulai: r.PeriodeMulai.UTC().Format(time.RFC3339),
		TerisiIkhwan: taken["L"], TerisiAkhwat: taken["P"], SisaIkhwan: sisa("L"), SisaAkhwat: sisa("P"),
		TanpaGender: taken[""], TidakSesuai: tidakSesuai,
	}
}

// List returns every schedule with its seat counts (admin page).
func (s *JadwalPenawaranService) List(ctx context.Context) ([]models.JadwalPenawaranResponse, error) {
	rows, err := s.querier.ListJadwalPenawaran(ctx)
	if err != nil {
		return nil, err
	}
	taken, err := s.seats(ctx, rows)
	if err != nil {
		return nil, err
	}
	out := make([]models.JadwalPenawaranResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, toJadwalResponse(r, taken[r.Kode]))
	}
	return out, nil
}

// Open returns the schedules the CS AI may offer: is_buka, with seat counts.
// Full schedules stay in the list (sisa 0) so the AI can say so. gender "L"
// or "P" keeps only schedules for that gender (and the shared ones).
func (s *JadwalPenawaranService) Open(ctx context.Context, gender string) ([]models.JadwalPenawaranResponse, error) {
	gender = strings.ToUpper(strings.TrimSpace(gender))
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.JadwalPenawaranResponse, 0, len(all))
	for _, j := range all {
		if j.IsBuka && (gender == "" || j.Gender == "" || j.Gender == gender) {
			out = append(out, j)
		}
	}
	return out, nil
}

func (s *JadwalPenawaranService) Create(ctx context.Context, req models.JadwalPenawaranRequest) error {
	req.Kode = strings.ToLower(strings.TrimSpace(req.Kode))
	if !jadwalKodeRe.MatchString(req.Kode) {
		return errors.New("kode 2–40 karakter: huruf kecil, angka, tanda minus (mis. selasa-malam)")
	}
	if err := normalizeJadwalInput(&req); err != nil {
		return err
	}
	if _, err := s.querier.GetJadwalPenawaranByKode(ctx, req.Kode); err == nil {
		return fmt.Errorf("kode %q sudah dipakai", req.Kode)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := s.now().UTC().Truncate(time.Second)
	_, err := s.querier.CreateJadwalPenawaran(ctx, queries.CreateJadwalPenawaranParams{
		Kode: req.Kode, Hari: req.Hari, JamMulai: req.JamMulai, JamSelesai: req.JamSelesai, Level: req.Level, Gender: req.Gender,
		KuotaPerGender: req.KuotaPerGender, IsBuka: boolInt(req.IsBuka), Urutan: req.Urutan, Catatan: req.Catatan,
		PeriodeMulai: now, CreatedAt: now, UpdatedAt: now,
	})
	return err
}

// Update changes everything but the kode (leads refer to it).
func (s *JadwalPenawaranService) Update(ctx context.Context, id int64, req models.JadwalPenawaranRequest) error {
	if _, err := s.get(ctx, id); err != nil {
		return err
	}
	if err := normalizeJadwalInput(&req); err != nil {
		return err
	}
	return s.querier.UpdateJadwalPenawaran(ctx, queries.UpdateJadwalPenawaranParams{
		Hari: req.Hari, JamMulai: req.JamMulai, JamSelesai: req.JamSelesai, Level: req.Level, Gender: req.Gender,
		KuotaPerGender: req.KuotaPerGender, IsBuka: boolInt(req.IsBuka), Urutan: req.Urutan, Catatan: req.Catatan,
		UpdatedAt: s.now().UTC().Truncate(time.Second), ID: id,
	})
}

// ResetPeriode starts a new intake: seats verified before now no longer count.
func (s *JadwalPenawaranService) ResetPeriode(ctx context.Context, id int64) error {
	if _, err := s.get(ctx, id); err != nil {
		return err
	}
	now := s.now().UTC().Truncate(time.Second)
	return s.querier.ResetPeriodeJadwalPenawaran(ctx, queries.ResetPeriodeJadwalPenawaranParams{PeriodeMulai: now, UpdatedAt: now, ID: id})
}

// Delete removes a schedule no lead has chosen; otherwise close it instead.
func (s *JadwalPenawaranService) Delete(ctx context.Context, id int64) error {
	j, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.querier.CountLeadsByJadwalKode(ctx, j.Kode)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("jadwal ini dipilih %d lead; tutup saja (jangan dihapus) agar riwayatnya tetap terbaca", n)
	}
	_, err = s.querier.DeleteJadwalPenawaran(ctx, id)
	return err
}

func (s *JadwalPenawaranService) get(ctx context.Context, id int64) (queries.JadwalPenawaran, error) {
	j, err := s.querier.GetJadwalPenawaranByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrJadwalNotFound
	}
	return j, err
}

// jadwalLabelByKode returns the label of a schedule kode, or "" when unknown.
func jadwalLabelByKode(ctx context.Context, q *queries.Querier, kode string) (queries.JadwalPenawaran, string, bool, error) {
	if kode == "" {
		return queries.JadwalPenawaran{}, "", false, nil
	}
	j, err := q.GetJadwalPenawaranByKode(ctx, kode)
	if errors.Is(err, sql.ErrNoRows) {
		return j, "", false, nil
	}
	if err != nil {
		return j, "", false, err
	}
	return j, jadwalFullLabel(j), true, nil
}

// jadwalFullLabel is the label with level and gender, e.g.
// "Selasa 19.30–21.00 WIB (level 2) · khusus ikhwan".
func jadwalFullLabel(j queries.JadwalPenawaran) string {
	label := JadwalLabel(j.Hari, j.JamMulai, j.JamSelesai)
	if j.Level != "" {
		label += " (level " + j.Level + ")"
	}
	if j.Gender != "" {
		label += " · khusus " + strings.ToLower(jadwalUntuk(j.Gender))
	}
	return label
}

func jadwalUntuk(gender string) string {
	switch gender {
	case "L":
		return "Ikhwan"
	case "P":
		return "Akhwat"
	}
	return "Ikhwan & Akhwat"
}

// jadwalGenderMismatch is true when a single-gender schedule is chosen for a
// lead of the other gender. An unknown lead gender is not a mismatch.
func jadwalGenderMismatch(j queries.JadwalPenawaran, leadGender string) bool {
	return j.Gender != "" && leadGender != "" && j.Gender != leadGender
}

func genderNama(g string) string {
	if g == "P" {
		return "akhwat"
	}
	return "ikhwan"
}
