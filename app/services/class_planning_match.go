package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// PlanningMatchResult is only a snapshot. It never reserves seats or teachers,
// creates classes, assigns santri, or publishes jadwal penawaran.
type PlanningMatchResult struct {
	AsOf               string                     `json:"as_of"`
	ConfirmedDemand    int                        `json:"confirmed_demand"`
	EligibleDemand     int                        `json:"eligible_demand"`
	SkippedAmbiguous   int                        `json:"skipped_ambiguous"`
	Truncated          bool                       `json:"truncated"`
	Suggestions        []PlanningMatchOption      `json:"suggestions"`
	AlternativeMatches []PlanningAlternativeMatch `json:"alternative_matches"`
	Warnings           []string                   `json:"warnings"`
}

type PlanningTeacherChoice struct {
	GuruID        int64  `json:"guru_id"`
	Name          string `json:"name"`
	Gender        string `json:"gender"`
	Status        string `json:"status"`
	WeeklyMinutes int    `json:"weekly_minutes"`
}

type PlanningMatchOption struct {
	Kind           string                  `json:"kind"`  // existing_class or new_class
	State          string                  `json:"state"` // ready_preview, needs_quota, needs_teacher
	Angkatan       string                  `json:"angkatan"`
	Gender         string                  `json:"gender"`
	Level          string                  `json:"level"`
	Frequency      int                     `json:"frequency"`
	OfferCode      string                  `json:"offer_code"`
	ClassID        int64                   `json:"class_id,omitempty"`
	CandidateIDs   []int64                 `json:"candidate_ids"`
	SeatsAvailable int                     `json:"seats_available"`
	MinimumToStart int                     `json:"minimum_to_start"`
	Slots          []PlanningSlot          `json:"slots"`
	TeacherOptions []PlanningTeacherChoice `json:"teacher_options"`
	Reason         string                  `json:"reason"`
}

type PlanningMatchSnapshot struct {
	Demand               []queries.PlanningDemand
	ApprovedAlternatives []queries.PlanningApprovedAlternative
	Classes              []queries.PlanningExistingClass
	ClassSlots           []queries.PlanningClassSlot
	Windows              []queries.PlanningTeacherWindow
	Levels               []queries.PlanningTeacherLevel
	Busy                 []queries.PlanningTeacherBusy
	UnsafeTeacher        map[int64]bool
	MaxWorkload          map[int64]int64 // absent means policy unknown
}

type PlanningMatchingService struct {
	querier *queries.Querier
}

func NewPlanningMatchingService(q *queries.Querier) *PlanningMatchingService {
	return &PlanningMatchingService{querier: q}
}

func (s *PlanningMatchingService) Recommend(ctx context.Context) (PlanningMatchResult, error) {
	zone, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return PlanningMatchResult{}, err
	}
	asOf := time.Now().In(zone).Format(time.DateOnly)
	demand, err := s.querier.ListPlanningDemand(ctx)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	alternatives, err := s.querier.ListPlanningApprovedAlternatives(ctx)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	classes, err := s.querier.ListPlanningExistingClasses(ctx)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	classSlots, err := s.querier.ListPlanningCurrentClassSlots(ctx, asOf)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	windows, err := s.querier.ListPlanningTeacherWindows(ctx, asOf)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	levels, err := s.querier.ListPlanningTeacherLevels(ctx)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	busy, err := s.querier.ListPlanningTeacherBusy(ctx, asOf)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	unsafe, err := s.querier.ListPlanningUnsafeTeacherIDs(ctx, asOf)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	limits, err := s.querier.ListPlanningWorkloadLimits(ctx)
	if err != nil {
		return PlanningMatchResult{}, err
	}
	maxWorkload := make(map[int64]int64)
	for id, max := range limits {
		if max.Valid {
			maxWorkload[id] = max.Int64
		}
	}
	return PlanMatchingPreview(asOf, PlanningMatchSnapshot{
		Demand: demand, ApprovedAlternatives: alternatives, Classes: classes, ClassSlots: classSlots,
		Windows: windows, Levels: levels, Busy: busy,
		UnsafeTeacher: unsafe, MaxWorkload: maxWorkload,
	}), nil
}

