// Mesin sesi tasbih. Satu sesi = hitungan berturut-turut untuk satu dzikir.
// - Mode "dzikir": sesi ditutup saat ganti/reset/selesai atau idle > 5 menit.
// - Mode "paket": idle > 5 menit menutup sesi item yang berjalan, tetapi posisi
//   putaran (item & hitungan terkumpul) tetap, sehingga bisa dilanjutkan; server
//   menjumlahkan semua sesi per item dalam satu putaran. Putaran yang
//   ditinggalkan > 12 jam dimulai ulang saat dipulihkan.
// Penutupan sesi + state aktif baru ditulis dalam satu transaksi IndexedDB.
import { aktifAmbil, aktifSimpan, tutupDanSimpanAktif } from "./db";
import { kirimSegera, simpanDiMemori } from "./sync.svelte";
import type { Dzikir, Paket, SesiTertutup, StateAktif } from "./types";
import { uuid } from "./types";

export const IDLE_MS = 5 * 60 * 1000;
export const PAKET_KADALUARSA_MS = 12 * 60 * 60 * 1000;

export interface Katalog {
	dzikir: Dzikir[];
	paket: Paket[];
}

export interface CounterCallbacks {
	onTarget(): void;
	onItemPaketSelesai(nextIndex: number): void;
	onPaketSelesai(paket: Paket): void;
	onSesiTutup(s: SesiTertutup): void;
}

export class TasbihCounter {
	state = $state<StateAktif | null>(null);
	/** true sejak target sesi dzikir tercapai (untuk tampilan). */
	selesaiTarget = $state(false);
	/** Ringkasan putaran paket terakhir yang selesai (untuk sheet ringkasan). */
	ringkasanPaket = $state<{ paket: Paket; items: { dzikir_id: number; jumlah: number }[]; durasiDetik: number } | null>(null);

	private getKatalog: () => Katalog;
	private cb: CounterCallbacks;
	private timer: ReturnType<typeof setTimeout> | null = null;
	private catatanRun: { dzikir_id: number; jumlah: number }[] = [];
	private mulaiRun = 0;

	constructor(katalog: () => Katalog, cb: CounterCallbacks) {
		this.getKatalog = katalog;
		this.cb = cb;
	}

	get katalog(): Katalog {
		return this.getKatalog();
	}

	get dzikir(): Dzikir | undefined {
		return this.state ? this.katalog.dzikir.find((d) => d.id === this.state!.dzikir_id) : undefined;
	}

	get paket(): Paket | undefined {
		const id = this.state?.paket_id;
		return id ? this.katalog.paket.find((p) => p.id === id) : undefined;
	}

	/** Hitungan yang ditampilkan = terkumpul dari sesi yang sudah ditutup sementara
	 *  (pindah tab/perangkat, atau jeda di paket) + sesi yang sedang berjalan. */
	get tampil(): number {
		const s = this.state;
		if (!s) return 0;
		return (s.item_terkumpul ?? 0) + s.jumlah;
	}

	/** Pulihkan state dari IndexedDB; bila tidak ada, pilih dzikir pertama. */
	/** true begitu user berinteraksi; state hasil pemulihan tidak boleh menimpanya. */
	private sudahBertindak = false;

	/** Siapkan state awal SEKARANG (counter langsung bisa dipakai), lalu pulihkan
	 *  state tersimpan dari IndexedDB bila ada — IndexedDB bisa lambat/macet di iPhone. */
	async pulihkan(): Promise<void> {
		if (!this.state) this.pilihPertama();
		const a = await aktifAmbil();
		if (!a) {
			if (!this.sudahBertindak) this.simpan();
			return;
		}
		if (this.sudahBertindak) {
			// user sudah mulai menghitung: jangan timpa, tapi simpan sesi lama yang masih terbuka
			const lama = this.sesiDari(a);
			if (lama) this.commit(lama);
			return;
		}
		this.state = a;
		const umur = Date.now() - Date.parse(a.tap_terakhir_at);
		const dikenal = this.katalog.dzikir.some((d) => d.id === a.dzikir_id);
		if (!dikenal) {
			// dzikir sudah diarsipkan: simpan sesinya, lalu pilih dzikir lain
			const sesi = this.lepasSesi();
			this.state = null;
			this.pilihPertama();
			this.commit(sesi);
			return;
		}
		if (a.mode === "paket" && umur > PAKET_KADALUARSA_MS) {
			const sesi = this.lepasSesi();
			this.mulaiPaket(a.paket_id!);
			this.commit(sesi);
		} else if (a.sesi_id && umur > IDLE_MS) {
			this.tutupKarenaIdle();
		}
		this.selesaiTarget = this.state?.mode === "dzikir" && this.tampil >= (this.state?.target ?? Infinity);
	}

