<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { formatDateTime } from "@lib/leads";
	import PeriodPicker from "@components/lead-dashboard/PeriodPicker.svelte";
	import OverviewChart from "@components/OverviewChart.svelte";
	import { fmt, shortDate } from "@components/lead-dashboard/types";
	import { ArrowLeft, RefreshCw } from "lucide-svelte";

	interface SyncInfo { trigger: string; status: string; rows: number; error: string; date_from: string; date_to: string; started_at: string; finished_at: string }
	interface Totals { spend: number; impressions: number; clicks: number; link_clicks: number; messaging: number; leads_all: number; leads_ads: number; won_ads: number }
	interface Campaign { id: string; name: string; spend: number; impressions: number; clicks: number; link_clicks: number; messaging: number; ads: number; source_id: number; source_nama: string }
	interface Ad { id: string; name: string; campaign_id: string; campaign_name: string; spend: number; impressions: number; clicks: number; link_clicks: number; messaging: number }
	interface SourceStat { id: number; nama: string; campaigns: number; spend: number; leads: number; won: number }
	interface Report {
		from: string; to: string; enabled: boolean; accounts: string[]; last_run: SyncInfo | null; last_success: SyncInfo | null; has_mapping: boolean;
		totals: Totals; daily: { day: string; spend: number; messaging: number; leads_ads: number }[];
		campaigns: Campaign[]; ads: Ad[]; sources: SourceStat[];
		campaign_options: { id: string; name: string; last_date: string; source_id: number }[];
		source_options: { id: number; nama: string; is_aktif: boolean }[];
	}
	interface Props { user?: User; report: Report; flash?: Flash }

	let props: Props = $props();
	let r = $derived(props.report);
	let t = $derived(r.totals);
	let isSuperAdmin = $derived(props.user?.role === "super_admin");
	let loading = $state(false);
	let syncing = $state(false);

	const rp = (n: number) => "Rp" + Math.round(n).toLocaleString("id-ID");
	const per = (cost: number, n: number) => (n > 0 ? rp(cost / n) : "—");
	const pct = (a: number, b: number) => (b > 0 ? ((a / b) * 100).toFixed(2).replace(".", ",") + "%" : "—");

	let periodLabel = $derived(r.from === r.to ? shortDate(r.from) : `${shortDate(r.from)} – ${shortDate(r.to)}`);
	let inPeriod = $derived(new Set(r.campaigns.map((c) => c.id)));
	let otherCampaigns = $derived(r.campaign_options.filter((c) => !inPeriod.has(c.id)));

	function applyRange(from: string, to: string) {
		router.get("/app/leads/iklan", { from, to }, { preserveScroll: true, onStart: () => (loading = true), onFinish: () => (loading = false) });
	}
	function sync() {
		syncing = true;
		router.post("/app/leads/iklan/sinkron", {}, { preserveScroll: true, onFinish: () => (syncing = false) });
	}
	function mapCampaign(id: string, sourceId: number) {
		router.put(`/app/leads/iklan/kampanye/${encodeURIComponent(id)}`, { source_id: sourceId }, { preserveScroll: true });
	}
	function openLeads(sourceId: number) {
		router.get("/app/leads?" + new URLSearchParams({ status: "all", sumber: String(sourceId), from: r.from, to: r.to }).toString());
	}

	// Spend of the campaigns mapped to a lead source: the base for cost per lead/santri.
	let mappedSpend = $derived(r.sources.reduce((s, x) => s + x.spend, 0));
	let kpis = $derived([
		{ label: "Biaya iklan", value: rp(t.spend), note: `${fmt(t.impressions)} impresi · ${fmt(t.link_clicks)} klik tautan`, hero: true },
		{ label: "Lead dari iklan (CRM)", value: r.has_mapping ? fmt(t.leads_ads) : "—", note: r.has_mapping ? `dari ${fmt(t.leads_all)} lead masuk` : "Hubungkan kampanye ke sumber lead" },
		{ label: "Biaya per lead", value: r.has_mapping ? per(mappedSpend, t.leads_ads) : "—", note: "biaya kampanye terhubung ÷ lead CRM" },
		{ label: "Jadi santri", value: r.has_mapping ? fmt(t.won_ads) : "—", note: r.has_mapping ? `biaya per santri ${per(mappedSpend, t.won_ads)}` : "dari lead sumber terhubung" },
		{ label: "Percakapan WA (Meta)", value: fmt(t.messaging), note: `biaya per percakapan ${per(t.spend, t.messaging)}` },
	]);

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const kicker = "text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400";
	const th = "px-3 py-2 text-right font-medium";
	const td = "px-3 py-2.5 text-right tabular-nums text-neutral-700 dark:text-neutral-300";
	const select = "h-8 max-w-[12rem] rounded-md border border-neutral-200 bg-neutral-50 px-2 text-xs text-neutral-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
