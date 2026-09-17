package services

import (
	"math"
	"testing"
)

// TestKategoriSkorDinaRiska reproduces the PRD fixture: Kompetensi has 6
// indicators, the 6th is empty, so the average is computed from 5 filled
// indicators (each 92%) → 92% × 40% = 36.80.
func TestKategoriSkorDinaRiska(t *testing.T) {
	rata, skor := kategoriSkor(40, []float64{92, 92, 92, 92, 92})
	if rata == nil || math.Abs(*rata-92) > 1e-9 {
		t.Fatalf("expected rata-rata 92, got %v", rata)
	}
	if math.Abs(skor-36.8) > 1e-9 {
		t.Fatalf("expected skor 36.80, got %v", skor)
	}
}

// TestKategoriSkorEmptyExcluded confirms empty indicators are dropped, not
// counted as zero: [90, 80] with weight 30 → avg 85 → 25.5, regardless of how
// many indicators the category nominally has.
func TestKategoriSkorEmptyExcluded(t *testing.T) {
	rata, skor := kategoriSkor(30, []float64{90, 80})
	if rata == nil || math.Abs(*rata-85) > 1e-9 {
		t.Fatalf("expected rata-rata 85, got %v", rata)
	}
	if math.Abs(skor-25.5) > 1e-9 {
		t.Fatalf("expected skor 25.5, got %v", skor)
	}
}

// TestKategoriSkorAllEmpty: no filled indicators → nil average, zero score.
func TestKategoriSkorAllEmpty(t *testing.T) {
	rata, skor := kategoriSkor(20, nil)
	if rata != nil {
		t.Fatalf("expected nil rata-rata, got %v", *rata)
	}
	if skor != 0 {
		t.Fatalf("expected skor 0, got %v", skor)
	}
}

func TestPredikat(t *testing.T) {
	cases := map[float64]string{95: "Sangat Baik", 90: "Sangat Baik", 85: "Baik", 80: "Baik", 75: "Cukup", 70: "Cukup", 69.9: "Perlu Pembinaan", 0: "Perlu Pembinaan"}
	for total, want := range cases {
		if got := Predikat(total); got != want {
			t.Errorf("Predikat(%v) = %q, want %q", total, got, want)
		}
	}
}

func TestTotalIsSumOfCategories(t *testing.T) {
	_, komp := kategoriSkor(40, []float64{92, 92, 92, 92, 92}) // 36.8
	_, kep := kategoriSkor(30, []float64{100})                 // 30.0
	_, dis := kategoriSkor(20, []float64{50})                  // 10.0
	_, kon := kategoriSkor(10, []float64{100})                 // 10.0
	total := komp + kep + dis + kon
	if math.Abs(total-86.8) > 1e-9 {
		t.Fatalf("expected total 86.8, got %v", total)
	}
	if Predikat(total) != "Baik" {
		t.Fatalf("expected Baik, got %q", Predikat(total))
	}
}