	private sesiDari(a: StateAktif): SesiTertutup | null {
		if (!a.sesi_id || a.jumlah <= 0) return null;
		return {
			id: a.sesi_id,
			dzikir_id: a.dzikir_id,
			paket_id: a.mode === "paket" ? a.paket_id : null,
			paket_run_id: a.mode === "paket" ? a.paket_run_id : null,
			jumlah: a.jumlah,
			target: a.target,
			mulai_at: a.mulai_at,
			selesai_at: a.tap_terakhir_at,
		};
	}

	private pilihPertama(): void {
		const d = this.katalog.dzikir[0];
		if (d) this.state = this.kosong("dzikir", d.id, d.target, null, null, 0);
	}

	pilihDzikir(id: number): void {
		this.sudahBertindak = true;
		const d = this.katalog.dzikir.find((x) => x.id === id);
		if (!d) return;
		const sesi = this.lepasSesi();
		this.selesaiTarget = false;
		this.state = this.kosong("dzikir", d.id, d.target, null, null, 0);
		this.commit(sesi);
	}

	pilihPaket(id: number): void {
		this.sudahBertindak = true;
		const sesi = this.lepasSesi();
		this.mulaiPaket(id);
		this.commit(sesi);
	}

	private mulaiPaket(id: number): void {
		const p = this.katalog.paket.find((x) => x.id === id);
		if (!p || p.items.length === 0) return;
		this.selesaiTarget = false;
		this.catatanRun = [];
		this.mulaiRun = 0;
		const it = p.items[0];
		this.state = this.kosong("paket", it.dzikir_id, it.target, p.id, uuid(), 0);
	}

	private kosong(mode: StateAktif["mode"], dzikirID: number, target: number, paketID: number | null, runID: string | null, idx: number): StateAktif {
		const now = new Date().toISOString();
		return { mode, dzikir_id: dzikirID, paket_id: paketID, paket_run_id: runID, item_index: idx, item_terkumpul: 0, sesi_id: null, jumlah: 0, target, mulai_at: now, tap_terakhir_at: now };
	}

	/** Ubah target dzikir aktif (mode dzikir) tanpa menutup sesi. */
	setTargetAktif(target: number): void {
		if (!this.state || this.state.mode !== "dzikir") return;
		this.state.target = target;
		this.selesaiTarget = this.tampil >= target;
		this.simpan();
	}

	tap(): void {
		this.sudahBertindak = true;
		if (!this.state) return;
		const s = this.state;
		const now = new Date().toISOString();
		if (!s.sesi_id) {
			s.sesi_id = uuid();
			s.mulai_at = now;
			s.jumlah = 0;
			if (s.mode === "paket" && s.item_index === 0 && !s.item_terkumpul && this.mulaiRun === 0) this.mulaiRun = Date.now();
		}
		s.jumlah += 1;
		s.tap_terakhir_at = now;

		if (s.mode === "paket") {
			if (this.tampil >= s.target) {
				this.selesaiItemPaket();
				return;
			}
		} else if (this.tampil === s.target) {
			this.selesaiTarget = true;
			this.cb.onTarget();
		}
		this.simpan();
	}

	private selesaiItemPaket(): void {
		const s = this.state!;
		const p = this.paket;
		this.catatanRun.push({ dzikir_id: s.dzikir_id, jumlah: this.tampil });
		const sesi = this.lepasSesi();
		if (!p) {
			this.commit(sesi);
			return;
		}
		const next = s.item_index + 1;
		if (next >= p.items.length) {
			this.ringkasanPaket = { paket: p, items: [...this.catatanRun], durasiDetik: this.mulaiRun ? Math.round((Date.now() - this.mulaiRun) / 1000) : 0 };
			this.mulaiPaket(p.id);
			this.commit(sesi);
			this.cb.onPaketSelesai(p);
			return;
		}
		const it = p.items[next];
		this.state = this.kosong("paket", it.dzikir_id, it.target, p.id, s.paket_run_id, next);
		this.commit(sesi);
		this.cb.onItemPaketSelesai(next);
	}

