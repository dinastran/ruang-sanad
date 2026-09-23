<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import BarChart from "@components/charts/BarChart.svelte";
	import LineChart from "@components/charts/LineChart.svelte";
	import DonutChart from "@components/charts/DonutChart.svelte";
	import type { User } from "@lib/types";
	import { Users, UserCheck, UserX, AlertCircle, XCircle, BarChart3, CalendarClock, Database, RotateCcw, ChevronLeft, ChevronRight, ArrowRight } from "lucide-svelte";

	interface MasterItem { id: number; kode: string; keterangan?: string; nama?: string }
	interface AnalyticsFilters {
		date_from: string; date_to: string; angkatan_pendaftaran: string; angkatan_kelas: string; level: string; tipe: string;
		segment_age_bucket: string; segment_gender: string; segment_domisili: string; segment_status: string;
		segment_registration_month: string; page: number; limit: number;
	}
	interface Point { label: string; value: string; total: number }
	interface Growth { month: string; total: number }
	interface Cohort { angkatan: string; aktif: number; cuti: number; nonaktif: number; tidak_lanjut: number; lainnya: number; total: number }
	interface Row {
		id: number; id_mahasantri: string; nama: string; usia: number; jenis_kelamin: string; domisili: string;
		tanggal_daftar: string; angkatan: string; angkatan_kelas: string; level: string; tipe: string; status: string;
	}
	interface Data {
		summary: { total: number; aktif: number; cuti: number; nonaktif: number; tidak_lanjut: number; active_rate: number; rata_rata_usia: number };
		growth: Growth[]; age_distribution: Point[]; gender_distribution: Point[]; domisili_distribution: Point[];
		status_distribution: Point[]; status_by_angkatan: Cohort[]; level_distribution: Point[]; tipe_distribution: Point[];
		data_quality: { total: number; usia_kosong: number; domisili_kosong: number; tanggal_daftar_kosong: number; jenis_kelamin_kosong: number };
		rows: Row[]; row_total: number;
	}
	interface Props { user?: User; filters: AnalyticsFilters; analytics: Data; angkatan?: MasterItem[]; levels?: MasterItem[]; tipes?: string[]; error?: string }

	let { user, filters, analytics, angkatan = [], levels = [], tipes = [], error }: Props = $props();
	let summary = $derived(analytics?.summary ?? { total: 0, aktif: 0, cuti: 0, nonaktif: 0, tidak_lanjut: 0, active_rate: 0, rata_rata_usia: 0 });
	let rowTotal = $derived(analytics?.row_total ?? 0);
	let page = $derived(filters?.page || 1);
	let limit = $derived(filters?.limit || 25);
	let totalPages = $derived(Math.max(1, Math.ceil(rowTotal / limit)));
	let startRow = $derived(rowTotal ? (page - 1) * limit + 1 : 0);
	let endRow = $derived(Math.min(page * limit, rowTotal));

	const ageLabels: Record<string, string> = { le17: "Usia ≤17", "18-24": "Usia 18-24", "25-34": "Usia 25-34", "35-44": "Usia 35-44", "45-54": "Usia 45-54", "55plus": "Usia 55+" };
	const genderLabels: Record<string, string> = { P: "Perempuan", L: "Laki-laki", unknown: "Gender tidak diketahui" };
	const statusLabels: Record<string, string> = { aktif: "Aktif", cuti: "Cuti", nonaktif: "Nonaktif", tidak_lanjut: "Tidak Lanjut" };
	let segmentChips = $derived([
		filters?.segment_age_bucket ? { key: "segment_age", label: ageLabels[filters.segment_age_bucket] || filters.segment_age_bucket } : null,
		filters?.segment_gender ? { key: "segment_gender", label: genderLabels[filters.segment_gender] || filters.segment_gender } : null,
		filters?.segment_domisili ? { key: "segment_domisili", label: `Domisili ${filters.segment_domisili}` } : null,
		filters?.segment_status ? { key: "segment_status", label: statusLabels[filters.segment_status] || filters.segment_status } : null,
		filters?.segment_registration_month ? { key: "segment_month", label: `Daftar ${formatMonth(filters.segment_registration_month)}` } : null,
	].filter(Boolean) as { key: string; label: string }[]);

	const segmentKeys = ["segment_age", "segment_gender", "segment_domisili", "segment_status", "segment_month"];

	function params() { return new URLSearchParams(typeof window !== "undefined" ? window.location.search : ""); }
	function navigate(p: URLSearchParams) { const query = p.toString(); router.get(`/app/analitik-santri${query ? `?${query}` : ""}`); }
	function applyFilters(event: SubmitEvent) {
		event.preventDefault();
		const data = new FormData(event.currentTarget as HTMLFormElement), p = new URLSearchParams();
		for (const [key, raw] of data.entries()) { const value = String(raw).trim(); if (value) p.set(key, value); }
		navigate(p);
	}
	function setSegment(key: string, value: string) { if (!value) return; const p = params(); p.delete("page"); p.set(key, value); navigate(p); }
	function setMainFilter(key: string, value: string) { if (!value) return; const p = params(); p.delete("page"); segmentKeys.forEach((x) => p.delete(x)); p.set(key, value); navigate(p); }
	function clearSegment(key: string) { const p = params(); p.delete("page"); p.delete(key); navigate(p); }
	function clearSegments() { const p = params(); [...segmentKeys, "page"].forEach((x) => p.delete(x)); navigate(p); }
	function reset() { router.get("/app/analitik-santri"); }
	function quickYear() {
		const p = params(), year = new Date().getFullYear();
		[...segmentKeys, "page"].forEach((x) => p.delete(x));
		p.set("date_from", `${year}-01-01`); p.set("date_to", `${year}-12-31`); navigate(p);
	}
	function goToPage(target: number) { if (target < 1 || target > totalPages) return; const p = params(); p.set("page", String(target)); navigate(p); }
	function selectCohortStatus(cohort: string, status: string) {
		const p = params(); p.delete("page"); if (cohort !== "Tanpa Angkatan") p.set("angkatan_pendaftaran", cohort); p.set("segment_status", status); navigate(p);
	}
	function formatMonth(month: string) {
		const [year, raw] = month.split("-"), index = Number(raw) - 1;
		if (!year || index < 0 || index > 11) return month;
		return new Intl.DateTimeFormat("id-ID", { month: "long", year: "numeric" }).format(new Date(Number(year), index, 1));
	}
	function pct(value: number, total: number) { return total > 0 ? value / total * 100 : 0; }
	function gender(value: string) { return genderLabels[value] || "Tidak diketahui"; }
	function status(value: string) { return statusLabels[value] || value || "-"; }
