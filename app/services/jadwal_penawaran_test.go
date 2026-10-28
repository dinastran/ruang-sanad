package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestJadwalLabel(t *testing.T) {
	require.Equal(t, "Selasa 19.30–21.00 WIB", JadwalLabel("2", "19:30", "21:00"))
	require.Equal(t, "Rabu & Sabtu 19.30–21.00 WIB", JadwalLabel("6,3", "19:30", "21:00"))
	require.Equal(t, "Senin, Rabu & Ahad 05.30–07.00 WIB", JadwalLabel("1,3,7", "05:30", "07:00"))
}

func TestJadwalSeededAndValidated(t *testing.T) {
	f := setupLeadFixture(t)
	svc := NewJadwalPenawaranService(f.querier)
	list, err := svc.List(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 4)
	require.Equal(t, "selasa-malam", list[0].Kode)
	require.Equal(t, "2", list[0].Level)
	require.Equal(t, 2, list[3].Frekuensi)
	require.Equal(t, "Rabu & Sabtu 19.30–21.00 WIB", list[3].Label)
	require.EqualValues(t, 10, list[0].SisaIkhwan)

	bad := []models.JadwalPenawaranRequest{
		{Kode: "Bad Kode", Hari: "1", JamMulai: "19:30", JamSelesai: "21:00", KuotaPerGender: 10},
		{Kode: "senin", Hari: "8", JamMulai: "19:30", JamSelesai: "21:00", KuotaPerGender: 10},
		{Kode: "senin", Hari: "1", JamMulai: "21:00", JamSelesai: "19:30", KuotaPerGender: 10},
		{Kode: "senin", Hari: "1", JamMulai: "19:30", JamSelesai: "21:00", Level: "3", KuotaPerGender: 10},
		{Kode: "senin", Hari: "1", JamMulai: "19:30", JamSelesai: "21:00", KuotaPerGender: 0},
		{Kode: "selasa-malam", Hari: "2", JamMulai: "19:30", JamSelesai: "21:00", KuotaPerGender: 10},
	}
	for _, req := range bad {
		require.Errorf(t, svc.Create(t.Context(), req), "%+v", req)
	}
	require.NoError(t, svc.Create(t.Context(), models.JadwalPenawaranRequest{
		Kode: "senin-pagi", Hari: "1", JamMulai: "06:00", JamSelesai: "07:30", KuotaPerGender: 10, IsBuka: false,
	}))
	open, err := svc.Open(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, open, 4, "a closed schedule is not offered")
}

