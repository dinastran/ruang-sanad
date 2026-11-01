<script lang="ts">
 import { inertia } from "@inertiajs/svelte";
 import { ArrowUpRight, AlertTriangle } from "lucide-svelte";
 import { fmt, panel } from "@components/lead-dashboard/types";
 import MoneyPath from "@components/meta-ads/MoneyPath.svelte";
 import TrendChart from "@components/meta-ads/TrendChart.svelte";
 import SourcePanel from "@components/meta-ads/SourcePanel.svelte";
 import SpendShare from "@components/meta-ads/SpendShare.svelte";
 import { daily, daysBetween, entities, scopeOf, sourceStats, totals, type Filters, type Report } from "@components/meta-ads/data";

 interface Props { report: Report }
 let { report }: Props = $props();
 let filters = $state<Filters>({ campaigns: [], source: 0 });
 let scope = $derived(scopeOf(report, filters));
 let mappedScope = $derived({
  ...scope,
  campaignIds: new Set(report.campaigns.filter(c => scope.campaignIds.has(c.id) && c.source_id > 0).map(c => c.id)),
 });
 let cur = $derived(totals(report, scope, report.from, report.to));
 let prev = $derived(totals(report, scope, report.prev_from, report.prev_to));
 let mappedSpend = $derived(totals(report, mappedScope, report.from, report.to).spend);
 let prevMappedSpend = $derived(totals(report, mappedScope, report.prev_from, report.prev_to).spend);
 let days = $derived(daysBetween(report.from, report.to));
 let prevDays = $derived(daysBetween(report.prev_from, report.prev_to));
 let curDaily = $derived(daily(report, scope, report.from, report.to));
 let prevDaily = $derived(daily(report, scope, report.prev_from, report.prev_to));
 let campaignRows = $derived(entities(report, scope, "campaign", report.from, report.to));
 let sourceRows = $derived(sourceStats(report, scope, report.from, report.to));
 let unmapped = $derived(campaignRows.filter(c => c.sourceId === 0).length);
 const label = "block text-xs font-semibold text-neutral-600 dark:text-neutral-300";
 const select = "mt-1 h-10 min-w-0 w-full rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white";
 function setCampaign(id: string) { filters = { campaigns: id ? [id] : [], source: 0 }; }
 function setSource(id: number) { filters = { campaigns: [], source: id }; }
</script>

<section class="space-y-4" aria-labelledby="ads-overview-title">
 <div class={panel+" flex flex-wrap items-end justify-between gap-4 p-5 sm:p-6"}>
  <div>
   <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">Integrasi Meta Ads</p>
   <h2 id="ads-overview-title" class="mt-1 text-lg font-semibold tracking-tight text-neutral-950 dark:text-white">Dari biaya iklan sampai santri</h2>
   <p class="mt-1 max-w-2xl text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Biaya dan chat dari Meta Ads, lead dan closing berdasarkan sumber CRM yang dipetakan. Filter di bawah hanya mengubah analisis iklan, bukan ringkasan CRM di atas.</p>
  </div>
  <a href="/app/leads/iklan" use:inertia class="inline-flex h-10 items-center gap-2 rounded-lg border border-neutral-200 px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:text-neutral-200">Analisis iklan lengkap <ArrowUpRight size={15}/></a>
  <div class="grid w-full gap-3 sm:grid-cols-2">
   <label class={label}>Kampanye
    <select class={select} value={filters.campaigns[0]??""} onchange={(e)=>setCampaign(e.currentTarget.value)}>
     <option value="">Semua kampanye</option>
     {#each report.campaigns as c (c.id)}
      <option value={c.id}>{c.name}</option>
     {/each}
    </select>
   </label>
   <label class={label}>Sumber lead
    <select class={select} value={filters.source} onchange={(e)=>setSource(Number(e.currentTarget.value))}>
     <option value={0}>Semua sumber terhubung</option>
     {#each report.sources as src (src.id)}<option value={src.id}>{src.nama}</option>{/each}
    </select>
   </label>
  </div>
 </div>

 {#if !report.enabled}
  <div class="rounded-xl border border-amber-500/20 bg-amber-500/[0.05] p-5 text-sm text-amber-800 dark:text-amber-200">
   Data iklan belum dikonfigurasi. Ringkasan CRM tetap dapat digunakan.
  </div>
 {:else}
  {#if report.last_run?.status === "failed"}
   <div role="alert" class="flex gap-2 rounded-lg border border-amber-500/20 p-3 text-xs text-amber-800 dark:text-amber-300">
    <AlertTriangle size={16} class="shrink-0"/> Sinkron iklan terakhir gagal. Data mungkin belum terbaru. Periksa halaman Performa Iklan.
   </div>
  {/if}
  {#if unmapped > 0}
   <p class="rounded-lg border border-amber-500/20 bg-amber-500/[0.04] px-4 py-3 text-xs text-amber-900 dark:text-amber-200">{fmt(unmapped)} kampanye pada filter ini belum dipetakan ke sumber lead. Biayanya tidak digunakan untuk menghitung biaya per lead/santri.</p>
  {/if}
  <MoneyPath {cur} {prev} {mappedSpend} {prevMappedSpend} hasMapping={scope.hasMapping}/>
  <TrendChart {days} {prevDays} cur={curDaily} prev={prevDaily} hasMapping={scope.hasMapping}/>
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
   <SourcePanel stats={sourceRows} hasMapping={scope.hasMapping} selected={filters.source} onselect={(id)=>setSource(id)}/>
   <SpendShare campaigns={campaignRows} onselect={setCampaign}/>
  </div>
 {/if}
 <p class="px-1 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Angka Meta memakai tanggal akun iklan; angka CRM memakai WIB. Satu sumber dapat digunakan beberapa kampanye, sehingga lead tidak boleh dijumlahkan per kampanye. Biaya per santri adalah estimasi akuisisi berdasar sumber yang terhubung, bukan atribusi individual atau laba.</p>
</section>
