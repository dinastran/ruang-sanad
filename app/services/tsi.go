package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type TSIService struct {
	querier *queries.Querier
}

func NewTSIService(querier *queries.Querier) *TSIService {
	return &TSIService{querier: querier}
}

// monthBounds turns "YYYY-MM" into the first and last calendar day.
func monthBounds(bulan string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01", bulan)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("format bulan tidak valid (YYYY-MM)")
	}
	end := start.AddDate(0, 1, -1)
	return start, end, nil
}

type autoResult struct {
	value *float64
	raw   string
}

func pct(part, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	v := float64(part) / float64(total) * 100
	return &v
}

func invPct(bad, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	v := (1 - float64(bad)/float64(total)) * 100
	if v < 0 {
		v = 0
	}
	return &v
}

// computeAuto returns the automatic value + raw explanation per indicator kode.
func (s *TSIService) computeAuto(guruID int64, bulan string) (map[string]autoResult, error) {
	start, end, err := monthBounds(bulan)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	gid := sql.NullInt64{Int64: guruID, Valid: true}
	out := map[string]autoResult{}

	selesai, _ := s.querier.TsiCountPertemuanSelesai(ctx, queries.TsiCountPertemuanSelesaiParams{GuruID: gid, Tanggal: start, Tanggal_2: end})
	withAbsen, _ := s.querier.TsiCountPertemuanWithAbsensi(ctx, queries.TsiCountPertemuanWithAbsensiParams{GuruID: gid, Tanggal: start, Tanggal_2: end})
	totalP, _ := s.querier.TsiCountPertemuanTotal(ctx, queries.TsiCountPertemuanTotalParams{GuruID: gid, Tanggal: start, Tanggal_2: end})
	reschedule, _ := s.querier.TsiCountReschedule(ctx, queries.TsiCountRescheduleParams{GuruID: gid, Tanggal: start, Tanggal_2: end})
	badal, _ := s.querier.TsiCountBadal(ctx, queries.TsiCountBadalParams{GuruID: gid, Tanggal: start, Tanggal_2: end})

	out["absen_dibuat"] = autoResult{pct(withAbsen, selesai), fmt.Sprintf("%d/%d pertemuan ada absensi", withAbsen, selesai)}
	out["absen_riayah"] = autoResult{pct(withAbsen, selesai), fmt.Sprintf("%d/%d pertemuan absensi diisi", withAbsen, selesai)}
	out["no_reschedule"] = autoResult{invPct(reschedule, totalP), fmt.Sprintf("%d reschedule dari %d pertemuan", reschedule, totalP)}
	out["no_badal"] = autoResult{invPct(badal, totalP), fmt.Sprintf("%d badal dari %d pertemuan", badal, totalP)}

	santriTotal, _ := s.querier.TsiCountSantriTotal(ctx, gid)
	tidakLanjut, _ := s.querier.TsiCountSantriTidakLanjut(ctx, gid)
	out["retensi"] = autoResult{invPct(tidakLanjut, santriTotal), fmt.Sprintf("%d mundur dari %d santri", tidakLanjut, santriTotal)}

	pembTerlaksana, _ := s.querier.CountPembinaanTerlaksana(ctx, queries.CountPembinaanTerlaksanaParams{Tanggal: start, Tanggal_2: end})
	pembHadir, _ := s.querier.CountPembinaanHadirByGuru(ctx, queries.CountPembinaanHadirByGuruParams{GuruID: guruID, Tanggal: start, Tanggal_2: end})
	out["pembinaan"] = autoResult{pct(pembHadir, pembTerlaksana), fmt.Sprintf("hadir %d/%d pembinaan", pembHadir, pembTerlaksana)}

	rapatTerlaksana, _ := s.querier.CountRapatTerlaksana(ctx, queries.CountRapatTerlaksanaParams{Tanggal: start, Tanggal_2: end})
	rapatHadir, _ := s.querier.CountRapatHadirByGuru(ctx, queries.CountRapatHadirByGuruParams{GuruID: guruID, Tanggal: start, Tanggal_2: end})
	out["rapat"] = autoResult{pct(rapatHadir, rapatTerlaksana), fmt.Sprintf("hadir %d/%d rapat", rapatHadir, rapatTerlaksana)}

	kalamTotal, _ := s.querier.CountKalamInRange(ctx, queries.CountKalamInRangeParams{Tanggal: start, Tanggal_2: end})
	kalamShare, _ := s.querier.CountKalamShareByGuruRange(ctx, queries.CountKalamShareByGuruRangeParams{GuruID: guruID, Tanggal: start, Tanggal_2: end})
	out["share_kalam"] = autoResult{pct(kalamShare, kalamTotal), fmt.Sprintf("share %d/%d kajian", kalamShare, kalamTotal)}

	return out, nil
}

