<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { formatDateTime } from "@lib/leads";
	import { ArrowLeft, RefreshCw } from "lucide-svelte";
	import FilterBar from "@components/meta-ads/FilterBar.svelte";
	import MoneyPath from "@components/meta-ads/MoneyPath.svelte";
	import TrendChart from "@components/meta-ads/TrendChart.svelte";
	import SourcePanel from "@components/meta-ads/SourcePanel.svelte";
	import SpendShare from "@components/meta-ads/SpendShare.svelte";
	import PerformanceTable from "@components/meta-ads/PerformanceTable.svelte";
	import { daily, daysBetween, entities, num, scopeOf, shortDay, sourceStats, totals, type Filters, type Level, type Report } from "@components/meta-ads/data";

	interface Props { user?: User; report: Report; flash?: Flash }
	let props: Props = $props();
	let r = $derived(props.report);
	let isSuperAdmin = $derived(props.user?.role === "super_admin");

	// Campaign / source / compare filters live in the URL so a view can be shared.
	function readParams() {
		const p = new URLSearchParams(window.location.search);
		return {
			filters: { campaigns: (p.get("c") ?? "").split(",").filter(Boolean), source: Number(p.get("s")) || 0 } as Filters,
			compare: p.get("cmp") !== "0",
		};
	}
	const initial = readParams();
	let filters = $state<Filters>(initial.filters);
	let compare = $state(initial.compare);
	let level = $state<Level>("campaign");
	let loading = $state(false);
	let syncing = $state(false);

	function writeParams() {
		const url = new URL(window.location.href);
		const set = (k: string, v: string) => (v ? url.searchParams.set(k, v) : url.searchParams.delete(k));
		set("c", filters.campaigns.join(","));
		set("s", filters.source ? String(filters.source) : "");
		set("cmp", compare ? "" : "0");
		window.history.replaceState(window.history.state, "", url.toString());
	}
	function setFilters(f: Filters) { filters = f; writeParams(); }
	function setCompare(on: boolean) { compare = on; writeParams(); }
	function filterCampaign(id: string) { setFilters({ ...filters, campaigns: [id] }); }
	function filterSource(id: number) { setFilters({ ...filters, source: id }); }

	function applyPeriod(from: string, to: string) {
		const p = new URLSearchParams(window.location.search);
		p.set("from", from);
		p.set("to", to);
		router.get("/app/leads/iklan?" + p.toString(), {}, { preserveScroll: true, preserveState: true, onStart: () => (loading = true), onFinish: () => (loading = false) });
	}
	function sync() {
		syncing = true;
		router.post("/app/leads/iklan/sinkron", {}, { preserveScroll: true, preserveState: true, onFinish: () => (syncing = false) });
	}
	function mapCampaign(id: string, sourceId: number) {
		router.put(`/app/leads/iklan/kampanye/${encodeURIComponent(id)}`, { source_id: sourceId }, { preserveScroll: true, preserveState: true });
	}

	// One filtered scope feeds every panel.
	let scope = $derived(scopeOf(r, filters));
	let mappedScope = $derived.by(() => {
		const ids = new Set(r.campaigns.filter((c) => scope.campaignIds.has(c.id) && c.source_id > 0).map((c) => c.id));
		return { ...scope, campaignIds: ids };
	});
	let cur = $derived(totals(r, scope, r.from, r.to));
	let prev = $derived(compare ? totals(r, scope, r.prev_from, r.prev_to) : null);
	let mappedSpend = $derived(totals(r, mappedScope, r.from, r.to).spend);
	let prevMappedSpend = $derived(compare ? totals(r, mappedScope, r.prev_from, r.prev_to).spend : NaN);
	let days = $derived(daysBetween(r.from, r.to));
	let prevDays = $derived(daysBetween(r.prev_from, r.prev_to));
	let curDaily = $derived(daily(r, scope, r.from, r.to));
	let prevDaily = $derived(compare ? daily(r, scope, r.prev_from, r.prev_to) : null);
	let campaignRows = $derived(entities(r, scope, "campaign", r.from, r.to));
	let tableRows = $derived(level === "campaign" ? campaignRows : entities(r, scope, level, r.from, r.to));
	let sources = $derived(sourceStats(r, scope, r.from, r.to));
	let unmapped = $derived(campaignRows.filter((c) => !c.sourceId).length);
	let periodLabel = $derived(r.from === r.to ? shortDay(r.from) : `${shortDay(r.from)} – ${shortDay(r.to)}`);
</script>

