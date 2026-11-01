<script lang="ts">
 import { inertia, router } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import type { User } from "@lib/types";
 import { LOST_REASON_LABELS } from "@lib/leads";
 import { ArrowUpRight, Activity, ClipboardList, Megaphone } from "lucide-svelte";
 import type { Report as MetaAdsReport } from "@components/meta-ads/data";
 import AdsOverview from "@components/lead-dashboard/AdsOverview.svelte";
 import PeriodPicker from "@components/lead-dashboard/PeriodPicker.svelte";
 import AcquisitionHero from "@components/lead-dashboard/AcquisitionHero.svelte";
 import ActionQueue from "@components/lead-dashboard/ActionQueue.svelte";
 import RegistrationPath from "@components/lead-dashboard/RegistrationPath.svelte";
 import StatusComposition from "@components/lead-dashboard/StatusComposition.svelte";
 import FollowupPanel from "@components/lead-dashboard/FollowupPanel.svelte";
 import ReasonBars from "@components/lead-dashboard/ReasonBars.svelte";
 import SourcePerformance from "@components/lead-dashboard/SourcePerformance.svelte";
 import { panel, shortDate, type Dashboard } from "@components/lead-dashboard/types";

 interface CAPI {
  enabled: boolean; test_mode: boolean; counts: Record<string, number>;
  sources_without_url: number; has_default_url: boolean;
 }
 interface Props { user?: User; dashboard: Dashboard; ads?: MetaAdsReport; capi?: CAPI; ads_error?: string; capi_error?: string }
 let props: Props = $props();
 let d = $derived(props.dashboard);
 let canApproveFollowup = $derived(props.user?.role === "cs" || props.user?.role === "super_admin");
 let canViewAds = $derived(canApproveFollowup);
 let isSuperAdmin = $derived(props.user?.role === "super_admin");
 let loading = $state(false);
 let periodLabel = $derived(d.from === d.to ? shortDate(d.from) : shortDate(d.from) + " – " + shortDate(d.to));
 let leadsHref = $derived("/app/leads?" + new URLSearchParams({ status: "all", from: d.from, to: d.to }).toString());

 function applyRange(from: string, to: string) {
  router.get("/app/leads/dashboard", { from, to }, { preserveScroll: true, onStart: () => { loading = true; }, onFinish: () => { loading = false; } });
 }
 function openLeads(extra: Record<string, string> = {}) {
  const params = new URLSearchParams({ status: "all", from: d.from, to: d.to, ...extra });
  router.get("/app/leads?" + params.toString());
 }
 function openClosing(day?: string) {
  const params = new URLSearchParams({ status: "won", date_field: "converted", from: day || d.from, to: day || d.to });
  router.get("/app/leads?" + params.toString());
 }
 function openStage(stage: string) {
  if (stage === "won") openLeads({ status: "won" });
  else openLeads({ reached_stage: stage });
 }
</script>

