// Kalimat insight dashboard tasbih: aturan sederhana atas data yang sudah ada
// (per_jam & heatmap). Bukan AI. Mengembalikan null bila data terlalu tipis
// atau polanya tidak cukup jelas — lebih baik diam daripada menyimpulkan keliru.
import type { TitikHari } from "./types";

export const MIN_HARI_AKTIF = 14;
const JENDELA_HARI = 84; // 12 pekan terakhir untuk pola hari

const WAKTU: { label: string; jam: (h: number) => boolean }[] = [
	{ label: "Subuh", jam: (h) => h >= 3 && h < 6 },
	{ label: "pagi", jam: (h) => h >= 6 && h < 11 },
	{ label: "siang", jam: (h) => h >= 11 && h < 15 },
	{ label: "Asar", jam: (h) => h >= 15 && h < 18 },
	{ label: "Maghrib–Isya", jam: (h) => h >= 18 && h < 21 },
	{ label: "malam", jam: (h) => h >= 21 || h < 3 },
];

const HARI = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];

function kalimatWaktu(perJam: number[]): string | null {
	const total = perJam.reduce((a, v) => a + v, 0);
	if (total <= 0) return null;
	const porsi = WAKTU.map((w) => ({ label: w.label, p: perJam.reduce((a, v, h) => a + (w.jam(h) ? v : 0), 0) / total })).sort((a, b) => b.p - a.p);
	const [a, b] = porsi;
	if (a.p + b.p >= 0.6 && b.p >= 0.2) return `Sebagian besar dzikir Anda terjadi pada waktu ${a.label} dan ${b.label}.`;
	if (a.p >= 0.4) return `Dzikir Anda paling banyak pada waktu ${a.label}.`;
	return null;
}

function kalimatHari(heatmap: TitikHari[]): string | null {
	const jendela = heatmap.slice(-JENDELA_HARI);
	if (jendela.length < 28) return null;
	const aktif = Array(7).fill(0);
	const ada = Array(7).fill(0);
	for (const t of jendela) {
		const [y, m, d] = t.tanggal.split("-").map(Number);
		const wd = new Date(y, m - 1, d).getDay();
		ada[wd]++;
		if (t.total > 0) aktif[wd]++;
	}
	const rate = aktif.map((n, i) => (ada[i] ? n / ada[i] : 1));
	const min = Math.min(...rate), max = Math.max(...rate);
	if (min > 0.5 || max - min < 0.25) return null;
	return `Hari ${HARI[rate.indexOf(min)]} paling sering terlewat.`;
}

export function insightDzikir(perJam: number[], heatmap: TitikHari[]): string | null {
	const hariAktif = heatmap.filter((t) => t.total > 0).length;
	if (hariAktif < MIN_HARI_AKTIF) return null;
	const k = [kalimatWaktu(perJam), kalimatHari(heatmap)].filter(Boolean);
	return k.length ? k.join(" ") : null;
}
