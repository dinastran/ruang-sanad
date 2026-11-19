package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// CRM programs. Every lead belongs to one program: chosen by hand on the lead,
// else the program of its lead source, else the built-in Tahsin Reguler.
// A regular lead closes into a santri; a short-program lead (jenis "program",
// e.g. Sepekan Bersama Al-Fatihah) closes into a participant, recorded as a
// Produk CRM order linked to the lead, without class, SPP, or onboarding.
// Short programs have sessions with a public attendance link.

const (
	LeadProgramJenisReguler = "reguler"
	LeadProgramJenisProgram = "program"

	// Attendance link opens before a session starts and closes some hours later.
	programHadirBuka  = 30 * time.Minute
	programHadirTutup = 3 * time.Hour
)

var (
	ErrLeadBukanProgramPendek = errors.New("lead ini masuk program Tahsin Reguler; gunakan Jadikan santri, atau ubah programnya dulu")
	ErrLeadProgramPendek      = errors.New("lead ini masuk program pendek; daftarkan sebagai peserta program, atau ubah programnya ke Tahsin Reguler dulu")
	ErrProgramSesiTidakAda    = errors.New("Link absensi tidak dikenal")
)

type LeadProgramService struct {
	querier *queries.Querier
	now     func() time.Time
}

func NewLeadProgramService(q *queries.Querier) *LeadProgramService {
	return &LeadProgramService{querier: q, now: time.Now}
}

func (s *LeadProgramService) normalize(ctx context.Context, id int64, r *models.LeadProgramRequest) error {
	r.Nama = strings.Join(strings.Fields(r.Nama), " ")
	r.Keterangan = strings.TrimSpace(r.Keterangan)
	if r.Nama == "" || utf8.RuneCountInString(r.Nama) > 80 {
		return errors.New("nama program wajib diisi, maksimal 80 karakter")
	}
	if utf8.RuneCountInString(r.Keterangan) > 300 {
		return errors.New("keterangan maksimal 300 karakter")
	}
	if id == queries.LeadProgramRegulerID {
		// The built-in program always closes into a santri and stays active.
		// It may point at a produk (kategori program) only as the price
		// source for public registration links; it never has a batch.
		r.Jenis, r.ProdukBatchID, r.IsAktif = LeadProgramJenisReguler, 0, true
		if r.ProdukID <= 0 {
			r.ProdukID = 0
			return nil
		}
		produk, err := s.querier.GetProduk(ctx, r.ProdukID)
		if err != nil {
			return errors.New("produk tidak ditemukan")
		}
		if produk.Kategori != "program" {
			return errors.New("produk harus berkategori program, bukan buku")
		}
		return nil
	}
	if r.Jenis == "" {
		r.Jenis = LeadProgramJenisProgram
	}
	if r.Jenis != LeadProgramJenisProgram {
		return errors.New("program tambahan harus berjenis program pendek; Tahsin Reguler sudah ada")
	}
	if r.ProdukID <= 0 {
		return errors.New("pilih produk program di Produk CRM (buat dulu produk berkategori program bila belum ada)")
	}
	produk, err := s.querier.GetProduk(ctx, r.ProdukID)
	if err != nil {
		return errors.New("produk program tidak ditemukan")
	}
	if produk.Kategori != "program" {
		return errors.New("produk harus berkategori program, bukan buku")
	}
	if r.ProdukBatchID > 0 {
		batch, err := s.querier.GetProdukBatch(ctx, r.ProdukBatchID)
		if err != nil || batch.ProdukID != r.ProdukID {
			return errors.New("batch tidak sesuai dengan produk program")
		}
	}
	return nil
}

func dupProgramErr(err error, nama string) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("program %q sudah ada", nama)
	}
	return err
}

func (s *LeadProgramService) Create(req models.LeadProgramRequest) (int64, error) {
	ctx := context.Background()
	if err := s.normalize(ctx, 0, &req); err != nil {
		return 0, err
	}
	id, err := s.querier.CreateLeadProgram(ctx, req, s.now())
	return id, dupProgramErr(err, req.Nama)
}