func TestJadwalSelectedByAIAndQuotaCountsOnlyVerified(t *testing.T) {
	f := setupLeadFixture(t)
	svc := NewJadwalPenawaranService(f.querier)
	svc.now = func() time.Time { return f.clock }
	// Seeded rows start their period at migration time (real now).
	_, err := f.db.Exec(`UPDATE jadwal_penawaran SET periode_mulai = ?`, f.clock.Add(-24*time.Hour))
	require.NoError(t, err)

	res := f.ingest(t,
		leadEvent("evt_j1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "W1", "body": "mau yang selasa"}),
		leadEvent("evt_j2", "schedule.selected", "2026-10-24T03:01:00Z", map[string]string{"jadwal_kode": "selasa-malam", "wa_message_id": "W2"}),
		leadEvent("evt_j3", "schedule.selected", "2026-10-24T03:02:00Z", map[string]string{"jadwal_kode": "tidak-ada"}),
		leadEvent("evt_j4", "schedule.selected", "2026-10-24T03:02:00Z", map[string]string{"jadwal_kode": "BAD KODE"}),
	)
	require.Equal(t, 3, res.Accepted)
	require.Len(t, res.Rejected, 1)
	l := f.lead(t)
	require.Equal(t, "selasa-malam", l.JadwalKode, "unknown kode is ignored, not applied")

	// Chosen but not paid: no seat taken.
	list, err := svc.List(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 10, list[0].SisaAkhwat)

	_, err = f.db.Exec(`UPDATE leads SET jenis_kelamin = 'P', verified_at = ?, stage = 'terverifikasi' WHERE id = ?`, f.clock.Add(time.Hour), l.ID)
	require.NoError(t, err)
	list, err = svc.List(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 9, list[0].SisaAkhwat)
	require.EqualValues(t, 10, list[0].SisaIkhwan)
	require.EqualValues(t, 1, list[0].TerisiAkhwat)

	// After verification the AI can no longer move the seat.
	f.ingest(t, leadEvent("evt_j5", "schedule.selected", "2026-10-24T06:00:00Z", map[string]string{"jadwal_kode": "jumat-malam"}))
	require.Equal(t, "selasa-malam", f.lead(t).JadwalKode)

	// A new period starts the count again.
	svc.now = func() time.Time { return f.clock.Add(2 * time.Hour) }
	require.NoError(t, svc.ResetPeriode(t.Context(), list[0].ID))
	list, err = svc.List(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 10, list[0].SisaAkhwat)

	// A chosen schedule cannot be deleted, only closed.
	require.Error(t, svc.Delete(t.Context(), list[0].ID))
	require.NoError(t, svc.Delete(t.Context(), list[1].ID))
}

func TestLeadUpdateDataJadwalKode(t *testing.T) {
	f := setupLeadFixture(t)
	f.ingest(t, leadEvent("evt_u1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "U1", "body": "halo"}))
	l := f.lead(t)
	base := models.LeadUpdateRequest{Nama: "Fulanah"}

	require.NoError(t, f.service.UpdateData(t.Context(), l.ID, f.csID, base))
	require.Equal(t, "", f.lead(t).JadwalKode, "nil leaves it unchanged")

	kode := "ahad-pagi"
	base.JadwalKode = &kode
	require.NoError(t, f.service.UpdateData(t.Context(), l.ID, f.csID, base))
	require.Equal(t, "ahad-pagi", f.lead(t).JadwalKode)

	d, err := f.service.Detail(t.Context(), l.ID)
	require.NoError(t, err)
	require.Equal(t, "Ahad 05.30–07.00 WIB", d.JadwalLabel)
	require.Len(t, d.JadwalOptions, 4)

	base.JadwalKode = nil
	require.NoError(t, f.service.UpdateData(t.Context(), l.ID, f.csID, base))
	require.Equal(t, "ahad-pagi", f.lead(t).JadwalKode)

	wrong := "tidak-ada"
	base.JadwalKode = &wrong
	require.Error(t, f.service.UpdateData(t.Context(), l.ID, f.csID, base))
}

func TestJadwalGenderKhusus(t *testing.T) {
	f := setupLeadFixture(t)
	svc := NewJadwalPenawaranService(f.querier)
	svc.now = func() time.Time { return f.clock }

	require.Error(t, svc.Create(t.Context(), models.JadwalPenawaranRequest{
		Kode: "kamis-x", Hari: "4", JamMulai: "19:30", JamSelesai: "21:00", Gender: "X", KuotaPerGender: 10,
	}), "gender must be L, P, or empty")
	for _, req := range []models.JadwalPenawaranRequest{
		{Kode: "kamis-ikhwan", Hari: "4", JamMulai: "19:30", JamSelesai: "21:00", Gender: "l", KuotaPerGender: 8, IsBuka: true},
		{Kode: "kamis-akhwat", Hari: "4", JamMulai: "20:00", JamSelesai: "21:30", Gender: "P", KuotaPerGender: 12, IsBuka: true},
	} {
		require.NoError(t, svc.Create(t.Context(), req))
	}
	byKode := func(list []models.JadwalPenawaranResponse) map[string]models.JadwalPenawaranResponse {
		out := map[string]models.JadwalPenawaranResponse{}
		for _, j := range list {
			out[j.Kode] = j
		}
		return out
	}
	list, err := svc.List(t.Context())
	require.NoError(t, err)
	all := byKode(list)
	ikhwan := all["kamis-ikhwan"]
	require.Equal(t, "L", ikhwan.Gender)
	require.Equal(t, "Ikhwan", ikhwan.Untuk)
	require.EqualValues(t, 8, ikhwan.SisaIkhwan)
	require.Zero(t, ikhwan.SisaAkhwat, "no seats for akhwat on an ikhwan schedule")
	require.Equal(t, "Ikhwan & Akhwat", all["selasa-malam"].Untuk, "existing schedules stay shared")

	openL, err := svc.Open(t.Context(), "L")
	require.NoError(t, err)
	require.Contains(t, byKode(openL), "kamis-ikhwan")
	require.NotContains(t, byKode(openL), "kamis-akhwat")
	require.Contains(t, byKode(openL), "selasa-malam")
	openAll, err := svc.Open(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, openAll, 6)

	// CS form: a schedule for the other gender is rejected.
	f.ingest(t, leadEvent("evt_g1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "G1", "body": "halo"}))
	l := f.lead(t)
	akhwat := "kamis-akhwat"
	err = f.service.UpdateData(t.Context(), l.ID, f.csID, models.LeadUpdateRequest{Nama: "Fulan", JenisKelamin: "L", JadwalKode: &akhwat})
	require.ErrorContains(t, err, "khusus akhwat")
	ikh := "kamis-ikhwan"
	require.NoError(t, f.service.UpdateData(t.Context(), l.ID, f.csID, models.LeadUpdateRequest{Nama: "Fulan", JenisKelamin: "L", JadwalKode: &ikh}))
	// Changing only the gender to a mismatch is rejected too.
	require.Error(t, f.service.UpdateData(t.Context(), l.ID, f.csID, models.LeadUpdateRequest{Nama: "Fulan", JenisKelamin: "P"}))
	d, err := f.service.Detail(t.Context(), l.ID)
	require.NoError(t, err)
	require.Equal(t, "Kamis 19.30–21.00 WIB · khusus ikhwan", d.JadwalLabel)

	// CS AI: the choice is still recorded, but the lead goes to review.
	_, err = f.db.Exec(`UPDATE leads SET needs_review = 0 WHERE id = ?`, l.ID)
	require.NoError(t, err)
	f.ingest(t, leadEvent("evt_g2", "schedule.selected", "2026-10-24T03:05:00Z", map[string]string{"jadwal_kode": "kamis-akhwat"}))
	got := f.lead(t)
	require.Equal(t, "kamis-akhwat", got.JadwalKode)
	require.EqualValues(t, 1, got.NeedsReview)

	// A verified lead of the other gender is reported, not counted in quota.
	_, err = f.db.Exec(`UPDATE leads SET verified_at = ?, stage = 'terverifikasi' WHERE id = ?`, f.clock.Add(time.Hour), l.ID)
	require.NoError(t, err)
	list, err = svc.List(t.Context())
	require.NoError(t, err)
	k := byKode(list)["kamis-akhwat"]
	require.EqualValues(t, 1, k.TidakSesuai)
	require.EqualValues(t, 12, k.SisaAkhwat)
}
