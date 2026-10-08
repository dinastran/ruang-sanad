package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// KunjunganService handles Koordinator Guru class visits: rubric scoring,
// sending the result to the guru, the guru's response, and follow-ups.
//
// A visit is a draft until it is sent. Sending locks the scores and notes;
// the koordinator must unlock it before editing, and sending again notifies
// the guru again.
type KunjunganService struct {
	querier *queries.Querier
}

func NewKunjunganService(querier *queries.Querier) *KunjunganService {
	return &KunjunganService{querier: querier}
}

const (
	TindakLanjutApresiasi  = "apresiasi"
	TindakLanjutMonitoring = "monitoring"
	TindakLanjutKoordinasi = "koordinasi"
	TindakLanjutCoaching   = "coaching"
	TindakLanjutPembinaan  = "pembinaan"

	maxTanggapanKunjungan = 2000
)

var tindakLanjutLabel = map[string]string{
	TindakLanjutApresiasi:  "Apresiasi & reward",
	TindakLanjutMonitoring: "Monitoring kunjungan berikutnya",
	TindakLanjutKoordinasi: "Koordinasi dengan pihak terkait",
	TindakLanjutCoaching:   "Coaching",
	TindakLanjutPembinaan:  "Pembinaan",
}

// tindakLanjutInternal marks follow-ups that stay between koordinator and
// admin and are never shown to the guru.
func tindakLanjutInternal(jenis string) bool {
	return jenis == TindakLanjutKoordinasi
}

var kunjunganAspek = []struct{ kode, label string }{
	{"kedisiplinan", "Kedisiplinan & adab"},
	{"materi", "Penguasaan materi"},
	{"metode", "Metode & pengelolaan kelas"},
	{"interaksi", "Interaksi dengan santri"},
}

func predikatNilai(n float64) string {
	switch {
	case n <= 0:
		return ""
	case n >= 3.5:
		return "Sangat baik"
	case n >= 2.5:
		return "Baik"
	case n >= 1.5:
		return "Cukup"
	default:
		return "Perlu perbaikan"
	}
}

func roundNilai(n float64) float64 {
	return math.Round(n*100) / 100
}

// kunjunganRow is the shared shape of the ListKunjungan/GetKunjungan/
// ListKunjunganTerkirimByGuru rows; sqlc emits one identical struct per query.
type kunjunganRow = queries.GetKunjunganRow

// ============ Koordinator ============

func (s *KunjunganService) Dashboard(today time.Time) (models.KunjunganDashboard, error) {
	ctx := context.Background()
	rows, err := s.querier.ListKunjungan(ctx)
	if err != nil {
		return models.KunjunganDashboard{}, err
	}
	tlRows, err := s.querier.ListTindakLanjut(ctx)
	if err != nil {
		return models.KunjunganDashboard{}, err
	}
	tlByKunjungan := map[int64][]models.TindakLanjutResponse{}
	terbuka := []models.TindakLanjutResponse{}
	terbukaPerGuru := map[int64]int{}
	for _, r := range tlRows {
		tl := tindakLanjutResponse(queries.KunjunganTindakLanjut{
			ID: r.ID, KunjunganID: r.KunjunganID, Jenis: r.Jenis, Catatan: r.Catatan,
			TargetTanggal: r.TargetTanggal, Status: r.Status, SelesaiAt: r.SelesaiAt,
			KunjunganBerikutnyaID: r.KunjunganBerikutnyaID,
		}, today)
		tlByKunjungan[r.KunjunganID] = append(tlByKunjungan[r.KunjunganID], tl)
		if r.Status == "terbuka" {
			tl.GuruID, tl.GuruNama, tl.KelasNama = r.GuruID, r.GuruNama, r.KelasNama
			tl.KunjunganTanggal = nullDateStr(r.KunjunganTanggal)
			terbuka = append(terbuka, tl)
			terbukaPerGuru[r.GuruID]++
		}
	}

	out := models.KunjunganDashboard{
		Kunjungan:           make([]models.KunjunganResponse, 0, len(rows)),
		TindakLanjutTerbuka: terbuka,
		BelumDitanggapi:     []models.KunjunganResponse{},
	}
	terkirimPerGuru := map[int64][]models.KunjunganResponse{}
	for _, r := range rows {
		k := kunjunganResponse(kunjunganRow(r))
		k.TindakLanjut = tlByKunjungan[k.ID]
		if k.TindakLanjut == nil {
			k.TindakLanjut = []models.TindakLanjutResponse{}
		}
		out.Kunjungan = append(out.Kunjungan, k)
		if k.StatusKirim != "draft" {
			terkirimPerGuru[k.GuruID] = append(terkirimPerGuru[k.GuruID], k)
		}
		if k.StatusKirim == "terkirim" || k.StatusKirim == "dibaca" {
			out.BelumDitanggapi = append(out.BelumDitanggapi, k)
		}
	}
	out.Rekap = rekapKunjunganGuru(terkirimPerGuru, terbukaPerGuru)
	return out, nil
}

