<script lang="ts">
 import { inertia, router } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import type { User } from "@lib/types";
 import { STAGE_LABELS, LOST_REASON_LABELS, todayWIB } from "@lib/leads";
 import { Activity, ArrowUpRight, CalendarDays, CheckCircle2, ChevronRight, CircleAlert, Clock3, MessageCircle, Users } from "lucide-svelte";

 interface Count { key: string; count: number }
 interface FunnelStage { stage: string; reached: number; conversion: number }
 interface Report {
  total_leads: number; won: number; lost: number; open: number; do_not_contact: number;
  funnel: FunnelStage[]; lost_reasons: Count[]; objections: Count[];
  followup_sent: number;
 }
 interface Day { day: string; leads: number; closing: number }
 interface Priority { id: number; name: string; stage: string; reason: string; activity_at: string }
 interface Dashboard {
  from: string; to: string; report: Report; closing_period: number;
  previous_leads: number; previous_closing: number; review_count: number; daily: Day[];
  followup_pending: number; followup_approved: number; followup_failed: number;
  followup_eligible: number; followup_replied: number; priorities: Priority[]; insights: string[];
 }
 interface Props { user?: User; dashboard: Dashboard }
 let props: Props = $props();
 let d = $derived(props.dashboard);
 let canApproveFollowup = $derived(props.user?.role === "cs" || props.user?.role === "super_admin");
 let r = $derived(d.report);
 let customFrom = $state(props.dashboard.from);
 let customTo = $state(props.dashboard.to);
 let customOpen = $state(false);
 let validation = $state("");
 let loading = $state(false);
 const presetClass = "rounded-lg border border-neutral-300 px-3 py-2 text-sm hover:bg-brand-500/10 dark:border-neutral-700";
 const selectedClass = " border-brand-600 bg-brand-600 text-white hover:bg-brand-700 dark:border-brand-600";
 const box = "rounded-xl border border-neutral-200/80 dark:border-white/[0.07] bg-white dark:bg-neutral-925/50 p-4 sm:p-5";
 const fmt = (n: number) => n.toLocaleString("id-ID");
 const pct = (n: number, total: number) => total ? (n / total * 100).toFixed(1).replace(".", ",") + "%" : "N/A";
 function dateOffset(offset: number) {
  const v = new Date(todayWIB() + "T12:00:00+07:00");
  v.setUTCDate(v.getUTCDate() + offset);
  return v.toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" });
 }
 let activePreset = $derived.by(() => {
  if (d.from === todayWIB() && d.to === todayWIB()) return 1;
  if (d.from === dateOffset(-1) && d.to === dateOffset(-1)) return -1;
  for (const n of [7,14,30]) {
   if (d.from === dateOffset(1-n) && d.to === todayWIB()) return n;
  }
  return 0;
 });
 function applyRange(from: string, to: string) {
  if (!from || !to || from > to) { validation = "Tanggal mulai harus sebelum atau sama dengan tanggal akhir."; return; }
  const start = new Date(from + "T00:00:00+07:00");
  const end = new Date(to + "T00:00:00+07:00");
  if ((end.getTime() - start.getTime()) / 86400000 >= 366) { validation = "Maksimal 366 hari."; return; }
  validation = "";
  router.get("/app/leads/dashboard", { from, to }, { preserveScroll: true, onStart: () => { loading = true; }, onFinish: () => { loading = false; } });
 }
 function preset(days: number) {
  customOpen = false;
  const end = days === -1 ? dateOffset(-1) : todayWIB();
  const from = days === -1 ? end : dateOffset(1 - days);
  applyRange(from, end);
 }
 function openLeads(extra: Record<string, string> = {}) {
  const params = new URLSearchParams({ status: "all", from: d.from, to: d.to, ...extra });
  router.get("/app/leads?" + params.toString());
 }
 function openClosing(day?: string) {
  const params = new URLSearchParams({ status: "won", date_field: "converted", from: day || d.from, to: day || d.to });
  router.get("/app/leads?" + params.toString());
 }
 let maxDaily = $derived(Math.max(1, ...d.daily.flatMap(x => [x.leads, x.closing])));
 let maxFunnel = $derived(Math.max(1, ...r.funnel.map(x => x.reached)));
 let maxLost = $derived(Math.max(1, ...r.lost_reasons.map(x => x.count)));
 let maxObjection = $derived(Math.max(1, ...r.objections.map(x => x.count)));
 let statusItems = $derived([
  { label: "Berjalan", key: "open", count: r.open, color: "#64748b" },
  { label: "Jadi santri", key: "won", count: r.won, color: "#168260" },
  { label: "Lost", key: "lost", count: r.lost, color: "#d19a39" },
  { label: "Jangan hubungi", key: "do_not_contact", count: r.do_not_contact, color: "#b75e65" },
 ]);
 let pieBackground = $derived.by(() => {
  let start = 0;
  const parts: string[] = [];
  for (const item of statusItems) {
   const end = start + (r.total_leads ? item.count / r.total_leads * 100 : 0);
   parts.push(item.color + " " + start + "% " + end + "%");
   start = end;
  }
  return "conic-gradient(" + parts.join(",") + ")";
 });
 function comparison(current: number, previous: number) {
  if (!previous) return current === 0 ? "Tetap 0" : "Tidak ada pembanding";
  const diff = (current - previous) / previous * 100;
  return (diff > 0 ? "+" : "") + diff.toFixed(1).replace(".", ",") + "% vs periode sebelumnya";
 }
 let cards = $derived([
  { title: "Leads masuk", value: fmt(r.total_leads), detail: comparison(r.total_leads, d.previous_leads), kind: "created", icon: Users },
  { title: "Closing periode", value: fmt(d.closing_period), detail: comparison(d.closing_period, d.previous_closing), kind: "closing", icon: CheckCircle2 },
  { title: "Konversi cohort", value: pct(r.won, r.total_leads), detail: fmt(r.won) + " dari " + fmt(r.total_leads) + " lead masuk", kind: "cohort", icon: Activity },
  { title: "Masih berjalan", value: fmt(r.open), detail: "Status terkini dari cohort", kind: "open", icon: Clock3 },
  { title: "Lost", value: fmt(r.lost), detail: "Status terkini dari cohort", kind: "lost", icon: CircleAlert },
  { title: "Perlu ditinjau", value: fmt(d.review_count), detail: "Lead aktif dari cohort", kind: "review", icon: MessageCircle },
 ]);
 function cardClick(kind: string) {
  if (kind === "closing") return openClosing();
  if (kind === "review") return openLeads({ status: "open", review: "true" });
  if (kind === "cohort") return openLeads({ status: "won" });
  if (kind === "open" || kind === "lost") return openLeads({ status: kind });
  openLeads();
 }
 function readableDate(day: string) {
  return new Date(day + "T12:00:00+07:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", timeZone: "Asia/Jakarta" });
 }
