<script lang="ts">
	import { onMount } from "svelte";
	import { fade, scale } from "svelte/transition";
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { BarChart3, ChevronDown, Flame, Maximize2, Minimize2, RotateCcw, Settings2, Undo2, CheckCheck, Pencil } from "lucide-svelte";
	import ProgressRing from "@components/tasbih/ProgressRing.svelte";
	import BeadStrip from "@components/tasbih/BeadStrip.svelte";
	import SyncIndicator from "@components/tasbih/SyncIndicator.svelte";
	import BadgeToast from "@components/tasbih/BadgeToast.svelte";
	import DzikirPicker from "@components/tasbih/DzikirPicker.svelte";
	import DzikirFormSheet from "@components/tasbih/DzikirFormSheet.svelte";
	import Sheet from "@components/tasbih/Sheet.svelte";
	import InstallBanner from "@components/pwa/InstallBanner.svelte";
	import KomposisiBar from "@components/tasbih/KomposisiBar.svelte";
	import { TasbihCounter } from "@lib/tasbih/counter.svelte";
	import { antreanSemua, setPemilik } from "@lib/tasbih/db";
	import { mulaiAutoSync, syncSekarang } from "@lib/tasbih/sync.svelte";
	import { bacaPref, getar, klik, kunciLayar, simpanPref } from "@lib/tasbih/feedback";
	import { tanggalWIB } from "@lib/tasbih/types";
	import type { Badge, Dzikir, HariIni, Level, Paket, Streak, SyncHasil } from "@lib/tasbih/types";
	import type { User } from "@lib/types";

	interface Props {
		user?: User;
		dzikir: Dzikir[];
		paket: Paket[];
		hari_ini: HariIni;
		target_harian: number;
		streak: Streak;
		level: Level;
		badge_baru: Badge[];
		komposisi_hari_ini?: Komposisi[];
	}
	type Komposisi = { dzikir_id: number; latin: string; total: number };
	let { user, dzikir, paket, hari_ini, target_harian, streak, level, badge_baru, komposisi_hari_ini }: Props = $props();

	// Antrean offline dipisah per user (perangkat bersama).
	if (user?.id) setPemilik(user.id);

	let siap = $state(false);
	let pref = $state(bacaPref());
	let pickerOpen = $state(false);
	let formOpen = $state(false);
	let formEdit = $state<Dzikir | null>(null);
	let pengaturanOpen = $state(false);
	let targetOpen = $state(false);
	let ringkasanOpen = $state(false);
	let fokus = $state(false);
	let rayakanKey = $state(0);
	let pengumuman = $state("");
	let badges = $state<Badge[]>([...(badge_baru ?? [])]);
	let serverHariIni = $state({ tanggal: hari_ini.tanggal, total: hari_ini.total });
	let antreanHariIni = $state(0);
	let targetInput = $state(33);
	let targetHarianInput = $state(target_harian);

	const counter = new TasbihCounter(() => ({ dzikir, paket }), {
		onTarget() {
			getar("target", pref.getar);
			if (pref.suara) klik("target");
			rayakanKey++;
			pengumuman = `Target ${counter.state?.target ?? ""} tercapai`;
		},
		onItemPaketSelesai(next) {
			getar("kelipatan", pref.getar);
			if (pref.suara) klik("target");
			rayakanKey++;
			const it = counter.paket?.items[next];
			const d = dzikir.find((x) => x.id === it?.dzikir_id);
			pengumuman = d ? `Lanjut: ${d.latin} ${it?.target}` : "";
		},
		onPaketSelesai(p) {
			getar("paket", pref.getar);
			rayakanKey++;
			ringkasanOpen = true;
			pengumuman = `${p.nama} selesai`;
		},
		onSesiTutup() {
			hitungAntreanHariIni();
			syncSekarang().then((h) => h && onSync(h));
		},
	});

	const s = $derived(counter.state);
	const d = $derived(counter.dzikir);
	const pk = $derived(counter.paket);
	const jumlah = $derived(s ? counter.tampil : 0);
	const jumlahSesi = $derived(s?.sesi_id ? s.jumlah : 0);
	const target = $derived(s?.target ?? 33);
	const progress = $derived(target > 0 ? Math.min(jumlah / target, 1) : 0);
	const lebih = $derived(s?.mode === "dzikir" && jumlah > target ? jumlah - target : 0);
	const totalHariIni = $derived(serverHariIni.total + antreanHariIni + jumlahSesi);
	const progressHarian = $derived(target_harian > 0 ? Math.min(totalHariIni / target_harian, 1) : 0);
	const streakTampil = $derived(streak.saat + (!streak.hari_ini_tercapai && totalHariIni >= target_harian ? 1 : 0));
	const levelProgress = $derived(
		level.ambang_berikut > 0 ? Math.min((level.total - level.ambang_ini) / (level.ambang_berikut - level.ambang_ini), 1) : 1,
	);
	const adaSheetTerbuka = $derived(pickerOpen || formOpen || pengaturanOpen || targetOpen || ringkasanOpen);

	// Komposisi hari ini = data server + antrean lokal belum tersinkron + sesi berjalan.
	let komposisiServer = $state<Komposisi[]>(komposisi_hari_ini ?? []);
	let lokalPerDzikir = $state<Record<number, number>>({});
	const komposisi = $derived.by(() => {
		const m = new Map<number, Komposisi>();
		for (const k of komposisiServer) m.set(k.dzikir_id, { ...k });
		const tambahKe = (id: number, n: number) => {
			if (!n) return;
			const cur = m.get(id) ?? { dzikir_id: id, latin: dzikir.find((x) => x.id === id)?.latin ?? "Dzikir", total: 0 };
			cur.total += n;
			m.set(id, cur);
		};
		for (const [id, n] of Object.entries(lokalPerDzikir)) tambahKe(Number(id), n);
		if (s?.sesi_id) tambahKe(s.dzikir_id, s.jumlah);
		return [...m.values()].sort((a, b) => b.total - a.total);
	});

	/** perbaruiKomposisi=false dipakai tepat setelah sync, sampai komposisi server
	 *  terbaru dimuat — supaya hitungan yang baru tersinkron tidak "hilang" sesaat. */
	async function hitungAntreanHariIni(perbaruiKomposisi = true) {
		const hari = tanggalWIB(new Date());
		let antrean: { dzikir_id: number; jumlah: number; mulai_at: string }[] = [];
		try {
			antrean = (await antreanSemua()).filter((x) => tanggalWIB(x.mulai_at) === hari);
		} catch {
			antrean = []; // IndexedDB tidak tersedia; status sync menampilkan "gagal"
		}
		antreanHariIni = antrean.reduce((a, x) => a + x.jumlah, 0);
		if (perbaruiKomposisi) {
			const per: Record<number, number> = {};
			for (const x of antrean) per[x.dzikir_id] = (per[x.dzikir_id] ?? 0) + x.jumlah;
			lokalPerDzikir = per;
		}
	}

	function muatUlangDataServer(props: string[]) {
		router.reload({
			only: props,
			onSuccess: (page) => {
				const p = page.props as unknown as { hari_ini?: HariIni; komposisi_hari_ini?: Komposisi[] };
				// page.props berisi SEMUA prop (yang tidak diminta = nilai lama): ambil yang diminta saja
				if (props.includes("hari_ini") && p.hari_ini) serverHariIni = { tanggal: p.hari_ini.tanggal, total: p.hari_ini.total };
				if (props.includes("komposisi_hari_ini") && p.komposisi_hari_ini) komposisiServer = p.komposisi_hari_ini;
				hitungAntreanHariIni();
			},
		});
	}

	function onSync(h: SyncHasil) {
		serverHariIni = { tanggal: h.hari_ini.tanggal, total: h.hari_ini.total };
		const ada = new Set(badges.map((b) => b.kode));
		badges = [...badges, ...h.badge_baru.filter((b) => !ada.has(b.kode))];
		hitungAntreanHariIni(false);
		if (h.diterima.length > 0) muatUlangDataServer(["streak", "level", "komposisi_hari_ini"]);
	}

	/** Ambil ulang angka dari server (mis. setelah berdzikir di perangkat lain). */
	async function segarkan() {
		const h = await syncSekarang(true);
		if (h) onSync(h);
		await hitungAntreanHariIni(false);
		muatUlangDataServer(["hari_ini", "streak", "level", "target_harian", "komposisi_hari_ini"]);
	}

	function tambah() {
		if (!s || adaSheetTerbuka) return;
		const berikut = jumlah + 1;
		const milestone = berikut % 33 === 0 && berikut !== target;
		getar(milestone ? "kelipatan" : "tap", pref.getar);
		if (pref.suara) klik("tap");
		counter.tap();
	}

	function onPointerDown(e: PointerEvent) {
		if (e.pointerType === "mouse" && e.button !== 0) return;
		e.preventDefault();
		tambah();
	}

	function onTapKey(e: KeyboardEvent) {
		if (e.key === " " || e.key === "Enter") {
			e.preventDefault();
			if (!e.repeat) tambah();
		}
	}

	function onWindowKey(e: KeyboardEvent) {
		if (adaSheetTerbuka) return;
		const el = e.target as HTMLElement | null;
		if (el && (el.closest("input, textarea, select, [contenteditable]") || (el.closest("button, a") && el.dataset.tap !== "1"))) return;
		if ((e.key === " " || e.key === "Enter") && el?.dataset.tap !== "1") {
			e.preventDefault();
			if (!e.repeat) tambah();
		} else if (e.key === "Backspace") {
			e.preventDefault();
			counter.undo();
		} else if (e.key === "Escape" && fokus) {
			keluarFokus();
		}
	}

	function pilihDzikir(id: number) {
		counter.pilihDzikir(id);
		pickerOpen = false;
	}

	function pilihPaket(id: number) {
		counter.pilihPaket(id);
		pickerOpen = false;
	}

	function bukaTarget() {
		targetInput = target;
		targetOpen = true;
	}

	function simpanTarget(e: SubmitEvent) {
		e.preventDefault();
		const id = s?.dzikir_id;
		const t = Math.round(targetInput);
		if (!id || !(t >= 1 && t <= 100000)) return;
		router.put(`/app/tasbih/dzikir/${id}/target`, { target: t }, {
			preserveScroll: true,
			onSuccess: () => {
				counter.setTargetAktif(t);
				targetOpen = false;
			},
		});
	}

	function simpanTargetHarian(e: SubmitEvent) {
		e.preventDefault();
		const t = Math.round(targetHarianInput);
		if (!(t >= 1 && t <= 100000)) return;
		router.put("/app/tasbih/target", { target: t }, { preserveScroll: true, onSuccess: () => (pengaturanOpen = false) });
	}

	async function masukFokus() {
		fokus = true;
		try {
			await document.documentElement.requestFullscreen?.();
		} catch {
			/* tidak didukung: tetap mode fokus tanpa fullscreen */
		}
	}

	function keluarFokus() {
		fokus = false;
		if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
	}

	function namaDzikirId(id: number) {
		return dzikir.find((x) => x.id === id)?.latin ?? "";
	}

	function formatDurasi(detik: number) {
		const m = Math.floor(detik / 60);
		return m > 0 ? `${m} menit ${detik % 60} detik` : `${detik} detik`;
	}

	$effect(() => {
		simpanPref({ getar: pref.getar, suara: pref.suara, layar: pref.layar });
	});

	$effect(() => {
		kunciLayar(pref.layar);
		return () => {
			kunciLayar(false);
		};
	});

	onMount(() => {
		// state awal disiapkan sinkron di dalam pulihkan(); counter langsung bisa dipakai
		// walau IndexedDB lambat/macet (iPhone).
		counter.pulihkan().catch((e) => console.warn("[tasbih] pemulihan state gagal", e));
		siap = true;
		hitungAntreanHariIni();
		const stop = mulaiAutoSync(onSync);
		const idle = setInterval(() => counter.cekIdle(), 30_000);
		const onVis = () => {
			if (document.visibilityState === "hidden") {
				// pindah tab/aplikasi/perangkat: kirim sesi berjalan sekarang
				counter.simpanSementara();
				counter.flush();
			} else {
				counter.cekIdle();
				segarkan();
			}
		};
		const onHide = () => {
			counter.simpanSementara();
			counter.flush();
		};
		const onFs = () => {
			if (!document.fullscreenElement && fokus) fokus = false;
		};
		document.addEventListener("visibilitychange", onVis);
		window.addEventListener("pagehide", onHide);
		document.addEventListener("fullscreenchange", onFs);
		return () => {
			stop();
			clearInterval(idle);
			counter.flush();
			document.removeEventListener("visibilitychange", onVis);
			window.removeEventListener("pagehide", onHide);
			document.removeEventListener("fullscreenchange", onFs);
		};
	});
