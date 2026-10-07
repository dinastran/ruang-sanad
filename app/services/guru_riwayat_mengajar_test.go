package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
)

func TestSummarizeRiwayatMengajar(t *testing.T) {
	items := []models.RiwayatMengajarItem{
		{ID: 1, KelasID: 10, Status: "selesai"},
		{ID: 2, KelasID: 10, Status: "berlangsung"},
		{ID: 3, KelasID: 11, Status: "selesai"},
	}

	got := summarizeRiwayatMengajar(items)
	if got.TotalDimulai != 3 {
		t.Fatalf("TotalDimulai = %d, want 3", got.TotalDimulai)
	}
	if got.Selesai != 2 {
		t.Fatalf("Selesai = %d, want 2", got.Selesai)
	}
	if got.BelumSelesai != 1 {
		t.Fatalf("BelumSelesai = %d, want 1", got.BelumSelesai)
	}
	if got.KelasDiajar != 2 {
		t.Fatalf("KelasDiajar = %d, want 2", got.KelasDiajar)
	}
}
