// Mesin sesi tasbih. Satu sesi = hitungan berturut-turut untuk satu dzikir.
// Mode "dzikir": sesi ditutup saat ganti/reset/selesai atau idle > 5 menit.
// Mode "paket": satu sesi per item; sesi item tetap terbuka (tersimpan di
// IndexedDB) sampai targetnya tercapai, sehingga putaran bisa dilanjutkan
// setelah reload/jeda. Putaran yang ditinggalkan > 12 jam ditutup saat pulih.
import { aktifAmbil, aktifSimpan, antreanTambah } from "./db";
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

	katalog: Katalog;
	private cb: CounterCallbacks;
	private timer: ReturnType<typeof setTimeout> | null = null;
	private catatanRun: { dzikir_id: number; jumlah: number }[] = [];
	private mulaiRun = 0;

	constructor(katalog: Katalog, cb: CounterCallbacks) {
		this.katalog = katalog;
		this.cb = cb;
	}

	get dzikir(): Dzikir | undefined {
		return this.state ? this.katalog.dzikir.find((d) => d.id === this.state!.dzikir_id) : undefined;
	}

	get paket(): Paket | undefined {
		const id = this.state?.paket_id;
		return id ? this.katalog.paket.find((p) => p.id === id) : undefined;
	}

	/** Pulihkan state dari IndexedDB; bila tidak ada, pilih dzikir pertama. */
	async pulihkan(): Promise<void> {
		const a = await aktifAmbil();
		const now = Date.now();
		if (a && this.katalog.dzikir.some((d) => d.id === a.dzikir_id)) {
			this.state = a;
			const umur = now - Date.parse(a.tap_terakhir_at);
			if (a.sesi_id && ((a.mode === "dzikir" && umur > IDLE_MS) || (a.mode === "paket" && umur > PAKET_KADALUARSA_MS))) {
				await this.tutupSesi();
				if (a.mode === "paket") this.mulaiPaket(a.paket_id!);
			}
			this.selesaiTarget = this.state?.mode === "dzikir" && (this.state?.jumlah ?? 0) >= (this.state?.target ?? Infinity);
			return;
		}
		if (this.katalog.dzikir.length > 0) this.pilihDzikir(this.katalog.dzikir[0].id);
	}

	pilihDzikir(id: number): void {
		const d = this.katalog.dzikir.find((x) => x.id === id);
		if (!d) return;
		void this.tutupSesi();
		this.selesaiTarget = false;
		this.state = this.kosong("dzikir", d.id, d.target, null, null, 0);
		this.simpan();
	}

	pilihPaket(id: number): void {
		void this.tutupSesi();
		this.mulaiPaket(id);
		this.simpan();
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
		return { mode, dzikir_id: dzikirID, paket_id: paketID, paket_run_id: runID, item_index: idx, sesi_id: null, jumlah: 0, target, mulai_at: now, tap_terakhir_at: now };
	}

	/** Ubah target dzikir aktif (mode dzikir) tanpa menutup sesi. */
	setTargetAktif(target: number): void {
		if (!this.state || this.state.mode !== "dzikir") return;
		this.state.target = target;
		this.selesaiTarget = this.state.jumlah >= target;
		this.simpan();
	}

	tap(): void {
		if (!this.state) return;
		const s = this.state;
		const now = new Date().toISOString();
		if (!s.sesi_id) {
			s.sesi_id = uuid();
			s.mulai_at = now;
			s.jumlah = 0;
			if (s.mode === "paket" && s.item_index === 0 && this.mulaiRun === 0) this.mulaiRun = Date.now();
		}
		s.jumlah += 1;
		s.tap_terakhir_at = now;

		if (s.mode === "paket") {
			if (s.jumlah >= s.target) this.selesaiItemPaket();
		} else if (s.jumlah === s.target) {
			this.selesaiTarget = true;
			this.cb.onTarget();
		}
		this.simpan();
	}

	private selesaiItemPaket(): void {
		const s = this.state!;
		const p = this.paket;
		this.catatanRun.push({ dzikir_id: s.dzikir_id, jumlah: s.jumlah });
		void this.tutupSesi();
		if (!p) return;
		const next = s.item_index + 1;
		if (next >= p.items.length) {
			this.ringkasanPaket = { paket: p, items: [...this.catatanRun], durasiDetik: this.mulaiRun ? Math.round((Date.now() - this.mulaiRun) / 1000) : 0 };
			this.cb.onPaketSelesai(p);
			this.mulaiPaket(p.id);
			return;
		}
		const it = p.items[next];
		this.state = this.kosong("paket", it.dzikir_id, it.target, p.id, s.paket_run_id, next);
		this.cb.onItemPaketSelesai(next);
	}

	undo(): void {
		const s = this.state;
		if (!s || !s.sesi_id || s.jumlah <= 0) return;
		s.jumlah -= 1;
		if (s.mode === "dzikir" && s.jumlah < s.target) this.selesaiTarget = false;
		this.simpan();
	}

	/** Reset: simpan sesi berjalan, mulai dari nol (paket: putaran baru). */
	reset(): void {
		const s = this.state;
		if (!s) return;
		void this.tutupSesi();
		if (s.mode === "paket" && s.paket_id) this.mulaiPaket(s.paket_id);
		else this.state = this.kosong("dzikir", s.dzikir_id, s.target, null, null, 0);
		this.selesaiTarget = false;
		this.simpan();
	}

	/** Selesai: sama dengan reset, dipakai tombol "Selesai". */
	selesai(): void {
		this.reset();
	}

	/** Mode dzikir: tutup sesi bila idle > 5 menit. */
	cekIdle(now: Date = new Date()): void {
		const s = this.state;
		if (!s || s.mode !== "dzikir" || !s.sesi_id) return;
		if (now.getTime() - Date.parse(s.tap_terakhir_at) > IDLE_MS) {
			void this.tutupSesi();
			this.state = this.kosong("dzikir", s.dzikir_id, s.target, null, null, 0);
			this.selesaiTarget = false;
			this.simpan();
		}
	}

	/** Pindahkan sesi terbuka (bila jumlah > 0) ke antrean sync. */
	private async tutupSesi(): Promise<void> {
		const s = this.state;
		if (!s || !s.sesi_id) return;
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
		if (sesi.jumlah <= 0) return;
		try {
			await antreanTambah(sesi);
		} catch (e) {
			console.warn("[tasbih] gagal menyimpan sesi ke antrean", e);
			return;
		}
		this.cb.onSesiTutup(sesi);
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
		aktifSimpan(snap).catch((e) => console.warn("[tasbih] gagal menyimpan state", e));
	}
}
