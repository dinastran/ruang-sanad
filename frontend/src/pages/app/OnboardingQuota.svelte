<script lang="ts">
 import { router } from "@inertiajs/svelte";
 import { onMount } from "svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import OnboardingWorkspaceHeader from "@components/OnboardingWorkspaceHeader.svelte";
 import type { User } from "@lib/types";
 import { ClipboardCopy, History, CalendarClock, ShieldCheck } from "lucide-svelte";

 interface ClassControl {
  kelas_id:number; nama:string; tipe:string; private:boolean; kapasitas:number;
  jumlah_santri:number; kandidat_aktif:number; target_kuota:number; suggested_target:number;
  kekurangan:number; first_candidate_at:string; day:number; phase:string;
  vacancy_required:boolean; vacancy_sent_at:string; can_send_alert:boolean;
  exception_approved:boolean; exception_reason:string; exception_approved_at:string;
  exception_approved_by:number; guru_confirmed:boolean; jadwal_confirmed:boolean;
  can_start:boolean; is_active:boolean; vacancy_text:string;
 }
 interface ReadinessCounts {awaiting_onboarding:number;legacy_preassigned:number;candidate_data_complete:number;crm_payment_verified:number;crm_verification_unknown:number;active_teachers:number;confirmed_availability_guru:number;approved_teaching_level_guru:number;usable_teacher_count:number;active_classes_no_routine_slots:number;regular_classes_above_max:number}
 interface ReadinessIssue {code:string;severity:string;message:string}
 interface ReadinessReport {as_of:string;counts:ReadinessCounts;ready_for_automatic_recommendation:boolean;issues:ReadinessIssue[]}
 interface Action {id:number;event_type:string;note:string;actor_name:string;created_at:string}
 interface Props {user?:User;board:{classes:ClassControl[]};notices:{id:number;type:string;title:string;message:string;action_url:string;read:boolean;created_at:string}[]}
 let props:Props=$props();
 let user=$derived(props.user);
 let classes=$derived(props.board.classes??[]);
 let superAdmin=$derived(user?.role==="super_admin");
 let adminKelas=$derived(superAdmin||user?.role==="admin_kelas");
 let cs=$derived(user?.role==="cs");
 let selected=$state<ClassControl|null>(null);
 let action=$state("");
 let target=$state(1);
 let note=$state("");
 let busy=$state(false);
 let errorMessage=$state("");
 let success=$state("");
 let historyId=$state<number|null>(null);
 let history=$state<Action[]>([]);
 let historyLoading=$state(false);
 let readiness=$state<ReadinessReport|null>(null);
 let readinessLoading=$state(false);
 let readinessError=$state("");
 onMount(async()=>{
  if(user?.role!=="admin_kelas"&&user?.role!=="super_admin")return;
  readinessLoading=true;
  try{
   const response=await fetch("/app/onboarding/planning/readiness",{credentials:"same-origin",headers:{Accept:"application/json"}});
   if(!response.ok)throw new Error("Data kesiapan belum tersedia. Pastikan migration planning sudah diterapkan.");
   readiness=await response.json() as ReadinessReport;
  }catch(e){readinessError=e instanceof Error?e.message:"Tidak dapat memuat kesiapan";}
  finally{readinessLoading=false;}
 });

 const btn="inline-flex items-center gap-1.5 rounded-md border border-neutral-200 px-2.5 py-1.5 text-xs font-medium hover:bg-neutral-100 disabled:opacity-40 dark:border-white/15 dark:hover:bg-white/10";
 const label:Record<string,string>={
  quota:"Ubah Target Kuota",vacancy_sent:"Tandai Alert Sudah Dikirim",
  cs_followup:"Catat Tindak Lanjut CS",marketing_followup:"Catat Koordinasi Marketing",
  exception_approved:"Setujui Exception",exception_revoked:"Cabut Exception"
 };
 function showForm(c:ClassControl,a:string) {
  selected=c;action=a;target=c.target_kuota||c.suggested_target;note="";errorMessage="";
 }
 function token():string {
  const v=document.cookie.split("; ").find(c=>c.startsWith("XSRF-TOKEN="));
  return v?decodeURIComponent(v.split("=").slice(1).join("=")):"";
 }
 async function submit() {
  if(!selected||busy)return;
  busy=true;errorMessage="";
  try {
   const quota=action==="quota";
   const response=await fetch(`/app/onboarding/classes/${selected.kelas_id}/${quota?"quota":"actions"}`,{
    method:quota?"PUT":"POST",credentials:"same-origin",
    headers:{"Content-Type":"application/json",Accept:"application/json","X-XSRF-TOKEN":token()},
    body:JSON.stringify(quota?{target}:{action,note})
   });
   const data:{ok?:boolean;error?:string}=await response.json().catch(()=>({}));
   if(!response.ok||!data.ok)throw new Error(data.error??"Gagal menyimpan tindakan");
   selected=null;success="Tindakan kelas telah disimpan.";router.reload({preserveScroll:true});
  } catch(e){errorMessage=e instanceof Error?e.message:"Gagal menyimpan"}
  finally {busy=false}
 }
 async function showHistory(c:ClassControl) {
  historyId=c.kelas_id;historyLoading=true;history=[];errorMessage="";
  try {
   const res=await fetch(`/app/onboarding/classes/${c.kelas_id}/actions`,{credentials:"same-origin",headers:{Accept:"application/json"}});
   if(!res.ok)throw new Error("Gagal membaca riwayat");
   const obj:{items:Action[]}=await res.json();
   history=obj.items??[];
  } catch(e){errorMessage=e instanceof Error?e.message:"Gagal membaca riwayat"}
  finally {historyLoading=false}
 }
 async function copyText(c:ClassControl) {
  try{await navigator.clipboard.writeText(c.vacancy_text);success="Teks Vacancy Alert disalin. Kirim secara manual, lalu tandai setelah benar-benar terkirim."}
  catch{errorMessage="Tidak dapat menyalin otomatis. Pilih dan salin teks secara manual."}
 }
 const date=(s:string)=>s?new Date(s).toLocaleDateString("id-ID",{timeZone:"Asia/Jakarta",day:"numeric",month:"short",year:"numeric"}):"-";
