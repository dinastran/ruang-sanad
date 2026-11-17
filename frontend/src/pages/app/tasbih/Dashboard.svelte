<script lang="ts">
	import { onMount } from "svelte";
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { ArrowDownRight, ArrowUpRight, CircleDot, CloudUpload, Flame, ListOrdered } from "lucide-svelte";
	import ProgressRing from "@components/tasbih/ProgressRing.svelte";
	import DailyBar from "@components/tasbih/DailyBar.svelte";
	import Heatmap from "@components/tasbih/Heatmap.svelte";
	import HourBar from "@components/tasbih/HourBar.svelte";
	import KomposisiBar from "@components/tasbih/KomposisiBar.svelte";
	import BadgeGrid from "@components/tasbih/BadgeGrid.svelte";
	import BadgeToast from "@components/tasbih/BadgeToast.svelte";
	import RiwayatList from "@components/tasbih/RiwayatList.svelte";
	import SyncIndicator from "@components/tasbih/SyncIndicator.svelte";
	import { angka } from "@components/tasbih/format";
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

	const d = $derived(dashboard);
	const lokalHariIni = $derived(lokal.filter((s) => tanggalWIB(s.mulai_at) === d.hari_ini.tanggal).reduce((a, s) => a + s.jumlah, 0));
	const hariIni = $derived(d.hari_ini.total + lokalHariIni);
	const tercapai = $derived(hariIni >= target_harian);
	const streak = $derived(d.streak.saat + (!d.streak.hari_ini_tercapai && tercapai ? 1 : 0));
	const delta = $derived(d.total_rentang_sebelum > 0 ? Math.round(((d.total_rentang - d.total_rentang_sebelum) / d.total_rentang_sebelum) * 100) : null);
	const levelPct = $derived(d.level.ambang_berikut > 0 ? Math.min((d.level.total - d.level.ambang_ini) / (d.level.ambang_berikut - d.level.ambang_ini), 1) : 1);
	const kosong = $derived(d.level.total === 0 && lokal.length === 0);
	const hariAktif = $derived(d.harian.filter((h) => h.total > 0).length);

	const RENTANG = [
		{ v: 7, l: "7 hari" },
		{ v: 30, l: "30 hari" },
		{ v: 90, l: "90 hari" },
		{ v: 365, l: "1 tahun" },
	];

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
</script>

