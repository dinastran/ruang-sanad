<script lang="ts">
 import { inertia, router } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import type { User } from "@lib/types";
 import { STAGE_LABELS, relativeTime } from "@lib/leads";
 import { fmt, panel, type Priority } from "@components/lead-dashboard/types";
 import { ArrowLeft, ArrowUpRight, Search, Inbox, ClipboardList } from "lucide-svelte";

 interface Queue {
  items: Priority[]; total: number; page: number; per_page: number; query: string; reason: string;
 }
 interface Props { user?: User; queue: Queue }
 let props: Props = $props();
 let queue = $derived(props.queue);
 let search = $state(props.queue.query);
 let loading = $state(false);
 const filters = [
  { key: "", title: "Semua prioritas" },
  { key: "verify", title: "Verifikasi pembayaran" },
  { key: "convert", title: "Konversi santri" },
  { key: "review", title: "Tinjau data" },
  { key: "reply", title: "Balas pesan" },
 ];
 const category = (text: string) => text.startsWith("Bukti") ? "verify" : text.startsWith("Terverifikasi") ? "convert" : text.startsWith("Data") ? "review" : "reply";
 const color = (text: string) => {
  const key = category(text);
  return key === "verify" ? "bg-amber-500/10 text-amber-800 dark:text-amber-300"
   : key === "convert" ? "bg-emerald-500/10 text-emerald-800 dark:text-emerald-300"
   : key === "review" ? "bg-blue-500/10 text-blue-800 dark:text-blue-300"
   : "bg-neutral-100 text-neutral-700 dark:bg-white/10 dark:text-neutral-200";
 };
 let pages = $derived(Math.max(1, Math.ceil(queue.total / queue.per_page)));

 function go(reason = queue.reason, page = 1, q = search) {
  const params = new URLSearchParams();
  if (q.trim()) params.set("q", q.trim());
  if (reason) params.set("reason", reason);
  if (page > 1) params.set("page", String(page));
  router.get("/app/leads/prioritas?" + params.toString(), {}, {
   preserveScroll: true,
   onStart: () => (loading = true),
   onFinish: () => (loading = false),
  });
 }
</script>

