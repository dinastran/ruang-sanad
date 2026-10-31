package services

import (
	"sort"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// PlanningAlternativeMatch is an optional, not-yet-activated placement choice.
// It is NOT incremental demand and never reserves a class seat.
type PlanningAlternativeMatch struct {
	SantriID  int64          `json:"santri_id"`
	ClassID   int64          `json:"class_id"`
	OfferCode string         `json:"offer_code"`
	Slots     []PlanningSlot `json:"slots"`
	Reason    string         `json:"reason"`
}

// appendPlanningAlternativeMatches is deterministic and mutation-free.
// Original verified preferences take precedence over approved alternatives.
func appendPlanningAlternativeMatches(result PlanningMatchResult, snap PlanningMatchSnapshot) PlanningMatchResult {
	result.AlternativeMatches = make([]PlanningAlternativeMatch, 0)
	if len(snap.ApprovedAlternatives) == 0 {
		return result
	}
	const maxAlternates = 80
	if len(snap.ApprovedAlternatives) > 1000 {
		result.Truncated = true
		result.Warnings = append(result.Warnings, "Alternatif yang disetujui melebihi batas pembacaan; hasil bukan alokasi lengkap.")
	}
	primary := make(map[int64]queries.PlanningDemand)
	for _, d := range snap.Demand {
		if _, exists := primary[d.SantriID]; !exists {
			primary[d.SantriID] = d
		}
	}
	// Existing primary-class previews and new-class previews that pass all
	// their checks consume their students before alternatives are considered.
	committedPreview := make(map[int64]bool)
	seats := make(map[int64]int)
	for _, k := range snap.Classes {
		space := int(k.Capacity)
		if space > 10 {
			space = 10
		}
		space -= int(k.Occupancy)
		if space < 0 {
			space = 0
		}
		seats[k.ID] = space
	}
	for _, opt := range result.Suggestions {
		if opt.State != "ready_preview" {
			continue
		}
		for _, id := range opt.CandidateIDs {
			committedPreview[id] = true
		}
		if opt.Kind == "existing_class" {
			remaining := seats[opt.ClassID] - len(opt.CandidateIDs)
			if remaining < 0 {
				remaining = 0
			}
			seats[opt.ClassID] = remaining
		}
	}
	classSlots := make(map[int64][]queries.PlanningClassSlot)
	for _, s := range snap.ClassSlots {
		classSlots[s.KelasID] = append(classSlots[s.KelasID], s)
	}
	classes := append([]queries.PlanningExistingClass(nil), snap.Classes...)
	sort.Slice(classes, func(i, j int) bool {
		if classes[i].Occupancy != classes[j].Occupancy {
			return classes[i].Occupancy > classes[j].Occupancy
		}
		return classes[i].ID < classes[j].ID
	})
	alts := append([]queries.PlanningApprovedAlternative(nil), snap.ApprovedAlternatives...)
	sort.Slice(alts, func(i, j int) bool {
		if alts[i].SantriID != alts[j].SantriID {
			return alts[i].SantriID < alts[j].SantriID
		}
		return alts[i].OfferCode < alts[j].OfferCode
	})
	moved := make(map[int64]bool)
	for _, a := range alts {
		if len(result.AlternativeMatches) >= maxAlternates {
			result.Truncated = true
			break
		}
		base, ok := primary[a.SantriID]
		if !ok || committedPreview[a.SantriID] || moved[a.SantriID] || a.OfferCode == base.OfferCode {
			continue
		}
		if _, _, originalValid := planningOfferSlots(base); !originalValid {
			continue
		}
		// Source query validates consent, active offer and payment. The pure
		// preview additionally validates exact duration/frequency/shape.
		want := base
		want.OfferCode = a.OfferCode
		want.OfferDays = a.OfferDays
		want.OfferStart = a.OfferStart
		want.OfferEnd = a.OfferEnd
		slots, frequency, valid := planningOfferSlots(want)
		if !valid {
			continue
		}
		for _, k := range classes {
			if seats[k.ID] <= 0 || k.Angkatan != base.Angkatan || k.Gender != base.Gender ||
				k.Level != base.Level || planningFrequency(k.Frequency) != frequency ||
				!planningGenderTeacherOK(k.TeacherGender, base.Gender) ||
				!planningSameSlots(slots, classSlots[k.ID]) {
				continue
			}
			result.AlternativeMatches = append(result.AlternativeMatches, PlanningAlternativeMatch{
				SantriID: a.SantriID, ClassID: k.ID, OfferCode: a.OfferCode, Slots: slots,
				Reason: "Alternatif jadwal tercatat disetujui santri; kelas existing cocok dan memiliki kursi pada snapshot.",
			})
			seats[k.ID]--
			moved[a.SantriID] = true
			break
		}
	}
	// Remove these students from speculative/blocked primary options, so
	// every candidate has at most one visible placement preview.
	if len(moved) > 0 {
		filtered := make([]PlanningMatchOption, 0, len(result.Suggestions))
		for _, option := range result.Suggestions {
			if option.State == "ready_preview" {
				filtered = append(filtered, option)
				continue
			}
			ids := make([]int64, 0, len(option.CandidateIDs))
			for _, id := range option.CandidateIDs {
				if !moved[id] {
					ids = append(ids, id)
				}
			}
			if len(ids) == 0 {
				continue
			}
			option.CandidateIDs = ids
			if option.Kind == "new_class" && len(ids) < 8 {
				option.State = "needs_quota"
				option.Reason = "Sisa demand utama kurang dari 8 setelah alternatif yang disetujui cocok ke kelas existing."
				option.TeacherOptions = nil
			}
			filtered = append(filtered, option)
		}
		result.Suggestions = filtered
	}
	if len(result.AlternativeMatches) > 0 {
		result.Warnings = append(result.Warnings,
			"Rekomendasi alternatif menggunakan persetujuan santri tercatat. Jumlahnya bukan demand tambahan dan harus diverifikasi ulang sebelum penempatan.")
	}
	if result.Truncated {
		result.Warnings = append(result.Warnings, "Alternatif ditampilkan sebagian; jangan gunakan hasil sebagai alokasi final.")
	}
	return result
}
