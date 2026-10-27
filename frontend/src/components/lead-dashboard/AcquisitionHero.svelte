<script lang="ts">
	import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-svelte";
	import DailyTrend from "./DailyTrend.svelte";
	import { delta, fmt, formatDelta, formatHours, panel, pct, type Dashboard } from "./types";

	interface Props {
		d: Dashboard;
		onleads: (extra?: Record<string, string>) => void;
		onclosing: (day?: string) => void;
	}
	let { d, onleads, onclosing }: Props = $props();
	let r = $derived(d.report);

	let headline = $derived([
		{ key: "leads", label: "Lead masuk", value: fmt(r.total_leads), change: delta(r.total_leads, d.previous_leads), previous: d.previous_leads, hero: true, go: () => onleads() },
		{ key: "closing", label: "Closing", value: fmt(d.closing_period), change: delta(d.closing_period, d.previous_closing), previous: d.previous_closing, hero: false, go: () => onclosing() },
		{ key: "konversi", label: "Konversi cohort", value: pct(r.won, r.total_leads), change: null, previous: -1, hero: false, go: () => onleads({ status: "won" }), note: `${fmt(r.won)} dari ${fmt(r.total_leads)} lead jadi santri` },
	]);
	let secondary = $derived([
		{ label: "Median sampai closing", value: formatHours(r.median_hours_to_won), go: () => onleads({ status: "won" }) },
		{ label: "Masih berjalan", value: fmt(r.open), go: () => onleads({ status: "open" }) },
		{ label: "Lost", value: fmt(r.lost), go: () => onleads({ status: "lost" }) },
	]);
</script>

<section class={panel + " min-w-0 overflow-hidden"} aria-labelledby="akuisisi-title">
	<div class="flex items-baseline justify-between gap-3 px-5 pt-5 sm:px-6">
		<h2 id="akuisisi-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Akuisisi periode ini</h2>
		<span class="hidden text-[11px] text-neutral-400 sm:inline dark:text-neutral-500">dibanding periode sebelumnya dengan panjang sama</span>
	</div>

	<div class="grid grid-cols-2 gap-x-6 gap-y-5 px-5 pb-6 pt-4 sm:grid-cols-3 sm:gap-x-8 sm:px-6">
		{#each headline as k}
			<button type="button" onclick={k.go} class={"group -m-2 flex min-w-0 flex-col items-start justify-start rounded-lg p-2 text-left transition-colors duration-150 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.03] " + (k.hero ? "col-span-2 sm:col-span-1" : "")}>
				<span class="text-sm font-medium text-neutral-600 dark:text-neutral-300">{k.label}</span>
				<span class={"mt-1 block font-semibold tabular-nums tracking-tight text-neutral-950 dark:text-white " + (k.hero ? "text-[40px] leading-[44px]" : "text-[32px] leading-[40px]")}>{k.value}</span>
				{#if k.note}
					<span class="mt-1.5 block text-xs text-neutral-500 dark:text-neutral-400">{k.note}</span>
				{:else if k.change === null}
					<span class="mt-1.5 inline-flex items-center gap-1 text-xs text-neutral-500 dark:text-neutral-400"><Minus size={12} />{k.previous === 0 ? "Periode lalu 0" : "Tanpa pembanding"}</span>
				{:else}
					<span class="mt-1.5 inline-flex items-center gap-1.5 text-xs">
						<span class={"inline-flex items-center gap-0.5 rounded px-1 py-px font-semibold tabular-nums " + (k.change > 0 ? "bg-success/10 text-emerald-700 dark:text-emerald-400" : k.change < 0 ? "bg-error/10 text-red-700 dark:text-red-400" : "bg-neutral-100 text-neutral-600 dark:bg-white/5 dark:text-neutral-300")}>
							{#if k.change > 0}<ArrowUpRight size={12} />{:else if k.change < 0}<ArrowDownRight size={12} />{/if}{formatDelta(k.change)}
						</span>
						<span class="tabular-nums text-neutral-500 dark:text-neutral-400">dari {fmt(k.previous)}</span>
					</span>
				{/if}
			</button>
		{/each}
	</div>

	<div class="border-t border-neutral-100 px-5 py-5 sm:px-6 dark:border-white/[0.05]">
		<DailyTrend daily={d.daily} onleads={(day) => onleads({ from: day, to: day })} onclosing={(day) => onclosing(day)} />
	</div>

	<dl class="grid grid-cols-3 divide-x divide-neutral-100 border-t border-neutral-100 bg-neutral-50/60 dark:divide-white/[0.05] dark:border-white/[0.05] dark:bg-white/[0.015]">
		{#each secondary as s}
			<div class="min-w-0">
				<button type="button" onclick={s.go} class="block w-full px-4 py-3 text-left transition-colors duration-150 hover:bg-neutral-100/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand-500 sm:px-6 dark:hover:bg-white/[0.03]">
					<dt class="text-xs leading-tight text-neutral-500 dark:text-neutral-400">{s.label}</dt>
					<dd class="mt-0.5 text-base font-semibold tabular-nums text-neutral-900 dark:text-white">{s.value}</dd>
				</button>
			</div>
		{/each}
	</dl>
</section>
