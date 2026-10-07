package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKelasServiceSetKapasitas(t *testing.T) {
	f := setupKelasPerubahan(t)
	service := NewKelasService(f.querier)

	require.NoError(t, service.SetKapasitas(f.kelasID, 30))

	kelas, err := f.querier.GetKelasByID(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 30, kelas.Kapasitas)
	require.EqualValues(t, 2, kelas.JumlahSantri)
}

func TestKelasServiceSetKapasitasRejectsBelowOccupiedSeats(t *testing.T) {
	f := setupKelasPerubahan(t)
	service := NewKelasService(f.querier)

	err := service.SetKapasitas(f.kelasID, 1)

	require.EqualError(t, err, "kapasitas tidak boleh lebih kecil dari jumlah santri aktif/cuti saat ini (2)")
}

func TestKelasServiceSetKapasitasRejectsNonPositiveValue(t *testing.T) {
	f := setupKelasPerubahan(t)
	service := NewKelasService(f.querier)

	err := service.SetKapasitas(f.kelasID, 0)

	require.EqualError(t, err, "kapasitas minimal 1 peserta")
}
