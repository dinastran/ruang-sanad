package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const defaultTagihanTemplate = "Assalamu'alaikum {nama},\n\nKami informasikan tagihan SPP Ruang Sanad bulan ke-{bulan_ke} sebesar Rp{nominal} telah terbit (tanggal {tanggal_tagih}) untuk kelas {kelas}.\n\nMohon konfirmasi pembayarannya. Jazaakumullah khairan."

type TagihanService struct {
	querier  *queries.Querier
	template string
}

func NewTagihanService(querier *queries.Querier) *TagihanService {
	return &TagihanService{querier: querier, template: defaultTagihanTemplate}
}

func frekuensiKePertemuan(frekuensi string) (int64, error) {
	switch strings.ToLower(strings.TrimSpace(frekuensi)) {
	case "1x/pekan", "reguler":
		return 4, nil
	case "2x/pekan":
		return 8, nil
	default:
		return 0, fmt.Errorf("frekuensi %q tidak didukung untuk tagihan", frekuensi)
	}
}

type tagihanTrigger struct {
	KelasID       sql.NullInt64
	PertemuanID   sql.NullInt64
	PertemuanKe   int64
	Tanggal       time.Time
	AngkatanKelas string
}

func (s *TagihanService) GenerateForPertemuan(pertemuanID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	if err := s.generateForPertemuanWithQuerier(ctx, q, pertemuanID, true); err != nil {
		return err
	}
	return tx.Commit()
}

// GenerateOnCompletion runs before the meeting is persisted as selesai. The
// caller's transaction therefore contains attendance, billing progress, invoice
// generation, and meeting completion atomically.
func (s *TagihanService) GenerateOnCompletion(pertemuanID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	if err := s.generateForPertemuanWithQuerier(ctx, q, pertemuanID, false); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TagihanService) GenerateOnCompletionWithQuerier(querier *queries.Querier, pertemuanID int64) error {
	return s.generateForPertemuanWithQuerier(context.Background(), querier, pertemuanID, false)
}

func (s *TagihanService) generateForPertemuanWithQuerier(ctx context.Context, q *queries.Querier, pertemuanID int64, requireSelesai bool) error {
	p, err := q.GetPertemuanByID(ctx, pertemuanID)
	if err != nil {
		return err
	}
	if requireSelesai && p.Status != "selesai" {
		return nil
	}

	kelas, err := q.GetKelasByID(ctx, p.KelasID)
	if err != nil {
		return err
	}
	absensi, err := q.GetAbsensiByPertemuan(ctx, p.ID)
	if err != nil {
		return err
	}

	for _, a := range absensi {
		// On the first run after the billing migration, rebuild this santri's
		// progress from every completed historical attendance first. Existing
		// bulan_ke invoices are anchors and will not be duplicated.
		if err := s.backfillSantriBilling(ctx, q, a.SantriID); err != nil {
			return err
		}
		event := queries.SantriBillingMeetingRow{
			PertemuanID:   p.ID,
			KelasID:       p.KelasID,
			PertemuanKe:   p.PertemuanKe,
			Tanggal:       p.Tanggal,
			Frekuensi:     kelas.Frekuensi,
			AngkatanKelas: kelas.Angkatan,
		}
		if err := s.processBillingMeeting(ctx, q, a.SantriID, event); err != nil {
			return err
		}
	}
	return nil
}

func (s *TagihanService) backfillSantriBilling(ctx context.Context, q *queries.Querier, santriID int64) error {
	events, err := q.ListUnprocessedBillingMeetingsForSantri(ctx, santriID)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := s.processBillingMeeting(ctx, q, santriID, event); err != nil {
			return err
		}
	}
	return nil
}