</script>

<AppLayout {user} group="analitik-santri">
	<div class="border-b border-neutral-200/80 pb-8 pt-8 dark:border-white/[0.04]">
		<div class="mx-auto max-w-7xl px-4 sm:px-6">
			<div class="flex flex-wrap items-start justify-between gap-4">
				<div>
					<div class="mb-3 flex items-center gap-2 text-sm text-neutral-500"><a href="/app" use:inertia class="hover:text-brand-600">Dashboard</a><span>/</span><span>Analisis Santri</span></div>
					<h1 class="text-2xl font-bold tracking-tight text-neutral-900 sm:text-3xl dark:text-white">Analisis Santri</h1>
					<p class="mt-2 text-neutral-600 dark:text-neutral-400">Profil, pertumbuhan, dan status santri Ruang Sanad.</p>
				</div>
				<div class="rounded-xl border border-neutral-200/80 bg-white px-4 py-3 text-right dark:border-white/[0.06] dark:bg-neutral-925/50">
					<p class="text-xs uppercase tracking-wide text-neutral-500">Rata-rata usia terisi</p>
					<p class="mt-1 text-xl font-bold text-neutral-900 dark:text-white">{summary.rata_rata_usia ? `${summary.rata_rata_usia.toFixed(1)} tahun` : "-"}</p>
				</div>
			</div>
		</div>
	</div>

	<div class="mx-auto max-w-7xl space-y-8 px-4 py-8 sm:px-6">
		{#if error}<div class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-400">{error}</div>{/if}

		<section class="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
			<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
				<div><h2 class="font-semibold text-neutral-900 dark:text-white">Filter Analisis</h2><p class="mt-1 text-xs text-neutral-500">Filter utama memengaruhi seluruh card dan grafik.</p></div>
				<div class="flex gap-2">
					<button type="button" onclick={reset} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 dark:border-white/[0.08] dark:text-neutral-300"><RotateCcw size="14" class="inline" /> Reset</button>
					<button type="button" onclick={quickYear} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 dark:border-white/[0.08] dark:text-neutral-300">Tahun ini</button>
				</div>
			</div>
			<form onsubmit={applyFilters} class="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
				<label class="space-y-1.5 text-xs text-neutral-600">Dari tanggal<input name="date_from" type="date" value={filters.date_from || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white" /></label>
				<label class="space-y-1.5 text-xs text-neutral-600">Sampai tanggal<input name="date_to" type="date" value={filters.date_to || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white" /></label>
				<label class="space-y-1.5 text-xs text-neutral-600">Angkatan pendaftaran<select name="angkatan_pendaftaran" value={filters.angkatan_pendaftaran || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option>{#each angkatan as item}<option value={item.kode}>{item.keterangan || item.kode}</option>{/each}</select></label>
				<label class="space-y-1.5 text-xs text-neutral-600">Angkatan kelas<select name="angkatan_kelas" value={filters.angkatan_kelas || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option>{#each angkatan as item}<option value={item.kode}>{item.keterangan || item.kode}</option>{/each}</select></label>
				<label class="space-y-1.5 text-xs text-neutral-600">Level<select name="level" value={filters.level || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option>{#each levels as item}<option value={item.kode}>{item.nama || item.kode}</option>{/each}</select></label>
				<div class="space-y-1.5"><label class="text-xs text-neutral-600">Tipe kelas</label><div class="flex gap-2"><select name="tipe" value={filters.tipe || ""} class="min-w-0 flex-1 rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option>{#each tipes as tipe}<option value={tipe}>{tipe}</option>{/each}</select><button type="submit" class="rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white">Terapkan</button></div></div>
			</form>
		</section>

		<section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-6">
			{#each [
				{ label: "Total Santri", value: summary.total, icon: Users },
				{ label: "Santri Aktif", value: summary.aktif, icon: UserCheck },
				{ label: "Active Rate", value: `${summary.active_rate.toFixed(1)}%`, icon: BarChart3 },
				{ label: "Cuti", value: summary.cuti, icon: AlertCircle },
				{ label: "Nonaktif", value: summary.nonaktif, icon: UserX },
				{ label: "Tidak Lanjut", value: summary.tidak_lanjut, icon: XCircle }
			] as card}
				{@const Icon = card.icon}
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
					<div class="mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-neutral-100 text-neutral-600 dark:bg-neutral-800"><Icon size="18" /></div>
					<p class="text-xs text-neutral-500">{card.label}</p>
					<p class="mt-1 font-mono text-2xl font-bold text-neutral-900 dark:text-white">{typeof card.value === "number" ? card.value.toLocaleString("id-ID") : card.value}</p>
				</div>
			{/each}
		</section>

		<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
			<div class="mb-5"><h2 class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><CalendarClock size="18" class="text-brand-500" /> Pertumbuhan Pendaftaran</h2><p class="mt-1 text-xs text-neutral-500">Klik titik untuk melihat santri pada bulan tersebut.</p></div>
			<LineChart points={analytics?.growth || []} onselect={(month) => setSegment("segment_month", month)} />
		</section>

		<section>
			<div class="mb-4"><h2 class="text-lg font-semibold text-neutral-900 dark:text-white">Demografi Santri</h2><p class="mt-1 text-sm text-neutral-500">Klik kategori untuk drill-down ke daftar santri.</p></div>
			<div class="grid gap-5 lg:grid-cols-2">
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><h3 class="mb-5 font-semibold text-neutral-900 dark:text-white">Distribusi Usia</h3><BarChart items={analytics?.age_distribution || []} onselect={(value) => setSegment("segment_age", value)} /></div>
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><h3 class="mb-5 font-semibold text-neutral-900 dark:text-white">Jenis Kelamin</h3><DonutChart items={analytics?.gender_distribution || []} onselect={(value) => setSegment("segment_gender", value)} /></div>
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 lg:col-span-2 dark:border-white/[0.06] dark:bg-neutral-925/50"><h3 class="font-semibold text-neutral-900 dark:text-white">Top Domisili</h3><p class="mb-5 mt-1 text-xs text-neutral-500">Menggunakan teks domisili existing tanpa normalisasi.</p><BarChart items={analytics?.domisili_distribution || []} onselect={(value) => setSegment("segment_domisili", value)} /></div>
			</div>
		</section>

		<section>
			<div class="mb-4"><h2 class="text-lg font-semibold text-neutral-900 dark:text-white">Status Santri</h2><p class="mt-1 text-sm text-neutral-500">Komposisi status saat ini, bukan historical retention rate.</p></div>
			<div class="grid gap-5 lg:grid-cols-2">
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><h3 class="mb-5 font-semibold text-neutral-900 dark:text-white">Komposisi Status</h3><DonutChart items={analytics?.status_distribution || []} onselect={(value) => statusLabels[value] && setSegment("segment_status", value)} /></div>
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
					<h3 class="font-semibold text-neutral-900 dark:text-white">Status per Angkatan Pendaftaran</h3><p class="mb-5 mt-1 text-xs text-neutral-500">Klik segmen untuk memilih angkatan dan status.</p>
					<div class="max-h-[360px] space-y-4 overflow-y-auto pr-1">
						{#each analytics?.status_by_angkatan || [] as cohort}
							<div><div class="mb-1.5 flex justify-between text-sm"><span class="font-medium text-neutral-700 dark:text-neutral-300">{cohort.angkatan}</span><span class="font-mono text-neutral-500">{cohort.total}</span></div>
								<div class="flex h-3 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
									{#if cohort.aktif}<button aria-label={`${cohort.angkatan} aktif ${cohort.aktif}`} onclick={() => selectCohortStatus(cohort.angkatan, "aktif")} style={`width:${pct(cohort.aktif, cohort.total)}%`} class="bg-green-500"></button>{/if}
									{#if cohort.cuti}<button aria-label={`${cohort.angkatan} cuti ${cohort.cuti}`} onclick={() => selectCohortStatus(cohort.angkatan, "cuti")} style={`width:${pct(cohort.cuti, cohort.total)}%`} class="bg-amber-500"></button>{/if}
									{#if cohort.nonaktif}<button aria-label={`${cohort.angkatan} nonaktif ${cohort.nonaktif}`} onclick={() => selectCohortStatus(cohort.angkatan, "nonaktif")} style={`width:${pct(cohort.nonaktif, cohort.total)}%`} class="bg-neutral-500"></button>{/if}
									{#if cohort.tidak_lanjut}<button aria-label={`${cohort.angkatan} tidak lanjut ${cohort.tidak_lanjut}`} onclick={() => selectCohortStatus(cohort.angkatan, "tidak_lanjut")} style={`width:${pct(cohort.tidak_lanjut, cohort.total)}%`} class="bg-red-500"></button>{/if}
									{#if cohort.lainnya}<div style={`width:${pct(cohort.lainnya, cohort.total)}%`} class="bg-violet-500"></div>{/if}
								</div>
							</div>
						{/each}
					</div>
					<div class="mt-5 flex flex-wrap gap-3 text-xs text-neutral-500"><span><i class="mr-1 inline-block h-2 w-2 rounded-full bg-green-500"></i>Aktif</span><span><i class="mr-1 inline-block h-2 w-2 rounded-full bg-amber-500"></i>Cuti</span><span><i class="mr-1 inline-block h-2 w-2 rounded-full bg-neutral-500"></i>Nonaktif</span><span><i class="mr-1 inline-block h-2 w-2 rounded-full bg-red-500"></i>Tidak lanjut</span></div>
				</div>
			</div>
		</section>

		<section>
			<div class="mb-4"><h2 class="text-lg font-semibold text-neutral-900 dark:text-white">Profil Pembelajaran</h2><p class="mt-1 text-sm text-neutral-500">Distribusi level dan tipe kelas pada populasi terpilih.</p></div>
			<div class="grid gap-5 lg:grid-cols-2">
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><h3 class="mb-5 font-semibold text-neutral-900 dark:text-white">Distribusi Level</h3><BarChart items={analytics?.level_distribution || []} onselect={(value) => setMainFilter("level", value)} /></div>
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><h3 class="mb-5 font-semibold text-neutral-900 dark:text-white">Distribusi Tipe Kelas</h3><BarChart items={analytics?.tipe_distribution || []} onselect={(value) => setMainFilter("tipe", value)} /></div>
			</div>
		</section>

		<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
			<div class="mb-5 flex items-center gap-2"><Database size="18" class="text-brand-500" /><div><h2 class="font-semibold text-neutral-900 dark:text-white">Kualitas Data</h2><p class="mt-1 text-xs text-neutral-500">Data kosong pada populasi yang sedang dianalisis.</p></div></div>
			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				{#each [
					{ label: "Usia kosong", value: analytics?.data_quality?.usia_kosong || 0 },
					{ label: "Domisili kosong", value: analytics?.data_quality?.domisili_kosong || 0 },
					{ label: "Tanggal daftar kosong", value: analytics?.data_quality?.tanggal_daftar_kosong || 0 },
					{ label: "Gender kosong/tidak valid", value: analytics?.data_quality?.jenis_kelamin_kosong || 0 }
				] as item}
					<div class="rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900/70"><p class="text-xs text-neutral-500">{item.label}</p><p class="mt-1 font-mono text-xl font-bold text-neutral-900 dark:text-white">{item.value.toLocaleString("id-ID")}</p></div>
				{/each}
			</div>
		</section>

		<section class="space-y-4">
			<div class="flex flex-wrap items-end justify-between gap-4"><div><h2 class="text-lg font-semibold text-neutral-900 dark:text-white">Santri dalam Segmen</h2><p class="mt-1 text-sm text-neutral-500">{rowTotal.toLocaleString("id-ID")} santri sesuai drill-down.</p></div>{#if segmentChips.length}<button type="button" onclick={clearSegments} class="text-xs font-semibold text-brand-600 dark:text-brand-400">Hapus semua drill-down</button>{/if}</div>
			{#if segmentChips.length}<div class="flex flex-wrap gap-2">{#each segmentChips as chip}<button type="button" onclick={() => clearSegment(chip.key)} class="rounded-full border border-brand-400/20 bg-brand-400/10 px-3 py-1.5 text-xs font-medium text-brand-700 dark:text-brand-300">{chip.label} ×</button>{/each}</div>{/if}
			<div class="overflow-hidden rounded-2xl border border-neutral-200/80 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="overflow-x-auto"><table class="w-full min-w-[1050px] text-sm">
					<thead><tr class="border-b border-neutral-200/80 bg-neutral-50 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:border-white/[0.04] dark:bg-neutral-900/60"><th class="px-4 py-3">No</th><th class="px-4 py-3">Santri</th><th class="px-4 py-3">Usia</th><th class="px-4 py-3">Gender</th><th class="px-4 py-3">Domisili</th><th class="px-4 py-3">Tanggal Daftar</th><th class="px-4 py-3">Angkatan</th><th class="px-4 py-3">Level</th><th class="px-4 py-3">Tipe</th><th class="px-4 py-3">Status</th><th class="px-4 py-3"></th></tr></thead>
					<tbody class="divide-y divide-neutral-200/70 dark:divide-white/[0.04]">
						{#each analytics?.rows || [] as row, index}
							<tr class="hover:bg-neutral-50/60 dark:hover:bg-white/[0.02]"><td class="px-4 py-3 text-neutral-500">{startRow + index}</td><td class="px-4 py-3"><p class="font-medium text-neutral-900 dark:text-white">{row.nama}</p><p class="font-mono text-xs text-neutral-500">{row.id_mahasantri || "-"}</p></td><td class="px-4 py-3">{row.usia > 0 ? row.usia : "-"}</td><td class="px-4 py-3">{gender(row.jenis_kelamin)}</td><td class="max-w-[180px] truncate px-4 py-3" title={row.domisili}>{row.domisili || "-"}</td><td class="px-4 py-3">{row.tanggal_daftar || "-"}</td><td class="px-4 py-3">{row.angkatan || "-"}</td><td class="px-4 py-3">{row.level || "-"}</td><td class="px-4 py-3">{row.tipe || "-"}</td><td class="px-4 py-3"><span class="rounded-full bg-neutral-100 px-2.5 py-1 text-xs dark:bg-neutral-800">{status(row.status)}</span></td><td class="px-4 py-3"><a href={`/app/santri/${row.id}`} use:inertia class="inline-flex items-center gap-1 text-xs font-semibold text-brand-600 dark:text-brand-400">Detail <ArrowRight size="13" /></a></td></tr>
						{/each}
						{#if !(analytics?.rows || []).length}<tr><td colspan="11" class="px-4 py-12 text-center text-sm text-neutral-500">Tidak ada santri pada segmen ini.</td></tr>{/if}
					</tbody>
				</table></div>
				<div class="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-200/80 px-4 py-3 dark:border-white/[0.04]"><p class="text-xs text-neutral-500">Menampilkan {startRow}-{endRow} dari {rowTotal.toLocaleString("id-ID")}</p><div class="flex items-center gap-2"><button type="button" onclick={() => goToPage(page - 1)} disabled={page <= 1} class="rounded-lg border border-neutral-200 p-2 disabled:opacity-40 dark:border-white/[0.08]"><ChevronLeft size="16" /></button><span class="px-2 text-xs">{page} / {totalPages}</span><button type="button" onclick={() => goToPage(page + 1)} disabled={page >= totalPages} class="rounded-lg border border-neutral-200 p-2 disabled:opacity-40 dark:border-white/[0.08]"><ChevronRight size="16" /></button></div></div>
			</div>
		</section>
	</div>
</AppLayout>
