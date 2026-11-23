<script lang="ts" module>
  export interface OnboardingListCase {
    id: number; santri_id: number; nama: string; id_mahasantri: string;
    stage: string; pic_user_id: number; pic_name: string;
    registered_at: string; deadline_at: string; actual_started_at?: string;
    kelas_id: number; kelas_nama: string;
    angkatan: string; level: string; tipe: string; frekuensi: string; jenis_kelamin: string; domisili: string;
  }
  export type OnboardingSortKey = "nama" | "angkatan" | "registered" | "stage" | "deadline";
</script>

<script lang="ts">
  import { inertia } from "@inertiajs/svelte";
  import { ArrowLeft, ArrowRight, Ban, History, ArrowUp, ArrowDown, ChevronsUpDown } from "lucide-svelte";
  import GenderBadge from "./GenderBadge.svelte";

  type SortKey = OnboardingSortKey;
  interface Props {
    cases: OnboardingListCase[];
    pics: { id: number; name: string; email: string }[];
    stages: { key: string; label: string }[];
    busy: boolean;
    sort: string;
    dir: string;
    onSort: (key: SortKey) => void;
    onAssignPIC: (card: OnboardingListCase, e: Event) => void;
    onMove: (card: OnboardingListCase, target: string) => void;
    onHistory: (card: OnboardingListCase) => void;
  }
  let { cases, pics, stages, busy, sort, dir, onSort, onAssignPIC, onMove, onHistory }: Props = $props();

  const index = (stage: string) => stages.findIndex(s => s.key === stage);
  const stageLabel = (stage: string) => stages.find(s => s.key === stage)?.label ?? (stage === "batal" ? "Batal" : stage);
  const stageTone: Record<string, string> = {
    onboarding_baru: "bg-neutral-100 text-neutral-700 dark:bg-white/10 dark:text-neutral-200",
    matching_kelas: "bg-brand-50 text-brand-800 dark:bg-brand-500/15 dark:text-brand-200",
    waiting_kuota: "bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:text-amber-200",
    siap_mulai: "bg-sky-50 text-sky-800 dark:bg-sky-500/15 dark:text-sky-200",
    masuk_kelas: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-200",
    batal: "bg-red-50 text-red-700 dark:bg-red-500/15 dark:text-red-300",
  };

  function dateWIB(value?: string): string {
    if (!value) return "-";
    return new Date(value).toLocaleDateString("id-ID", { timeZone: "Asia/Jakarta", day: "numeric", month: "short", year: "numeric" });
  }
  function sla(card: OnboardingListCase): { text: string; cls: string } {
    if (card.stage === "masuk_kelas") return { text: "Selesai", cls: "text-emerald-700 dark:text-emerald-300" };
    if (card.stage === "batal") return { text: "-", cls: "text-neutral-400" };
    const days = Math.ceil((new Date(card.deadline_at).getTime() - Date.now()) / 86400000);
    if (days <= 0) return { text: "Lewat SLA", cls: "font-semibold text-red-700 dark:text-red-300" };
    if (days <= 4) return { text: `${days} hari lagi`, cls: "font-medium text-amber-700 dark:text-amber-300" };
    return { text: `${days} hari lagi`, cls: "text-neutral-600 dark:text-neutral-300" };
  }
  const columns: { key: SortKey | null; label: string; cls?: string }[] = [
    { key: "nama", label: "Mahasantri", cls: "sticky left-0 z-10 min-w-[200px] bg-neutral-50 dark:bg-neutral-950" },
    { key: "angkatan", label: "Angkatan" },
    { key: "registered", label: "Tgl Daftar" },
    { key: "stage", label: "Tahapan" },
    { key: "deadline", label: "SLA" },
    { key: null, label: "Kelas" },
    { key: null, label: "Level" },
    { key: null, label: "Tipe" },
    { key: null, label: "Frekuensi" },
    { key: null, label: "JK" },
    { key: null, label: "Domisili" },
    { key: null, label: "PIC Admin Kelas" },
    { key: null, label: "Mulai Aktual" },
    { key: null, label: "Aksi", cls: "text-right" },
  ];
  const ariaSort = (key: SortKey | null) => key && key === sort ? (dir === "desc" ? "descending" : "ascending") : undefined;
  const btnClass = "inline-flex items-center gap-1 rounded-md border border-neutral-200 px-2 py-1 text-xs font-medium hover:bg-neutral-100 disabled:opacity-40 dark:border-white/10 dark:hover:bg-white/10";
</script>

