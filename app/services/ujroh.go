package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrUjrohBulanTidakValid = errors.New("format bulan harus YYYY-MM")
	ErrUjrohSudahDikunci    = errors.New("ujroh bulan ini sudah dikunci")
	ErrUjrohBelumDikunci    = errors.New("kunci ujroh bulan ini terlebih dahulu sebelum menandai pembayaran")
	ErrUjrohBulanMendatang  = errors.New("bulan yang belum berjalan tidak dapat dikunci")
	ErrUjrohSudahAdaDibayar = errors.New("tidak dapat membuka kunci: sudah ada guru yang ditandai dibayar")
	ErrUjrohGuruTidakAda    = errors.New("guru tidak ada dalam rekap ujroh bulan ini")
	ErrUjrohNominalTidakSah = errors.New("nominal ujroh tidak boleh negatif")
	ErrUjrohTanggalTidakSah = errors.New("format tanggal bayar harus YYYY-MM-DD")
)

// UjrohService computes monthly teacher pay: completed meetings × the
// teaching guru's rate (per-guru override, else the status default).
type UjrohService struct {
	querier *queries.Querier
	now     func() time.Time
}

func NewUjrohService(querier *queries.Querier) *UjrohService {
	return &UjrohService{querier: querier, now: HariIniRiayah}
}

// NormalizeBulan validates YYYY-MM and defaults to the current WIB month.
func (s *UjrohService) NormalizeBulan(bulan string) (string, error) {
	bulan = strings.TrimSpace(bulan)
	if bulan == "" {
		return s.now().Format("2006-01"), nil
	}
	t, err := time.Parse("2006-01", bulan)
	if err != nil || t.Format("2006-01") != bulan {
		return "", ErrUjrohBulanTidakValid
	}
	return bulan, nil
}

func (s *UjrohService) GetRekap(bulan string) (*models.UjrohRekap, error) {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	rekap := &models.UjrohRekap{Bulan: bulan, Guru: []models.UjrohGuru{}, TanpaGuru: []models.UjrohPertemuan{}, GuruBaruSelisih: []models.UjrohGuru{}}

	live, tanpaGuru, err := s.hitungLive(ctx, s.querier, bulan)
	if err != nil {
		return nil, err
	}
	rekap.TanpaGuru = tanpaGuru
	if rekap.Berlangsung, err = s.querier.CountPertemuanBerlangsungBulan(ctx, bulan); err != nil {
		return nil, err
	}

	status, err := s.querier.GetUjrohBulan(ctx, bulan)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		rekap.DibukaAt = formatNullDateTime(status.DibukaAt)
		rekap.DibukaOleh = status.DibukaOleh
		if status.DikunciAt.Valid {
			rekap.Terkunci = true
			rekap.DikunciAt = formatNullDateTime(status.DikunciAt)
			rekap.DikunciOleh = status.DikunciOleh
		}
	}

	if !rekap.Terkunci {
		rekap.Guru = live
	} else {
		snapshot, err := s.snapshot(ctx, bulan)
		if err != nil {
			return nil, err
		}
		liveByGuru := make(map[int64]models.UjrohGuru, len(live))
		for _, g := range live {
			liveByGuru[g.GuruID] = g
		}
		for i := range snapshot {
			cur := liveByGuru[snapshot[i].GuruID]
			snapshot[i].LiveJumlah = cur.JumlahPertemuan
			snapshot[i].LiveTotal = cur.Total
			snapshot[i].Selisih = cur.JumlahPertemuan != snapshot[i].JumlahPertemuan || cur.Total != snapshot[i].Total
			delete(liveByGuru, snapshot[i].GuruID)
		}
		for _, g := range live {
			if _, baru := liveByGuru[g.GuruID]; baru {
				g.Selisih = true
				g.LiveJumlah = g.JumlahPertemuan
				g.LiveTotal = g.Total
				rekap.GuruBaruSelisih = append(rekap.GuruBaruSelisih, g)
			}
		}
		rekap.Guru = snapshot
	}

	for _, g := range rekap.Guru {
		rekap.Ringkasan.TotalUjroh += g.Total
		rekap.Ringkasan.JumlahPertemuan += g.JumlahPertemuan
		if g.Dibayar {
			rekap.Ringkasan.TotalDibayar += g.Total
		}
	}
	rekap.Ringkasan.JumlahGuru = int64(len(rekap.Guru))
	rekap.Ringkasan.TotalBelum = rekap.Ringkasan.TotalUjroh - rekap.Ringkasan.TotalDibayar
	return rekap, nil
}

