<script lang="ts">
 import { inertia } from "@inertiajs/svelte";
 import AppLayout from "@layouts/AppLayout.svelte";
 import OnboardingNotices from "../../components/OnboardingNotices.svelte";
 import type { User } from "@lib/types";
 import { CalendarClock, TriangleAlert, CheckCircle2, Bell, Users, BookOpen, ArrowRight, ChartColumnIncreasing } from "lucide-svelte";

 interface Report {
  as_of:string;total:number;in_progress:number;completed:number;cancelled:number;
  due_cases:number;on_time_cases:number;sla_rate:number|null;
  at_risk:number;overdue:number;avg_waiting_days:number|null;
  alert_eligible:number;alert_due:number;alert_on_time:number;
  alert_pending:number;alert_late:number;alert_on_time_rate:number|null;
  stages:Record<string,number>;
 }
 interface Props {user?:User;report:Report;notices:{id:number;type:string;title:string;message:string;action_url:string;read:boolean;created_at:string}[]}
 let props:Props=$props();
 let report=$derived(props.report);
 let user=$derived(props.user);
 const fmt=(n:number)=>n.toLocaleString("id-ID");
 const rate=(n:number|null)=>n===null?"N/A":n.toLocaleString("id-ID",{maximumFractionDigits:1})+"%";
 const wait=(n:number|null)=>n===null?"N/A":n.toLocaleString("id-ID",{maximumFractionDigits:1})+" hari";
 const stamp=(s:string)=>new Date(s).toLocaleString("id-ID",{timeZone:"Asia/Jakarta",day:"numeric",month:"long",year:"numeric",hour:"2-digit",minute:"2-digit"});
 const columns=[
  {stage:"onboarding_baru",label:"Onboarding Baru"},
  {stage:"matching_kelas",label:"Matching Kelas"},
  {stage:"waiting_kuota",label:"Waiting Kuota"},
  {stage:"siap_mulai",label:"Siap Mulai"},
  {stage:"masuk_kelas",label:"Masuk Kelas"}
 ];
</script>

