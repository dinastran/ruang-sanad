package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrGuruTidakDitemukan = errors.New("Guru tidak ditemukan")
	ErrGuruMasihMengajar  = errors.New("Guru masih menjadi guru utama di kelas aktif; ganti guru kelasnya terlebih dahulu")
	ErrGuruPunyaRiwayat   = errors.New("Guru sudah memiliki riwayat sehingga tidak bisa dihapus; nonaktifkan saja")
)

// InfoHapus reports whether a guru can be permanently deleted. Deletion is
// only allowed for a guru with no class and no history at all (typically a
// duplicate or mistaken entry); everyone else is deactivated instead so their
// riayah, absensi, TSI, kunjungan, and ujroh records stay intact.
func (s *GuruService) InfoHapus(guruID int64) (*models.GuruHapusInfo, error) {
	ctx := context.Background()
	guru, err := s.querier.GuruGetByID(ctx, guruID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGuruTidakDitemukan
	}
	if err != nil {
		return nil, err
	}
	riwayat, err := s.querier.GetGuruRiwayat(ctx, guruID)
	if err != nil {
		return nil, err
	}

	info := &models.GuruHapusInfo{
		GuruID:     guru.ID,
		Nama:       guru.Nama,
		IsAktif:    guru.IsAktif == 1,
		KelasAktif: []string{},
		Riwayat:    guruRiwayatItems(riwayat),
	}
	if riwayat.KelasAktif > 0 {
		kelas, err := s.querier.ListKelasByGuruID(ctx, guruID)
		if err != nil {
			return nil, err
		}
		for _, k := range kelas {
			info.KelasAktif = append(info.KelasAktif, k.NamaKelas)
		}
	}
	info.BisaDihapus = riwayat.KelasAktif == 0 && len(info.Riwayat) == 0
	return info, nil
}

// Hapus permanently deletes a guru without history. The check is repeated
// inside the transaction so data added after the dialog opened still blocks it.
func (s *GuruService) Hapus(guruID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	riwayat, err := q.GetGuruRiwayat(ctx, guruID)
	if err != nil {
		return err
	}
	if riwayat.KelasAktif > 0 {
		return ErrGuruMasihMengajar
	}
	if len(guruRiwayatItems(riwayat)) > 0 {
		return ErrGuruPunyaRiwayat
	}
	n, err := q.DeleteGuruByID(ctx, guruID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGuruTidakDitemukan
	}
	return tx.Commit()
}

// Nonaktifkan hides a guru from active lists while keeping all history. A guru
// who still leads an active class must be replaced there first.
func (s *GuruService) Nonaktifkan(guruID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	riwayat, err := q.GetGuruRiwayat(ctx, guruID)
	if err != nil {
		return err
	}
	if riwayat.KelasAktif > 0 {
		return ErrGuruMasihMengajar
	}
	n, err := q.SetGuruNonaktif(ctx, guruID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGuruTidakDitemukan
	}
	return tx.Commit()
}

func guruRiwayatItems(r queries.GuruRiwayat) []models.GuruRiwayatItem {
	candidates := []models.GuruRiwayatItem{
		{Label: "Kelas nonaktif yang pernah diampu", Jumlah: r.KelasNonaktif},
		{Label: "Pertemuan sebagai badal", Jumlah: r.PertemuanBadal},
		{Label: "Jadwal sebagai badal", Jumlah: r.JadwalBadal},
		{Label: "Data ujroh", Jumlah: r.Ujroh},
		{Label: "Catatan riayah (ditulis atau tentang guru) & kontak", Jumlah: r.Riayah},
		{Label: "Absensi pembinaan & rapat", Jumlah: r.AbsensiKegiatan},
		{Label: "Check-in tilawah", Jumlah: r.Tilawah},
		{Label: "Kunjungan kelas", Jumlah: r.Kunjungan},
		{Label: "Penilaian TSI", Jumlah: r.Tsi},
		{Label: "Kalam bersanad dibagikan", Jumlah: r.KalamShare},
	}
	out := make([]models.GuruRiwayatItem, 0, len(candidates))
	for _, c := range candidates {
		if c.Jumlah > 0 {
			out = append(out, c)
		}
	}
	return out
}
