package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	StatusSantriAktif       = "aktif"
	StatusSantriCuti        = "cuti"
	StatusSantriNonaktif    = "nonaktif"
	StatusSantriTidakLanjut = "tidak_lanjut"

	alasanCutiBerakhir       = "Cuti berakhir (otomatis)"
	alasanCutiTerjadwalLewat = "Cuti terjadwal terlewat tanpa dimulai (otomatis)"
)

var (
	ErrStatusSantriTidakValid   = errors.New("status harus aktif, cuti, atau nonaktif")
	ErrStatusSantriTerkunci     = errors.New("santri berstatus Tidak Lanjut diatur oleh Keuangan")
	ErrStatusSantriBukanDiKelas = errors.New("santri tidak terdaftar di kelas ini")
	// SelesaiPertemuan requires attendance for exactly the aktif santri, so the
	// roster must not change while a meeting is in progress.
	ErrRosterPertemuanBerlangsung = errors.New("kelas sedang ada pertemuan berlangsung; coba lagi setelah guru menyelesaikan pertemuan")
)

// statusCutiTerjadwal is the log label for a cuti recorded ahead of its start
// date: the santri stays aktif until cuti_mulai, then the background job
// switches them to cuti.
const statusCutiTerjadwal = "cuti_terjadwal"

