package services

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	UjrohKomponenTetap      = "tetap"
	UjrohKomponenInsidental = "insidental"

	maxNamaUjrohKomponen = 60
)

var (
	ErrUjrohKomponenNamaKosong  = errors.New("nama komponen wajib diisi")
	ErrUjrohKomponenNamaPanjang = errors.New("nama komponen maksimal 60 karakter")
	ErrUjrohKomponenNamaDipakai = errors.New("nama komponen sudah dipakai")
	ErrUjrohKomponenJenis       = errors.New("jenis komponen harus tetap atau insidental")
	ErrUjrohKomponenTidakAda    = errors.New("komponen ujroh tidak ditemukan")
	ErrUjrohKomponenBukanTetap  = errors.New("penerima tetap hanya untuk komponen jenis tetap")
	ErrUjrohKomponenJumlah      = errors.New("jumlah komponen insidental minimal 1")
	ErrUjrohGuruTidakDitemukan  = errors.New("guru tidak ditemukan")
	ErrKasbonNominal            = errors.New("nominal kasbon harus lebih dari 0")
	ErrKasbonCicilan            = errors.New("cicilan per bulan tidak boleh negatif atau melebihi nominal kasbon")
	ErrKasbonTidakAda           = errors.New("kasbon tidak ditemukan")
	ErrKasbonDibatalkan         = errors.New("kasbon sudah dibatalkan")
	ErrKasbonSudahDipotong      = errors.New("kasbon sudah dipotong di bulan yang dikunci dan tidak dapat dibatalkan")
	ErrKasbonMulaiBulanDikunci  = errors.New("bulan mulai potong sudah dikunci; pilih bulan berikutnya")
	ErrUjrohBulanBelumBerjalan  = errors.New("bulan yang belum berjalan tidak dapat diisi")
)

// lengkapiLive adds the month's components and planned kasbon deductions to
// the per-guru pokok. Guru who earn only components are added. A kasbon
// deduction never exceeds the guru's pokok + components; whatever does not fit
// stays as sisa for the next month.
func (s *UjrohService) lengkapiLive(ctx context.Context, q *queries.Querier, bulan string, pokok []models.UjrohGuru) ([]models.UjrohGuru, error) {
	out := make([]models.UjrohGuru, len(pokok))
	copy(out, pokok)
	index := make(map[int64]int, len(out))
	for i := range out {
		out[i].Komponen = []models.UjrohKomponenLine{}
		out[i].Kasbon = []models.UjrohKasbonLine{}
		index[out[i].GuruID] = i
	}

	komponen, err := q.ListUjrohKomponenLive(ctx, bulan)
	if err != nil {
		return nil, err
	}
	for _, r := range komponen {
		i, ok := index[r.GuruID]
		if !ok {
			out = append(out, models.UjrohGuru{
				GuruID: r.GuruID, GuruNama: r.GuruNama, GuruStatus: r.GuruStatus,
				Pertemuan: []models.UjrohPertemuan{}, Komponen: []models.UjrohKomponenLine{}, Kasbon: []models.UjrohKasbonLine{},
			})
			i = len(out) - 1
			index[r.GuruID] = i
		}
		line := models.UjrohKomponenLine{
			KomponenID: r.KomponenID, KomponenNama: r.KomponenNama, Jenis: r.Jenis,
			Jumlah: r.Jumlah, Tarif: r.Tarif, Total: r.Jumlah * r.Tarif, Catatan: r.Catatan,
			DiubahBulan: r.DiubahBulan == 1, NominalTetap: r.NominalTetap,
		}
		out[i].Komponen = append(out[i].Komponen, line)
		out[i].TotalKomponen += line.Total
	}

	kasbon, err := q.ListKasbonUntukBulan(ctx, bulan)
	if err != nil {
		return nil, err
	}
	sisaBruto := make(map[int64]int64, len(out))
	for _, g := range out {
		sisaBruto[g.GuruID] = g.Total + g.TotalKomponen
	}
	for _, k := range kasbon {
		i, ok := index[k.GuruID]
		if !ok {
			continue
		}
		sisa := k.Nominal - k.Terpotong
		if sisa <= 0 {
			continue
		}
		rencana := sisa
		if k.Cicilan > 0 && k.Cicilan < sisa {
			rencana = k.Cicilan
		}
		potong := min(rencana, sisaBruto[k.GuruID])
		sisaBruto[k.GuruID] -= potong
		out[i].Kasbon = append(out[i].Kasbon, models.UjrohKasbonLine{
			KasbonID: k.ID, Tanggal: k.Tanggal, Keterangan: k.Keterangan, TotalKasbon: k.Nominal,
			SisaSebelum: sisa, Potongan: potong, SisaSetelah: sisa - potong,
		})
		out[i].TotalKasbon += potong
	}

	for i := range out {
		out[i].Diterima = out[i].Total + out[i].TotalKomponen - out[i].TotalKasbon
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].GuruNama) < strings.ToLower(out[j].GuruNama) })
	return out, nil
}

