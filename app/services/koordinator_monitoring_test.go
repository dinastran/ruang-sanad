package services

import (
	"context"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestKoordinatorMonitoringMenandaiJadwalTerlewat(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err := f.db.Exec(`INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai, status) VALUES (?, ?, '08:00', 'dijadwalkan')`, f.kelasID, yesterday)
	require.NoError(t, err)

	service := NewKoordinatorMonitoringService(f.querier, f.service)
	result, err := service.List(models.ClassMonitoringFilter{StartDate: yesterday, EndDate: yesterday})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "tidak_terlaksana", result.Items[0].Status)
	require.Equal(t, "belum_hadir", result.Items[0].TeacherAttendance)
	require.True(t, result.Items[0].NeedsAction)
	require.True(t, result.Items[0].JatuhTempo)
	require.EqualValues(t, 1, result.Summary.Total)
	require.EqualValues(t, 0, result.Summary.BelumMulai)
	require.EqualValues(t, 1, result.Summary.PerluTindakan)
	require.EqualValues(t, 1, result.Summary.TotalJadwal)
	require.EqualValues(t, 1, result.Summary.JatuhTempo)
	require.EqualValues(t, 1, result.Summary.TidakTerlaksana)
	require.Equal(t, float64(0), result.Summary.KepatuhanJadwal)
}

func TestKoordinatorMonitoringTetapMenandaiTerlambatPadaHariYangSama(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	now := time.Now().In(wib)
	if now.Hour() == 0 && now.Minute() < 2 {
		t.Skip("test membutuhkan waktu lewat 00:01 WIB")
	}
	today := now.Format("2006-01-02")
	pastTime := now.Add(-time.Minute).Format("15:04")
	_, err := f.db.Exec(`INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai, status) VALUES (?, ?, ?, 'dijadwalkan')`, f.kelasID, today, pastTime)
	require.NoError(t, err)

	service := NewKoordinatorMonitoringService(f.querier, f.service)
	result, err := service.List(models.ClassMonitoringFilter{StartDate: today, EndDate: today})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "terlambat", result.Items[0].Status)
	require.True(t, result.Items[0].JatuhTempo)
	require.EqualValues(t, 1, result.Summary.BelumMulai)
	require.EqualValues(t, 0, result.Summary.TidakTerlaksana)
}

func TestKoordinatorMonitoringMengecualikanSesiTambahanDariKPI(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	today := time.Now().In(wib).Format("2006-01-02")
	_, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID: f.kelasID,
		Tanggal: today,
		JamMulai: "23:59",
		Catatan: "Sesi tambahan",
	})
	require.NoError(t, err)

	service := NewKoordinatorMonitoringService(f.querier, f.service)
	result, err := service.List(models.ClassMonitoringFilter{StartDate: today, EndDate: today})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.True(t, result.Items[0].IsTambahan)
	require.EqualValues(t, 1, result.Summary.Total)
	require.EqualValues(t, 0, result.Summary.TotalJadwal)
	require.EqualValues(t, 0, result.Summary.JatuhTempo)
	require.Equal(t, float64(0), result.Summary.KepatuhanJadwal)
}

func TestKoordinatorMonitoringMengirimPengingatDanMencatatTindakLanjut(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	scheduleID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID:  f.kelasID,
		Tanggal:  tomorrow,
		JamMulai: "09:00",
	})
	require.NoError(t, err)

	service := NewKoordinatorMonitoringService(f.querier, f.service)
	require.NoError(t, service.SendReminder(f.guruLainUser, scheduleID, f.kelasID, models.MonitoringReminderRequest{Kind: "presensi_guru"}))
	require.NoError(t, service.AddNote(f.guruLainUser, scheduleID, f.kelasID, models.MonitoringNoteRequest{Note: "Hubungi guru bila belum check-in."}))

	notifications, err := NewNotificationService(f.querier).ListForUser(f.guruUtamaUser)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, "presensi_guru", notifications[0].Type)
	require.False(t, notifications[0].Read)

	result, err := service.List(models.ClassMonitoringFilter{StartDate: tomorrow, EndDate: tomorrow})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Len(t, result.Items[0].Notes, 1)
	require.Equal(t, "Hubungi guru bila belum check-in.", result.Items[0].Notes[0].Note)
	require.Len(t, result.Items[0].Activities, 1)
	require.Equal(t, "pengingat_dikirim", result.Items[0].Activities[0].Action)

	require.NoError(t, NewNotificationService(f.querier).MarkRead(f.guruUtamaUser, notifications[0].ID))
	stored, err := f.querier.ListNotificationsForUser(context.Background(), f.guruUtamaUser)
	require.NoError(t, err)
	require.True(t, stored[0].ReadAt.Valid)
}

func TestKoordinatorMonitoringMemvalidasiRentangDanJenisPengingat(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	service := NewKoordinatorMonitoringService(f.querier, f.service)

	_, err := service.List(models.ClassMonitoringFilter{StartDate: "2026-02-02", EndDate: "2026-02-01"})
	require.EqualError(t, err, "tanggal akhir tidak boleh sebelum tanggal awal")

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	scheduleID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{KelasID: f.kelasID, Tanggal: tomorrow, JamMulai: "09:00"})
	require.NoError(t, err)
	err = service.SendReminder(f.guruLainUser, scheduleID, f.kelasID, models.MonitoringReminderRequest{Kind: "tidak_valid"})
	require.EqualError(t, err, "jenis pengingat tidak valid")
}
