import type { AbsenRow } from "@lib/types";

// Isian absensi pembinaan/rapat yang belum tersimpan disalin ke localStorage,
// supaya tidak hilang bila penyimpanan gagal, halaman tertutup, atau koneksi
// putus. Draf dihapus setelah absensi berhasil disimpan.

type DraftRow = Pick<AbsenRow, "guru_id" | "kehadiran" | "jam_masuk" | "keterangan" | "alasan">;

const kunci = (jenis: "pembinaan" | "rapat", id: number) => `absen-draf:${jenis}:${id}`;

function ringkas(rows: AbsenRow[]): DraftRow[] {
	return rows.map(({ guru_id, kehadiran, jam_masuk, keterangan, alasan }) => ({ guru_id, kehadiran, jam_masuk, keterangan, alasan }));
}

/** Gabungkan draf tersimpan ke data server. `dipulihkan` true bila draf mengubah isian. */
export function pulihkanDraf(jenis: "pembinaan" | "rapat", id: number, absen: AbsenRow[]): { rows: AbsenRow[]; dipulihkan: boolean } {
	const rows = absen.map((a) => ({ ...a }));
	let draf: DraftRow[] = [];
	try {
		draf = JSON.parse(localStorage.getItem(kunci(jenis, id)) ?? "[]");
	} catch {
		return { rows, dipulihkan: false };
	}
	if (!Array.isArray(draf) || draf.length === 0) return { rows, dipulihkan: false };
	const byGuru = new Map(draf.map((d) => [d.guru_id, d]));
	let dipulihkan = false;
	const hasil = rows.map((r) => {
		const d = byGuru.get(r.guru_id);
		if (!d) return r;
		const baru = { ...r, kehadiran: d.kehadiran ?? "", jam_masuk: d.jam_masuk ?? "", keterangan: d.keterangan ?? "", alasan: d.alasan ?? "" };
		baru.hadir = baru.kehadiran === "hadir";
		if (baru.kehadiran !== r.kehadiran || baru.jam_masuk !== r.jam_masuk || baru.keterangan !== r.keterangan || baru.alasan !== r.alasan) dipulihkan = true;
		return baru;
	});
	return { rows: hasil, dipulihkan };
}

export function simpanDraf(jenis: "pembinaan" | "rapat", id: number, rows: AbsenRow[]) {
	try {
		localStorage.setItem(kunci(jenis, id), JSON.stringify(ringkas(rows)));
	} catch {
		/* storage penuh/diblokir: form tetap jalan tanpa draf */
	}
}

export function hapusDraf(jenis: "pembinaan" | "rapat", id: number) {
	try {
		localStorage.removeItem(kunci(jenis, id));
	} catch {
		/* abaikan */
	}
}

/** Tanggal hari ini (YYYY-MM-DD) di zona WIB, sama dengan acuan server. */
export function hariIniWIB(): string {
	return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(new Date());
}
