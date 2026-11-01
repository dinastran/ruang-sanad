<script lang="ts">
 import { inertia, router } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import type { User } from "@lib/types";
 import { formatDateTime } from "@lib/leads";
 import { ArrowRight, ArrowUpRight, Activity, AlertTriangle, CheckCircle2, ClipboardList, Megaphone, MessageCircle, ShieldAlert } from "lucide-svelte";
 import AcquisitionHero from "@components/lead-dashboard/AcquisitionHero.svelte";
 import ActionQueue from "@components/lead-dashboard/ActionQueue.svelte";
 import RegistrationPath from "@components/lead-dashboard/RegistrationPath.svelte";
 import FollowupPanel from "@components/lead-dashboard/FollowupPanel.svelte";
 import { fmt, panel, pct, shortDate, type Dashboard } from "@components/lead-dashboard/types";
 import { scopeOf, totals, rpShort, type Report as MetaAdsReport } from "@components/meta-ads/data";
 import type { CapiAnalytics } from "@components/meta-capi/data";

 interface CAPIStatus {
  enabled:boolean;
  test_mode:boolean;
  counts:Record<string,number>;
  sources_without_url:number;
  has_default_url:boolean;
 }
 interface Props {
  user?:User;
  dashboard:Dashboard;
  summary_days:number;
  ads?:MetaAdsReport;
  ads_error?:string;
  capi_status?:CAPIStatus;
  capi_analytics?:CapiAnalytics;
  capi_error?:string;
 }
 let props:Props=$props();
 let d=$derived(props.dashboard);
 let canAds=$derived(props.user?.role==="cs"||props.user?.role==="super_admin");
 let canCapi=$derived(props.user?.role==="super_admin");
 let canApprove=$derived(canAds);
 let loading=$state(false);
 let campaign=$state("");
 let adScope=$derived(props.ads?scopeOf(props.ads,{campaigns:campaign?[campaign]:[],source:0}):null);
 let adTotals=$derived(props.ads&&adScope?totals(props.ads,adScope,d.from,d.to):null);
 let mappedCampaigns=$derived(props.ads?.campaigns.filter(x=>x.source_id>0)??[]);
 let selectedMapped=$derived(props.ads?.campaigns.filter(x=>(!campaign||x.id===campaign)&&x.source_id>0).length??0);
 let mappedSpend=$derived(props.ads?.rows.reduce((sum,row)=>{
  if(row.d<d.from||row.d>d.to)return sum;
  const map=props.ads!.campaigns.find(x=>x.id===row.c);
  if(!map?.source_id||campaign&&row.c!==campaign)return sum;
  return sum+row.sp;
 },0)??0);
 const capiNames=[
  {event:"Lead",label:"Chat pertama"},
  {event:"CompleteRegistration",label:"Data terisi"},
  {event:"InitiateCheckout",label:"Info pembayaran"},
  {event:"AddPaymentInfo",label:"Bukti diterima"},
  {event:"Purchase",label:"Menjadi santri"},
 ];
 let capiRows=$derived(capiNames.map(item=>({
  ...item,
  data:props.capi_analytics?.events.find(x=>x.event===item.event)
 })));
 let maxSent=$derived(Math.max(1,...capiRows.map(x=>x.data?.sent??0)));
 let capiAttention=$derived((props.capi_status?.counts.failed??0)>0||
  Boolean(props.capi_status?.test_mode)||
  Boolean(props.capi_status && !props.capi_status.enabled)||
  Boolean(props.capi_status && props.capi_status.sources_without_url>0 && !props.capi_status.has_default_url));
 const tab="inline-flex min-h-10 items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
 const activeTab="bg-neutral-900 text-white dark:bg-white dark:text-neutral-950";
 const idleTab="text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-white/[0.06]";
 function period(days:number){
  router.get("/app/leads",{summary_days:days},{preserveScroll:true,onStart:()=>loading=true,onFinish:()=>loading=false});
 }
 function openLeads(extra:Record<string,string>={}){
  const params=new URLSearchParams({view:"list",status:"all",from:d.from,to:d.to,...extra});
  router.get("/app/leads?"+params.toString());
 }
 function openClosing(day?:string){
  const params=new URLSearchParams({view:"list",status:"won",date_field:"converted",from:day||d.from,to:day||d.to});
  router.get("/app/leads?"+params.toString());
 }
 function openStage(stage:string){
  if(stage==="won")openLeads({status:"won"});
  else openLeads({reached_stage:stage});
 }