</script>

{#snippet sourceSelect(id: string, current: number)}
	<select class={select} aria-label="Sumber lead untuk kampanye" value={String(current || 0)} onchange={(e) => mapCampaign(id, Number(e.currentTarget.value) || 0)}>
		<option value="0">— belum dihubungkan —</option>
		{#each r.source_options as o}<option value={String(o.id)}>{o.nama}{o.is_aktif ? "" : " (nonaktif)"}</option>{/each}
	</select>
{/snippet}

<svelte:head><title>Performa Iklan · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-7xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-4 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Performa iklan</h1>
				<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
					<span class="tabular-nums">{periodLabel} · WIB</span>
					{#if r.last_success}· data Meta per {formatDateTime(r.last_success.finished_at || r.last_success.started_at)}{/if}
				</p>
			</div>
			<PeriodPicker from={r.from} to={r.to} {loading} onapply={applyRange} />
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		{#if !r.enabled}
			<div class="rounded-xl border border-amber-500/30 bg-amber-500/[0.06] px-4 py-4 text-sm text-amber-900 sm:px-5 dark:text-amber-200">
				<p class="font-semibold">Belum terhubung ke Meta Ads</p>
				<p class="mt-1">Isi <code class="font-mono text-xs">META_ADS_TOKEN</code> dan <code class="font-mono text-xs">META_AD_ACCOUNT_IDS</code> di <code class="font-mono text-xs">.env</code> server lalu restart. Langkahnya ada di <code class="font-mono text-xs">docs/panduan/meta-ads-api.md</code> (Langkah 1–9).</p>
			</div>
		{/if}

		<!-- Sync status -->
		<div class={panel + " flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm sm:px-5"}>
			<p class="text-neutral-600 dark:text-neutral-300">
				{#if r.last_run}
					Sinkron terakhir ({r.last_run.trigger}) {formatDateTime(r.last_run.started_at)}:
					{#if r.last_run.status === "success"}<span class="font-medium text-emerald-700 dark:text-emerald-400">berhasil</span>, {fmt(r.last_run.rows)} baris ({shortDate(r.last_run.date_from)} – {shortDate(r.last_run.date_to)})
					{:else if r.last_run.status === "running"}<span class="font-medium text-neutral-700 dark:text-neutral-300">sedang berjalan</span>
					{:else}<span class="font-medium text-red-700 dark:text-red-400">gagal</span> — <span class="break-words">{r.last_run.error}</span>{/if}
				{:else}
					Belum pernah sinkron. Data ditarik otomatis setiap hari setelah 02.00 WIB.
				{/if}
				{#if r.accounts.length}<span class="text-neutral-400 dark:text-neutral-500"> · akun {r.accounts.join(", ")}</span>{/if}
			</p>
			{#if isSuperAdmin && r.enabled}
				<button type="button" onclick={sync} disabled={syncing} class="inline-flex h-9 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border border-neutral-200 px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]">
					<RefreshCw size={15} class={syncing ? "animate-spin motion-reduce:animate-none" : ""} /> {syncing ? "Menyinkron…" : "Sinkron sekarang"}
				</button>
			{/if}
		</div>

		<div class={"space-y-4 transition-opacity duration-150 " + (loading ? "pointer-events-none opacity-60" : "")} aria-busy={loading}>
			<!-- KPIs -->
			<section class="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-neutral-200/80 bg-neutral-100 sm:grid-cols-3 xl:grid-cols-5 dark:border-white/[0.07] dark:bg-white/[0.05]" aria-label="Ringkasan periode">
				{#each kpis as k, i (k.label)}
					<div class={"bg-white px-4 py-4 sm:px-5 dark:bg-neutral-925 " + (i === 0 ? "col-span-2 sm:col-span-1" : "")}>
						<p class="text-xs font-medium text-neutral-600 dark:text-neutral-300">{k.label}</p>
						<p class={"mt-1 font-semibold tracking-[-0.02em] tabular-nums text-neutral-950 dark:text-white " + (k.hero ? "text-[28px] leading-9" : "text-[22px] leading-8")}>{k.value}</p>
						<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">{k.note}</p>
					</div>
				{/each}
			</section>

			<!-- Trends -->
			<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-2">
				<section class={panel + " min-w-0 p-5 sm:p-6"} aria-labelledby="tren-biaya">
					<h2 id="tren-biaya" class={kicker}>Biaya iklan per hari</h2>
					<OverviewChart title="Biaya iklan per hari" money labels={r.daily.map((d) => shortDate(d.day))} series={[{ name: "Biaya iklan", values: r.daily.map((d) => d.spend) }]} />
				</section>
				<section class={panel + " min-w-0 p-5 sm:p-6"} aria-labelledby="tren-lead">
					<h2 id="tren-lead" class={kicker}>Lead CRM vs percakapan WA (Meta)</h2>
					<OverviewChart title="Lead CRM dari sumber terhubung dan percakapan WA menurut Meta per hari" labels={r.daily.map((d) => shortDate(d.day))}
						series={[{ name: "Lead CRM (sumber terhubung)", values: r.daily.map((d) => d.leads_ads) }, { name: "Percakapan WA (Meta)", values: r.daily.map((d) => d.messaging) }]} />
				</section>
			</div>

			<!-- Per lead source -->
			<section class={panel + " overflow-hidden"} aria-labelledby="per-sumber">
				<div class="px-5 pt-5 sm:px-6">
					<h2 id="per-sumber" class={kicker}>Biaya vs hasil per sumber lead</h2>
					<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Biaya dari kampanye yang dihubungkan; lead dan santri dari CRM untuk lead yang masuk pada periode ini.</p>
				</div>
				{#if !r.has_mapping}
					<p class="px-5 py-8 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Belum ada kampanye yang dihubungkan ke sumber lead. Pilih sumbernya di tabel kampanye di bawah.</p>
				{:else}
					<div class="mt-3 overflow-x-auto">
						<table class="w-full min-w-[640px] text-sm">
							<thead><tr class="border-y border-neutral-100 bg-neutral-50/60 text-[11px] text-neutral-500 dark:border-white/[0.05] dark:bg-white/[0.015] dark:text-neutral-400">
								<th scope="col" class="py-2 pl-5 pr-3 text-left font-medium sm:pl-6">Sumber</th><th scope="col" class={th}>Kampanye</th><th scope="col" class={th}>Biaya</th>
								<th scope="col" class={th}>Lead</th><th scope="col" class={th}>Biaya/lead</th><th scope="col" class={th}>Santri</th><th scope="col" class={th + " pr-5 sm:pr-6"}>Biaya/santri</th>
							</tr></thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
								{#each r.sources as s (s.id)}
									<tr>
										<th scope="row" class="py-2.5 pl-5 pr-3 text-left font-medium sm:pl-6"><button type="button" class="text-neutral-900 hover:text-brand-700 dark:text-white dark:hover:text-brand-400" onclick={() => openLeads(s.id)}>{s.nama}</button></th>
										<td class={td}>{s.campaigns}</td><td class={td + " font-semibold text-neutral-900 dark:text-white"}>{rp(s.spend)}</td>
										<td class={td}>{fmt(s.leads)}</td><td class={td}>{per(s.spend, s.leads)}</td>
										<td class={td}>{fmt(s.won)}</td><td class={td + " pr-5 sm:pr-6"}>{per(s.spend, s.won)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- Campaigns -->
			<section class={panel + " overflow-hidden"} aria-labelledby="per-kampanye">
				<div class="px-5 pt-5 sm:px-6">
					<h2 id="per-kampanye" class={kicker}>Kampanye</h2>
					<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Hubungkan setiap kampanye ke sumber lead (LP/iklan) agar biaya bisa dibandingkan dengan lead dan santri di CRM.</p>
				</div>
				{#if r.campaigns.length === 0}
					<p class="px-5 py-8 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">{r.enabled ? "Belum ada data iklan pada periode ini." : "Data iklan muncul setelah Meta Ads terhubung dan sinkron pertama selesai."}</p>
				{:else}
					<div class="mt-3 overflow-x-auto">
						<table class="w-full min-w-[900px] text-sm">
							<thead><tr class="border-y border-neutral-100 bg-neutral-50/60 text-[11px] text-neutral-500 dark:border-white/[0.05] dark:bg-white/[0.015] dark:text-neutral-400">
								<th scope="col" class="py-2 pl-5 pr-3 text-left font-medium sm:pl-6">Kampanye</th><th scope="col" class={th}>Biaya</th><th scope="col" class={th}>Impresi</th>
								<th scope="col" class={th}>Klik tautan</th><th scope="col" class={th}>CTR</th><th scope="col" class={th}>Percakapan WA</th><th scope="col" class={th}>Biaya/percakapan</th>
								<th scope="col" class="py-2 pl-3 pr-5 text-left font-medium sm:pr-6">Sumber lead</th>
							</tr></thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
								{#each r.campaigns as c (c.id)}
									<tr>
										<th scope="row" class="py-2.5 pl-5 pr-3 text-left font-normal sm:pl-6"><span class="block max-w-xs truncate font-medium text-neutral-900 dark:text-white" title={c.name}>{c.name || c.id}</span><span class="text-[11px] text-neutral-400">{c.ads} iklan</span></th>
										<td class={td + " font-semibold text-neutral-900 dark:text-white"}>{rp(c.spend)}</td><td class={td}>{fmt(c.impressions)}</td>
										<td class={td}>{fmt(c.link_clicks)}</td><td class={td}>{pct(c.link_clicks, c.impressions)}</td>
										<td class={td}>{fmt(c.messaging)}</td><td class={td}>{per(c.spend, c.messaging)}</td>
										<td class="py-2.5 pl-3 pr-5 sm:pr-6">{@render sourceSelect(c.id, c.source_id)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
				{#if otherCampaigns.length}
					<details class="border-t border-neutral-100 px-5 py-3 text-sm sm:px-6 dark:border-white/[0.05]">
						<summary class="w-fit cursor-pointer text-xs font-medium text-neutral-600 dark:text-neutral-300">Kampanye lain tanpa biaya di periode ini ({otherCampaigns.length})</summary>
						<ul class="mt-2 space-y-2">
							{#each otherCampaigns as c (c.id)}
								<li class="flex flex-wrap items-center justify-between gap-2"><span class="text-neutral-700 dark:text-neutral-300">{c.name || c.id} <span class="text-[11px] text-neutral-400">terakhir {shortDate(c.last_date)}</span></span>{@render sourceSelect(c.id, c.source_id)}</li>
							{/each}
						</ul>
					</details>
				{/if}
			</section>

			<!-- Ads -->
			{#if r.ads.length}
				<details class={panel + " overflow-hidden"}>
					<summary class="cursor-pointer px-5 py-4 text-sm font-medium text-neutral-800 sm:px-6 dark:text-neutral-100">Rincian per iklan ({r.ads.length})</summary>
					<div class="overflow-x-auto">
						<table class="w-full min-w-[820px] text-sm">
							<thead><tr class="border-y border-neutral-100 bg-neutral-50/60 text-[11px] text-neutral-500 dark:border-white/[0.05] dark:bg-white/[0.015] dark:text-neutral-400">
								<th scope="col" class="py-2 pl-5 pr-3 text-left font-medium sm:pl-6">Iklan</th><th scope="col" class={th}>Biaya</th><th scope="col" class={th}>Impresi</th>
								<th scope="col" class={th}>Klik tautan</th><th scope="col" class={th}>Percakapan WA</th><th scope="col" class={th + " pr-5 sm:pr-6"}>Biaya/percakapan</th>
							</tr></thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
								{#each r.ads as a (a.id)}
									<tr>
										<th scope="row" class="py-2.5 pl-5 pr-3 text-left font-normal sm:pl-6"><span class="block max-w-sm truncate font-medium text-neutral-900 dark:text-white" title={a.name}>{a.name || a.id}</span><span class="block max-w-sm truncate text-[11px] text-neutral-400">{a.campaign_name}</span></th>
										<td class={td + " font-semibold text-neutral-900 dark:text-white"}>{rp(a.spend)}</td><td class={td}>{fmt(a.impressions)}</td>
										<td class={td}>{fmt(a.link_clicks)}</td><td class={td}>{fmt(a.messaging)}</td><td class={td + " pr-5 sm:pr-6"}>{per(a.spend, a.messaging)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</details>
			{/if}

			<details class="rounded-xl border border-neutral-200/80 px-5 py-3 text-sm sm:px-6 dark:border-white/[0.07]">
				<summary class="w-fit cursor-pointer text-xs font-medium text-neutral-500 dark:text-neutral-400">Cara membaca angka di halaman ini</summary>
				<ul class="mt-2 list-disc space-y-1 pl-5 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
					<li><strong>Biaya, impresi, klik, percakapan WA</strong> berasal dari Meta (per tanggal di zona waktu akun iklan). Meta masih bisa merevisi angka beberapa hari terakhir; Dashboard menarik ulang 28 hari setiap hari.</li>
					<li><strong>Lead dan santri</strong> berasal dari CRM: lead yang masuk pada periode ini dan sumbernya terhubung ke kampanye. Status santri adalah kondisi saat ini, jadi angka bisa naik setelah lead lama closing.</li>
					<li><strong>Biaya per lead/santri</strong> = biaya kampanye terhubung ÷ lead/santri dari sumber terhubung. Lead tanpa sumber atau dari sumber yang tidak dihubungkan tidak dihitung.</li>
					<li><strong>Percakapan WA (Meta)</strong> dihitung Meta saat orang memulai percakapan setelah minimal 7 hari tidak aktif, jadi bisa berbeda dari lead CRM.</li>
				</ul>
			</details>
		</div>
	</div>
</AppLayout>
