package services

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const keuanganTrenBulan = 6

var namaBulanID = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func labelBulanID(t time.Time) string {
	return fmt.Sprintf("%s %d", namaBulanID[t.Month()-1], t.Year())
}

func labelBulanPendekID(t time.Time) string {
	return fmt.Sprintf("%s %02d", namaBulanID[t.Month()-1][:3], t.Year()%100)
}

// KeuanganDashboardService aggregates SPP, ujroh and product data for the
// finance dashboard. Cancelled tagihan are excluded everywhere.
type KeuanganDashboardService struct {
	querier *queries.Querier
	ujroh   *UjrohService
	now     func() time.Time
}

func NewKeuanganDashboardService(querier *queries.Querier, ujroh *UjrohService) *KeuanganDashboardService {
	return &KeuanganDashboardService{querier: querier, ujroh: ujroh, now: HariIniRiayah}
}

// Get builds the dashboard for bulan (YYYY-MM, default the current WIB month).
// Piutang, aging and follow-up priorities are always as of today.
func (s *KeuanganDashboardService) Get(bulan string) (*models.KeuanganDashboard, error) {
	bulan, err := s.ujroh.NormalizeBulan(bulan)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	now := s.now()
	hariIni := now.Format("2006-01-02")
	bulanIni := now.Format("2006-01")
	awal, _ := time.Parse("2006-01", bulan)

	out := &models.KeuanganDashboard{
		Bulan: bulan, BulanLabel: labelBulanID(awal), HariIni: hariIni,
		BulanSebelum: awal.AddDate(0, -1, 0).Format("2006-01"),
	}
	if bulan < bulanIni {
		out.BulanBerikut = awal.AddDate(0, 1, 0).Format("2006-01")
	}

	tren, err := s.tren(ctx, awal)
	if err != nil {
		return nil, err
	}
	out.Tren = tren
	out.Sekarang = tren[len(tren)-1]
	out.Sebelumnya = tren[len(tren)-2]

	aging, err := s.querier.KeuanganAgingPiutang(ctx, hariIni)
	if err != nil {
		return nil, err
	}
	out.Aging = []models.KeuanganAgingBucket{
		{Key: "belum_tempo", Label: "Belum jatuh tempo", Jumlah: aging.BelumTempoJumlah, Nominal: aging.BelumTempoNominal},
		{Key: "telat_30", Label: "Telat 1–30 hari", Jumlah: aging.Telat30Jumlah, Nominal: aging.Telat30Nominal},
		{Key: "telat_60", Label: "Telat 31–60 hari", Jumlah: aging.Telat60Jumlah, Nominal: aging.Telat60Nominal},
		{Key: "telat_lebih", Label: "Telat > 60 hari", Jumlah: aging.TelatLebihJumlah, Nominal: aging.TelatLebihNominal},
	}
	for i, b := range out.Aging {
		out.TotalPiutang += b.Nominal
		out.JumlahPiutang += b.Jumlah
		if i > 0 {
			out.JumlahTerlambat += b.Jumlah
		}
	}

	if out.Prioritas, err = s.prioritas(ctx, hariIni); err != nil {
		return nil, err
	}
	if out.PiutangKelas, out.PiutangAngkatan, err = s.piutangGroups(ctx); err != nil {
		return nil, err
	}
	if out.Ujroh, err = s.ujrohPanel(ctx, bulan, bulanIni); err != nil {
		return nil, err
	}
	produk, err := s.querier.KeuanganPenjualanProdukBulan(ctx, bulan)
	if err != nil {
		return nil, err
	}
	out.Produk = models.KeuanganProdukPanel{JumlahOrder: produk.JumlahOrder, JumlahQty: produk.JumlahQty, Omzet: produk.Omzet}
	return out, nil
}

// tren returns keuanganTrenBulan months ending at akhir, oldest first.
func (s *KeuanganDashboardService) tren(ctx context.Context, akhir time.Time) ([]models.KeuanganBulan, error) {
	mulai := akhir.AddDate(0, -(keuanganTrenBulan - 1), 0)
	dari, sampai := mulai.Format("2006-01"), akhir.Format("2006-01")

	masuk, err := s.querier.KeuanganKasMasukSppPerBulan(ctx, queries.KeuanganKasMasukSppPerBulanParams{DariBulan: dari, SampaiBulan: sampai})
	if err != nil {
		return nil, err
	}
	keluar, err := s.querier.KeuanganUjrohKasKeluarPerBulan(ctx, dari, sampai)
	if err != nil {
		return nil, err
	}
	produk, err := s.querier.KeuanganOmzetProdukPerBulan(ctx, queries.KeuanganOmzetProdukPerBulanParams{DariBulan: dari, SampaiBulan: sampai})
	if err != nil {
		return nil, err
	}
	terbit, err := s.querier.KeuanganKolektibilitasPerBulan(ctx, queries.KeuanganKolektibilitasPerBulanParams{DariBulan: dari, SampaiBulan: sampai})
	if err != nil {
		return nil, err
	}

	out := make([]models.KeuanganBulan, keuanganTrenBulan)
	index := make(map[string]int, keuanganTrenBulan)
	for i := range out {
		t := mulai.AddDate(0, i, 0)
		out[i] = models.KeuanganBulan{Bulan: t.Format("2006-01"), Label: labelBulanPendekID(t)}
		index[out[i].Bulan] = i
	}
	for _, r := range masuk {
		if i, ok := index[r.Bulan]; ok {
			out[i].KasMasukSpp, out[i].JumlahBayarSpp = r.Nominal, r.Jumlah
		}
	}
	for _, r := range produk {
		if i, ok := index[r.Bulan]; ok {
			out[i].KasMasukProduk = r.Nominal
		}
	}
	for _, r := range keluar {
		if i, ok := index[r.Bulan]; ok {
			out[i].KasKeluar, out[i].JumlahUjroh = r.Nominal, r.Jumlah
		}
	}
	for _, r := range terbit {
		if i, ok := index[r.Bulan]; ok {
			out[i].TerbitJumlah, out[i].TerbitNominal, out[i].TerbitLunas = r.Jumlah, r.Nominal, r.NominalLunas
		}
	}
	for i := range out {
		b := &out[i]
		b.KasMasuk = b.KasMasukSpp + b.KasMasukProduk
		b.KasBersih = b.KasMasuk - b.KasKeluar
		if b.TerbitNominal > 0 {
			b.Kolektibilitas = float64(b.TerbitLunas) / float64(b.TerbitNominal) * 100
		}
	}
	return out, nil
}