</script>

<svelte:window onkeydown={onWindowKey} />

<AppLayout {user}>
	<div class="mx-auto w-full max-w-xl pb-6">
		<InstallBanner />
		<!-- Baris atas: pilih dzikir, status sync, aksi -->
		<div class="mb-4 flex items-center gap-2">
			<button
				type="button"
				onclick={() => (pickerOpen = true)}
				class="flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded-xl border border-neutral-200 bg-white px-3.5 text-left text-sm font-semibold text-neutral-900 transition hover:border-brand-300 dark:border-neutral-800 dark:bg-neutral-900 dark:text-white"
				data-testid="pilih-dzikir"
			>
				<span class="truncate">{pk ? pk.nama : (d?.latin ?? "Pilih dzikir")}</span>
				<ChevronDown size={16} class="ml-auto shrink-0 text-neutral-400" aria-hidden="true" />
			</button>
			<button type="button" onclick={masukFokus} class="grid h-11 w-11 place-items-center rounded-xl text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800" aria-label="Mode fokus">
				<Maximize2 size={18} />
			</button>
			<button type="button" onclick={() => { targetHarianInput = target_harian; pengaturanOpen = true; }} class="grid h-11 w-11 place-items-center rounded-xl text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800" aria-label="Pengaturan tasbih">
				<Settings2 size={18} />
			</button>
		</div>

		<!-- Panel counter -->
		<section
			class="relative flex flex-col items-center overflow-hidden rounded-[2rem] border border-neutral-200 bg-white px-4 pb-5 pt-6 dark:border-neutral-800 dark:bg-neutral-900
				{fokus ? 'fixed inset-0 z-40 justify-center rounded-none border-0' : ''}"
			aria-label="Tasbih"
		>
			{#if fokus}
				<button type="button" onclick={keluarFokus} class="absolute right-4 top-4 grid h-11 w-11 place-items-center rounded-full text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Keluar mode fokus">
					<Minimize2 size={18} />
				</button>
			{/if}

			{#if pk}
				<ol class="mb-4 flex flex-wrap items-center justify-center gap-1.5" aria-label="Urutan paket">
					{#each pk.items as it, i}
						<li
							class="rounded-full px-2.5 py-1 text-[11px] font-medium tabular-nums transition
								{i < (s?.item_index ?? 0) ? 'bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300' : ''}
								{i === (s?.item_index ?? 0) ? 'bg-brand-600 text-white dark:bg-brand-500 dark:text-neutral-950' : ''}
								{i > (s?.item_index ?? 0) ? 'bg-neutral-100 text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400' : ''}"
							aria-current={i === (s?.item_index ?? 0) ? "step" : undefined}
						>
							{namaDzikirId(it.dzikir_id)} {it.target}
						</li>
					{/each}
				</ol>
			{/if}

			<!-- Teks dzikir -->
			<div class="min-h-[6.5rem] w-full text-center">
				{#if d}
					{#key d.id}
						<div in:fade={{ duration: 200 }}>
							{#if d.arab}
								<p class="font-arab text-3xl leading-[1.9] text-neutral-900 sm:text-4xl dark:text-white" dir="rtl" lang="ar">{d.arab}</p>
							{/if}
							<p class="mt-1 text-sm font-semibold text-brand-700 dark:text-brand-300">{d.latin}</p>
							{#if d.arti}<p class="mx-auto mt-0.5 max-w-sm text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">{d.arti}</p>{/if}
						</div>
					{/key}
				{/if}
			</div>

			<!-- Area tap -->
			<div
				role="button"
				tabindex="0"
				data-tap="1"
				data-testid="tap-area"
				aria-label="Tambah hitungan. {jumlah} dari {target}"
				onpointerdown={onPointerDown}
				onkeydown={onTapKey}
				class="tap-area relative mt-4 grid w-full select-none place-items-center rounded-[1.75rem] py-6 outline-none focus-visible:ring-2 focus-visible:ring-brand-400 {fokus ? 'flex-1' : ''}"
			>
				{#key rayakanKey}
					{#if rayakanKey > 0}
						<span class="ripple pointer-events-none absolute h-64 w-64 rounded-full border-2 border-success" aria-hidden="true"></span>
						<span class="ripple ripple-2 pointer-events-none absolute h-64 w-64 rounded-full border border-success" aria-hidden="true"></span>
					{/if}
				{/key}
				<div class="press">
					<ProgressRing {progress} tercapai={s?.mode === "dzikir" && jumlah >= target} size={256}>
						<div class="text-center">
							{#key jumlah}
								<span class="pop block text-7xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-white" data-testid="hitungan">{siap ? jumlah : "–"}</span>
							{/key}
							<span class="mt-1 block text-sm tabular-nums text-neutral-500 dark:text-neutral-400">
								dari {target}{#if lebih > 0}<span class="ml-1 font-semibold text-success">+{lebih}</span>{/if}
							</span>
						</div>
					</ProgressRing>
				</div>
				<div class="mt-5 flex w-full justify-center">
					<BeadStrip count={jumlah} />
				</div>
				<p class="mt-3 text-xs text-neutral-400 dark:text-neutral-500">Ketuk di mana saja pada area ini</p>
			</div>

			{#if s?.mode === "dzikir"}
				<button type="button" onclick={bukaTarget} class="mt-1 inline-flex min-h-11 items-center gap-1.5 rounded-full px-3 text-xs font-medium text-neutral-500 hover:bg-neutral-100 dark:text-neutral-400 dark:hover:bg-neutral-800">
					<Pencil size={12} aria-hidden="true" /> Ubah target ({target})
				</button>
			{/if}

			<!-- Kontrol -->
			<div class="mt-3 grid w-full max-w-sm grid-cols-3 gap-2">
				<button type="button" onclick={() => counter.undo()} disabled={jumlahSesi === 0} class="flex h-12 flex-col items-center justify-center gap-0.5 rounded-xl bg-neutral-100 text-xs font-medium text-neutral-700 transition active:scale-[0.97] disabled:opacity-40 dark:bg-neutral-800 dark:text-neutral-200">
					<Undo2 size={16} aria-hidden="true" /> Batal 1
				</button>
				<button type="button" onclick={() => counter.reset()} disabled={jumlah === 0 && !(s?.mode === "paket" && (s?.item_index ?? 0) > 0)} class="flex h-12 flex-col items-center justify-center gap-0.5 rounded-xl bg-neutral-100 text-xs font-medium text-neutral-700 transition active:scale-[0.97] disabled:opacity-40 dark:bg-neutral-800 dark:text-neutral-200">
					<RotateCcw size={16} aria-hidden="true" /> Ulang
				</button>
				<button type="button" onclick={() => counter.selesai()} disabled={jumlah === 0 && !(s?.mode === "paket" && (s?.item_index ?? 0) > 0)} class="flex h-12 flex-col items-center justify-center gap-0.5 rounded-xl bg-brand-600 text-xs font-semibold text-white transition active:scale-[0.97] disabled:opacity-40 dark:bg-brand-500 dark:text-neutral-950" data-testid="selesai">
					<CheckCheck size={16} aria-hidden="true" /> Simpan sesi
				</button>
			</div>
			<div class="mt-4"><SyncIndicator /></div>
		</section>

		<!-- Ringkasan hari ini -->
		<section class="mt-4 grid grid-cols-[1fr_auto] gap-3 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900" aria-label="Ringkasan hari ini">
			<div class="min-w-0">
				<div class="flex items-baseline justify-between gap-2">
					<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Hari ini</span>
					<span class="text-sm tabular-nums text-neutral-900 dark:text-white" data-testid="total-hari-ini"><b>{totalHariIni.toLocaleString("id-ID")}</b><span class="text-neutral-400"> / {target_harian.toLocaleString("id-ID")}</span></span>
				</div>
				<div class="mt-2 h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800" role="progressbar" aria-valuemin="0" aria-valuemax={target_harian} aria-valuenow={totalHariIni} aria-label="Progres target harian">
					<div class="bar h-full rounded-full {progressHarian >= 1 ? 'bg-success' : 'bg-brand-500'}" style="width:{progressHarian * 100}%"></div>
				</div>
				<div class="mt-3 flex items-center gap-2 text-xs text-neutral-500 dark:text-neutral-400">
					<span class="font-semibold text-neutral-800 dark:text-neutral-200">Level {level.level}</span>
					<div class="h-1.5 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800" aria-hidden="true">
						<div class="h-full rounded-full bg-secondary-500" style="width:{levelProgress * 100}%"></div>
					</div>
					{#if level.ambang_berikut > 0}<span class="tabular-nums">{level.total.toLocaleString("id-ID")}/{level.ambang_berikut.toLocaleString("id-ID")}</span>{/if}
				</div>
			</div>
			<div class="flex flex-col items-center justify-center rounded-xl bg-warning/10 px-3 text-warning" aria-label="Streak {streakTampil} hari">
				<Flame size={20} aria-hidden="true" />
				<span class="text-lg font-bold tabular-nums leading-none">{streakTampil}</span>
				<span class="text-[10px] font-medium">hari</span>
			</div>
		</section>

		<section class="mt-4 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900" aria-label="Komposisi dzikir hari ini" data-testid="komposisi-hari-ini">
			<div class="mb-3 flex items-baseline justify-between gap-2">
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Komposisi hari ini</h2>
				{#if komposisi.length}<span class="text-xs text-neutral-500 dark:text-neutral-400">{komposisi.length} dzikir</span>{/if}
			</div>
			{#if komposisi.length}
				<KomposisiBar data={komposisi} />
			{:else}
				<p class="text-sm text-neutral-500 dark:text-neutral-400">Belum ada dzikir hari ini.</p>
			{/if}
		</section>

		<a href="/app/tasbih/dashboard" use:inertia class="mt-3 flex min-h-12 items-center justify-center gap-2 rounded-xl text-sm font-medium text-brand-700 hover:bg-brand-50 dark:text-brand-300 dark:hover:bg-brand-950/40">
			<BarChart3 size={16} aria-hidden="true" /> Lihat progres & riwayat
		</a>
	</div>

	<p class="sr-only" aria-live="polite">{pengumuman}</p>

	<BadgeToast {badges} ondismiss={(kode) => (badges = badges.filter((b) => b.kode !== kode))} />

	<DzikirPicker
		open={pickerOpen}
		{dzikir}
		{paket}
		aktifDzikir={s?.dzikir_id ?? null}
		aktifPaket={s?.mode === "paket" ? s.paket_id : null}
		onclose={() => (pickerOpen = false)}
		onpilihdzikir={pilihDzikir}
		onpilihpaket={pilihPaket}
		ontambah={() => { formEdit = null; formOpen = true; }}
		onubah={(x) => { formEdit = x; formOpen = true; }}
	/>
	<DzikirFormSheet open={formOpen} edit={formEdit} onclose={() => (formOpen = false)} />

	<Sheet open={targetOpen} title="Target {d?.latin ?? ''}" onclose={() => (targetOpen = false)}>
		<form class="space-y-4" onsubmit={simpanTarget}>
			<div class="flex flex-wrap gap-2">
				{#each [33, 99, 100, 300, 1000] as t}
					<button type="button" onclick={() => (targetInput = t)} class="h-11 rounded-full px-4 text-sm font-medium tabular-nums {targetInput === t ? 'bg-brand-600 text-white dark:bg-brand-500 dark:text-neutral-950' : 'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200'}">{t}</button>
				{/each}
			</div>
			<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
				Target lain
				<input type="number" inputmode="numeric" min="1" max="100000" bind:value={targetInput} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-base dark:border-neutral-700 dark:bg-neutral-950 dark:text-white" />
			</label>
			<button type="submit" class="h-12 w-full rounded-xl bg-brand-600 font-semibold text-white dark:bg-brand-500 dark:text-neutral-950">Simpan target</button>
		</form>
	</Sheet>

	<Sheet open={pengaturanOpen} title="Pengaturan tasbih" onclose={() => (pengaturanOpen = false)}>
		<div class="space-y-1">
			{#each [["getar", "Getar setiap ketukan", "Tidak tersedia di iPhone (Safari)."], ["suara", "Suara klik", ""], ["layar", "Layar tetap menyala", "Saat halaman tasbih terbuka."]] as [key, label, hint]}
				{@const k = key as "getar" | "suara" | "layar"}
				<button type="button" role="switch" aria-checked={pref[k]} onclick={() => (pref[k] = !pref[k])} class="flex min-h-14 w-full items-center justify-between gap-4 rounded-xl px-2 text-left hover:bg-neutral-50 dark:hover:bg-neutral-800">
					<span>
						<span class="block text-sm font-medium text-neutral-900 dark:text-white">{label}</span>
						{#if hint}<span class="block text-xs text-neutral-500">{hint}</span>{/if}
					</span>
					<span class="relative h-7 w-12 shrink-0 rounded-full transition {pref[k] ? 'bg-brand-600 dark:bg-brand-500' : 'bg-neutral-300 dark:bg-neutral-700'}">
						<span class="absolute top-1 h-5 w-5 rounded-full bg-white shadow transition-all {pref[k] ? 'left-6' : 'left-1'}"></span>
					</span>
				</button>
			{/each}
		</div>
		<form class="mt-5 border-t border-neutral-100 pt-5 dark:border-neutral-800" onsubmit={simpanTargetHarian}>
			<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
				Target harian (total semua dzikir)
				<input type="number" inputmode="numeric" min="1" max="100000" bind:value={targetHarianInput} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-base dark:border-neutral-700 dark:bg-neutral-950 dark:text-white" />
			</label>
			<p class="mt-1.5 text-xs text-neutral-500">Hari dihitung untuk streak bila total hari itu mencapai target ini.</p>
			<button type="submit" class="mt-4 h-12 w-full rounded-xl bg-brand-600 font-semibold text-white dark:bg-brand-500 dark:text-neutral-950">Simpan target harian</button>
		</form>
	</Sheet>

	<Sheet open={ringkasanOpen} title="Paket selesai" onclose={() => (ringkasanOpen = false)}>
		{#if counter.ringkasanPaket}
			{@const r = counter.ringkasanPaket}
			<div class="text-center" in:scale={{ start: 0.95, duration: 200 }}>
				<p class="text-lg font-semibold text-neutral-900 dark:text-white">{r.paket.nama}</p>
				{#if r.durasiDetik > 0}<p class="text-sm text-neutral-500">Selesai dalam {formatDurasi(r.durasiDetik)}</p>{/if}
			</div>
			<ul class="mt-4 divide-y divide-neutral-100 dark:divide-neutral-800">
				{#each r.items as it}
					<li class="flex items-center justify-between py-2.5 text-sm">
						<span class="text-neutral-700 dark:text-neutral-300">{namaDzikirId(it.dzikir_id)}</span>
						<span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{it.jumlah}</span>
					</li>
				{/each}
			</ul>
			<p class="mt-3 text-xs text-neutral-500">Rujukan: {r.paket.rujukan}</p>
			<div class="mt-5 grid grid-cols-2 gap-2">
				<button type="button" onclick={() => (ringkasanOpen = false)} class="h-12 rounded-xl bg-neutral-100 text-sm font-semibold text-neutral-800 dark:bg-neutral-800 dark:text-neutral-100">Ulangi paket</button>
				<a href="/app/tasbih/dashboard" use:inertia class="grid h-12 place-items-center rounded-xl bg-brand-600 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950">Lihat progres</a>
			</div>
		{/if}
	</Sheet>
</AppLayout>

<style>
	.tap-area {
		touch-action: manipulation;
		-webkit-tap-highlight-color: transparent;
		cursor: pointer;
	}
	.tap-area:active .press {
		transform: scale(0.985);
	}
	.press {
		transition: transform 120ms cubic-bezier(0.32, 0.72, 0, 1);
	}
	.pop {
		animation: pop 180ms cubic-bezier(0.32, 0.72, 0, 1);
	}
	@keyframes pop {
		from {
			transform: scale(1.12);
		}
		to {
			transform: scale(1);
		}
	}
	.ripple {
		animation: ripple 900ms cubic-bezier(0.22, 1, 0.36, 1) forwards;
		opacity: 0;
	}
	.ripple-2 {
		animation-delay: 150ms;
	}
	@keyframes ripple {
		from {
			transform: scale(0.85);
			opacity: 0.7;
		}
		to {
			transform: scale(1.45);
			opacity: 0;
		}
	}
	.bar {
		transition: width 300ms cubic-bezier(0.32, 0.72, 0, 1);
	}
	@media (prefers-reduced-motion: reduce) {
		.pop,
		.ripple {
			animation: none;
		}
		.press,
		.bar {
			transition: none;
		}
	}
</style>
