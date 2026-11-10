<script lang="ts">
  import { inertia } from "@inertiajs/svelte";
  import { Users } from "lucide-svelte";

  interface Member { id:number; nama:string; id_mahasantri:string; status:string; stage:string }
  interface Roster { kelas_id:number; nama:string; members:Member[]; total:number; page:number; page_size:number }
  interface Props { kelasID:number; kelasNama:string }
  let {kelasID,kelasNama}:Props=$props();

  let open=$state(false);
  let loading=$state(false);
  let error=$state("");
  let roster=$state<Roster|null>(null);

  const stages:Record<string,string>={
    onboarding_baru:"Onboarding Baru",
    matching_kelas:"Matching Kelas",
    waiting_kuota:"Waiting Kuota",
    siap_mulai:"Siap Mulai",
    masuk_kelas:"Masuk Kelas",
    batal:"Batal",
  };
  const btn="inline-flex items-center gap-1.5 rounded-md border border-neutral-200 px-2.5 py-1.5 text-xs font-medium hover:bg-neutral-100 disabled:opacity-40 dark:border-white/15 dark:hover:bg-white/10";

  async function load(page=1) {
    if(loading)return;
    open=true;loading=true;error="";
    try {
      const response=await fetch("/app/onboarding/classes/"+kelasID+"/members?page="+page,{
        credentials:"same-origin",headers:{Accept:"application/json"}
      });
      if(!response.ok)throw new Error(response.status===403?"Akses daftar anggota kelas ditolak.":"Gagal mengambil anggota kelas.");
      roster=await response.json() as Roster;
    }catch(e){error=e instanceof Error?e.message:"Gagal mengambil daftar mahasantri.";}
    finally{loading=false;}
  }
</script>

<button type="button" class={btn} onclick={()=>{roster=null;void load();}}>
  <Users size={13}/> Lihat mahasantri
</button>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation">
    <div role="dialog" aria-modal="true" aria-label="Anggota kelas onboarding" class="max-h-[85vh] w-full max-w-3xl overflow-y-auto rounded-xl bg-white p-5 shadow-xl dark:bg-neutral-900">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold">Mahasantri di kelas</h2>
          <p class="text-sm text-neutral-500">{roster?.nama??kelasNama}</p>
        </div>
        <button type="button" class={btn} onclick={()=>{open=false;roster=null;error="";}}>Tutup</button>
      </div>
      <p class="mt-2 text-xs text-neutral-500">Hanya anggota aktif atau cuti yang terhubung ke kelas. Kandidat onboarding termasuk dalam jumlah ini. Santri lama dapat tidak mempunyai kasus onboarding.</p>
      {#if loading}<p class="mt-4 text-sm text-neutral-500" role="status">Memuat anggota kelas...</p>{/if}
      {#if error}<p role="alert" class="mt-4 text-sm text-red-700">{error}</p>{/if}
      {#if roster && !loading}
        <p class="mt-4 text-sm font-semibold">{roster.total} mahasantri terhubung</p>
        <div class="mt-3 overflow-x-auto">
          <table class="min-w-full text-left text-sm">
            <thead><tr class="border-b border-neutral-200 text-xs text-neutral-500 dark:border-white/10"><th class="p-2">Nama</th><th class="p-2">ID</th><th class="p-2">Status</th><th class="p-2">Tahap onboarding</th></tr></thead>
            <tbody>
              {#each roster.members as m (m.id)}
                <tr class="border-b border-neutral-100 dark:border-white/10">
                  <td class="p-2"><a href={"/app/santri/"+m.id} use:inertia class="font-medium text-brand-700 hover:underline dark:text-brand-300">{m.nama}</a></td>
                  <td class="p-2">{m.id_mahasantri||"Belum ada ID"}</td>
                  <td class="p-2">{m.status}</td>
                  <td class="p-2">{stages[m.stage]??"Belum ada kasus onboarding"}</td>
                </tr>
              {:else}
                <tr><td colspan="4" class="p-6 text-center text-neutral-500">Belum ada anggota aktif atau cuti.</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
        {#if roster.total>roster.page_size}
          <nav class="mt-3 flex items-center justify-between gap-3 text-sm" aria-label="Halaman anggota kelas">
            <button type="button" class={btn} disabled={loading||roster.page<=1} onclick={()=>load(roster!.page-1)}>Sebelumnya</button>
            <span>Halaman {roster.page} dari {Math.ceil(roster.total/roster.page_size)}</span>
            <button type="button" class={btn} disabled={loading||roster.page>=Math.ceil(roster.total/roster.page_size)} onclick={()=>load(roster!.page+1)}>Berikutnya</button>
          </nav>
        {/if}
      {/if}
    </div>
  </div>
{/if}
