package services

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

const testJadwal = "tst1"

// submitFixture: registration fixture + one open jadwal (kuota 1/gender)
// and a link "tahsin-reguler" (short, after_submit per arg).
func setupSubmitFixture(t *testing.T, afterSubmit string) (*registrationFixture, int64) {
	t.Helper()
	f := setupRegistrationFixture(t)
	_, err := f.db.Exec(`INSERT INTO jadwal_penawaran (kode, hari, jam_mulai, jam_selesai, kuota_per_gender, is_buka, periode_mulai, created_at, updated_at)
		VALUES (?, '1,4', '19:30', '20:30', 1, 1, '2026-01-01 00:00:00', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testJadwal)
	require.NoError(t, err)
	req := f.linkReq("tahsin-reguler")
	req.AfterSubmit = afterSubmit
	id, err := f.reg.CreateLink(t.Context(), req, f.adminID)
	require.NoError(t, err)
	return f, id
}

// submit renders a token, advances the clock past the minimum age and submits.
func (f *registrationFixture) submit(t *testing.T, in models.RegistrationSubmit) (*models.RegistrationResult, error) {
	t.Helper()
	in.FormToken = f.reg.FormToken("tahsin-reguler")
	f.clock = f.clock.Add(10 * time.Second)
	return f.reg.Submit(t.Context(), "tahsin-reguler", in, models.RequestMeta{IP: "1.2.3.4", UserAgent: "test"})
}

func validSubmit(phone string) models.RegistrationSubmit {
	return models.RegistrationSubmit{
		Nama: "Fulanah", NoWA: phone, JenisKelamin: "P", Usia: 30, Domisili: "Bandung", JadwalKode: testJadwal,
	}
}

func (f *registrationFixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestSubmitCreatesLead(t *testing.T) {
	f, linkID := setupSubmitFixture(t, "wa")
	res, err := f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	require.Equal(t, "wa", res.Next)

	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.Equal(t, LeadStageDataTerisi, l.Stage)
	require.Equal(t, "Fulanah", l.Nama)
	require.Equal(t, testJadwal, l.JadwalKode)
	require.EqualValues(t, f.sourceID, l.LeadSourceID.Int64)
	require.Equal(t, LeadSourceByForm, l.LeadSourceBy)
	require.EqualValues(t, 1, l.ProgramID.Int64)
	require.Equal(t, "L-"+itoa(l.ID), res.LeadRef)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM lead_events WHERE lead_id = ? AND actor = ? AND json_extract(payload, '$.stage_to') = 'data_terisi'`,
		l.ID, "form:"+itoa(linkID)))
	require.Equal(t, 1, f.count(t, `SELECT COALESCE(SUM(submit), 0) FROM registration_link_stats WHERE link_id = ?`, linkID))

	n, err := f.service.RematchSources(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "rematch tidak menimpa sumber dari form")
}

func TestSubmitPhoneFormatsDedup(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	_, err := f.submit(t, validSubmit("0812-3456-7890"))
	require.NoError(t, err)
	_, err = f.submit(t, validSubmit("+62 812 3456 7890"))
	require.NoError(t, err)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM leads`))
	require.Equal(t, 2, f.count(t, `SELECT COUNT(*) FROM lead_registrations`))
}

func TestSubmitMatchesLeadByPhoneColumn(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, created_at, updated_at) VALUES ('abc@lid', '6281234567890', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM leads`))
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM leads WHERE wa_chat_id = 'abc@lid' AND nama = 'Fulanah'`))
}

func TestSubmitDoesNotOverwrite(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, nama, domisili, created_at, updated_at)
		VALUES (?, '6281234567890', 'Fulan (CS)', 'Jakarta', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testChatID)
	require.NoError(t, err)
	_, err = f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.Equal(t, "Fulan (CS)", l.Nama)
	require.Equal(t, "Jakarta", l.Domisili)
	require.EqualValues(t, 30, l.Usia.Int64, "field kosong diisi")
}

func TestSubmitStageNeverRegresses(t *testing.T) {
	f, _ := setupSubmitFixture(t, "bayar")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, stage, created_at, updated_at)
		VALUES (?, '6281234567890', 'bukti_masuk', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testChatID)
	require.NoError(t, err)
	_, err = f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.Equal(t, LeadStageBuktiMasuk, l.Stage)
}

func TestSubmitBayarSetsAmount(t *testing.T) {
	f, _ := setupSubmitFixture(t, "bayar")
	f.addVarian(t, "1x / pekan", 200000, "1x")
	v2 := f.addVarian(t, "2x / pekan", 350000, "2x")
	in := validSubmit("081234567890")
	_, err := f.submit(t, in)
	var fe *FieldError
	require.True(t, errors.As(err, &fe), "varian wajib dipilih")
	require.Equal(t, "varian_id", fe.Field)

	in.VarianID = v2
	res, err := f.submit(t, in)
	require.NoError(t, err)
	require.Equal(t, "bayar", res.Next)
	require.NotEmpty(t, res.BayarRef)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.Equal(t, LeadStageInfoBayarDikirim, l.Stage)
	require.EqualValues(t, 350000, l.PaymentAmount)
	require.Equal(t, "2x", l.PaketKode)
}

func TestSubmitLostReopens(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, status, lost_reason, created_at, updated_at)
		VALUES (?, '6281234567890', 'lost', 'harga', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testChatID)
	require.NoError(t, err)
	_, err = f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.Equal(t, LeadStatusOpen, l.Status)
	require.Equal(t, LeadStageDataTerisi, l.Stage)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM lead_events WHERE lead_id = ? AND type = 'lead.reopened'`, l.ID))
}