// rekapKunjunganGuru expects each guru's visits newest first (ListKunjungan
// order) so the first two entries are the latest and the previous one.
func rekapKunjunganGuru(perGuru map[int64][]models.KunjunganResponse, terbuka map[int64]int) []models.KunjunganRekapGuru {
	out := make([]models.KunjunganRekapGuru, 0, len(perGuru))
	for guruID, list := range perGuru {
		r := models.KunjunganRekapGuru{
			GuruID:              guruID,
			GuruNama:            list[0].GuruNama,
			JumlahKunjungan:     len(list),
			TanggalTerakhir:     list[0].Tanggal,
			NilaiTerakhir:       list[0].NilaiRataRata,
			PredikatTerakhir:    list[0].Predikat,
			TindakLanjutTerbuka: terbuka[guruID],
		}
		if len(list) > 1 {
			r.NilaiSebelumnya = list[1].NilaiRataRata
			switch {
			case r.NilaiTerakhir > r.NilaiSebelumnya:
				r.Tren = "naik"
			case r.NilaiTerakhir < r.NilaiSebelumnya:
				r.Tren = "turun"
			default:
				r.Tren = "tetap"
			}
		}
		for i, a := range kunjunganAspek {
			var total float64
			for _, k := range list {
				total += float64(k.Aspek[i].Nilai)
			}
			r.RataRataAspek = append(r.RataRataAspek, models.KunjunganAspekRataRata{
				Kode: a.kode, Label: a.label, RataRata: roundNilai(total / float64(len(list))),
			})
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NilaiTerakhir != out[j].NilaiTerakhir {
			return out[i].NilaiTerakhir < out[j].NilaiTerakhir
		}
		return out[i].GuruNama < out[j].GuruNama
	})
	return out
}

func (s *KunjunganService) Detail(id int64, today time.Time) (models.KunjunganResponse, error) {
	ctx := context.Background()
	row, err := s.querier.GetKunjungan(ctx, id)
	if err != nil {
		return models.KunjunganResponse{}, fmt.Errorf("kunjungan tidak ditemukan")
	}
	tl, err := s.querier.ListTindakLanjutByKunjungan(ctx, id)
	if err != nil {
		return models.KunjunganResponse{}, err
	}
	k := kunjunganResponse(row)
	k.TindakLanjut = make([]models.TindakLanjutResponse, 0, len(tl))
	for _, t := range tl {
		k.TindakLanjut = append(k.TindakLanjut, tindakLanjutResponse(t, today))
	}
	if k.StatusKirim != "draft" {
		k.WALink = kunjunganWALink(row.GuruNoWa, k)
	}
	return k, nil
}

func (s *KunjunganService) Create(req models.KunjunganRequest) (int64, error) {
	req, err := normalizeKunjunganRequest(req)
	if err != nil {
		return 0, err
	}
	return s.querier.CreateKunjungan(context.Background(), queries.CreateKunjunganParams{
		GuruID:            req.GuruID,
		KelasID:           nullInt(req.KelasID),
		TargetMulai:       parseNullableDate(req.TargetMulai),
		TargetSelesai:     parseNullableDate(req.TargetSelesai),
		Tanggal:           parseNullableDate(req.Tanggal),
		Jam:               req.Jam,
		Status:            req.Status,
		Catatan:           req.Catatan,
		NilaiKedisiplinan: nullInt(req.NilaiKedisiplinan),
		NilaiMateri:       nullInt(req.NilaiMateri),
		NilaiMetode:       nullInt(req.NilaiMetode),
		NilaiInteraksi:    nullInt(req.NilaiInteraksi),
	})
}

