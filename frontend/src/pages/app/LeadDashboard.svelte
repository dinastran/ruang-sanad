<script lang="ts">
 import { inertia, router } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import type { User } from "@lib/types";
 import { LOST_REASON_LABELS } from "@lib/leads";
 import { ArrowUpRight } from "lucide-svelte";
 import PeriodPicker from "@components/lead-dashboard/PeriodPicker.svelte";
 import AcquisitionHero from "@components/lead-dashboard/AcquisitionHero.svelte";
 import ActionQueue from "@components/lead-dashboard/ActionQueue.svelte";
 import RegistrationPath from "@components/lead-dashboard/RegistrationPath.svelte";
 import StatusComposition from "@components/lead-dashboard/StatusComposition.svelte";
 import FollowupPanel from "@components/lead-dashboard/FollowupPanel.svelte";
 import ReasonBars from "@components/lead-dashboard/ReasonBars.svelte";
 import SourcePerformance from "@components/lead-dashboard/SourcePerformance.svelte";
 import { panel, shortDate, type Dashboard } from "@components/lead-dashboard/types";

 interface Props { user?: User; dashboard: Dashboard }
 let props: Props = $props();
 let d = $derived(props.dashboard);
 let canApproveFollowup = $derived(props.user?.role === "cs" || props.user?.role === "super_admin");
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
    <h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 [text-wrap:balance] dark:text-white">Dashboard pendaftaran</h1>
    <p class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-neutral-500 dark:text-neutral-400">
     <span class="tabular-nums">{periodLabel} · WIB</span>
     <a href={leadsHref} use:inertia class="inline-flex items-center gap-1 font-medium text-neutral-700 underline-offset-4 hover:text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-neutral-300 dark:hover:text-brand-400">Buka daftar lead <ArrowUpRight size={14} /></a>
    </p>
   </div>
   <PeriodPicker from={d.from} to={d.to} {loading} onapply={applyRange} />
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

  <p class="max-w-3xl pb-3 text-xs leading-relaxed text-neutral-500 [text-wrap:pretty] dark:text-neutral-400">Data hanya berasal dari lead WhatsApp yang tercatat di CRM, belum mencakup seluruh lead marketing, biaya iklan, atau CAC. Konversi cohort dapat berubah setelah lead lama closing.</p>
 </div>
</AppLayout>
