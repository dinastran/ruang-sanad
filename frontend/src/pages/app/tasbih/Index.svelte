<script lang="ts">
	import { onMount } from "svelte";
	import { fade, scale } from "svelte/transition";
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Check, ChevronDown, Flame, ListOrdered, Maximize2, Minimize2, RotateCcw, Settings2, Undo2, CheckCheck, Pencil } from "lucide-svelte";
	import BeadCircle from "@components/tasbih/BeadCircle.svelte";
	import SyncIndicator from "@components/tasbih/SyncIndicator.svelte";
	import BadgeToast from "@components/tasbih/BadgeToast.svelte";
	import DzikirPicker from "@components/tasbih/DzikirPicker.svelte";
	import DzikirFormSheet from "@components/tasbih/DzikirFormSheet.svelte";
	import Sheet from "@components/tasbih/Sheet.svelte";
	import InstallBanner from "@components/pwa/InstallBanner.svelte";
	import { TasbihCounter } from "@lib/tasbih/counter.svelte";
	import { antreanSemua, setPemilik } from "@lib/tasbih/db";
	import { mulaiAutoSync, syncSekarang } from "@lib/tasbih/sync.svelte";
	import { bacaPref, getar, klik, kunciLayar, simpanPref } from "@lib/tasbih/feedback";
	import { tanggalWIB } from "@lib/tasbih/types";
	import type { Badge, Dzikir, HariIni, Paket, Streak, SyncHasil } from "@lib/tasbih/types";
	import type { User } from "@lib/types";

	interface Props {
		user?: User;
		dzikir: Dzikir[];
		paket: Paket[];
		hari_ini: HariIni;
		target_harian: number;
		streak: Streak;
		badge_baru: Badge[];
	}
	// level & komposisi_hari_ini tetap dikirim server; di layar hitung sengaja tidak ditampilkan (ada di Progres).
	let { user, dzikir, paket, hari_ini, target_harian, streak, badge_baru }: Props = $props();

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
	const adaSheetTerbuka = $derived(pickerOpen || formOpen || pengaturanOpen || targetOpen || ringkasanOpen);

	async function hitungAntreanHariIni() {
		const hari = tanggalWIB(new Date());
		try {
			antreanHariIni = (await antreanSemua()).filter((x) => tanggalWIB(x.mulai_at) === hari).reduce((a, x) => a + x.jumlah, 0);
		} catch {
			antreanHariIni = 0; // IndexedDB tidak tersedia; status sync menampilkan "gagal"
		}
	}

	function muatUlangDataServer(props: string[]) {
		router.reload({
			only: props,
			onSuccess: (page) => {
				const p = page.props as unknown as { hari_ini?: HariIni };
				// page.props berisi SEMUA prop (yang tidak diminta = nilai lama): ambil yang diminta saja
				if (props.includes("hari_ini") && p.hari_ini) serverHariIni = { tanggal: p.hari_ini.tanggal, total: p.hari_ini.total };
				hitungAntreanHariIni();
			},
		});
	}

	function onSync(h: SyncHasil) {
		serverHariIni = { tanggal: h.hari_ini.tanggal, total: h.hari_ini.total };
		const ada = new Set(badges.map((b) => b.kode));
		badges = [...badges, ...h.badge_baru.filter((b) => !ada.has(b.kode))];
		hitungAntreanHariIni();
		if (h.diterima.length > 0) muatUlangDataServer(["streak"]);
	}

	/** Ambil ulang angka dari server (mis. setelah berdzikir di perangkat lain). */
	async function segarkan() {
		const h = await syncSekarang(true);
		if (h) onSync(h);
		await hitungAntreanHariIni();
		muatUlangDataServer(["hari_ini", "streak", "target_harian"]);
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

	function formatDurasiPendek(detik: number) {
		const m = Math.floor(detik / 60);
		return m > 0 ? `${m}m ${detik % 60}d` : `${detik}d`;
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
	<div class="mx-auto flex w-full max-w-xl flex-col pb-4">
		<InstallBanner />
		<!-- Baris atas: pilih dzikir/paket + dua aksi. Sengaja ringkas agar fokus ke hitungan. -->
		<div class="flex items-center gap-1">
			<button
				type="button"
				onclick={() => (pickerOpen = true)}
				class="mr-auto flex h-11 min-w-0 max-w-[75%] items-center gap-2 rounded-full border border-neutral-200 bg-white pl-3.5 pr-3 text-left text-sm font-semibold text-neutral-900 transition hover:border-brand-300 dark:border-brand-200/10 dark:bg-brand-950/40 dark:text-white"
				data-testid="pilih-dzikir"
			>
				{#if pk}<ListOrdered size={15} class="shrink-0 text-neutral-400 dark:text-brand-200/50" aria-hidden="true" />{/if}
				<span class="truncate">{pk ? pk.nama : (d?.latin ?? "Pilih dzikir")}</span>
				{#if pk}<span class="shrink-0 tabular-nums text-neutral-400 dark:text-brand-200/50">· {(s?.item_index ?? 0) + 1}/{pk.items.length}</span>{/if}
				<ChevronDown size={15} class="shrink-0 text-neutral-400" aria-hidden="true" />
			</button>
			<button type="button" onclick={masukFokus} class="grid h-11 w-11 place-items-center rounded-xl text-neutral-500 hover:bg-neutral-100 dark:text-brand-100/60 dark:hover:bg-brand-950/60" aria-label="Mode fokus">
				<Maximize2 size={18} />
			</button>
			<button type="button" onclick={() => { targetHarianInput = target_harian; pengaturanOpen = true; }} class="grid h-11 w-11 place-items-center rounded-xl text-neutral-500 hover:bg-neutral-100 dark:text-brand-100/60 dark:hover:bg-brand-950/60" aria-label="Pengaturan tasbih">
				<Settings2 size={18} />
			</button>
		</div>

		<!-- Counter -->
		<section
			class="relative flex flex-col items-center {fokus ? 'fixed inset-0 z-40 justify-center bg-neutral-50 px-4 dark:bg-neutral-950' : 'mt-2'}"
			aria-label="Tasbih"
		>
			{#if fokus}
				<button type="button" onclick={keluarFokus} class="absolute right-4 top-[calc(1rem+env(safe-area-inset-top))] grid h-11 w-11 place-items-center rounded-full text-neutral-500 hover:bg-neutral-100 dark:hover:bg-brand-950/60" aria-label="Keluar mode fokus">
					<Minimize2 size={18} />
				</button>
			{/if}

			<!-- Teks dzikir -->
			<div class="flex min-h-[7rem] w-full flex-col justify-center text-center">
				{#if d}
					{#key d.id}
						<div in:fade={{ duration: 200 }}>
							{#if d.arab}
								<p class="font-arab text-[2.15rem] leading-[1.75] text-neutral-900 sm:text-4xl dark:text-white" dir="rtl" lang="ar">{d.arab}</p>
							{/if}
							<p class="text-[15px] font-medium text-neutral-700 dark:text-brand-100/80">{d.latin}</p>
							{#if d.arti}<p class="mx-auto mt-0.5 max-w-sm text-[13px] leading-relaxed text-neutral-500 dark:text-brand-100/45">{d.arti}</p>{/if}
						</div>
					{/key}
				{/if}
			</div>

			{#if pk}
				<ol class="mt-3 flex items-center justify-center gap-1.5" aria-label="Urutan paket">
					{#each pk.items as it, i}
						{@const idx = s?.item_index ?? 0}
						<li aria-current={i === idx ? "step" : undefined}>
							<span class="block h-1 w-7 rounded-full transition-colors {i < idx ? 'bg-brand-500/50 dark:bg-brand-300/50' : i === idx ? 'bg-brand-500 dark:bg-brand-300' : 'bg-neutral-200 dark:bg-brand-950'}"></span>
							<span class="sr-only">{namaDzikirId(it.dzikir_id)} {it.target}{i < idx ? ", selesai" : ""}</span>
						</li>
					{/each}
				</ol>
			{/if}

			<!-- Area tap: seluruh blok ring, cukup lebar untuk ibu jari -->
			<div
				role="button"
				tabindex="0"
				data-tap="1"
				data-testid="tap-area"
				aria-label="Tambah hitungan. {jumlah} dari {target}"
				onpointerdown={onPointerDown}
				onkeydown={onTapKey}
				class="tap-area relative mt-4 grid w-full select-none place-items-center rounded-[2rem] py-4 outline-none focus-visible:ring-2 focus-visible:ring-brand-400 {fokus ? 'flex-1' : ''}"
			>
				{#key rayakanKey}
					{#if rayakanKey > 0}
						<span class="ripple pointer-events-none absolute h-64 w-64 rounded-full border-2 border-success" aria-hidden="true"></span>
					{/if}
				{/key}
				<div class="press">
					<BeadCircle count={jumlah} {target} tercapai={s?.mode === "dzikir" && jumlah >= target} size={272}>
						<div class="text-center">
							{#key jumlah}
								<span class="pop block text-[5.5rem] font-light leading-none tracking-tighter tabular-nums text-neutral-900 dark:text-white" data-testid="hitungan">{siap ? jumlah : "–"}</span>
							{/key}
							<span class="mt-2 block text-sm tabular-nums text-neutral-500 dark:text-brand-100/45">
								dari {target}{#if lebih > 0}<span class="ml-1 font-semibold text-success">+{lebih}</span>{/if}
							</span>
						</div>
					</BeadCircle>
				</div>
				<p class="mt-3 text-xs text-neutral-400 dark:text-brand-100/35">Ketuk di mana saja pada area ini</p>
			</div>

			<!-- Kontrol sekunder: kecil dan tenang -->
			<div class="mt-1 flex items-center justify-center gap-1">
				<button type="button" onclick={() => counter.undo()} disabled={jumlahSesi === 0} class="ctl" aria-label="Batal 1">
					<Undo2 size={16} aria-hidden="true" /><span>Batal 1</span>
				</button>
				<button type="button" onclick={() => counter.reset()} disabled={jumlah === 0 && !(s?.mode === "paket" && (s?.item_index ?? 0) > 0)} class="ctl" aria-label="Ulang">
					<RotateCcw size={16} aria-hidden="true" /><span>Ulang</span>
				</button>
				{#if s?.mode === "dzikir"}
					<button type="button" onclick={bukaTarget} class="ctl" aria-label="Ubah target, sekarang {target}">
						<Pencil size={15} aria-hidden="true" /><span>Target</span>
					</button>
				{/if}
				<button type="button" onclick={() => counter.selesai()} disabled={jumlah === 0 && !(s?.mode === "paket" && (s?.item_index ?? 0) > 0)} class="ctl ctl-utama" data-testid="selesai">
					<CheckCheck size={16} aria-hidden="true" /><span>Simpan</span>
				</button>
			</div>
			<div class="mt-3"><SyncIndicator /></div>
		</section>

		<!-- Hari ini: progres harian + streak (level & komposisi ada di Progres) -->
		<a
			href="/app/tasbih/dashboard"
			use:inertia
			class="mt-5 flex items-center gap-4 rounded-2xl border border-neutral-200 bg-white px-4 py-3.5 transition hover:border-brand-300 dark:border-brand-200/10 dark:bg-brand-950/40"
			aria-label="Hari ini {totalHariIni} dari {target_harian}, streak {streakTampil} hari. Buka progres"
		>
			<div class="min-w-0 flex-1">
				<div class="flex items-baseline justify-between gap-2 text-[13px]">
					<span class="text-neutral-500 dark:text-brand-100/60">Hari ini</span>
					<span class="tabular-nums text-neutral-500 dark:text-brand-100/60" data-testid="total-hari-ini"><b class="font-semibold text-neutral-900 dark:text-white">{totalHariIni.toLocaleString("id-ID")}</b> / {target_harian.toLocaleString("id-ID")}</span>
				</div>
				<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-brand-950" aria-hidden="true">
					<div class="bar h-full rounded-full {progressHarian >= 1 ? 'bg-success' : 'bg-brand-500 dark:bg-brand-300'}" style="width:{progressHarian * 100}%"></div>
				</div>
			</div>
			<span class="flex items-center gap-1 text-sm font-semibold tabular-nums text-sand-600 dark:text-sand-300">
				<Flame size={16} aria-hidden="true" />{streakTampil}
			</span>
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
					<button type="button" onclick={() => (targetInput = t)} class="h-11 rounded-full px-4 text-sm font-medium tabular-nums {targetInput === t ? 'bg-brand-600 text-white dark:bg-brand-300 dark:text-brand-950' : 'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200'}">{t}</button>
				{/each}
			</div>
			<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
				Target lain
				<input type="number" inputmode="numeric" min="1" max="100000" bind:value={targetInput} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-base dark:border-neutral-700 dark:bg-neutral-950 dark:text-white" />
			</label>
			<button type="submit" class="h-12 w-full rounded-xl bg-brand-600 font-semibold text-white dark:bg-brand-300 dark:text-brand-950">Simpan target</button>
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
					<span class="relative h-7 w-12 shrink-0 rounded-full transition {pref[k] ? 'bg-brand-600 dark:bg-brand-400' : 'bg-neutral-300 dark:bg-neutral-700'}">
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
			<button type="submit" class="mt-4 h-12 w-full rounded-xl bg-brand-600 font-semibold text-white dark:bg-brand-300 dark:text-brand-950">Simpan target harian</button>
		</form>
	</Sheet>

	<!-- Paket tuntas: ringkasan tenang, tanpa confetti -->
	<Sheet open={ringkasanOpen} title="Paket selesai" onclose={() => (ringkasanOpen = false)}>
		{#if counter.ringkasanPaket}
			{@const r = counter.ringkasanPaket}
			{@const totalPaket = r.items.reduce((a, it) => a + it.jumlah, 0)}
			<div class="text-center" in:scale={{ start: 0.96, duration: 200 }}>
				<span class="mx-auto grid h-16 w-16 place-items-center rounded-full bg-brand-50 text-brand-600 dark:bg-brand-300/15 dark:text-brand-300"><Check size={30} strokeWidth={1.75} aria-hidden="true" /></span>
				<p class="mt-4 text-xl font-semibold tracking-tight text-neutral-900 dark:text-white">{r.paket.nama} tuntas</p>
				<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
					{totalHariIni >= target_harian ? "Target harian juga sudah tercapai." : `${(target_harian - totalHariIni).toLocaleString("id-ID")} lagi menuju target harian.`}
				</p>
			</div>
			<dl class="mt-5 grid grid-cols-3 divide-x divide-neutral-100 border-y border-neutral-100 text-center dark:divide-neutral-800 dark:border-neutral-800">
				<div class="py-3.5"><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{totalPaket}</dd><dt class="text-[11px] tracking-wide text-neutral-500 uppercase">Hitungan</dt></div>
				<div class="py-3.5"><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{r.durasiDetik > 0 ? formatDurasiPendek(r.durasiDetik) : "—"}</dd><dt class="text-[11px] tracking-wide text-neutral-500 uppercase">Durasi</dt></div>
				<div class="py-3.5"><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{totalHariIni.toLocaleString("id-ID")}</dd><dt class="text-[11px] tracking-wide text-neutral-500 uppercase">Hari ini</dt></div>
			</dl>
			<ul class="mt-3 divide-y divide-neutral-100 dark:divide-neutral-800">
				{#each r.items as it}
					<li class="flex items-center justify-between py-2 text-sm">
						<span class="text-neutral-600 dark:text-neutral-300">{namaDzikirId(it.dzikir_id)}</span>
						<span class="font-medium tabular-nums text-neutral-900 dark:text-white">{it.jumlah}</span>
					</li>
				{/each}
			</ul>
			{#if r.paket.rujukan}<p class="mt-2 text-xs text-neutral-500">Rujukan: {r.paket.rujukan}</p>{/if}
			<div class="mt-5 grid grid-cols-2 gap-2">
				<button type="button" onclick={() => (ringkasanOpen = false)} class="inline-flex h-12 items-center justify-center gap-1.5 rounded-xl bg-neutral-100 text-sm font-semibold text-neutral-800 dark:bg-neutral-800 dark:text-neutral-100"><RotateCcw size={15} aria-hidden="true" /> Ulangi</button>
				<a href="/app/tasbih/dashboard" use:inertia class="grid h-12 place-items-center rounded-xl bg-brand-600 text-sm font-semibold text-white dark:bg-brand-300 dark:text-brand-950">Lihat progres</a>
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
	.ctl {
		display: inline-flex;
		min-height: 2.75rem;
		align-items: center;
		gap: 0.375rem;
		border-radius: 9999px;
		padding: 0 0.875rem;
		font-size: 0.8125rem;
		font-weight: 500;
		white-space: nowrap;
		color: var(--color-neutral-500);
		transition: background-color 150ms ease, transform 120ms ease;
	}
	.ctl:hover:not(:disabled) {
		background-color: var(--color-neutral-100);
	}
	.ctl:active:not(:disabled) {
		transform: scale(0.97);
	}
	.ctl:disabled {
		opacity: 0.35;
	}
	.ctl-utama {
		color: var(--color-brand-700);
		font-weight: 600;
	}
	:global(.dark) .ctl {
		color: color-mix(in oklab, var(--color-brand-100) 60%, transparent);
	}
	:global(.dark) .ctl:hover:not(:disabled) {
		background-color: color-mix(in oklab, var(--color-brand-950) 60%, transparent);
	}
	:global(.dark) .ctl-utama {
		color: var(--color-brand-300);
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