<AppLayout {user}>
	<div class="mx-auto w-full min-w-0 max-w-6xl pb-10">
		<!-- Header -->
		<header class="mb-5 flex flex-wrap items-end justify-between gap-3">
			<div>
				<h1 class="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-white">Progres dzikir</h1>
				<div class="mt-1"><SyncIndicator /></div>
			</div>
			<div class="flex w-full items-center gap-2 sm:w-auto">
				<div class="grid flex-1 grid-cols-4 gap-1 rounded-xl bg-neutral-100 p-1 sm:flex-none dark:bg-neutral-800" role="group" aria-label="Rentang waktu">
					{#each RENTANG as r}
						<button
							type="button"
							onclick={() => gantiRentang(r.v)}
							aria-pressed={d.rentang === r.v}
							class="h-9 rounded-lg px-3 text-xs font-medium whitespace-nowrap transition {d.rentang === r.v ? 'bg-white text-neutral-900 shadow-sm dark:bg-neutral-950 dark:text-white' : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400'}"
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
				<span class="mx-auto grid h-14 w-14 place-items-center rounded-full bg-brand-50 text-brand-600 dark:bg-brand-950 dark:text-brand-300"><CircleDot size={26} aria-hidden="true" /></span>
				<h2 class="mt-4 text-lg font-semibold text-neutral-900 dark:text-white">Belum ada dzikir tercatat</h2>
				<p class="mx-auto mt-1 max-w-sm text-sm text-neutral-500 dark:text-neutral-400">Mulai sesi pertama di tasbih. Progres, riwayat, dan pencapaian akan muncul di sini.</p>
				<a href="/app/tasbih" use:inertia class="mt-6 inline-flex h-12 items-center rounded-xl bg-brand-600 px-6 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950">Mulai dzikir</a>
			</section>
		{:else}
			<!-- Hero: hari ini (focal point) -->
			<section class="grid gap-4 rounded-3xl border border-neutral-200 bg-white p-5 sm:grid-cols-[auto_1fr] sm:p-6 dark:border-neutral-800 dark:bg-neutral-900" aria-label="Hari ini">
				<div class="flex items-center gap-5">
					<ProgressRing progress={target_harian ? hariIni / target_harian : 0} tercapai={tercapai} size={132} stroke={10}>
						<div class="text-center">
							<span class="block text-2xl font-bold tabular-nums text-neutral-900 dark:text-white" data-testid="dash-hari-ini">{angka(hariIni)}</span>
							<span class="block text-[11px] tabular-nums text-neutral-500">/ {angka(target_harian)}</span>
						</div>
					</ProgressRing>
					<div class="sm:hidden">
						<p class="text-xs font-medium text-neutral-500">Hari ini</p>
						<p class="mt-1 text-sm text-neutral-700 dark:text-neutral-300">{tercapai ? "Target harian tercapai" : `${angka(Math.max(target_harian - hariIni, 0))} lagi menuju target`}</p>
					</div>
				</div>
				<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 sm:items-center">
					<div class="hidden sm:block">
						<p class="text-xs font-medium text-neutral-500">Hari ini</p>
						<p class="mt-1 text-sm text-neutral-700 dark:text-neutral-300">{tercapai ? "Target harian tercapai" : `${angka(Math.max(target_harian - hariIni, 0))} lagi menuju target`}</p>
					</div>
					<div class="flex items-center gap-3">
						<span class="grid h-11 w-11 place-items-center rounded-full bg-warning/10 text-warning"><Flame size={20} aria-hidden="true" /></span>
						<div>
							<p class="text-2xl font-bold tabular-nums leading-none text-neutral-900 dark:text-white">{streak}<span class="ml-1 text-sm font-medium text-neutral-500">hari</span></p>
							<p class="mt-1 text-[11px] text-neutral-500">Streak · terpanjang {d.streak.terpanjang}</p>
						</div>
					</div>
					<div>
						<p class="text-sm font-semibold text-neutral-900 dark:text-white">Level {d.level.level}</p>
						<div class="mt-1.5 h-2 rounded-full bg-neutral-100 dark:bg-neutral-800" role="progressbar" aria-valuenow={Math.round(levelPct * 100)} aria-valuemin="0" aria-valuemax="100" aria-label="Progres level">
							<div class="h-full rounded-full bg-secondary-500" style="width:{levelPct * 100}%"></div>
						</div>
						<p class="mt-1 text-[11px] tabular-nums text-neutral-500">{angka(d.level.total)}{d.level.ambang_berikut ? ` / ${angka(d.level.ambang_berikut)}` : " · level tertinggi"}</p>
					</div>
				</div>
			</section>

			<!-- Statistik rentang -->
			<dl class="mt-4 grid grid-cols-2 gap-x-6 gap-y-4 px-1 sm:grid-cols-4">
				<div>
					<dt class="text-xs text-neutral-500">Total {RENTANG.find((r) => r.v === d.rentang)?.l}</dt>
					<dd class="mt-0.5 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(d.total_rentang)}</dd>
				</div>
				<div>
					<dt class="text-xs text-neutral-500">Rata-rata per hari</dt>
					<dd class="mt-0.5 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(d.rata_rata)}</dd>
				</div>
				<div>
					<dt class="text-xs text-neutral-500">Hari aktif</dt>
					<dd class="mt-0.5 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{hariAktif}<span class="text-sm font-normal text-neutral-400"> / {d.rentang}</span></dd>
				</div>
				<div>
					<dt class="text-xs text-neutral-500">Dibanding periode sebelumnya</dt>
					<dd class="mt-0.5 flex items-center gap-1 text-xl font-semibold tabular-nums {delta === null ? 'text-neutral-400' : delta >= 0 ? 'text-success' : 'text-neutral-700 dark:text-neutral-300'}">
						{#if delta === null}—{:else}{#if delta >= 0}<ArrowUpRight size={18} aria-hidden="true" />{:else}<ArrowDownRight size={18} aria-hidden="true" />{/if}{delta > 0 ? "+" : ""}{delta}%{/if}
					</dd>
				</div>
			</dl>

			<div class="mt-6 grid gap-4 lg:grid-cols-3">
				<section class="min-w-0 rounded-2xl border border-neutral-200 bg-white p-5 lg:col-span-2 dark:border-neutral-800 dark:bg-neutral-900">
					<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Tren harian</h2>
					<p class="mb-3 text-xs text-neutral-500">Batang gelap = target harian tercapai</p>
					<DailyBar data={d.harian} target={target_harian} />
				</section>
				<section class="min-w-0 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
					<h2 class="mb-4 text-sm font-semibold text-neutral-900 dark:text-white">Komposisi dzikir</h2>
					{#if d.komposisi.length}
						<KomposisiBar data={d.komposisi} />
					{:else}
						<p class="py-6 text-center text-sm text-neutral-500">Belum ada data di rentang ini.</p>
					{/if}
				</section>
			</div>

			<section class="mt-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
				<h2 class="mb-3 text-sm font-semibold text-neutral-900 dark:text-white">Kalender istiqamah · 12 bulan</h2>
				<Heatmap data={d.heatmap} target={target_harian} />
			</section>

			<div class="mt-4 grid gap-4 lg:grid-cols-3">
				<section class="min-w-0 rounded-2xl border border-neutral-200 bg-white p-5 lg:col-span-2 dark:border-neutral-800 dark:bg-neutral-900">
					<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Waktu berdzikir</h2>
					<p class="mb-3 text-xs text-neutral-500">Jumlah per jam (WIB) · batang gelap = jam paling sering</p>
					<HourBar data={d.per_jam} />
				</section>
				<section class="min-w-0 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
					<h2 class="mb-4 text-sm font-semibold text-neutral-900 dark:text-white">Paket wirid tuntas</h2>
					<ul class="space-y-3">
						{#each d.paket as p (p.paket_id)}
							<li class="flex items-center gap-3">
								<span class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-950 dark:text-brand-300"><ListOrdered size={16} aria-hidden="true" /></span>
								<span class="min-w-0 flex-1 truncate text-sm text-neutral-700 dark:text-neutral-300">{p.nama}</span>
								<span class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{p.selesai}<span class="ml-0.5 text-xs font-normal text-neutral-400">×</span></span>
							</li>
						{/each}
					</ul>
				</section>
			</div>
		{/if}

		<section class="mt-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
			<h2 class="mb-1 text-sm font-semibold text-neutral-900 dark:text-white">Pencapaian</h2>
			<BadgeGrid {badges} />
		</section>

		<section class="mt-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
			<h2 class="mb-3 text-sm font-semibold text-neutral-900 dark:text-white">Riwayat sesi</h2>
			{#key riwayat}
				<RiwayatList awal={riwayat.items} adaLagiAwal={riwayat.ada_lagi} {dzikir} {lokal} />
			{/key}
		</section>
	</div>

	<BadgeToast badges={badgeToast} ondismiss={(k) => (badgeToast = badgeToast.filter((b) => b.kode !== k))} />
</AppLayout>
