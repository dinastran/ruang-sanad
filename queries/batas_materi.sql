-- name: GetBatasMateriTerakhirByKelas :many
SELECT santri_id, batas_materi
FROM (
    SELECT
        a.santri_id,
        a.batas_materi,
        ROW_NUMBER() OVER (PARTITION BY a.santri_id ORDER BY p.pertemuan_ke DESC, a.id DESC) AS urutan
    FROM absensi a
    JOIN pertemuan p ON p.id = a.pertemuan_id
    WHERE p.kelas_id = ? AND p.status = 'selesai' AND a.batas_materi != ''
)
WHERE urutan = 1;