func (s *TagihanService) processBillingMeeting(ctx context.Context, q *queries.Querier, santriID int64, event queries.SantriBillingMeetingRow) error {
	processed, err := q.MarkSantriBillingMeetingProcessed(ctx, santriID, event.PertemuanID, event.Frekuensi)
	if err != nil {
		return err
	}
	if !processed {
		return nil
	}

	threshold, err := frekuensiKePertemuan(event.Frekuensi)
	if err != nil {
		// Unsupported products remain outside the recurring SPP engine, matching
		// the previous behavior. The event is still recorded so Sync cannot
		// repeatedly reconsider the same meeting.
		return nil
	}

	progress, err := q.GetOrCreateSantriBillingProgress(ctx, santriID)
	if err != nil {
		return err
	}
	progress.MeetingCount++

	trigger := tagihanTrigger{
		KelasID:       sql.NullInt64{Int64: event.KelasID, Valid: event.KelasID > 0},
		PertemuanID:   sql.NullInt64{Int64: event.PertemuanID, Valid: event.PertemuanID > 0},
		PertemuanKe:   event.PertemuanKe,
		Tanggal:       event.Tanggal,
		AngkatanKelas: event.AngkatanKelas,
	}
	if err := s.consumeBillingThreshold(ctx, q, santriID, &progress, threshold, trigger); err != nil {
		return err
	}
	return q.UpdateSantriBillingProgress(ctx, santriID, progress.MeetingCount, progress.LastBilledMonth)
}

func (s *TagihanService) consumeBillingThreshold(ctx context.Context, q *queries.Querier, santriID int64, progress *queries.SantriBillingProgressRow, threshold int64, trigger tagihanTrigger) error {
	for threshold > 0 && progress.MeetingCount >= threshold {
		nextMonth := progress.LastBilledMonth + 1
		existing, err := q.CountTagihanSantriBulan(ctx, queries.CountTagihanSantriBulanParams{
			SantriID: santriID,
			BulanKe:  nextMonth,
		})
		if err != nil {
			return err
		}
		if existing == 0 {
			if err := s.createTagihanForMonth(ctx, q, santriID, nextMonth, trigger); err != nil {
				return err
			}
		}
		progress.LastBilledMonth = nextMonth
		progress.MeetingCount -= threshold
	}
	return nil
}

func (s *TagihanService) createTagihanForMonth(ctx context.Context, q *queries.Querier, santriID, bulanKe int64, trigger tagihanTrigger) error {
	santri, err := q.GetSantriByID(ctx, santriID)
	if err != nil {
		return err
	}
	angkatanKelas := trigger.AngkatanKelas
	if angkatanKelas == "" {
		angkatanKelas = santri.AngkatanKelas
	}
	if trigger.Tanggal.IsZero() {
		trigger.Tanggal = time.Now()
	}
	_, err = q.CreateTagihan(ctx, queries.CreateTagihanParams{
		SantriID:      santriID,
		KelasID:       trigger.KelasID,
		PertemuanID:   trigger.PertemuanID,
		BulanKe:       bulanKe,
		PertemuanKe:   trigger.PertemuanKe,
		Nominal:       santri.Nominal,
		TanggalTagih:  trigger.Tanggal,
		JatuhTempo:    sql.NullTime{Time: trigger.Tanggal.AddDate(0, 0, 7), Valid: true},
		AngkatanKelas: angkatanKelas,
	})
	return err
}

// ReconcileFrequencyChangeWithQuerier applies a new frequency to the existing
// per-santri counter without resetting it. Example: 6/8 changed to 1x/pekan
// immediately bills one period at the change date and carries 2/4 forward.
func (s *TagihanService) ReconcileFrequencyChangeWithQuerier(q *queries.Querier, santriID int64, oldFrekuensi, newFrekuensi string, changedAt time.Time) error {
	if strings.EqualFold(strings.TrimSpace(oldFrekuensi), strings.TrimSpace(newFrekuensi)) {
		return nil
	}
	ctx := context.Background()
	if err := s.backfillSantriBilling(ctx, q, santriID); err != nil {
		return err
	}
	threshold, err := frekuensiKePertemuan(newFrekuensi)
	if err != nil {
		return nil
	}
	progress, err := q.GetOrCreateSantriBillingProgress(ctx, santriID)
	if err != nil {
		return err
	}
	if progress.MeetingCount < threshold {
		return nil
	}

	santri, err := q.GetSantriByID(ctx, santriID)
	if err != nil {
		return err
	}
	trigger := tagihanTrigger{
		KelasID:       santri.KelasID,
		Tanggal:       changedAt,
		AngkatanKelas: santri.AngkatanKelas,
	}
	if trigger.Tanggal.IsZero() {
		trigger.Tanggal = time.Now()
	}
	if santri.KelasID.Valid {
		if kelas, err := q.GetKelasByID(ctx, santri.KelasID.Int64); err == nil {
			trigger.AngkatanKelas = kelas.Angkatan
		}
	}
	if latest, err := q.GetLatestProcessedBillingMeetingForSantri(ctx, santriID); err == nil {
		trigger.PertemuanKe = latest.PertemuanKe
	}

	if err := s.consumeBillingThreshold(ctx, q, santriID, &progress, threshold, trigger); err != nil {
		return err
	}
	return q.UpdateSantriBillingProgress(ctx, santriID, progress.MeetingCount, progress.LastBilledMonth)
}

