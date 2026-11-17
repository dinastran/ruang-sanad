/// <reference types="vite/client" />
// PWA aplikasi Tasbih: registrasi service worker, deteksi mode aplikasi,
// ajakan pasang (Android/iOS), dan pembaruan versi.

interface BeforeInstallPromptEvent extends Event {
	prompt(): Promise<void>;
	userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

function deteksiStandalone(): boolean {
	if (typeof window === "undefined") return false;
	return window.matchMedia?.("(display-mode: standalone)").matches || (navigator as unknown as { standalone?: boolean }).standalone === true;
}

function deteksiIOS(): boolean {
	if (typeof navigator === "undefined") return false;
	const ua = navigator.userAgent;
	const iDevice = /iPhone|iPad|iPod/.test(ua) || (ua.includes("Macintosh") && navigator.maxTouchPoints > 1);
	const safari = /Safari/.test(ua) && !/CriOS|FxiOS|EdgiOS|OPiOS/.test(ua);
	return iDevice && safari;
}

export const pwa = $state({
	standalone: deteksiStandalone(),
	ios: deteksiIOS(),
	bisaPasang: false, // beforeinstallprompt tersedia (Android/Chrome)
	terpasang: false,
	updateSiap: false,
});

let promptTertunda: BeforeInstallPromptEvent | null = null;
let registrasi: ServiceWorkerRegistration | null = null;
let userTerakhir: number | null = null;

/** Dipanggil sekali dari main.ts sebelum app dibuat (event install bisa datang awal). */
export function mulaiPWA(): void {
	if (typeof window === "undefined") return;

	window.addEventListener("beforeinstallprompt", (e) => {
		e.preventDefault();
		promptTertunda = e as BeforeInstallPromptEvent;
		pwa.bisaPasang = true;
	});
	window.addEventListener("appinstalled", () => {
		promptTertunda = null;
		pwa.bisaPasang = false;
		pwa.terpasang = true;
	});
	window.matchMedia?.("(display-mode: standalone)").addEventListener?.("change", (e) => (pwa.standalone = e.matches));

	if (!("serviceWorker" in navigator) || !import.meta.env.PROD) return;
	window.addEventListener("load", () => {
		navigator.serviceWorker
			.register("/sw.js", { scope: "/" })
			.then((reg) => {
				registrasi = reg;
				const cekMenunggu = () => {
					// ada SW baru menunggu & halaman sudah dikontrol SW lama = versi baru tersedia
					if (reg.waiting && navigator.serviceWorker.controller) pwa.updateSiap = true;
				};
				cekMenunggu();
				reg.addEventListener("updatefound", () => {
					reg.installing?.addEventListener("statechange", cekMenunggu);
				});
				if (userTerakhir !== null) kirimUser(userTerakhir);
			})
			.catch((e) => console.warn("[pwa] registrasi service worker gagal", e));
	});
}

/** Beri tahu SW user yang login (cache halaman dipisah per user). */
export function kirimUser(id: number | null | undefined): void {
	if (!id || typeof navigator === "undefined" || !("serviceWorker" in navigator)) return;
	userTerakhir = id;
	navigator.serviceWorker.ready
		.then((reg) => reg.active?.postMessage({ type: "SET_USER", id }))
		.catch(() => {});
}

/** Hapus halaman tersimpan (dipanggil sebelum logout). */
export async function hapusCacheHalaman(): Promise<void> {
	try {
		navigator.serviceWorker?.controller?.postMessage({ type: "CLEAR_PAGES" });
		if (typeof caches !== "undefined") {
			const keys = await caches.keys();
			await Promise.all(keys.filter((k) => k.startsWith("tasbih-pages-u")).map((k) => caches.delete(k)));
		}
	} catch {
		/* cache API tidak tersedia */
	}
}

/** Tampilkan dialog pasang bawaan browser (Android/Chrome). */
export async function pasangAplikasi(): Promise<boolean> {
	if (!promptTertunda) return false;
	const ev = promptTertunda;
	promptTertunda = null;
	pwa.bisaPasang = false;
	await ev.prompt();
	const pilihan = await ev.userChoice;
	return pilihan.outcome === "accepted";
}

/** Aktifkan SW versi baru lalu muat ulang. */
export function muatVersiBaru(): void {
	const waiting = registrasi?.waiting;
	if (!waiting) {
		window.location.reload();
		return;
	}
	navigator.serviceWorker.addEventListener("controllerchange", () => window.location.reload(), { once: true });
	waiting.postMessage({ type: "SKIP_WAITING" });
}

// ---- banner pasang: tidak tampil lagi 30 hari setelah ditutup ----
const KUNCI_TUTUP = "tasbih:install-tutup";
const TIGA_PULUH_HARI = 30 * 24 * 60 * 60 * 1000;

export function bannerDitutupBaruBaru(): boolean {
	try {
		const t = Number(localStorage.getItem(KUNCI_TUTUP) || 0);
		return t > 0 && Date.now() - t < TIGA_PULUH_HARI;
	} catch {
		return false;
	}
}

export function tutupBanner(): void {
	try {
		localStorage.setItem(KUNCI_TUTUP, String(Date.now()));
	} catch {
		/* storage diblokir */
	}
}
