<script lang="ts">
 import { inertia, router } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import type { Flash, User } from "@lib/types";
 import { formatDateTime, STAGE_LABELS, todayWIB } from "@lib/leads";
 import { fmt, panel } from "@components/lead-dashboard/types";
 import Trend from "@components/meta-capi/Trend.svelte";
 import { currency, eventLabels, statLabels, type CAPIStatus, type CapiAnalytics } from "@components/meta-capi/data";
 import { ArrowLeft, ArrowUpRight, AlertTriangle, CircleHelp, RefreshCw, Search, Settings2 } from "lucide-svelte";

 interface Props { user?:User; status:CAPIStatus; analytics:CapiAnalytics; flash?:Flash }
 let props:Props=$props();
 let s=$derived(props.status);
 let a=$derived(props.analytics);
 let busy=$state(false);
 let loading=$state(false);
 let from=$state(props.analytics.from);
 let to=$state(props.analytics.to);
 let periodError=$state("");
 let eventFilter=$state("");
 let statusFilter=$state("");
 let search=$state("");
 let filtered=$derived(s.events.filter(e=>
  (!eventFilter||e.event===eventFilter) &&
  (!statusFilter||e.status===statusFilter) &&
  (!search.trim()||[e.lead_nama,e.event,e.event_id,String(e.lead_id)].join(" ").toLowerCase().includes(search.trim().toLowerCase()))
 ));
 let totalQueue=$derived(Object.values(s.counts).reduce((sum,n)=>sum+n,0));
 let maxEvent=$derived(Math.max(1,...a.events.map(x=>x.total)));
 const statTone:Record<string,string>={
  sent:"bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  pending:"bg-amber-500/10 text-amber-800 dark:text-amber-300",
  failed:"bg-rose-500/10 text-rose-700 dark:text-rose-400",
  skipped:"bg-neutral-100 text-neutral-600 dark:bg-white/10 dark:text-neutral-300",
 };
 const inputStyle="h-10 w-full rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white";
 const actionStyle="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 dark:border-white/10 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-white/[0.04]";
 function applyPeriod(f=from,t=to){
  if(!f||!t||f>t){periodError="Masukkan rentang tanggal yang benar.";return;}
  const length=(Date.parse(t+"T00:00:00Z")-Date.parse(f+"T00:00:00Z"))/86400000+1;
  if(!Number.isFinite(length)||length>90){periodError="Maksimal 90 hari.";return;}
  periodError="";
  router.get("/app/leads/meta-capi",{from:f,to:t},{
   preserveScroll:true,onStart:()=>loading=true,onFinish:()=>loading=false,
  });
 }
 function recent(n:number){
  const d=new Date(todayWIB()+"T12:00:00+07:00");
  d.setUTCDate(d.getUTCDate()-(n-1));
  from=d.toLocaleDateString("en-CA",{timeZone:"Asia/Jakarta"});
  to=todayWIB();
  applyPeriod(from,to);
 }
 function retry(){
  busy=true;
  router.post("/app/leads/meta-capi/retry",{},{
   preserveScroll:true,onFinish:()=>busy=false,
  });
 }
 function resendTest(){
  if(!confirm(`Kirim ulang ${fmt(s.resendable_test)} event mode uji yang masih valid ke produksi?`))return;
  busy=true;
  router.post("/app/leads/meta-capi/resend-test",{},{
   preserveScroll:true,onFinish:()=>busy=false,
  });
 }
 function reload(){
  router.reload({only:["status","analytics"],onStart:()=>loading=true,onFinish:()=>loading=false});
 }
 const stageName=(v:string)=>STAGE_LABELS[v]?"Tahap "+STAGE_LABELS[v]:v;
</script>