	undo(): void {
		const s = this.state;
		if (!s || !s.sesi_id || s.jumlah <= 0) return;
		s.jumlah -= 1;
		if (s.mode === "dzikir" && this.tampil < s.target) this.selesaiTarget = false;
		this.simpan();
	}

	/** Reset: simpan sesi berjalan, mulai dari nol (paket: putaran baru). */
	reset(): void {
		this.sudahBertindak = true;
		const s = this.state;
		if (!s) return;
		const sesi = this.lepasSesi();
		if (s.mode === "paket" && s.paket_id) this.mulaiPaket(s.paket_id);
		else this.state = this.kosong("dzikir", s.dzikir_id, s.target, null, null, 0);
		this.selesaiTarget = false;
		this.commit(sesi);
	}

	/** Selesai: sama dengan reset, dipakai tombol "Simpan sesi". */
	selesai(): void {
		this.reset();
	}

	/** Tab disembunyikan/ditutup: kirim sesi berjalan SEKARANG agar perangkat lain
	 *  langsung melihatnya, tanpa mengubah angka di layar (bisa dilanjutkan). */
	simpanSementara(): void {
		const s = this.state;
		if (!s || !s.sesi_id) return;
		const sesi = this.lepasSesi();
		if (!sesi) return;
		s.item_terkumpul = (s.item_terkumpul ?? 0) + sesi.jumlah;
		s.jumlah = 0;
		this.commit(sesi, true);
	}

	/** Tutup sesi bila idle > 5 menit. Paket: posisi putaran dipertahankan. */
	cekIdle(now: Date = new Date()): void {
		const s = this.state;
		if (!s || !s.sesi_id) return;
		if (now.getTime() - Date.parse(s.tap_terakhir_at) > IDLE_MS) this.tutupKarenaIdle();
	}

	private tutupKarenaIdle(): void {
		const s = this.state;
		if (!s) return;
		const terkumpul = (s.item_terkumpul ?? 0) + s.jumlah;
		const sesi = this.lepasSesi();
		if (s.mode === "paket") {
			this.state = { ...this.kosong("paket", s.dzikir_id, s.target, s.paket_id, s.paket_run_id, s.item_index), item_terkumpul: terkumpul };
		} else {
			this.state = this.kosong("dzikir", s.dzikir_id, s.target, null, null, 0);
			this.selesaiTarget = false;
		}
		this.commit(sesi);
	}

	/** Lepas sesi terbuka dari state (sinkron). Mengembalikan sesi bila jumlah > 0. */
	private lepasSesi(): SesiTertutup | null {
		const s = this.state;
		if (!s || !s.sesi_id) return null;
		const sesi: SesiTertutup = {
			id: s.sesi_id,
			dzikir_id: s.dzikir_id,
			paket_id: s.mode === "paket" ? s.paket_id : null,
			paket_run_id: s.mode === "paket" ? s.paket_run_id : null,
			jumlah: s.jumlah,
			target: s.target,
			mulai_at: s.mulai_at,
			selesai_at: s.tap_terakhir_at,
		};
		s.sesi_id = null;
		return sesi.jumlah > 0 ? sesi : null;
	}

	/** Tulis sesi tertutup + state aktif saat ini secara atomik; cadangan memori bila gagal. */
	private commit(sesi: SesiTertutup | null, segera = false): void {
		if (!sesi) {
			this.simpan();
			return;
		}
		// Halaman bisa segera mati: kirim langsung (keepalive) di samping antrean IndexedDB.
		// Aman dobel karena server idempoten per ID sesi.
		if (segera) kirimSegera(sesi);
		if (this.timer) clearTimeout(this.timer);
		this.timer = null;
		const snap = this.state ? ($state.snapshot(this.state) as StateAktif) : null;
		tutupDanSimpanAktif(sesi, snap)
			.catch((e) => {
				console.warn("[tasbih] IndexedDB gagal, sesi disimpan di memori", e);
				simpanDiMemori(sesi);
			})
			.finally(() => this.cb.onSesiTutup(sesi));
	}

	/** Debounce 300ms penyimpanan state aktif. */
	simpan(): void {
		if (this.timer) clearTimeout(this.timer);
		this.timer = setTimeout(() => this.flush(), 300);
	}

	flush(): void {
		if (this.timer) clearTimeout(this.timer);
		this.timer = null;
		const snap = this.state ? ($state.snapshot(this.state) as StateAktif) : null;
		aktifSimpan(snap).catch(() => {
			/* IndexedDB tidak tersedia: state aktif hanya di memori */
		});
	}
}
