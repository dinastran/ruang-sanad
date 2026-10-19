package queries

// Handwritten read-only queries for the AI agent (MCP) tools. Following
// overview.go: aggregate SQL that never writes. No query here selects a
// contact column (no_wa, email, voice_note_url, domisili, nama_pembeli).
//
// Date columns are stored by the driver as "YYYY-MM-DD 00:00:00 +0000 UTC",
// so days/months are compared with substr(col, 1, 10) / substr(col, 1, 7).
import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
)

// AIProgramDimensions maps a program dimension to its santri column.
var AIProgramDimensions = map[string]string{
	"level":     "s.level",
	"tipe":      "s.tipe",
	"frekuensi": "s.frekuensi",
	"angkatan":  "s.angkatan",
}

func aiDimExpr(dim string) (string, error) {
	col, ok := AIProgramDimensions[dim]
	if !ok {
		return "", fmt.Errorf("dimensi %q tidak dikenal", dim)
	}
	return "COALESCE(NULLIF(TRIM(" + col + "), ''), 'Belum diisi')", nil
}

func aiDate(ns sql.NullString) string {
	if !ns.Valid || len(ns.String) < 10 {
		return ""
	}
	return ns.String[:10]
}

func aiOptInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

// --- Santri ---------------------------------------------------------------

const aiSantriRingkasCols = `s.id, s.id_mahasantri, s.nama, s.status, s.angkatan, s.angkatan_kelas,
 s.level, s.tipe, s.frekuensi, COALESCE(s.kelas_id, 0), COALESCE(k.nama_kelas, ''), COALESCE(g.nama, '')`

