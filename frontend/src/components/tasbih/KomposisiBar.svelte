<script lang="ts">
	// Komposisi dzikir: bar horizontal berurutan (lebih mudah dibandingkan daripada donut).
	import { angka } from "./format";
	interface Props {
		data: { dzikir_id: number; latin: string; total: number }[];
	}
	let { data }: Props = $props();
	const total = $derived(data.reduce((a, d) => a + d.total, 0));
	const max = $derived(Math.max(...data.map((d) => d.total), 1));
</script>

<ul class="space-y-3">
	{#each data as d (d.latin)}
		<li>
			<div class="flex items-baseline justify-between gap-3 text-sm">
				<span class="truncate font-medium text-neutral-800 dark:text-neutral-200">{d.latin}</span>
				<span class="shrink-0 tabular-nums text-neutral-500 dark:text-neutral-400">{angka(d.total)} <span class="text-xs">· {total ? Math.round((d.total / total) * 100) : 0}%</span></span>
			</div>
			<div class="mt-1.5 h-2 rounded-full bg-neutral-100 dark:bg-neutral-800">
				<div class="h-full rounded-full {d.dzikir_id === 0 ? 'bg-neutral-400 dark:bg-neutral-600' : 'bg-brand-500 dark:bg-brand-400'}" style="width:{(d.total / max) * 100}%"></div>
			</div>
		</li>
	{/each}
</ul>