func (s *TSIService) getOrCreatePeriode(guruID int64, bulan string) (queries.TsiPeriode, error) {
	ctx := context.Background()
	p, err := s.querier.GetTsiPeriode(ctx, queries.GetTsiPeriodeParams{GuruID: guruID, Bulan: bulan})
	if err == nil {
		return p, nil
	}
	id, cerr := s.querier.CreateTsiPeriode(ctx, queries.CreateTsiPeriodeParams{GuruID: guruID, Bulan: bulan})
	if cerr != nil {
		return queries.TsiPeriode{}, cerr
	}
	return s.querier.GetTsiPeriodeByID(ctx, id)
}

// Predikat maps a total score to a qualitative band.
// Default thresholds (owner-overridable): ≥90 Sangat Baik, 80–89 Baik,
// 70–79 Cukup, <70 Perlu Pembinaan.
func Predikat(total float64) string {
	switch {
	case total >= 90:
		return "Sangat Baik"
	case total >= 80:
		return "Baik"
	case total >= 70:
		return "Cukup"
	default:
		return "Perlu Pembinaan"
	}
}

var kategoriOrder = []string{"kompetensi", "kepuasan", "kedisiplinan", "kontribusi"}

// buildPenilaian assembles the full evaluation for a guru/month, computing
// effective values (override > auto > manual) and category/total scores.
func (s *TSIService) buildPenilaian(guruID int64, guruNama, bulan string, periode queries.TsiPeriode) (*models.TsiPenilaianResponse, error) {
	ctx := context.Background()
	kriteria, err := s.querier.ListTsiKriteria(ctx)
	if err != nil {
		return nil, err
	}
	auto, err := s.computeAuto(guruID, bulan)
	if err != nil {
		return nil, err
	}

	nilaiRows, _ := s.querier.ListTsiNilaiByPeriode(ctx, periode.ID)
	stored := map[int64]queries.TsiNilai{}
	for _, n := range nilaiRows {
		stored[n.KriteriaID] = n
	}

	items := make([]models.TsiKriteriaNilai, 0, len(kriteria))
	// bobot per kategori (constant across its indicators)
	bobotKategori := map[string]int64{}
	valuesKategori := map[string][]float64{}
	totalKategori := map[string]int{}

	for _, k := range kriteria {
		bobotKategori[k.Kategori] = k.Bobot
		totalKategori[k.Kategori]++

		var autoVal *float64
		raw := ""
		if k.Kode != "" {
			if ar, ok := auto[k.Kode]; ok {
				autoVal = ar.value
				raw = ar.raw
			}
		}

		st, hasStored := stored[k.ID]
		var effective *float64
		isOverride := false
		alasan := ""
		catatan := ""
		if hasStored {
			isOverride = st.IsOverride == 1
			alasan = st.AlasanOverride
			catatan = st.Catatan
		}

		switch {
		case hasStored && isOverride:
			if st.Nilai.Valid {
				v := st.Nilai.Float64
				effective = &v
			}
		case autoVal != nil:
			effective = autoVal
		case hasStored && st.Nilai.Valid:
			v := st.Nilai.Float64
			effective = &v
		}

		if effective != nil {
			valuesKategori[k.Kategori] = append(valuesKategori[k.Kategori], *effective)
		}

		items = append(items, models.TsiKriteriaNilai{
			KriteriaID:     k.ID,
			Kategori:       k.Kategori,
			Bobot:          k.Bobot,
			Urutan:         k.Urutan,
			Sumber:         k.Sumber,
			Kode:           k.Kode,
			Nama:           k.Nama,
			Nilai:          effective,
			AutoNilai:      autoVal,
			RawDisplay:     raw,
			IsOverride:     isOverride,
			AlasanOverride: alasan,
			Catatan:        catatan,
		})
	}

	kategori := make([]models.TsiKategoriSkor, 0, len(kategoriOrder))
	total := 0.0
	for _, kat := range kategoriOrder {
		bobot := bobotKategori[kat]
		vals := valuesKategori[kat]
		rata, skor := kategoriSkor(bobot, vals)
		total += skor
		kategori = append(kategori, models.TsiKategoriSkor{
			Kategori: kat,
			Bobot:    bobot,
			RataRata: rata,
			Skor:     round2(skor),
			Terisi:   len(vals),
			Total:    totalKategori[kat],
		})
	}

	return &models.TsiPenilaianResponse{
		PeriodeID: periode.ID,
		GuruID:    guruID,
		GuruNama:  guruNama,
		Bulan:     bulan,
		Status:    periode.Status,
		Kriteria:  items,
		Kategori:  kategori,
		Total:     round2(total),
		Predikat:  Predikat(total),
	}, nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// kategoriSkor implements the TSI rule: empty indicators are excluded from the
// average; category score = (bobot/100) × average(filled indicators). When no
// indicator is filled, the score is 0 and the average is nil.
func kategoriSkor(bobot int64, values []float64) (*float64, float64) {
	if len(values) == 0 {
		return nil, 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))
	return &avg, float64(bobot) / 100.0 * avg
}