</script>

<svelte:head><title>Pusat CRM · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
 <main class="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:py-8">
  <header class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
   <div class="min-w-0">
    <p class="text-[11px] font-semibold uppercase tracking-[0.13em] text-brand-700 dark:text-brand-400">Ruang Sanad / Akuisisi & CRM</p>
    <h1 class="mt-1 text-[29px] font-semibold tracking-[-0.03em] text-neutral-950 dark:text-white">Pusat kendali CRM</h1>
    <p class="mt-1 max-w-2xl text-sm leading-relaxed text-neutral-500 dark:text-neutral-400">Lihat calon santri yang masuk, pekerjaan CS yang tertahan, dan apakah perjalanan event dari CRM sudah tercatat di Meta.</p>
   </div>
   <div class="flex flex-col items-start gap-3 lg:items-end">
    <nav class="flex max-w-full flex-wrap gap-1 rounded-xl border border-neutral-200 bg-white p-1 dark:border-white/10 dark:bg-neutral-925" aria-label="Navigasi CRM">
     <a href="/app/leads" use:inertia aria-current="page" class={tab+" "+activeTab}>Ringkasan</a>
     <a href="/app/leads?view=list" use:inertia class={tab+" "+idleTab}>Daftar lead <ArrowUpRight size={14}/></a>
     <a href="/app/leads/dashboard" use:inertia class={tab+" "+idleTab}>Analisis lengkap</a>
    </nav>
    <div class="flex flex-wrap items-center gap-2">
     <span class="mr-1 text-xs tabular-nums text-neutral-500 dark:text-neutral-400">{shortDate(d.from)} sampai {shortDate(d.to)} · WIB</span>
     <div role="group" aria-label="Periode ringkasan CRM" class="inline-flex gap-0.5 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900">
      {#each [7,14,30] as days}
       <button type="button" aria-pressed={props.summary_days===days} disabled={loading} onclick={()=>period(days)}
        class={"rounded-md px-3 py-1.5 text-xs font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 "+(props.summary_days===days?"bg-white text-neutral-950 shadow-sm dark:bg-neutral-800 dark:text-white":"text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white")}>{days} hari</button>
      {/each}
     </div>
    </div>
   </div>
  </header>

  <div class={"space-y-5 transition-opacity duration-150 "+(loading?"pointer-events-none opacity-50":"")} aria-busy={loading}>
   <div class="grid gap-4 lg:grid-cols-[minmax(0,1.6fr)_minmax(20rem,0.85fr)]">
    <AcquisitionHero {d} onleads={openLeads} onclosing={openClosing}/>
    <ActionQueue {d} canApproveFollowup={canApprove} onreview={()=>openLeads({status:"open",review:"true"})}/>
   </div>

   <section class={panel+" overflow-hidden"} aria-labelledby="connection-heading">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-neutral-100 px-5 py-5 sm:px-6 dark:border-white/[0.06]">
     <div>
      <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-brand-700 dark:text-brand-400">Jejak akuisisi</p>
      <h2 id="connection-heading" class="mt-1 text-lg font-semibold tracking-tight text-neutral-950 dark:text-white">Iklan → percakapan → lead → event Meta</h2>
      <p class="mt-1 max-w-3xl text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Tiga sumber data yang dibaca bersama. Pemetaan kampanye memakai sumber lead, sedangkan event CAPI berasal dari milestone CRM. Tidak berarti tiap konversi pasti teratribusi ke iklan.</p>
     </div>
     {#if canAds}<a href="/app/leads/iklan" use:inertia class="inline-flex min-h-10 items-center gap-1.5 text-sm font-semibold text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Analisis iklan <ArrowUpRight size={15}/></a>{/if}
    </div>

    <div class="grid lg:grid-cols-3">
     <div class="min-w-0 border-b border-neutral-100 p-5 sm:p-6 lg:border-b-0 lg:border-r dark:border-white/[0.06]">
      <div class="flex items-center justify-between gap-2">
       <span class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><Megaphone size={17} class="text-brand-700 dark:text-brand-400"/> Meta Ads</span>
       <span class="rounded-md bg-neutral-100 px-2 py-1 font-mono text-[10px] text-neutral-500 dark:bg-white/[0.06] dark:text-neutral-400">01 / INPUT</span>
      </div>
      {#if !canAds}
       <p class="mt-5 text-sm text-neutral-500 dark:text-neutral-400">Data biaya iklan hanya tersedia untuk CS dan Super Admin.</p>
      {:else if props.ads_error}
       <p role="alert" class="mt-5 rounded-lg border border-amber-500/20 px-3 py-3 text-sm text-amber-800 dark:text-amber-300">{props.ads_error}</p>
      {:else if !props.ads || !props.ads.enabled}
       <p class="mt-5 text-sm text-neutral-500 dark:text-neutral-400">Integrasi iklan belum aktif. Hubungkan akun dan sumber kampanye melalui halaman Iklan.</p>
      {:else}
       <label class="mt-4 block text-xs font-medium text-neutral-600 dark:text-neutral-300">Kampanye
        <select bind:value={campaign} class="mt-1 h-10 w-full rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white">
         <option value="">Seluruh kampanye</option>
         {#each props.ads.campaigns as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
        </select>
       </label>
       <div class="mt-4">
        <p class="text-xs text-neutral-500 dark:text-neutral-400">Belanja iklan terpilih</p>
        <p class="mt-1 text-[30px] font-semibold leading-tight tracking-[-0.025em] tabular-nums text-neutral-950 dark:text-white">{rpShort(adTotals?.spend??0)}</p>
       </div>
       <dl class="mt-4 grid grid-cols-2 gap-3 border-t border-neutral-100 pt-3 dark:border-white/[0.06]">
        <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">Percakapan Meta</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(adTotals?.chats??0)}</dd></div>
        <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">Lead sumber terpetakan</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{selectedMapped?fmt(adTotals?.leads??0):"—"}</dd></div>
       </dl>
       {#if selectedMapped===0}
        <p class="mt-3 flex items-start gap-2 rounded-md bg-amber-500/[0.06] p-2.5 text-xs leading-relaxed text-amber-800 dark:text-amber-300"><AlertTriangle size={14} class="mt-0.5 shrink-0"/> Belum ada pemetaan kampanye ke sumber lead untuk pilihan ini.</p>
       {:else}
        <p class="mt-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">{fmt(selectedMapped)} kampanye terpetakan. Biaya yang terpetakan {rpShort(mappedSpend)}. Lead dihitung unik berdasarkan sumber, bukan atribusi per kampanye.</p>
       {/if}
       <p class="mt-3 text-[11px] leading-relaxed text-neutral-500 dark:text-neutral-400">Data iklan terakhir berhasil disinkronkan: {props.ads.last_success?.finished_at?formatDateTime(props.ads.last_success.finished_at):"belum tersedia"}.</p>
       {#if props.ads.last_run?.status==="failed"}<p role="alert" class="mt-2 text-xs text-amber-700 dark:text-amber-300">Sinkron terakhir gagal. Data mungkin tidak terbaru.</p>{/if}
      {/if}
     </div>

     <div class="min-w-0 border-b border-neutral-100 p-5 sm:p-6 lg:border-b-0 lg:border-r dark:border-white/[0.06]">
      <div class="flex items-center justify-between gap-2">
       <span class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><MessageCircle size={17} class="text-brand-700 dark:text-brand-400"/> CRM Leads</span>
       <span class="rounded-md bg-neutral-100 px-2 py-1 font-mono text-[10px] text-neutral-500 dark:bg-white/[0.06] dark:text-neutral-400">02 / PROSES</span>
      </div>
      <div class="mt-5">
       <p class="text-xs text-neutral-500 dark:text-neutral-400">Lead masuk pada periode ini</p>
       <p class="mt-1 text-[30px] font-semibold leading-tight tracking-[-0.025em] tabular-nums text-neutral-950 dark:text-white">{fmt(d.report.total_leads)}</p>
      </div>
      <dl class="mt-5 grid grid-cols-2 gap-x-4 gap-y-4 border-t border-neutral-100 pt-4 dark:border-white/[0.06]">
       <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">Closing periode</dt><dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(d.closing_period)}</dd></div>
       <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">Konversi cohort</dt><dd class="mt-1 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{pct(d.report.won,d.report.total_leads)}</dd></div>
       <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">Masih berjalan</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(d.report.open)}</dd></div>
       <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">Butuh review</dt><dd class="mt-1 text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(d.review_count)}</dd></div>
      </dl>
      <p class="mt-4 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Closing periode dihitung dari tanggal konversi. Konversi cohort mengikuti lead yang masuk di periode terpilih.</p>
      <button type="button" onclick={()=>openLeads()} class="mt-4 inline-flex min-h-10 items-center gap-2 rounded-lg border border-neutral-200 px-3 text-sm font-semibold text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:text-neutral-200 dark:hover:bg-white/[0.05]">Buka daftar lead <ArrowUpRight size={14}/></button>
     </div>

     <div class="min-w-0 p-5 sm:p-6">
      <div class="flex items-center justify-between gap-2">
       <span class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><Activity size={17} class="text-brand-700 dark:text-brand-400"/> Meta CAPI</span>
       <span class="rounded-md bg-neutral-100 px-2 py-1 font-mono text-[10px] text-neutral-500 dark:bg-white/[0.06] dark:text-neutral-400">03 / OUTPUT</span>
      </div>
      {#if !canCapi}
       <p class="mt-5 text-sm leading-relaxed text-neutral-500 dark:text-neutral-400">Detail event dan status CAPI hanya dapat diperiksa oleh Super Admin.</p>
      {:else if props.capi_error}
       <p role="alert" class="mt-5 rounded-lg border border-amber-500/20 p-3 text-sm text-amber-800 dark:text-amber-300">{props.capi_error}</p>
      {:else if !props.capi_status || !props.capi_analytics}
       <p class="mt-5 text-sm text-neutral-500 dark:text-neutral-400">Data monitoring event belum tersedia.</p>
      {:else}
       <p class="mt-4 inline-flex items-center gap-1.5 text-xs font-medium">
        {#if !props.capi_status.enabled}<ShieldAlert size={15} class="text-amber-600"/><span class="text-amber-800 dark:text-amber-300">Belum aktif</span>
        {:else if props.capi_status.test_mode}<AlertTriangle size={15} class="text-amber-600"/><span class="text-amber-800 dark:text-amber-300">Mode uji aktif</span>
        {:else}<CheckCircle2 size={15} class="text-emerald-600"/><span class="text-emerald-700 dark:text-emerald-400">Konfigurasi produksi</span>{/if}
       </p>
       <div class="mt-3">
        <p class="text-xs text-neutral-500 dark:text-neutral-400">Event terkirim ke produksi</p>
        <p class="mt-1 text-[30px] font-semibold leading-tight tracking-[-0.025em] tabular-nums text-neutral-950 dark:text-white">{fmt(props.capi_analytics.sent)}</p>
       </div>
       <dl class="mt-4 grid grid-cols-3 gap-2 border-t border-neutral-100 pt-3 dark:border-white/[0.06]">
        <div><dt class="text-[11px] text-neutral-500">Mode uji</dt><dd class="mt-1 text-base font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(props.capi_analytics.test_sent)}</dd></div>
        <div><dt class="text-[11px] text-neutral-500">Menunggu</dt><dd class="mt-1 text-base font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(props.capi_analytics.pending)}</dd></div>
        <div><dt class="text-[11px] text-neutral-500">Gagal</dt><dd class="mt-1 text-base font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(props.capi_analytics.failed)}</dd></div>
       </dl>
       {#if capiAttention}
        <p class="mt-3 rounded-lg bg-amber-500/[0.06] px-3 py-2 text-xs leading-relaxed text-amber-800 dark:text-amber-300">Ada kondisi integrasi yang perlu diperiksa. Lihat status worker, mode uji, kegagalan event, atau URL sumber pada halaman CAPI.</p>
       {/if}
       <p class="mt-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Angka periode berdasarkan waktu milestone. Status terkirim berarti API menerima request, bukan konversi yang teratribusi.</p>
       <a href="/app/leads/meta-capi" use:inertia class="mt-4 inline-flex min-h-10 items-center gap-1.5 text-sm font-semibold text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Lihat log dan parameter event <ArrowUpRight size={15}/></a>
      {/if}
     </div>
    </div>
   </section>

   {#if canCapi && props.capi_analytics}
    <section class={panel+" p-5 sm:p-6"} aria-labelledby="capi-trigger-heading">
     <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
       <h2 id="capi-trigger-heading" class="text-base font-semibold text-neutral-950 dark:text-white">Event yang dipicu oleh perjalanan lead</h2>
       <p class="mt-1 max-w-3xl text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Setiap milestone direkonsiliasi worker. Grafik menunjukkan event terkirim ke produksi pada periode, bukan jumlah lead unik atau hasil atribusi iklan.</p>
      </div>
      <a href="/app/leads/meta-capi" use:inertia class="inline-flex min-h-10 items-center gap-1.5 text-sm font-semibold text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Audit event <ArrowUpRight size={14}/></a>
     </div>
     <div class="mt-5 grid gap-3 lg:grid-cols-5">
      {#each capiRows as item,i}
       <div class="relative min-w-0 rounded-lg border border-neutral-200/80 bg-neutral-50/50 p-4 dark:border-white/[0.07] dark:bg-white/[0.02]">
        <p class="text-[10px] font-semibold uppercase tracking-[0.08em] text-neutral-500">{String(i+1).padStart(2,"0")} / {item.label}</p>
        <p class="mt-1 break-words font-mono text-[11px] text-neutral-700 dark:text-neutral-300">{item.event}</p>
        <p class="mt-4 text-2xl font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(item.data?.sent??0)}</p>
        <p class="mt-0.5 text-[11px] text-neutral-500">terkirim produksi</p>
        <div role="img" aria-label={`${item.event}: ${fmt(item.data?.sent??0)} event produksi`} class="mt-3 h-1.5 overflow-hidden rounded-full bg-neutral-200 dark:bg-white/[0.08]"><span class="block h-full rounded-full bg-brand-600 dark:bg-brand-400" style:width={((item.data?.sent??0)/maxSent*100)+"%"}></span></div>
        <p class="mt-2 text-[11px] tabular-nums text-neutral-500">Uji {fmt(item.data?.test_sent??0)} · Gagal {fmt(item.data?.failed??0)}</p>
        {#if i<4}<ArrowRight size={14} aria-hidden="true" class="absolute -right-[14px] top-[46%] z-10 hidden bg-white text-neutral-400 lg:block dark:bg-neutral-925"/>{/if}
       </div>
      {/each}
     </div>
     <p class="mt-4 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Pemicu: chat pertama, data terisi, informasi pembayaran, bukti masuk, lalu lead menjadi santri. Satu event per lead per jenis, dengan jendela pengiriman maksimum tujuh hari.</p>
    </section>
   {/if}

   <div class="grid gap-4 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,0.9fr)]">
    <RegistrationPath funnel={d.report.funnel} onstage={openStage}/>
    <FollowupPanel {d} canApproveFollowup={canApprove}/>
   </div>

   <footer class="pb-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
    Ringkasan CRM menggunakan WIB. Data iklan mengikuti tanggal akun Meta, sementara event CAPI dihitung dari waktu milestone CRM. Ketiganya tidak boleh dijumlahkan sebagai satu funnel atribusi individu. Untuk analisis terperinci, gunakan <a href="/app/leads/dashboard" use:inertia class="font-semibold text-brand-700 hover:underline dark:text-brand-400">Dashboard CRM</a>, <a href="/app/leads/iklan" use:inertia class="font-semibold text-brand-700 hover:underline dark:text-brand-400">Iklan</a>, dan halaman CAPI sesuai hak akses.
   </footer>
  </div>
 </main>
</AppLayout>