<svelte:head><title>Performa Iklan · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-7xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-2 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 [text-wrap:balance] dark:text-white">Performa iklan</h1>
				<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400"><span class="tabular-nums">{periodLabel} · WIB</span> · dari biaya Meta Ads sampai santri di CRM</p>
			</div>
			<div class="flex items-center gap-3 text-xs text-neutral-500 dark:text-neutral-400">
				<span class="text-right">
					{#if r.last_run}
						Sinkron {r.last_run.trigger} {formatDateTime(r.last_run.started_at)}
						{#if r.last_run.status === "success"}<span class="text-emerald-700 dark:text-emerald-400">· berhasil</span>{:else if r.last_run.status === "failed"}<span class="text-red-700 dark:text-red-400">· gagal</span>{:else}· berjalan{/if}
					{:else}Belum pernah sinkron{/if}
					{#if r.accounts.length}<span class="block text-neutral-400">{r.accounts.join(", ")}</span>{/if}
				</span>
				{#if isSuperAdmin && r.enabled}
					<button type="button" onclick={sync} disabled={syncing} class="inline-flex h-9 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border border-neutral-200 bg-white px-3 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 dark:border-white/10 dark:bg-neutral-925 dark:text-neutral-300 dark:hover:bg-white/[0.04]">
						<RefreshCw size={15} class={syncing ? "animate-spin motion-reduce:animate-none" : ""} /> {syncing ? "Menyinkron…" : "Sinkron"}
					</button>
				{/if}
			</div>
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}
		{#if !r.enabled}
			<div class="rounded-xl border border-amber-500/30 bg-amber-500/[0.06] px-4 py-4 text-sm text-amber-900 sm:px-5 dark:text-amber-200">
				<p class="font-semibold">Belum terhubung ke Meta Ads</p>
				<p class="mt-1">Isi <code class="font-mono text-xs">META_ADS_TOKEN</code> dan <code class="font-mono text-xs">META_AD_ACCOUNT_IDS</code> di <code class="font-mono text-xs">.env</code> server lalu restart (panduan <code class="font-mono text-xs">docs/panduan/meta-ads-api.md</code>, Langkah 1–9).</p>
			</div>
		{:else if r.last_run?.status === "failed"}
			<div role="alert" class="rounded-xl border border-red-500/30 bg-red-500/[0.05] px-4 py-3 text-sm text-red-800 sm:px-5 dark:text-red-300"><strong class="font-semibold">Sinkron terakhir gagal:</strong> <span class="break-words">{r.last_run.error}</span>. Data yang tampil adalah hasil sinkron berhasil sebelumnya.</div>
		{/if}

		<FilterBar from={r.from} to={r.to} prevFrom={r.prev_from} prevTo={r.prev_to} {loading} {compare} {filters} campaigns={r.campaigns} sources={r.sources}
			onperiod={applyPeriod} oncompare={setCompare} onfilters={setFilters} />

		<div class={"space-y-4 transition-opacity duration-150 " + (loading ? "pointer-events-none opacity-60" : "")} aria-busy={loading}>
			<MoneyPath {cur} {prev} {mappedSpend} {prevMappedSpend} hasMapping={scope.hasMapping} />

			{#if unmapped > 0 && r.campaigns.length}
				<p class="rounded-lg border border-amber-500/25 bg-amber-500/[0.05] px-4 py-2.5 text-xs text-amber-900 dark:text-amber-200">
					<strong class="font-semibold">{num(unmapped)} kampanye</strong> belum dihubungkan ke sumber lead, jadi biayanya belum dihitung ke lead &amp; santri. Pilih sumbernya di kolom <em>Sumber lead</em> pada tabel di bawah.
				</p>
			{/if}

			<TrendChart {days} cur={curDaily} prev={prevDaily} {prevDays} hasMapping={scope.hasMapping} />

			<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)]">
				<SourcePanel stats={sources} hasMapping={scope.hasMapping} selected={filters.source} onselect={filterSource} />
				<SpendShare campaigns={campaignRows} onselect={filterCampaign} />
			</div>

			<PerformanceTable rows={tableRows} {level} sources={r.sources} onlevel={(l) => (level = l)} onfiltercampaign={filterCampaign} onmap={mapCampaign} />

			<details class="rounded-xl border border-neutral-200/80 px-5 py-3 text-sm sm:px-6 dark:border-white/[0.07]">
				<summary class="w-fit cursor-pointer text-xs font-medium text-neutral-500 dark:text-neutral-400">Cara membaca angka di halaman ini</summary>
				<ul class="mt-2 list-disc space-y-1 pl-5 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
					<li><strong>Biaya, impresi, klik, chat WA</strong> dari Meta (per tanggal zona waktu akun iklan). Meta masih bisa merevisi beberapa hari terakhir; Dashboard menarik ulang 28 hari setiap hari.</li>
					<li><strong>Lead dan santri</strong> dari CRM: lead yang masuk pada periode ini dari sumber yang terhubung ke kampanye. Status santri adalah kondisi sekarang, jadi bisa naik setelah lead lama closing.</li>
					<li><strong>Biaya per lead/santri</strong> = biaya kampanye yang terhubung ÷ lead/santri dari sumbernya. Bila satu sumber dipakai beberapa kampanye, lead &amp; santri dihitung per sumber.</li>
					<li><strong>Periode lalu</strong> = rentang dengan panjang sama tepat sebelum periode terpilih; grafik membandingkan hari ke-n yang sama.</li>
					<li><strong>Chat WA (Meta)</strong> dihitung Meta saat orang memulai percakapan setelah minimal 7 hari tidak aktif, sehingga bisa berbeda dari lead CRM.</li>
				</ul>
			</details>
		</div>
	</div>
</AppLayout>
