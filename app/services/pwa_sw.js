/* Service worker aplikasi Tasbih Ruang Sanad.
 * Dibangkitkan server (app/services/pwa.go): versi & daftar precache diisi
 * dari manifest Vite sehingga berganti otomatis setiap deploy. */
const VERSION = "__VERSION__";
const PRECACHE = __PRECACHE__;

const STATIC_CACHE = "tasbih-static-" + VERSION;
const FONT_CACHE = "tasbih-font-v1";
const META_CACHE = "tasbih-meta";
const PAGE_PREFIX = "tasbih-pages-u";
const OFFLINE_URL = "/public/offline.html";
const HALAMAN = ["/app/tasbih", "/app/tasbih/dashboard"];
const USER_KEY = "/__sw/user";

// ---------- user aktif (cache halaman dipisah per user) ----------
async function bacaUser() {
	const res = await (await caches.open(META_CACHE)).match(USER_KEY);
	return res ? res.text() : "";
}

async function setUser(id) {
	id = String(id || "");
	const lama = await bacaUser();
	await (await caches.open(META_CACHE)).put(USER_KEY, new Response(id));
	if (lama !== id) await hapusHalaman(id); // perangkat bersama: buang halaman user lain
}

async function hapusHalaman(kecuali) {
	const keys = await caches.keys();
	await Promise.all(keys.filter((k) => k.startsWith(PAGE_PREFIX) && k !== PAGE_PREFIX + kecuali).map((k) => caches.delete(k)));
}

// ---------- lifecycle ----------
self.addEventListener("install", (event) => {
	event.waitUntil(
		caches.open(STATIC_CACHE).then((c) =>
			// satu aset gagal tidak boleh menggagalkan instalasi
			Promise.all(PRECACHE.map((u) => c.add(new Request(u, { cache: "reload" })).catch(() => {}))),
		),
	);
});

self.addEventListener("activate", (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((k) => k.startsWith("tasbih-static-") && k !== STATIC_CACHE).map((k) => caches.delete(k))))
			.then(() => self.clients.claim()),
	);
});

self.addEventListener("message", (event) => {
	const d = event.data || {};
	if (d.type === "SKIP_WAITING") self.skipWaiting();
	else if (d.type === "SET_USER") event.waitUntil(setUser(d.id));
	else if (d.type === "CLEAR_PAGES") event.waitUntil(hapusHalaman("\u0000"));
});

// ---------- strategi ----------
async function cacheDulu(request, cacheName) {
	const cache = await caches.open(cacheName);
	const hit = await cache.match(request, { ignoreSearch: false });
	if (hit) return hit;
	const res = await fetch(request);
	if (res.ok || res.type === "opaque") cache.put(request, res.clone()).catch(() => {});
	return res;
}

// Catatan: Cache API mengabaikan fragmen (#...), jadi pembeda HTML vs JSON
// Inertia harus lewat query string agar keduanya tidak saling menimpa.
function kunciHalaman(url, request) {
	return url.origin + url.pathname + (request.headers.get("X-Inertia") ? "?__sw=inertia" : "?__sw=html");
}

async function jaringanDulu(event, url) {
	const req = event.request;
	const user = await bacaUser();
	const kunci = kunciHalaman(url, req);
	const parsial = req.headers.has("X-Inertia-Partial-Data");
	try {
		const res = await fetch(req);
		if (user && res.ok && !res.redirected && res.type === "basic" && !parsial) {
			const salinan = res.clone();
			event.waitUntil(caches.open(PAGE_PREFIX + user).then((c) => c.put(kunci, salinan)));
		}
		return res;
	} catch (err) {
		if (user) {
			// respons Inertia membawa "Vary: X-Inertia"; kunci kita sudah memisahkan html/inertia
			const hit = await (await caches.open(PAGE_PREFIX + user)).match(kunci, { ignoreVary: true });
			if (hit) return hit;
		}
		if (req.mode === "navigate") {
			const off = await caches.match(OFFLINE_URL);
			if (off) return off;
		}
		throw err;
	}
}

self.addEventListener("fetch", (event) => {
	const req = event.request;
	const url = new URL(req.url);

	// logout: buang semua halaman tersimpan (privasi perangkat bersama)
	if (req.method === "POST" && url.origin === self.location.origin && url.pathname === "/logout") {
		event.waitUntil(hapusHalaman("\u0000"));
		return;
	}
	if (req.method !== "GET") return;

	if (url.hostname === "fonts.googleapis.com" || url.hostname === "fonts.gstatic.com") {
		event.respondWith(cacheDulu(req, FONT_CACHE));
		return;
	}
	if (url.origin !== self.location.origin) return;

	if (url.pathname.startsWith("/assets/") || url.pathname.startsWith("/public/")) {
		event.respondWith(cacheDulu(req, STATIC_CACHE));
		return;
	}
	if (HALAMAN.includes(url.pathname)) {
		event.respondWith(jaringanDulu(event, url));
		return;
	}
	if (req.mode === "navigate") {
		event.respondWith(fetch(req).catch(() => caches.match(OFFLINE_URL).then((r) => r || Response.error())));
	}
});
