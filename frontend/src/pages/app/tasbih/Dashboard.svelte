<script lang="ts">
	import { onMount } from "svelte";
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { ArrowDownRight, ArrowUpRight, Check, ChevronRight, CircleDot, CloudUpload, Flame, Info, Play, Trophy } from "lucide-svelte";
	import BeadRing from "@components/tasbih/BeadRing.svelte";
	import ProgressRing from "@components/tasbih/ProgressRing.svelte";
	import DailyBar from "@components/tasbih/DailyBar.svelte";
	import Heatmap from "@components/tasbih/Heatmap.svelte";
	import HourBar from "@components/tasbih/HourBar.svelte";
	import KomposisiBar from "@components/tasbih/KomposisiBar.svelte";
	import BadgeGrid from "@components/tasbih/BadgeGrid.svelte";
	import BadgeToast from "@components/tasbih/BadgeToast.svelte";
	import SyncIndicator from "@components/tasbih/SyncIndicator.svelte";
	import { angka, tglPanjang, tglPendek } from "@components/tasbih/format";
	import { insightDzikir } from "@lib/tasbih/insight";
	import { antreanSemua, setPemilik } from "@lib/tasbih/db";
	import { syncSekarang } from "@lib/tasbih/sync.svelte";
	import { tanggalWIB } from "@lib/tasbih/types";
	import type { Badge, DashboardData, Dzikir, SesiTertutup } from "@lib/tasbih/types";
	import type { User } from "@lib/types";

	interface Props {
		user?: User;
		dashboard: DashboardData;
		badges: Badge[];
		dzikir: Dzikir[];
		target_harian: number;
		badge_baru: Badge[];
	}
	let { user, dashboard, badges, dzikir, target_harian, badge_baru }: Props = $props();

	if (user?.id) setPemilik(user.id);

	let lokal = $state<SesiTertutup[]>([]);
	let badgeToast = $state<Badge[]>([...(badge_baru ?? [])]);

	async function bacaLokal() {
		try {
			lokal = await antreanSemua();
		} catch {
			lokal = [];
		}
	}

	const RENTANG = [
		{ v: 7, l: "7H", panjang: "7 hari" },
		{ v: 30, l: "30H", panjang: "30 hari" },
		{ v: 90, l: "90H", panjang: "90 hari" },
		{ v: 365, l: "1T", panjang: "1 tahun" },
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
	const hariTercapai = $derived(d.harian.filter((h) => h.total >= target_harian).length);
	const hariAktifTahun = $derived(d.heatmap.filter((h) => h.total > 0).length);
	const hariTercapaiTahun = $derived(d.heatmap.filter((h) => h.total >= target_harian).length);
	const labelRentang = $derived(RENTANG.find((r) => r.v === d.rentang)?.panjang ?? `${d.rentang} hari`);
	const insight = $derived(insightDzikir(d.per_jam, d.heatmap));
	const namaHari = $derived(new Intl.DateTimeFormat("id-ID", { weekday: "long" }).format(new Date(d.hari_ini.tanggal + "T00:00:00")));

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
		if (paksaReload || (h?.diterima.length ?? 0) > 0) router.reload({ only: ["dashboard", "badges"] });
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

	const kartu = "min-w-0 rounded-[1.375rem] border border-neutral-200 bg-white p-[1.125rem] dark:border-brand-200/10 dark:bg-brand-950/40";
	const judul = "text-[15px] font-semibold text-neutral-900 dark:text-white";
	const sub = "text-xs text-neutral-500 dark:text-brand-100/45";
</script>

<AppLayout {user}>
	<div class="@container mx-auto w-full min-w-0 max-w-xl pb-8 lg:max-w-5xl">
		<header class="mb-4 flex items-end justify-between gap-3">
			<div class="min-w-0">
				<h1 class="text-[22px] font-semibold tracking-tight text-neutral-900 dark:text-white">Progres</h1>
				<p class="mt-0.5 truncate text-xs text-neutral-500 dark:text-brand-100/45">{tglPanjang(d.hari_ini.tanggal)}{#if tanggalHijri} · {tanggalHijri}{/if}</p>
			</div>
			<SyncIndicator />
		</header>

		{#if lokal.length > 0}
			<p class="mb-4 flex items-center gap-2 rounded-xl bg-warning/10 px-4 py-2.5 text-xs font-medium text-warning">
				<CloudUpload size={14} aria-hidden="true" />
				{lokal.length} sesi menunggu sinkron. Grafik diperbarui setelah tersimpan di server.
			</p>
		{/if}

		{#if kosong}
			<section class="{kartu} px-6 py-14 text-center">
				<div class="mx-auto w-fit">
					<BeadRing progress={0} size={120}>
						<CircleDot size={26} class="text-brand-600 dark:text-brand-300" aria-hidden="true" />
					</BeadRing>
				</div>
				<h2 class="mt-4 text-lg font-semibold text-neutral-900 dark:text-white">Belum ada dzikir tercatat</h2>
				<p class="mx-auto mt-1 max-w-sm text-sm text-pretty text-neutral-500 dark:text-brand-100/50">Mulai sesi pertama di tasbih. Progres, pola waktu, dan pencapaian akan muncul di sini.</p>
				<a href="/app/tasbih" use:inertia class="press mt-6 inline-flex h-12 items-center gap-2 rounded-xl bg-brand-600 px-6 text-sm font-semibold text-white dark:bg-brand-300 dark:text-brand-950"><Play size={16} aria-hidden="true" /> Mulai dzikir</a>
			</section>
		{:else}
			<div class="grid gap-3 lg:grid-cols-2 lg:items-start">
				<div class="grid min-w-0 gap-3">
					<!-- Hari ini: jawaban untuk "sudah sampai mana hari ini?" -->
					<section class={kartu} aria-label="Hari ini">
						<div class="flex items-center gap-4">
							<ProgressRing progress={target_harian ? hariIni / target_harian : 0} size={112} stroke={9} {tercapai}>
								<span class="text-base font-semibold tabular-nums text-neutral-900 dark:text-white">{target_harian ? Math.min(Math.round((hariIni / target_harian) * 100), 999) : 0}%</span>
							</ProgressRing>
							<div class="min-w-0">
								<p class="text-[11px] font-semibold tracking-[0.09em] text-neutral-500 uppercase dark:text-brand-100/45">Hari ini · {namaHari}</p>
								<p class="mt-0.5 text-[1.875rem] leading-tight font-semibold tracking-tight tabular-nums text-neutral-900 dark:text-white" data-testid="dash-hari-ini">
									{angka(hariIni)}<span class="text-[15px] font-medium text-neutral-400 dark:text-brand-100/45"> / {angka(target_harian)}</span>
								</p>
								<p class="mt-1 flex items-center gap-1 text-[13px] font-medium text-sand-600 dark:text-sand-300">
									<Flame size={14} aria-hidden="true" />{streak} hari beruntun
								</p>
								<p class="text-xs text-neutral-400 dark:text-brand-100/40">Terpanjang {d.streak.terpanjang} hari</p>
							</div>
						</div>

						{#if dzikirHariIni.length}
							<ul class="mt-4 grid gap-2.5" aria-label="Dzikir hari ini">
								{#each dzikirHariIni.slice(0, 4) as z (z.id)}
									{@const selesai = z.target > 0 && z.total >= z.target}
									<li class="min-w-0">
										<div class="flex items-baseline justify-between gap-3 text-[13px]">
											<span class="truncate text-neutral-600 dark:text-brand-100/70">{z.latin}</span>
											<span class="flex shrink-0 items-center gap-1 tabular-nums {selesai ? 'text-success' : 'text-neutral-500 dark:text-brand-100/60'}">
												{angka(z.total)}{#if z.target && !selesai}<span> / {angka(z.target)}</span>{/if}{#if selesai}<Check size={13} aria-label="Target tercapai" />{/if}
											</span>
										</div>
										<div class="mt-1 h-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-brand-950">
											<div class="h-full rounded-full {selesai ? 'bg-success' : 'bg-brand-500 dark:bg-brand-300'}" style="width:{z.target ? Math.min((z.total / z.target) * 100, 100) : 100}%"></div>
										</div>
									</li>
								{/each}
							</ul>
							{#if dzikirHariIni.length > 4}<p class="mt-2 text-xs text-neutral-500 dark:text-brand-100/45">+{dzikirHariIni.length - 4} dzikir lain</p>{/if}
						{:else}
							<p class="mt-4 text-sm text-neutral-500 dark:text-brand-100/50">Belum ada dzikir hari ini. Satu untaian 33 butir sudah cukup untuk memulai.</p>
						{/if}

						{#if !tercapai}
							<a href="/app/tasbih" use:inertia class="press mt-4 flex h-11 items-center justify-center gap-1.5 rounded-xl bg-brand-600 text-sm font-semibold text-white dark:bg-brand-300 dark:text-brand-950">
								<Play size={14} aria-hidden="true" /> Lanjut dzikir · {angka(sisa)} lagi
							</a>
						{/if}
					</section>

					<!-- Rentang hanya memengaruhi bagian di bawahnya -->
					<div class="grid grid-cols-4 gap-1 rounded-[0.875rem] border border-neutral-200 bg-white p-1 dark:border-brand-200/10 dark:bg-brand-950/40" role="group" aria-label="Rentang waktu">
						{#each RENTANG as r}
							<button
								type="button"
								onclick={() => gantiRentang(r.v)}
								aria-pressed={d.rentang === r.v}
								aria-label={r.panjang}
								class="h-9 rounded-[0.625rem] text-[13px] font-semibold transition-colors {d.rentang === r.v ? 'bg-neutral-100 text-neutral-900 dark:bg-brand-900/60 dark:text-white' : 'text-neutral-500 hover:text-neutral-800 dark:text-brand-100/45 dark:hover:text-white'}"
							>{r.l}</button>
						{/each}
					</div>

					<section class={kartu}>
						<div class="mb-3 flex items-baseline justify-between gap-3">
							<h2 class={judul}>{labelRentang} terakhir</h2>
							{#if delta !== null}
								<span class="inline-flex items-center gap-0.5 text-xs tabular-nums {delta >= 0 ? 'text-success' : 'text-neutral-500 dark:text-brand-100/50'}" title="Dibanding {labelRentang} sebelumnya">
									{#if delta >= 0}<ArrowUpRight size={13} aria-hidden="true" />{:else}<ArrowDownRight size={13} aria-hidden="true" />{/if}{delta > 0 ? "+" : ""}{delta}% dari sebelumnya
								</span>
							{/if}
						</div>
						<DailyBar data={d.harian} target={target_harian} />
						<dl class="mt-3 grid grid-cols-3 divide-x divide-neutral-100 dark:divide-brand-200/10">
							<div class="pr-3"><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(d.total_rentang)}</dd><dt class={sub}>Total</dt></div>
							<div class="px-3"><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{hariAktif}<span class="text-sm font-normal text-neutral-400 dark:text-brand-100/40">/{d.rentang}</span></dd><dt class={sub}>Hari aktif</dt></div>
							<div class="pl-3"><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{hariTercapai}<span class="text-sm font-normal text-neutral-400 dark:text-brand-100/40">/{d.rentang}</span></dd><dt class={sub}>Target tercapai</dt></div>
						</dl>
					</section>

					{#if insight}
						<p class="flex items-start gap-3 rounded-2xl bg-brand-50 px-4 py-3.5 text-[13.5px] leading-relaxed text-neutral-800 dark:bg-brand-300/10 dark:text-brand-50" data-testid="insight">
							<Info size={16} class="mt-0.5 shrink-0 text-brand-600 dark:text-brand-300" aria-hidden="true" />
							<span>{insight}</span>
						</p>
					{/if}
				</div>

				<div class="grid min-w-0 gap-3">
					<section class={kartu}>
						<div class="mb-3 flex items-baseline justify-between gap-3">
							<h2 class={judul}>Kalender istiqamah</h2>
							<span class={sub}>12 bulan</span>
						</div>
						<Heatmap data={d.heatmap} target={target_harian} />
						<dl class="mt-3 flex flex-wrap gap-x-6 gap-y-2 border-t border-neutral-100 pt-3 dark:border-brand-200/10">
							<div><dt class={sub}>Hari aktif</dt><dd class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{hariAktifTahun}<span class="font-normal text-neutral-400"> / 365</span></dd></div>
							<div><dt class={sub}>Target tercapai</dt><dd class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{hariTercapaiTahun} hari</dd></div>
							{#if d.rekor_tahun}
								<div><dt class="{sub} flex items-center gap-1"><Trophy size={11} aria-hidden="true" /> Rekor</dt><dd class="text-sm font-semibold tabular-nums text-neutral-900 dark:text-white" title={tglPanjang(d.rekor_tahun.tanggal)}>{angka(d.rekor_tahun.total)}<span class="font-normal text-neutral-400"> · {tglPendek(d.rekor_tahun.tanggal)}</span></dd></div>
							{/if}
						</dl>
					</section>

					<section class={kartu}>
						<div class="mb-2 flex items-baseline justify-between gap-3">
							<h2 class={judul}>Waktu berdzikir</h2>
							<span class={sub}>WIB · {labelRentang}</span>
						</div>
						<HourBar data={d.per_jam} tinggi={100} />
					</section>

					<section class={kartu}>
						<div class="mb-3 flex items-baseline justify-between gap-3">
							<h2 class={judul}>Komposisi</h2>
							<span class={sub}>{labelRentang}</span>
						</div>
						{#if d.komposisi.length}
							<KomposisiBar data={d.komposisi} />
						{:else}
							<p class="py-4 text-center text-sm text-neutral-500">Belum ada data di rentang ini.</p>
						{/if}
					</section>

					<section class={kartu} aria-label="Level dan pencapaian">
						<div class="flex items-center gap-3.5">
							<span class="grid h-12 w-12 shrink-0 place-items-center rounded-2xl bg-sand-50 text-lg font-bold text-sand-600 dark:bg-sand-300/10 dark:text-sand-300">{d.level.level}</span>
							<div class="min-w-0 flex-1">
								<div class="flex items-baseline justify-between gap-2 text-[13px]">
									<span class="font-semibold text-neutral-900 dark:text-white">Level {d.level.level}</span>
									<span class="tabular-nums text-neutral-500 dark:text-brand-100/45">{angka(d.level.total)}{d.level.ambang_berikut ? ` / ${angka(d.level.ambang_berikut)}` : " · tertinggi"}</span>
								</div>
								<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-brand-950" role="progressbar" aria-valuenow={Math.round(levelPct * 100)} aria-valuemin="0" aria-valuemax="100" aria-label="Progres level">
									<div class="h-full rounded-full bg-sand-400 dark:bg-sand-300" style="width:{levelPct * 100}%"></div>
								</div>
							</div>
						</div>
						<div class="mt-4 border-t border-neutral-100 pt-4 dark:border-brand-200/10">
							<BadgeGrid {badges} />
						</div>
					</section>

					<a href="/app/tasbih/riwayat" use:inertia class="{kartu} flex items-center justify-between !py-4 text-[15px] font-semibold text-neutral-900 transition hover:border-brand-300 dark:text-white">
						Riwayat sesi
						<ChevronRight size={18} class="text-neutral-400" aria-hidden="true" />
					</a>
				</div>
			</div>
		{/if}
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
		background-color: var(--color-brand-200);
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