func (s *TagihanService) Sync() error {
	pertemuan, err := s.querier.ListPertemuanSelesai(context.Background())
	if err != nil {
		return err
	}
	for _, p := range pertemuan {
		if err := s.GenerateForPertemuan(p.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *TagihanService) List(filter models.TagihanFilter) ([]models.TagihanResponse, error) {
	var status, kelasID, guruID, bulanKe, dari, sampai interface{}
	if filter.Status != "" && filter.Status != "terlambat" {
		status = filter.Status
	}
	if filter.KelasID > 0 {
		kelasID = filter.KelasID
	}
	if filter.GuruID > 0 {
		guruID = filter.GuruID
	}
	if filter.BulanKe > 0 {
		bulanKe = filter.BulanKe
	}
	if filter.TanggalDari != "" {
		dari = filter.TanggalDari
	}
	if filter.TanggalSampai != "" {
		sampai = filter.TanggalSampai
	}
	ctx := context.Background()
	rows, err := s.querier.ListTagihan(ctx, queries.ListTagihanParams{
		Status: status, KelasID: kelasID, AngkatanKelas: filter.AngkatanKelas, GuruID: guruID,
		Frekuensi: filter.Frekuensi, Level: filter.Level, Gender: filter.Gender,
		BulanKe: bulanKe, TanggalDari: dari, TanggalSampai: sampai, Search: filter.Search,
	})
	if err != nil {
		return nil, err
	}
	tagihanIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		tagihanIDs = append(tagihanIDs, row.ID)
	}
	overrides, err := s.querier.ListTagihanNominalOverrideIDs(ctx, tagihanIDs)
	if err != nil {
		return nil, err
	}
	out := make([]models.TagihanResponse, 0, len(rows))
	for _, row := range rows {
		item := tagihanResponse(row.ID, row.SantriID, row.KelasID, row.BulanKe, row.PertemuanKe, row.Nominal, row.TanggalTagih, row.JatuhTempo, row.Status, row.TanggalBayar, row.Metode, row.Catatan, row.FuTerakhir, row.FuCount, row.SantriNama, row.IDMahasantri, row.NoWa, row.Angkatan, row.AngkatanKelas, row.Frekuensi, row.NamaKelas, row.GuruNama)
		item.NominalOverride = overrides[row.ID]
		if filter.Status == "terlambat" && (item.Status != "belum_bayar" || item.JatuhTempo == "" || !time.Now().After(row.JatuhTempo.Time)) {
			continue
		}
		if item.Status == "belum_bayar" && row.JatuhTempo.Valid && time.Now().After(row.JatuhTempo.Time) {
			item.Status = "terlambat"
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *TagihanService) Get(id int64) (*models.TagihanResponse, error) {
	r, err := s.querier.GetTagihanByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	item := tagihanResponse(r.ID, r.SantriID, r.KelasID, r.BulanKe, r.PertemuanKe, r.Nominal, r.TanggalTagih, r.JatuhTempo, r.Status, r.TanggalBayar, r.Metode, r.Catatan, r.FuTerakhir, r.FuCount, r.SantriNama, r.IDMahasantri, r.NoWa, r.Angkatan, r.AngkatanKelas, r.Frekuensi, r.NamaKelas, r.GuruNama)
	if item.Status == "belum_bayar" && r.JatuhTempo.Valid && time.Now().After(r.JatuhTempo.Time) {
		item.Status = "terlambat"
	}
	override, err := s.querier.HasTagihanNominalOverride(context.Background(), id)
	if err != nil {
		return nil, err
	}
	item.NominalOverride = override
	return &item, nil
}

func (s *TagihanService) MarkLunas(id, userID int64, req models.MarkTagihanLunasRequest) error {
	tagihan, err := s.Get(id)
	if err != nil {
		return err
	}
	if tagihan.Status != "belum_bayar" && tagihan.Status != "terlambat" {
		return fmt.Errorf("tagihan tidak dapat ditandai lunas")
	}
	tanggal := time.Now()
	if req.TanggalBayar != "" {
		parsed, err := time.Parse("2006-01-02", req.TanggalBayar)
		if err != nil {
			return fmt.Errorf("format tanggal bayar tidak valid")
		}
		tanggal = parsed
	}
	return s.querier.MarkTagihanLunas(context.Background(), queries.MarkTagihanLunasParams{ID: id, TanggalBayar: sql.NullTime{Time: tanggal, Valid: true}, Metode: strings.TrimSpace(req.Metode), Catatan: strings.TrimSpace(req.Catatan), DicatatOleh: sql.NullInt64{Int64: userID, Valid: true}})
}

func (s *TagihanService) Batal(id int64, catatan string) error {
	tagihan, err := s.Get(id)
	if err != nil {
		return err
	}
	if tagihan.Status == "lunas" || tagihan.Status == "batal" {
		return fmt.Errorf("tagihan tidak dapat dibatalkan")
	}
	return s.querier.BatalkanTagihan(context.Background(), queries.BatalkanTagihanParams{ID: id, Catatan: strings.TrimSpace(catatan)})
}

func (s *TagihanService) RingkasanPeriode(dari, sampai time.Time) (*models.TagihanRingkasan, error) {
	r, err := s.querier.RingkasanTagihanPeriode(context.Background(), queries.RingkasanTagihanPeriodeParams{TanggalTagih: dari, TanggalTagih_2: sampai})
	if err != nil {
		return nil, err
	}
	out := &models.TagihanRingkasan{TotalTagihan: r.TotalTagihan, NominalTagihan: interfaceInt64(r.NominalTagihan), TotalLunas: interfaceInt64(r.TotalLunas), NominalLunas: interfaceInt64(r.NominalLunas), TotalBelumBayar: interfaceInt64(r.TotalBelumBayar), NominalBelumBayar: interfaceInt64(r.NominalBelumBayar), TotalTerlambat: interfaceInt64(r.TotalTerlambat)}
	if out.NominalTagihan > 0 {
		out.Kolektibilitas = float64(out.NominalLunas) / float64(out.NominalTagihan) * 100
	}
	return out, nil
}

func (s *TagihanService) SetNominalOverride(id, userID, nominal int64) error {
	if nominal < 0 {
		return fmt.Errorf("nominal tidak boleh negatif")
	}
	tagihan, err := s.Get(id)
	if err != nil {
		return err
	}
	if tagihan.Status != "belum_bayar" && tagihan.Status != "terlambat" {
		return fmt.Errorf("nominal hanya dapat diubah untuk tagihan yang belum dibayar")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	rows, err := q.UpdateTagihanNominal(ctx, id, nominal)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("tagihan tidak dapat diubah")
	}
	if err := q.UpsertTagihanNominalOverride(ctx, id, nominal, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TagihanService) ResetNominalOverride(id int64) error {
	tagihan, err := s.Get(id)
	if err != nil {
		return err
	}
	if tagihan.Status != "belum_bayar" && tagihan.Status != "terlambat" {
		return fmt.Errorf("nominal hanya dapat direset untuk tagihan yang belum dibayar")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	if err := q.DeleteTagihanNominalOverride(ctx, id); err != nil {
		return err
	}
	rows, err := q.ResetTagihanNominalToSantri(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("tagihan tidak dapat direset")
	}
	return tx.Commit()
}

func (s *TagihanService) ListTagihanTemplates() ([]models.WaTemplateResponse, error) {
	rows, err := s.querier.ListWaTemplate(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.WaTemplateResponse, 0, len(rows))
	for _, r := range rows {
		if r.TargetType != "tagihan" {
			continue
		}
		out = append(out, models.WaTemplateResponse{
			ID: r.ID, Nama: r.Nama, TargetType: r.TargetType, Body: r.Body, IsAktif: r.IsAktif == 1,
		})
	}
	return out, nil
}

func (s *TagihanService) CreateTagihanTemplate(req models.WaTemplateRequest) (int64, error) {
	if strings.TrimSpace(req.Nama) == "" || strings.TrimSpace(req.Body) == "" {
		return 0, fmt.Errorf("nama dan isi template wajib diisi")
	}
	return s.querier.CreateWaTemplate(context.Background(), queries.CreateWaTemplateParams{
		Nama: strings.TrimSpace(req.Nama), TargetType: "tagihan", Body: strings.TrimSpace(req.Body),
		IsAktif: boolToInt(req.IsAktif),
	})
}

func (s *TagihanService) UpdateTagihanTemplate(id int64, req models.WaTemplateRequest) error {
	current, err := s.querier.GetWaTemplate(context.Background(), id)
	if err != nil {
		return err
	}
	if current.TargetType != "tagihan" {
		return fmt.Errorf("template bukan milik modul tagihan")
	}
	if strings.TrimSpace(req.Nama) == "" || strings.TrimSpace(req.Body) == "" {
		return fmt.Errorf("nama dan isi template wajib diisi")
	}
	return s.querier.UpdateWaTemplate(context.Background(), queries.UpdateWaTemplateParams{
		Nama: strings.TrimSpace(req.Nama), TargetType: "tagihan", Body: strings.TrimSpace(req.Body),
		IsAktif: boolToInt(req.IsAktif), ID: id,
	})
}

func (s *TagihanService) DeleteTagihanTemplate(id int64) error {
	current, err := s.querier.GetWaTemplate(context.Background(), id)
	if err != nil {
		return err
	}
	if current.TargetType != "tagihan" {
		return fmt.Errorf("template bukan milik modul tagihan")
	}
	return s.querier.DeleteWaTemplate(context.Background(), id)
}

func (s *TagihanService) ListFollowUpLogs(tagihanID int64) ([]models.TagihanFollowUpLogResponse, error) {
	rows, err := s.querier.ListTagihanFollowUpLogs(context.Background(), tagihanID)
	if err != nil {
		return nil, err
	}
	out := make([]models.TagihanFollowUpLogResponse, 0, len(rows))
	for _, row := range rows {
		var templateID *int64
		if row.TemplateID.Valid {
			v := row.TemplateID.Int64
			templateID = &v
		}
		out = append(out, models.TagihanFollowUpLogResponse{
			ID: row.ID, TemplateID: templateID, TemplateNama: row.TemplateNama,
			MessageBody: row.MessageBody, PetugasNama: row.PetugasNama,
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

func (s *TagihanService) FollowUpURL(id, userID int64, req models.FollowUpTagihanRequest) (string, error) {
	t, err := s.Get(id)
	if err != nil {
		return "", err
	}
	nomor := normalisasiNomorWA(t.NoWA)
	if nomor == "" {
		return "", fmt.Errorf("nomor WA belum diisi atau tidak valid")
	}

	templateID := req.TemplateID
	templateName := "Custom"
	templateBody := s.template
	if templateID > 0 {
		candidate, err := s.querier.GetWaTemplate(context.Background(), templateID)
		if err != nil {
			return "", fmt.Errorf("template tidak ditemukan")
		}
		if candidate.TargetType != "tagihan" || candidate.IsAktif != 1 {
			return "", fmt.Errorf("template tagihan tidak aktif atau tidak valid")
		}
		templateName = candidate.Nama
		templateBody = candidate.Body
	} else if templates, err := s.ListTagihanTemplates(); err == nil {
		for _, candidate := range templates {
			if candidate.IsAktif {
				templateID = candidate.ID
				templateName = candidate.Nama
				templateBody = candidate.Body
				break
			}
		}
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = renderTagihanTemplate(templateBody, t)
	}
	if message == "" {
		return "", fmt.Errorf("pesan follow-up tidak boleh kosong")
	}
	if len(message) > 5000 {
		return "", fmt.Errorf("pesan follow-up terlalu panjang")
	}

	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	if err := q.TouchFollowUp(ctx, id); err != nil {
		return "", err
	}
	if err := q.CreateTagihanFollowUpLog(
		ctx, id,
		sql.NullInt64{Int64: templateID, Valid: templateID > 0},
		templateName, message, userID,
	); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return "https://wa.me/" + nomor + "?text=" + url.QueryEscape(message), nil
}

func renderTagihanTemplate(template string, t *models.TagihanResponse) string {
	return strings.NewReplacer(
		"{nama}", t.SantriNama,
		"{id_mahasantri}", t.IDMahasantri,
		"{bulan_ke}", strconv.FormatInt(t.BulanKe, 10),
		"{pertemuan_ke}", strconv.FormatInt(t.PertemuanKe, 10),
		"{nominal}", formatRibuan(t.Nominal),
		"{tanggal_tagih}", t.TanggalTagih,
		"{jatuh_tempo}", t.JatuhTempo,
		"{kelas}", t.KelasNama,
		"{guru}", t.GuruNama,
		"{angkatan}", t.Angkatan,
		"{angkatan_kelas}", t.AngkatanKelas,
		"{frekuensi}", t.Frekuensi,
	).Replace(template)
}

func normalisasiNomorWA(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	n := b.String()
	if strings.HasPrefix(n, "0") {
		n = "62" + n[1:]
	}
	if strings.HasPrefix(n, "8") {
		n = "62" + n
	}
	if !strings.HasPrefix(n, "62") || len(n) < 10 {
		return ""
	}
	return n
}

func formatRibuan(v int64) string {
	raw := strconv.FormatInt(v, 10)
	start := 0
	if strings.HasPrefix(raw, "-") {
		start = 1
	}
	for i := len(raw) - 3; i > start; i -= 3 {
		raw = raw[:i] + "." + raw[i:]
	}
	return raw
}
func interfaceInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case []byte:
		i, _ := strconv.ParseInt(string(n), 10, 64)
		return i
	default:
		return 0
	}
}
func nullableDate(v sql.NullTime) string {
	if v.Valid {
		return v.Time.Format("2006-01-02")
	}
	return ""
}
func nullableDateTime(v sql.NullTime) string {
	if v.Valid {
		return v.Time.Format(time.RFC3339)
	}
	return ""
}
func tagihanResponse(id, santriID int64, kelasID sql.NullInt64, bulanKe, pertemuanKe, nominal int64, tanggalTagih time.Time, jatuhTempo sql.NullTime, status string, tanggalBayar sql.NullTime, metode, catatan string, fuTerakhir sql.NullTime, fuCount int64, nama, idMahasantri, noWA, angkatan, angkatanKelas, frekuensi string, namaKelas sql.NullString, guruNama string) models.TagihanResponse {
	var kelas *int64
	if kelasID.Valid {
		value := kelasID.Int64
		kelas = &value
	}
	return models.TagihanResponse{ID: id, SantriID: santriID, SantriNama: nama, IDMahasantri: idMahasantri, NoWA: noWA, KelasID: kelas, KelasNama: namaKelas.String, GuruNama: guruNama, Angkatan: angkatan, AngkatanKelas: angkatanKelas, Frekuensi: frekuensi, BulanKe: bulanKe, PertemuanKe: pertemuanKe, Nominal: nominal, TanggalTagih: tanggalTagih.Format("2006-01-02"), JatuhTempo: nullableDate(jatuhTempo), Status: status, TanggalBayar: nullableDate(tanggalBayar), Metode: metode, Catatan: catatan, FuTerakhir: nullableDateTime(fuTerakhir), FuCount: fuCount}
}
