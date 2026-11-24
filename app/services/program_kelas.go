package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Classes of short programs. Participants are split ikhwan (L) / akhwat (P)
// and filled in order: a class takes participants (oldest order first) until
// it holds lead_programs.kapasitas_kelas, then the next class of that gender
// is created. A placed participant is never moved automatically; admins can
// move them by hand. Participants whose gender is unknown (an order recorded
// by hand without a lead) wait under "Belum ada kelas" until an admin sets it.

const DefaultKapasitasKelas int64 = 30

var errKelasTidakAda = errors.New("kelas tidak ditemukan")

// ProgramKelasNama is the display name: "Ikhwan 1", "Akhwat 2".
func ProgramKelasNama(jk string, urutan int64) string {
	if jk == "P" {
		return fmt.Sprintf("Akhwat %d", urutan)
	}
	return fmt.Sprintf("Ikhwan %d", urutan)
}

func normalJK(jk string) string {
	switch strings.ToUpper(strings.TrimSpace(jk)) {
	case "L":
		return "L"
	case "P":
		return "P"
	}
	return ""
}

// syncProgramKelas places every active participant with a known gender that
// has no (valid) class yet. It returns the classes with their active counts.
// Call it inside a transaction.
func syncProgramKelas(ctx context.Context, q *queries.Querier, p *models.LeadProgram, peserta []models.ProgramPeserta, now func() time.Time) ([]models.ProgramKelas, map[int64]models.ProgramPesertaKelas, error) {
	kelas, err := q.ListProgramKelas(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := q.ListProgramPesertaKelas(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	kapasitas := p.KapasitasKelas
	if kapasitas <= 0 {
		kapasitas = DefaultKapasitasKelas
	}
	byID := map[int64]int{}
	for i := range kelas {
		byID[kelas[i].ID] = i
	}
	valid := func(r models.ProgramPesertaKelas) bool {
		i, ok := byID[r.KelasID]
		return ok && kelas[i].JenisKelamin == r.JenisKelamin
	}
	// Count only active participants: cancelled orders free their seat.
	for _, x := range peserta {
		if r, ok := rows[x.OrderID]; ok && valid(r) {
			kelas[byID[r.KelasID]].Jumlah++
		}
	}
	place := func(jk string) (int64, error) {
		var last int64
		for i := range kelas {
			if kelas[i].JenisKelamin != jk {
				continue
			}
			last = max(last, kelas[i].Urutan)
			if kelas[i].Jumlah < kapasitas {
				kelas[i].Jumlah++
				return kelas[i].ID, nil
			}
		}
		id, err := q.CreateProgramKelas(ctx, p.ID, jk, last+1, now())
		if err != nil {
			return 0, err
		}
		kelas = append(kelas, models.ProgramKelas{ID: id, ProgramID: p.ID, JenisKelamin: jk, Urutan: last + 1, Jumlah: 1})
		byID[id] = len(kelas) - 1
		return id, nil
	}
	for _, x := range peserta {
		r, ok := rows[x.OrderID]
		if ok && valid(r) {
			continue
		}
		jk := r.JenisKelamin
		if !ok {
			jk = normalJK(x.LeadJenisKelamin)
		}
		if jk == "" {
			continue
		}
		id, err := place(jk)
		if err != nil {
			return nil, nil, err
		}
		r = models.ProgramPesertaKelas{OrderID: x.OrderID, JenisKelamin: jk, KelasID: id, Cara: "auto"}
		if err := q.UpsertProgramPesertaKelas(ctx, p.ID, x.OrderID, jk, id, r.Cara, now()); err != nil {
			return nil, nil, err
		}
		rows[x.OrderID] = r
	}
	for i := range kelas {
		kelas[i].Nama = ProgramKelasNama(kelas[i].JenisKelamin, kelas[i].Urutan)
	}
	return kelas, rows, nil
}

// applyKelas copies the class assignment onto the participant list.
func applyKelas(peserta []models.ProgramPeserta, rows map[int64]models.ProgramPesertaKelas) {
	for i := range peserta {
		if r, ok := rows[peserta[i].OrderID]; ok {
			peserta[i].JenisKelamin, peserta[i].KelasID, peserta[i].KelasManual = r.JenisKelamin, r.KelasID, r.Cara == "manual"
		} else {
			peserta[i].JenisKelamin = normalJK(peserta[i].LeadJenisKelamin)
		}
	}
}

// withTx runs fn on a transaction of the program service's database.
func (s *LeadProgramService) withTx(ctx context.Context, fn func(q *queries.Querier) error) error {
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.querier.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// syncKelas places new participants of a short program (no-op otherwise).
func (s *LeadProgramService) syncKelas(ctx context.Context, p *models.LeadProgram) ([]models.ProgramKelas, map[int64]models.ProgramPesertaKelas, []models.ProgramPeserta, error) {
	if p.Jenis != LeadProgramJenisProgram || p.ProdukID <= 0 {
		return []models.ProgramKelas{}, map[int64]models.ProgramPesertaKelas{}, []models.ProgramPeserta{}, nil
	}
	var (
		kelas   []models.ProgramKelas
		rows    map[int64]models.ProgramPesertaKelas
		peserta []models.ProgramPeserta
	)
	err := s.withTx(ctx, func(q *queries.Querier) error {
		var err error
		if peserta, err = q.ListProgramPeserta(ctx, p.ProdukID, p.ProdukBatchID); err != nil {
			return err
		}
		kelas, rows, err = syncProgramKelas(ctx, q, p, peserta, s.now)
		return err
	})
	return kelas, rows, peserta, err
}

// SetPesertaKelas sets a participant's gender (auto class) or moves them to
// a class. Returns the program id for the redirect.
func (s *LeadProgramService) SetPesertaKelas(programID, orderID int64, r models.ProgramPesertaKelasRequest) error {
	ctx := context.Background()
	p, err := s.querier.GetLeadProgram(ctx, programID)
	if err != nil {
		return err
	}
	if p.Jenis != LeadProgramJenisProgram || p.ProdukID <= 0 {
		return errors.New("kelas hanya untuk program pendek yang sudah punya produk")
	}
	return s.withTx(ctx, func(q *queries.Querier) error {
		peserta, err := q.ListProgramPeserta(ctx, p.ProdukID, p.ProdukBatchID)
		if err != nil {
			return err
		}
		found := false
		for _, x := range peserta {
			found = found || x.OrderID == orderID
		}
		if !found {
			return errors.New("peserta tidak ditemukan di program ini")
		}
		if r.KelasID <= 0 {
			jk := normalJK(r.JenisKelamin)
			if jk == "" {
				return errors.New("pilih ikhwan atau akhwat")
			}
			if err := q.UpsertProgramPesertaKelas(ctx, p.ID, orderID, jk, 0, "auto", s.now()); err != nil {
				return err
			}
			_, _, err = syncProgramKelas(ctx, q, p, peserta, s.now)
			return err
		}
		k, err := q.GetProgramKelas(ctx, r.KelasID)
		if err != nil || k.ProgramID != p.ID {
			return errKelasTidakAda
		}
		rows, err := q.ListProgramPesertaKelas(ctx, p.ID)
		if err != nil {
			return err
		}
		if cur, ok := rows[orderID]; ok && cur.KelasID == k.ID {
			return nil
		}
		var jumlah int64
		for _, x := range peserta {
			if rows[x.OrderID].KelasID == k.ID {
				jumlah++
			}
		}
		kapasitas := max(p.KapasitasKelas, 1)
		if jumlah >= kapasitas {
			return fmt.Errorf("%s sudah penuh (%d/%d)", ProgramKelasNama(k.JenisKelamin, k.Urutan), jumlah, kapasitas)
		}
		return q.UpsertProgramPesertaKelas(ctx, p.ID, orderID, k.JenisKelamin, k.ID, "manual", s.now())
	})
}

// UpdateKelas sets a class's teacher and Zoom link. Returns its program id.
func (s *LeadProgramService) UpdateKelas(kelasID int64, r models.ProgramKelasRequest) (int64, error) {
	ctx := context.Background()
	k, err := s.querier.GetProgramKelas(ctx, kelasID)
	if err != nil {
		return 0, errKelasTidakAda
	}
	r.Pengajar = strings.Join(strings.Fields(r.Pengajar), " ")
	r.ZoomURL = strings.TrimSpace(r.ZoomURL)
	if utf8.RuneCountInString(r.Pengajar) > 80 {
		return k.ProgramID, errors.New("nama pengajar maksimal 80 karakter")
	}
	if r.ZoomURL != "" {
		u, err := url.Parse(r.ZoomURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(r.ZoomURL) > 500 {
			return k.ProgramID, errors.New("link Zoom harus berupa URL lengkap, mis. https://zoom.us/j/123")
		}
	}
	return k.ProgramID, s.querier.UpdateProgramKelas(ctx, kelasID, r.Pengajar, r.ZoomURL)
}

// DeleteKelas removes an empty class (e.g. after admins moved everyone out).
func (s *LeadProgramService) DeleteKelas(kelasID int64) (int64, error) {
	ctx := context.Background()
	k, err := s.querier.GetProgramKelas(ctx, kelasID)
	if err != nil {
		return 0, errKelasTidakAda
	}
	p, err := s.querier.GetLeadProgram(ctx, k.ProgramID)
	if err != nil {
		return k.ProgramID, err
	}
	kelas, _, _, err := s.syncKelas(ctx, p)
	if err != nil {
		return k.ProgramID, err
	}
	for _, x := range kelas {
		if x.ID == kelasID && x.Jumlah > 0 {
			return k.ProgramID, fmt.Errorf("%s masih berisi %d peserta; pindahkan dulu", x.Nama, x.Jumlah)
		}
	}
	return k.ProgramID, s.querier.DeleteProgramKelas(ctx, kelasID)
}

// pesertaKelasInfo is the class shown to a participant after check-in.
func pesertaKelasInfo(ctx context.Context, q *queries.Querier, programID, orderID int64) (string, string) {
	rows, err := q.ListProgramPesertaKelas(ctx, programID)
	if err != nil {
		return "", ""
	}
	r, ok := rows[orderID]
	if !ok || r.KelasID == 0 {
		return "", ""
	}
	k, err := q.GetProgramKelas(ctx, r.KelasID)
	if err != nil {
		return "", ""
	}
	return ProgramKelasNama(k.JenisKelamin, k.Urutan), k.ZoomURL
}
