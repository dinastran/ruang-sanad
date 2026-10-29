<script lang="ts">
    import { inertia, router } from "@inertiajs/svelte";
    import AppLayout from "@layouts/AppLayout.svelte";
    import type { User, Flash } from "@lib/types";
    import { Kanban, CalendarClock, History, ArrowLeft, ArrowRight, Ban } from "lucide-svelte";

    type Stage = "onboarding_baru" | "matching_kelas" | "waiting_kuota" | "siap_mulai" | "masuk_kelas";
    interface PIC { id: number; name: string }
    interface Case {
        id: number; santri_id: number; nama: string; id_mahasantri: string;
        stage: string; pic_user_id: number; pic_name: string;
        registered_at: string; deadline_at: string; actual_started_at?: string;
        kelas_id: number; kelas_nama: string;
    }
    interface Activity { id: number; event_type: string; from_stage: string; to_stage: string; note: string; actor_name: string; created_at: string }
    interface Board { cases: Case[]; counts: Record<string, number>; pics: PIC[]; total: number; page: number; page_size: number }
    interface Props { user?: User; board: Board; flash?: Flash }

    let props: Props = $props();
    let board = $derived(props.board);
    let user = $derived(props.user);
    let busy = $state(false);
    let feedback = $state("");
    let errorMessage = $state("");
    let dragged = $state<number | null>(null);
    let hovering = $state<string | null>(null);
    let pending = $state<{ card: Case; target: string } | null>(null);
    let reason = $state("");
    let actualStartedAt = $state("");
    let historyId = $state<number | null>(null);
    let history = $state<Activity[]>([]);
    let historyLoading = $state(false);

    const stages: {key:Stage; label:string; help:string}[] = [
        {key:"onboarding_baru",label:"Onboarding Baru",help:"Santri baru terdaftar"},
        {key:"matching_kelas",label:"Matching Kelas",help:"Lengkapi profil dan kelas"},
        {key:"waiting_kuota",label:"Waiting Kuota",help:"Menunggu kelas siap"},
        {key:"siap_mulai",label:"Siap Mulai",help:"Guru dan jadwal dikonfirmasi"},
        {key:"masuk_kelas",label:"Masuk Kelas",help:"Tanggal mulai aktual tercatat"}
    ];
    const stageLabels: Record<string,string> = Object.fromEntries(stages.map(x=>[x.key,x.label]));
    const index = (stage:string) => stages.findIndex(s => s.key===stage);
    const canMove = (card:Case, target:string) => index(card.stage)>=0 && Math.abs(index(target)-index(card.stage))===1;

    function dateWIB(value:string):string {
        if (!value) return "-";
        return new Date(value).toLocaleDateString("id-ID",{timeZone:"Asia/Jakarta",day:"numeric",month:"short",year:"numeric"});
    }
    function daysRemaining(value:string):number {
        return Math.ceil((new Date(value).getTime()-Date.now())/86400000);
    }
    function beginMove(card:Case,target:string) {
        if (busy || (target!=="batal" && !canMove(card,target))) return;
        pending={card,target};
        reason="";
        actualStartedAt="";
        errorMessage="";
    }

    function csrfToken():string {
        const cookie=document.cookie.split("; ").find(c=>c.startsWith("XSRF-TOKEN="));
        return cookie ? decodeURIComponent(cookie.split("=").slice(1).join("=")) : "";
    }
    async function updateCase(url:string,body:Record<string,string|number>):Promise<void> {
        if(busy) return;
        busy=true; errorMessage="";
        try {
            const res=await fetch(url,{
                method:"PUT", credentials:"same-origin",
                headers:{"Content-Type":"application/json","Accept":"application/json","X-XSRF-TOKEN":csrfToken()},
                body:JSON.stringify(body)
            });
            const data: {error?:string;ok?:boolean}=await res.json().catch(()=>({}));
            if (!res.ok || !data.ok) throw new Error(data.error ?? "Permintaan gagal. Silakan muat ulang.");
            pending=null; feedback="Perubahan onboarding berhasil disimpan.";
            router.get("/app/onboarding?page="+board.page,{}, {preserveScroll:true});
        } catch (e) {
            errorMessage=e instanceof Error ? e.message : "Terjadi kesalahan";
        } finally { busy=false; }
    }
    function assignPIC(card:Case,e:Event) {
        const id=Number((e.currentTarget as HTMLSelectElement).value);
        if(id===card.pic_user_id) return;
        void updateCase(`/app/onboarding/cases/${card.id}/pic`,{pic_user_id:id});
    }
    function saveMove() {
        if(!pending) return;
        void updateCase(`/app/onboarding/cases/${pending.card.id}/stage`,{
            stage:pending.target,reason,actual_started_at:actualStartedAt
        });
    }
    function dropOn(stage:string) {
        const card=board.cases.find(c=>c.id===dragged);
        dragged=null;hovering=null;
        if(card && canMove(card,stage)) beginMove(card,stage);
    }
    async function showHistory(card:Case) {
        historyId=card.id;history=[];historyLoading=true;errorMessage="";
        try {
            const res=await fetch(`/app/onboarding/cases/${card.id}/activity`, {credentials:"same-origin",headers:{Accept:"application/json"}});
            if(!res.ok) throw new Error("Gagal memuat riwayat");
            const data:{items:Activity[]}=await res.json();
            history=data.items??[];
        } catch (e) {errorMessage=e instanceof Error?e.message:"Gagal memuat riwayat";}
        finally {historyLoading=false;}
    }
    const cardClass="rounded-xl border border-neutral-200 bg-white p-3 shadow-sm dark:border-white/10 dark:bg-neutral-900";
    const btnClass="inline-flex items-center gap-1 rounded-md border border-neutral-200 px-2.5 py-1.5 text-xs font-medium hover:bg-neutral-100 disabled:opacity-40 dark:border-white/10 dark:hover:bg-white/10";