func (s *LeadProgramService) Update(id int64, req models.LeadProgramRequest) error {
	ctx := context.Background()
	if _, err := s.querier.GetLeadProgram(ctx, id); err != nil {
		return err
	}
	if err := s.normalize(ctx, id, &req); err != nil {
		return err
	}
	return dupProgramErr(s.querier.UpdateLeadProgram(ctx, id, req, s.now()), req.Nama)
}

// List returns every program with its sources and lead/participant counts.
func (s *LeadProgramService) List() ([]models.LeadProgram, error) {
	ctx := context.Background()
	programs, err := s.querier.ListLeadPrograms(ctx)
	if err != nil {
		return nil, err
	}
	sources, err := s.querier.ListLeadSourcePrograms(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.querier.LeadProgramCounts(ctx)
	if err != nil {
		return nil, err
	}
	for i := range programs {
		p := &programs[i]
		p.SumberIDs = []int64{}
		for sid, pid := range sources {
			if pid == p.ID {
				p.SumberIDs = append(p.SumberIDs, sid)
			}
		}
		p.LeadOpen, p.LeadWon = counts[p.ID]["open"], counts[p.ID]["won"]
		if p.Jenis == LeadProgramJenisProgram {
			sesi, err := s.querier.ListProgramSesi(ctx, p.ID)
			if err != nil {
				return nil, err
			}
			p.JumlahSesi = int64(len(sesi))
			if p.ProdukID > 0 {
				peserta, err := s.querier.ListProgramPeserta(ctx, p.ProdukID, p.ProdukBatchID)
				if err != nil {
					return nil, err
				}
				p.JumlahPeserta = int64(len(peserta))
			}
		}
	}
	return programs, nil
}

func (s *LeadProgramService) Options() ([]models.LeadProgramOption, error) {
	programs, err := s.querier.ListLeadPrograms(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.LeadProgramOption, 0, len(programs))
	for _, p := range programs {
		out = append(out, models.LeadProgramOption{ID: p.ID, Nama: p.Nama, Jenis: p.Jenis, IsAktif: p.IsAktif})
	}
	return out, nil
}

// SetSources makes exactly these lead sources belong to the program; sources
// previously mapped to it and not listed go back to Tahsin Reguler.
func (s *LeadProgramService) SetSources(programID int64, sourceIDs []int64) error {
	ctx := context.Background()
	if _, err := s.querier.GetLeadProgram(ctx, programID); err != nil {
		return err
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)
	if err := q.ClearLeadSourcesProgram(ctx, programID); err != nil {
		return err
	}
	if programID != queries.LeadProgramRegulerID {
		for _, sid := range sourceIDs {
			n, err := q.SetLeadSourceProgram(ctx, sid, programID)
			if err != nil {
				return err
			}
			if n == 0 {
				return fmt.Errorf("sumber lead %d tidak ditemukan", sid)
			}
		}
	}
	return tx.Commit()
}

// Detail is the program page: sessions, participants and their attendance.
type ProgramDetail struct {
	Program models.LeadProgram      `json:"program"`
	Sesi    []models.ProgramSesi    `json:"sesi"`
	Peserta []models.ProgramPeserta `json:"peserta"`
}

func (s *LeadProgramService) Detail(id int64) (*ProgramDetail, error) {
	ctx := context.Background()
	p, err := s.querier.GetLeadProgram(ctx, id)
	if err != nil {
		return nil, err
	}
	sources, err := s.querier.ListLeadSourcePrograms(ctx)
	if err != nil {
		return nil, err
	}
	p.SumberIDs = []int64{}
	for sid, pid := range sources {
		if pid == p.ID {
			p.SumberIDs = append(p.SumberIDs, sid)
		}
	}
	d := &ProgramDetail{Program: *p, Sesi: []models.ProgramSesi{}, Peserta: []models.ProgramPeserta{}}
	if d.Sesi, err = s.querier.ListProgramSesi(ctx, id); err != nil {
		return nil, err
	}
	if p.ProdukID > 0 {
		if d.Peserta, err = s.querier.ListProgramPeserta(ctx, p.ProdukID, p.ProdukBatchID); err != nil {
			return nil, err
		}
	}
	hadir, err := s.querier.ListProgramKehadiran(ctx, id)
	if err != nil {
		return nil, err
	}
	byOrder := map[int64][]int64{}
	for _, k := range hadir {
		byOrder[k.OrderID] = append(byOrder[k.OrderID], k.SesiID)
	}
	for i := range d.Peserta {
		if h := byOrder[d.Peserta[i].OrderID]; h != nil {
			d.Peserta[i].Hadir = h
		}
	}
	return d, nil
}

func normalizeSesi(r *models.ProgramSesiRequest) error {
	r.Judul = strings.TrimSpace(r.Judul)
	r.Tanggal = strings.TrimSpace(r.Tanggal)
	r.JamMulai = strings.TrimSpace(r.JamMulai)
	if r.Ke < 1 || r.Ke > 100 {
		return errors.New("nomor sesi 1–100")
	}
	if _, err := time.Parse("2006-01-02", r.Tanggal); err != nil {
		return errors.New("tanggal sesi wajib berformat YYYY-MM-DD")
	}
	if r.JamMulai != "" {
		if _, err := time.Parse("15:04", r.JamMulai); err != nil {
			return errors.New("jam mulai berformat JJ:MM, mis. 19:30")
		}
	}
	if utf8.RuneCountInString(r.Judul) > 120 {
		return errors.New("judul sesi maksimal 120 karakter")
	}
	return nil
}

func dupSesiErr(err error, ke int64) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("sesi ke-%d sudah ada", ke)
	}
	return err
}

