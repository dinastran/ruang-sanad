<script lang="ts">
	import { onMount } from "svelte";
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { ArrowDownRight, ArrowUpRight, Check, CircleDot, CloudUpload, Flame, ListOrdered, Play, Trophy } from "lucide-svelte";
	import BeadRing from "@components/tasbih/BeadRing.svelte";
	import DailyBar from "@components/tasbih/DailyBar.svelte";
	import Heatmap from "@components/tasbih/Heatmap.svelte";
	import JamDzikir from "@components/tasbih/JamDzikir.svelte";
	import PolaPekan from "@components/tasbih/PolaPekan.svelte";
	import KomposisiBar from "@components/tasbih/KomposisiBar.svelte";
	import BadgeGrid from "@components/tasbih/BadgeGrid.svelte";
	import BadgeToast from "@components/tasbih/BadgeToast.svelte";
	import RiwayatList from "@components/tasbih/RiwayatList.svelte";
	import SyncIndicator from "@components/tasbih/SyncIndicator.svelte";
	import { angka, durasi, tglPanjang, tglPendek } from "@components/tasbih/format";
	import { antreanSemua, setPemilik } from "@lib/tasbih/db";
	import { syncSekarang } from "@lib/tasbih/sync.svelte";
	import { tanggalWIB } from "@lib/tasbih/types";
	import type { Badge, DashboardData, Dzikir, RiwayatItem, SesiTertutup } from "@lib/tasbih/types";
	import type { User } from "@lib/types";

	interface Props {
		user?: User;
		dashboard: DashboardData;
		badges: Badge[];
		riwayat: { items: RiwayatItem[]; ada_lagi: boolean };
		dzikir: Dzikir[];
		target_harian: number;
		badge_baru: Badge[];
	}
	let { user, dashboard, badges, riwayat, dzikir, target_harian, badge_baru }: Props = $props();

	if (user?.id) setPemilik(user.id);

	let lokal = $state<SesiTertutup[]>([]);

	async function bacaLokal() {
		try {
			lokal = await antreanSemua();
		} catch {
			lokal = [];
		}
	}
	let badgeToast = $state<Badge[]>([...(badge_baru ?? [])]);

	const RENTANG = [
		{ v: 7, l: "7 hari" },
		{ v: 30, l: "30 hari" },
		{ v: 90, l: "90 hari" },
		{ v: 365, l: "1 tahun" },
	];

	const d = $derived(dashboard);
	const lokalHariIniSesi = $derived(lokal.filter((s) => tanggalWIB(s.mulai_at) === d.hari_ini.tanggal));
	const hariIni = $derived(d.hari_ini.total + lokalHariIniSesi.reduce((a, s) => a + s.jumlah, 0));
	const tercapai = $derived(hariIni >= target_harian);
	const sisa = $derived(Math.max(target_harian - hariIni, 0));
	const streak = $derived(d.streak.saat + (!d.streak.hari_ini_tercapai && tercapai ? 1 : 0));
	const delta = $derived(d.total_rentang_sebelum > 0 ? Math.round(((d.total_rentang - d.total_rentang_sebelum) / d.total_rentang_sebelum) * 100) : null);
	const levelPct = $derived(d.level.ambang_berikut > 0 ? Math.min((d.level.total - d.level.ambang_ini) / (d.level.ambang_berikut - d.level.ambang_ini), 1) : 1);
	const kosong = $derived(d.level.total === 0 && lokal.length === 0);
	const hariAktif = $derived(d.harian.filter((h) => h.total > 0).length);
	const hariAktifTahun = $derived(d.heatmap.filter((h) => h.total > 0).length);
	const hariTercapaiTahun = $derived(d.heatmap.filter((h) => h.total >= target_harian).length);
	const labelRentang = $derived(RENTANG.find((r) => r.v === d.rentang)?.l ?? `${d.rentang} hari`);
	const pctTuntas = $derived(d.sesi.jumlah ? Math.round((d.sesi.tuntas / d.sesi.jumlah) * 100) : 0);

	// Dzikir hari ini dibandingkan target per dzikir (server + sesi lokal belum sinkron).
	const dzikirHariIni = $derived.by(() => {
		const map = new Map<number, { id: number; latin: string; total: number; target: number }>();
		for (const k of d.hari_ini_dzikir ?? []) {
			const z = dzikir.find((x) => x.id === k.dzikir_id);
			map.set(k.dzikir_id, { id: k.dzikir_id, latin: k.latin, total: k.total, target: z?.target ?? 0 });
		}
		for (const s of lokalHariIniSesi) {
			const ada = map.get(s.dzikir_id);
			if (ada) ada.total += s.jumlah;
			else {
				const z = dzikir.find((x) => x.id === s.dzikir_id);
				map.set(s.dzikir_id, { id: s.dzikir_id, latin: z?.latin ?? "Dzikir", total: s.jumlah, target: z?.target ?? 0 });
			}
		}
		return [...map.values()].sort((a, b) => b.total - a.total);
	});

	const tanggalHariIni = $derived(tglPanjang(d.hari_ini.tanggal));
	const tanggalHijri = $derived.by(() => {
		try {
			const [y, m, dd] = d.hari_ini.tanggal.split("-").map(Number);
			return new Intl.DateTimeFormat("id-ID-u-ca-islamic-umalqura", { day: "numeric", month: "long", year: "numeric" }).format(new Date(y, m - 1, dd));
		} catch {
			return "";
		}
	});

	function gantiRentang(v: number) {
		router.get("/app/tasbih/dashboard", { rentang: v }, { preserveScroll: true, preserveState: true, only: ["dashboard"] });
	}

	async function segarkan(paksaReload: boolean) {
		await bacaLokal();
		const h = await syncSekarang(true);
		await bacaLokal();
		if (h?.badge_baru.length) badgeToast = [...badgeToast, ...h.badge_baru];
		if (paksaReload || (h?.diterima.length ?? 0) > 0) router.reload({ only: ["dashboard", "badges", "riwayat"] });
	}

	onMount(() => {
		segarkan(false);
		// kembali ke tab ini (mis. setelah berdzikir di perangkat lain): ambil data terbaru
		const onVis = () => {
			if (document.visibilityState === "visible") segarkan(true);
		};
		document.addEventListener("visibilitychange", onVis);
		return () => document.removeEventListener("visibilitychange", onVis);
	});

	const panel = "min-w-0 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900";
	const judul = "text-sm font-semibold text-neutral-900 dark:text-white";
	const sub = "mt-0.5 text-xs text-neutral-500 dark:text-neutral-400";
