<script lang="ts">
	import { num, ratio, rp, rpShort, type SourceStat } from "./data";

	interface Props { stats: SourceStat[]; hasMapping: boolean; selected: number; onselect: (id: number) => void }
	let { stats, hasMapping, selected, onselect }: Props = $props();

	let maxSpend = $derived(Math.max(1, ...stats.map((s) => s.spend)));
	// Cheapest santri among sources that produced at least one.
	let best = $derived.by(() => {
		const withWon = stats.filter((s) => s.won > 0);
		if (withWon.length < 2) return 0;
		return withWon.reduce((a, b) => (ratio(a.spend, a.won) <= ratio(b.spend, b.won) ? a : b)).id;
	});
</script>

<section class="min-w-0 rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60" aria-labelledby="sumber-title">
	<div class="px-5 pt-5 sm:px-6">
		<h2 id="sumber-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Per sumber lead (LP)</h2>
		<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Biaya kampanye yang terhubung dibanding lead dan santri dari sumber itu. Klik untuk memfilter.</p>
	</div>
	{#if !hasMapping}
		<p class="px-5 py-10 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Belum ada kampanye yang dihubungkan ke sumber lead. Pilih sumbernya di tabel Kampanye.</p>
	{:else}
		<ul class="mt-3 divide-y divide-neutral-100 border-t border-neutral-100 dark:divide-white/[0.05] dark:border-white/[0.05]">
			{#each stats as s (s.id)}
				<li>
					<button type="button" onclick={() => onselect(selected === s.id ? 0 : s.id)} aria-pressed={selected === s.id}
						class={"grid w-full grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-1.5 px-5 py-3 text-left transition-colors duration-150 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand-500 sm:px-6 dark:hover:bg-white/[0.03] " + (selected === s.id ? "bg-brand-500/[0.06]" : "")}>
						<span class="flex min-w-0 items-center gap-2">
							<span class="truncate text-sm font-medium text-neutral-900 dark:text-white">{s.nama}</span>
							{#if s.id === best}<span class="shrink-0 rounded bg-emerald-500/10 px-1.5 py-px text-[10px] font-semibold text-emerald-700 dark:text-emerald-400">termurah/santri</span>{/if}
						</span>
						<span class="text-right text-sm font-semibold tabular-nums text-neutral-950 dark:text-white" title={rp(s.spend)}>{rpShort(s.spend)}</span>
						<span class="h-1.5 self-center overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]"><span class="block h-full rounded-full bg-brand-600 dark:bg-brand-500" style:width={`${(s.spend / maxSpend) * 100}%`}></span></span>
						<span class="text-right text-xs tabular-nums text-neutral-500 dark:text-neutral-400">{s.campaigns} kampanye</span>
						<span class="col-span-2 flex flex-wrap gap-x-4 gap-y-0.5 text-xs tabular-nums text-neutral-600 dark:text-neutral-300">
							<span>{num(s.leads)} lead · <span class="text-neutral-500">{rp(ratio(s.spend, s.leads))}/lead</span></span>
							<span><strong class="font-semibold text-neutral-900 dark:text-white">{num(s.won)}</strong> santri · <span class={s.won > 0 ? "text-neutral-500" : "text-amber-700 dark:text-amber-400"}>{s.won > 0 ? `${rp(ratio(s.spend, s.won))}/santri` : "belum ada santri"}</span></span>
						</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</section>