func (s *LeadProgramService) CreateSesi(programID int64, r models.ProgramSesiRequest) (int64, error) {
	ctx := context.Background()
	p, err := s.querier.GetLeadProgram(ctx, programID)
	if err != nil {
		return 0, err
	}
	if p.Jenis != LeadProgramJenisProgram {
		return 0, errors.New("sesi & absensi hanya untuk program pendek")
	}
	if err := normalizeSesi(&r); err != nil {
		return 0, err
	}
	raw := make([]byte, 9)
	if _, err := rand.Read(raw); err != nil {
		return 0, err
	}
	id, err := s.querier.CreateProgramSesi(ctx, programID, r, base64.RawURLEncoding.EncodeToString(raw))
	return id, dupSesiErr(err, r.Ke)
}

func (s *LeadProgramService) UpdateSesi(id int64, r models.ProgramSesiRequest) error {
	ctx := context.Background()
	if _, err := s.querier.GetProgramSesi(ctx, id); err != nil {
		return errors.New("sesi tidak ditemukan")
	}
	if err := normalizeSesi(&r); err != nil {
		return err
	}
	return dupSesiErr(s.querier.UpdateProgramSesi(ctx, id, r), r.Ke)
}

// DeleteSesi removes a session that has no attendance yet.
func (s *LeadProgramService) DeleteSesi(id int64) (int64, error) {
	ctx := context.Background()
	sesi, err := s.querier.GetProgramSesi(ctx, id)
	if err != nil {
		return 0, errors.New("sesi tidak ditemukan")
	}
	if sesi.Hadir > 0 {
		return 0, errors.New("sesi sudah punya data kehadiran, jadi tidak bisa dihapus")
	}
	return sesi.ProgramID, s.querier.DeleteProgramSesi(ctx, id)
}

// SetHadir records or clears attendance by hand (admin correction).
func (s *LeadProgramService) SetHadir(sesiID, orderID int64, hadir bool) (int64, error) {
	ctx := context.Background()
	sesi, err := s.querier.GetProgramSesi(ctx, sesiID)
	if err != nil {
		return 0, errors.New("sesi tidak ditemukan")
	}
	if err := s.ensurePeserta(ctx, sesi.ProgramID, orderID); err != nil {
		return 0, err
	}
	if !hadir {
		return sesi.ProgramID, s.querier.DeleteProgramKehadiran(ctx, sesiID, orderID)
	}
	_, err = s.querier.InsertProgramKehadiran(ctx, sesiID, orderID, "manual", s.now())
	return sesi.ProgramID, err
}

