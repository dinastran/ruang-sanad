package services

import (
	"errors"
	"testing"
)

func validGuruAvailabilityInput() GuruAvailabilityInput {
	return GuruAvailabilityInput{
		Hari: 6, JamMulai: "19:30", JamSelesai: "21:00",
		BerlakuMulai: "2026-11-01",
	}
}

func TestGuruAvailabilityValidation(t *testing.T) {
	valid := validGuruAvailabilityInput()
	if err := validateGuruAvailabilityInput(&valid); err != nil {
		t.Fatalf("valid availability rejected: %v", err)
	}
	cases := []struct {
		name string
		edit func(*GuruAvailabilityInput)
	}{
		{"day zero", func(r *GuruAvailabilityInput) { r.Hari = 0 }},
		{"day eight", func(r *GuruAvailabilityInput) { r.Hari = 8 }},
		{"start time invalid", func(r *GuruAvailabilityInput) { r.JamMulai = "25:00" }},
		{"end time invalid", func(r *GuruAvailabilityInput) { r.JamSelesai = "7pm" }},
		{"short availability", func(r *GuruAvailabilityInput) { r.JamSelesai = "20:59" }},
		{"overnight", func(r *GuruAvailabilityInput) { r.JamSelesai = "01:00" }},
		{"invalid starting date", func(r *GuruAvailabilityInput) { r.BerlakuMulai = "2026-02-30" }},
		{"invalid ending date", func(r *GuruAvailabilityInput) { r.BerlakuSampai = "2026-13-01" }},
		{"end before start", func(r *GuruAvailabilityInput) { r.BerlakuSampai = "2026-10-31" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validGuruAvailabilityInput()
			tc.edit(&input)
			if err := validateGuruAvailabilityInput(&input); !errors.Is(err, ErrGuruAvailabilityInvalid) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func TestGuruAvailabilityAllowsLongerBlocks(t *testing.T) {
	input := validGuruAvailabilityInput()
	input.JamSelesai = "23:00"
	input.BerlakuSampai = "2026-12-31"
	if err := validateGuruAvailabilityInput(&input); err != nil {
		t.Fatalf("longer confirmed availability must be supported: %v", err)
	}
}