<svelte:head><title>Monitoring Meta CAPI · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
 <main class="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:py-8">
  <header class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
   <div class="min-w-0">
    <a href="/app/leads/dashboard" use:inertia class="inline-flex items-center gap-1.5 text-xs font-medium text-neutral-500 hover:text-brand-700 dark:text-neutral-400 dark:hover:text-brand-400"><ArrowLeft size={14}/> Dashboard CRM</a>
    <p class="mt-4 text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">Integrasi CRM dan Meta</p>
    <h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.025em] text-neutral-950 dark:text-white">Monitoring Conversions API</h1>
    <p class="mt-1 max-w-2xl text-sm text-neutral-500 dark:text-neutral-400">Pantau perjalanan event CRM ke Meta, parameter yang dikirim, dan masalah yang perlu ditindaklanjuti.</p>
    <div class="mt-3 flex flex-wrap items-center gap-2 text-xs">
     <span class={"rounded-md px-2 py-1 font-semibold "+(!s.enabled?"bg-amber-500/10 text-amber-800 dark:text-amber-300":s.test_mode?"bg-blue-500/10 text-blue-700 dark:text-blue-300":"bg-emerald-500/10 text-emerald-700 dark:text-emerald-300")}>{!s.enabled?"Belum terhubung":s.test_mode?"Mode uji aktif":"Mode produksi"}</span>
     <span class="text-neutral-500 dark:text-neutral-400">Dataset {s.dataset_id||"belum diatur"} · API {s.api_version}</span>
    </div>
   </div>
   <div class="flex flex-wrap gap-2">
    <button type="button" onclick={reload} disabled={loading||busy} class={actionStyle}><RefreshCw size={15} class={loading?"animate-spin motion-reduce:animate-none":""}/> {loading?"Memuat…":"Perbarui data"}</button>
    <a href="/app/leads/sumber" use:inertia class={actionStyle}><Settings2 size={15}/> Sumber lead</a>
   </div>
  </header>

  {#if props.flash?.success}<div role="status" class="rounded-lg border border-emerald-500/20 bg-emerald-500/[0.05] px-4 py-3 text-sm text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
  {#if props.flash?.error}<div role="alert" class="rounded-lg border border-rose-500/20 bg-rose-500/[0.05] px-4 py-3 text-sm text-rose-800 dark:text-rose-300">{props.flash.error}</div>{/if}

  {#if !s.enabled}
   <section role="alert" class="flex gap-3 rounded-xl border border-amber-500/25 bg-amber-500/[0.05] px-5 py-4">
    <AlertTriangle size={20} class="mt-0.5 shrink-0 text-amber-600"/>
    <div class="text-sm text-amber-900 dark:text-amber-200">
     <p class="font-semibold">Pengiriman CAPI belum aktif</p>
     <p class="mt-1">Periksa konfigurasi dataset ID dan access token pada server. Jangan masukkan kredensial ke halaman atau chat. Data event baru dapat dikirim setelah konfigurasi valid.</p>
    </div>
   </section>
  {/if}
  {#if s.enabled && s.test_mode}
   <p role="status" class="rounded-lg border border-blue-500/20 bg-blue-500/[0.04] px-4 py-3 text-sm text-blue-800 dark:text-blue-300">Mode uji aktif. Event yang terkirim dalam mode ini tidak boleh dihitung sebagai event produksi.</p>
  {/if}
  {#if s.enabled && !s.event_source_url && s.sources_without_url.length>0}
   <section role="alert" class="rounded-xl border border-rose-500/25 bg-rose-500/[0.04] p-4 text-sm text-rose-800 dark:text-rose-300">
    <p class="font-semibold">Ada {fmt(s.sources_without_url.length)} sumber tanpa URL landing page</p>
    <p class="mt-1 leading-relaxed">Tidak tersedia URL sumber maupun URL default. Event tertentu dapat ditolak oleh Meta. Periksa <a href="/app/leads/sumber" use:inertia class="font-semibold underline">pengaturan sumber lead</a>.</p>
   </section>
  {/if}
  {#if s.enabled && (s.counts.failed??0)>0}
   <section class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-rose-500/20 bg-rose-500/[0.04] p-4">
    <div>
     <p class="text-sm font-semibold text-rose-800 dark:text-rose-300">{fmt(s.counts.failed??0)} event berstatus gagal dalam antrean keseluruhan</p>
     <p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Periksa penyebab pada log event. Kirim ulang hanya event yang masih dalam batas 7 hari.</p>
    </div>
    <button type="button" onclick={retry} disabled={busy} class={actionStyle}><RefreshCw size={15}/> Kirim ulang yang valid</button>
   </section>
  {/if}
  {#if s.enabled && !s.test_mode && s.resendable_test>0}
   <section class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-500/20 bg-amber-500/[0.04] p-4">
    <div>
     <p class="text-sm font-semibold text-amber-800 dark:text-amber-300">{fmt(s.resendable_test)} event mode uji masih bisa dikirim ke produksi</p>
     <p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Aksi ini akan membuat event masuk antrean kembali dengan event ID yang sama.</p>
    </div>
    <button type="button" onclick={resendTest} disabled={busy} class={actionStyle}>Kirim ulang event uji <ArrowUpRight size={15}/></button>
   </section>
  {/if}

  <section class={panel+" p-4 sm:p-5"} aria-label="Filter rentang laporan">
   <div class="flex flex-wrap items-end justify-between gap-4">
    <div class="min-w-0">
     <p class="text-xs font-semibold text-neutral-800 dark:text-neutral-200">Periode event</p>
     <p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Berdasarkan waktu terjadinya event, bukan waktu pengirimannya.</p>
     <div class="mt-3 flex flex-wrap gap-1.5" role="group" aria-label="Preset periode">
      {#each [7,14,30,90] as days}
       <button type="button" onclick={()=>recent(days)} disabled={loading} class="rounded-md border border-neutral-200 px-3 py-1.5 text-xs font-semibold text-neutral-600 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.05]">{days} hari</button>
      {/each}
     </div>
    </div>
    <form class="flex flex-wrap items-end gap-2" onsubmit={e=>{e.preventDefault();applyPeriod();}}>
     <label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Dari<input type="date" bind:value={from} class={inputStyle+" mt-1"} required/></label>
     <label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Sampai<input type="date" bind:value={to} class={inputStyle+" mt-1"} required/></label>
     <button type="submit" disabled={loading} class="h-10 rounded-lg bg-brand-700 px-4 text-sm font-semibold text-white hover:bg-brand-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50">Terapkan</button>
    </form>
   </div>
   {#if periodError}<p role="alert" class="mt-2 text-xs text-rose-700 dark:text-rose-400">{periodError}</p>{/if}
  </section>

  <div class={"space-y-5 transition-opacity duration-150 "+(loading?"pointer-events-none opacity-60":"")} aria-busy={loading}>
   <section aria-label="Indikator pengiriman" class="grid grid-cols-2 gap-3 lg:grid-cols-5">
    <div class={panel+" col-span-2 p-5 lg:col-span-1 lg:p-5"}>
     <p class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Terkirim ke produksi</p>
     <p class="mt-2 text-[36px] font-semibold leading-tight tracking-tight tabular-nums text-neutral-950 dark:text-white">{fmt(a.sent)}</p>
     <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Event dalam periode</p>
    </div>
    {#each [
     {label:"Mode uji terkirim",value:a.test_sent,extra:"Terpisah dari produksi"},
     {label:"Gagal",value:a.failed,extra:"Event pada periode"},
     {label:"Menunggu",value:a.pending,extra:"Belum terkirim"},
     {label:"Total event",value:a.total,extra:"Termasuk dilewati"}
    ] as item}
     <div class={panel+" p-5"}>
      <p class="text-xs font-medium text-neutral-500 dark:text-neutral-400">{item.label}</p>
      <p class="mt-2 text-[28px] font-semibold tracking-tight tabular-nums text-neutral-950 dark:text-white">{fmt(item.value)}</p>
      <p class="mt-1 text-[11px] text-neutral-500 dark:text-neutral-400">{item.extra}</p>
     </div>
    {/each}
   </section>

   <div class="grid gap-4 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,0.75fr)]">
    <Trend days={a.days}/>
    <section class={panel+" flex flex-col p-5 sm:p-6"} aria-labelledby="value-title">
     <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-brand-700 dark:text-brand-400">Nilai yang tercatat</p>
     <h2 id="value-title" class="mt-1 text-base font-semibold text-neutral-950 dark:text-white">Purchase terkirim</h2>
     <p class="mt-4 break-words text-[30px] font-semibold leading-tight tracking-tight tabular-nums text-neutral-950 dark:text-white">{currency(a.purchase_value)}</p>
     <p class="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Jumlah `value` pada event Purchase berstatus terkirim ke produksi dalam periode. Bukan omzet seluruh santri dan bukan pendapatan yang diatribusikan Meta.</p>
     <div class="mt-auto border-t border-neutral-100 pt-5 dark:border-white/[0.07]">
      <p class="text-xs font-semibold text-neutral-700 dark:text-neutral-200">Antrean keseluruhan</p>
      <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Tidak mengikuti filter tanggal.</p>
      <dl class="mt-3 grid grid-cols-2 gap-3">
       {#each ["sent","pending","failed","skipped"] as key}
        <div><dt class="text-[11px] text-neutral-500">{statLabels[key]}</dt><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(s.counts[key]??0)}</dd></div>
       {/each}
      </dl>
      {#if totalQueue===0}
       <p class="mt-3 text-xs leading-relaxed text-amber-800 dark:text-amber-300">Antrean belum berisi event. Periksa apakah ada lead baru yang memenuhi syarat, status worker, dan konfigurasi CAPI.</p>
      {/if}
     </div>
    </section>
   </div>

   <section class={panel+" p-5 sm:p-6"} aria-labelledby="funnel-title">
    <div class="flex flex-wrap items-baseline justify-between gap-3">
     <div>
      <h2 id="funnel-title" class="text-base font-semibold text-neutral-950 dark:text-white">Event sepanjang funnel</h2>
      <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Distribusi event menurut tahapan CRM, bukan conversion funnel atribusi iklan.</p>
     </div>
     <span class="text-xs text-neutral-500">{fmt(a.total)} event periode ini</span>
    </div>
    <div class="mt-5 space-y-4">
     {#each a.events as ev,i}
      <div class="grid items-center gap-2 sm:grid-cols-[minmax(0,15rem)_minmax(0,1fr)_5rem]">
       <div class="min-w-0">
        <p class="truncate text-sm font-semibold text-neutral-800 dark:text-neutral-100">{eventLabels[ev.event]||ev.event}</p>
        <p class="truncate font-mono text-[11px] text-neutral-500 dark:text-neutral-400">{ev.event}</p>
       </div>
       <div class="flex h-2.5 w-full overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.07]" role="img" aria-label={`${ev.event}: ${ev.total} event, ${ev.sent} terkirim produksi`}>
        <span class="h-full bg-brand-600 dark:bg-brand-400" style:width={ev.sent/maxEvent*100+"%"}></span>
        <span class="h-full bg-slate-400" style:width={ev.test_sent/maxEvent*100+"%"}></span>
        <span class="h-full bg-rose-500" style:width={ev.failed/maxEvent*100+"%"}></span>
        <span class="h-full bg-amber-500" style:width={ev.pending/maxEvent*100+"%"}></span>
        <span class="h-full bg-neutral-300" style:width={ev.skipped/maxEvent*100+"%"}></span>
       </div>
       <p class="text-right text-sm font-semibold tabular-nums text-neutral-800 dark:text-neutral-100">{fmt(ev.total)}</p>
      </div>
     {/each}
    </div>
    <div class="mt-5 flex flex-wrap gap-x-4 gap-y-2 border-t border-neutral-100 pt-4 text-xs text-neutral-500 dark:border-white/[0.07] dark:text-neutral-400">
     <span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-brand-600 dark:bg-brand-400"></i>Produksi</span>
     <span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-slate-400"></i>Uji</span>
     <span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-rose-500"></i>Gagal</span>
     <span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-amber-500"></i>Menunggu</span>
     <span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-neutral-300"></i>Dilewati</span>
    </div>
   </section>

   <section class={panel+" overflow-hidden"} aria-labelledby="eventlog-title">
    <div class="px-5 py-5 sm:px-6">
     <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
       <h2 id="eventlog-title" class="text-base font-semibold text-neutral-950 dark:text-white">Log event terbaru</h2>
       <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">50 event paling baru dari keseluruhan data. Filter berikut berlaku pada 50 baris ini, bukan pada seluruh histori.</p>
      </div>
      <span class="rounded-md bg-neutral-100 px-2.5 py-1 text-xs font-medium text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">{filtered.length} ditampilkan</span>
     </div>
     <div class="mt-4 grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,12rem)_minmax(0,10rem)]">
      <label class="relative block">
       <span class="sr-only">Cari lead atau event ID</span>
       <Search size={15} class="pointer-events-none absolute left-3 top-3 text-neutral-400"/>
       <input type="search" bind:value={search} placeholder="Cari nama, ID lead, event ID" class={inputStyle+" pl-9"}/>
      </label>
      <label><span class="sr-only">Jenis event</span>
       <select bind:value={eventFilter} class={inputStyle}><option value="">Semua event</option>{#each a.events as ev}<option value={ev.event}>{ev.event}</option>{/each}</select>
      </label>
      <label><span class="sr-only">Status event</span>
       <select bind:value={statusFilter} class={inputStyle}><option value="">Semua status</option>{#each ["sent","pending","failed","skipped"] as state}<option value={state}>{statLabels[state]}</option>{/each}</select>
      </label>
     </div>
    </div>
    {#if s.events.length===0}
     <div class="flex flex-col items-center gap-2 px-5 py-12 text-center">
      <CircleHelp size={24} class="text-neutral-400"/>
      <p class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">Belum ada event di antrean CAPI</p>
      <p class="max-w-lg text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Event hanya dibuat jika CAPI aktif, lead memiliki nomor yang valid, dan event berada dalam jendela waktu yang didukung. Periksa konfigurasi serta proses worker sebelum menganggap data hilang.</p>
     </div>
    {:else if filtered.length===0}
     <p class="px-5 py-10 text-center text-sm text-neutral-500">Tidak ada event yang cocok dengan pencarian ini.</p>
    {:else}
     <div class="overflow-x-auto">
      <table class="w-full min-w-[780px] text-left text-sm">
       <thead class="border-y border-neutral-100 bg-neutral-50/70 text-[11px] text-neutral-500 dark:border-white/[0.06] dark:bg-white/[0.02] dark:text-neutral-400">
        <tr><th scope="col" class="py-3 pl-5 pr-3 font-semibold sm:pl-6">Lead / parameter</th><th scope="col" class="px-3 py-3 font-semibold">Event</th><th scope="col" class="px-3 py-3 font-semibold">Waktu event</th><th scope="col" class="px-3 py-3 text-right font-semibold">Nilai tersimpan</th><th scope="col" class="py-3 pl-3 pr-5 font-semibold sm:pr-6">Status</th></tr>
       </thead>
       <tbody class="divide-y divide-neutral-100 dark:divide-white/[0.06]">
        {#each filtered as e (e.id)}
         <tr class="align-top hover:bg-neutral-50/60 dark:hover:bg-white/[0.02]">
          <td class="py-3 pl-5 pr-3 sm:pl-6">
           <a href={"/app/leads/"+e.lead_id} use:inertia class="font-semibold text-neutral-900 hover:text-brand-700 dark:text-white dark:hover:text-brand-400">{e.lead_nama}</a>
           <p class="mt-0.5 text-[11px] tabular-nums text-neutral-400">Lead L-{e.lead_id}</p>
           <details class="mt-2 max-w-[22rem]">
            <summary class="w-fit cursor-pointer rounded text-xs font-semibold text-brand-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Lihat parameter tersimpan</summary>
            <dl class="mt-2 space-y-1 rounded-lg border border-neutral-200 bg-neutral-50 p-3 text-[11px] dark:border-white/10 dark:bg-neutral-900">
             <div><dt class="text-neutral-500">event_id</dt><dd class="break-all font-mono text-neutral-800 dark:text-neutral-200">{e.event_id}</dd></div>
             <div><dt class="text-neutral-500">event_name</dt><dd class="font-mono">{e.event}</dd></div>
             <div><dt class="text-neutral-500">event_time (WIB)</dt><dd>{formatDateTime(e.event_time)}</dd></div>
             <div><dt class="text-neutral-500">action_source</dt><dd class="font-mono">system_generated</dd></div>
             <div><dt class="text-neutral-500">custom_data.value (snapshot)</dt><dd>{e.event==="Purchase" ? currency(e.value) : "Tidak disimpan per event"}</dd></div>
             <div><dt class="text-neutral-500">fbtrace_id (respons Meta)</dt><dd class="break-all font-mono">{e.fbtrace_id||"Belum tersedia"}</dd></div>
             <div><dt class="text-neutral-500">waktu terkirim</dt><dd>{e.sent_at||"Belum tercatat"}</dd></div>
             <p class="pt-1 text-neutral-500">URL sumber, nilai non-Purchase, dan data hash tidak disimpan sebagai snapshot payload historis.</p>
            </dl>
           </details>
          </td>
          <td class="px-3 py-3">
           <p class="font-mono text-xs font-semibold text-neutral-800 dark:text-neutral-200">{e.event}</p>
           {#if e.test_mode}<span class="mt-1 inline-block rounded bg-blue-500/10 px-2 py-0.5 text-[10px] font-semibold text-blue-700 dark:text-blue-300">Mode uji</span>{/if}
          </td>
          <td class="px-3 py-3 tabular-nums text-xs text-neutral-600 dark:text-neutral-300">{formatDateTime(e.event_time)}</td>
          <td class="px-3 py-3 text-right text-xs tabular-nums text-neutral-700 dark:text-neutral-300">{e.event==="Purchase"?(e.value>0?currency(e.value):"Rp0, nominal belum tercatat"):"Tidak disimpan"}</td>
          <td class="py-3 pl-3 pr-5 sm:pr-6">
           <span class={"inline-flex rounded-md px-2 py-1 text-xs font-semibold "+(statTone[e.status]||"bg-neutral-100 text-neutral-700")}>{statLabels[e.status]||e.status}</span>
           {#if e.attempts>1}<p class="mt-1 text-[11px] text-neutral-500">{e.attempts} kali percobaan</p>{/if}
           {#if e.last_error}<p class="mt-2 max-w-xs break-words text-[11px] leading-snug text-rose-700 dark:text-rose-300">{e.last_error}</p>{/if}
          </td>
         </tr>
        {/each}
       </tbody>
      </table>
     </div>
    {/if}
   </section>

   <div class="grid gap-4 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]">
    <section class={panel+" p-5 sm:p-6"} aria-labelledby="parameters-title">
     <h2 id="parameters-title" class="text-base font-semibold text-neutral-950 dark:text-white">Parameter event yang dibentuk aplikasi</h2>
     <p class="mt-1 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Kontrak payload dari kode pengiriman. Ini bukan rekaman lengkap parameter setiap event historis.</p>
     <dl class="mt-4 divide-y divide-neutral-100 text-sm dark:divide-white/[0.06]">
      {#each [
       {name:"event_name",desc:"Lead, CompleteRegistration, InitiateCheckout, AddPaymentInfo, Purchase"},
       {name:"event_time",desc:"Waktu milestone, dikirim sebagai Unix timestamp"},
       {name:"event_id",desc:"ID unik tetap untuk mencegah duplikasi event"},
       {name:"action_source",desc:"system_generated"},
       {name:"event_source_url",desc:"URL sumber lead atau URL default, jika tersedia"},
       {name:"user_data",desc:"external_id, country, serta ph/fn/ln/ct/ge jika tersedia; di-hash SHA-256"},
       {name:"custom_data.currency",desc:"IDR"},
       {name:"custom_data.value",desc:"Purchase dari nominal terverifikasi/tagihan. Event lain dari tagihan saat dikirim jika diketahui"},
       {name:"custom_data.lead_event_source",desc:"Ruang Sanad Dashboard"},
       {name:"custom_data.event_source",desc:"crm"},
      ] as param}
       <div class="grid gap-1 py-2.5 sm:grid-cols-[11.5rem_minmax(0,1fr)] sm:gap-3">
        <dt class="break-all font-mono text-xs font-semibold text-neutral-800 dark:text-neutral-200">{param.name}</dt>
        <dd class="text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">{param.desc}</dd>
       </div>
      {/each}
     </dl>
     <p class="mt-3 rounded-lg bg-neutral-50 px-3 py-3 text-xs leading-relaxed text-neutral-500 dark:bg-white/[0.03] dark:text-neutral-400">Meta tidak mengembalikan nilai atribusi, match quality, atau parameter lengkap setiap event dalam respons yang saat ini disimpan aplikasi. Verifikasi penerimaan sebenarnya melalui Events Manager.</p>
    </section>
    <section class={panel+" p-5 sm:p-6"} aria-labelledby="configuration-title">
     <h2 id="configuration-title" class="text-base font-semibold text-neutral-950 dark:text-white">Konfigurasi dan pemetaan</h2>
     <dl class="mt-4 space-y-3 text-sm">
      <div class="flex justify-between gap-3"><dt class="text-neutral-500">CAPI</dt><dd class="font-semibold text-neutral-900 dark:text-white">{s.enabled?"Terhubung secara konfigurasi":"Belum aktif"}</dd></div>
      <div class="flex justify-between gap-3"><dt class="text-neutral-500">Mode</dt><dd class="font-medium text-neutral-900 dark:text-white">{s.test_mode?"Uji":"Produksi"}</dd></div>
      <div class="flex justify-between gap-3"><dt class="text-neutral-500">appsecret_proof</dt><dd class="text-neutral-900 dark:text-white">{s.app_secret_proof?"Dipakai":"Tidak dipakai"}</dd></div>
      <div class="flex justify-between gap-3"><dt class="text-neutral-500">URL default</dt><dd class="max-w-[14rem] truncate text-right text-neutral-900 dark:text-white" title={s.event_source_url}>{s.event_source_url||"Tidak diisi"}</dd></div>
      <div class="flex justify-between gap-3"><dt class="text-neutral-500">Sumber tanpa URL</dt><dd class="font-medium tabular-nums text-neutral-900 dark:text-white">{fmt(s.sources_without_url.length)}</dd></div>
     </dl>
     <div class="mt-6 border-t border-neutral-100 pt-5 dark:border-white/[0.07]">
      <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Pemetaan tahap CRM</h3>
      <ol class="mt-3 space-y-3">
       {#each s.mapping as m,i}
        <li class="flex items-center gap-3">
         <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-neutral-100 text-xs font-semibold tabular-nums text-neutral-600 dark:bg-white/[0.07] dark:text-neutral-300">{i+1}</span>
         <span class="min-w-0 flex-1 text-xs text-neutral-700 dark:text-neutral-300">{stageName(m.tahap)}</span>
         <span class="shrink-0 text-right font-mono text-[11px] text-neutral-500 dark:text-neutral-400">{m.event}</span>
        </li>
       {/each}
      </ol>
     </div>
     <p class="mt-5 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Worker mengantre milestone lead dan mengirimnya ke Meta. Event berusia lebih dari 7 hari tidak dapat dikirim ulang.</p>
    </section>
   </div>

   <p class="pb-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Indikator di halaman ini berasal dari database antrean CAPI lokal. Status “terkirim” berarti API Meta menerima request, bukan jaminan event berhasil dicocokkan, diatribusikan, atau dilaporkan sebagai konversi di Ads Manager. Nilai dan status dapat berubah setelah pengiriman atau retry.</p>
  </div>
 </main>
</AppLayout>