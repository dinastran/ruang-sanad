// IndexedDB untuk tasbih offline: "antrean" (sesi tertutup menunggu sync) dan
// "aktif" (sesi/paket yang sedang berjalan, dipulihkan setelah reload).
import type { SesiTertutup, StateAktif } from "./types";

// Satu database per user: perangkat bersama tidak boleh mencampur antrean
// atau sesi aktif antar akun (review #1).
const DB_PREFIX = "ruang-sanad-tasbih-u";
const DB_VERSION = 1;
let pemilik: number | null = null;
const ANTREAN = "antrean";
const AKTIF = "aktif";
const AKTIF_KEY = "current";

let dbPromise: Promise<IDBDatabase> | null = null;

/** Wajib dipanggil (dengan id user login) sebelum memakai fungsi lain di modul ini. */
export function setPemilik(userID: number): void {
	if (pemilik === userID) return;
	pemilik = userID;
	dbPromise = null;
}

// iPhone (WebKit): indexedDB.open() kadang tidak pernah merespons di aplikasi
// terpasang. Dua pengaman: "bangunkan" IndexedDB lewat databases() (pola
// safari-14-idb-fix) dan batas waktu; setelah macet sekali, operasi berikutnya
// langsung gagal agar pemanggil memakai cadangan memori tanpa menunggu.
const BATAS_BUKA_MS = 2500;
let idbMacet = false;

function bangunkanSafari(): Promise<void> {
	const idb = indexedDB as IDBFactory & { databases?: () => Promise<unknown> };
	if (typeof idb.databases !== "function") return Promise.resolve();
	return new Promise((resolve) => {
		let beres = false;
		let n = 0;
		const akhiri = () => {
			if (!beres) {
				beres = true;
				resolve();
			}
		};
		// panggil databases() tiap 100 ms sampai ada yang merespons (maks ~1 detik)
		const coba = () => {
			if (beres) return;
			n++;
			idb.databases!().then(akhiri, akhiri);
			setTimeout(n < 10 ? coba : akhiri, 100);
		};
		coba();
	});
}

function bukaDB(): Promise<IDBDatabase> {
	if (dbPromise) return dbPromise;
	dbPromise = new Promise((resolve, reject) => {
		if (!pemilik) {
			reject(new Error("pemilik antrean tasbih belum diset"));
			return;
		}
		if (idbMacet || typeof indexedDB === "undefined") {
			reject(new Error("IndexedDB tidak tersedia"));
			return;
		}
		let selesai = false;
		const batas = setTimeout(() => {
			if (selesai) return;
			selesai = true;
			idbMacet = true;
			reject(new Error("IndexedDB tidak merespons"));
		}, BATAS_BUKA_MS);
		bangunkanSafari().then(() => {
			if (selesai) return;
			let req: IDBOpenDBRequest;
			try {
				req = indexedDB.open(DB_PREFIX + pemilik, DB_VERSION);
			} catch (e) {
				selesai = true;
				clearTimeout(batas);
				reject(e);
				return;
			}
			req.onupgradeneeded = () => {
				const db = req.result;
				if (!db.objectStoreNames.contains(ANTREAN)) db.createObjectStore(ANTREAN, { keyPath: "id" });
				if (!db.objectStoreNames.contains(AKTIF)) db.createObjectStore(AKTIF);
			};
			req.onsuccess = () => {
				if (selesai) {
					req.result.close();
					return;
				}
				selesai = true;
				clearTimeout(batas);
				resolve(req.result);
			};
			req.onerror = () => {
				if (selesai) return;
				selesai = true;
				clearTimeout(batas);
				reject(req.error);
			};
		});
	});
	dbPromise.catch(() => (dbPromise = null));
	return dbPromise;
}

function jalankan<T>(store: string | string[], mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest<T> | void): Promise<T | undefined> {
	return bukaDB().then(
		(db) =>
			new Promise<T | undefined>((resolve, reject) => {
				const tx = db.transaction(store, mode);
				const req = fn(tx.objectStore(Array.isArray(store) ? store[0] : store));
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

/** Melempar error bila IndexedDB tidak bisa dibaca (jangan pura-pura kosong). */
export async function antreanSemua(): Promise<SesiTertutup[]> {
	const out = await jalankan<SesiTertutup[]>(ANTREAN, "readonly", (st) => st.getAll());
	return (out ?? []).sort((a, b) => a.mulai_at.localeCompare(b.mulai_at));
}

/** Simpan sesi tertutup ke antrean DAN state aktif baru dalam SATU transaksi,
 *  sehingga tidak ada jendela di mana sesi hilang atau ID-nya dipakai ulang. */
export async function tutupDanSimpanAktif(sesi: SesiTertutup, aktif: StateAktif | null): Promise<void> {
	const db = await bukaDB();
	await new Promise<void>((resolve, reject) => {
		const tx = db.transaction([ANTREAN, AKTIF], "readwrite");
		tx.objectStore(ANTREAN).put(sesi);
		if (aktif) tx.objectStore(AKTIF).put(aktif, AKTIF_KEY);
		else tx.objectStore(AKTIF).delete(AKTIF_KEY);
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(tx.error);
		tx.onabort = () => reject(tx.error);
	});
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