<AppLayout {user}>
 <main class="mx-auto max-w-[1450px] space-y-6 px-4 py-6 sm:px-6">
  <header class="flex flex-wrap items-start justify-between gap-4">
   <div>
    <p class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-brand-700 dark:text-brand-400"><ChartColumnIncreasing size={15}/> Monitoring Onboarding</p>
    <h1 class="mt-1 text-2xl font-semibold text-neutral-950 dark:text-white">KPI & SLA 14 Hari</h1>
    <p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Semua kasus sejak onboarding diaktifkan, zona waktu WIB. Target SLA dan Vacancy Alert tepat waktu: 100%.</p>
    <p class="mt-1 text-xs text-neutral-500">Diperbarui saat halaman dibuka · {stamp(report.as_of)} WIB</p>
   </div>
   <div class="flex flex-wrap gap-2">
    <a href="/app/onboarding" use:inertia class="inline-flex items-center gap-2 rounded-lg border border-neutral-200 px-3 py-2 text-sm hover:bg-neutral-50 dark:border-white/15 dark:hover:bg-white/10"><Users size={15}/> Kanban <ArrowRight size={14}/></a>
    <a href="/app/onboarding/classes" use:inertia class="inline-flex items-center gap-2 rounded-lg border border-neutral-200 px-3 py-2 text-sm hover:bg-neutral-50 dark:border-white/15 dark:hover:bg-white/10"><BookOpen size={15}/> Kontrol Kuota</a>
   </div>
  </header>

  <OnboardingNotices items={props.notices??[]}/>

  <section aria-label="KPI utama" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
   <div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
    <p class="flex items-center gap-2 text-sm text-neutral-500"><CheckCircle2 size={16}/> SLA Tepat Waktu</p>
    <p class="mt-3 text-3xl font-semibold tabular-nums">{rate(report.sla_rate)}</p>
    <p class="mt-2 text-xs text-neutral-500">{fmt(report.on_time_cases)} tepat waktu dari {fmt(report.due_cases)} kasus selesai atau jatuh tempo</p>
   </div>
   <div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
    <p class="flex items-center gap-2 text-sm text-neutral-500"><Bell size={16}/> Vacancy Alert Tepat Waktu</p>
    <p class="mt-3 text-3xl font-semibold tabular-nums">{rate(report.alert_on_time_rate)}</p>
    <p class="mt-2 text-xs text-neutral-500">{fmt(report.alert_on_time)} tepat waktu dari {fmt(report.alert_due)} kewajiban matang (D8+)</p>
   </div>
   <div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
    <p class="flex items-center gap-2 text-sm text-neutral-500"><CalendarClock size={16}/> Rata-rata Waktu Tunggu</p>
    <p class="mt-3 text-3xl font-semibold tabular-nums">{wait(report.avg_waiting_days)}</p>
    <p class="mt-2 text-xs text-neutral-500">Hanya kasus selesai, dari tanggal registrasi ke tanggal mulai aktual (WIB)</p>
   </div>
   <div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
    <p class="flex items-center gap-2 text-sm text-neutral-500"><Users size={16}/> Kasus Tercatat</p>
    <p class="mt-3 text-3xl font-semibold tabular-nums">{fmt(report.total)}</p>
    <p class="mt-2 text-xs text-neutral-500">{fmt(report.in_progress)} aktif · {fmt(report.completed)} selesai · {fmt(report.cancelled)} batal</p>
   </div>
  </section>

  <section class="grid gap-4 lg:grid-cols-2">
   <div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
    <h2 class="text-base font-semibold">Perlu perhatian — SLA santri</h2>
    <div class="mt-4 grid grid-cols-2 gap-3">
     <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-500/10">
      <p class="flex items-center gap-2 text-xs text-amber-800 dark:text-amber-200"><CalendarClock size={14}/> Mendekati deadline</p>
      <p class="mt-2 text-2xl font-semibold tabular-nums text-amber-900 dark:text-amber-100">{fmt(report.at_risk)}</p>
      <p class="mt-1 text-xs text-amber-800 dark:text-amber-200">Kasus aktif sejak D11 sebelum deadline</p>
     </div>
     <div class="rounded-lg bg-red-50 p-4 dark:bg-red-500/10">
      <p class="flex items-center gap-2 text-xs text-red-800 dark:text-red-200"><TriangleAlert size={14}/> Overdue</p>
      <p class="mt-2 text-2xl font-semibold tabular-nums text-red-900 dark:text-red-100">{fmt(report.overdue)}</p>
      <p class="mt-1 text-xs text-red-800 dark:text-red-200">Kasus aktif melampaui akhir D14</p>
     </div>
    </div>
    <p class="mt-4 text-xs text-neutral-500">Notifikasi aplikasi dikirim satu kali per milestone dan penerima; Admin Kelas serta Super Admin menerima eskalasi sesuai fase. Tindak lanjut dilakukan di Kanban.</p>
   </div>
   <div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
    <h2 class="text-base font-semibold">Perlu perhatian — Vacancy Alert</h2>
    <div class="mt-4 grid grid-cols-2 gap-3">
     <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-500/10">
      <p class="text-xs text-amber-800 dark:text-amber-200">Belum dicatat terkirim</p>
      <p class="mt-2 text-2xl font-semibold tabular-nums text-amber-900 dark:text-amber-100">{fmt(report.alert_pending)}</p>
      <p class="mt-1 text-xs text-amber-800 dark:text-amber-200">Kelas non-Private di bawah target, D4+</p>
     </div>
     <div class="rounded-lg bg-red-50 p-4 dark:bg-red-500/10">
      <p class="text-xs text-red-800 dark:text-red-200">Tidak tepat waktu</p>
      <p class="mt-2 text-2xl font-semibold tabular-nums text-red-900 dark:text-red-100">{fmt(report.alert_late)}</p>
      <p class="mt-1 text-xs text-red-800 dark:text-red-200">Belum terkirim / terkirim di luar D4–D7, pada D8+</p>
     </div>
    </div>
    <p class="mt-4 text-xs text-neutral-500">Kepatuhan dihitung untuk kewajiban yang telah mencapai D8. Kelas Private dikecualikan; pengiriman WhatsApp tidak dilakukan otomatis.</p>
   </div>
  </section>

  <section class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900">
   <h2 class="text-base font-semibold">Distribusi Status Kanban</h2>
   <div class="mt-4 grid gap-3 md:grid-cols-5">
    {#each columns as col}
     <div class="rounded-lg bg-neutral-50 p-3 dark:bg-white/5">
      <p class="text-xs text-neutral-500">{col.label}</p>
      <p class="mt-2 text-2xl font-semibold tabular-nums">{fmt(report.stages[col.stage]??0)}</p>
     </div>
    {/each}
   </div>
   <p class="mt-3 text-xs text-neutral-500">Batal: {fmt(report.cancelled)} kasus. Pembatalan tidak dihitung sebagai keberhasilan SLA dan dilaporkan terpisah.</p>
  </section>
  <p class="text-xs leading-relaxed text-neutral-500">Definisi: SLA tepat waktu = tanggal mulai aktual sebelum akhir D14; denominator mencakup kasus selesai atau yang sudah melewati deadline, tidak termasuk Batal. Angka N/A berarti belum ada denominator. Kepatuhan Vacancy Alert dinilai berdasarkan kelas yang pernah dicatat mengirim alert atau masih di bawah target pada D8+. Jika kelas sudah memenuhi kuota tanpa catatan alert sebelum D8, riwayat kewajibannya tidak dapat direkonstruksi dari skema saat ini.</p>
 </main>
</AppLayout>