type planningDemandGroup struct {
	key       string
	angkatan  string
	gender    string
	level     string
	frequency int
	offer     string
	slots     []PlanningSlot
	ids       []int64
}

func planningFrequency(raw string) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1x/pekan":
		return 1
	case "2x/pekan":
		return 2
	default:
		return 0
	}
}

func planningOfferSlots(row queries.PlanningDemand) ([]PlanningSlot, int, bool) {
	days, err := parseHari(row.OfferDays)
	frequency := planningFrequency(row.Frequency)
	if err != nil || len(days) != frequency || frequency == 0 {
		return nil, 0, false
	}
	start, startOK := planningMinutes(row.OfferStart)
	end, endOK := planningMinutes(row.OfferEnd)
	if !startOK || !endOK || end-start != planningLessonMinutes {
		return nil, 0, false
	}
	out := make([]PlanningSlot, 0, len(days))
	for _, day := range days {
		out = append(out, PlanningSlot{Day: day, Start: row.OfferStart, End: row.OfferEnd})
	}
	return out, frequency, true
}

func planningSameSlots(desired []PlanningSlot, actual []queries.PlanningClassSlot) bool {
	if len(desired) != len(actual) {
		return false
	}
	for _, want := range desired {
		found := false
		for _, got := range actual {
			if int64(want.Day) == got.Day && want.Start == got.Start {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func planningGenderTeacherOK(teacherGender, studentGender string) bool {
	return (teacherGender == "L" || teacherGender == "P") &&
		(studentGender == "L" || studentGender == "P") &&
		!(teacherGender == "P" && studentGender == "L")
}

func planningTeacherRank(t PlanningTeacherChoice, group planningDemandGroup) int {
	score := 0
	if group.gender == "P" && t.Gender == "P" {
		score += 100
	}
	if strings.EqualFold(t.Status, "tetap") {
		score += 20
	}
	// Smaller current load is preferred until a confirmed target exists.
	score -= t.WeeklyMinutes / 90
	return score
}

func planningChoices(group planningDemandGroup, snap PlanningMatchSnapshot, occupied map[int64][]PlanningSlot) []PlanningTeacherChoice {
	windows := make(map[int64][]PlanningSlot)
	details := make(map[int64]PlanningTeacherChoice)
	approved := make(map[int64]bool)
	for _, level := range snap.Levels {
		if level.Level == group.level {
			approved[level.GuruID] = true
		}
	}
	for _, window := range snap.Windows {
		if !planningGenderTeacherOK(window.Gender, group.gender) || !approved[window.GuruID] || snap.UnsafeTeacher[window.GuruID] {
			continue
		}
		windows[window.GuruID] = append(windows[window.GuruID], PlanningSlot{
			Day: int(window.Day), Start: window.Start, End: window.End,
		})
		details[window.GuruID] = PlanningTeacherChoice{GuruID: window.GuruID, Name: window.GuruName, Gender: window.Gender, Status: window.Status}
	}
	var choices []PlanningTeacherChoice
	for id, slots := range windows {
		busy := occupied[id]
		valid := true
		for _, requested := range group.slots {
			if !planningContained(requested, slots) {
				valid = false
				break
			}
			for _, taken := range busy {
				_, _, ok := planningInterval(taken)
				// Malformed source intervals cannot safely establish freedom.
				if !ok || planningOverlaps(requested, taken) {
					valid = false
					break
				}
			}
			if !valid {
				break
			}
		}
		if !valid {
			continue
		}
		choice := details[id]
		choice.WeeklyMinutes = len(busy) * planningLessonMinutes
		if limit, ok := snap.MaxWorkload[id]; ok && (limit < 0 || int64(choice.WeeklyMinutes+len(group.slots)*planningLessonMinutes) > limit) {
			continue
		}
		choices = append(choices, choice)
	}
	sort.Slice(choices, func(i, j int) bool {
		si, sj := planningTeacherRank(choices[i], group), planningTeacherRank(choices[j], group)
		if si != sj {
			return si > sj
		}
		return choices[i].GuruID < choices[j].GuruID
	})
	if len(choices) > 3 {
		choices = choices[:3]
	}
	return choices
}

// PlanMatchingPreview is pure and deterministic given the same snapshot.
// A class with started meetings is excluded by the reader; no approval,
// candidate reservation, or final assignment is implied by this preview.
func PlanMatchingPreview(asOf string, snap PlanningMatchSnapshot) PlanningMatchResult {
	result := PlanningMatchResult{
		AsOf: asOf, ConfirmedDemand: len(snap.Demand), Suggestions: make([]PlanningMatchOption, 0),
		Warnings: make([]string, 0),
	}
	if len(snap.Demand) > 500 {
		result.Truncated = true
		result.Warnings = append(result.Warnings, "Demand melebihi batas pembacaan 500; hasil preview belum lengkap.")
		snap.Demand = snap.Demand[:500]
	}
	groups := make(map[string]*planningDemandGroup)
	seen := make(map[int64]bool)
	for _, row := range snap.Demand {
		if row.SantriID < 1 || seen[row.SantriID] || row.Gender != "L" && row.Gender != "P" ||
			strings.TrimSpace(row.Level) == "" || strings.TrimSpace(row.Angkatan) == "" {
			result.SkippedAmbiguous++
			continue
		}
		seen[row.SantriID] = true
		slots, frequency, ok := planningOfferSlots(row)
		if !ok {
			result.SkippedAmbiguous++
			continue
		}
		key := fmt.Sprintf("%q|%q|%q|%d|%q|%q|%q", row.Angkatan, row.Gender, row.Level,
			frequency, row.OfferCode, row.OfferDays, row.OfferStart)
		g, exists := groups[key]
		if !exists {
			g = &planningDemandGroup{key: key, angkatan: row.Angkatan, gender: row.Gender,
				level: row.Level, frequency: frequency, offer: row.OfferCode, slots: slots}
			groups[key] = g
		}
		g.ids = append(g.ids, row.SantriID)
		result.EligibleDemand++
	}
	if result.SkippedAmbiguous > 0 {
		result.Warnings = append(result.Warnings, "Sebagian jadwal pilihan tidak bisa dinormalisasi persis menjadi sesi 90 menit atau frekuensi cocok; perlu verifikasi manual.")
	}
	if result.ConfirmedDemand == 0 {
		result.Warnings = append(result.Warnings, "Belum ada demand terverifikasi dengan jadwal penawaran terstruktur dan tanpa penempatan kelas.")
	}
	ordered := make([]*planningDemandGroup, 0, len(groups))
	for _, g := range groups {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if len(ordered[i].ids) != len(ordered[j].ids) {
			return len(ordered[i].ids) > len(ordered[j].ids)
		}
		return ordered[i].key < ordered[j].key
	})
	classSlots := make(map[int64][]queries.PlanningClassSlot)
	for _, s := range snap.ClassSlots {
		classSlots[s.KelasID] = append(classSlots[s.KelasID], s)
	}
	occupied := make(map[int64][]PlanningSlot)
	for _, b := range snap.Busy {
		occupied[b.GuruID] = append(occupied[b.GuruID], PlanningSlot{
			Day: int(b.Day), Start: b.Start, End: b.End,
		})
	}
	classes := append([]queries.PlanningExistingClass(nil), snap.Classes...)
	sort.Slice(classes, func(i, j int) bool {
		if classes[i].Occupancy != classes[j].Occupancy {
			return classes[i].Occupancy > classes[j].Occupancy
		}
		return classes[i].ID < classes[j].ID
	})
	// Enforce one physical capacity across all primary offer groups sharing a class.
	primarySeatsReserved := make(map[int64]int)
	const maxOptions = 40
	for _, g := range ordered {
		remaining := append([]int64(nil), g.ids...)
		for _, k := range classes {
			if len(result.Suggestions) >= maxOptions {
				result.Truncated = true
				break
			}
			if len(remaining) == 0 {
				break
			}
			if k.Angkatan != g.angkatan || k.Gender != g.gender || k.Level != g.level ||
				planningFrequency(k.Frequency) != g.frequency ||
				!planningGenderTeacherOK(k.TeacherGender, g.gender) ||
				!planningSameSlots(g.slots, classSlots[k.ID]) {
				continue
			}
			space := int(k.Capacity)
			if space > 10 {
				space = 10
			}
			space -= int(k.Occupancy) + primarySeatsReserved[k.ID]
			if space <= 0 {
				continue
			}
			n := space
			if n > len(remaining) {
				n = len(remaining)
			}
			result.Suggestions = append(result.Suggestions, PlanningMatchOption{
				Kind: "existing_class", State: "ready_preview", ClassID: k.ID,
				Angkatan: g.angkatan, Gender: g.gender, Level: g.level,
				Frequency: g.frequency, OfferCode: g.offer,
				CandidateIDs:   append([]int64(nil), remaining[:n]...),
				SeatsAvailable: space, MinimumToStart: 8, Slots: g.slots,
				Reason: "Kelas existing cocok secara gender, level, angkatan, frekuensi, jadwal rutin dan belum memiliki pertemuan.",
			})
			primarySeatsReserved[k.ID] += n
			remaining = remaining[n:]
		}
		for len(remaining) > 0 && len(result.Suggestions) < maxOptions {
			n := len(remaining)
			if n > 10 {
				n = 10
			}
			// E.g. 16 -> 8+8 instead of 10+6 when two classes could meet quota.
			if len(remaining) > 10 && len(remaining)-n > 0 && len(remaining)-n < 8 && len(remaining) >= 16 && len(remaining) <= 18 {
				n = len(remaining) - 8
			}
			option := PlanningMatchOption{
				Kind: "new_class", State: "needs_quota",
				Angkatan: g.angkatan, Gender: g.gender, Level: g.level,
				Frequency: g.frequency, OfferCode: g.offer,
				CandidateIDs:   append([]int64(nil), remaining[:n]...),
				SeatsAvailable: 10, MinimumToStart: 8, Slots: g.slots,
				Reason: "Belum ada kelas existing yang cocok dan memiliki kursi.",
			}
			teacherChoices := planningChoices(*g, snap, occupied)
			option.TeacherOptions = teacherChoices
			if n >= 8 {
				if len(teacherChoices) > 0 {
					option.State = "ready_preview"
					option.Reason = "Demand terverifikasi minimal 8; guru memiliki availability terkonfirmasi, kompetensi level dan tidak bentrok berdasarkan jadwal rutin yang diketahui."
					winner := teacherChoices[0].GuruID
					occupied[winner] = append(occupied[winner], g.slots...)
				} else {
					option.State = "needs_teacher"
					option.Reason = "Demand cukup, tetapi tidak ada guru yang lolos cek availability, kompetensi, beban dan konflik jadwal."
				}
			} else {
				option.Reason = "Demand terverifikasi kurang dari 8; jangan aktivasi tanpa exception yang disetujui."
			}
			result.Suggestions = append(result.Suggestions, option)
			remaining = remaining[n:]
		}
		if len(remaining) > 0 {
			result.Truncated = true
		}
	}
	if result.Truncated {
		result.Warnings = append(result.Warnings, "Preview dibatasi; jangan menganggap seluruh demand sudah teralokasi.")
	}
	result.Warnings = append(result.Warnings,
		"Preview hanya mencakup pembayaran CRM terverifikasi. Data pembayaran manual, alternatif jadwal, kelas multi-level, perubahan setelah snapshot dan histori retensi belum masuk.",
		"Jam kelas existing diasumsikan 90 menit; keputusan penempatan tetap memerlukan konfirmasi guru, approval Admin Kelas dan validasi ulang.")
	return appendPlanningAlternativeMatches(result, snap)
}