<svelte:head><title>Prioritas Lead Aktif · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
 <main class="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:py-8">
  <header class="flex flex-wrap items-end justify-between gap-4">
   <div>
    <a href="/app/leads/dashboard" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-brand-700 dark:text-neutral-400"><ArrowLeft size={14}/> Dashboard CRM</a>
    <p class="mt-4 text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">Antrean kerja CS</p>
    <h1 class="mt-1 text-[28px] font-semibold tracking-tight text-neutral-950 dark:text-white">Prioritas lead aktif</h1>
    <p class="mt-1 max-w-2xl text-sm text-neutral-500 dark:text-neutral-400">Lead yang memerlukan tindakan sekarang, di seluruh periode. Prioritas berdasarkan tahap dan kondisi percakapan, bukan prediksi closing.</p>
   </div>
   <a href="/app/leads?view=list" use:inertia class="inline-flex h-10 items-center gap-2 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-neutral-200"><ClipboardList size={16}/> Semua leads <ArrowUpRight size={14}/></a>
  </header>

  <section class={panel+" overflow-hidden"} aria-label="Filter prioritas lead">
   <div class="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-100 px-5 py-4 dark:border-white/[0.07]">
    <div>
     <p class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total antrean sesuai filter</p>
     <p class="mt-0.5 text-3xl font-semibold tracking-tight tabular-nums text-neutral-950 dark:text-white">{fmt(queue.total)} <span class="text-sm font-normal text-neutral-500">lead</span></p>
    </div>
    <form onsubmit={(e) => {e.preventDefault();go(queue.reason,1);}} class="flex w-full max-w-sm gap-2 sm:w-auto" role="search">
     <label for="priority-search" class="sr-only">Cari nama atau ID lead</label>
     <div class="relative min-w-0 flex-1">
      <Search size={16} class="pointer-events-none absolute left-3 top-3 text-neutral-400"/>
      <input id="priority-search" type="search" placeholder="Nama atau ID lead" bind:value={search} class="h-10 w-full rounded-lg border border-neutral-200 bg-white pl-9 pr-3 text-sm text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white" />
     </div>
     <button type="submit" disabled={loading} class="rounded-lg bg-brand-700 px-4 text-sm font-semibold text-white hover:bg-brand-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50">Cari</button>
    </form>
   </div>
   <div class="flex gap-2 overflow-x-auto p-3 sm:px-5" role="group" aria-label="Filter alasan tindakan">
    {#each filters as f}
     <button type="button" onclick={() => go(f.key,1)} aria-pressed={queue.reason===f.key} disabled={loading}
      class={"shrink-0 rounded-lg border px-3 py-2 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 "+(queue.reason===f.key?"border-brand-500/30 bg-brand-500/10 text-brand-800 dark:text-brand-300":"border-neutral-200 bg-white text-neutral-600 hover:bg-neutral-50 dark:border-white/10 dark:bg-neutral-900 dark:text-neutral-300")}>
      {f.title}
     </button>
    {/each}
   </div>
  </section>

  <section class={panel+" overflow-hidden"} aria-labelledby="priority-list-heading" aria-busy={loading}>
   <div class="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-100 px-5 py-4 dark:border-white/[0.07]">
    <h2 id="priority-list-heading" class="text-sm font-semibold text-neutral-950 dark:text-white">Daftar tindakan</h2>
    <span class="text-xs text-neutral-500 dark:text-neutral-400">Urutan: verifikasi, konversi, peninjauan, balas pesan. Yang lebih lama didahulukan.</span>
   </div>
   {#if queue.items.length===0}
    <div class="flex flex-col items-center gap-2 px-5 py-16 text-center">
     <Inbox size={24} class="text-neutral-400"/>
     <p class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">Tidak ada lead yang cocok</p>
     <p class="text-xs text-neutral-500 dark:text-neutral-400">Coba ganti filter atau kata pencarian.</p>
     <button type="button" onclick={() => {search="";go("",1,"");}} class="mt-2 text-sm font-medium text-brand-700 hover:underline dark:text-brand-400">Hapus filter</button>
    </div>
   {:else}
    <div class="divide-y divide-neutral-100 dark:divide-white/[0.06]">
     {#each queue.items as lead (lead.id)}
      <a href={"/app/leads/"+lead.id} use:inertia class="group flex flex-col gap-2 px-5 py-4 transition-colors hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand-500 sm:flex-row sm:items-center sm:justify-between dark:hover:bg-white/[0.03]">
       <div class="min-w-0">
        <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-neutral-950 dark:text-white">
         <span class="truncate">{lead.name || "Tanpa nama"}</span><span class="text-xs font-normal tabular-nums text-neutral-400">L-{lead.id}</span>
        </p>
        <div class="mt-1.5 flex flex-wrap items-center gap-2">
         <span class={"rounded-md px-2 py-0.5 text-xs font-medium "+color(lead.reason)}>{lead.reason}</span>
         <span class="text-xs text-neutral-500 dark:text-neutral-400">{STAGE_LABELS[lead.stage] || lead.stage}</span>
        </div>
       </div>
       <span class="flex shrink-0 items-center gap-2 text-xs text-neutral-500 dark:text-neutral-400">
        Aktivitas {relativeTime(lead.activity_at)} <ArrowUpRight size={15} class="text-neutral-400 group-hover:text-brand-700"/>
       </span>
      </a>
     {/each}
    </div>
   {/if}
   <footer class="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-100 px-5 py-4 text-xs text-neutral-500 dark:border-white/[0.07] dark:text-neutral-400">
    <span>Halaman {queue.page} dari {pages} · {fmt(queue.total)} lead</span>
    <div class="flex gap-2">
     <button type="button" onclick={() => go(queue.reason,queue.page-1,queue.query)} disabled={queue.page<=1||loading} class="rounded-md border border-neutral-200 px-3 py-1.5 text-sm disabled:opacity-40 dark:border-white/10">Sebelumnya</button>
     <button type="button" onclick={() => go(queue.reason,queue.page+1,queue.query)} disabled={queue.page>=pages||loading} class="rounded-md border border-neutral-200 px-3 py-1.5 text-sm disabled:opacity-40 dark:border-white/10">Berikutnya</button>
    </div>
   </footer>
  </section>
 </main>
</AppLayout>
