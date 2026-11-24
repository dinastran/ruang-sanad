package services

import (
	"fmt"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// addOrder records a participant order for the SBAF product; jk "" = an order
// recorded by hand without a lead (gender unknown).
func (f *sbafFixture) addOrder(t *testing.T, n int, jk string) int64 {
	t.Helper()
	var leadID any
	if jk != "" {
		res, err := f.db.Exec(`INSERT INTO leads (wa_chat_id, phone, jenis_kelamin, status, created_at, updated_at)
			VALUES (?, ?, ?, 'won', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, fmt.Sprintf("62899%04d@s.whatsapp.net", n), fmt.Sprintf("62899%04d", n), jk)
		require.NoError(t, err)
		leadID, _ = res.LastInsertId()
	}
	res, err := f.db.Exec(`INSERT INTO product_orders (order_no, tanggal, nama_pembeli, no_wa, sumber, lead_id) VALUES (?, '2026-11-20', ?, ?, 'whatsapp', ?)`,
		fmt.Sprintf("T-%04d", n), fmt.Sprintf("Peserta %d", n), fmt.Sprintf("0899%04d", n), leadID)
	require.NoError(t, err)
	oid, _ := res.LastInsertId()
	_, err = f.db.Exec(`INSERT INTO product_order_items (order_id, produk_id, qty) VALUES (?, ?, 1)`, oid, f.produkID)
	require.NoError(t, err)
	return oid
}

func (f *sbafFixture) setKapasitas(t *testing.T, n int64) {
	t.Helper()
	p, err := f.programs.querier.GetLeadProgram(t.Context(), f.programID)
	require.NoError(t, err)
	require.NoError(t, f.programs.Update(f.programID, models.LeadProgramRequest{
		Nama: p.Nama, Jenis: p.Jenis, ProdukID: p.ProdukID, Keterangan: p.Keterangan, IsAktif: p.IsAktif, KapasitasKelas: n,
	}))
}

func kelasCounts(d *ProgramDetail) map[string]int64 {
	out := map[string]int64{}
	for _, k := range d.Kelas {
		out[k.Nama] = k.Jumlah
	}
	return out
}

func TestProgramKelasAutoSplitByGenderAndCapacity(t *testing.T) {
	f := setupSBAF(t)
	p, err := f.programs.querier.GetLeadProgram(t.Context(), f.programID)
	require.NoError(t, err)
	require.EqualValues(t, 30, p.KapasitasKelas, "default capacity")
	f.setKapasitas(t, 2)

	var order []int64
	for i, jk := range []string{"P", "L", "P", "P", "L", "P", "P", ""} {
		order = append(order, f.addOrder(t, i, jk))
	}
	d, err := f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"Ikhwan 1": 2, "Akhwat 1": 2, "Akhwat 2": 2, "Akhwat 3": 1}, kelasCounts(d))

	byOrder := map[int64]models.ProgramPeserta{}
	for _, x := range d.Peserta {
		byOrder[x.OrderID] = x
	}
	nama := map[int64]string{}
	for _, k := range d.Kelas {
		nama[k.ID] = k.Nama
	}
	// Filled in order: the oldest akhwat orders take Akhwat 1.
	require.Equal(t, "Akhwat 1", nama[byOrder[order[0]].KelasID])
	require.Equal(t, "Akhwat 1", nama[byOrder[order[2]].KelasID])
	require.Equal(t, "Akhwat 2", nama[byOrder[order[3]].KelasID])
	require.Equal(t, "Akhwat 3", nama[byOrder[order[6]].KelasID])
	unknown := byOrder[order[7]]
	require.Zero(t, unknown.KelasID, "no gender = no class yet")
	require.Empty(t, unknown.JenisKelamin)

	// Viewing again changes nothing (placed participants are never moved).
	d2, err := f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Equal(t, kelasCounts(d), kelasCounts(d2))

	// Admin sets the unknown participant to ikhwan → Ikhwan 1 is full → Ikhwan 2.
	require.NoError(t, f.programs.SetPesertaKelas(f.programID, order[7], models.ProgramPesertaKelasRequest{JenisKelamin: "L"}))
	d, err = f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.EqualValues(t, 1, kelasCounts(d)["Ikhwan 2"])

	// A cancelled order frees its seat; the next akhwat fills Akhwat 1 again.
	_, err = f.db.Exec(`UPDATE product_orders SET status = 'dibatalkan' WHERE id = ?`, order[0])
	require.NoError(t, err)
	late := f.addOrder(t, 20, "P")
	d, err = f.programs.Detail(f.programID)
	require.NoError(t, err)
	for _, x := range d.Peserta {
		if x.OrderID == late {
			require.Equal(t, "Akhwat 1", nama[x.KelasID])
		}
	}
}

func TestProgramKelasManualMoveRespectsGenderAndCapacity(t *testing.T) {
	f := setupSBAF(t)
	f.setKapasitas(t, 2)
	a := f.addOrder(t, 1, "P")
	f.addOrder(t, 2, "P")
	c := f.addOrder(t, 3, "P")
	ikh := f.addOrder(t, 4, "L")
	d, err := f.programs.Detail(f.programID)
	require.NoError(t, err)
	ids := map[string]int64{}
	for _, k := range d.Kelas {
		ids[k.Nama] = k.ID
	}

	err = f.programs.SetPesertaKelas(f.programID, c, models.ProgramPesertaKelasRequest{KelasID: ids["Akhwat 1"]})
	require.ErrorContains(t, err, "penuh")
	require.NoError(t, f.programs.SetPesertaKelas(f.programID, a, models.ProgramPesertaKelasRequest{KelasID: ids["Akhwat 2"]}))
	// Moving an ikhwan into an akhwat class changes the stored gender with it.
	require.NoError(t, f.programs.SetPesertaKelas(f.programID, ikh, models.ProgramPesertaKelasRequest{KelasID: ids["Akhwat 1"]}))

	d, err = f.programs.Detail(f.programID)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"Ikhwan 1": 0, "Akhwat 1": 2, "Akhwat 2": 2}, kelasCounts(d))
	for _, x := range d.Peserta {
		if x.OrderID == a {
			require.True(t, x.KelasManual)
		}
	}

	// Only an empty class can be deleted.
	_, err = f.programs.DeleteKelas(ids["Akhwat 2"])
	require.ErrorContains(t, err, "masih berisi")
	_, err = f.programs.DeleteKelas(ids["Ikhwan 1"])
	require.NoError(t, err)

	// Teacher + Zoom link; the participant sees class and link after check-in.
	_, err = f.programs.UpdateKelas(ids["Akhwat 2"], models.ProgramKelasRequest{ZoomURL: "bukan url"})
	require.Error(t, err)
	_, err = f.programs.UpdateKelas(ids["Akhwat 2"], models.ProgramKelasRequest{Pengajar: " Ustadzah  Fulanah ", ZoomURL: "https://zoom.us/j/123"})
	require.NoError(t, err)
	k, err := f.programs.querier.GetProgramKelas(t.Context(), ids["Akhwat 2"])
	require.NoError(t, err)
	require.Equal(t, "Ustadzah Fulanah", k.Pengajar)
	nama, zoom := pesertaKelasInfo(t.Context(), f.programs.querier, f.programID, a)
	require.Equal(t, "Akhwat 2", nama)
	require.Equal(t, "https://zoom.us/j/123", zoom)
}

func TestDaftarkanPesertaMasukKelas(t *testing.T) {
	f := setupSBAF(t)
	ctx := t.Context()
	_, err := f.service.DaftarkanPeserta(ctx, f.leadID, f.csID, models.LeadPesertaRequest{})
	require.ErrorContains(t, err, "ikhwan atau akhwat", "gender required when the lead has none")
	orderID, err := f.service.DaftarkanPeserta(ctx, f.leadID, f.csID, models.LeadPesertaRequest{JenisKelamin: "P"})
	require.NoError(t, err)
	rows, err := f.programs.querier.ListProgramPesertaKelas(ctx, f.programID)
	require.NoError(t, err)
	require.Equal(t, "P", rows[orderID].JenisKelamin)
	require.NotZero(t, rows[orderID].KelasID, "placed in a class right away")
}