// hitungLive groups the month's completed meetings by teaching guru. Meetings
// whose teacher cannot be resolved are returned separately: nobody is paid for
// them until the class or meeting data is fixed.
func (s *UjrohService) hitungLive(ctx context.Context, q *queries.Querier, bulan string) ([]models.UjrohGuru, []models.UjrohPertemuan, error) {
	rows, err := q.ListUjrohPertemuanLive(ctx, bulan)
	if err != nil {
		return nil, nil, err
	}
	byGuru := map[int64]*models.UjrohGuru{}
	order := []int64{}
	tanpaGuru := []models.UjrohPertemuan{}
	for _, r := range rows {
		p := models.UjrohPertemuan{
			PertemuanID: r.PertemuanID,
			KelasID:     r.KelasID,
			KelasNama:   r.KelasNama,
			Tanggal:     r.Tanggal,
			PertemuanKe: r.PertemuanKe,
			IsBadal:     r.IsBadal == 1,
			Tarif:       r.Tarif,
		}
		if !r.GuruID.Valid {
			tanpaGuru = append(tanpaGuru, p)
			continue
		}
		g, ok := byGuru[r.GuruID.Int64]
		if !ok {
			g = &models.UjrohGuru{
				GuruID:      r.GuruID.Int64,
				GuruNama:    r.GuruNama,
				GuruStatus:  r.GuruStatus,
				Tarif:       r.Tarif,
				TarifKhusus: r.TarifKhusus == 1,
				Pertemuan:   []models.UjrohPertemuan{},
			}
			byGuru[r.GuruID.Int64] = g
			order = append(order, r.GuruID.Int64)
		}
		g.JumlahPertemuan++
		if p.IsBadal {
			g.JumlahBadal++
		}
		g.Total += r.Tarif
		g.Pertemuan = append(g.Pertemuan, p)
	}
	out := make([]models.UjrohGuru, 0, len(order))
	for _, id := range order {
		out = append(out, *byGuru[id])
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].GuruNama) < strings.ToLower(out[j].GuruNama) })
	return out, tanpaGuru, nil
}

func (s *UjrohService) snapshot(ctx context.Context, bulan string) ([]models.UjrohGuru, error) {
	guruRows, err := s.querier.ListUjrohGuru(ctx, bulan)
	if err != nil {
		return nil, err
	}
	pertemuanRows, err := s.querier.ListUjrohPertemuan(ctx, bulan)
	if err != nil {
		return nil, err
	}
	perGuru := map[int64][]models.UjrohPertemuan{}
	for _, r := range pertemuanRows {
		perGuru[r.GuruID] = append(perGuru[r.GuruID], models.UjrohPertemuan{
			PertemuanID: r.PertemuanID,
			KelasID:     r.KelasID,
			KelasNama:   r.KelasNama,
			Tanggal:     r.Tanggal,
			PertemuanKe: r.PertemuanKe,
			IsBadal:     r.IsBadal == 1,
			Tarif:       r.Tarif,
		})
	}
	out := make([]models.UjrohGuru, 0, len(guruRows))
	for _, r := range guruRows {
		items := perGuru[r.GuruID]
		if items == nil {
			items = []models.UjrohPertemuan{}
		}
		out = append(out, models.UjrohGuru{
			GuruID:          r.GuruID,
			GuruNama:        r.GuruNama,
			GuruStatus:      r.GuruStatus,
			Tarif:           r.Tarif,
			JumlahPertemuan: r.JumlahPertemuan,
			JumlahBadal:     r.JumlahBadal,
			Total:           r.Total,
			Dibayar:         r.DibayarAt.Valid,
			DibayarAt:       formatNullDate(r.DibayarAt),
			DibayarOleh:     r.DibayarOleh,
			CatatanBayar:    r.CatatanBayar,
			Pertemuan:       items,
		})
	}
	return out, nil
}

