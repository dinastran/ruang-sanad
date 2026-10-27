<script lang="ts">
	import { fmt, longDate, shortDate, type Day } from "./types";

	interface Props { daily: Day[]; onleads: (day: string) => void; onclosing: (day: string) => void }
	let { daily, onleads, onclosing }: Props = $props();

	let hovered = $state<number | null>(null);
	let max = $derived(Math.max(1, ...daily.flatMap((d) => [d.leads, d.closing])));
	// Round the axis up to a friendly number so gridlines read 0 / half / top.
	let top = $derived.by(() => {
		const step = max <= 10 ? 2 : max <= 50 ? 10 : Math.pow(10, Math.floor(Math.log10(max)));
		return Math.ceil(max / step) * step;
	});
	let labelEvery = $derived(Math.max(1, Math.ceil(daily.length / 8)));
	let dense = $derived(daily.length > 45);
	let readout = $derived(hovered === null ? null : daily[hovered]);
	let totalLeads = $derived(daily.reduce((s, d) => s + d.leads, 0));
	let totalClosing = $derived(daily.reduce((s, d) => s + d.closing, 0));
	const h = (v: number) => (v ? Math.max(v / top * 100, 2) : 0) + "%";
</script>

<div>
	<div class="mb-3 flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-xs">
		<p class="tabular-nums text-neutral-600 dark:text-neutral-300" aria-live="polite">
			{#if readout}
				<span class="font-medium text-neutral-900 dark:text-white">{longDate(readout.day)}</span>
				· {fmt(readout.leads)} lead · {fmt(readout.closing)} closing
			{:else}
				<span class="text-neutral-500 dark:text-neutral-400">{totalLeads + totalClosing === 0 ? "Belum ada lead atau closing pada periode ini" : `Arahkan ke batang untuk detail harian · total ${fmt(totalLeads)} lead, ${fmt(totalClosing)} closing`}</span>
			{/if}
		</p>
		<div class="flex items-center gap-3 text-neutral-500 dark:text-neutral-400">
			<span class="inline-flex items-center gap-1.5"><i class="h-2 w-2 rounded-[2px] bg-brand-500"></i>Lead masuk</span>
			<span class="inline-flex items-center gap-1.5"><i class="h-2 w-2 rounded-[2px] bg-success"></i>Closing</span>
		</div>
	</div>

	<div class="flex gap-2">
		<div class="flex h-40 w-6 shrink-0 flex-col justify-between text-right text-[10px] tabular-nums text-neutral-400 dark:text-neutral-500" aria-hidden="true">
			<span class="-translate-y-1.5">{fmt(top)}</span><span>{fmt(top / 2)}</span><span class="translate-y-1.5">0</span>
		</div>
		<div class="min-w-0 flex-1 overflow-x-auto">
			<div class="relative" style={dense ? `min-width:${daily.length * 7}px` : ""}>
				<div class="pointer-events-none absolute inset-x-0 top-0 h-40" aria-hidden="true">
					<div class="absolute inset-x-0 top-0 border-t border-dashed border-neutral-200 dark:border-white/[0.06]"></div>
					<div class="absolute inset-x-0 top-1/2 border-t border-dashed border-neutral-200 dark:border-white/[0.06]"></div>
					<div class="absolute inset-x-0 bottom-0 border-t border-neutral-300 dark:border-white/15"></div>
				</div>
				<div class="relative flex h-40 items-end" role="list" aria-label="Lead masuk dan closing per hari" onmouseleave={() => (hovered = null)}>
					{#each daily as d, i}
						<div role="listitem" class={"group flex h-full min-w-0 flex-1 items-end justify-center " + (dense ? "gap-px px-px" : "gap-0.5 px-0.5 sm:px-1")}
							onmouseenter={() => (hovered = i)}>
							<button type="button" onclick={() => onleads(d.day)} onfocus={() => (hovered = i)} onblur={() => (hovered = null)}
								aria-label={`${longDate(d.day)}: ${fmt(d.leads)} lead masuk`}
								class={"w-full max-w-4 rounded-t-[3px] bg-brand-500 transition-[opacity,background-color] duration-150 hover:bg-brand-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-1 dark:bg-brand-500/90 " + (hovered !== null && hovered !== i ? "opacity-45" : "")}
								style={`height:${h(d.leads)}`}></button>
							<button type="button" onclick={() => onclosing(d.day)} onfocus={() => (hovered = i)} onblur={() => (hovered = null)}
								aria-label={`${longDate(d.day)}: ${fmt(d.closing)} closing`}
								class={"w-full max-w-4 rounded-t-[3px] bg-success transition-[opacity,background-color] duration-150 hover:bg-emerald-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-success focus-visible:ring-offset-1 " + (hovered !== null && hovered !== i ? "opacity-45" : "")}
								style={`height:${h(d.closing)}`}></button>
						</div>
					{/each}
				</div>
				<div class="mt-2 flex text-[10px] tabular-nums text-neutral-500 dark:text-neutral-400" aria-hidden="true">
					{#each daily as d, i}
						<span class="min-w-0 flex-1 overflow-visible whitespace-nowrap text-center">{i % labelEvery === 0 ? shortDate(d.day) : ""}</span>
					{/each}
				</div>
			</div>
		</div>
	</div>
</div>
