// Turunan data halaman Riwayat Absensi koordinator: normalisasi status,
// filter periode, dan agregasi untuk KPI, grafik, serta tabel.

export interface RiwayatAbsensi {
	guru_id: number;
	guru_nama: string;
	kegiatan: string; // pembinaan / rapat / kunjungan
	kegiatan_id: number;
	tanggal: string; // YYYY-MM-DD
	judul: string;
	hadir: boolean;
	kehadiran: string; // hadir/izin/sakit/alfa; kosong untuk kunjungan
	jam_masuk: string;
	keterangan: string;
	alasan: string;
}

export type StatusKehadiran = "hadir" | "izin" | "sakit" | "alfa";
export type StatusKunjungan = "terlaksana" | "dijadwalkan" | "batal";
export type StatusRiwayat = StatusKehadiran | StatusKunjungan;
export type Periode = "30h" | "3b" | "6b" | "12b" | "semua";
export type Kegiatan = "pembinaan" | "rapat" | "kunjungan";

export const STATUS_KEHADIRAN: StatusKehadiran[] = ["hadir", "izin", "sakit", "alfa"];

export const LABEL_STATUS: Record<StatusRiwayat, string> = {
	hadir: "Hadir",
	izin: "Izin",
	sakit: "Sakit",
	alfa: "Alfa",
	terlaksana: "Terlaksana",
	dijadwalkan: "Dijadwalkan",
	batal: "Batal",
};

/** Kelas warna per status: `pill` untuk badge, `bar` untuk isian grafik. */
export const WARNA_STATUS: Record<StatusRiwayat, { pill: string; bar: string }> = {
	hadir: { pill: "bg-success/10 text-emerald-700 dark:text-emerald-400", bar: "bg-emerald-500" },
	izin: { pill: "bg-sky-500/10 text-sky-700 dark:text-sky-400", bar: "bg-sky-500" },
	sakit: { pill: "bg-amber-500/10 text-amber-700 dark:text-amber-400", bar: "bg-amber-400" },
	alfa: { pill: "bg-error/10 text-red-700 dark:text-red-400", bar: "bg-red-500" },
	terlaksana: { pill: "bg-brand-400/10 text-brand-700 dark:text-brand-300", bar: "bg-brand-500" },
	dijadwalkan: { pill: "bg-neutral-500/10 text-neutral-600 dark:text-neutral-400", bar: "bg-neutral-400" },
	batal: { pill: "bg-neutral-500/10 text-neutral-500 dark:text-neutral-500 line-through", bar: "bg-neutral-300" },
};

export const LABEL_KEGIATAN: Record<Kegiatan, string> = { pembinaan: "Pembinaan", rapat: "Rapat", kunjungan: "Kunjungan" };

export const LABEL_PERIODE: Record<Periode, string> = {
	"30h": "30 hari",
	"3b": "3 bulan",
	"6b": "6 bulan",
	"12b": "12 bulan",
	semua: "Semua",
};

/** Status satu baris. Kunjungan memakai status kunjungan, bukan kehadiran. */
export function statusRiwayat(r: RiwayatAbsensi): StatusRiwayat {
	if (r.kegiatan === "kunjungan") {
		if (r.hadir) return "terlaksana";
		return r.alasan === "batal" ? "batal" : "dijadwalkan";
	}
	if (r.kehadiran === "hadir" || r.kehadiran === "izin" || r.kehadiran === "sakit" || r.kehadiran === "alfa") return r.kehadiran;
	return r.hadir ? "hadir" : "alfa";
}

export function isAbsensi(r: RiwayatAbsensi): boolean {
	return r.kegiatan === "pembinaan" || r.kegiatan === "rapat";
}

function isoLokal(d: Date): string {
	const y = d.getFullYear();
	const m = String(d.getMonth() + 1).padStart(2, "0");
	const day = String(d.getDate()).padStart(2, "0");
	return `${y}-${m}-${day}`;
}

/** Rentang [mulai, sampai] inklusif (YYYY-MM-DD). `null` = tanpa batas. */
export interface Rentang {
	mulai: string | null;
	sampai: string;
}

export function rentangPeriode(periode: Periode, today: Date): Rentang {
	const sampai = isoLokal(today);
	const d = new Date(today);
	switch (periode) {
		case "30h":
			d.setDate(d.getDate() - 29);
			break;
		case "3b":
			d.setMonth(d.getMonth() - 3);
			d.setDate(d.getDate() + 1);
			break;
		case "6b":
			d.setMonth(d.getMonth() - 6);
			d.setDate(d.getDate() + 1);
			break;
		case "12b":
			d.setFullYear(d.getFullYear() - 1);
			d.setDate(d.getDate() + 1);
			break;
		case "semua":
			return { mulai: null, sampai };
	}
	return { mulai: isoLokal(d), sampai };
}