// ensureBulanTerbuka refuses changes to a month that is locked or has not started.
func (s *UjrohService) ensureBulanTerbuka(ctx context.Context, bulan string) error {
	if bulan > s.now().Format("2006-01") {
		return ErrUjrohBulanBelumBerjalan
	}
	status, err := s.querier.GetUjrohBulan(ctx, bulan)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && status.DikunciAt.Valid {
		return ErrUjrohSudahDikunci
	}
	return nil
}

func (s *UjrohService) ensureGuru(ctx context.Context, guruID int64) error {
	if _, err := s.querier.GetGuruByID(ctx, guruID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUjrohGuruTidakDitemukan
		}
		return err
	}
	return nil
}

func (s *UjrohService) ListKomponen() ([]models.UjrohKomponen, error) {
	ctx := context.Background()
	rows, err := s.querier.ListUjrohKomponen(ctx)
	if err != nil {
		return nil, err
	}
	penerima, err := s.querier.ListUjrohKomponenGuru(ctx)
	if err != nil {
		return nil, err
	}
	byKomponen := map[int64]map[int64]int64{}
	for _, p := range penerima {
		if byKomponen[p.KomponenID] == nil {
			byKomponen[p.KomponenID] = map[int64]int64{}
		}
		byKomponen[p.KomponenID][p.GuruID] = p.Nominal
	}
	out := make([]models.UjrohKomponen, 0, len(rows))
	for _, r := range rows {
		m := byKomponen[r.ID]
		if m == nil {
			m = map[int64]int64{}
		}
		out = append(out, models.UjrohKomponen{
			ID: r.ID, Nama: r.Nama, Jenis: r.Jenis, NominalDefault: r.NominalDefault, IsAktif: r.IsAktif == 1, Penerima: m,
		})
	}
	return out, nil
}

