<script lang="ts">
	import { STAGE_LABELS, STATUS_LABELS } from "@lib/leads";
	import { panel, fmt } from "@components/lead-dashboard/types";
	import { CLOSED_COLUMNS, type FilterChange, type Filters } from "./types";

	interface Props { stages: string[]; counts: Record<string, number>; total: number; filters: Filters; onfilter: (c: FilterChange) => void }
	let { stages, counts, total, filters, onfilter }: Props = $props();

	let closed = $derived(CLOSED_COLUMNS.filter((c) => (counts[c] ?? 0) > 0));
	let openTotal = $derived(stages.reduce((s, k) => s + (counts[k] ?? 0), 0));
	let active = $derived(filters.stage || (CLOSED_COLUMNS.includes(filters.status) ? filters.status : ""));
	// Stage cells read as one path; the darkest step marks where most leads wait.
	let busiest = $derived(stages.reduce((b, k) => ((counts[k] ?? 0) > (counts[b] ?? 0) ? k : b), stages[0] ?? ""));

	function pick(key: string) {
		if (active === key) {
			onfilter(CLOSED_COLUMNS.includes(key) ? { status: "" } : { stage: "" });
			return;
		}
		if (CLOSED_COLUMNS.includes(key)) onfilter({ status: key, stage: "" });
		else onfilter({ stage: key, status: CLOSED_COLUMNS.includes(filters.status) ? "" : filters.status });
	}
	const fill = (key: string) =>
		key === "won" ? "bg-success" : key === "lost" ? "bg-warning" : key === "do_not_contact" ? "bg-error" : key === "bukti_masuk" ? "bg-warning" : key === "terverifikasi" ? "bg-success/70" : "bg-brand-500";
</script>

<section class={panel + " overflow-hidden"} aria-labelledby="pipeline-title">
	<div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-4 pt-4 sm:px-5">
		<h2 id="pipeline-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Pipeline sesuai filter</h2>
		<p class="text-xs tabular-nums text-neutral-500 dark:text-neutral-400">
			{#if active}Klik lagi tahap aktif untuk melepas filter{:else}{fmt(openTotal)} berjalan{#if closed.length}{" · "}{fmt(total - openTotal)} selesai{/if}{/if}
		</p>
	</div>

	{#if total > 0}
		<div class="mx-4 mt-3 flex h-1.5 gap-px overflow-hidden rounded-full bg-neutral-100 sm:mx-5 dark:bg-white/[0.06]" aria-hidden="true">
			{#each [...stages, ...closed] as key}
				{#if (counts[key] ?? 0) > 0}
					<span class={"h-full transition-opacity duration-150 " + fill(key) + (active && active !== key ? " opacity-25" : "")} style={`width:${(counts[key] ?? 0) / total * 100}%`}></span>
				{/if}
			{/each}
		</div>
	{/if}

	<div class="mt-2 overflow-x-auto px-2 pb-2 sm:px-3">
		<div class="flex gap-0.5" role="group" aria-label="Filter tahap">
			{#each stages as key}
				{@const n = counts[key] ?? 0}
				{@render cell(key, STAGE_LABELS[key] ?? key, n)}
			{/each}
			{#if closed.length}
				<span class="mx-1.5 my-2 w-px shrink-0 bg-neutral-200 dark:bg-white/10" aria-hidden="true"></span>
				{#each closed as key}
					{@render cell(key, STATUS_LABELS[key] ?? key, counts[key] ?? 0)}
				{/each}
			{/if}
		</div>
	</div>
</section>

{#snippet cell(key: string, label: string, n: number)}
	<button type="button" onclick={() => pick(key)} aria-pressed={active === key}
		class={"group flex min-w-[6.5rem] flex-1 flex-col items-start rounded-lg px-2.5 py-2 text-left transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 " +
			(active === key ? "bg-brand-500/10 ring-1 ring-inset ring-brand-500/40" : "hover:bg-neutral-50 dark:hover:bg-white/[0.03]")}>
		<span class="flex min-h-[2lh] w-full items-start gap-1.5 text-[11px] leading-tight text-neutral-500 dark:text-neutral-400">
			<span class={"mt-[0.3em] h-1.5 w-1.5 shrink-0 rounded-full " + fill(key) + (n === 0 ? " opacity-30" : "")} aria-hidden="true"></span>
			<span class="line-clamp-2">{label}</span>
		</span>
		<span class={"mt-0.5 text-lg font-semibold tabular-nums " + (n === 0 ? "text-neutral-300 dark:text-neutral-600" : key === busiest && !active ? "text-neutral-950 dark:text-white" : "text-neutral-800 dark:text-neutral-200")}>{fmt(n)}</span>
	</button>
{/snippet}