/** Rentang sebelumnya dengan panjang yang sama, untuk delta KPI. */
export function rentangSebelumnya(r: Rentang): Rentang | null {
	if (!r.mulai) return null;
	const mulai = new Date(`${r.mulai}T00:00:00`);
	const sampai = new Date(`${r.sampai}T00:00:00`);
	const hari = Math.round((sampai.getTime() - mulai.getTime()) / 86_400_000) + 1;
	const prevSampai = new Date(mulai);
	prevSampai.setDate(prevSampai.getDate() - 1);
	const prevMulai = new Date(prevSampai);
	prevMulai.setDate(prevMulai.getDate() - (hari - 1));
	return { mulai: isoLokal(prevMulai), sampai: isoLokal(prevSampai) };
}

export function dalamRentang(tanggal: string, r: Rentang): boolean {
	const t = tanggal.slice(0, 10);
	return (!r.mulai || t >= r.mulai) && t <= r.sampai;
}

export interface RekapKehadiran {
	hadir: number;
	izin: number;
	sakit: number;
	alfa: number;
	total: number;
	/** Persentase hadir; null bila belum ada catatan. */
	persen: number | null;
}

export function rekapKehadiran(rows: RiwayatAbsensi[]): RekapKehadiran {
	const rekap: RekapKehadiran = { hadir: 0, izin: 0, sakit: 0, alfa: 0, total: 0, persen: null };
	for (const r of rows) {
		if (!isAbsensi(r)) continue;
		rekap[statusRiwayat(r) as StatusKehadiran]++;
		rekap.total++;
	}
	rekap.persen = rekap.total > 0 ? (rekap.hadir / rekap.total) * 100 : null;
	return rekap;
}

export interface BucketTren {
	key: string;
	label: string;
	rekap: RekapKehadiran;
}

/** Kelompokkan absensi per pekan (rentang ≤ ~3 bulan) atau per bulan; maks. 14 bucket terbaru. */
export function trenKehadiran(rows: RiwayatAbsensi[], rentang: Rentang): BucketTren[] {
	const mingguan = rentang.mulai !== null && hariRentang(rentang) <= 93;
	const groups = new Map<string, RiwayatAbsensi[]>();
	for (const r of rows) {
		if (!isAbsensi(r)) continue;
		const key = mingguan ? awalPekan(r.tanggal) : r.tanggal.slice(0, 7);
		const list = groups.get(key);
		if (list) list.push(r);
		else groups.set(key, [r]);
	}
	return [...groups.keys()]
		.sort()
		.slice(-14)
		.map((key) => ({ key, label: mingguan ? labelPekan(key) : labelBulan(key), rekap: rekapKehadiran(groups.get(key) ?? []) }));
}

function hariRentang(r: Rentang): number {
	if (!r.mulai) return Infinity;
	return Math.round((new Date(`${r.sampai}T00:00:00`).getTime() - new Date(`${r.mulai}T00:00:00`).getTime()) / 86_400_000) + 1;
}

function awalPekan(tanggal: string): string {
	const d = new Date(`${tanggal.slice(0, 10)}T00:00:00`);
	const geser = (d.getDay() + 6) % 7; // Senin = 0
	d.setDate(d.getDate() - geser);
	return isoLokal(d);
}

function labelPekan(iso: string): string {
	return new Date(`${iso}T00:00:00`).toLocaleDateString("id-ID", { day: "numeric", month: "short" });
}

export function labelBulan(ym: string): string {
	const d = new Date(`${ym}-01T00:00:00`);
	return Number.isNaN(d.getTime()) ? ym : d.toLocaleDateString("id-ID", { month: "short", year: "2-digit" });
}

export interface RekapGuru {
	guru_id: number;
	guru_nama: string;
	rekap: RekapKehadiran;
}

/** Guru dengan alfa terbanyak (lalu persentase hadir terendah). */
export function guruAlfaTerbanyak(rows: RiwayatAbsensi[], batas = 8): RekapGuru[] {
	const groups = new Map<number, { nama: string; rows: RiwayatAbsensi[] }>();
	for (const r of rows) {
		if (!isAbsensi(r)) continue;
		const g = groups.get(r.guru_id);
		if (g) g.rows.push(r);
		else groups.set(r.guru_id, { nama: r.guru_nama, rows: [r] });
	}
	return [...groups.entries()]
		.map(([guru_id, g]) => ({ guru_id, guru_nama: g.nama, rekap: rekapKehadiran(g.rows) }))
		.filter((g) => g.rekap.alfa > 0)
		.sort((a, b) => b.rekap.alfa - a.rekap.alfa || (a.rekap.persen ?? 0) - (b.rekap.persen ?? 0) || a.guru_nama.localeCompare(b.guru_nama))
		.slice(0, batas);
}

/** Jumlah sesi (pembinaan/rapat) unik yang punya catatan absen. */
export function jumlahSesi(rows: RiwayatAbsensi[]): number {
	const set = new Set<string>();
	for (const r of rows) if (isAbsensi(r)) set.add(`${r.kegiatan}-${r.kegiatan_id}`);
	return set.size;
}

export function formatTanggal(iso: string): string {
	if (!iso) return "–";
	const d = new Date(`${iso.slice(0, 10)}T00:00:00`);
	return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
}

export function formatPersen(v: number | null): string {
	return v === null ? "–" : `${v.toLocaleString("id-ID", { maximumFractionDigits: 1 })}%`;
}

export function formatAngka(n: number): string {
	return n.toLocaleString("id-ID");
}
