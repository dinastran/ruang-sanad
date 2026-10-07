package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// Meetings started through the app store tanggal as a full driver timestamp
// (e.g. 2026-10-07T15:13:58.031948647Z), which SQLite's date() cannot parse.
func TestListRiwayatMengajarMenampilkanPertemuanDariAplikasi(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	_, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{KonfirmasiTambahan: true})
	require.NoError(t, err)

	tanggal := startOfToday().Format("2006-01-02")
	guruID := f.guruUtamaID
	items, summary, err := NewGuruService(f.querier).ListRiwayatMengajar(&guruID, tanggal, tanggal)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, f.kelasID, items[0].KelasID)
	require.Equal(t, tanggal, items[0].Tanggal)
	require.EqualValues(t, 1, summary.KelasDiajar)
}