</script>

<AppLayout user={props.user}>
 <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 space-y-5">
  <header class="flex flex-wrap items-start justify-between gap-4">
   <div>
    <h1 class="text-2xl sm:text-3xl font-bold tracking-tight text-neutral-900 dark:text-white">Dashboard CRM Leads</h1>
    <p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Performa lead WhatsApp dan pekerjaan CS · {readableDate(d.from)} sampai {readableDate(d.to)} (WIB)</p>
   </div>
   <a href="/app/leads" use:inertia class="inline-flex items-center gap-2 rounded-lg border border-neutral-200 px-3 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800">Buka Leads <ArrowUpRight size={15} /></a>
  </header>

  <section class={box + " space-y-3"} aria-label="Filter periode">
   <div class="flex items-center gap-2 text-sm font-semibold text-neutral-800 dark:text-neutral-100"><CalendarDays size={17} /> Filter periode</div>
   <div class="flex flex-wrap gap-2">
    <button type="button" onclick={() => preset(1)} aria-pressed={activePreset === 1} class={presetClass + (activePreset === 1 ? selectedClass : "")}>Hari ini</button>
    <button type="button" onclick={() => preset(-1)} aria-pressed={activePreset === -1} class={presetClass + (activePreset === -1 ? selectedClass : "")}>Kemarin</button>
    {#each [7,14,30] as days}
     <button type="button" onclick={() => preset(days)} aria-pressed={activePreset === days} class={presetClass + (activePreset === days ? selectedClass : "")}>{days} hari</button>
    {/each}
    <button type="button" onclick={() => (customOpen = !customOpen)} aria-expanded={customOpen} aria-pressed={activePreset === 0} class={presetClass + (activePreset === 0 || customOpen ? selectedClass : "")}>Custom</button>
   </div>
   {#if customOpen}
    <form class="flex flex-wrap items-end gap-3" onsubmit={(e) => { e.preventDefault(); applyRange(customFrom, customTo); }}>
     <label class="text-xs text-neutral-500">Dari<input aria-label="Tanggal mulai" type="date" bind:value={customFrom} class="mt-1 block rounded-lg border border-neutral-300 bg-transparent p-2 text-sm dark:border-neutral-700" /></label>
     <label class="text-xs text-neutral-500">Sampai<input aria-label="Tanggal akhir" type="date" bind:value={customTo} class="mt-1 block rounded-lg border border-neutral-300 bg-transparent p-2 text-sm dark:border-neutral-700" /></label>
     <button type="submit" class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white">Terapkan</button>
    </form>
   {/if}
   {#if validation}<p role="alert" class="text-sm text-error">{validation}</p>{/if}
   {#if loading}<p role="status" aria-live="polite" class="text-sm text-neutral-500">Memuat data dashboard...</p>{/if}
  </section>

  <section aria-label="Ringkasan performa" class="grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-6">
   {#each cards as c}
    <button type="button" onclick={() => cardClick(c.kind)} class={box + " group text-left transition hover:border-brand-500/50 focus-visible:ring-2 focus-visible:ring-brand-500"}>
     <div class="flex items-center justify-between gap-2 text-xs text-neutral-500 dark:text-neutral-400">
      <span>{c.title}</span><c.icon size={17} />
     </div>
     <p class="mt-3 text-2xl font-bold tabular-nums text-neutral-900 dark:text-white">{c.value}</p>
     <p class="mt-1 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">{c.detail}</p>
    </button>
   {/each}
  </section>

  <section class={box}>
   <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
    <div><h2 class="font-semibold text-neutral-900 dark:text-white">Tren lead masuk & closing</h2><p class="mt-1 text-xs text-neutral-500">Dikelompokkan berdasarkan tanggal masing-masing peristiwa, bukan cohort yang sama.</p></div>
    <div class="flex gap-3 text-xs text-neutral-600 dark:text-neutral-400"><span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-brand-500"></i>Leads</span><span><i class="mr-1 inline-block h-2.5 w-2.5 rounded-sm bg-emerald-500"></i>Closing</span></div>
   </div>
   <div class="overflow-x-auto pb-2">
    <div class="flex min-w-max items-end gap-2 border-b border-neutral-200 pb-3 dark:border-neutral-700" style="height: 236px">
     {#each d.daily as day}
      <div class="flex h-full w-14 shrink-0 flex-col justify-end gap-1">
       <div class="flex h-44 items-end justify-center gap-1">
        <button type="button" onclick={() => openLeads({ from: day.day, to: day.day })} title={readableDate(day.day) + ": " + fmt(day.leads) + " lead masuk"} aria-label={readableDate(day.day) + ", " + fmt(day.leads) + " lead masuk"} class="w-5 rounded-t-sm bg-brand-500 hover:bg-brand-600 focus-visible:ring-2 focus-visible:ring-brand-500" style={"height:" + Math.max(day.leads / maxDaily * 100, day.leads ? 3 : 0) + "%"}></button>
        <button type="button" onclick={() => openClosing(day.day)} title={readableDate(day.day) + ": " + fmt(day.closing) + " closing"} aria-label={readableDate(day.day) + ", " + fmt(day.closing) + " closing"} class="w-5 rounded-t-sm bg-emerald-500 hover:bg-emerald-600 focus-visible:ring-2 focus-visible:ring-500" style={"height:" + Math.max(day.closing / maxDaily * 100, day.closing ? 3 : 0) + "%"}></button>
       </div>
       <span class="text-center text-[11px] text-neutral-500">{readableDate(day.day)}</span>
      </div>
     {/each}
    </div>
   </div>
  </section>

  <div class="grid gap-4 lg:grid-cols-5">
   <section class={box + " lg:col-span-3"}>
    <h2 class="font-semibold text-neutral-900 dark:text-white">Funnel pendaftaran</h2>
    <p class="mb-4 mt-1 text-xs text-neutral-500">Estimasi tahap yang dicapai berdasarkan posisi pipeline saat ini, bukan histori transisi terverifikasi.</p>
    <div class="space-y-2.5">
     {#each r.funnel as f, i}
      <button type="button" onclick={() => f.stage === "won" ? openLeads({status:"won"}) : openLeads({reached_stage:f.stage})} class="grid w-full grid-cols-[6.5rem_1fr_4.5rem] items-center gap-2 text-left text-xs sm:grid-cols-[11rem_1fr_6rem]" aria-label={(STAGE_LABELS[f.stage] || f.stage) + ": " + f.reached + " lead"}>
       <span class="truncate text-neutral-600 dark:text-neutral-300">{STAGE_LABELS[f.stage] || f.stage}</span>
       <span class="h-3.5 overflow-hidden rounded-sm bg-neutral-100 dark:bg-neutral-800"><span class={"block h-full rounded-sm " + (f.stage === "won" ? "bg-emerald-500" : "bg-brand-500")} style={"width:" + (f.reached / maxFunnel * 100) + "%"}></span></span>
       <span class="text-right font-mono text-neutral-600 dark:text-neutral-300">{fmt(f.reached)}{#if i > 0}<span class="hidden text-neutral-400 sm:inline"> · {Math.round(f.conversion * 100)}%</span>{/if}</span>
      </button>
     {/each}
    </div>
    <p class="mt-3 text-xs text-neutral-500">Klik tahap untuk melihat seluruh lead dalam cohort yang dianggap sudah mencapainya berdasarkan aturan funnel.</p>
   </section>
   <section class={box + " lg:col-span-2"}>
    <h2 class="font-semibold text-neutral-900 dark:text-white">Komposisi status</h2>
    <p class="mt-1 text-xs text-neutral-500">Status terkini dari leads yang masuk dalam periode.</p>
    {#if r.total_leads > 0}
     <div class="mx-auto my-6 h-40 w-40 rounded-full" style={"background:" + pieBackground} role="img" aria-label="Distribusi status leads"><div class="relative left-8 top-8 flex h-24 w-24 items-center justify-center rounded-full bg-white text-center text-xl font-bold dark:bg-neutral-925">{fmt(r.total_leads)}</div></div>
    {:else}<p class="py-12 text-center text-sm text-neutral-500">Belum ada leads untuk periode ini.</p>{/if}
    <div class="space-y-1">
     {#each statusItems as item}
      <button type="button" onclick={() => openLeads({status:item.key})} class="flex w-full items-center justify-between rounded-lg px-2 py-2 text-sm hover:bg-neutral-50 dark:hover:bg-neutral-800">
       <span class="flex items-center gap-2 text-neutral-700 dark:text-neutral-300"><i class="h-2.5 w-2.5 rounded-full" style={"background:" + item.color}></i>{item.label}</span>
       <span class="font-mono text-neutral-600 dark:text-neutral-300">{fmt(item.count)} <span class="text-xs text-neutral-400">({pct(item.count, r.total_leads)})</span></span>
      </button>
     {/each}
    </div>
   </section>
  </div>

  <div class="grid gap-4 lg:grid-cols-2">
   <section class={box}>
    <h2 class="font-semibold text-neutral-900 dark:text-white">Alasan lost</h2>
    <p class="mb-4 mt-1 text-xs text-neutral-500">Alasan yang tercatat manual pada leads lost.</p>
    {#if r.lost_reasons.length === 0}<p class="py-8 text-sm text-neutral-500">Belum ada lead lost pada cohort ini.</p>{/if}
    <div class="space-y-3">
     {#each r.lost_reasons as x}
      <button type="button" onclick={() => openLeads({ status:"lost", lost_reason:x.key })} class="block w-full space-y-1 text-left hover:opacity-80">
       <div class="flex justify-between text-sm"><span class="text-neutral-700 dark:text-neutral-300">{LOST_REASON_LABELS[x.key] || x.key || "Tidak diketahui"}</span><strong class="font-mono">{fmt(x.count)}</strong></div>
       <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-amber-500" style={"width:" + x.count / maxLost * 100 + "%"}></div></div>
      </button>
     {/each}
    </div>
   </section>
   <section class={box}>
    <h2 class="font-semibold text-neutral-900 dark:text-white">Keberatan utama menurut AI</h2>
    <p class="mb-4 mt-1 text-xs text-neutral-500">Hasil analisis AI terakhir, berupa dugaan dan bukan alasan lost terkonfirmasi.</p>
    {#if r.objections.length === 0}<p class="py-8 text-sm text-neutral-500">Belum tersedia analisis AI.</p>{/if}
    <div class="space-y-3">
     {#each r.objections as x}
      <div class="space-y-1">
       <div class="flex justify-between gap-2 text-sm"><span class="break-words text-neutral-700 dark:text-neutral-300">{x.key.replaceAll("_", " ")}</span><strong class="font-mono">{fmt(x.count)}</strong></div>
       <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-brand-500" style={"width:" + x.count / maxObjection * 100 + "%"}></div></div>
      </div>
     {/each}
    </div>
   </section>
  </div>

  <div class="grid gap-4 lg:grid-cols-2">
   <section class={box}>
    <div class="flex items-center justify-between"><h2 class="font-semibold text-neutral-900 dark:text-white">Monitoring follow-up</h2>{#if canApproveFollowup}<a href="/app/leads/followups" use:inertia class="text-xs font-medium text-brand-600 dark:text-brand-400">Buka follow-up <ChevronRight size={13} class="inline" /></a>{/if}</div>
    <p class="mt-1 text-xs text-neutral-500">Antrean saat ini, terpisah dari filter akuisisi.</p>
    <div class="mt-4 grid grid-cols-3 gap-2 text-center">
     {#each [{title:"Menunggu approval",value:d.followup_pending},{title:"Disetujui",value:d.followup_approved},{title:"Gagal",value:d.followup_failed}] as x}
      <div class="rounded-lg bg-neutral-50 p-3 dark:bg-neutral-800/50"><p class="text-xl font-bold">{fmt(x.value)}</p><p class="mt-1 text-xs text-neutral-500">{x.title}</p></div>
     {/each}
    </div>
    <div class="mt-4 rounded-lg border border-neutral-200 p-3 dark:border-neutral-700">
     <p class="text-sm text-neutral-700 dark:text-neutral-300">Follow-up terkirim dalam periode: <strong>{fmt(r.followup_sent)}</strong></p>
     <p class="mt-2 text-sm text-neutral-700 dark:text-neutral-300">Balasan setelah 72 jam observasi: <strong>{fmt(d.followup_replied)} / {fmt(d.followup_eligible)}</strong> ({pct(d.followup_replied, d.followup_eligible)})</p>
     <p class="mt-1 text-xs text-neutral-500">Hanya follow-up dengan jendela 72 jam yang sudah lengkap dihitung untuk rasio respons.</p>
    </div>
   </section>
   <section class={box}>
    <h2 class="font-semibold text-neutral-900 dark:text-white">Prioritas CS saat ini</h2>
    <p class="mb-3 mt-1 text-xs text-neutral-500">Antrean seluruh lead aktif, tidak dibatasi tanggal lead masuk. Maksimal 12 item.</p>
    {#if d.priorities.length === 0}<p class="py-8 text-sm text-neutral-500">Belum ada lead yang membutuhkan tindakan berdasarkan aturan ini.</p>{/if}
    <div class="max-h-80 divide-y divide-neutral-100 overflow-y-auto dark:divide-neutral-800">
     {#each d.priorities as p}
      <a href={"/app/leads/" + p.id} use:inertia class="flex items-center justify-between gap-3 py-2.5 text-sm hover:text-brand-600 dark:hover:text-brand-400">
       <div class="min-w-0"><p class="truncate font-medium">{p.name}</p><p class="mt-0.5 text-xs text-neutral-500">{p.reason} · {STAGE_LABELS[p.stage] || p.stage}</p></div><ChevronRight size={16} class="shrink-0 text-neutral-400" />
      </a>
     {/each}
    </div>
   </section>
  </div>

  <section class={box}>
   <h2 class="font-semibold text-neutral-900 dark:text-white">Insight periode</h2>
   <p class="mt-1 text-xs text-neutral-500">Dihitung dari data CRM, bukan prediksi AI atau atribusi iklan.</p>
   <ul class="mt-3 grid gap-2 md:grid-cols-2">
    {#each d.insights as insight, i}
     <li class="flex gap-3 rounded-lg bg-neutral-50 p-3 text-sm text-neutral-700 dark:bg-neutral-800/40 dark:text-neutral-300"><span class="font-mono text-brand-600">{String(i+1).padStart(2,"0")}</span><span>{insight}</span></li>
    {/each}
   </ul>
  </section>
  <p class="pb-3 text-xs text-neutral-500">Data hanya berasal dari leads WhatsApp yang tercatat di CRM. Belum mencakup seluruh leads marketing, biaya iklan, atau CAC. Angka konversi cohort dapat berubah setelah lead lama closing.</p>
 </div>
</AppLayout>