func TestSubmitWonFlagsReview(t *testing.T) {
	for _, status := range []string{"won", "do_not_contact"} {
		t.Run(status, func(t *testing.T) {
			f, _ := setupSubmitFixture(t, "bayar")
			_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, status, stage, created_at, updated_at)
				VALUES (?, '6281234567890', ?, 'rekomendasi', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testChatID, status)
			require.NoError(t, err)
			res, err := f.submit(t, validSubmit("081234567890"))
			require.NoError(t, err)
			require.Equal(t, "wa", res.Next, "tidak diarahkan ke halaman bayar")
			l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
			require.NoError(t, err)
			require.Equal(t, status, l.Status)
			require.Equal(t, "rekomendasi", l.Stage)
			require.EqualValues(t, 1, l.NeedsReview)
			require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM lead_registrations WHERE lead_id = ?`, l.ID))
		})
	}
}

func TestSubmitFullScheduleRejected(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	_, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, jenis_kelamin, jadwal_kode, verified_at, created_at, updated_at)
		VALUES ('6289999999999@s.whatsapp.net', '6289999999999', 'P', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testJadwal)
	require.NoError(t, err)
	_, err = f.submit(t, validSubmit("081234567890"))
	var fe *FieldError
	require.True(t, errors.As(err, &fe))
	require.Equal(t, "jadwal_kode", fe.Field)
	require.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM leads`), "tidak ada lead baru")
	require.Zero(t, f.count(t, `SELECT COUNT(*) FROM lead_registrations`))

	in := validSubmit("081234567890")
	in.JenisKelamin = "L"
	_, err = f.submit(t, in)
	require.NoError(t, err, "kuota ikhwan masih ada")
}

func TestSubmitProgramPendekNoJadwal(t *testing.T) {
	f := setupRegistrationFixture(t)
	res, err := f.db.Exec(`INSERT INTO lead_programs (nama, jenis, produk_id) VALUES ('Sepekan Al-Fatihah', 'program', ?)`, f.produkID)
	require.NoError(t, err)
	progID, _ := res.LastInsertId()
	req := f.linkReq("tahsin-reguler")
	req.ProgramID = progID
	_, err = f.reg.CreateLink(t.Context(), req, f.adminID)
	require.NoError(t, err)
	in := validSubmit("081234567890")
	in.JadwalKode = ""
	_, err = f.submit(t, in)
	require.NoError(t, err)
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.Empty(t, l.JadwalKode)
	require.EqualValues(t, progID, l.ProgramID.Int64)
}

func TestSubmitHoneypotWritesNothing(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	in := validSubmit("081234567890")
	in.Honeypot = "http://spam"
	_, err := f.submit(t, in)
	require.NoError(t, err)
	require.Zero(t, f.count(t, `SELECT COUNT(*) FROM leads`))
	require.Zero(t, f.count(t, `SELECT COUNT(*) FROM lead_registrations`))
}

func TestSubmitValidation(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	cases := map[string]func(*models.RegistrationSubmit){
		"nama":          func(in *models.RegistrationSubmit) { in.Nama = "A" },
		"no_wa":         func(in *models.RegistrationSubmit) { in.NoWA = "123" },
		"jenis_kelamin": func(in *models.RegistrationSubmit) { in.JenisKelamin = "X" },
		"usia":          func(in *models.RegistrationSubmit) { in.Usia = 200 },
		"jadwal_kode":   func(in *models.RegistrationSubmit) { in.JadwalKode = "" },
	}
	for field, mutate := range cases {
		in := validSubmit("081234567890")
		mutate(&in)
		_, err := f.submit(t, in)
		var fe *FieldError
		require.True(t, errors.As(err, &fe), field)
		require.Equal(t, field, fe.Field)
	}
	require.Zero(t, f.count(t, `SELECT COUNT(*) FROM leads`))
}

func TestSubmitWaURL(t *testing.T) {
	f, _ := setupSubmitFixture(t, "wa")
	res, err := f.submit(t, validSubmit("081234567890"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(res.WaURL, "https://wa.me/6281100000000?text="))
	u, err := url.Parse(res.WaURL)
	require.NoError(t, err)
	text := u.Query().Get("text")
	require.Contains(t, text, "Fulanah")
	require.Contains(t, text, res.LeadRef)

	req := f.linkReq("tahsin-reguler")
	req.WaTemplate = "Daftar {program} jadwal {jadwal}"
	links, err := f.reg.ListLinks(t.Context())
	require.NoError(t, err)
	require.NoError(t, f.reg.UpdateLink(t.Context(), links[0].ID, req))
	in := validSubmit("081299999999")
	res, err = f.submit(t, in)
	require.NoError(t, err)
	u, err = url.Parse(res.WaURL)
	require.NoError(t, err)
	require.Equal(t, "Daftar Tahsin Reguler jadwal "+JadwalLabel("1,4", "19:30", "20:30"), u.Query().Get("text"))
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