<div class="overflow-x-auto rounded-xl border border-neutral-200 bg-white dark:border-white/10 dark:bg-neutral-900">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-b border-neutral-200 bg-neutral-50 dark:border-white/10 dark:bg-neutral-950">
        {#each columns as col (col.label)}
          <th scope="col" aria-sort={ariaSort(col.key)} class={"whitespace-nowrap px-3 py-2.5 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 " + (col.cls ?? "")}>
            {#if col.key}
              {@const key = col.key}
              <button type="button" class="inline-flex items-center gap-1 hover:text-neutral-900 dark:hover:text-white" onclick={() => onSort(key)}>
                {col.label}
                {#if sort === key}{#if dir === "desc"}<ArrowDown size={12}/>{:else}<ArrowUp size={12}/>{/if}{:else}<ChevronsUpDown size={12} class="opacity-40"/>{/if}
              </button>
            {:else}{col.label}{/if}
          </th>
        {/each}
      </tr>
    </thead>
    <tbody class="divide-y divide-neutral-100 dark:divide-white/5">
      {#each cases as card (card.id)}
        {@const s = sla(card)}
        {@const terminal = card.stage === "masuk_kelas" || card.stage === "batal"}
        <tr class={"align-top hover:bg-neutral-50 dark:hover:bg-white/[0.03] " + (card.stage === "batal" ? "opacity-70" : "")}>
          <td class="sticky left-0 z-10 bg-white px-3 py-2.5 dark:bg-neutral-900">
            <a href={`/app/santri/${card.santri_id}`} use:inertia class="block max-w-[220px] truncate font-medium text-neutral-900 hover:text-brand-700 dark:text-white dark:hover:text-brand-300">{card.nama}</a>
            <p class="text-xs text-neutral-500">{card.id_mahasantri || "Belum ada ID"}</p>
            {#if !terminal && (!card.pic_user_id || (index(card.stage) >= 1 && !card.kelas_id))}
              <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">{!card.pic_user_id ? "PIC belum ditetapkan" : "Penempatan kelas belum ada"}</p>
            {/if}
          </td>
          <td class="whitespace-nowrap px-3 py-2.5">{card.angkatan || "-"}</td>
          <td class="whitespace-nowrap px-3 py-2.5 tabular-nums">{dateWIB(card.registered_at)}</td>
          <td class="whitespace-nowrap px-3 py-2.5"><span class={"rounded-md px-2 py-0.5 text-xs font-medium " + (stageTone[card.stage] ?? stageTone.onboarding_baru)}>{stageLabel(card.stage)}</span></td>
          <td class="whitespace-nowrap px-3 py-2.5 text-xs">
            <span class={s.cls}>{s.text}</span>
            {#if !terminal}<p class="text-neutral-400">Batas {dateWIB(card.deadline_at)}</p>{/if}
          </td>
          <td class="max-w-[200px] truncate px-3 py-2.5" title={card.kelas_nama}>{card.kelas_nama || "-"}</td>
          <td class="whitespace-nowrap px-3 py-2.5">{card.level || "-"}</td>
          <td class="whitespace-nowrap px-3 py-2.5">{card.tipe || "-"}</td>
          <td class="whitespace-nowrap px-3 py-2.5">{card.frekuensi || "-"}</td>
          <td class="whitespace-nowrap px-3 py-2.5">{#if card.jenis_kelamin}<GenderBadge gender={card.jenis_kelamin}/>{:else}-{/if}</td>
          <td class="max-w-[160px] truncate px-3 py-2.5" title={card.domisili}>{card.domisili || "-"}</td>
          <td class="px-3 py-2.5">
            <select aria-label={`PIC Admin Kelas untuk ${card.nama}`} value={card.pic_user_id} disabled={busy || terminal}
              onchange={(e) => onAssignPIC(card, e)}
              class="w-44 rounded-md border border-neutral-200 bg-white px-2 py-1 text-xs text-neutral-900 disabled:opacity-50 dark:border-white/15 dark:bg-neutral-800 dark:text-white">
              <option value={0}>Belum ditetapkan</option>
              {#each pics as pic (pic.id)}<option value={pic.id}>{pic.name}</option>{/each}
            </select>
          </td>
          <td class="whitespace-nowrap px-3 py-2.5 tabular-nums">{dateWIB(card.actual_started_at)}</td>
          <td class="px-3 py-2.5">
            <div class="flex justify-end gap-1">
              {#if !terminal && index(card.stage) > 0}
                <button class={btnClass} type="button" disabled={busy} title="Mundur satu tahap" aria-label={`Mundurkan ${card.nama}`} onclick={() => onMove(card, stages[index(card.stage) - 1].key)}><ArrowLeft size={12}/></button>
              {/if}
              {#if !terminal && index(card.stage) >= 0 && index(card.stage) < stages.length - 1}
                <button class={btnClass} type="button" disabled={busy} title={`Lanjut ke ${stages[index(card.stage) + 1].label}`} onclick={() => onMove(card, stages[index(card.stage) + 1].key)}>Lanjut <ArrowRight size={12}/></button>
              {/if}
              {#if !terminal}
                <button class={btnClass} type="button" disabled={busy} title="Batalkan onboarding" aria-label={`Batalkan ${card.nama}`} onclick={() => onMove(card, "batal")}><Ban size={12}/></button>
              {/if}
              <button class={btnClass} type="button" title="Riwayat" aria-label={`Riwayat ${card.nama}`} onclick={() => onHistory(card)}><History size={12}/></button>
            </div>
          </td>
        </tr>
      {/each}
      {#if cases.length === 0}
        <tr><td colspan={columns.length} class="px-3 py-10 text-center text-sm text-neutral-500">Tidak ada kasus di halaman ini</td></tr>
      {/if}
    </tbody>
  </table>
</div>