func (s *LeadProgramService) ensurePeserta(ctx context.Context, programID, orderID int64) error {
	p, err := s.querier.GetLeadProgram(ctx, programID)
	if err != nil {
		return err
	}
	peserta, err := s.querier.ListProgramPeserta(ctx, p.ProdukID, p.ProdukBatchID)
	if err != nil {
		return err
	}
	for _, x := range peserta {
		if x.OrderID == orderID {
			return nil
		}
	}
	return errors.New("peserta tidak ditemukan di program ini")
}

// --- Public attendance link ----------------------------------------------------

// HadirInfo is what the public attendance page shows (no participant data).
type HadirInfo struct {
	Program string `json:"program"`
	Ke      int64  `json:"ke"`
	Judul   string `json:"judul"`
	Tanggal string `json:"tanggal"`
	Jam     string `json:"jam"`
	// Status: "buka", "belum" (not open yet), "tutup" (closed).
	Status    string `json:"status"`
	BukaPukul string `json:"buka_pukul"`
}

// sesiWindow returns when the attendance link opens and closes (WIB). A
// session without a start time is open the whole day.
func sesiWindow(sesi *models.ProgramSesi) (time.Time, time.Time) {
	day, _ := time.ParseInLocation("2006-01-02", sesi.Tanggal, ZonaWaktuRiayah)
	if sesi.JamMulai == "" {
		return day, day.AddDate(0, 0, 1)
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", sesi.Tanggal+" "+sesi.JamMulai, ZonaWaktuRiayah)
	if err != nil {
		return day, day.AddDate(0, 0, 1)
	}
	return start.Add(-programHadirBuka), start.Add(programHadirTutup)
}

func (s *LeadProgramService) HadirInfo(token string) (*HadirInfo, *models.ProgramSesi, error) {
	ctx := context.Background()
	sesi, err := s.querier.GetProgramSesiByToken(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, nil, ErrProgramSesiTidakAda
	}
	p, err := s.querier.GetLeadProgram(ctx, sesi.ProgramID)
	if err != nil || !p.IsAktif {
		return nil, nil, ErrProgramSesiTidakAda
	}
	buka, tutup := sesiWindow(sesi)
	now := s.now()
	status := "buka"
	switch {
	case now.Before(buka):
		status = "belum"
	case !now.Before(tutup):
		status = "tutup"
	}
	return &HadirInfo{
		Program: p.Nama, Ke: sesi.Ke, Judul: sesi.Judul, Tanggal: sesi.Tanggal, Jam: sesi.JamMulai,
		Status: status, BukaPukul: buka.In(ZonaWaktuRiayah).Format("15:04"),
	}, sesi, nil
}

// HadirHasil is shown after a participant checks in. Nama is masked so the
// page never reveals who is registered.
type HadirHasil struct {
	Nama          string `json:"nama"`
	SudahTercatat bool   `json:"sudah_tercatat"`
}

// Hadir records attendance from the public link: the WhatsApp number must
// belong to a participant of the session's program.
func (s *LeadProgramService) Hadir(token, noWA string) (*HadirHasil, error) {
	info, sesi, err := s.HadirInfo(token)
	if err != nil {
		return nil, err
	}
	switch info.Status {
	case "belum":
		return nil, fmt.Errorf("Absensi sesi ini baru dibuka pukul %s WIB", info.BukaPukul)
	case "tutup":
		return nil, errors.New("Absensi sesi ini sudah ditutup; hubungi admin bila Anda hadir")
	}
	want := NormalizeLeadPhone(noWA)
	if len(want) < 9 {
		return nil, errors.New("Masukkan nomor WhatsApp yang dipakai saat mendaftar, mis. 0812xxxxxxx")
	}
	ctx := context.Background()
	p, err := s.querier.GetLeadProgram(ctx, sesi.ProgramID)
	if err != nil {
		return nil, err
	}
	peserta, err := s.querier.ListProgramPeserta(ctx, p.ProdukID, p.ProdukBatchID)
	if err != nil {
		return nil, err
	}
	for _, x := range peserta {
		if NormalizeLeadPhone(x.NoWA) != want {
			continue
		}
		baru, err := s.querier.InsertProgramKehadiran(ctx, sesi.ID, x.OrderID, "link", s.now())
		if err != nil {
			return nil, err
		}
		return &HadirHasil{Nama: samarkanNama(x.Nama), SudahTercatat: !baru}, nil
	}
	return nil, errors.New("Nomor ini belum terdaftar sebagai peserta; pastikan nomornya sama dengan saat mendaftar, atau hubungi admin")
}

// samarkanNama keeps the first letter and last letter of each word:
// "Ahmad Budi" → "A***d B**i".
func samarkanNama(nama string) string {
	words := strings.Fields(nama)
	for i, w := range words {
		r := []rune(w)
		if len(r) <= 2 {
			words[i] = string(r[0]) + "*"
			continue
		}
		words[i] = string(r[0]) + strings.Repeat("*", len(r)-2) + string(r[len(r)-1])
	}
	return strings.Join(words, " ")
}

// --- Lead side: program, participant closing, offer regular class -----------------

// LeadProgramInfo is the program shown on the lead page.
type LeadProgramInfo struct {
	Program models.LeadProgram       `json:"program"`
	Manual  bool                     `json:"manual"`
	Order   *models.LeadProgramOrder `json:"order"`
}

func (s *LeadService) ProgramInfo(ctx context.Context, leadID int64) (*LeadProgramInfo, error) {
	pid, err := s.querier.LeadEffectiveProgramID(ctx, leadID)
	if err != nil {
		return nil, err
	}
	p, err := s.querier.GetLeadProgram(ctx, pid)
	if err != nil {
		return nil, err
	}
	l, err := loadLead(ctx, s.querier, leadID)
	if err != nil {
		return nil, err
	}
	order, err := s.querier.GetLeadProgramOrder(ctx, leadID)
	if err != nil {
		return nil, err
	}
	return &LeadProgramInfo{Program: *p, Manual: l.ProgramID.Valid, Order: order}, nil
}

func (s *LeadService) effectiveProgram(ctx context.Context, q *queries.Querier, leadID int64) (*models.LeadProgram, error) {
	pid, err := q.LeadEffectiveProgramID(ctx, leadID)
	if err != nil {
		return nil, err
	}
	return q.GetLeadProgram(ctx, pid)
}

// SetProgram chooses the lead's program by hand (0 = follow its source).
func (s *LeadService) SetProgram(ctx context.Context, leadID, programID, actorID int64) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, leadID)
		if err != nil {
			return err
		}
		if l.Status == LeadStatusWon {
			return errors.New("lead sudah closing; programnya tidak bisa diubah")
		}
		before, err := s.effectiveProgram(ctx, q, leadID)
		if err != nil {
			return err
		}
		if programID > 0 {
			p, err := q.GetLeadProgram(ctx, programID)
			if err != nil {
				return err
			}
			if !p.IsAktif {
				return errors.New("program ini sudah tidak aktif")
			}
		}
		now := s.nowUTC()
		if err := q.SetLeadProgram(ctx, leadID, programID, now); err != nil {
			return err
		}
		after, err := s.effectiveProgram(ctx, q, leadID)
		if err != nil {
			return err
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("usr"), "lead.program_changed", userActor(actorID), map[string]any{
			"from": before.Nama, "to": after.Nama, "manual": programID > 0,
		}, now)
	})
}

