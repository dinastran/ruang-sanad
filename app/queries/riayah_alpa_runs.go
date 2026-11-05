package queries

import "context"

// RiayahAlpaRun adalah rangkaian alpa terakhir yang pernah memenuhi ambang
// dua kali berturut-turut, termasuk setelah santri kembali hadir.
type RiayahAlpaRun struct {
	SantriID int64
	AlpaBeruntun int64
	AlpaTerakhir string
}

// Tidak dibatasi 12 absensi: kasus belum selesai tidak boleh hilang sendiri.
func (q *Queries) ListRiayahAlpaRuns(ctx context.Context, guruID interface{}) ([]RiayahAlpaRun, error) {
	const statement = `WITH riwayat AS (
    SELECT a.santri_id, a.status, substr(CAST(p.tanggal AS TEXT), 1, 10) AS tanggal,
        SUM(CASE WHEN a.status = 'alpa' THEN 0 ELSE 1 END)
        OVER (PARTITION BY a.santri_id ORDER BY p.tanggal, p.id
              ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS kelompok
    FROM absensi a
    JOIN pertemuan p ON p.id = a.pertemuan_id
    JOIN santri s ON s.id = a.santri_id
    LEFT JOIN kelas k ON k.id = s.kelas_id
    WHERE p.status = 'selesai'
      AND s.status = 'aktif'
      AND (? IS NULL OR k.guru_id = ?)
), rangkaian AS (
    SELECT santri_id, kelompok, COUNT(*) AS alpa_beruntun, MAX(tanggal) AS alpa_terakhir
    FROM riwayat WHERE status = 'alpa'
    GROUP BY santri_id, kelompok
    HAVING COUNT(*) >= 2
), terbaru AS (
    SELECT santri_id, alpa_beruntun, alpa_terakhir,
        ROW_NUMBER() OVER (
            PARTITION BY santri_id ORDER BY alpa_terakhir DESC, kelompok DESC
        ) AS urutan
    FROM rangkaian
)
SELECT santri_id, alpa_beruntun, alpa_terakhir
FROM terbaru WHERE urutan = 1`
	rows, err := q.db.QueryContext(ctx, statement, guruID, guruID)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]RiayahAlpaRun, 0)
	for rows.Next() {
		var item RiayahAlpaRun
		if err := rows.Scan(&item.SantriID, &item.AlpaBeruntun, &item.AlpaTerakhir); err != nil { return nil, err }
		out = append(out, item)
	}
	return out, rows.Err()
}
