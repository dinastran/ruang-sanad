package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Form persis seperti template yang dikirim bridge lewat [[FORM]].
const leadFormTemplateFilled = `Bismillah, berikut data saya kak:
*Form Data Pendaftaran*
1. Nama Lengkap: Fulanah binti Fulan
2. Usia: 27
3. Domisili: Bandung
4. Ikhwan/Akhwat: Akhwat
5. Pilihan Kelas: 1x sepekan
6. Waktu Belajar: malam
7. Apakah sebelumnya pernah belajar tahsin?: Pernah
8. Jika iya/pernah belajar dimana: Masjid dekat rumah`

func TestParseLeadFormTemplate(t *testing.T) {
	p := ParseLeadForm(leadFormTemplateFilled)
	require.True(t, p.Filled())
	require.Equal(t, "Fulanah binti Fulan", p.Nama)
	require.Equal(t, int64(27), p.Usia)
	require.Equal(t, "Bandung", p.Domisili)
	require.Equal(t, "P", p.JenisKelamin)
	require.Equal(t, "1x", p.PaketKode)
	require.Equal(t, "malam", p.WaktuBelajar)
	require.Equal(t, "Pernah", p.RiwayatBelajar)
	require.Equal(t, "Masjid dekat rumah", p.BelajarDimana)
	require.Empty(t, p.Review)
	require.Len(t, p.Recognized, 8)
}

func TestParseLeadFormWithDeletedLines(t *testing.T) {
	p := ParseLeadForm("Nama: Ahmad\nUmur: 30\n- Kota: Depok\n")
	require.True(t, p.Filled())
	require.Equal(t, "Ahmad", p.Nama)
	require.Equal(t, int64(30), p.Usia)
	require.Equal(t, "Depok", p.Domisili)
	require.Empty(t, p.JenisKelamin)
	require.Empty(t, p.Review)
}

func TestParseLeadFormAllInOneLineFails(t *testing.T) {
	p := ParseLeadForm("Nama: Ahmad Usia: 30 Domisili: Depok Ikhwan/Akhwat: Ikhwan")
	require.False(t, p.Filled())
	require.Equal(t, []string{"nama"}, p.Recognized)
}

func TestParseLeadFormAgeWithText(t *testing.T) {
	p := ParseLeadForm("Nama: Budi\nUsia: 25 th\nDomisili: Bogor")
	require.True(t, p.Filled())
	require.Equal(t, int64(25), p.Usia)

	p = ParseLeadForm("Nama: Budi\nUsia: dua puluh\nDomisili: Bogor\nGender: L")
	require.True(t, p.Filled())
	require.Zero(t, p.Usia)
	require.Equal(t, []string{"usia"}, p.Review)
}

func TestParseLeadFormCaseVariations(t *testing.T) {
	p := ParseLeadForm("NAMA LENGKAP: Siti\nusia: 40\nDOMISILI: Jakarta\nikhwan / akhwat: AKHWAT\nPAKET: 2X")
	require.True(t, p.Filled())
	require.Equal(t, "Siti", p.Nama)
	require.Equal(t, "P", p.JenisKelamin)
	require.Equal(t, "2x", p.PaketKode)
}

func TestParseLeadFormNeverGuesses(t *testing.T) {
	p := ParseLeadForm("Nama: Siti\nUsia: 40\nIkhwan/Akhwat: -\nPilihan Kelas: 1x / 2x")
	require.True(t, p.Filled())
	require.Empty(t, p.JenisKelamin)
	require.Empty(t, p.PaketKode)
	require.ElementsMatch(t, []string{"jenis_kelamin", "paket_kode"}, p.Review)
}

func TestIsLeadStopRequest(t *testing.T) {
	for _, msg := range []string{"STOP", "stop.", "Berhenti", "jangan hubungi saya lagi", "Unsubscribe", "stop dulu ya, nanti saya kabari"} {
		require.True(t, IsLeadStopRequest(msg), msg)
	}
	for _, msg := range []string{"belum dulu kak", "nanti saja", "stopkontak rusak", "saya mau daftar, jangan berhenti"} {
		require.False(t, IsLeadStopRequest(msg), msg)
	}
}

func TestLeadPhoneAndRupiah(t *testing.T) {
	require.Equal(t, "6281234567890", NormalizeLeadPhone("0812-3456-7890"))
	require.Equal(t, "6281234567890", NormalizeLeadPhone("+62 812 3456 7890"))
	require.Equal(t, "6281234567890", phoneFromChatID("6281234567890@s.whatsapp.net"))
	require.Equal(t, "", phoneFromChatID("123456789@lid"))
	require.Equal(t, int64(149000), ParseRupiah("Rp149.000"))
	require.Equal(t, int64(298000), ParseRupiah("Rp 298.000,00"))
	require.Equal(t, int64(0), ParseRupiah("gratis"))
}