// Kunci freezes the month: the current calculation is stored and later rate,
// status or meeting changes no longer alter it.
func (s *UjrohService) Kunci(bulan string, userID int64) error {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return err
	}
	if bulan > s.now().Format("2006-01") {
		return ErrUjrohBulanMendatang
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	locked, err := q.KunciUjrohBulan(ctx, bulan, userID, time.Now())
	if err != nil {
		return err
	}
	if !locked {
		return ErrUjrohSudahDikunci
	}
	live, _, err := s.hitungLive(ctx, q, bulan)
	if err != nil {
		return err
	}
	for _, g := range live {
		if err := q.InsertUjrohGuru(ctx, bulan, queries.UjrohGuruRow{
			GuruID:          g.GuruID,
			GuruNama:        g.GuruNama,
			GuruStatus:      g.GuruStatus,
			Tarif:           g.Tarif,
			JumlahPertemuan: g.JumlahPertemuan,
			JumlahBadal:     g.JumlahBadal,
			Total:           g.Total,
		}); err != nil {
			return err
		}
		for _, p := range g.Pertemuan {
			if err := q.InsertUjrohPertemuan(ctx, bulan, queries.UjrohPertemuanRow{
				GuruID:      g.GuruID,
				PertemuanID: p.PertemuanID,
				KelasID:     p.KelasID,
				KelasNama:   p.KelasNama,
				Tanggal:     p.Tanggal,
				PertemuanKe: p.PertemuanKe,
				IsBadal:     boolToInt64(p.IsBadal),
				Tarif:       p.Tarif,
			}); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// Buka unlocks a month so it recalculates live. Refused once any guru has been
// marked paid, so a paid figure can never silently change.
func (s *UjrohService) Buka(bulan string, userID int64) error {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	paid, err := q.CountUjrohGuruDibayar(ctx, bulan)
	if err != nil {
		return err
	}
	if paid > 0 {
		return ErrUjrohSudahAdaDibayar
	}
	unlocked, err := q.BukaUjrohBulan(ctx, bulan, userID, time.Now())
	if err != nil {
		return err
	}
	if !unlocked {
		return ErrUjrohBelumDikunci
	}
	return tx.Commit()
}

func (s *UjrohService) TandaiDibayar(bulan string, guruID, userID int64, req models.TandaiUjrohDibayarRequest) error {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return err
	}
	tanggal := s.now()
	if strings.TrimSpace(req.Tanggal) != "" {
		t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.Tanggal), ZonaWaktuRiayah)
		if err != nil {
			return ErrUjrohTanggalTidakSah
		}
		tanggal = t
	}
	return s.setDibayar(bulan, guruID, sql.NullTime{Time: tanggal, Valid: true}, sql.NullInt64{Int64: userID, Valid: true}, strings.TrimSpace(req.Catatan))
}

func (s *UjrohService) BatalDibayar(bulan string, guruID int64) error {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return err
	}
	return s.setDibayar(bulan, guruID, sql.NullTime{}, sql.NullInt64{}, "")
}

func (s *UjrohService) setDibayar(bulan string, guruID int64, at sql.NullTime, userID sql.NullInt64, catatan string) error {
	ctx := context.Background()
	status, err := s.querier.GetUjrohBulan(ctx, bulan)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !status.DikunciAt.Valid) {
		return ErrUjrohBelumDikunci
	} else if err != nil {
		return err
	}
	ok, err := s.querier.SetUjrohGuruDibayar(ctx, bulan, guruID, at, userID, catatan)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUjrohGuruTidakAda
	}
	return nil
}

func (s *UjrohService) GetTarif() (models.UjrohTarif, error) {
	rows, err := s.querier.ListUjrohTarif(context.Background())
	if err != nil {
		return models.UjrohTarif{}, err
	}
	var tarif models.UjrohTarif
	for _, r := range rows {
		switch r.Status {
		case "tetap":
			tarif.Tetap = r.Nominal
		case "part_time":
			tarif.PartTime = r.Nominal
		}
	}
	return tarif, nil
}

func (s *UjrohService) UpdateTarif(req models.UpdateUjrohTarifRequest, userID int64) error {
	if req.Tetap < 0 || req.PartTime < 0 {
		return ErrUjrohNominalTidakSah
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)
	if err := q.UpsertUjrohTarif(ctx, "tetap", req.Tetap, userID); err != nil {
		return err
	}
	if err := q.UpsertUjrohTarif(ctx, "part_time", req.PartTime, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *UjrohService) ListTarifGuru() ([]models.UjrohTarifGuru, error) {
	rows, err := s.querier.ListUjrohTarifGuru(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.UjrohTarifGuru, 0, len(rows))
	for _, r := range rows {
		item := models.UjrohTarifGuru{GuruID: r.GuruID, Nama: r.Nama, Status: r.Status, IsAktif: r.IsAktif == 1}
		if r.TarifKhusus.Valid {
			v := r.TarifKhusus.Int64
			item.TarifKhusus = &v
		}
		out = append(out, item)
	}
	return out, nil
}

// SetTarifGuru sets a per-guru rate; a nil nominal removes it so the guru
// falls back to the status default.
func (s *UjrohService) SetTarifGuru(guruID int64, nominal *int64, userID int64) error {
	ctx := context.Background()
	if nominal == nil {
		return s.querier.DeleteUjrohTarifGuru(ctx, guruID)
	}
	if *nominal < 0 {
		return ErrUjrohNominalTidakSah
	}
	if _, err := s.querier.GetGuruByID(ctx, guruID); err != nil {
		return fmt.Errorf("guru tidak ditemukan: %w", err)
	}
	return s.querier.UpsertUjrohTarifGuru(ctx, guruID, *nominal, userID)
}

func formatNullDateTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(ZonaWaktuRiayah).Format("2006-01-02 15:04")
}

func formatNullDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(ZonaWaktuRiayah).Format("2006-01-02")
}