</script>

<AppLayout {user}>
  <div class="mx-auto max-w-[1700px] space-y-5 px-4 py-6 sm:px-6">
    <header class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <p class="flex items-center gap-2 text-xs font-semibold uppercase tracking-widest text-brand-700 dark:text-brand-400"><Kanban size={15}/> Operasional Admin Kelas</p>
        <h1 class="mt-1 text-2xl font-semibold text-neutral-950 dark:text-white">Onboarding Santri</h1>
        <p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Status dipindahkan secara manual. SLA 14 hari kalender dihitung dari tanggal registrasi dan tidak berubah saat kartu dipindah.</p>
      </div>
      <span class="rounded-lg bg-neutral-100 px-3 py-2 text-sm font-medium tabular-nums dark:bg-white/10">{board.total} kasus</span>
    </header>

    {#if feedback}<p role="status" class="rounded-lg border border-emerald-200 px-3 py-2 text-sm text-emerald-700 dark:text-emerald-300">{feedback}</p>{/if}
    {#if props.flash?.success}<p role="status" class="text-sm text-emerald-700">{props.flash.success}</p>{/if}
    {#if errorMessage}<p role="alert" class="rounded-lg border border-red-200 px-3 py-2 text-sm text-red-700 dark:text-red-300">{errorMessage}</p>{/if}

    <div class="flex gap-3 overflow-x-auto pb-4">
      {#each stages as stage}
        {@const cards=board.cases.filter(card=>card.stage===stage.key)}
        <section aria-label={stage.label}
          class={"flex w-[272px] shrink-0 flex-col rounded-xl border bg-neutral-50 dark:bg-neutral-950 "+(hovering===stage.key?"border-brand-500":"border-neutral-200 dark:border-white/10")}
          ondragover={(e)=>{const c=board.cases.find(c=>c.id===dragged);if(c&&canMove(c,stage.key)){e.preventDefault();hovering=stage.key;}}}
          ondragleave={()=>{if(hovering===stage.key)hovering=null;}}
          ondrop={(e)=>{e.preventDefault();dropOn(stage.key);}}>
          <div class="border-b border-neutral-200 px-3 py-3 dark:border-white/10">
            <div class="flex items-center justify-between gap-2"><h2 class="text-sm font-semibold">{stage.label}</h2><span class="rounded-md bg-white px-2 py-0.5 text-xs font-medium tabular-nums dark:bg-white/10">{board.counts[stage.key]??0}</span></div>
            <p class="mt-1 text-xs text-neutral-500">{stage.help}</p>
          </div>
          <div class="min-h-40 flex-1 space-y-2 p-2">
            {#each cards as card (card.id)}
              <article class={cardClass} draggable={card.stage!=="masuk_kelas"&&!busy}
                ondragstart={()=>{dragged=card.id;}} ondragend={()=>{dragged=null;hovering=null;}}>
                <a href={`/app/santri/${card.santri_id}`} use:inertia class="block truncate text-sm font-semibold text-neutral-900 hover:text-brand-600 dark:text-white">{card.nama}</a>
                <p class="mt-0.5 truncate text-xs text-neutral-500">{card.id_mahasantri||"Belum ada ID"}</p>
                {#if card.kelas_nama}<p class="mt-1 truncate text-xs text-neutral-600 dark:text-neutral-300">{card.kelas_nama}</p>{/if}
                <div class="mt-2 flex flex-wrap gap-1.5 text-xs">
                  <span class="flex items-center gap-1 text-neutral-500"><CalendarClock size={12}/>{dateWIB(card.registered_at)}</span>
                  <span class={"rounded px-1.5 py-0.5 font-medium "+(card.stage!=="masuk_kelas"&&daysRemaining(card.deadline_at)<=0?"bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300":"bg-neutral-100 text-neutral-600 dark:bg-white/10 dark:text-neutral-300")}>
                    {card.stage==="masuk_kelas"?"Selesai":daysRemaining(card.deadline_at)<=0?"Lewat SLA":`Batas ${dateWIB(card.deadline_at)}`}
                  </span>
                </div>
                <label class="mt-3 block text-xs text-neutral-500" for={`pic-${card.id}`}>PIC Admin Kelas</label>
                <select id={`pic-${card.id}`} value={String(card.pic_user_id)} disabled={busy||card.stage==="masuk_kelas"}
                  onchange={(e)=>assignPIC(card,e)}
                  class="mt-1 w-full rounded-md border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-900 disabled:opacity-50 dark:border-white/15 dark:bg-neutral-800 dark:text-white">
                  <option value="0">Belum ditetapkan</option>
                  {#each board.pics as pic}<option value={pic.id}>{pic.name}</option>{/each}
                </select>
                <div class="mt-3 flex flex-wrap gap-1">
                  {#if index(card.stage)>0&&card.stage!=="masuk_kelas"}
                    <button class={btnClass} type="button" disabled={busy} title="Kembali satu tahap" onclick={()=>beginMove(card,stages[index(card.stage)-1].key)}><ArrowLeft size={12}/> Mundur</button>
                  {/if}
                  {#if index(card.stage)>=0&&index(card.stage)<stages.length-1}
                    <button class={btnClass} type="button" disabled={busy} onclick={()=>beginMove(card,stages[index(card.stage)+1].key)}>Lanjut <ArrowRight size={12}/></button>
                  {/if}
                  {#if card.stage!=="masuk_kelas"}
                    <button class={btnClass} type="button" disabled={busy} onclick={()=>beginMove(card,"batal")}><Ban size={12}/> Batal</button>
                  {/if}
                  <button class={btnClass} type="button" onclick={()=>showHistory(card)}><History size={12}/> Riwayat</button>
                </div>
              </article>
            {/each}
            {#if cards.length===0}<p class="py-8 text-center text-xs text-neutral-400">Belum ada kartu di halaman ini</p>{/if}
          </div>
        </section>
      {/each}
    </div>

    {#if (board.counts.batal??0)>0}
      <section class="rounded-xl border border-neutral-200 p-4 dark:border-white/10">
        <h2 class="text-sm font-semibold">Batal ({board.counts.batal})</h2>
        <p class="mb-3 text-xs text-neutral-500">Status terminal, di luar lima kolom aktif.</p>
        <div class="flex flex-wrap gap-2">
          {#each board.cases.filter(c=>c.stage==="batal") as card (card.id)}
            <button type="button" class={btnClass} onclick={()=>showHistory(card)}>{card.nama} · Riwayat</button>
          {/each}
        </div>
      </section>
    {/if}

    {#if board.total>board.page_size}
      <nav class="flex items-center justify-center gap-3 text-sm" aria-label="Halaman Kanban">
        <a href={`/app/onboarding?page=${board.page-1}`} use:inertia aria-disabled={board.page<=1} class={btnClass+(board.page<=1?" pointer-events-none opacity-40":"")}>Sebelumnya</a>
        <span>Halaman {board.page} dari {Math.ceil(board.total/board.page_size)} · 100 kasus per halaman</span>
        <a href={`/app/onboarding?page=${board.page+1}`} use:inertia aria-disabled={board.page>=Math.ceil(board.total/board.page_size)} class={btnClass+(board.page>=Math.ceil(board.total/board.page_size)?" pointer-events-none opacity-40":"")}>Berikutnya</a>
      </nav>
    {/if}

    <p class="text-xs text-neutral-500">Seret kartu ke kolom bersebelahan atau gunakan tombol Lanjut/Mundur. Perpindahan ke Siap Mulai memerlukan target kuota dan kesiapan kelas; konfigurasi kuota tersedia pada PR 3.</p>
  </div>

  {#if pending}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation">
      <div class="w-full max-w-md rounded-xl bg-white p-5 shadow-2xl dark:bg-neutral-900" role="dialog" aria-modal="true" aria-label="Konfirmasi perpindahan onboarding">
        <h2 class="text-lg font-semibold">Konfirmasi {pending.target==="batal"?"pembatalan":stageLabels[pending.target]}</h2>
        <p class="mt-1 text-sm text-neutral-500">{pending.card.nama} · {stageLabels[pending.card.stage]} → {stageLabels[pending.target]??"Batal"}</p>
        {#if pending.target==="masuk_kelas"}
          <label for="actual-start" class="mt-4 block text-sm font-medium">Tanggal mulai kelas aktual (WIB)</label>
          <input id="actual-start" type="date" bind:value={actualStartedAt} required class="mt-1 w-full rounded-md border border-neutral-300 p-2 text-sm dark:border-white/20 dark:bg-neutral-800"/>
        {/if}
        <label for="move-reason" class="mt-4 block text-sm font-medium">Alasan {pending.target==="batal"||index(pending.target)<index(pending.card.stage)?"(wajib)":"(opsional)"}</label>
        <textarea id="move-reason" bind:value={reason} maxlength="300" rows="3" class="mt-1 w-full rounded-md border border-neutral-300 p-2 text-sm dark:border-white/20 dark:bg-neutral-800" placeholder="Catatan aktivitas onboarding"></textarea>
        {#if errorMessage}<p role="alert" class="mt-2 text-sm text-red-600">{errorMessage}</p>{/if}
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" disabled={busy} class={btnClass} onclick={()=>{pending=null;errorMessage="";}}>Tutup</button>
          <button type="button" disabled={busy||(pending.target==="masuk_kelas"&&!actualStartedAt)||((pending.target==="batal"||index(pending.target)<index(pending.card.stage))&&!reason.trim())} class="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-40" onclick={saveMove}>{busy?"Menyimpan…":"Konfirmasi"}</button>
        </div>
      </div>
    </div>
  {/if}

  {#if historyId!==null}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation">
      <div class="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-5 shadow-2xl dark:bg-neutral-900" role="dialog" aria-modal="true" aria-label="Riwayat onboarding">
        <div class="flex items-center justify-between"><h2 class="font-semibold">Riwayat onboarding #{historyId}</h2><button class={btnClass} type="button" onclick={()=>{historyId=null;errorMessage="";}}>Tutup</button></div>
        {#if historyLoading}<p class="mt-4 text-sm text-neutral-500">Memuat riwayat…</p>{:else}
          <ol class="mt-4 space-y-3">
            {#each history as item (item.id)}
              <li class="border-l-2 border-brand-300 pl-3 text-sm">
                <p class="font-medium">{item.event_type}</p>
                <p class="text-xs text-neutral-500">{item.actor_name} · {dateWIB(item.created_at)}</p>
                {#if item.from_stage!==item.to_stage}<p class="text-xs">{stageLabels[item.from_stage]??item.from_stage} → {stageLabels[item.to_stage]??item.to_stage}</p>{/if}
                {#if item.note}<p class="mt-1 text-sm">{item.note}</p>{/if}
              </li>
            {/each}
            {#if !history.length}<p class="text-xs text-neutral-500">Belum ada riwayat</p>{/if}
          </ol>
        {/if}
      </div>
    </div>
  {/if}
</AppLayout>
