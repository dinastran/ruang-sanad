<script lang="ts">
  import { inertia } from "@inertiajs/svelte";
  import { Bell, Kanban, CalendarClock, ChartColumnIncreasing } from "lucide-svelte";
  import OnboardingNotices from "./OnboardingNotices.svelte";

  type WorkspaceTab = "kanban" | "classes" | "monitoring";
  interface Summary { total:number; at_risk:number; overdue:number; alert_pending:number }
  interface ClassSummary { target_kuota:number; vacancy_required:boolean; vacancy_sent_at:string; day:number }
  interface Notice {id:number;type:string;title:string;message:string;action_url:string;read:boolean;created_at:string}
  interface Props {
    role:string;
    active:WorkspaceTab;
    summary?:Summary;
    classes?:ClassSummary[];
    notices?:Notice[];
  }
  let {role,active,summary,classes=[],notices=[]}:Props=$props();
  let noticesOpen=$state(false);
  let acknowledged=$state<number[]>([]);
  const canOperate=$derived(role==="super_admin"||role==="admin_kelas");
  const tabs=[
    {key:"kanban",label:"Papan Kanban",href:"/app/onboarding",icon:Kanban},
    {key:"classes",label:"Kelas & Kuota",href:"/app/onboarding/classes",icon:CalendarClock},
    {key:"monitoring",label:"Kinerja & SLA",href:"/app/onboarding/monitoring",icon:ChartColumnIncreasing}
  ] as const;
  const number=(n:number)=>n.toLocaleString("id-ID");
  const pendingNotices=$derived(notices.filter(n=>!n.read&&!acknowledged.includes(n.id)).length);
  const missingQuota=$derived(classes.filter(c=>!c.target_kuota).length);
  const pendingVacancies=$derived(classes.filter(c=>c.vacancy_required&&c.day>=4&&!c.vacancy_sent_at).length);
</script>

<section aria-label="Workspace onboarding" class="space-y-4">
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <p class="text-xs font-semibold uppercase tracking-wide text-brand-700 dark:text-brand-400">Akademik / Onboarding</p>
      <h1 class="mt-1 text-2xl font-semibold text-neutral-950 dark:text-white">Manajemen Onboarding</h1>
      <p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Proses santri baru, kesiapan kelas, dan SLA 14 hari dalam satu workspace.</p>
    </div>
    <button
      type="button"
      aria-label={"Notifikasi onboarding, "+pendingNotices+" belum dibaca"}
      aria-controls="onboarding-notification-panel"
      aria-expanded={noticesOpen}
      onclick={()=>noticesOpen=!noticesOpen}
      class="inline-flex items-center gap-2 rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm font-medium hover:bg-neutral-50 dark:border-white/15 dark:bg-neutral-900 dark:hover:bg-white/10"
    >
      <Bell size={17}/>
      Notifikasi
      {#if pendingNotices>0}<span class="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-semibold tabular-nums text-amber-900 dark:bg-amber-500/20 dark:text-amber-200">{pendingNotices}</span>{/if}
    </button>
  </div>

  {#if noticesOpen}
    <div id="onboarding-notification-panel">
      <OnboardingNotices items={notices} onRead={(id)=>{acknowledged=[...acknowledged,id];}}/>
    </div>
  {/if}

  <nav aria-label="Bagian onboarding" class="flex gap-1 overflow-x-auto border-b border-neutral-200 dark:border-white/10">
    {#each tabs.filter(t=>canOperate||t.key==="classes") as tab (tab.key)}
      {@const Icon = tab.icon}
      <a
        href={tab.href}
        use:inertia
        aria-current={active===tab.key?"page":undefined}
        class={"inline-flex shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-3 py-3 text-sm font-medium transition-colors "+
          (active===tab.key?"border-brand-500 text-brand-700 dark:text-brand-300":"border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-900 dark:hover:text-white")}
      >
        <Icon size={16}/>
        {tab.label}
      </a>
    {/each}
  </nav>

  <div aria-label="Ringkasan onboarding" class="grid grid-cols-2 gap-3 md:grid-cols-4">
    {#if canOperate && summary}
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">Kasus tercatat</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(summary.total)}</p>
      </div>
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">SLA berisiko</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(summary.at_risk)}</p>
      </div>
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">Overdue</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(summary.overdue)}</p>
      </div>
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">Vacancy Alert tertunda</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(summary.alert_pending)}</p>
      </div>
    {:else if !canOperate}
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">Kelas onboarding</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(classes.length)}</p>
      </div>
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">Target belum diatur</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(missingQuota)}</p>
      </div>
      <div class="rounded-xl border border-neutral-200 bg-white p-3 dark:border-white/10 dark:bg-neutral-900">
        <p class="text-xs text-neutral-500">Vacancy Alert tertunda</p><p class="mt-1 text-xl font-semibold tabular-nums">{number(pendingVacancies)}</p>
      </div>
    {/if}
  </div>
</section>
