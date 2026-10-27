<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { STAGE_LABELS, STATUS_LABELS, leadName, relativeTime } from "@lib/leads";
	import { fmt } from "@components/lead-dashboard/types";
	import LeadFlags from "./LeadFlags.svelte";
	import { CLOSED_COLUMNS, type LeadItem } from "./types";

	interface Props { items: LeadItem[]; stages: string[]; counts: Record<string, number>; onmove: (lead: LeadItem, stage: string) => void }
	let { items, stages, counts, onmove }: Props = $props();

	let dragId = $state<number | null>(null);
	let dropStage = $state<string | null>(null);
	// Open leads sit in their stage column; closed leads get their own columns
	// so the board adds up to the list total.
	let columns = $derived([...stages, ...CLOSED_COLUMNS.filter((c) => (counts[c] ?? 0) > 0)]);
	const columnOf = (l: LeadItem) => (l.status === "open" ? l.stage : l.status);
	const columnLabel = (c: string) => (CLOSED_COLUMNS.includes(c) ? STATUS_LABELS[c] : STAGE_LABELS[c]) ?? c;
	// Terverifikasi is shown but only the verify action can move a lead there.
	const canDropOn = (c: string) => c !== "terverifikasi" && !CLOSED_COLUMNS.includes(c);

	function drop(stage: string) {
		const lead = items.find((l) => l.id === dragId);
		dragId = null;
		dropStage = null;
		if (lead && lead.stage !== stage) onmove(lead, stage);
	}
</script>

<div class="flex gap-3 overflow-x-auto pb-2">
	{#each columns as stage}
		{@const col = items.filter((l) => columnOf(l) === stage)}
		<section aria-label={columnLabel(stage)}
			class={"flex w-64 shrink-0 flex-col rounded-xl border bg-neutral-100/60 transition-colors duration-150 dark:bg-white/[0.02] " + (dropStage === stage ? "border-brand-500 bg-brand-500/5" : "border-neutral-200/80 dark:border-white/[0.06]")}
			ondragover={(e) => { if (!canDropOn(stage)) return; e.preventDefault(); dropStage = stage; }}
			ondragleave={() => { if (dropStage === stage) dropStage = null; }}
			ondrop={(e) => { e.preventDefault(); if (canDropOn(stage)) drop(stage); }}>
			<header class="flex items-center justify-between px-3 py-2.5">
				<h2 class="truncate text-xs font-semibold text-neutral-700 dark:text-neutral-300">{columnLabel(stage)}</h2>
				<span class="rounded bg-white px-1.5 text-xs font-medium tabular-nums text-neutral-500 shadow-[0_0_0_1px_rgba(15,23,42,.06)] dark:bg-white/5 dark:text-neutral-400 dark:shadow-none" title={col.length < (counts[stage] ?? 0) ? `${col.length} dimuat dari ${counts[stage]}` : undefined}>{fmt(counts[stage] ?? 0)}</span>
			</header>
			<div class="min-h-24 flex-1 space-y-2 px-2 pb-2">
				{#each col as lead (lead.id)}
					<a href="/app/leads/{lead.id}" use:inertia draggable={lead.status === "open"}
						ondragstart={() => (dragId = lead.id)} ondragend={() => { dragId = null; dropStage = null; }}
						class={"block rounded-lg bg-white p-3 shadow-[0_0_0_1px_rgba(15,23,42,.06),0_1px_2px_rgba(15,23,42,.05)] transition-shadow duration-150 hover:shadow-[0_0_0_1px_rgba(6,182,212,.45),0_2px_4px_rgba(15,23,42,.06)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:bg-neutral-925 dark:shadow-[0_0_0_1px_rgba(255,255,255,.07)] " + (dragId === lead.id ? "opacity-50" : "")}>
						<span class="block truncate text-sm font-medium text-neutral-900 dark:text-white">{leadName(lead)}</span>
						<span class="mt-0.5 block text-xs text-neutral-500 dark:text-neutral-400">{relativeTime(lead.last_activity_at)}{lead.paket_kode ? ` · ${lead.paket_kode}` : ""}</span>
						<span class="mt-2 block empty:hidden"><LeadFlags {lead} /></span>
					</a>
				{/each}
				{#if col.length < (counts[stage] ?? 0)}
					<p class="px-1 text-[11px] text-neutral-500 dark:text-neutral-400">+{fmt((counts[stage] ?? 0) - col.length)} lagi, klik Muat lagi</p>
				{/if}
			</div>
		</section>
	{/each}
</div>
<p class="text-xs text-neutral-500 dark:text-neutral-400">Seret kartu ke kolom lain untuk mengubah tahap. Tahap Terverifikasi hanya lewat tombol Verifikasi pembayaran di halaman lead.</p>
