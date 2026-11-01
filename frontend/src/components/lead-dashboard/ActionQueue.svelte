<script lang="ts">
 import { inertia } from "@inertiajs/svelte";
 import { ArrowUpRight, ChevronRight, ClipboardList } from "lucide-svelte";
 import { fmt, panel, type Dashboard } from "./types";

 interface Props { d: Dashboard; canApproveFollowup: boolean; onreview: () => void }
 let { d, canApproveFollowup, onreview }: Props = $props();
 let counters = $derived([
  { label: "Perlu ditinjau", hint: "lead masuk pada periode", value: d.review_count, go: onreview, href: "" },
  { label: "Menunggu approval", hint: "follow-up", value: d.followup_pending, go: null, href: canApproveFollowup ? "/app/leads/followups" : "" },
  { label: "Gagal dikirim", hint: "follow-up", value: d.followup_failed, go: null, href: canApproveFollowup ? "/app/leads/followups" : "" },
 ]);
 const clickable = "flex w-full items-center justify-between gap-3 rounded-lg px-3 py-3 text-left transition-colors hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.04]";
</script>

<section class={panel+" flex h-full flex-col overflow-hidden"} aria-labelledby="tindakan-title">
 <div class="px-5 pb-3 pt-5">
  <p class="text-[11px] font-semibold uppercase tracking-[0.09em] text-brand-700 dark:text-brand-400">Operasional CS</p>
  <h2 id="tindakan-title" class="mt-1 text-base font-semibold text-neutral-950 dark:text-white">Perlu tindakan</h2>
  <p class="mt-1 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Antrean saat ini. Angka follow-up tidak terikat periode akuisisi.</p>
 </div>
 <div class="space-y-0.5 px-2">
  {#each counters as item}
   {#snippet content()}
    <span class="min-w-0"><span class="block text-sm font-medium text-neutral-800 dark:text-neutral-200">{item.label}</span><span class="block text-[11px] text-neutral-500">{item.hint}</span></span>
    <span class="flex shrink-0 items-center gap-2"><span class="rounded-md bg-neutral-100 px-2 py-0.5 text-sm font-semibold tabular-nums text-neutral-800 dark:bg-white/10 dark:text-neutral-200">{fmt(item.value)}</span><ChevronRight size={14} class="text-neutral-400"/></span>
   {/snippet}
   {#if item.go}<button type="button" onclick={item.go} class={clickable}>{@render content()}</button>
   {:else if item.href}<a href={item.href} use:inertia class={clickable}>{@render content()}</a>
   {:else}<div class={clickable.replace("hover:bg-neutral-50","")}>{@render content()}</div>{/if}
  {/each}
 </div>
 <div class="mt-auto p-4 pt-5">
  <a href="/app/leads/prioritas" use:inertia class="group flex items-center gap-3 rounded-xl border border-brand-600/20 bg-brand-700/[0.06] p-4 transition-colors hover:bg-brand-700/[0.10] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-brand-400/15">
   <span class="rounded-lg bg-brand-700/10 p-2 text-brand-700 dark:text-brand-400"><ClipboardList size={19}/></span>
   <span class="min-w-0 flex-1"><span class="block text-sm font-semibold text-neutral-950 dark:text-white">Prioritas lead aktif</span><span class="mt-0.5 block text-xs text-neutral-600 dark:text-neutral-400">Buka semua tindakan dan filter antrean</span></span>
   <ArrowUpRight size={17} class="shrink-0 text-brand-700 dark:text-brand-400"/>
  </a>
 </div>
</section>
