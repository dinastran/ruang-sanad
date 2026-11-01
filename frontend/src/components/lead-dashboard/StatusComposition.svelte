<script lang="ts">
 import { fmt, panel, pct, type Report } from "./types";
 interface Props { r: Report; onstatus: (status: string) => void }
 let { r, onstatus }: Props = $props();
 const circumference = 2 * Math.PI * 44;
 let items = $derived([
  { key: "open", label: "Berjalan", count: r.open, stroke: "stroke-slate-400", dot: "bg-slate-400" },
  { key: "won", label: "Jadi santri", count: r.won, stroke: "stroke-emerald-600", dot: "bg-emerald-600" },
  { key: "lost", label: "Lost", count: r.lost, stroke: "stroke-amber-500", dot: "bg-amber-500" },
  { key: "do_not_contact", label: "Jangan hubungi", count: r.do_not_contact, stroke: "stroke-rose-500", dot: "bg-rose-500" },
 ]);
 let arcs = $derived.by(()=>{
  let offset = 0;
  return items.map(item => {
   const length = r.total_leads ? item.count / r.total_leads * circumference : 0;
   const arc = { ...item, length, offset };
   offset += length;
   return arc;
  });
 });
</script>

<section class={panel+" p-5 sm:p-6"} aria-labelledby="status-title">
 <h2 id="status-title" class="text-base font-semibold text-neutral-950 dark:text-white">Komposisi status</h2>
 <p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Status terkini dari {fmt(r.total_leads)} lead yang masuk pada periode.</p>
 {#if r.total_leads===0}
  <p class="py-10 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada lead pada periode ini.</p>
 {:else}
  <div class="mt-4 grid items-center gap-5 sm:grid-cols-[140px_minmax(0,1fr)] lg:grid-cols-1 xl:grid-cols-[140px_minmax(0,1fr)]">
   <div class="relative mx-auto h-36 w-36">
    <svg viewBox="0 0 124 124" class="h-full w-full" role="img" aria-label={items.map(i=>i.label+": "+pct(i.count,r.total_leads)).join(", ")}>
     <circle cx="62" cy="62" r="44" fill="none" stroke-width="13" class="stroke-neutral-100 dark:stroke-neutral-800"/>
     {#each arcs as arc (arc.key)}
      {#if arc.count>0}
       <circle cx="62" cy="62" r="44" fill="none" stroke-width="13"
        class={arc.stroke} stroke-dasharray={arc.length+" "+(circumference-arc.length)} stroke-dashoffset={-arc.offset}
        transform="rotate(-90 62 62)"/>
      {/if}
     {/each}
    </svg>
    <div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
     <span class="text-2xl font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(r.total_leads)}</span>
     <span class="text-[11px] text-neutral-500 dark:text-neutral-400">lead masuk</span>
    </div>
   </div>
   <div class="grid grid-cols-2 gap-1 sm:grid-cols-1 lg:grid-cols-2 xl:grid-cols-1">
    {#each items as item (item.key)}
     <button type="button" onclick={()=>onstatus(item.key)} class="flex items-center justify-between gap-2 rounded-lg px-2 py-2 text-left transition-colors hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.04]">
      <span class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-300"><span class={"h-2.5 w-2.5 shrink-0 rounded-sm "+item.dot}></span>{item.label}</span>
      <span class="shrink-0 text-sm font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(item.count)} <span class="text-xs font-normal text-neutral-500">{pct(item.count,r.total_leads)}</span></span>
     </button>
    {/each}
   </div>
  </div>
 {/if}
</section>
