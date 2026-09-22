// Nomor pertemuan yang ditampilkan ke pengguna dihitung per level kelas
// (mulai lagi dari 1 setelah kelas ganti level). pertemuan_ke tetap nomor
// internal berurutan yang dipakai tagihan.
export function nomorPertemuan(p: { pertemuan_ke: number; pertemuan_level_ke?: number }): number {
	return p.pertemuan_level_ke || p.pertemuan_ke;
}
