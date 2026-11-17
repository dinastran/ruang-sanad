// Tipe tasbih — cermin JSON dari Go (app/models/tasbih.go).

export interface Dzikir {
	id: number;
	custom: boolean;
	arab: string;
	latin: string;
	arti: string;
	rujukan: string;
	target: number;
	target_default: number;
}

export interface PaketItem {
	dzikir_id: number;
	target: number;
}

export interface Paket {
	id: number;
	slug: string;
	nama: string;
	deskripsi: string;
	rujukan: string;
	items: PaketItem[];
}

export interface SesiTertutup {
	id: string;
	dzikir_id: number;
	paket_id: number | null;
	paket_run_id: string | null;
	jumlah: number;
	target: number;
	mulai_at: string;
	selesai_at: string;
}

export interface StateAktif {
	mode: "dzikir" | "paket";
	dzikir_id: number;
	paket_id: number | null;
	paket_run_id: string | null;
	item_index: number;
	item_terkumpul?: number; // paket: hitungan item ini dari sesi sebelumnya (setelah jeda)
	sesi_id: string | null; // null = belum ada sesi terbuka
	jumlah: number;
	target: number;
	mulai_at: string;
	tap_terakhir_at: string;
}

export interface Badge {
	kode: string;
	nama: string;
	syarat: string;
	ikon: string;
	diraih_at: string | null;
}

export interface HariIni {
	tanggal: string;
	total: number;
	target: number;
}

export interface Streak {
	saat: number;
	terpanjang: number;
	hari_ini_tercapai: boolean;
}

export interface Level {
	level: number;
	total: number;
	ambang_ini: number;
	ambang_berikut: number;
}

export interface SyncHasil {
	diterima: string[];
	ditolak: { id: string; alasan: string }[];
	badge_baru: Badge[];
	hari_ini: HariIni;
}

export interface TitikHari {
	tanggal: string;
	total: number;
}

export interface DashboardData {
	rentang: number;
	hari_ini: HariIni;
	streak: Streak;
	level: Level;
	rata_rata: number;
	harian: TitikHari[];
	total_rentang: number;
	total_rentang_sebelum: number;
	heatmap: TitikHari[];
	komposisi: { dzikir_id: number; latin: string; total: number }[];
	per_jam: number[];
	paket: { paket_id: number; nama: string; selesai: number }[];
}

export interface RiwayatItem {
	id: string;
	tanggal: string;
	jam: string;
	dzikir_latin: string;
	jumlah: number;
	target: number;
	durasi_detik: number;
	paket_nama: string;
}

/** Tanggal WIB (YYYY-MM-DD) dari ISO string — dipakai untuk menggabung antrean lokal. */
export function tanggalWIB(iso: string | Date): string {
	const d = typeof iso === "string" ? new Date(iso) : iso;
	return new Date(d.getTime() + 7 * 3600 * 1000).toISOString().slice(0, 10);
}

export function uuid(): string {
	if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") return crypto.randomUUID();
	const b = new Uint8Array(16);
	if (typeof crypto !== "undefined" && crypto.getRandomValues) crypto.getRandomValues(b);
	else for (let i = 0; i < 16; i++) b[i] = Math.floor(Math.random() * 256);
	b[6] = (b[6] & 0x0f) | 0x40;
	b[8] = (b[8] & 0x3f) | 0x80;
	const h = Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
	return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}
