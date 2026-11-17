// Umpan balik perangkat: getar, suara klik, layar tetap menyala.
// Preferensi per perangkat di localStorage (bukan data akun).

export interface PrefTasbih {
	getar: boolean;
	suara: boolean;
	layar: boolean;
}

const KEY = "tasbih:pref";
const DEFAULT: PrefTasbih = { getar: true, suara: false, layar: true };

export function bacaPref(): PrefTasbih {
	try {
		const raw = localStorage.getItem(KEY);
		return raw ? { ...DEFAULT, ...JSON.parse(raw) } : { ...DEFAULT };
	} catch {
		return { ...DEFAULT };
	}
}

export function simpanPref(p: PrefTasbih): void {
	try {
		localStorage.setItem(KEY, JSON.stringify(p));
	} catch {
		/* storage diblokir: abaikan */
	}
}

const POLA: Record<"tap" | "kelipatan" | "target" | "paket", number | number[]> = {
	tap: 8,
	kelipatan: [25, 40, 25],
	target: [60, 50, 60, 50, 120],
	paket: [80, 60, 80, 60, 200],
};

export function getar(pola: keyof typeof POLA, aktif = true): void {
	if (!aktif || typeof navigator === "undefined" || typeof navigator.vibrate !== "function") return;
	try {
		navigator.vibrate(POLA[pola]);
	} catch {
		/* iOS Safari tidak mendukung */
	}
}

let audio: AudioContext | null = null;
export function klik(nada: "tap" | "target" = "tap"): void {
	try {
		const Ctx = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
		if (!Ctx) return;
		audio ??= new Ctx();
		const t = audio.currentTime;
		const osc = audio.createOscillator();
		const gain = audio.createGain();
		osc.type = "sine";
		osc.frequency.setValueAtTime(nada === "tap" ? 880 : 660, t);
		gain.gain.setValueAtTime(0.0001, t);
		gain.gain.exponentialRampToValueAtTime(nada === "tap" ? 0.08 : 0.15, t + 0.005);
		gain.gain.exponentialRampToValueAtTime(0.0001, t + (nada === "tap" ? 0.06 : 0.35));
		osc.connect(gain).connect(audio.destination);
		osc.start(t);
		osc.stop(t + 0.4);
	} catch {
		/* audio tidak tersedia */
	}
}

let lock: { release(): Promise<void> } | null = null;
let ingin = false;

async function minta(): Promise<void> {
	const wl = (navigator as unknown as { wakeLock?: { request(t: "screen"): Promise<{ release(): Promise<void> }> } }).wakeLock;
	if (!wl || document.visibilityState !== "visible") return;
	try {
		lock = await wl.request("screen");
	} catch {
		lock = null;
	}
}

function onVisible() {
	if (ingin && document.visibilityState === "visible") minta();
}

export async function kunciLayar(aktif: boolean): Promise<void> {
	ingin = aktif;
	if (typeof document === "undefined") return;
	document.removeEventListener("visibilitychange", onVisible);
	if (aktif) {
		document.addEventListener("visibilitychange", onVisible);
		await minta();
	} else if (lock) {
		await lock.release().catch(() => {});
		lock = null;
	}
}
