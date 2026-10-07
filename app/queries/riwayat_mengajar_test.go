package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestListRiwayatMengajarPreservesHistoricalGuruAttribution(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE guru (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			nama TEXT NOT NULL
		);
		CREATE TABLE kelas (
			id INTEGER PRIMARY KEY,
			nama_kelas TEXT NOT NULL,
			guru_id INTEGER
		);
		CREATE TABLE pertemuan (
			id INTEGER PRIMARY KEY,
			kelas_id INTEGER NOT NULL,
			pertemuan_ke INTEGER NOT NULL,
			pertemuan_level_ke INTEGER NOT NULL DEFAULT 1,
			level_nama TEXT NOT NULL DEFAULT '',
			tanggal DATETIME NOT NULL,
			jam_mulai TEXT NOT NULL DEFAULT '',
			jam_selesai TEXT NOT NULL DEFAULT '',
			materi TEXT NOT NULL DEFAULT '',
			catatan TEXT NOT NULL DEFAULT '',
			is_reschedule INTEGER NOT NULL DEFAULT 0,
			jadwal_semula TEXT NOT NULL DEFAULT '',
			alasan_reschedule TEXT NOT NULL DEFAULT '',
			is_badal INTEGER NOT NULL DEFAULT 0,
			guru_pengganti_id INTEGER,
			alasan_badal TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			dibuat_oleh INTEGER
		);

		INSERT INTO guru (id, user_id, nama) VALUES
			(1, 101, 'Guru Lama'),
			(2, 202, 'Guru Baru'),
			(3, 303, 'Guru Badal');
		INSERT INTO kelas (id, nama_kelas, guru_id) VALUES (10, 'Kelas A', 2);

		INSERT INTO pertemuan (
			id, kelas_id, pertemuan_ke, pertemuan_level_ke, level_nama, tanggal,
			jam_mulai, jam_selesai, materi, status, dibuat_oleh, is_badal, guru_pengganti_id
		) VALUES
			(1, 10, 1, 1, 'TQ', '2026-09-20', '10:00', '11:00', 'Materi 1', 'selesai', 101, 0, NULL),
			(2, 10, 2, 2, 'TQ', '2026-10-02', '10:00', '11:00', 'Materi 2', 'selesai', 202, 0, NULL),
			(3, 10, 3, 3, 'TQ', '2026-10-05', '10:00', '', '', 'berlangsung', 202, 1, 3);
	`)
	require.NoError(t, err)

	q := New(db)
	ctx := context.Background()

	oldGuru, err := q.ListRiwayatMengajar(ctx, ListRiwayatMengajarParams{
		GuruID:         sql.NullInt64{Int64: 1, Valid: true},
		TanggalMulai:   "2026-09-01",
		TanggalSelesai: "2026-10-07",
	})
	require.NoError(t, err)
	require.Len(t, oldGuru, 1)
	require.Equal(t, int64(1), oldGuru[0].ID)
	require.Equal(t, "Guru Lama", oldGuru[0].GuruNama)

	newGuru, err := q.ListRiwayatMengajar(ctx, ListRiwayatMengajarParams{
		GuruID:         sql.NullInt64{Int64: 2, Valid: true},
		TanggalMulai:   "2026-09-01",
		TanggalSelesai: "2026-10-07",
	})
	require.NoError(t, err)
	require.Len(t, newGuru, 1)
	require.Equal(t, int64(2), newGuru[0].ID)

	badal, err := q.ListRiwayatMengajar(ctx, ListRiwayatMengajarParams{
		GuruID:         sql.NullInt64{Int64: 3, Valid: true},
		TanggalMulai:   "2026-10-01",
		TanggalSelesai: "2026-10-07",
	})
	require.NoError(t, err)
	require.Len(t, badal, 1)
	require.Equal(t, int64(3), badal[0].ID)
	require.Equal(t, "Guru Badal", badal[0].GuruNama)
	require.Equal(t, "berlangsung", badal[0].Status)
}