func (s *KeuanganDashboardService) prioritas(ctx context.Context, hariIni string) ([]models.KeuanganPrioritasFU, error) {
	rows, err := s.querier.KeuanganPrioritasFollowUp(ctx, queries.KeuanganPrioritasFollowUpParams{HariIni: hariIni, Batas: 10})
	if err != nil {
		return nil, err
	}
	out := make([]models.KeuanganPrioritasFU, 0, len(rows))
	for _, r := range rows {
		item := tagihanResponse(r.ID, r.SantriID, r.KelasID, r.BulanKe, r.PertemuanKe, r.Nominal, r.TanggalTagih, r.JatuhTempo, r.Status, r.TanggalBayar, r.Metode, r.Catatan, r.FuTerakhir, r.FuCount, r.SantriNama, r.IDMahasantri, r.NoWa, r.Angkatan, r.AngkatanKelas, r.Frekuensi, r.NamaKelas, r.GuruNama)
		item.Status = "terlambat"
		out = append(out, models.KeuanganPrioritasFU{TagihanResponse: item, HariTelat: r.HariTelat})
	}
	return out, nil
}

func (s *KeuanganDashboardService) piutangGroups(ctx context.Context) ([]models.KeuanganPiutangGroup, []models.KeuanganPiutangGroup, error) {
	kelasRows, err := s.querier.KeuanganPiutangPerKelas(ctx, 10)
	if err != nil {
		return nil, nil, err
	}
	kelas := make([]models.KeuanganPiutangGroup, 0, len(kelasRows))
	for _, r := range kelasRows {
		g := models.KeuanganPiutangGroup{Label: r.Label, JumlahSantri: r.JumlahSantri, JumlahTagihan: r.JumlahTagihan, Nominal: r.Nominal}
		if r.KelasID.Valid {
			g.Href = fmt.Sprintf("/app/keuangan/tagihan?status=belum_bayar&kelas_id=%d", r.KelasID.Int64)
		}
		kelas = append(kelas, g)
	}
	angkatanRows, err := s.querier.KeuanganPiutangPerAngkatan(ctx, 10)
	if err != nil {
		return nil, nil, err
	}
	angkatan := make([]models.KeuanganPiutangGroup, 0, len(angkatanRows))
	for _, r := range angkatanRows {
		g := models.KeuanganPiutangGroup{Label: r.Label, JumlahSantri: r.JumlahSantri, JumlahTagihan: r.JumlahTagihan, Nominal: r.Nominal}
		if r.Label == "" {
			g.Label = "Tanpa angkatan"
		} else {
			g.Href = "/app/keuangan/tagihan?status=belum_bayar&angkatan_kelas=" + url.QueryEscape(r.Label)
		}
		angkatan = append(angkatan, g)
	}
	return kelas, angkatan, nil
}

func (s *KeuanganDashboardService) ujrohPanel(ctx context.Context, bulan, bulanIni string) (models.KeuanganUjrohPanel, error) {
	rekap, err := s.ujroh.GetRekap(bulan)
	if err != nil {
		return models.KeuanganUjrohPanel{}, err
	}
	panel := models.KeuanganUjrohPanel{
		Bulan: bulan, Terkunci: rekap.Terkunci, TotalUjroh: rekap.Ringkasan.TotalUjroh,
		TotalDibayar: rekap.Ringkasan.TotalDibayar, TotalBelum: rekap.Ringkasan.TotalBelum, JumlahGuru: rekap.Ringkasan.JumlahGuru,
	}
	for _, g := range rekap.Guru {
		if g.Dibayar {
			panel.GuruDibayar++
		}
	}
	jumlahGuru, nominal, err := s.querier.KeuanganUjrohTerkunciBelumDibayarBersih(ctx)
	if err != nil {
		return panel, err
	}
	panel.KewajibanTerkunci, panel.KewajibanTerkunciGuru = nominal, jumlahGuru

	berjalan := rekap
	if bulan != bulanIni {
		if berjalan, err = s.ujroh.GetRekap(bulanIni); err != nil {
			return panel, err
		}
	}
	if !berjalan.Terkunci {
		panel.EstimasiBerjalan = berjalan.Ringkasan.TotalUjroh
	}
	return panel, nil
}