func (s *KunjunganService) Update(id int64, req models.KunjunganRequest) error {
	req, err := normalizeKunjunganRequest(req)
	if err != nil {
		return err
	}
	n, err := s.querier.UpdateKunjungan(context.Background(), queries.UpdateKunjunganParams{
		GuruID:            req.GuruID,
		KelasID:           nullInt(req.KelasID),
		TargetMulai:       parseNullableDate(req.TargetMulai),
		TargetSelesai:     parseNullableDate(req.TargetSelesai),
		Tanggal:           parseNullableDate(req.Tanggal),
		Jam:               req.Jam,
		Status:            req.Status,
		Catatan:           req.Catatan,
		NilaiKedisiplinan: nullInt(req.NilaiKedisiplinan),
		NilaiMateri:       nullInt(req.NilaiMateri),
		NilaiMetode:       nullInt(req.NilaiMetode),
		NilaiInteraksi:    nullInt(req.NilaiInteraksi),
		ID:                id,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return s.alasanTerkunci(id)
	}
	return nil
}

func (s *KunjunganService) Delete(id int64) error {
	n, err := s.querier.DeleteKunjungan(context.Background(), id)
	if err != nil {
		return err
	}
	if n == 0 {
		return s.alasanTerkunci(id)
	}
	return nil
}

// alasanTerkunci explains why a guarded update/delete touched no row.
func (s *KunjunganService) alasanTerkunci(id int64) error {
	if _, err := s.querier.GetKunjungan(context.Background(), id); err != nil {
		return fmt.Errorf("kunjungan tidak ditemukan")
	}
	return fmt.Errorf("hasil kunjungan sudah dikirim ke guru; buka kunci dulu untuk mengubah")
}

// Kirim sends the visit result to the guru: it locks the result, notifies the
// guru, and closes any monitoring follow-up that scheduled this visit.
func (s *KunjunganService) Kirim(id, userID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	row, err := q.GetKunjungan(ctx, id)
	if err != nil {
		return fmt.Errorf("kunjungan tidak ditemukan")
	}
	if row.DikirimAt.Valid {
		return fmt.Errorf("hasil kunjungan sudah dikirim")
	}
	k := kunjunganResponse(row)
	if k.Status != "terlaksana" {
		return fmt.Errorf("hanya kunjungan berstatus Terlaksana yang bisa dikirim")
	}
	if !k.NilaiLengkap {
		return fmt.Errorf("lengkapi nilai semua aspek sebelum mengirim")
	}
	if strings.TrimSpace(k.Catatan) == "" {
		return fmt.Errorf("catatan kunjungan wajib diisi sebelum mengirim")
	}
	now := time.Now()
	if err := q.KirimKunjungan(ctx, queries.KirimKunjunganParams{
		DikirimAt: sql.NullTime{Time: now, Valid: true}, DikirimOleh: nullUserID(userID), ID: id,
	}); err != nil {
		return err
	}
	if err := q.SelesaikanMonitoringOlehKunjungan(ctx, queries.SelesaikanMonitoringOlehKunjunganParams{
		SelesaiAt: sql.NullTime{Time: now, Valid: true}, KunjunganBerikutnyaID: sql.NullInt64{Int64: id, Valid: true},
	}); err != nil {
		return err
	}
	judul := "Hasil kunjungan kelas"
	if row.DibacaAt.Valid || row.TanggapanAt.Valid {
		judul = "Hasil kunjungan kelas diperbarui"
	}
	pesan := fmt.Sprintf("Koordinator Guru mengirim hasil kunjungan %s pada %s. Nilai rata-rata %s (%s).",
		kelasAtauUmum(k.KelasNama), formatTanggalID(k.Tanggal), formatNilai(k.NilaiRataRata), k.Predikat)
	if err := notifyGuruKunjungan(ctx, q, row.GuruUserID, id, userID, judul, pesan); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *KunjunganService) BukaKunci(id int64) error {
	ctx := context.Background()
	row, err := s.querier.GetKunjungan(ctx, id)
	if err != nil {
		return fmt.Errorf("kunjungan tidak ditemukan")
	}
	if !row.DikirimAt.Valid {
		return fmt.Errorf("hasil kunjungan belum dikirim")
	}
	return s.querier.BukaKunciKunjungan(ctx, id)
}

func (s *KunjunganService) TambahTindakLanjut(kunjunganID int64, req models.TindakLanjutRequest, userID int64) error {
	jenis := strings.TrimSpace(req.Jenis)
	if _, ok := tindakLanjutLabel[jenis]; !ok {
		return fmt.Errorf("jenis tindak lanjut tidak valid")
	}
	catatan := strings.TrimSpace(req.Catatan)
	if len([]rune(catatan)) > 1000 {
		return fmt.Errorf("catatan tindak lanjut maksimal 1000 karakter")
	}
	target := sql.NullTime{}
	if t := strings.TrimSpace(req.TargetTanggal); t != "" {
		parsed, err := parseDateStrict(t)
		if err != nil {
			return fmt.Errorf("target tanggal tidak valid")
		}
		target = sql.NullTime{Time: parsed, Valid: true}
	}
	if jenis == TindakLanjutMonitoring && !target.Valid {
		return fmt.Errorf("target tanggal kunjungan berikutnya wajib diisi untuk monitoring")
	}

	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	row, err := q.GetKunjungan(ctx, kunjunganID)
	if err != nil {
		return fmt.Errorf("kunjungan tidak ditemukan")
	}
	if row.Status != "terlaksana" {
		return fmt.Errorf("tindak lanjut hanya untuk kunjungan yang sudah terlaksana")
	}
	tlID, err := q.CreateTindakLanjut(ctx, queries.CreateTindakLanjutParams{
		KunjunganID: kunjunganID, Jenis: jenis, Catatan: catatan, TargetTanggal: target, DibuatOleh: nullUserID(userID),
	})
	if err != nil {
		return err
	}
	if jenis == TindakLanjutMonitoring {
		ket := fmt.Sprintf("Monitoring lanjutan dari kunjungan %s.", formatTanggalID(nullDateStr(row.Tanggal)))
		if catatan != "" {
			ket += " Fokus: " + catatan
		}
		nextID, err := q.CreateKunjungan(ctx, queries.CreateKunjunganParams{
			GuruID: row.GuruID, KelasID: row.KelasID, TargetMulai: target, TargetSelesai: target,
			Status: "dijadwalkan", Catatan: ket,
		})
		if err != nil {
			return err
		}
		if err := q.SetTindakLanjutKunjunganBerikutnya(ctx, queries.SetTindakLanjutKunjunganBerikutnyaParams{
			KunjunganBerikutnyaID: sql.NullInt64{Int64: nextID, Valid: true}, ID: tlID,
		}); err != nil {
			return err
		}
	}
	if row.DikirimAt.Valid && !tindakLanjutInternal(jenis) {
		pesan := fmt.Sprintf("Tindak lanjut kunjungan %s: %s.", kelasAtauUmum(row.KelasNama), tindakLanjutLabel[jenis])
		if catatan != "" {
			pesan += " " + catatan
		}
		if err := notifyGuruKunjungan(ctx, q, row.GuruUserID, kunjunganID, userID, "Tindak lanjut kunjungan kelas", pesan); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *KunjunganService) SetStatusTindakLanjut(kunjunganID, tlID int64, status string) error {
	ctx := context.Background()
	tl, err := s.querier.GetTindakLanjut(ctx, tlID)
	if err != nil || tl.KunjunganID != kunjunganID {
		return fmt.Errorf("tindak lanjut tidak ditemukan")
	}
	selesai := sql.NullTime{}
	switch status {
	case "selesai":
		selesai = sql.NullTime{Time: time.Now(), Valid: true}
	case "terbuka":
	default:
		return fmt.Errorf("status tindak lanjut tidak valid")
	}
	return s.querier.UpdateTindakLanjutStatus(ctx, queries.UpdateTindakLanjutStatusParams{Status: status, SelesaiAt: selesai, ID: tlID})
}

// HapusTindakLanjut removes a follow-up. A monitoring follow-up also removes
// the visit it scheduled while that visit is still an untouched schedule.
func (s *KunjunganService) HapusTindakLanjut(kunjunganID, tlID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	tl, err := q.GetTindakLanjut(ctx, tlID)
	if err != nil || tl.KunjunganID != kunjunganID {
		return fmt.Errorf("tindak lanjut tidak ditemukan")
	}
	if err := q.DeleteTindakLanjut(ctx, tlID); err != nil {
		return err
	}
	if tl.Jenis == TindakLanjutMonitoring && tl.KunjunganBerikutnyaID.Valid {
		next, err := q.GetKunjungan(ctx, tl.KunjunganBerikutnyaID.Int64)
		if err == nil && next.Status == "dijadwalkan" && !next.DikirimAt.Valid {
			if _, err := q.DeleteKunjungan(ctx, next.ID); err != nil {
				return err
			}
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	return tx.Commit()
}

// ============ Guru ============

// ListUntukGuru returns the results sent to a guru, without internal
// follow-ups.
func (s *KunjunganService) ListUntukGuru(guruID int64, today time.Time) ([]models.KunjunganResponse, error) {
	ctx := context.Background()
	rows, err := s.querier.ListKunjunganTerkirimByGuru(ctx, guruID)
	if err != nil {
		return nil, err
	}
	out := make([]models.KunjunganResponse, 0, len(rows))
	for _, r := range rows {
		k := kunjunganResponse(kunjunganRow(r))
		tl, err := s.querier.ListTindakLanjutByKunjungan(ctx, k.ID)
		if err != nil {
			return nil, err
		}
		k.TindakLanjut = tindakLanjutUntukGuru(tl, today)
		out = append(out, k)
	}
	return out, nil
}

// DetailUntukGuru returns one sent result owned by guruID and marks it read.
func (s *KunjunganService) DetailUntukGuru(id, guruID int64, today time.Time) (models.KunjunganResponse, error) {
	ctx := context.Background()
	row, err := s.querier.GetKunjungan(ctx, id)
	if err != nil || row.GuruID != guruID || !row.DikirimAt.Valid {
		return models.KunjunganResponse{}, fmt.Errorf("hasil kunjungan tidak ditemukan")
	}
	if !row.DibacaAt.Valid {
		now := time.Now()
		if err := s.querier.TandaiKunjunganDibaca(ctx, queries.TandaiKunjunganDibacaParams{
			DibacaAt: sql.NullTime{Time: now, Valid: true}, ID: id, GuruID: guruID,
		}); err != nil {
			return models.KunjunganResponse{}, err
		}
		row.DibacaAt = sql.NullTime{Time: now, Valid: true}
	}
	tl, err := s.querier.ListTindakLanjutByKunjungan(ctx, id)
	if err != nil {
		return models.KunjunganResponse{}, err
	}
	k := kunjunganResponse(row)
	k.TindakLanjut = tindakLanjutUntukGuru(tl, today)
	return k, nil
}

func (s *KunjunganService) SimpanTanggapan(id, guruID int64, tanggapan string) error {
	tanggapan = strings.TrimSpace(tanggapan)
	if tanggapan == "" {
		return fmt.Errorf("tanggapan tidak boleh kosong")
	}
	if len([]rune(tanggapan)) > maxTanggapanKunjungan {
		return fmt.Errorf("tanggapan maksimal %d karakter", maxTanggapanKunjungan)
	}
	now := sql.NullTime{Time: time.Now(), Valid: true}
	n, err := s.querier.SimpanTanggapanKunjungan(context.Background(), queries.SimpanTanggapanKunjunganParams{
		TanggapanGuru: tanggapan, Waktu: now, ID: id, GuruID: guruID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("hasil kunjungan tidak ditemukan")
	}
	return nil
}

// ============ Helpers ============

func normalizeKunjunganRequest(req models.KunjunganRequest) (models.KunjunganRequest, error) {
	if req.GuruID < 1 {
		return req, fmt.Errorf("guru wajib dipilih")
	}
	req.Jam = strings.TrimSpace(req.Jam)
	req.Catatan = strings.TrimSpace(req.Catatan)
	switch req.Status {
	case "terlaksana", "ditunda", "batal", "dijadwalkan":
	default:
		req.Status = "dijadwalkan"
	}
	for _, n := range []int64{req.NilaiKedisiplinan, req.NilaiMateri, req.NilaiMetode, req.NilaiInteraksi} {
		if n < 0 || n > 4 {
			return req, fmt.Errorf("nilai aspek harus 1 sampai 4")
		}
	}
	if req.Status != "terlaksana" {
		// Scores only describe a visit that happened.
		req.NilaiKedisiplinan, req.NilaiMateri, req.NilaiMetode, req.NilaiInteraksi = 0, 0, 0, 0
		return req, nil
	}
	if _, err := parseDateStrict(req.Tanggal); err != nil {
		return req, fmt.Errorf("tanggal pelaksanaan wajib dan harus valid")
	}
	if req.Jam == "" {
		return req, fmt.Errorf("jam masuk wajib diisi untuk kunjungan terlaksana")
	}
	if req.Catatan == "" {
		return req, fmt.Errorf("catatan kunjungan wajib diisi")
	}
	return req, nil
}

func kunjunganResponse(r kunjunganRow) models.KunjunganResponse {
	var kelasID *int64
	if r.KelasID.Valid {
		kelasID = &r.KelasID.Int64
	}
	nilai := []int64{r.NilaiKedisiplinan.Int64, r.NilaiMateri.Int64, r.NilaiMetode.Int64, r.NilaiInteraksi.Int64}
	k := models.KunjunganResponse{
		ID:                r.ID,
		GuruID:            r.GuruID,
		GuruNama:          r.GuruNama,
		KelasID:           kelasID,
		KelasNama:         r.KelasNama,
		TargetMulai:       nullDateStr(r.TargetMulai),
		TargetSelesai:     nullDateStr(r.TargetSelesai),
		Tanggal:           nullDateStr(r.Tanggal),
		Jam:               r.Jam,
		Status:            r.Status,
		Catatan:           r.Catatan,
		NilaiKedisiplinan: nilai[0],
		NilaiMateri:       nilai[1],
		NilaiMetode:       nilai[2],
		NilaiInteraksi:    nilai[3],
		Aspek:             make([]models.KunjunganNilaiAspek, 0, len(kunjunganAspek)),
		NilaiLengkap:      true,
		DikirimAt:         nullDateTimeStr(r.DikirimAt),
		DibacaAt:          nullDateTimeStr(r.DibacaAt),
		TanggapanGuru:     r.TanggapanGuru,
		TanggapanAt:       nullDateTimeStr(r.TanggapanAt),
		TindakLanjut:      []models.TindakLanjutResponse{},
	}
	var total int64
	for i, a := range kunjunganAspek {
		k.Aspek = append(k.Aspek, models.KunjunganNilaiAspek{Kode: a.kode, Label: a.label, Nilai: nilai[i], Predikat: predikatNilai(float64(nilai[i]))})
		if nilai[i] == 0 {
			k.NilaiLengkap = false
		}
		total += nilai[i]
	}
	if k.NilaiLengkap {
		k.NilaiRataRata = roundNilai(float64(total) / float64(len(kunjunganAspek)))
		k.Predikat = predikatNilai(k.NilaiRataRata)
	}
	switch {
	case !r.DikirimAt.Valid:
		k.StatusKirim = "draft"
	case r.TanggapanAt.Valid:
		k.StatusKirim = "ditanggapi"
	case r.DibacaAt.Valid:
		k.StatusKirim = "dibaca"
	default:
		k.StatusKirim = "terkirim"
	}
	return k
}

func tindakLanjutResponse(t queries.KunjunganTindakLanjut, today time.Time) models.TindakLanjutResponse {
	var next *int64
	if t.KunjunganBerikutnyaID.Valid {
		next = &t.KunjunganBerikutnyaID.Int64
	}
	target := nullDateStr(t.TargetTanggal)
	return models.TindakLanjutResponse{
		ID:                    t.ID,
		KunjunganID:           t.KunjunganID,
		Jenis:                 t.Jenis,
		JenisLabel:            tindakLanjutLabel[t.Jenis],
		Internal:              tindakLanjutInternal(t.Jenis),
		Catatan:               t.Catatan,
		TargetTanggal:         target,
		Status:                t.Status,
		SelesaiAt:             nullDateTimeStr(t.SelesaiAt),
		Terlambat:             t.Status == "terbuka" && target != "" && target < today.Format("2006-01-02"),
		KunjunganBerikutnyaID: next,
	}
}

func tindakLanjutUntukGuru(rows []queries.KunjunganTindakLanjut, today time.Time) []models.TindakLanjutResponse {
	out := make([]models.TindakLanjutResponse, 0, len(rows))
	for _, t := range rows {
		if !tindakLanjutInternal(t.Jenis) {
			out = append(out, tindakLanjutResponse(t, today))
		}
	}
	return out
}

func notifyGuruKunjungan(ctx context.Context, q *queries.Querier, guruUserID sql.NullInt64, kunjunganID, actorID int64, title, message string) error {
	if !guruUserID.Valid {
		return nil
	}
	_, err := q.CreateNotification(ctx, queries.CreateNotificationParams{
		UserID:        guruUserID.Int64,
		Type:          "kunjungan_kelas",
		Title:         title,
		Message:       message,
		ActionUrl:     fmt.Sprintf("/app/guru/kunjungan/%d", kunjunganID),
		ReferenceType: "kunjungan_kelas",
		ReferenceID:   sql.NullInt64{Int64: kunjunganID, Valid: true},
		CreatedBy:     nullUserID(actorID),
	})
	return err
}

// kunjunganWALink builds a wa.me link with the result summary for the guru.
// Internal follow-ups are left out.
func kunjunganWALink(noWA string, k models.KunjunganResponse) string {
	nomor := normalisasiNomorWA(noWA)
	if nomor == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Assalamu'alaikum %s,\n\nBerikut hasil kunjungan %s pada %s:\n", k.GuruNama, kelasAtauUmum(k.KelasNama), formatTanggalID(k.Tanggal))
	for _, a := range k.Aspek {
		fmt.Fprintf(&b, "- %s: %d (%s)\n", a.Label, a.Nilai, a.Predikat)
	}
	fmt.Fprintf(&b, "Rata-rata: %s (%s)\n\nCatatan:\n%s\n", formatNilai(k.NilaiRataRata), k.Predikat, k.Catatan)
	first := true
	for _, t := range k.TindakLanjut {
		if t.Internal {
			continue
		}
		if first {
			b.WriteString("\nTindak lanjut:\n")
			first = false
		}
		fmt.Fprintf(&b, "- %s", t.JenisLabel)
		if t.Catatan != "" {
			fmt.Fprintf(&b, ": %s", t.Catatan)
		}
		if t.TargetTanggal != "" {
			fmt.Fprintf(&b, " (target %s)", formatTanggalID(t.TargetTanggal))
		}
		b.WriteString("\n")
	}
	b.WriteString("\nDetail dan kolom tanggapan ada di menu Hasil Kunjungan pada aplikasi Ruang Sanad. Jazakumullahu khairan.")
	return "https://wa.me/" + nomor + "?text=" + url.QueryEscape(b.String())
}

func kelasAtauUmum(kelasNama string) string {
	if kelasNama == "" {
		return "kelas"
	}
	return "kelas " + kelasNama
}

var bulanID = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func formatTanggalID(tanggal string) string {
	t, err := time.Parse("2006-01-02", tanggal)
	if err != nil {
		return tanggal
	}
	return fmt.Sprintf("%d %s %d", t.Day(), bulanID[t.Month()], t.Year())
}

func formatNilai(n float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", n), ".", ",", 1)
}

func nullDateTimeStr(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(wib).Format("2006-01-02 15:04")
}
