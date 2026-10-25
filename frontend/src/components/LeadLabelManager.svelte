<script lang="ts">
 import { labelTone } from "../lib/leads";
 type Label = { id: number; nama: string; warna: string };
 let { labels, busy, save, remove }: {
  labels: Label[]; busy: boolean;
  save: (id: number, nama: string, warna: string, done: () => void) => void;
  remove: (id: number, done: () => void) => void;
 } = $props();
 let open = $state(false);
 let editID = $state(0);
 let nama = $state("");
 let warna = $state("neutral");
 let deleting = $state(0);
 function reset() { editID = 0; nama = ""; warna = "neutral"; }
</script>

<button type="button" class="rounded-full border border-neutral-300 px-2 py-0.5 text-xs dark:border-neutral-700" aria-expanded={open} onclick={() => open = !open}>Kelola label</button>
{#if open}
 <section aria-label="Katalog label tim" class="mt-3 w-full rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-700 dark:bg-neutral-900">
  <h3 class="text-sm font-semibold">Katalog label tim</h3>
  <p class="mt-1 text-xs text-neutral-500">Perubahan nama dan warna berlaku untuk seluruh lead. Buat label di sini, lalu pilih lewat + Label.</p>
  <ul class="my-3 divide-y divide-neutral-200 dark:divide-neutral-700">
   {#each labels as label (label.id)}
    <li class="flex flex-wrap items-center gap-2 py-2">
     <span class="mr-auto rounded-full border px-2 py-0.5 text-xs {labelTone(label.warna)}">{label.nama}</span>
     <button type="button" disabled={busy} class="text-xs text-brand-700 dark:text-brand-300" onclick={() => { editID = label.id; nama = label.nama; warna = label.warna; deleting = 0; }}>Ubah<span class="sr-only"> {label.nama}</span></button>
     <button type="button" disabled={busy} class="text-xs text-error" onclick={() => deleting = label.id}>Hapus<span class="sr-only"> {label.nama}</span></button>
     {#if deleting === label.id}
      <div class="w-full text-xs">
       <p>Hapus “{label.nama}” dari katalog dan semua lead? Data lead tetap tersimpan.</p>
       <button type="button" disabled={busy} class="mr-3 mt-2 text-error" onclick={() => remove(label.id, () => { deleting = 0; if (editID === label.id) reset(); })}>Ya, hapus</button>
       <button type="button" disabled={busy} onclick={() => deleting = 0}>Batal</button>
      </div>
     {/if}
    </li>
   {/each}
  </ul>
  <form class="flex flex-wrap items-end gap-2" onsubmit={(e) => { e.preventDefault(); save(editID, nama, warna, reset); }}>
   <label class="min-w-0 flex-1 text-xs">Nama label
    <input required maxlength="60" bind:value={nama} disabled={busy} class="mt-1 block w-full rounded-lg border border-neutral-300 bg-transparent px-3 py-2 text-sm dark:border-neutral-700" placeholder="Contoh: Minta dihubungi sore" />
   </label>
   <label class="text-xs">Warna
    <select bind:value={warna} disabled={busy} class="mt-1 block rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900">
     <option value="neutral">Abu-abu</option><option value="brand">Toska</option><option value="secondary">Ungu</option>
    </select>
   </label>
   <button type="submit" disabled={busy || !nama.trim()} class="rounded-lg bg-brand-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{editID ? "Simpan" : "Buat label"}</button>
   {#if editID}<button type="button" disabled={busy} class="px-2 py-2 text-sm" onclick={reset}>Batal ubah</button>{/if}
  </form>
 </section>
{/if}
