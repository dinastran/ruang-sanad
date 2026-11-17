// IndexedDB untuk tasbih offline: "antrean" (sesi tertutup menunggu sync) dan
// "aktif" (sesi/paket yang sedang berjalan, dipulihkan setelah reload).
import type { SesiTertutup, StateAktif } from "./types";

const DB_NAME = "ruang-sanad-tasbih";
const DB_VERSION = 1;
const ANTREAN = "antrean";
const AKTIF = "aktif";
const AKTIF_KEY = "current";

let dbPromise: Promise<IDBDatabase> | null = null;

function bukaDB(): Promise<IDBDatabase> {
	if (dbPromise) return dbPromise;
	dbPromise = new Promise((resolve, reject) => {
		if (typeof indexedDB === "undefined") {
			reject(new Error("IndexedDB tidak tersedia"));
			return;
		}
		const req = indexedDB.open(DB_NAME, DB_VERSION);
		req.onupgradeneeded = () => {
			const db = req.result;
			if (!db.objectStoreNames.contains(ANTREAN)) db.createObjectStore(ANTREAN, { keyPath: "id" });
			if (!db.objectStoreNames.contains(AKTIF)) db.createObjectStore(AKTIF);
		};
		req.onsuccess = () => resolve(req.result);
		req.onerror = () => reject(req.error);
	});
	dbPromise.catch(() => (dbPromise = null));
	return dbPromise;
}

function jalankan<T>(store: string, mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest<T> | void): Promise<T | undefined> {
	return bukaDB().then(
		(db) =>
			new Promise<T | undefined>((resolve, reject) => {
				const tx = db.transaction(store, mode);
				const req = fn(tx.objectStore(store));
				let hasil: T | undefined;
				if (req) req.onsuccess = () => (hasil = req.result);
				tx.oncomplete = () => resolve(hasil);
				tx.onerror = () => reject(tx.error);
				tx.onabort = () => reject(tx.error);
			}),
	);
}

export async function antreanTambah(s: SesiTertutup): Promise<void> {
	await jalankan(ANTREAN, "readwrite", (st) => st.put(s));
}

export async function antreanSemua(): Promise<SesiTertutup[]> {
	try {
		const out = await jalankan<SesiTertutup[]>(ANTREAN, "readonly", (st) => st.getAll());
		return (out ?? []).sort((a, b) => a.mulai_at.localeCompare(b.mulai_at));
	} catch {
		return [];
	}
}

export async function antreanHapus(ids: string[]): Promise<void> {
	if (ids.length === 0) return;
	await jalankan(ANTREAN, "readwrite", (st) => {
		for (const id of ids) st.delete(id);
	});
}

export async function aktifSimpan(a: StateAktif | null): Promise<void> {
	await jalankan(AKTIF, "readwrite", (st) => {
		if (a) st.put(a, AKTIF_KEY);
		else st.delete(AKTIF_KEY);
	});
}

export async function aktifAmbil(): Promise<StateAktif | null> {
	try {
		return (await jalankan<StateAktif>(AKTIF, "readonly", (st) => st.get(AKTIF_KEY))) ?? null;
	} catch {
		return null;
	}
}
