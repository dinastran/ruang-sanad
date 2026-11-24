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
// Event IDs already fired on this page. The same event_id is one event by
// definition, so a repeat (e.g. a component effect re-running after Inertia
// re-sends the page props) must never reach Meta twice.
const terkirim = new Set<string>();

export interface PixelEvent {
	name: string;
	id?: string;
	value?: number;
	currency?: string;
	custom?: boolean;
}

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

function kirim(method: "track" | "trackCustom", name: string, params: Record<string, unknown>, eventID?: string) {
	if (!window.fbq || !aktif.size) return;
	if (eventID) {
		if (terkirim.has(eventID)) return;
		terkirim.add(eventID);
	}
	try {
		window.fbq(method, name, params, eventID ? { eventID } : undefined);
	} catch {
		/* tracking tidak boleh mengganggu pendaftaran */
	}
}

/** Standard event; fired at most once per eventID. */
export function trackPixel(name: string, params: Record<string, unknown> = {}, eventID?: string) {
	kirim("track", name, params, eventID);
}

/** Custom (non-standard) event; fired at most once per eventID. */
export function trackPixelCustom(name: string, params: Record<string, unknown> = {}, eventID?: string) {
	kirim("trackCustom", name, params, eventID);
}

/** Fire events configured on the server (standard or custom). */
export function firePixelEvents(events: PixelEvent[] | undefined | null, params: Record<string, unknown> = {}) {
	for (const ev of events ?? []) {
		const p: Record<string, unknown> = { ...params };
		if (ev.currency) {
			p.value = ev.value ?? 0;
			p.currency = ev.currency;
		}
		kirim(ev.custom ? "trackCustom" : "track", ev.name, p, ev.id || undefined);
	}
}