func validasiNamaKomponen(nama string) (string, error) {
	nama = strings.TrimSpace(nama)
	if nama == "" {
		return "", ErrUjrohKomponenNamaKosong
	}
	if len([]rune(nama)) > maxNamaUjrohKomponen {
		return "", ErrUjrohKomponenNamaPanjang
	}
	return nama, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// CreateKomponen adds a component to the master list.
func (s *UjrohService) CreateKomponen(req models.SimpanUjrohKomponenRequest, userID int64) error {
	nama, err := validasiNamaKomponen(req.Nama)
	if err != nil {
		return err
	}
	if req.Jenis != UjrohKomponenTetap && req.Jenis != UjrohKomponenInsidental {
		return ErrUjrohKomponenJenis
	}
	if req.NominalDefault < 0 {
		return ErrUjrohNominalTidakSah
	}
	_, err = s.querier.InsertUjrohKomponen(context.Background(), nama, req.Jenis, req.NominalDefault, userID)
	if isUniqueViolation(err) {
		return ErrUjrohKomponenNamaDipakai
	}
	return err
}

// UpdateKomponen renames, re-prices or (de)activates a component. The jenis is
// fixed once created. Locked months keep their own snapshot.
func (s *UjrohService) UpdateKomponen(id int64, req models.SimpanUjrohKomponenRequest, userID int64) error {
	ctx := context.Background()
	cur, err := s.querier.GetUjrohKomponen(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUjrohKomponenTidakAda
	}
	if err != nil {
		return err
	}
	nama, err := validasiNamaKomponen(req.Nama)
	if err != nil {
		return err
	}
	if req.NominalDefault < 0 {
		return ErrUjrohNominalTidakSah
	}
	aktif := cur.IsAktif
	if req.IsAktif != nil {
		aktif = boolToInt64(*req.IsAktif)
	}
	_, err = s.querier.UpdateUjrohKomponen(ctx, id, nama, req.NominalDefault, aktif, userID)
	if isUniqueViolation(err) {
		return ErrUjrohKomponenNamaDipakai
	}
	return err
}

// SetKomponenGuru sets (or with nil removes) a guru's recurring amount for a
// tetap component.
func (s *UjrohService) SetKomponenGuru(komponenID, guruID int64, nominal *int64, userID int64) error {
	ctx := context.Background()
	k, err := s.querier.GetUjrohKomponen(ctx, komponenID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUjrohKomponenTidakAda
	}
	if err != nil {
		return err
	}
	if k.Jenis != UjrohKomponenTetap {
		return ErrUjrohKomponenBukanTetap
	}
	if nominal == nil {
		return s.querier.DeleteUjrohKomponenGuru(ctx, komponenID, guruID)
	}
	if *nominal < 0 {
		return ErrUjrohNominalTidakSah
	}
	if err := s.ensureGuru(ctx, guruID); err != nil {
		return err
	}
	return s.querier.UpsertUjrohKomponenGuru(ctx, komponenID, guruID, *nominal, userID)
}

// SetKomponenBulan records a component for one guru in one unlocked month: an
// insidental entry, or a tetap amount adjusted for this month (jumlah 0 means
// not given this month).
func (s *UjrohService) SetKomponenBulan(bulan string, guruID, komponenID int64, req models.SetUjrohKomponenBulanRequest, userID int64) error {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := s.ensureBulanTerbuka(ctx, bulan); err != nil {
		return err
	}
	k, err := s.querier.GetUjrohKomponen(ctx, komponenID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUjrohKomponenTidakAda
	}
	if err != nil {
		return err
	}
	if req.Jumlah < 0 || req.Tarif < 0 {
		return ErrUjrohNominalTidakSah
	}
	if k.Jenis == UjrohKomponenInsidental && req.Jumlah < 1 {
		return ErrUjrohKomponenJumlah
	}
	if err := s.ensureGuru(ctx, guruID); err != nil {
		return err
	}
	return s.querier.UpsertUjrohKomponenBulan(ctx, bulan, guruID, komponenID, req.Jumlah, req.Tarif, strings.TrimSpace(req.Catatan), userID)
}

// HapusKomponenBulan removes a month's entry: an insidental line disappears and
// a tetap component returns to its recurring amount.
func (s *UjrohService) HapusKomponenBulan(bulan string, guruID, komponenID int64) error {
	bulan, err := s.NormalizeBulan(bulan)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := s.ensureBulanTerbuka(ctx, bulan); err != nil {
		return err
	}
	return s.querier.DeleteUjrohKomponenBulan(ctx, bulan, guruID, komponenID)
}

func kasbonResponse(r queries.KasbonRow) models.Kasbon {
	k := models.Kasbon{
		ID: r.ID, GuruID: r.GuruID, GuruNama: r.GuruNama, Tanggal: r.Tanggal, Nominal: r.Nominal, Cicilan: r.Cicilan,
		MulaiBulan: r.MulaiBulan, Keterangan: r.Keterangan, DibuatOleh: r.DibuatOlehNama,
		Terpotong: r.Terpotong, Sisa: r.Nominal - r.Terpotong, Status: "berjalan",
	}
	switch {
	case r.DibatalkanAt.Valid:
		k.Status = "dibatalkan"
	case k.Sisa <= 0:
		k.Status = "lunas"
	}
	return k
}

func (s *UjrohService) ListKasbon() ([]models.Kasbon, error) {
	rows, err := s.querier.ListKasbon(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.Kasbon, 0, len(rows))
	for _, r := range rows {
		out = append(out, kasbonResponse(r))
	}
	return out, nil
}

// CreateKasbon records a cash advance. It is deducted from ujroh starting at
// mulai_bulan: cicilan per month, or all at once when cicilan is 0.
func (s *UjrohService) CreateKasbon(req models.CreateKasbonRequest, userID int64) error {
	ctx := context.Background()
	if err := s.ensureGuru(ctx, req.GuruID); err != nil {
		return err
	}
	tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(req.Tanggal))
	if err != nil {
		return errors.New("format tanggal kasbon harus YYYY-MM-DD")
	}
	if req.Nominal <= 0 {
		return ErrKasbonNominal
	}
	if req.Cicilan < 0 || req.Cicilan > req.Nominal {
		return ErrKasbonCicilan
	}
	mulai := strings.TrimSpace(req.MulaiBulan)
	if mulai == "" {
		mulai = tanggal.Format("2006-01")
	}
	if mulai, err = s.NormalizeBulan(mulai); err != nil {
		return err
	}
	status, err := s.querier.GetUjrohBulan(ctx, mulai)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && status.DikunciAt.Valid {
		return ErrKasbonMulaiBulanDikunci
	}
	_, err = s.querier.InsertKasbon(ctx, req.GuruID, tanggal.Format("2006-01-02"), req.Nominal, req.Cicilan, mulai, strings.TrimSpace(req.Keterangan), userID)
	return err
}

func (s *UjrohService) getKasbonAktif(ctx context.Context, id int64) (queries.KasbonRow, error) {
	k, err := s.querier.GetKasbon(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return k, ErrKasbonTidakAda
	}
	if err != nil {
		return k, err
	}
	if k.DibatalkanAt.Valid {
		return k, ErrKasbonDibatalkan
	}
	return k, nil
}

// UpdateKasbon changes the monthly installment and note. Locked months keep
// the deduction they froze.
func (s *UjrohService) UpdateKasbon(id int64, req models.UpdateKasbonRequest) error {
	ctx := context.Background()
	k, err := s.getKasbonAktif(ctx, id)
	if err != nil {
		return err
	}
	if req.Cicilan < 0 || req.Cicilan > k.Nominal {
		return ErrKasbonCicilan
	}
	return s.querier.UpdateKasbonCicilan(ctx, id, req.Cicilan, strings.TrimSpace(req.Keterangan))
}

// BatalkanKasbon voids a kasbon entered by mistake. Refused once a locked month
// deducted it, so a paid slip never loses its deduction.
func (s *UjrohService) BatalkanKasbon(id, userID int64) error {
	ctx := context.Background()
	k, err := s.getKasbonAktif(ctx, id)
	if err != nil {
		return err
	}
	if k.Terpotong > 0 {
		return ErrKasbonSudahDipotong
	}
	return s.querier.BatalkanKasbon(ctx, id, userID)
}