// GetPenilaian returns the (auto-populated) evaluation for koordinator editing.
func (s *TSIService) GetPenilaian(guruID int64, guruNama, bulan string) (*models.TsiPenilaianResponse, error) {
	periode, err := s.getOrCreatePeriode(guruID, bulan)
	if err != nil {
		return nil, err
	}
	return s.buildPenilaian(guruID, guruNama, bulan, periode)
}

// SaveNilai upserts manual/override values and records an audit trail on change.
func (s *TSIService) SaveNilai(periodeID int64, userID int64, inputs []models.TsiNilaiInput) error {
	ctx := context.Background()
	periode, err := s.querier.GetTsiPeriodeByID(ctx, periodeID)
	if err != nil {
		return fmt.Errorf("periode tidak ditemukan")
	}
	if periode.Status == "final" {
		return fmt.Errorf("periode sudah difinalisasi, buka kembali untuk mengubah")
	}
	for _, in := range inputs {
		old, hasOld := s.querier.GetTsiNilai(ctx, queries.GetTsiNilaiParams{PeriodeID: periodeID, KriteriaID: in.KriteriaID})
		_ = hasOld
		nilai := sql.NullFloat64{}
		if in.Nilai != nil {
			nilai = sql.NullFloat64{Float64: *in.Nilai, Valid: true}
		}
		override := int64(0)
		if in.IsOverride {
			override = 1
		}
		if err := s.querier.UpsertTsiNilai(ctx, queries.UpsertTsiNilaiParams{
			PeriodeID:      periodeID,
			KriteriaID:     in.KriteriaID,
			Nilai:          nilai,
			IsOverride:     override,
			AlasanOverride: in.AlasanOverride,
			Catatan:        in.Catatan,
		}); err != nil {
			return err
		}
		// Audit when an override value changes.
		if in.IsOverride {
			var oldVal sql.NullFloat64
			if hasOld == nil {
				oldVal = old.Nilai
			}
			_ = s.querier.InsertTsiAudit(ctx, queries.InsertTsiAuditParams{
				PeriodeID:  periodeID,
				KriteriaID: in.KriteriaID,
				NilaiLama:  oldVal,
				NilaiBaru:  nilai,
				Alasan:     in.AlasanOverride,
				Oleh:       sql.NullInt64{Int64: userID, Valid: userID > 0},
			})
		}
	}
	return nil
}

func (s *TSIService) SetStatus(periodeID int64, status string) error {
	if status != "draft" && status != "final" {
		return fmt.Errorf("status tidak valid")
	}
	return s.querier.SetTsiPeriodeStatus(context.Background(), queries.SetTsiPeriodeStatusParams{Status: status, ID: periodeID})
}

// GetRekap builds the guru × category matrix for a month (only guru with a periode).
func (s *TSIService) GetRekap(bulan string) ([]models.TsiRekapRow, error) {
	ctx := context.Background()
	periodes, err := s.querier.ListTsiPeriodeByBulan(ctx, bulan)
	if err != nil {
		return nil, err
	}
	out := make([]models.TsiRekapRow, 0, len(periodes))
	for _, p := range periodes {
		pen, err := s.buildPenilaian(p.GuruID, p.GuruNama, bulan, queries.TsiPeriode{ID: p.ID, GuruID: p.GuruID, Bulan: p.Bulan, Status: p.Status})
		if err != nil {
			continue
		}
		row := models.TsiRekapRow{GuruID: p.GuruID, GuruNama: p.GuruNama, GuruStatus: p.GuruStatus, Bulan: bulan, Status: p.Status, Total: pen.Total, Predikat: pen.Predikat}
		for _, kat := range pen.Kategori {
			switch kat.Kategori {
			case "kompetensi":
				row.Kompetensi = kat.Skor
			case "kepuasan":
				row.Kepuasan = kat.Skor
			case "kedisiplinan":
				row.Kedisiplinan = kat.Skor
			case "kontribusi":
				row.Kontribusi = kat.Skor
			}
		}
		out = append(out, row)
	}
	return out, nil
}