// DaftarkanPeserta closes a verified short-program lead into a participant:
// a Produk CRM order for the program's product, linked to the lead, created
// in the same transaction as the won status. No santri is created.
func (s *LeadService) DaftarkanPeserta(ctx context.Context, leadID, actorID int64, req models.LeadPesertaRequest) (int64, error) {
	var orderID int64
	err := s.withTx(ctx, func(q *queries.Querier) error {
		l, err := loadLead(ctx, q, leadID)
		if err != nil {
			return err
		}
		if l.SantriID.Valid || l.Status == LeadStatusWon {
			return errors.New("lead ini sudah closing")
		}
		if l.Status != LeadStatusOpen || l.Stage != LeadStageTerverifikasi {
			return errors.New("lead harus berstatus Terverifikasi sebelum didaftarkan sebagai peserta")
		}
		p, err := s.effectiveProgram(ctx, q, leadID)
		if err != nil {
			return err
		}
		if p.Jenis != LeadProgramJenisProgram {
			return ErrLeadBukanProgramPendek
		}
		if p.ProdukID <= 0 {
			return fmt.Errorf("program %s belum punya produk di Produk CRM; atur di halaman Program", p.Nama)
		}
		nama := strings.TrimSpace(req.Nama)
		if nama == "" {
			nama = strings.TrimSpace(l.Nama)
		}
		if nama == "" {
			return errors.New("nama peserta wajib diisi")
		}
		noWA := strings.TrimSpace(req.NoWA)
		if noWA == "" {
			noWA = l.Phone
		}
		nominal := req.Nominal
		if nominal == 0 {
			nominal = l.PaymentAmount
		}
		if nominal < 0 {
			return errors.New("nominal tidak valid")
		}
		tanggal := strings.TrimSpace(req.Tanggal)
		if tanggal == "" {
			tanggal = HariIniRiayah().Format("2006-01-02")
		}
		orderID, err = CreateProductOrderWithQuerier(ctx, q, models.CreateProductOrderRequest{
			Tanggal: tanggal, NamaPembeli: nama, NoWA: noWA, Sumber: "whatsapp",
			Catatan: fmt.Sprintf("Peserta %s dari CRM %s", p.Nama, aiLeadRef(leadID)),
			Items:   []models.CreateProductOrderItemRequest{{ProdukID: p.ProdukID, ProdukBatchID: p.ProdukBatchID, Qty: 1, HargaSatuan: &nominal}},
		}, actorID)
		if err != nil {
			return err
		}
		if err := q.SetProductOrderLead(ctx, orderID, leadID); err != nil {
			return err
		}
		now := s.nowUTC()
		n, err := q.MarkLeadWonProgram(ctx, leadID, actorID, now)
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("lead ini sudah closing")
		}
		if _, err := cancelActiveFollowups(ctx, q, leadID, "lead menjadi peserta program", now); err != nil {
			return err
		}
		if err := writeLeadEvent(ctx, q, leadID, newLeadEventID("usr"), "lead.converted", userActor(actorID), map[string]any{
			"program": p.Nama, "program_id": p.ID, "order_id": orderID, "status_from": LeadStatusOpen, "status_to": LeadStatusWon,
		}, now); err != nil {
			return err
		}
		return s.enqueuePurchase(ctx, q, leadID, l.Phone, l.PaymentAmount, now)
	})
	return orderID, err
}

// TawarkanReguler reopens a participant's lead in Tahsin Reguler so CS can
// offer the regular class; it may later be converted into a santri. The
// participant order and attendance stay.
func (s *LeadService) TawarkanReguler(ctx context.Context, leadID, actorID int64) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		order, err := q.GetLeadProgramOrder(ctx, leadID)
		if err != nil {
			return err
		}
		if order == nil {
			return errors.New("hanya lead peserta program pendek yang bisa ditawari kelas reguler")
		}
		before, err := s.effectiveProgram(ctx, q, leadID)
		if err != nil {
			return err
		}
		now := s.nowUTC()
		n, err := q.ReopenLeadForReguler(ctx, leadID, now)
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("lead ini sudah dibuka di Tahsin Reguler atau sudah menjadi santri")
		}
		return writeLeadEvent(ctx, q, leadID, newLeadEventID("usr"), "lead.offered_reguler", userActor(actorID), map[string]any{
			"from": before.Nama, "to": "Tahsin Reguler", "order_id": order.OrderID, "status_from": LeadStatusWon, "status_to": LeadStatusOpen,
		}, now)
	})
}
