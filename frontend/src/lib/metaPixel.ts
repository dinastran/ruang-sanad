// Meta Pixel for the public registration pages (/daftar/*).
//
// Loads fbevents.js once and fires events with an eventID; the server sends
// the same events through the Conversions API with the same event_id, so
// Meta counts each event once. Pixel IDs and event IDs come from the server.

type Fbq = ((...args: unknown[]) => void) & {
	callMethod?: (...args: unknown[]) => void;
	queue: unknown[];
	push: Fbq;
	loaded: boolean;
	version: string;
};

declare global {
	interface Window {
		fbq?: Fbq;
		_fbq?: Fbq;
	}
}

const aktif = new Set<string>();

function pasangSkrip() {
	if (window.fbq) return;
	const n = function (...args: unknown[]) {
		if (n.callMethod) n.callMethod(...args);
		else n.queue.push(args);
	} as Fbq;
	n.push = n;
	n.loaded = true;
	n.version = "2.0";
	n.queue = [];
	window.fbq = n;
	if (!window._fbq) window._fbq = n;
	const s = document.createElement("script");
	s.async = true;
	s.src = "https://connect.facebook.net/en_US/fbevents.js";
	document.head.appendChild(s);
}

/** Load the Pixel(s) and fire PageView. Safe to call more than once. */
export function initPixel(ids: string[] | undefined | null) {
	if (typeof window === "undefined" || !ids?.length) return false;
	try {
		pasangSkrip();
		let baru = false;
		for (const id of ids) {
			if (aktif.has(id)) continue;
			window.fbq!("init", id);
			aktif.add(id);
			baru = true;
		}
		if (baru) window.fbq!("track", "PageView");
		return true;
	} catch {
		return false;
	}
}

/** Advanced matching with already-hashed values (sha256 hex from the server). */
export function setPixelUser(user: Record<string, string> | undefined | null) {
	if (!window.fbq || !user || !Object.keys(user).length) return;
	try {
		for (const id of aktif) window.fbq("init", id, user);
	} catch {
		/* tracking tidak boleh mengganggu pendaftaran */
	}
}

export function trackPixel(name: string, params: Record<string, unknown> = {}, eventID?: string) {
	if (!window.fbq || !aktif.size) return;
	try {
		window.fbq("track", name, params, eventID ? { eventID } : undefined);
	} catch {
		/* abaikan */
	}
}

/** Custom (non-standard) browser-only event, e.g. form step reached. */
export function trackPixelCustom(name: string, params: Record<string, unknown> = {}) {
	if (!window.fbq || !aktif.size) return;
	try {
		window.fbq("trackCustom", name, params);
	} catch {
		/* abaikan */
	}
}
