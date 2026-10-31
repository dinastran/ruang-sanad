package services

import (
	"fmt"
	"strings"
	"time"
)

// PlanningSlot uses local WIB wall-clock times. A slot never crosses midnight.
// Day: 1=Senin .. 7=Ahad. All class meetings are 90 minutes.
type PlanningSlot struct {
	Day   int
	Start string // HH:MM
	End   string // HH:MM
}

// PlanningTeacher is a validated read model, not an instruction to mutate guru.
// ApprovedLevels must come from coordinator-approved teaching eligibility.
type PlanningTeacher struct {
	ID                   int64
	Gender               string // L or P
	Active               bool
	ApprovedLevels       map[string]bool
	Available            []PlanningSlot
	Occupied             []PlanningSlot
	CurrentWeeklyMinutes int
	MaxWeeklyMinutes     *int // nil until a formal workload limit exists
}

// PlanningCandidate describes a santri's placement eligibility as established
// by the caller from current authoritative data (not inferred from status alone).
type PlanningCandidate struct {
	ID                int64
	Gender            string
	Level             string
	PlacementEligible bool
	ScheduleApproved  bool
	CurrentClassID    int64
}

// PlanningProposal is an immutable input to the activation rule checker.
// A preview may display violations; only an empty violation list can activate.
type PlanningProposal struct {
	Program           string
	Gender            string
	Frequency         int
	Capacity          int
	Slots             []PlanningSlot
	Teacher           PlanningTeacher
	Candidates        []PlanningCandidate
	ExceptionApproved bool // must be obtained from the persisted approval record
}

type PlanningViolation struct {
	Code    string
	Message string
}

const planningLessonMinutes = 90

func planningMinutes(raw string) (int, bool) {
	if len(raw) != 5 || raw[2] != ':' {
		return 0, false
	}
	t, err := time.Parse("15:04", raw)
	if err != nil || t.Format("15:04") != raw {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func planningInterval(slot PlanningSlot) (int, int, bool) {
	start, startOK := planningMinutes(slot.Start)
	end, endOK := planningMinutes(slot.End)
	return start, end, slot.Day >= 1 && slot.Day <= 7 && startOK && endOK && end > start
}

func planningOverlaps(a, b PlanningSlot) bool {
	as, ae, aOK := planningInterval(a)
	bs, be, bOK := planningInterval(b)
	return aOK && bOK && a.Day == b.Day && as < be && bs < ae
}

func planningContained(slot PlanningSlot, windows []PlanningSlot) bool {
	start, end, ok := planningInterval(slot)
	if !ok {
		return false
	}
	for _, window := range windows {
		ws, we, valid := planningInterval(window)
		if valid && window.Day == slot.Day && ws <= start && end <= we {
			return true
		}
	}
	return false
}

// ValidatePlanningActivation is side-effect-free and deterministic. The DB
// transaction must re-read live availability, roster, approval and conflicts
// before calling it; a passed preview never authorizes later activation.
func ValidatePlanningActivation(p PlanningProposal) []PlanningViolation {
	issues := make([]PlanningViolation, 0)
	add := func(code, message string) { issues = append(issues, PlanningViolation{code, message}) }

	if p.Gender != "L" && p.Gender != "P" {
		add("INVALID_GENDER", "Gender kelas wajib L atau P")
	}
	if p.Frequency != 1 && p.Frequency != 2 {
		add("INVALID_FREQUENCY", "Frekuensi kelas harus 1x atau 2x per pekan")
	}
	if p.Capacity < 1 {
		add("INVALID_CAPACITY", "Kapasitas harus positif")
	}
	regular := strings.EqualFold(strings.TrimSpace(p.Program), "reguler") || strings.EqualFold(strings.TrimSpace(p.Program), "regular")
	if regular && (p.Capacity < 8 || p.Capacity > 10) {
		add("REGULAR_CAPACITY", "Kapasitas reguler harus 8 sampai 10")
	}
	if len(p.Candidates) > p.Capacity && p.Capacity > 0 {
		add("CAPACITY_EXCEEDED", "Jumlah kandidat melebihi kapasitas")
	}
	if regular && len(p.Candidates) < 8 && !p.ExceptionApproved {
		add("MIN_QUOTA_NOT_MET", "Reguler memerlukan 8 santri atau exception yang disetujui")
	}

	if p.Teacher.ID <= 0 || !p.Teacher.Active {
		add("TEACHER_INACTIVE", "Guru wajib aktif dan teridentifikasi")
	}
	if p.Teacher.Gender != "L" && p.Teacher.Gender != "P" {
		add("INVALID_TEACHER_GENDER", "Gender guru wajib L atau P")
	} else if p.Teacher.Gender == "P" && p.Gender == "L" {
		add("TEACHER_GENDER_MISMATCH", "Guru akhwat tidak boleh mengajar ikhwan")
	}
	if p.Teacher.MaxWeeklyMinutes != nil {
		if *p.Teacher.MaxWeeklyMinutes < 0 || p.Teacher.CurrentWeeklyMinutes < 0 ||
			p.Teacher.CurrentWeeklyMinutes+p.Frequency*planningLessonMinutes > *p.Teacher.MaxWeeklyMinutes {
			add("TEACHER_WORKLOAD_EXCEEDED", "Beban mingguan guru melampaui batas terkonfirmasi")
		}
	}

	if len(p.Slots) != p.Frequency {
		add("SLOT_COUNT_MISMATCH", "Jumlah sesi harus sesuai frekuensi")
	}
	seenDays := make(map[int]bool, len(p.Slots))
	for i, slot := range p.Slots {
		start, end, ok := planningInterval(slot)
		if !ok {
			add("INVALID_SLOT", fmt.Sprintf("Sesi %d memiliki waktu atau hari tidak valid", i+1))
			continue
		}
		if end-start != planningLessonMinutes {
			add("INVALID_DURATION", fmt.Sprintf("Sesi %d harus tepat 90 menit", i+1))
		}
		if seenDays[slot.Day] {
			add("DUPLICATE_DAY", "Sesi 2x/pekan harus pada hari berbeda")
		}
		seenDays[slot.Day] = true
		if !planningContained(slot, p.Teacher.Available) {
			add("TEACHER_UNAVAILABLE", fmt.Sprintf("Guru tidak tersedia untuk sesi %d", i+1))
		}
		for _, existing := range p.Teacher.Occupied {
			if planningOverlaps(slot, existing) {
				add("SCHEDULE_CONFLICT", fmt.Sprintf("Sesi %d bentrok dengan jadwal guru", i+1))
				break
			}
		}
	}

	seenStudents := make(map[int64]bool, len(p.Candidates))
	for _, student := range p.Candidates {
		if student.ID <= 0 || seenStudents[student.ID] {
			add("DUPLICATE_CANDIDATE", "Identitas santri tidak valid atau berulang")
		}
		seenStudents[student.ID] = true
		if student.Gender != p.Gender {
			add("STUDENT_GENDER_MISMATCH", "Gender santri tidak sesuai kelas")
		}
		if student.Level == "" || !p.Teacher.ApprovedLevels[student.Level] {
			add("LEVEL_NOT_APPROVED", "Guru belum disetujui mengajar level santri")
		}
		if !student.PlacementEligible || student.CurrentClassID != 0 {
			add("STUDENT_NOT_ELIGIBLE", "Santri tidak eligible atau sudah ditempatkan")
		}
		if !student.ScheduleApproved {
			add("SCHEDULE_NOT_APPROVED", "Pilihan jadwal santri belum disetujui")
		}
	}
	return issues
}
