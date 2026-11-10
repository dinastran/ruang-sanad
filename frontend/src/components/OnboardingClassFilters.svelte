<script lang="ts">
  import { inertia, router } from "@inertiajs/svelte";
  import { Search, X } from "lucide-svelte";

  interface Props {
    total: number;
    page: number;
    pageSize: number;
    filters?: { q:string; status:string };
    pagination?: boolean;
  }
  let { total, page, pageSize, filters, pagination=false }: Props = $props();
  let query=$state(filters?.q??"");
  let status=$state(filters?.status??"");
  let active=$derived(Boolean(filters?.q||filters?.status));
  function url(pageNumber=1, useCurrent=false):string {
    const search=new URLSearchParams();
    const q=useCurrent?(filters?.q??""):query.trim();
    const s=useCurrent?(filters?.status??""):status;
    if(q)search.set("q",q);
    if(s)search.set("status",s);
    if(pageNumber>1)search.set("page",String(pageNumber));
    return "/app/onboarding/classes"+(search.size?"?"+search.toString():"");
  }
  function submit(e:SubmitEvent) {
    e.preventDefault();
    router.get(url(),{}, {preserveScroll:true});
  }
</script>

{#if pagination}
  {#if total>pageSize}
    <nav class="flex items-center justify-center gap-3 text-sm" aria-label="Halaman kelas onboarding">
      <a href={url(page-1,true)} use:inertia aria-disabled={page<=1} class={"rounded-md border border-neutral-200 px-3 py-2 dark:border-white/15"+(page<=1?" pointer-events-none opacity-40":"")}>Sebelumnya</a>
      <span>Halaman {page} dari {Math.ceil(total/pageSize)}</span>
      <a href={url(page+1,true)} use:inertia aria-disabled={page>=Math.ceil(total/pageSize)} class={"rounded-md border border-neutral-200 px-3 py-2 dark:border-white/15"+(page>=Math.ceil(total/pageSize)?" pointer-events-none opacity-40":"")}>Berikutnya</a>
    </nav>
  {/if}
{:else}
  <form class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900" onsubmit={submit}>
    <div class="flex flex-wrap items-end gap-3">
      <label class="min-w-[220px] flex-1 text-xs font-medium text-neutral-600 dark:text-neutral-300">
        Cari kelas
        <div class="mt-1 flex items-center gap-2 rounded-lg border border-neutral-200 px-3 dark:border-white/15">
          <Search size={16}/>
          <input type="search" maxlength="100" aria-label="Cari kelas onboarding" bind:value={query} placeholder="Nama kelas atau ID kelas" class="w-full bg-transparent py-2.5 text-sm outline-none"/>
        </div>
      </label>
      <label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">
        Kondisi kelas
        <select bind:value={status} class="mt-1 block w-full rounded-lg border border-neutral-200 bg-white p-2.5 text-sm dark:border-white/15 dark:bg-neutral-800">
          <option value="">Semua kondisi</option>
          <option value="needs_target">Target belum diatur</option>
          <option value="shortage">Di bawah target</option>
          <option value="ready">Siap mulai</option>
        </select>
      </label>
      <button type="submit" class="rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white">Terapkan</button>
      {#if active}
        <a href="/app/onboarding/classes" use:inertia class="flex items-center gap-1 rounded-lg border border-neutral-200 px-3 py-2.5 text-sm dark:border-white/15"><X size={14}/> Reset</a>
      {/if}
    </div>
    <p class="mt-2 text-xs text-neutral-500" aria-live="polite">{total} kelas cocok, dihitung sebelum pagination.</p>
  </form>
{/if}
