package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

// sbafFixture: a short program "Sepekan Bersama Al-Fatihah" sold as Produk
// CRM product, mapped to its LP source, and a lead that came from that LP and
// is already verified.
type sbafFixture struct {
	*leadFixture
	programs  *LeadProgramService
	programID int64
	produkID  int64
	leadID    int64
}

func setupSBAF(t *testing.T) *sbafFixture {
	t.Helper()
	f := setupLeadFixture(t)
	ctx := t.Context()
	srcID, err := f.service.CreateSource(ctx, models.LeadSourceRequest{Nama: "SBAF LPWA", URL: "https://post.ruangsanad.com/sepekan-bersama-al-fatihah", IsAktif: true})
	require.NoError(t, err)
	require.NoError(t, f.service.AddSourcePattern(ctx, srcID, "saya ingin daftar Sepekan Bersama Al-Fatihah"))

	res, err := f.db.Exec(`INSERT INTO produk (nama, kategori, track_stok, is_aktif, minimum_stock, harga) VALUES ('Sepekan Bersama Al-Fatihah', 'program', 0, 1, 0, 50000)`)
	require.NoError(t, err)
	produkID, _ := res.LastInsertId()

	programs := NewLeadProgramService(f.querier)
	_, err = programs.Create(models.LeadProgramRequest{Nama: "Sepekan Bersama Al-Fatihah", ProdukID: 999})
	require.ErrorContains(t, err, "tidak ditemukan")
	pid, err := programs.Create(models.LeadProgramRequest{Nama: "Sepekan Bersama Al-Fatihah", ProdukID: produkID, IsAktif: true, Keterangan: "22, 25 & 28 Nov"})
	require.NoError(t, err)
	require.NoError(t, programs.SetSources(pid, []int64{srcID}))

	f.ingest(t, leadEvent("evt_s1", "message.in", "2026-11-18T03:00:00Z", map[string]string{"wa_message_id": "S1", "body": "Assalamualaikum, saya ingin daftar Sepekan Bersama Al-Fatihah"}))
	l := f.lead(t)
	_, err = f.db.Exec(`UPDATE leads SET nama = 'Ahmad Budi', stage = 'terverifikasi', payment_amount = 50000 WHERE id = ?`, l.ID)
	require.NoError(t, err)
	return &sbafFixture{leadFixture: f, programs: programs, programID: pid, produkID: produkID, leadID: l.ID}
}