</script>

<AppLayout {user}>
	<div class="@container mx-auto w-full min-w-0 max-w-6xl pb-10">
		<!-- Header -->
		<header class="mb-5 flex flex-wrap items-end justify-between gap-3">
			<div class="min-w-0">
				<p class="text-xs font-medium text-neutral-500 dark:text-neutral-400">
					{tanggalHariIni}{#if tanggalHijri}<span class="text-neutral-400 dark:text-neutral-500"> · {tanggalHijri}</span>{/if}
				</p>
				<h1 class="mt-0.5 text-2xl font-semibold tracking-tight text-balance text-neutral-900 dark:text-white">Progres dzikir</h1>
				<div class="mt-1"><SyncIndicator /></div>
			</div>
			<div class="flex w-full items-center gap-2 @xl:w-auto">
				<div class="grid flex-1 grid-cols-4 gap-1 rounded-xl bg-neutral-100 p-1 @xl:flex-none dark:bg-neutral-800" role="group" aria-label="Rentang waktu">
					{#each RENTANG as r}
						<button
							type="button"
							onclick={() => gantiRentang(r.v)}
							aria-pressed={d.rentang === r.v}
							class="h-9 rounded-lg px-3 text-xs font-medium whitespace-nowrap transition-colors {d.rentang === r.v ? 'bg-white text-neutral-900 shadow-sm dark:bg-neutral-950 dark:text-white' : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-200'}"
						>{r.l}</button>
					{/each}
				</div>
			</div>
		</header>

		{#if lokal.length > 0}
			<p class="mb-4 flex items-center gap-2 rounded-xl bg-warning/10 px-4 py-2.5 text-xs font-medium text-warning">
				<CloudUpload size={14} aria-hidden="true" />
				{lokal.length} sesi menunggu sinkron — grafik akan diperbarui setelah tersimpan di server.
			</p>
		{/if}

		{#if kosong}
			<section class="rounded-3xl border border-neutral-200 bg-white px-6 py-14 text-center dark:border-neutral-800 dark:bg-neutral-900">
				<div class="mx-auto w-fit">
					<BeadRing progress={0} size={120}>
						<CircleDot size={26} class="text-brand-600 dark:text-brand-300" aria-hidden="true" />
					</BeadRing>
				</div>
				<h2 class="mt-4 text-lg font-semibold text-neutral-900 dark:text-white">Belum ada dzikir tercatat</h2>
				<p class="mx-auto mt-1 max-w-sm text-sm text-pretty text-neutral-500 dark:text-neutral-400">Mulai sesi pertama di tasbih. Progres, pola waktu, riwayat, dan pencapaian akan muncul di sini.</p>
				<a href="/app/tasbih" use:inertia class="press mt-6 inline-flex h-12 items-center gap-2 rounded-xl bg-brand-600 px-6 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950"><Play size={16} aria-hidden="true" /> Mulai dzikir</a>
			</section>
		{:else}
			<!-- Hero: hari ini (focal point) -->
			<section class="rounded-3xl border border-neutral-200 bg-white p-5 @xl:p-6 dark:border-neutral-800 dark:bg-neutral-900" aria-label="Hari ini">
				<div class="grid gap-6 @4xl:grid-cols-[auto_minmax(0,1fr)_15rem] @4xl:items-center">
					<div class="flex items-center gap-5">
						<BeadRing progress={target_harian ? hariIni / target_harian : 0} size={184}>
							<div class="text-center">
								<span class="block text-[2.5rem] leading-none font-semibold tracking-tight tabular-nums text-neutral-900 dark:text-white" data-testid="dash-hari-ini">{angka(hariIni)}</span>
								<span class="mt-1.5 block text-xs tabular-nums text-neutral-500">dari {angka(target_harian)}</span>
							</div>
						</BeadRing>
						<div class="@4xl:hidden">
							<p class="text-xs font-medium tracking-wide text-neutral-500 uppercase">Hari ini</p>
							<p class="mt-1 text-sm font-medium text-pretty {tercapai ? 'text-success' : 'text-neutral-800 dark:text-neutral-200'}">{tercapai ? "Target harian tercapai" : `${angka(sisa)} lagi menuju target`}</p>
							<a href="/app/tasbih" use:inertia class="press mt-3 inline-flex h-10 items-center gap-1.5 rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950"><Play size={14} aria-hidden="true" /> Lanjut dzikir</a>
						</div>
					</div>

					<div class="min-w-0">
						<div class="hidden items-start justify-between gap-3 @4xl:flex">
							<div>
								<p class="text-xs font-medium tracking-wide text-neutral-500 uppercase">Hari ini</p>
								<p class="mt-1 text-lg font-semibold text-balance {tercapai ? 'text-success' : 'text-neutral-900 dark:text-white'}">{tercapai ? "Target harian tercapai" : `${angka(sisa)} lagi menuju target`}</p>
							</div>
							<a href="/app/tasbih" use:inertia class="press inline-flex h-10 shrink-0 items-center gap-1.5 rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950"><Play size={14} aria-hidden="true" /> Lanjut dzikir</a>
						</div>
						{#if dzikirHariIni.length}
							<ul class="mt-0 grid gap-x-6 gap-y-3 @xl:grid-cols-2 @4xl:mt-4" aria-label="Dzikir hari ini">
								{#each dzikirHariIni.slice(0, 6) as z (z.id)}
									{@const selesai = z.target > 0 && z.total >= z.target}
									<li class="min-w-0">
										<div class="flex items-baseline justify-between gap-3 text-sm">
											<span class="flex min-w-0 items-center gap-1.5 truncate font-medium text-neutral-800 dark:text-neutral-200">
												{#if selesai}<Check size={14} class="shrink-0 text-success" aria-label="Target tercapai" />{/if}
												<span class="truncate">{z.latin}</span>
											</span>
											<span class="shrink-0 tabular-nums text-neutral-500"><b class="font-semibold text-neutral-900 dark:text-white">{angka(z.total)}</b>{#if z.target}<span class="text-xs"> / {angka(z.target)}</span>{/if}</span>
										</div>
										<div class="mt-1.5 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
											<div class="h-full rounded-full {selesai ? 'bg-success' : 'bg-brand-500 dark:bg-brand-400'}" style="width:{z.target ? Math.min((z.total / z.target) * 100, 100) : 100}%"></div>
										</div>
									</li>
								{/each}
							</ul>
							{#if dzikirHariIni.length > 6}<p class="mt-2 text-xs text-neutral-500">+{dzikirHariIni.length - 6} dzikir lain</p>{/if}
						{:else}
							<p class="rounded-xl bg-neutral-50 px-4 py-5 text-sm text-neutral-500 @4xl:mt-4 dark:bg-neutral-800/50 dark:text-neutral-400">Belum ada dzikir hari ini. Satu untaian 33 butir sudah cukup untuk memulai.</p>
						{/if}
					</div>

					<div class="grid grid-cols-2 gap-3 border-t border-neutral-100 pt-5 @4xl:grid-cols-1 @4xl:border-t-0 @4xl:border-l @4xl:pt-0 @4xl:pl-6 dark:border-neutral-800">
						<div class="flex items-center gap-3">
							<span class="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-warning/10 text-warning"><Flame size={20} aria-hidden="true" /></span>
							<div>
								<p class="text-2xl leading-none font-semibold tabular-nums text-neutral-900 dark:text-white">{streak}<span class="ml-1 text-sm font-medium text-neutral-500">hari</span></p>
								<p class="mt-1 text-[11px] text-neutral-500">Istiqamah · terpanjang {d.streak.terpanjang}</p>
							</div>
						</div>
						<div>
							<div class="flex items-baseline justify-between">
								<p class="text-sm font-semibold text-neutral-900 dark:text-white">Level {d.level.level}</p>
								<p class="text-[11px] tabular-nums text-neutral-500">{Math.round(levelPct * 100)}%</p>
							</div>
							<div class="mt-1.5 h-2 rounded-full bg-neutral-100 dark:bg-neutral-800" role="progressbar" aria-valuenow={Math.round(levelPct * 100)} aria-valuemin="0" aria-valuemax="100" aria-label="Progres level">
								<div class="h-full rounded-full bg-secondary-500 dark:bg-secondary-400" style="width:{levelPct * 100}%"></div>
							</div>
							<p class="mt-1 text-[11px] tabular-nums text-neutral-500">{angka(d.level.total)}{d.level.ambang_berikut ? ` / ${angka(d.level.ambang_berikut)} total` : " · level tertinggi"}</p>
						</div>
					</div>
				</div>
			</section>

			<!-- Ringkasan periode: angka pendukung, sengaja lebih tenang dari hero -->
			<dl class="mt-5 grid grid-cols-2 gap-x-6 gap-y-4 px-1 @xl:grid-cols-3 @4xl:grid-cols-6" aria-label="Ringkasan {labelRentang}">
				<div>
					<dt class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Rata-rata / hari</dt>
					<dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(d.rata_rata)}</dd>
				</div>
				<div>
					<dt class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Hari aktif</dt>
					<dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{hariAktif}<span class="text-sm font-normal text-neutral-400"> / {d.rentang}</span></dd>
				</div>
				<div>
					<dt class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Sesi</dt>
					<dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(d.sesi.jumlah)}<span class="text-sm font-normal text-neutral-400"> · {pctTuntas}% tuntas</span></dd>
				</div>
				<div>
					<dt class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Waktu berdzikir</dt>
					<dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{d.sesi.durasi_detik ? durasi(d.sesi.durasi_detik) : "—"}</dd>
				</div>
				<div>
					<dt class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Tempo</dt>
					<dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{d.sesi.per_menit ? angka(d.sesi.per_menit) : "—"}<span class="text-sm font-normal text-neutral-400"> /menit</span></dd>
				</div>
				<div>
					<dt class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Hari terbaik</dt>
					<dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">
						{#if d.hari_terbaik}{angka(d.hari_terbaik.total)}<span class="text-sm font-normal text-neutral-400"> · {tglPendek(d.hari_terbaik.tanggal)}</span>{:else}—{/if}
					</dd>
				</div>
			</dl>

			<!-- Tren + jam dzikir -->
			<div class="mt-6 grid gap-4 @4xl:grid-cols-[minmax(0,1fr)_20rem]">
				<section class={panel}>
					<div class="mb-4 flex flex-wrap items-end justify-between gap-3">
						<div>
							<h2 class={judul}>Tren harian</h2>
							<p class={sub}>Batang gelap = target harian tercapai</p>
						</div>
						<div class="@xl:text-right">
							<p class="text-[11px] font-medium tracking-wide text-neutral-500 uppercase">Total {labelRentang}</p>
							<p class="flex items-center gap-2 @xl:justify-end">
								<span class="text-[1.75rem] leading-tight font-semibold tracking-tight tabular-nums text-neutral-900 dark:text-white">{angka(d.total_rentang)}</span>
								{#if delta !== null}
									<span
										class="inline-flex items-center gap-0.5 rounded-full px-2 py-0.5 text-xs font-medium tabular-nums {delta >= 0 ? 'bg-success/10 text-success' : 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300'}"
										title="Dibanding {labelRentang} sebelumnya"
									>
										{#if delta >= 0}<ArrowUpRight size={13} aria-hidden="true" />{:else}<ArrowDownRight size={13} aria-hidden="true" />{/if}{delta > 0 ? "+" : ""}{delta}%
									</span>
								{/if}
							</p>
						</div>
					</div>
					<DailyBar data={d.harian} target={target_harian} />
				</section>
				<section class={panel}>
					<h2 class={judul}>Jam dzikir</h2>
					<p class="{sub} mb-3">Sebaran per jam (WIB) · pita amber = waktu dzikir pagi & petang</p>
					<JamDzikir data={d.per_jam} />
				</section>
			</div>

			<!-- Komposisi, pola pekan, paket -->
			<div class="mt-4 grid gap-4 @2xl:grid-cols-2 @4xl:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)_minmax(0,0.85fr)]">
				<section class={panel}>
					<h2 class="{judul} mb-4">Komposisi dzikir</h2>
					{#if d.komposisi.length}
						<KomposisiBar data={d.komposisi} />
					{:else}
						<p class="py-6 text-center text-sm text-neutral-500">Belum ada data di rentang ini.</p>
					{/if}
				</section>
				<section class={panel}>
					<h2 class={judul}>Pola pekanan</h2>
					<p class="{sub} mb-4">Rata-rata per hari · {labelRentang}</p>
					<PolaPekan data={d.harian} />
				</section>
				<section class="{panel} @2xl:col-span-2 @4xl:col-span-1">
					<h2 class={judul}>Paket wirid tuntas</h2>
					<p class="{sub} mb-4">Putaran lengkap · {labelRentang}</p>
					{#if d.paket.length}
						<ul class="space-y-3">
							{#each d.paket as p (p.paket_id)}
								<li class="flex items-center gap-3">
									<span class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-950 dark:text-brand-300"><ListOrdered size={16} aria-hidden="true" /></span>
									<span class="min-w-0 flex-1 truncate text-sm text-neutral-700 dark:text-neutral-300">{p.nama}</span>
									<span class="text-lg font-semibold tabular-nums {p.selesai ? 'text-neutral-900 dark:text-white' : 'text-neutral-300 dark:text-neutral-600'}">{p.selesai}<span class="ml-0.5 text-xs font-normal text-neutral-400">×</span></span>
								</li>
							{/each}
						</ul>
					{:else}
						<p class="py-6 text-center text-sm text-neutral-500">Belum ada paket wirid.</p>
					{/if}
				</section>
			</div>

			<!-- Kalender istiqamah -->
			<section class="{panel} mt-4">
				<div class="mb-3 flex flex-wrap items-end justify-between gap-3">
					<div>
						<h2 class={judul}>Kalender istiqamah</h2>
						<p class={sub}>12 bulan terakhir · satu butir = satu hari</p>
					</div>
					<dl class="flex gap-6 text-right">
						<div>
							<dt class="text-[11px] text-neutral-500">Hari aktif</dt>
							<dd class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{hariAktifTahun}<span class="font-normal text-neutral-400"> / 365</span></dd>
						</div>
						<div>
							<dt class="text-[11px] text-neutral-500">Target tercapai</dt>
							<dd class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{hariTercapaiTahun} hari</dd>
						</div>
						{#if d.rekor_tahun}
							<div>
								<dt class="flex items-center justify-end gap-1 text-[11px] text-neutral-500"><Trophy size={11} aria-hidden="true" /> Rekor</dt>
								<dd class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white" title={tglPanjang(d.rekor_tahun.tanggal)}>{angka(d.rekor_tahun.total)}<span class="font-normal text-neutral-400"> · {tglPendek(d.rekor_tahun.tanggal)}</span></dd>
							</div>
						{/if}
					</dl>
				</div>
				<Heatmap data={d.heatmap} target={target_harian} />
			</section>
		{/if}

		<section class="{panel} mt-4">
			<h2 class="{judul} mb-1">Pencapaian</h2>
			<BadgeGrid {badges} />
		</section>

		<section class="{panel} mt-4">
			<h2 class="{judul} mb-3">Riwayat sesi</h2>
			{#key riwayat}
				<RiwayatList awal={riwayat.items} adaLagiAwal={riwayat.ada_lagi} {dzikir} {lokal} />
			{/key}
		</section>
	</div>

	<BadgeToast badges={badgeToast} ondismiss={(k) => (badgeToast = badgeToast.filter((b) => b.kode !== k))} />
</AppLayout>

<style>
	.press {
		transition: transform 140ms cubic-bezier(0.23, 1, 0.32, 1), background-color 150ms ease;
	}
	.press:hover {
		background-color: var(--color-brand-700);
	}
	:global(.dark) .press:hover {
		background-color: var(--color-brand-400);
	}
	.press:active {
		transform: scale(0.97);
	}
	@media (prefers-reduced-motion: reduce) {
		.press {
			transition: background-color 150ms ease;
		}
	}
</style>