</script>

<AppLayout {user}>
 <div class="mx-auto max-w-[1500px] space-y-5 px-4 py-6 sm:px-6">
  <OnboardingWorkspaceHeader role={user?.role??""} active="classes" classes={classes} notices={props.notices??[]}/>
  <div>
    <h2 class="text-lg font-semibold">Kelas & Kuota</h2>
    <p class="mt-1 text-sm text-neutral-500">Atur target operasional, pantau kebutuhan Vacancy Alert, dan catat tindak lanjut. Fase D0–D15+ dihitung otomatis dalam WIB.</p>
  </div>
  {#if success}<p role="status" class="rounded-lg border border-emerald-200 p-3 text-sm text-emerald-700">{success}</p>{/if}
  {#if errorMessage}<p role="alert" class="rounded-lg border border-red-200 p-3 text-sm text-red-700">{errorMessage}</p>{/if}


  {#if adminKelas}
   <section class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900" aria-label="Kesiapan data rekomendasi kelas">
    <div class="flex flex-wrap items-center justify-between gap-2">
     <h2 class="font-semibold">Kesiapan Data Rekomendasi Kelas</h2>
     <span class="text-xs text-neutral-500">Read-only, belum membuat kelas atau menugaskan guru</span>
    </div>
    {#if readinessLoading}<p class="mt-3 text-sm text-neutral-500">Memeriksa sumber data...</p>
    {:else if readinessError}<p role="alert" class="mt-3 text-sm text-amber-700 dark:text-amber-300">{readinessError}</p>
    {:else if readiness}
     <div class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
      <div class="rounded-lg bg-neutral-50 p-3 dark:bg-white/5"><p class="text-2xl font-semibold tabular-nums">{readiness.counts.awaiting_onboarding}</p><p class="text-xs text-neutral-500">Menunggu tahap onboarding</p></div>
      <div class="rounded-lg bg-neutral-50 p-3 dark:bg-white/5"><p class="text-2xl font-semibold tabular-nums">{readiness.counts.crm_payment_verified}</p><p class="text-xs text-neutral-500">Kandidat terverifikasi CRM</p></div>
      <div class="rounded-lg bg-neutral-50 p-3 dark:bg-white/5"><p class="text-2xl font-semibold tabular-nums">{readiness.counts.usable_teacher_count}</p><p class="text-xs text-neutral-500">Guru: available + level disetujui</p></div>
      <div class="rounded-lg bg-neutral-50 p-3 dark:bg-white/5"><p class="text-2xl font-semibold tabular-nums">{readiness.counts.active_classes_no_routine_slots}</p><p class="text-xs text-neutral-500">Kelas aktif tanpa jadwal rutin tercatat</p></div>
     </div>
     <p class="mt-3 text-xs font-medium {readiness.ready_for_automatic_recommendation?'text-emerald-700 dark:text-emerald-300':'text-amber-700 dark:text-amber-300'}">{readiness.ready_for_automatic_recommendation?"Syarat agregat awal terpenuhi. Matching individu tetap perlu validasi.":"Belum aman untuk rekomendasi otomatis. Periksa blocker di bawah."}</p>
     {#if readiness.issues.length>0}
      <ul class="mt-2 space-y-1.5">
       {#each readiness.issues as issue (issue.code)}
        <li class="text-xs {issue.severity==='blocker'?'text-red-700 dark:text-red-300':'text-amber-700 dark:text-amber-300'}">{issue.severity==='blocker'?'Blocker':'Perhatian'}: {issue.message}</li>
       {/each}
      </ul>
     {/if}
     <p class="mt-3 text-xs text-neutral-500">Data berlaku {readiness.as_of} WIB. Pembayaran tanpa lead CRM tidak dianggap tidak bayar; perlu verifikasi dari sumber lain. Rekomendasi belum diaktifkan.</p>
    {/if}
   </section>
  {/if}

  {#if classes.length===0}
   <section class="rounded-xl border border-dashed border-neutral-300 p-10 text-center dark:border-white/15">
    <h2 class="font-semibold">Belum ada kelas dengan kasus onboarding baru</h2>
    <p class="mt-2 text-sm text-neutral-500">Kontrol kelas akan muncul ketika santri baru memiliki kelas dari engine existing. Data santri historis tidak di-backfill.</p>
   </section>
  {:else}
   <div class="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
    {#each classes as c (c.kelas_id)}
     <section class="space-y-3 rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
      <div class="flex items-start justify-between gap-2">
       <div><h2 class="font-semibold">{c.nama}</h2><p class="mt-1 text-xs text-neutral-500">{c.tipe} · kelas #{c.kelas_id}</p></div>
       <span class="shrink-0 rounded-md bg-brand-50 px-2 py-1 text-xs font-semibold text-brand-700 dark:bg-brand-500/10 dark:text-brand-300">{c.phase} · D{c.day}</span>
      </div>
      <div class="grid grid-cols-3 gap-2 text-center text-xs">
       <div class="rounded-lg bg-neutral-50 p-2 dark:bg-white/5"><p class="text-lg font-semibold tabular-nums">{c.jumlah_santri}</p><p class="text-neutral-500">Santri di kelas</p></div>
       <div class="rounded-lg bg-neutral-50 p-2 dark:bg-white/5"><p class="text-lg font-semibold tabular-nums">{c.target_kuota||"—"}</p><p class="text-neutral-500">Target</p></div>
       <div class="rounded-lg bg-neutral-50 p-2 dark:bg-white/5"><p class="text-lg font-semibold tabular-nums">{c.kapasitas}</p><p class="text-neutral-500">Maks.</p></div>
      </div>
      <p class="flex items-center gap-1 text-xs text-neutral-500"><CalendarClock size={13}/> Kandidat pertama {date(c.first_candidate_at)} · {c.kandidat_aktif} kasus aktif</p>
      {#if !c.target_kuota}<p class="text-xs font-medium text-amber-700 dark:text-amber-300">Target kuota belum ditentukan oleh Super Admin. Saran awal: {c.suggested_target}.</p>
      {:else if c.kekurangan>0}<p class="text-xs font-medium text-amber-700 dark:text-amber-300">Kurang {c.kekurangan} dari target operasional.</p>
      {:else}<p class="text-xs font-medium text-emerald-700 dark:text-emerald-300">Target kuota terpenuhi.</p>{/if}
      <p class="text-xs text-neutral-500">Guru: {c.guru_confirmed?"Terkonfirmasi":"Belum"} · Jadwal: {c.jadwal_confirmed?"Ada":"Belum"} · Status: {c.can_start?"Siap dicek pada Kanban":"Belum siap"}</p>
      {#if c.private}
       <p class="rounded-md bg-neutral-100 p-2 text-xs text-neutral-600 dark:bg-white/10 dark:text-neutral-300">Private: dikecualikan dari Vacancy Alert reguler.</p>
      {:else if c.vacancy_required}
       <div class="rounded-lg border border-amber-200 p-3 dark:border-amber-800">
        <h3 class="text-xs font-semibold">Vacancy Alert {c.vacancy_sent_at?"· Sudah dicatat terkirim":c.day>=4?"· Perlu dikirim":"· Menunggu D4"}</h3>
        <textarea readonly rows="5" aria-label={"Teks Vacancy Alert "+c.nama} value={c.vacancy_text} class="mt-2 w-full rounded-md border border-neutral-200 bg-white p-2 text-xs dark:border-white/10 dark:bg-neutral-950"></textarea>
        <div class="mt-2 flex flex-wrap gap-2">
         <button type="button" class={btn} onclick={()=>copyText(c)}><ClipboardCopy size={13}/> Salin teks</button>
         {#if c.can_send_alert&&adminKelas}
          <button type="button" class={btn} onclick={()=>showForm(c,"vacancy_sent")}>Tandai telah dikirim</button>
         {/if}
        </div>
       </div>
      {/if}
      {#if c.exception_approved}
       <p class="rounded-md bg-emerald-50 p-2 text-xs text-emerald-800 dark:bg-emerald-500/10 dark:text-emerald-300"><ShieldCheck size={13} class="inline"/> Exception disetujui {date(c.exception_approved_at)}. {c.exception_reason}</p>
      {/if}
      <div class="flex flex-wrap gap-2 border-t border-neutral-100 pt-3 dark:border-white/10">
       {#if superAdmin}<button type="button" class={btn} onclick={()=>showForm(c,"quota")}>Atur target</button>{/if}
       {#if c.target_kuota>0&&!c.private&&c.kekurangan>0&&c.day>=8}
        {#if cs||adminKelas}<button type="button" class={btn} onclick={()=>showForm(c,"cs_followup")}>Tindak lanjut CS</button>{/if}
        {#if adminKelas}<button type="button" class={btn} onclick={()=>showForm(c,"marketing_followup")}>Marketing</button>{/if}
       {/if}
       {#if superAdmin&&c.target_kuota>0&&c.kekurangan>0&&c.day>=11&&!c.exception_approved}
        <button type="button" class={btn} onclick={()=>showForm(c,"exception_approved")}>Setujui exception</button>
       {/if}
       {#if superAdmin&&c.exception_approved}<button type="button" class={btn} onclick={()=>showForm(c,"exception_revoked")}>Cabut exception</button>{/if}
       <button type="button" class={btn} onclick={()=>showHistory(c)}><History size={13}/> Riwayat</button>
      </div>
     </section>
    {/each}
   </div>
  {/if}
  <p class="text-xs text-neutral-500">Kuota adalah target operasional, tidak menggantikan kapasitas maksimum kelas. Alert tidak dikirim otomatis; penandaan hanya dilakukan setelah pengiriman manual. Super Admin mengendalikan target dan exception.</p>
 </div>

 {#if selected}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation">
   <div role="dialog" aria-modal="true" aria-label={label[action]??action} class="w-full max-w-md rounded-xl bg-white p-5 shadow-xl dark:bg-neutral-900">
    <h2 class="text-lg font-semibold">{label[action]??action}</h2>
    <p class="mt-1 text-sm text-neutral-500">{selected.nama}</p>
    {#if action==="quota"}
     <label for="target-quota" class="mt-4 block text-sm">Target operasional (1–{selected.kapasitas})</label>
     <input id="target-quota" type="number" min="1" max={selected.kapasitas} bind:value={target} class="mt-1 w-full rounded-md border border-neutral-300 p-2 dark:border-white/20 dark:bg-neutral-800"/>
     <p class="mt-2 text-xs text-neutral-500">Mengganti target akan mencabut exception yang sebelumnya disetujui.</p>
    {:else}
     <label for="action-note" class="mt-4 block text-sm">Catatan / alasan (wajib)</label>
     <textarea id="action-note" rows="3" maxlength="300" bind:value={note} placeholder={action==="vacancy_sent"?"Sebutkan saluran dan penerima pesan yang sudah dikirim":"Jelaskan tindakan atau keputusan"} class="mt-1 w-full rounded-md border border-neutral-300 p-2 text-sm dark:border-white/20 dark:bg-neutral-800"></textarea>
     {#if action==="vacancy_sent"}<p class="mt-2 text-xs text-amber-700">Tombol ini hanya mencatat pengiriman. Pastikan alert benar-benar sudah dikirim secara manual.</p>{/if}
    {/if}
    {#if errorMessage}<p role="alert" class="mt-2 text-sm text-red-700">{errorMessage}</p>{/if}
    <div class="mt-4 flex justify-end gap-2">
     <button type="button" class={btn} disabled={busy} onclick={()=>{selected=null;errorMessage=""}}>Batal</button>
     <button type="button" disabled={busy||(action==="quota"?(target<1||target>selected.kapasitas||target===selected.target_kuota):!note.trim())} onclick={submit} class="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-40">{busy?"Menyimpan…":"Simpan"}</button>
    </div>
   </div>
  </div>
 {/if}
 {#if historyId!==null}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation">
   <div role="dialog" aria-modal="true" aria-label="Riwayat tindakan kelas" class="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-5 dark:bg-neutral-900">
    <div class="flex items-center justify-between"><h2 class="font-semibold">Riwayat tindakan kelas #{historyId}</h2><button type="button" class={btn} onclick={()=>{historyId=null;errorMessage=""}}>Tutup</button></div>
    {#if historyLoading}<p class="mt-4 text-sm">Memuat…</p>{:else}
     <ol class="mt-4 space-y-3">
      {#each history as row (row.id)}
       <li class="border-l-2 border-brand-300 pl-3 text-sm">
        <p class="font-medium">{row.event_type.replaceAll("_"," ")}</p>
        <p class="text-xs text-neutral-500">{row.actor_name} · {date(row.created_at)}</p>
        <p class="mt-1">{row.note}</p>
       </li>
      {/each}
      {#if !history.length}<li class="text-sm text-neutral-500">Belum ada tindakan kelas</li>{/if}
     </ol>
    {/if}
   </div>
  </div>
 {/if}
</AppLayout>
