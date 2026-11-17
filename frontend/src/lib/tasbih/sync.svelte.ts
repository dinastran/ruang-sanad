// Antrean sync: kirim sesi tertutup ke server dalam batch (≤100), idempoten
// di server. Hanya ID yang dikonfirmasi (diterima/ditolak) yang dihapus.
import { getCSRFToken } from "@lib/utils/csrf";
import { antreanHapus, antreanSemua } from "./db";
import type { SesiTertutup, SyncHasil } from "./types";

export type StatusSync = "tersimpan" | "menunggu" | "offline" | "menyinkron" | "gagal";

/** Cadangan di memori bila IndexedDB tidak bisa ditulis (review #2).
 *  Tetap dikirim ke server; hilang hanya bila tab ditutup sebelum terkirim. */
const antreanMemori: SesiTertutup[] = [];
let idbGagal = false;

export function simpanDiMemori(s: SesiTertutup): void {
	idbGagal = true;
	if (!antreanMemori.some((x) => x.id === s.id)) antreanMemori.push(s);
}

async function semuaAntrean(): Promise<SesiTertutup[]> {
	let disk: SesiTertutup[] = [];
	try {
		disk = await antreanSemua();
	} catch {
		idbGagal = true;
	}
	const ids = new Set(disk.map((x) => x.id));
	return [...disk, ...antreanMemori.filter((x) => !ids.has(x.id))];
}

async function hapusAntrean(ids: string[]): Promise<void> {
	const set = new Set(ids);
	for (let i = antreanMemori.length - 1; i >= 0; i--) if (set.has(antreanMemori[i].id)) antreanMemori.splice(i, 1);
	try {
		await antreanHapus(ids);
	} catch {
		idbGagal = true;
	}
}

export const syncState = $state<{ status: StatusSync; pending: number }>({ status: "tersimpan", pending: 0 });

const BATCH = 100;
const BACKOFF_MIN = 5_000;
const BACKOFF_MAX = 300_000;
let backoff = 0;
let tungguSampai = 0;
let berjalan: Promise<SyncHasil | null> | null = null;

function online(): boolean {
	return typeof navigator === "undefined" || navigator.onLine !== false;
}

export async function hitungPending(): Promise<number> {
	const n = (await semuaAntrean()).length;
	syncState.pending = n;
	if (syncState.status !== "menyinkron") {
		syncState.status = idbGagal ? "gagal" : n === 0 ? "tersimpan" : online() ? "menunggu" : "offline";
	}
	return n;
}

/** Kirim seluruh antrean. Mengembalikan hasil batch terakhir yang berhasil (atau null). */
export function syncSekarang(paksa = false): Promise<SyncHasil | null> {
	if (berjalan) return berjalan;
	berjalan = jalankanSync(paksa).finally(() => (berjalan = null));
	return berjalan;
}

async function jalankanSync(paksa: boolean): Promise<SyncHasil | null> {
	if (!paksa && Date.now() < tungguSampai) {
		await hitungPending();
		return null;
	}
	if (!online()) {
		await hitungPending();
		return null;
	}
	let terakhir: SyncHasil | null = null;
	const badge: SyncHasil["badge_baru"] = [];
	try {
		for (;;) {
			const antrean = await semuaAntrean();
			syncState.pending = antrean.length;
			if (antrean.length === 0) break;
			syncState.status = "menyinkron";
			const batch = antrean.slice(0, BATCH);
			const res = await fetch("/app/tasbih/sync", {
				method: "POST",
				credentials: "same-origin",
				headers: { "Content-Type": "application/json", Accept: "application/json", "X-XSRF-TOKEN": getCSRFToken() },
				body: JSON.stringify({ sesi: batch }),
			});
			if (!res.ok) throw new Error(`sync gagal: ${res.status}`);
			const hasil = (await res.json()) as SyncHasil;
			for (const d of hasil.ditolak ?? []) console.warn("[tasbih] sesi ditolak server", d.id, d.alasan);
			const selesai = [...(hasil.diterima ?? []), ...(hasil.ditolak ?? []).map((d) => d.id)];
			if (selesai.length === 0) throw new Error("sync tanpa konfirmasi");
			await hapusAntrean(selesai);
			badge.push(...(hasil.badge_baru ?? []));
			terakhir = hasil;
		}
		backoff = 0;
		tungguSampai = 0;
	} catch (e) {
		console.warn("[tasbih]", e);
		backoff = backoff === 0 ? BACKOFF_MIN : Math.min(backoff * 2, BACKOFF_MAX);
		tungguSampai = Date.now() + backoff;
	} finally {
		syncState.status = "tersimpan";
		await hitungPending();
	}
	if (terakhir) terakhir = { ...terakhir, badge_baru: badge };
	return terakhir;
}

/** Sync otomatis: saat mulai, saat kembali online, dan tiap 60 detik selama ada antrean. */
export function mulaiAutoSync(onHasil: (h: SyncHasil) => void): () => void {
	const jalan = (paksa = false) =>
		syncSekarang(paksa).then((h) => {
			if (h) onHasil(h);
		});
	const onOnline = () => jalan(true);
	const onOffline = () => hitungPending();
	window.addEventListener("online", onOnline);
	window.addEventListener("offline", onOffline);
	const timer = window.setInterval(() => {
		if (syncState.pending > 0) jalan();
	}, 60_000);
	jalan(true);
	return () => {
		window.removeEventListener("online", onOnline);
		window.removeEventListener("offline", onOffline);
		window.clearInterval(timer);
	};
}