// ensureTanpaPertemuanBerlangsung returns ErrRosterPertemuanBerlangsung when the
// class has a meeting in progress.
func ensureTanpaPertemuanBerlangsung(ctx context.Context, q *queries.Querier, kelasID sql.NullInt64) error {
	if !kelasID.Valid {
		return nil
	}
	if _, err := q.GetActivePertemuanByKelas(ctx, kelasID.Int64); err == nil {
		return ErrRosterPertemuanBerlangsung
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

// wib is the timezone used for cuti dates; the pesantren operates in WIB.
var wib = time.FixedZone("WIB", 7*60*60)

func tanggalWIB(t time.Time) string {
	return t.In(wib).Format(time.DateOnly)
}

// SantriStatusService manages the aktif/cuti/nonaktif lifecycle set by Admin
// Kelas and records every status transition in santri_status_log.
type SantriStatusService struct {
	querier *queries.Querier
	now     func() time.Time
}

func NewSantriStatusService(querier *queries.Querier) *SantriStatusService {
	return &SantriStatusService{querier: querier, now: time.Now}
}

type statusSantriBaru struct {
	status      string
	alasan      string
	cutiMulai   string
	cutiSelesai string
}

func (s *SantriStatusService) validasi(req models.UpdateSantriStatusRequest) (statusSantriBaru, error) {
	baru := statusSantriBaru{status: strings.TrimSpace(req.Status), alasan: strings.TrimSpace(req.Alasan)}
	switch baru.status {
	case StatusSantriAktif:
		baru.alasan = ""
		return baru, nil
	case StatusSantriNonaktif:
		if baru.alasan == "" {
			return baru, errors.New("alasan nonaktif wajib diisi")
		}
		return baru, nil
	case StatusSantriCuti:
	default:
		return baru, ErrStatusSantriTidakValid
	}

	if baru.alasan == "" {
		return baru, errors.New("alasan cuti wajib diisi")
	}
	mulai, err := time.Parse(time.DateOnly, strings.TrimSpace(req.CutiMulai))
	if err != nil {
		return baru, errors.New("tanggal mulai cuti tidak valid")
	}
	selesai, err := time.Parse(time.DateOnly, strings.TrimSpace(req.CutiSelesai))
	if err != nil {
		return baru, errors.New("tanggal selesai cuti tidak valid")
	}
	if selesai.Before(mulai) {
		return baru, errors.New("tanggal selesai cuti tidak boleh sebelum tanggal mulai")
	}
	baru.cutiMulai = mulai.Format(time.DateOnly)
	baru.cutiSelesai = selesai.Format(time.DateOnly)
	if baru.cutiSelesai < tanggalWIB(s.now()) {
		return baru, errors.New("tanggal selesai cuti sudah lewat")
	}
	return baru, nil
}

// UbahStatus changes a santri's status from the class detail page. Only
// aktif/cuti/nonaktif are allowed; tidak_lanjut stays under Keuangan's control.
func (s *SantriStatusService) UbahStatus(kelasID, santriID, actorID int64, req models.UpdateSantriStatusRequest) error {
	baru, err := s.validasi(req)
	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	santri, err := q.GetSantriByID(ctx, santriID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrStatusSantriBukanDiKelas
		}
		return err
	}
	if !santri.KelasID.Valid || santri.KelasID.Int64 != kelasID {
		return ErrStatusSantriBukanDiKelas
	}
	if santri.Status == StatusSantriTidakLanjut {
		return ErrStatusSantriTerkunci
	}
	// A cuti starting after today is only scheduled: the santri stays aktif
	// (and in attendance/billing) until cuti_mulai.
	efektif := baru
	if baru.status == StatusSantriCuti && baru.cutiMulai > tanggalWIB(s.now()) {
		efektif.status = StatusSantriAktif
	}
	// Setting aktif on an aktif santri is allowed when it cancels a scheduled cuti.
	batalkanCutiTerjadwal := santri.Status == StatusSantriAktif && baru.status == StatusSantriAktif && santri.CutiMulai != ""
	if santri.Status == baru.status && baru.status != StatusSantriCuti && !batalkanCutiTerjadwal {
		return fmt.Errorf("status santri sudah %s", baru.status)
	}
	if (santri.Status == StatusSantriAktif) != (efektif.status == StatusSantriAktif) {
		if err := ensureTanpaPertemuanBerlangsung(ctx, q, santri.KelasID); err != nil {
			return err
		}
	}

	// Nonaktif released the seat; taking it back needs room in the class.
	if santri.Status == StatusSantriNonaktif && baru.status != StatusSantriNonaktif {
		kelas, err := q.GetKelasByID(ctx, kelasID)
		if err != nil {
			return err
		}
		if kelas.JumlahSantri >= kelas.Kapasitas {
			return fmt.Errorf("kelas sudah penuh (kapasitas %d)", kelas.Kapasitas)
		}
	}

	actor := sql.NullInt64{Int64: actorID, Valid: actorID > 0}
	if efektif.status != baru.status {
		if err := s.simpanDenganLabel(ctx, q, santri, efektif, statusCutiTerjadwal, actor); err != nil {
			return err
		}
	} else if err := s.simpan(ctx, q, santri, baru, actor); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SantriStatusService) simpan(ctx context.Context, q *queries.Querier, santri queries.Santri, baru statusSantriBaru, actor sql.NullInt64) error {
	return s.simpanDenganLabel(ctx, q, santri, baru, baru.status, actor)
}

// simpanDenganLabel stores baru on the santri but records logStatus as the new
// status in the history (used for scheduled cuti, which stays aktif for now).
func (s *SantriStatusService) simpanDenganLabel(ctx context.Context, q *queries.Querier, santri queries.Santri, baru statusSantriBaru, logStatus string, actor sql.NullInt64) error {
	if err := q.UpdateSantriStatusDetail(ctx, queries.UpdateSantriStatusDetailParams{
		Status:       baru.status,
		StatusAlasan: baru.alasan,
		CutiMulai:    baru.cutiMulai,
		CutiSelesai:  baru.cutiSelesai,
		UpdatedAt:    s.now(),
		ID:           santri.ID,
	}); err != nil {
		return err
	}
	return q.CreateSantriStatusLog(ctx, queries.CreateSantriStatusLogParams{
		SantriID:    santri.ID,
		KelasID:     santri.KelasID,
		StatusLama:  santri.Status,
		StatusBaru:  logStatus,
		Alasan:      baru.alasan,
		CutiMulai:   baru.cutiMulai,
		CutiSelesai: baru.cutiSelesai,
		DibuatOleh:  actor,
	})
}

// AktifkanCutiBerakhir reactivates santri whose last cuti day has passed (WIB).
// It is idempotent and safe to run repeatedly from a background ticker. A
// santri whose class has a meeting in progress is skipped until the next run,
// and one failing santri does not block the rest.
func (s *SantriStatusService) AktifkanCutiBerakhir() (int, error) {
	ctx := context.Background()
	list, err := s.querier.ListSantriCutiBerakhir(ctx, tanggalWIB(s.now()))
	if err != nil {
		return 0, err
	}
	total := 0
	var errs []error
	for _, santri := range list {
		ok, err := s.aktifkanDariCuti(ctx, santri.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("santri %d: %w", santri.ID, err))
			continue
		}
		if ok {
			total++
		}
	}
	return total, errors.Join(errs...)
}