func scanAISantriRingkas(sc interface{ Scan(...any) error }, r *models.AISantriRingkas, extra ...any) error {
	dest := []any{&r.ID, &r.IDMahasantri, &r.Nama, &r.Status, &r.Angkatan, &r.AngkatanKelas,
		&r.Level, &r.Tipe, &r.Frekuensi, &r.KelasID, &r.NamaKelas, &r.GuruNama}
	return sc.Scan(append(dest, extra...)...)
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// AICariSantri matches every word of term against nama or id_mahasantri.
func (q *Queries) AICariSantri(ctx context.Context, term, status string, limit int64) ([]models.AISantriRingkas, error) {
	where := []string{"1=1"}
	args := []any{}
	for _, w := range strings.Fields(term) {
		where = append(where, `(s.nama LIKE ? ESCAPE '\' OR s.id_mahasantri LIKE ? ESCAPE '\')`)
		p := "%" + escapeLike(w) + "%"
		args = append(args, p, p)
	}
	if status != "" {
		where = append(where, "s.status = ?")
		args = append(args, status)
	}
	args = append(args, limit)
	rows, err := q.db.QueryContext(ctx, `SELECT `+aiSantriRingkasCols+`
	 FROM santri s LEFT JOIN kelas k ON k.id = s.kelas_id LEFT JOIN guru g ON g.id = k.guru_id
	 WHERE `+strings.Join(where, " AND ")+` ORDER BY s.status = 'aktif' DESC, s.nama LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AISantriRingkas, 0)
	for rows.Next() {
		var r models.AISantriRingkas
		if err := scanAISantriRingkas(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) AIGetSantriProfil(ctx context.Context, id int64) (*models.AISantriProfil, error) {
	var p models.AISantriProfil
	var usia sql.NullInt64
	var tglDaftar, mulai, created, updated sql.NullString
	var lengkap, nominal int64
	row := q.db.QueryRowContext(ctx, `SELECT `+aiSantriRingkasCols+`,
	 s.jenis_kelamin, s.usia, CAST(s.tanggal_daftar AS TEXT), CAST(s.mulai_belajar AS TEXT), s.status_alasan, s.cuti_mulai, s.cuti_selesai,
	 s.nominal, s.is_lengkap, s.keterangan_tidak_lanjut, s.jadwal_catatan, s.pertemuan_awal,
	 CAST(s.created_at AS TEXT), CAST(s.updated_at AS TEXT)
	 FROM santri s LEFT JOIN kelas k ON k.id = s.kelas_id LEFT JOIN guru g ON g.id = k.guru_id
	 WHERE s.id = ?`, id)
	err := scanAISantriRingkas(row, &p.AISantriRingkas, &p.JenisKelamin, &usia, &tglDaftar, &mulai, &p.StatusAlasan,
		&p.CutiMulai, &p.CutiSelesai, &nominal, &lengkap, &p.KeteranganTL, &p.JadwalCatatan, &p.PertemuanAwal, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Usia, p.TanggalDaftar, p.MulaiBelajar = aiOptInt(usia), aiDate(tglDaftar), aiDate(mulai)
	p.Nominal = &nominal
	p.IsLengkap, p.DibuatPada, p.TerakhirDiubah = lengkap == 1, aiDate(created), aiDate(updated)
	return &p, nil
}

func (q *Queries) AIListSantriStatusLog(ctx context.Context, santriID, limit int64) ([]models.AIStatusLog, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT CAST(created_at AS TEXT), status_lama, status_baru, alasan
	 FROM santri_status_log WHERE santri_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`, santriID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIStatusLog, 0)
	for rows.Next() {
		var r models.AIStatusLog
		var tgl sql.NullString
		if err := rows.Scan(&tgl, &r.StatusLama, &r.StatusBaru, &r.Alasan); err != nil {
			return nil, err
		}
		r.Tanggal = aiDate(tgl)
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Tagihan --------------------------------------------------------------

// AITunggakanPerSantri groups overdue unpaid tagihan per santri, using the
// same overdue rule as the finance dashboard (jatuh_tempo, else tanggal_tagih).
func (q *Queries) AITunggakanPerSantri(ctx context.Context, hariIni string) (map[int64]models.AITunggakanSantri, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT santri_id, COUNT(*), CAST(COALESCE(SUM(nominal), 0) AS INTEGER),
	 CAST(MAX(julianday(?) - julianday(substr(COALESCE(jatuh_tempo, tanggal_tagih), 1, 10))) AS INTEGER),
	 COALESCE(MAX(CAST(fu_terakhir AS TEXT)), ''), CAST(COALESCE(SUM(fu_count), 0) AS INTEGER)
	 FROM tagihan WHERE status = 'belum_bayar' AND substr(COALESCE(jatuh_tempo, tanggal_tagih), 1, 10) < ?
	 GROUP BY santri_id`, hariIni, hariIni)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]models.AITunggakanSantri{}
	for rows.Next() {
		var r models.AITunggakanSantri
		if err := rows.Scan(&r.SantriID, &r.JumlahTagihan, &r.Nominal, &r.HariTelatMaks, &r.FUTerakhir, &r.FUCount); err != nil {
			return nil, err
		}
		if len(r.FUTerakhir) > 10 {
			r.FUTerakhir = r.FUTerakhir[:10]
		}
		out[r.SantriID] = r
	}
	return out, rows.Err()
}

func (q *Queries) AIListTagihanSantri(ctx context.Context, santriID int64, hariIni string, limit int64) ([]models.AITagihanSantri, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, bulan_ke, nominal, CAST(tanggal_tagih AS TEXT), CAST(jatuh_tempo AS TEXT), status,
	 CAST(tanggal_bayar AS TEXT), fu_count, CAST(fu_terakhir AS TEXT),
	 CAST(julianday(?) - julianday(substr(COALESCE(jatuh_tempo, tanggal_tagih), 1, 10)) AS INTEGER)
	 FROM tagihan WHERE santri_id = ? ORDER BY bulan_ke DESC LIMIT ?`, hariIni, santriID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AITagihanSantri, 0)
	for rows.Next() {
		var r models.AITagihanSantri
		var tagih, tempo, bayar, fu sql.NullString
		var hari sql.NullInt64
		if err := rows.Scan(&r.ID, &r.BulanKe, &r.Nominal, &tagih, &tempo, &r.Status, &bayar, &r.FUCount, &fu, &hari); err != nil {
			return nil, err
		}
		r.TanggalTagih, r.JatuhTempo, r.TanggalBayar, r.FUTerakhir = aiDate(tagih), aiDate(tempo), aiDate(bayar), aiDate(fu)
		if r.Status == "belum_bayar" && hari.Valid && hari.Int64 > 0 {
			r.HariTelat = hari.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Program trend ----------------------------------------------------------

func (q *Queries) AIProgramSnapshot(ctx context.Context, dim string) ([]models.AIProgramSnapshot, error) {
	expr, err := aiDimExpr(dim)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, `SELECT `+expr+` AS nilai,
	 SUM(CASE WHEN s.status = 'aktif' THEN 1 ELSE 0 END),
	 SUM(CASE WHEN s.status = 'cuti' THEN 1 ELSE 0 END),
	 SUM(CASE WHEN s.status IN ('nonaktif', 'tidak_lanjut') THEN 1 ELSE 0 END),
	 COUNT(*)
	 FROM santri s GROUP BY 1 ORDER BY 2 DESC, 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIProgramSnapshot, 0)
	for rows.Next() {
		var r models.AIProgramSnapshot
		if err := rows.Scan(&r.Nilai, &r.Aktif, &r.Cuti, &r.Berhenti, &r.Total); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) aiProgramMonthly(ctx context.Context, query string, args ...any) ([]models.AIProgramBulan, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIProgramBulan, 0)
	for rows.Next() {
		var r models.AIProgramBulan
		if err := rows.Scan(&r.Nilai, &r.Bulan, &r.Angka, &r.Angka2); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AIProgramPendaftar counts new registrations per month (tanggal_daftar).
func (q *Queries) AIProgramPendaftar(ctx context.Context, dim, dariBulan string) ([]models.AIProgramBulan, error) {
	expr, err := aiDimExpr(dim)
	if err != nil {
		return nil, err
	}
	return q.aiProgramMonthly(ctx, `SELECT `+expr+`, substr(s.tanggal_daftar, 1, 7) AS bulan, COUNT(*), 0
	 FROM santri s WHERE s.tanggal_daftar IS NOT NULL AND substr(s.tanggal_daftar, 1, 7) >= ?
	 GROUP BY 1, 2 ORDER BY 2, 1`, dariBulan)
}

// AIProgramBerhenti counts status changes to nonaktif/tidak_lanjut per month.
// Angka2 counts changes to cuti in the same month.
func (q *Queries) AIProgramBerhenti(ctx context.Context, dim, dariBulan string) ([]models.AIProgramBulan, error) {
	expr, err := aiDimExpr(dim)
	if err != nil {
		return nil, err
	}
	return q.aiProgramMonthly(ctx, `SELECT `+expr+`, substr(l.created_at, 1, 7) AS bulan,
	 SUM(CASE WHEN l.status_baru IN ('nonaktif', 'tidak_lanjut') THEN 1 ELSE 0 END),
	 SUM(CASE WHEN l.status_baru = 'cuti' THEN 1 ELSE 0 END)
	 FROM santri_status_log l JOIN santri s ON s.id = l.santri_id
	 WHERE substr(l.created_at, 1, 7) >= ? GROUP BY 1, 2 ORDER BY 2, 1`, dariBulan)
}

// AIProgramKehadiran returns attendance records (Angka) and hadir+telat
// (Angka2) per month from finished meetings.
func (q *Queries) AIProgramKehadiran(ctx context.Context, dim, dariBulan string) ([]models.AIProgramBulan, error) {
	expr, err := aiDimExpr(dim)
	if err != nil {
		return nil, err
	}
	return q.aiProgramMonthly(ctx, `SELECT `+expr+`, substr(p.tanggal, 1, 7) AS bulan, COUNT(*),
	 SUM(CASE WHEN a.status IN ('hadir', 'telat') THEN 1 ELSE 0 END)
	 FROM absensi a JOIN pertemuan p ON p.id = a.pertemuan_id JOIN santri s ON s.id = a.santri_id
	 WHERE p.status = 'selesai' AND substr(p.tanggal, 1, 7) >= ? GROUP BY 1, 2 ORDER BY 2, 1`, dariBulan)
}

// AIProdukPenjualan returns quantity (Angka) and omzet (Angka2) per product per
// order month for active orders. Buyer names and numbers are not selected.
func (q *Queries) AIProdukPenjualan(ctx context.Context, dariBulan string) ([]models.AIProgramBulan, error) {
	return q.aiProgramMonthly(ctx, `SELECT pr.nama, substr(o.tanggal, 1, 7) AS bulan,
	 CAST(COALESCE(SUM(i.qty), 0) AS INTEGER), CAST(COALESCE(SUM(i.qty * i.harga_satuan), 0) AS INTEGER)
	 FROM product_orders o JOIN product_order_items i ON i.order_id = o.id JOIN produk pr ON pr.id = i.produk_id
	 WHERE o.status = 'aktif' AND substr(o.tanggal, 1, 7) >= ? GROUP BY 1, 2 ORDER BY 2, 1`, dariBulan)
}

// AIProdukPenugasan counts products assigned to mahasantri per month.
func (q *Queries) AIProdukPenugasan(ctx context.Context, dariBulan string) ([]models.AIProgramBulan, error) {
	return q.aiProgramMonthly(ctx, `SELECT pr.nama, substr(mp.tanggal, 1, 7) AS bulan, COUNT(*), 0
	 FROM mahasantri_produk mp JOIN produk pr ON pr.id = mp.produk_id
	 WHERE mp.status = 'aktif' AND substr(mp.tanggal, 1, 7) >= ? GROUP BY 1, 2 ORDER BY 2, 1`, dariBulan)
}

// --- Kelas ------------------------------------------------------------------

// AIKelasMetrik returns operational metrics for one class (kelasID > 0,
// regardless of its active flag) or for all active classes, plus inactive
// ones when termasukNonaktif.
func (q *Queries) AIKelasMetrik(ctx context.Context, hariIni, sejak string, kelasID int64, termasukNonaktif bool) ([]models.AIKelasMetrik, error) {
	semua := 0
	if termasukNonaktif {
		semua = 1
	}
	rows, err := q.db.QueryContext(ctx, `SELECT k.id, k.nama_kelas, k.angkatan, k.level, k.tipe, k.frekuensi, k.jadwal,
	 COALESCE(k.guru_id, 0), COALESCE(g.nama, ''), COALESCE(g.is_aktif, 0), k.kapasitas,
	 (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'aktif'),
	 (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'cuti'),
	 (SELECT COUNT(*) FROM pertemuan p WHERE p.kelas_id = k.id AND p.status = 'selesai' AND substr(p.tanggal, 1, 10) >= ?),
	 (SELECT COUNT(*) FROM pertemuan p WHERE p.kelas_id = k.id AND p.status = 'selesai' AND p.is_badal = 1 AND substr(p.tanggal, 1, 10) >= ?),
	 COALESCE((SELECT MAX(substr(p.tanggal, 1, 10)) FROM pertemuan p WHERE p.kelas_id = k.id AND p.status = 'selesai'), ''),
	 (SELECT COUNT(*) FROM pertemuan p WHERE p.kelas_id = k.id AND p.status = 'berlangsung' AND substr(p.tanggal, 1, 10) < ?),
	 (SELECT COUNT(*) FROM absensi a JOIN pertemuan p ON p.id = a.pertemuan_id WHERE p.kelas_id = k.id AND p.status = 'selesai' AND substr(p.tanggal, 1, 10) >= ?),
	 (SELECT COUNT(*) FROM absensi a JOIN pertemuan p ON p.id = a.pertemuan_id WHERE p.kelas_id = k.id AND p.status = 'selesai' AND substr(p.tanggal, 1, 10) >= ? AND a.status IN ('hadir', 'telat'))
	 FROM kelas k LEFT JOIN guru g ON g.id = k.guru_id
	 WHERE (? > 0 AND k.id = ?) OR (? = 0 AND (k.is_aktif = 1 OR ? = 1))
	 ORDER BY k.is_aktif DESC, k.nama_kelas`, sejak, sejak, hariIni, sejak, sejak, kelasID, kelasID, kelasID, semua)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIKelasMetrik, 0)
	for rows.Next() {
		var r models.AIKelasMetrik
		var guruAktif int64
		if err := rows.Scan(&r.ID, &r.NamaKelas, &r.Angkatan, &r.Level, &r.Tipe, &r.Frekuensi, &r.Jadwal,
			&r.GuruID, &r.GuruNama, &guruAktif, &r.Kapasitas, &r.SantriAktif, &r.SantriCuti,
			&r.PertemuanSelesai30, &r.PertemuanBadal30, &r.PertemuanTerakhir, &r.PertemuanBelumTutup,
			&r.Absensi30, &r.Hadir30); err != nil {
			return nil, err
		}
		r.GuruAktif = guruAktif == 1
		if r.Absensi30 > 0 {
			v := float64(r.Hadir30) * 100 / float64(r.Absensi30)
			r.PersenHadir30 = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) AIListPertemuanKelas(ctx context.Context, kelasID, limit int64) ([]models.AIPertemuanRingkas, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT p.id, p.pertemuan_ke, substr(p.tanggal, 1, 10), p.status, p.materi, p.catatan,
	 p.is_badal, p.is_reschedule,
	 (SELECT COUNT(*) FROM absensi a WHERE a.pertemuan_id = p.id AND a.status IN ('hadir', 'telat')),
	 (SELECT COUNT(*) FROM absensi a WHERE a.pertemuan_id = p.id)
	 FROM pertemuan p WHERE p.kelas_id = ? ORDER BY p.tanggal DESC, p.pertemuan_ke DESC LIMIT ?`, kelasID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIPertemuanRingkas, 0)
	for rows.Next() {
		var r models.AIPertemuanRingkas
		var badal, resch int64
		if err := rows.Scan(&r.ID, &r.PertemuanKe, &r.Tanggal, &r.Status, &r.Materi, &r.Catatan, &badal, &resch, &r.Hadir, &r.TotalAbsen); err != nil {
			return nil, err
		}
		r.Badal, r.Reschedule = badal == 1, resch == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Guru -------------------------------------------------------------------

// AIGuruMetrik returns metrics for one guru (guruID > 0) or all active guru,
// plus inactive ones when termasukNonaktif.
func (q *Queries) AIGuruMetrik(ctx context.Context, hariIni, sejak30, sejak90 string, guruID int64, termasukNonaktif bool) ([]models.AIGuruMetrik, error) {
	semua := 0
	if termasukNonaktif {
		semua = 1
	}
	rows, err := q.db.QueryContext(ctx, `SELECT g.id, g.nama, g.gelar, g.status, g.is_aktif, COALESCE(substr(g.tanggal_gabung, 1, 10), ''),
	 (SELECT COUNT(*) FROM kelas k WHERE k.guru_id = g.id AND k.is_aktif = 1),
	 (SELECT COUNT(*) FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE k.guru_id = g.id AND k.is_aktif = 1 AND s.status = 'aktif'),
	 (SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id WHERE p.status = 'selesai' AND substr(p.tanggal, 1, 10) >= ?
	   AND ((p.is_badal = 1 AND p.guru_pengganti_id = g.id) OR (p.is_badal = 0 AND k.guru_id = g.id))),
	 (SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id WHERE p.status = 'berlangsung' AND substr(p.tanggal, 1, 10) < ? AND k.guru_id = g.id),
	 (SELECT COUNT(*) FROM kunjungan_tindak_lanjut t JOIN kunjungan_kelas kk ON kk.id = t.kunjungan_id WHERE kk.guru_id = g.id AND t.status = 'terbuka'),
	 (SELECT COUNT(*) FROM kunjungan_tindak_lanjut t JOIN kunjungan_kelas kk ON kk.id = t.kunjungan_id WHERE kk.guru_id = g.id AND t.status = 'terbuka'
	   AND t.target_tanggal IS NOT NULL AND substr(t.target_tanggal, 1, 10) < ?),
	 COALESCE((SELECT MAX(substr(kk.tanggal, 1, 10)) FROM kunjungan_kelas kk WHERE kk.guru_id = g.id AND kk.status = 'terlaksana'), ''),
	 (SELECT COUNT(*) FROM pembinaan_absen pa JOIN pembinaan pb ON pb.id = pa.pembinaan_id WHERE pa.guru_id = g.id AND pa.hadir = 1 AND substr(pb.tanggal, 1, 10) >= ?),
	 (SELECT COUNT(*) FROM pembinaan_absen pa JOIN pembinaan pb ON pb.id = pa.pembinaan_id WHERE pa.guru_id = g.id AND substr(pb.tanggal, 1, 10) >= ?),
	 (SELECT COUNT(*) FROM rapat_absen ra JOIN rapat_guru r ON r.id = ra.rapat_id WHERE ra.guru_id = g.id AND ra.hadir = 1 AND substr(r.tanggal, 1, 10) >= ?),
	 (SELECT COUNT(*) FROM rapat_absen ra JOIN rapat_guru r ON r.id = ra.rapat_id WHERE ra.guru_id = g.id AND substr(r.tanggal, 1, 10) >= ?)
	 FROM guru g WHERE (? > 0 AND g.id = ?) OR (? = 0 AND (g.is_aktif = 1 OR ? = 1)) ORDER BY g.is_aktif DESC, g.nama`,
		sejak30, hariIni, hariIni, sejak90, sejak90, sejak90, sejak90, guruID, guruID, guruID, semua)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIGuruMetrik, 0)
	for rows.Next() {
		var r models.AIGuruMetrik
		var aktif int64
		if err := rows.Scan(&r.ID, &r.Nama, &r.Gelar, &r.Status, &aktif, &r.TanggalGabung, &r.KelasAktif, &r.SantriAktif,
			&r.PertemuanSelesai30, &r.PertemuanBelumTutup, &r.TindakLanjutTerbuka, &r.TindakLanjutLewat, &r.KunjunganTerakhir,
			&r.PembinaanHadir, &r.PembinaanTotal, &r.RapatHadir, &r.RapatTotal); err != nil {
			return nil, err
		}
		r.Aktif = aktif == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) AIListKunjunganGuru(ctx context.Context, guruID, limit int64) ([]models.AIKunjunganRingkas, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT kk.id, COALESCE(substr(kk.tanggal, 1, 10), ''), kk.status, COALESCE(kk.kelas_id, 0), COALESCE(k.nama_kelas, ''),
	 kk.nilai_kedisiplinan, kk.nilai_materi, kk.nilai_metode, kk.nilai_interaksi, kk.catatan, kk.tanggapan_guru,
	 (SELECT COUNT(*) FROM kunjungan_tindak_lanjut t WHERE t.kunjungan_id = kk.id AND t.status = 'terbuka')
	 FROM kunjungan_kelas kk LEFT JOIN kelas k ON k.id = kk.kelas_id
	 WHERE kk.guru_id = ? ORDER BY COALESCE(kk.tanggal, kk.created_at) DESC, kk.id DESC LIMIT ?`, guruID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIKunjunganRingkas, 0)
	for rows.Next() {
		var r models.AIKunjunganRingkas
		var d, m, me, i sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Tanggal, &r.Status, &r.KelasID, &r.NamaKelas, &d, &m, &me, &i, &r.Catatan, &r.TanggapanGuru, &r.TLTerbuka); err != nil {
			return nil, err
		}
		r.NilaiDisiplin, r.NilaiMateri, r.NilaiMetode, r.NilaiInteraksi = aiOptInt(d), aiOptInt(m), aiOptInt(me), aiOptInt(i)
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Catatan search ----------------------------------------------------------

// aiCatatanSources lists every free-text note in the dashboard as
// (sumber, id, tanggal, judul, isi, penulis, santri_id, kelas_id, guru_id).
var aiCatatanSources = map[string]string{
	"riayah": `SELECT 'riayah', cr.id, substr(cr.created_at, 1, 10), 'Catatan riayah santri', cr.catatan,
	 COALESCE(gr.nama, u.name, ''), cr.target_id, COALESCE(s.kelas_id, 0), COALESCE(cr.guru_id, 0)
	 FROM catatan_riayah cr LEFT JOIN guru gr ON gr.id = cr.guru_id LEFT JOIN users u ON u.id = cr.author_user_id
	 LEFT JOIN santri s ON s.id = cr.target_id WHERE cr.target_type = 'santri'`,
	"kontak_riayah": `SELECT 'kontak_riayah', rk.id, substr(rk.tanggal, 1, 10), 'Kontak ' || rk.jenis || ' via ' || rk.media, rk.catatan,
	 COALESCE(gr.nama, u.name, ''), rk.santri_id, COALESCE(s.kelas_id, 0), COALESCE(rk.guru_id, 0)
	 FROM riayah_kontak rk LEFT JOIN guru gr ON gr.id = rk.guru_id LEFT JOIN users u ON u.id = rk.author_user_id
	 LEFT JOIN santri s ON s.id = rk.santri_id WHERE rk.catatan != ''`,
	"catatan_admin": `SELECT 'catatan_admin', n.id, substr(n.created_at, 1, 10), 'Catatan admin santri', n.catatan,
	 COALESCE(u.name, ''), n.santri_id, COALESCE(s.kelas_id, 0), 0
	 FROM santri_admin_note n LEFT JOIN users u ON u.id = n.author_user_id LEFT JOIN santri s ON s.id = n.santri_id`,
	"absensi": `SELECT 'absensi', a.id, substr(p.tanggal, 1, 10), 'Catatan absensi (' || a.status || ') pertemuan ke-' || p.pertemuan_ke, a.catatan,
	 '', a.santri_id, p.kelas_id, 0
	 FROM absensi a JOIN pertemuan p ON p.id = a.pertemuan_id WHERE a.catatan != ''`,
	"pertemuan": `SELECT 'pertemuan', p.id, substr(p.tanggal, 1, 10), 'Catatan pertemuan ke-' || p.pertemuan_ke,
	 TRIM(p.catatan || CASE WHEN p.alasan_badal != '' THEN ' | Alasan badal: ' || p.alasan_badal ELSE '' END
	   || CASE WHEN p.alasan_reschedule != '' THEN ' | Alasan reschedule: ' || p.alasan_reschedule ELSE '' END),
	 '', 0, p.kelas_id, COALESCE(k.guru_id, 0)
	 FROM pertemuan p LEFT JOIN kelas k ON k.id = p.kelas_id
	 WHERE p.catatan != '' OR p.alasan_badal != '' OR p.alasan_reschedule != ''`,
	"monitoring_kelas": `SELECT 'monitoring_kelas', n.id, substr(n.created_at, 1, 10), 'Catatan monitoring kelas', n.note,
	 COALESCE(u.name, ''), 0, jp.kelas_id, COALESCE(k.guru_id, 0)
	 FROM class_monitoring_notes n JOIN jadwal_pertemuan jp ON jp.id = n.jadwal_pertemuan_id
	 LEFT JOIN kelas k ON k.id = jp.kelas_id LEFT JOIN users u ON u.id = n.created_by`,
	"kunjungan": `SELECT 'kunjungan', kk.id, COALESCE(substr(kk.tanggal, 1, 10), substr(kk.created_at, 1, 10)), 'Kunjungan kelas (' || kk.status || ')',
	 TRIM(kk.catatan || CASE WHEN kk.tanggapan_guru != '' THEN ' | Tanggapan guru: ' || kk.tanggapan_guru ELSE '' END),
	 '', 0, COALESCE(kk.kelas_id, 0), kk.guru_id
	 FROM kunjungan_kelas kk WHERE kk.catatan != '' OR kk.tanggapan_guru != ''`,
	"tindak_lanjut": `SELECT 'tindak_lanjut', t.id, substr(t.created_at, 1, 10), 'Tindak lanjut ' || t.jenis || ' (' || t.status || ')', t.catatan,
	 COALESCE(u.name, ''), 0, COALESCE(kk.kelas_id, 0), kk.guru_id
	 FROM kunjungan_tindak_lanjut t JOIN kunjungan_kelas kk ON kk.id = t.kunjungan_id LEFT JOIN users u ON u.id = t.dibuat_oleh
	 WHERE t.catatan != ''`,
	"status_santri": `SELECT 'status_santri', l.id, substr(l.created_at, 1, 10), 'Perubahan status ' || l.status_lama || ' → ' || l.status_baru, l.alasan,
	 COALESCE(u.name, ''), l.santri_id, COALESCE(l.kelas_id, 0), 0
	 FROM santri_status_log l LEFT JOIN users u ON u.id = l.dibuat_oleh WHERE l.alasan != ''`,
	"pembinaan": `SELECT 'pembinaan', id, substr(tanggal, 1, 10), 'Pembinaan guru: ' || topik, keterangan, '', 0, 0, 0 FROM pembinaan`,
	"rapat":     `SELECT 'rapat', id, substr(tanggal, 1, 10), 'Rapat guru: ' || judul, catatan, '', 0, 0, 0 FROM rapat_guru`,
	"kalam": `SELECT 'kalam', id, substr(tanggal, 1, 10), 'Kalam bersanad: ' || topik || CASE WHEN kitab != '' THEN ' (' || kitab || ')' ELSE '' END,
	 keterangan, '', 0, 0, 0 FROM kalam_bersanad`,
	"todo": `SELECT 'todo', id, COALESCE(substr(deadline, 1, 10), substr(created_at, 1, 10)), 'Todo koordinator (' || status || '): ' || judul,
	 TRIM(teknis || ' ' || kebutuhan || ' ' || catatan), pic, 0, 0, 0 FROM todo_koordinator`,
}

// AICatatanSourceNames lists the searchable sources in a stable order.
var AICatatanSourceNames = []string{"riayah", "kontak_riayah", "catatan_admin", "absensi", "pertemuan", "monitoring_kelas",
	"kunjungan", "tindak_lanjut", "status_santri", "pembinaan", "rapat", "kalam", "todo"}

type AICatatanFilter struct {
	Sumber   []string
	Kata     []string
	Sejak    string
	SantriID int64
	KelasID  int64
	GuruID   int64
	Limit    int64
}

func (q *Queries) AICariCatatan(ctx context.Context, f AICatatanFilter) ([]models.AICatatan, error) {
	parts := make([]string, 0, len(f.Sumber))
	for _, name := range f.Sumber {
		src, ok := aiCatatanSources[name]
		if !ok {
			return nil, fmt.Errorf("sumber catatan %q tidak dikenal", name)
		}
		parts = append(parts, src)
	}
	if len(parts) == 0 {
		return []models.AICatatan{}, nil
	}
	where := []string{"1=1"}
	args := []any{}
	for _, w := range f.Kata {
		where = append(where, `(judul || ' ' || isi) LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(w)+"%")
	}
	if f.Sejak != "" {
		where = append(where, "tanggal >= ?")
		args = append(args, f.Sejak)
	}
	if f.SantriID > 0 {
		where = append(where, "santri_id = ?")
		args = append(args, f.SantriID)
	}
	if f.KelasID > 0 {
		where = append(where, "kelas_id = ?")
		args = append(args, f.KelasID)
	}
	if f.GuruID > 0 {
		where = append(where, "guru_id = ?")
		args = append(args, f.GuruID)
	}
	args = append(args, f.Limit)
	query := `WITH src(sumber, id, tanggal, judul, isi, penulis, santri_id, kelas_id, guru_id) AS (` +
		strings.Join(parts, "\n UNION ALL\n ") + `)
	 SELECT sumber, id, COALESCE(tanggal, ''), COALESCE(judul, ''), COALESCE(isi, ''), COALESCE(penulis, ''),
	 COALESCE(santri_id, 0), COALESCE(kelas_id, 0), COALESCE(guru_id, 0)
	 FROM src WHERE ` + strings.Join(where, " AND ") + ` ORDER BY tanggal DESC, id DESC LIMIT ?`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AICatatan, 0)
	for rows.Next() {
		var r models.AICatatan
		if err := rows.Scan(&r.Sumber, &r.ID, &r.Tanggal, &r.Judul, &r.Isi, &r.Penulis, &r.SantriID, &r.KelasID, &r.GuruID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Daftar (list) tools -----------------------------------------------------

// AIDaftarSantri lists santri with flexible filters and returns the total
// match count for paging. Contact columns are never selected.
func (q *Queries) AIDaftarSantri(ctx context.Context, f models.AISantriDaftarFilter) ([]models.AISantriDaftar, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, a ...any) {
		where = append(where, clause)
		args = append(args, a...)
	}
	if f.Status != "" {
		add("s.status = ?", f.Status)
	}
	if f.KelasID > 0 {
		add("s.kelas_id = ?", f.KelasID)
	}
	if f.GuruID > 0 {
		add("k.guru_id = ?", f.GuruID)
	}
	if f.Level != "" {
		add("s.level = ?", f.Level)
	}
	if f.Tipe != "" {
		add("s.tipe = ?", f.Tipe)
	}
	if f.Frekuensi != "" {
		add("s.frekuensi = ?", f.Frekuensi)
	}
	if f.Angkatan != "" {
		add("s.angkatan = ?", f.Angkatan)
	}
	if f.AngkatanKelas != "" {
		add("s.angkatan_kelas = ?", f.AngkatanKelas)
	}
	if f.JenisKelamin != "" {
		add("s.jenis_kelamin = ?", f.JenisKelamin)
	}
	for _, w := range strings.Fields(f.Kata) {
		p := "%" + escapeLike(w) + "%"
		add(`(s.nama LIKE ? ESCAPE '\' OR s.id_mahasantri LIKE ? ESCAPE '\')`, p, p)
	}
	if f.DaftarDari != "" {
		add("substr(s.tanggal_daftar, 1, 10) >= ?", f.DaftarDari)
	}
	if f.DaftarSampai != "" {
		add("substr(s.tanggal_daftar, 1, 10) <= ?", f.DaftarSampai)
	}
	if f.TanpaKelasAktif {
		// Same rule as OverviewOperations.WithoutClass.
		where = append(where, "(k.id IS NULL OR k.is_aktif != 1)")
	}
	if f.BelumLengkap {
		// Same rule as OverviewOperations.Incomplete.
		where = append(where, "s.is_lengkap = 0")
	}
	from := ` FROM santri s LEFT JOIN kelas k ON k.id = s.kelas_id LEFT JOIN guru g ON g.id = k.guru_id WHERE ` + strings.Join(where, " AND ")
	var total int64
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.db.QueryContext(ctx, `SELECT `+aiSantriRingkasCols+`, s.jenis_kelamin,
	 COALESCE(substr(s.tanggal_daftar, 1, 10), ''), COALESCE(substr(s.mulai_belajar, 1, 10), ''),
	 COALESCE(k.is_aktif, 0), s.is_lengkap`+from+`
	 ORDER BY s.nama COLLATE NOCASE, s.id LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]models.AISantriDaftar, 0)
	for rows.Next() {
		var r models.AISantriDaftar
		var aktif, lengkap int64
		if err := scanAISantriRingkas(rows, &r.AISantriRingkas, &r.JenisKelamin, &r.TanggalDaftar, &r.MulaiBelajar, &aktif, &lengkap); err != nil {
			return nil, 0, err
		}
		r.KelasAktif, r.DataLengkap = aktif == 1, lengkap == 1
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// AIDaftarTagihan lists tagihan with filters. Terlambat uses the finance
// dashboard rule (belum_bayar and past jatuh_tempo, else tanggal_tagih).
func (q *Queries) AIDaftarTagihan(ctx context.Context, f models.AITagihanDaftarFilter) ([]models.AITagihanDaftar, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, a ...any) {
		where = append(where, clause)
		args = append(args, a...)
	}
	if f.Status != "" {
		add("t.status = ?", f.Status)
	}
	if f.Terlambat {
		add("t.status = 'belum_bayar' AND substr(COALESCE(t.jatuh_tempo, t.tanggal_tagih), 1, 10) < ?", f.HariIni)
	}
	if f.SantriID > 0 {
		add("t.santri_id = ?", f.SantriID)
	}
	if f.KelasID > 0 {
		add("t.kelas_id = ?", f.KelasID)
	}
	if f.AngkatanKelas != "" {
		add("t.angkatan_kelas = ?", f.AngkatanKelas)
	}
	if f.TagihDari != "" {
		add("substr(t.tanggal_tagih, 1, 10) >= ?", f.TagihDari)
	}
	if f.TagihSampai != "" {
		add("substr(t.tanggal_tagih, 1, 10) <= ?", f.TagihSampai)
	}
	from := ` FROM tagihan t JOIN santri s ON s.id = t.santri_id LEFT JOIN kelas k ON k.id = t.kelas_id LEFT JOIN guru g ON g.id = k.guru_id
	 WHERE ` + strings.Join(where, " AND ")
	var total int64
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.db.QueryContext(ctx, `SELECT t.id, t.bulan_ke, t.nominal, CAST(t.tanggal_tagih AS TEXT), CAST(t.jatuh_tempo AS TEXT), t.status,
	 CAST(t.tanggal_bayar AS TEXT), t.fu_count, CAST(t.fu_terakhir AS TEXT),
	 CAST(julianday(?) - julianday(substr(COALESCE(t.jatuh_tempo, t.tanggal_tagih), 1, 10)) AS INTEGER),
	 t.santri_id, s.nama, s.id_mahasantri, s.status, COALESCE(k.nama_kelas, ''), COALESCE(g.nama, ''), t.angkatan_kelas`+from+`
	 ORDER BY t.status = 'belum_bayar' DESC, substr(COALESCE(t.jatuh_tempo, t.tanggal_tagih), 1, 10), t.id LIMIT ? OFFSET ?`,
		append(append([]any{f.HariIni}, args...), f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]models.AITagihanDaftar, 0)
	for rows.Next() {
		var r models.AITagihanDaftar
		var tagih, tempo, bayar, fu sql.NullString
		var hari sql.NullInt64
		if err := rows.Scan(&r.ID, &r.BulanKe, &r.Nominal, &tagih, &tempo, &r.Status, &bayar, &r.FUCount, &fu, &hari,
			&r.SantriID, &r.SantriNama, &r.IDMahasantri, &r.SantriStatus, &r.NamaKelas, &r.GuruNama, &r.Angkatan); err != nil {
			return nil, 0, err
		}
		r.TanggalTagih, r.JatuhTempo, r.TanggalBayar, r.FUTerakhir = aiDate(tagih), aiDate(tempo), aiDate(bayar), aiDate(fu)
		if r.Status == "belum_bayar" && hari.Valid && hari.Int64 > 0 {
			r.HariTelat = hari.Int64
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}
