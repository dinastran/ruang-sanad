<script lang="ts">
	import { STAGE_LABELS } from "@lib/leads";
	import { fmt, panel, type FunnelStage } from "./types";

	interface Props { funnel: FunnelStage[]; onstage: (stage: string) => void }
	let { funnel, onstage }: Props = $props();

	let top = $derived(Math.max(1, funnel[0]?.reached ?? 0));
	let drops = $derived(funnel.map((f, i) => (i === 0 ? 0 : Math.max(0, funnel[i - 1].reached - f.reached))));
	let worst = $derived(drops.reduce((best, v, i) => (v > drops[best] ? i : best), 0));
	let total = $derived(funnel[0]?.reached ?? 0);
</script>

<section class={panel + " flex flex-col p-5 sm:p-6"} aria-labelledby="jalur-title">
	<div class="flex flex-wrap items-baseline justify-between gap-2">
		<div>
			<h2 id="jalur-title" class="text-base font-semibold text-neutral-950 dark:text-white">Jalur pendaftaran</h2>
			<p class="mt-0.5 max-w-prose text-xs text-neutral-500 dark:text-neutral-400">Estimasi tahap yang dicapai lead periode ini, dari posisi pipeline saat ini. Klik tahap untuk melihat lead-nya.</p>
		</div>
		{#if drops[worst] > 0}
			<p class="text-xs text-neutral-600 dark:text-neutral-300">Paling banyak gugur menuju <span class="font-semibold text-amber-700 dark:text-amber-400">{STAGE_LABELS[funnel[worst].stage] || funnel[worst].stage}</span></p>
		{/if}
	</div>

	{#if total === 0}
		<p class="flex flex-1 items-center justify-center py-12 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada lead pada periode ini.</p>
	{:else}
		<ol class="mt-5">
			{#each funnel as f, i}
				{@const won = f.stage === "won"}
				{@const last = i === funnel.length - 1}
				{#if i > 0}
					<li class="grid grid-cols-[1.5rem_1fr] gap-x-3" aria-hidden={drops[i] === 0}>
						<span class="flex justify-center"><span class="h-full w-px bg-neutral-200 dark:bg-white/10"></span></span>
						<span class="py-0.5 text-[11px] tabular-nums">
							{#if drops[i] > 0}
								<span class={i === worst ? "font-semibold text-amber-700 dark:text-amber-400" : "text-neutral-400 dark:text-neutral-500"}>
									−{fmt(drops[i])} gugur · lanjut {Math.round(f.conversion * 100)}%
								</span>
							{:else}
								<span class="text-neutral-300 dark:text-neutral-600">semua lanjut</span>
							{/if}
						</span>
					</li>
				{/if}
				<li>
					<button type="button" onclick={() => onstage(f.stage)}
						aria-label={`${STAGE_LABELS[f.stage] || f.stage}: ${fmt(f.reached)} lead`}
						class="group grid w-full grid-cols-[1.5rem_1fr_auto] items-center gap-x-3 gap-y-1 rounded-md py-1 text-left sm:grid-cols-[1.5rem_minmax(7rem,11rem)_1fr_3.5rem] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">
						<span class="flex justify-center">
							<span class={"flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-semibold tabular-nums ring-1 transition-colors duration-150 " +
								(won ? "bg-success text-white ring-success" : i === worst && drops[i] > 0 ? "bg-white text-amber-700 ring-warning dark:bg-neutral-925 dark:text-amber-400" : "bg-white text-neutral-600 ring-neutral-300 group-hover:ring-brand-500 dark:bg-neutral-925 dark:text-neutral-300 dark:ring-white/15")}>{won ? "✓" : i + 1}</span>
						</span>
						<span class={"truncate text-sm group-hover:text-brand-700 dark:group-hover:text-brand-400 " + (won || last ? "font-semibold text-neutral-950 dark:text-white" : "text-neutral-700 dark:text-neutral-300")}>{STAGE_LABELS[f.stage] || f.stage}</span>
						<span class="order-last col-start-2 col-end-4 h-2 overflow-hidden rounded-full bg-neutral-100 sm:order-none sm:col-auto dark:bg-white/[0.06]">
							<span class={"block h-full rounded-full transition-[width] duration-500 " + (won ? "bg-success" : "bg-brand-500 group-hover:bg-brand-600")} style={`width:${Math.max(f.reached / top * 100, f.reached ? 1.5 : 0)}%`}></span>
						</span>
						<span class="text-right text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(f.reached)}</span>
					</button>
				</li>
			{/each}
		</ol>
	{/if}
</section>