<AppLayout user={props.user}>
 <div class="mx-auto max-w-7xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
  <header class="flex flex-col gap-4 pb-2 sm:flex-row sm:items-end sm:justify-between">
   <div class="min-w-0">
    <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">CRM Leads</p>
    <h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 [text-wrap:balance] dark:text-white">Dashboard CRM & akuisisi</h1>
    <p class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-neutral-500 dark:text-neutral-400">
     <span class="tabular-nums">{periodLabel} · WIB</span>
     <a href={leadsHref} use:inertia class="inline-flex items-center gap-1 font-medium text-neutral-700 underline-offset-4 hover:text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-neutral-300 dark:hover:text-brand-400">Daftar lead <ArrowUpRight size={14} /></a>
    </p>
   </div>
   <div class="flex flex-wrap items-center gap-2">
    <a href="/app/leads/prioritas" use:inertia class="inline-flex h-10 items-center gap-2 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-neutral-200"><ClipboardList size={15}/> Prioritas CS</a>
    {#if canViewAds}<a href="/app/leads/iklan" use:inertia class="inline-flex h-10 items-center gap-2 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-neutral-200"><Megaphone size={15}/> Iklan</a>{/if}
    <PeriodPicker from={d.from} to={d.to} {loading} onapply={applyRange} />
   </div>
  </header>

  {#if loading}<p role="status" aria-live="polite" class="sr-only">Memuat data dashboard…</p>{/if}
  <div class={"space-y-4 transition-opacity duration-150 " + (loading ? "pointer-events-none opacity-60" : "")} aria-busy={loading}>
   <div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
    <AcquisitionHero {d} onleads={openLeads} onclosing={openClosing} />
    <!-- On desktop the queue follows the hero's height and its list scrolls. -->
    <div class="lg:relative">
     <ActionQueue {d} {canApproveFollowup} onreview={() => openLeads({ status: "open", review: "true" })} />
    </div>
   </div>

   {#if props.ads_error}
    <p role="alert" class="rounded-lg border border-amber-500/20 bg-amber-500/[0.05] px-4 py-3 text-sm text-amber-800 dark:text-amber-200">{props.ads_error}</p>
   {/if}
   {#if props.ads}
    <AdsOverview report={props.ads}/>
   {/if}

   {#if isSuperAdmin}
    <section class={panel+" p-5 sm:p-6"} aria-labelledby="capi-heading">
     <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
       <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-brand-700 dark:text-brand-400">Kesehatan integrasi</p>
       <h2 id="capi-heading" class="mt-1 flex items-center gap-2 text-base font-semibold text-neutral-950 dark:text-white"><Activity size={17}/> Meta Conversions API</h2>
       <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Status antrean event yang dikirim ke Meta, bukan jumlah konversi yang diatribusikan.</p>
      </div>
      <a href="/app/leads/meta-capi" use:inertia class="inline-flex items-center gap-1 text-sm font-medium text-brand-700 hover:underline dark:text-brand-400">Detail CAPI <ArrowUpRight size={15}/></a>
     </div>
     {#if props.capi}
      <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
       {#each [{label:"Terkirim",value:props.capi.counts.sent??0},{label:"Menunggu",value:props.capi.counts.pending??0},{label:"Gagal",value:props.capi.counts.failed??0},{label:"Dilewati",value:props.capi.counts.skipped??0}] as x}
        <div class="rounded-lg border border-neutral-200/70 bg-neutral-50/60 px-4 py-3 dark:border-white/[0.06] dark:bg-white/[0.025]">
         <p class="text-xs text-neutral-500 dark:text-neutral-400">{x.label}</p>
         <p class="mt-1 text-xl font-semibold tabular-nums text-neutral-950 dark:text-white">{x.value.toLocaleString("id-ID")}</p>
        </div>
       {/each}
      </div>
      {#if !props.capi.enabled || props.capi.test_mode || (props.capi.sources_without_url>0 && !props.capi.has_default_url) || (props.capi.counts.failed??0)>0}
       <p class="mt-3 text-xs leading-relaxed text-amber-800 dark:text-amber-300">
        {#if !props.capi.enabled}CAPI belum aktif. {/if}
        {#if props.capi.test_mode}Mode uji masih aktif. {/if}
        {#if props.capi.sources_without_url>0 && !props.capi.has_default_url}{props.capi.sources_without_url} sumber tidak memiliki URL LP atau URL default. {/if}
        {#if (props.capi.counts.failed??0)>0}Ada event gagal kirim. {/if}
        Periksa halaman CAPI sebelum mengambil keputusan optimasi.
       </p>
      {:else}
       <p class="mt-3 text-xs text-neutral-500 dark:text-neutral-400">CAPI terkonfigurasi. Periksa Events Manager untuk memvalidasi penerimaan dan kualitas pencocokan event.</p>
      {/if}
     {:else}
      <p role="status" class="mt-4 text-sm text-neutral-500 dark:text-neutral-400">{props.capi_error || "Status CAPI tidak tersedia."}</p>
     {/if}
    </section>
   {/if}

   <div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
    <RegistrationPath funnel={d.report.funnel} onstage={openStage} />
    <div class="grid content-start gap-4">
     <StatusComposition r={d.report} onstatus={(status) => openLeads({ status })} />
     <FollowupPanel {d} {canApproveFollowup} />
    </div>
   </div>

   <SourcePerformance sources={d.sources ?? []} onselect={(sumber) => openLeads({ sumber })} />

   <div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-2">
    <ReasonBars title="Alasan lost" subtitle="Dicatat manual oleh CS pada lead lost." empty="Belum ada lead lost pada periode ini."
     items={d.report.lost_reasons} tone="warning" label={(k) => LOST_REASON_LABELS[k] || k || "Tidak diketahui"}
     onselect={(k) => openLeads({ status: "lost", lost_reason: k })} />
    <ReasonBars title="Keberatan menurut AI" subtitle="Dugaan dari analisis AI terakhir, bukan alasan lost terkonfirmasi." empty="Belum tersedia analisis AI."
     items={d.report.objections} label={(k) => k.replaceAll("_", " ")} />
   </div>

   {#if d.insights.length > 0}
    <section class={panel + " p-5 sm:p-6"} aria-labelledby="insight-title">
     <h2 id="insight-title" class="text-base font-semibold text-neutral-950 dark:text-white">Catatan periode</h2>
     <p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Dihitung dari data CRM, bukan prediksi AI atau atribusi iklan.</p>
     <ol class="mt-4 grid gap-x-8 gap-y-3 md:grid-cols-2">
      {#each d.insights as insight, i}
       <li class="flex gap-3 text-sm leading-relaxed text-neutral-700 [text-wrap:pretty] dark:text-neutral-300">
        <span class="mt-px font-mono text-xs tabular-nums text-neutral-400 dark:text-neutral-500">{String(i + 1).padStart(2, "0")}</span><span>{insight}</span>
       </li>
      {/each}
     </ol>
    </section>
   {/if}
  </div>

  <p class="max-w-4xl pb-3 text-xs leading-relaxed text-neutral-500 [text-wrap:pretty] dark:text-neutral-400">Ringkasan CRM dihitung dari lead WhatsApp yang tercatat dan mengikuti periode WIB. Angka iklan hanya terlihat bagi role berwenang dan berasal dari data sinkron Meta. Filter kampanye/sumber tidak mengubah cohort CRM. Konversi cohort dapat berubah setelah lead lama closing.</p>
 </div>
</AppLayout>