func (s *SantriStatusService) aktifkanDariCuti(ctx context.Context, santriID int64) (bool, error) {
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	santri, err := q.GetSantriByID(ctx, santriID)
	if err != nil {
		return false, err
	}
	if santri.Status != StatusSantriCuti {
		return false, nil
	}
	if err := ensureTanpaPertemuanBerlangsung(ctx, q, santri.KelasID); err != nil {
		if errors.Is(err, ErrRosterPertemuanBerlangsung) {
			return false, nil
		}
		return false, err
	}
	if err := s.simpan(ctx, q, santri, statusSantriBaru{status: StatusSantriAktif, alasan: alasanCutiBerakhir}, sql.NullInt64{}); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// MulaiCutiTerjadwal switches aktif santri to cuti once their scheduled
// cuti_mulai arrives (WIB). A schedule whose end already passed is dropped.
// Like AktifkanCutiBerakhir it skips classes with a meeting in progress.
func (s *SantriStatusService) MulaiCutiTerjadwal() (int, error) {
	ctx := context.Background()
	hariIni := tanggalWIB(s.now())
	list, err := s.querier.ListSantriCutiTerjadwal(ctx, hariIni)
	if err != nil {
		return 0, err
	}
	total := 0
	var errs []error
	for _, santri := range list {
		ok, err := s.mulaiCuti(ctx, santri.ID, hariIni)
		if err != nil {
			errs = append(errs, fmt.Errorf("santri %d: %w", santri.ID, err))
			continue
		}
		if ok {
			total++
		}
	}
	return total, errors.Join(errs...)
}

func (s *SantriStatusService) mulaiCuti(ctx context.Context, santriID int64, hariIni string) (bool, error) {
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	santri, err := q.GetSantriByID(ctx, santriID)
	if err != nil {
		return false, err
	}
	if santri.Status != StatusSantriAktif || santri.CutiMulai == "" || santri.CutiMulai > hariIni {
		return false, nil
	}
	if santri.CutiSelesai < hariIni {
		// The whole cuti window passed without the job running; clear it and
		// record why, so the history doesn't end at "cuti terjadwal".
		if err := s.simpan(ctx, q, santri, statusSantriBaru{status: StatusSantriAktif, alasan: alasanCutiTerjadwalLewat}, sql.NullInt64{}); err != nil {
			return false, err
		}
		return false, tx.Commit()
	}
	if err := ensureTanpaPertemuanBerlangsung(ctx, q, santri.KelasID); err != nil {
		if errors.Is(err, ErrRosterPertemuanBerlangsung) {
			return false, nil
		}
		return false, err
	}
	baru := statusSantriBaru{status: StatusSantriCuti, alasan: santri.StatusAlasan, cutiMulai: santri.CutiMulai, cutiSelesai: santri.CutiSelesai}
	if err := s.simpan(ctx, q, santri, baru, sql.NullInt64{}); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// StartAutoAktifCuti runs AktifkanCutiBerakhir and MulaiCutiTerjadwal on startup
// and then hourly, so cuti starts and ends take effect within the first hour.
func (s *SantriStatusService) StartAutoAktifCuti(interval time.Duration, logf func(job string, total int, err error)) {
	run := func() {
		total, err := s.AktifkanCutiBerakhir()
		logf("auto-aktif cuti", total, err)
		total, err = s.MulaiCutiTerjadwal()
		logf("mulai cuti terjadwal", total, err)
	}
	go func() {
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}

func (s *SantriStatusService) ListRiwayatKelas(kelasID int64) ([]models.SantriStatusLogResponse, error) {
	rows, err := s.querier.ListSantriStatusLogByKelas(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]models.SantriStatusLogResponse, len(rows))
	for i, r := range rows {
		out[i] = models.SantriStatusLogResponse{
			ID:             r.ID,
			SantriID:       r.SantriID,
			SantriNama:     r.SantriNama,
			StatusLama:     r.StatusLama,
			StatusBaru:     r.StatusBaru,
			Alasan:         r.Alasan,
			CutiMulai:      r.CutiMulai,
			CutiSelesai:    r.CutiSelesai,
			DibuatOlehNama: r.DibuatOlehNama,
			CreatedAt:      r.CreatedAt.In(wib).Format("2006-01-02 15:04"),
		}
	}
	return out, nil
}