func TestProgramPendekClosingJadiPesertaBukanSantri(t *testing.T) {
	f := setupSBAF(t)
	ctx := t.Context()

	// The lead follows its source into the short program.
	info, err := f.service.ProgramInfo(ctx, f.leadID)
	require.NoError(t, err)
	require.Equal(t, f.programID, info.Program.ID)
	require.False(t, info.Manual)

	// The program filter on the lead list.
	list, err := f.service.List(ctx, models.LeadListParams{Status: "all", ProgramID: f.programID})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	list, err = f.service.List(ctx, models.LeadListParams{Status: "all", ProgramID: queries.LeadProgramRegulerID})
	require.NoError(t, err)
	require.Len(t, list.Items, 0)

	// "Jadikan santri" is refused for a short-program lead.
	_, err = f.service.Convert(ctx, f.leadID, f.csID, models.LeadConvertRequest{Nama: "Ahmad Budi", JenisKelamin: "L", Angkatan: "2026", TanggalDaftar: "2026-11-18", NoWA: "081234567890"})
	require.ErrorIs(t, err, ErrLeadProgramPendek)

	orderID, err := f.service.DaftarkanPeserta(ctx, f.leadID, f.csID, models.LeadPesertaRequest{JenisKelamin: "L"})
	require.NoError(t, err)
	l := f.lead(t)
	require.Equal(t, LeadStatusWon, l.Status)
	require.False(t, l.SantriID.Valid, "no santri for a short program")
	var santri int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM santri`).Scan(&santri))
	require.Zero(t, santri)

	var nama, noWA string
	var leadID, harga int64
	require.NoError(t, f.db.QueryRow(`SELECT o.nama_pembeli, o.no_wa, o.lead_id, i.harga_satuan FROM product_orders o JOIN product_order_items i ON i.order_id = o.id WHERE o.id = ?`, orderID).Scan(&nama, &noWA, &leadID, &harga))
	require.Equal(t, "Ahmad Budi", nama)
	require.Equal(t, "6281234567890", noWA)
	require.Equal(t, f.leadID, leadID)
	require.EqualValues(t, 50000, harga)

	_, err = f.service.DaftarkanPeserta(ctx, f.leadID, f.csID, models.LeadPesertaRequest{JenisKelamin: "L"})
	require.Error(t, err, "closing twice is refused")

	d, err := f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Len(t, d.Peserta, 1)
	list2, err := f.programs.List()
	require.NoError(t, err)
	for _, p := range list2 {
		if p.ID == f.programID {
			require.EqualValues(t, 1, p.LeadWon)
			require.EqualValues(t, 1, p.JumlahPeserta)
		}
	}

	// Offer the regular class: the lead reopens in Tahsin Reguler; the
	// participant order stays.
	require.NoError(t, f.service.TawarkanReguler(ctx, f.leadID, f.csID))
	l = f.lead(t)
	require.Equal(t, LeadStatusOpen, l.Status)
	require.Equal(t, "rekomendasi", l.Stage)
	info, err = f.service.ProgramInfo(ctx, f.leadID)
	require.NoError(t, err)
	require.Equal(t, queries.LeadProgramRegulerID, info.Program.ID)
	require.NotNil(t, info.Order)
	require.Error(t, f.service.TawarkanReguler(ctx, f.leadID, f.csID))
	d, err = f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Len(t, d.Peserta, 1)
}

func TestProgramAbsensiLinkPublik(t *testing.T) {
	f := setupSBAF(t)
	ctx := t.Context()
	_, err := f.service.DaftarkanPeserta(ctx, f.leadID, f.csID, models.LeadPesertaRequest{NoWA: "0812-3456-7890", JenisKelamin: "L"})
	require.NoError(t, err)

	_, err = f.programs.CreateSesi(queries.LeadProgramRegulerID, models.ProgramSesiRequest{Ke: 1, Tanggal: "2026-11-22"})
	require.Error(t, err, "Tahsin Reguler has no sessions")
	sid, err := f.programs.CreateSesi(f.programID, models.ProgramSesiRequest{Ke: 1, Tanggal: "2026-11-22", JamMulai: "19:30", Judul: "Makharijul huruf"})
	require.NoError(t, err)
	_, err = f.programs.CreateSesi(f.programID, models.ProgramSesiRequest{Ke: 1, Tanggal: "2026-11-25"})
	require.ErrorContains(t, err, "sudah ada")
	sesi, err := f.querier.GetProgramSesi(ctx, sid)
	require.NoError(t, err)
	require.Len(t, sesi.Token, 12)

	at := func(s string) {
		tm, err := time.ParseInLocation("2006-01-02 15:04", s, ZonaWaktuRiayah)
		require.NoError(t, err)
		f.programs.now = func() time.Time { return tm }
	}
	at("2026-11-22 18:00")
	info, _, err := f.programs.HadirInfo(sesi.Token)
	require.NoError(t, err)
	require.Equal(t, "belum", info.Status)
	_, err = f.programs.Hadir(sesi.Token, "081234567890")
	require.ErrorContains(t, err, "19:00")

	at("2026-11-22 19:40")
	_, err = f.programs.Hadir(sesi.Token, "081299999999")
	require.ErrorContains(t, err, "belum terdaftar")
	_, err = f.programs.Hadir("salah", "081234567890")
	require.ErrorIs(t, err, ErrProgramSesiTidakAda)
	hasil, err := f.programs.Hadir(sesi.Token, "+62 812-3456-7890")
	require.NoError(t, err)
	require.Equal(t, "A***d B**i", hasil.Nama)
	require.False(t, hasil.SudahTercatat)
	hasil, err = f.programs.Hadir(sesi.Token, "081234567890")
	require.NoError(t, err)
	require.True(t, hasil.SudahTercatat)

	at("2026-11-22 22:31")
	_, err = f.programs.Hadir(sesi.Token, "081234567890")
	require.ErrorContains(t, err, "ditutup")

	d, err := f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Equal(t, []int64{sid}, d.Peserta[0].Hadir)
	_, err = f.programs.DeleteSesi(sid)
	require.ErrorContains(t, err, "kehadiran", "a session with attendance cannot be deleted")

	// Admin correction.
	_, err = f.programs.SetHadir(sid, d.Peserta[0].OrderID, false)
	require.NoError(t, err)
	d, err = f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Empty(t, d.Peserta[0].Hadir)
}

func TestProgramManualDanValidasi(t *testing.T) {
	f := setupSBAF(t)
	ctx := t.Context()
	// Moving the lead back to Tahsin Reguler by hand allows "Jadikan santri".
	require.NoError(t, f.service.SetProgram(ctx, f.leadID, queries.LeadProgramRegulerID, f.csID))
	info, err := f.service.ProgramInfo(ctx, f.leadID)
	require.NoError(t, err)
	require.True(t, info.Manual)
	require.Equal(t, queries.LeadProgramRegulerID, info.Program.ID)
	_, err = f.service.DaftarkanPeserta(ctx, f.leadID, f.csID, models.LeadPesertaRequest{JenisKelamin: "L"})
	require.ErrorIs(t, err, ErrLeadBukanProgramPendek)
	// 0 = follow the source again.
	require.NoError(t, f.service.SetProgram(ctx, f.leadID, 0, f.csID))
	info, err = f.service.ProgramInfo(ctx, f.leadID)
	require.NoError(t, err)
	require.Equal(t, f.programID, info.Program.ID)

	// The built-in program cannot become a short program; a book is not a program product.
	require.NoError(t, f.programs.Update(queries.LeadProgramRegulerID, models.LeadProgramRequest{Nama: "Tahsin Reguler", Jenis: "program", ProdukID: f.produkID}))
	p, err := f.querier.GetLeadProgram(ctx, queries.LeadProgramRegulerID)
	require.NoError(t, err)
	require.Equal(t, LeadProgramJenisReguler, p.Jenis)
	res, err := f.db.Exec(`INSERT INTO produk (nama, kategori, track_stok, is_aktif, minimum_stock, harga) VALUES ('Buku Tahsin', 'buku', 0, 1, 0, 50000)`)
	require.NoError(t, err)
	bukuID, _ := res.LastInsertId()
	_, err = f.programs.Create(models.LeadProgramRequest{Nama: "Daurah", ProdukID: bukuID})
	require.ErrorContains(t, err, "kategori program")
	_, err = f.programs.Create(models.LeadProgramRequest{Nama: "sepekan bersama  al-fatihah", ProdukID: f.produkID})
	require.Error(t, err)
}

func TestProgramRegulerBolehPunyaProdukHarga(t *testing.T) {
	f := setupLeadFixture(t)
	programs := NewLeadProgramService(f.querier)
	res, err := f.db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Tahsin Reguler', 'program', 150000)`)
	require.NoError(t, err)
	produkID, _ := res.LastInsertId()
	res, err = f.db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Buku Tajwid', 'buku', 50000)`)
	require.NoError(t, err)
	bukuID, _ := res.LastInsertId()

	require.Error(t, programs.Update(1, models.LeadProgramRequest{Nama: "Tahsin Reguler", ProdukID: bukuID}), "produk buku ditolak")
	require.NoError(t, programs.Update(1, models.LeadProgramRequest{Nama: "Tahsin Reguler", Jenis: "program", ProdukID: produkID, ProdukBatchID: 99, IsAktif: false}))
	p, err := f.querier.GetLeadProgram(t.Context(), 1)
	require.NoError(t, err)
	require.Equal(t, LeadProgramJenisReguler, p.Jenis, "jenis tetap reguler")
	require.True(t, p.IsAktif, "tetap aktif")
	require.Equal(t, produkID, p.ProdukID)
	require.Zero(t, p.ProdukBatchID, "reguler tanpa batch")

	require.NoError(t, programs.Update(1, models.LeadProgramRequest{Nama: "Tahsin Reguler"}))
	p, err = f.querier.GetLeadProgram(t.Context(), 1)
	require.NoError(t, err)
	require.Zero(t, p.ProdukID, "produk boleh dilepas")
}
