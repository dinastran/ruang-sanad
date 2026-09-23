<script lang="ts">
	interface Item { label: string; value: string; total: number }
	interface Props { items?: Item[]; onselect?: (value: string) => void; emptyText?: string }
	let { items = [], onselect, emptyText = "Belum ada data" }: Props = $props();
	const radius = 54, circumference = 2 * Math.PI * radius;
	let total = $derived(items.reduce((sum, item) => sum + item.total, 0));
	let segments = $derived.by(() => {
		let offset = 0;
		return items.map((item, index) => {
			const length = total > 0 ? (item.total / total) * circumference : 0;
			const segment = { ...item, index, length, offset };
			offset += length;
			return segment;
		});
	});
	function colorClass(index: number): string {
		return ["text-brand-500", "text-blue-500", "text-amber-500", "text-red-500", "text-violet-500", "text-neutral-500"][index % 6];
	}
</script>

{#if total > 0}
	<div class="grid gap-5 sm:grid-cols-[180px_1fr] sm:items-center">
		<div class="mx-auto h-44 w-44">
			<svg viewBox="0 0 140 140" class="h-full w-full -rotate-90" role="img" aria-label="Grafik komposisi santri">
				<circle cx="70" cy="70" r={radius} fill="none" class="stroke-neutral-100 dark:stroke-neutral-800" stroke-width="20" />
				{#each segments as segment}
					<circle cx="70" cy="70" r={radius} fill="none" stroke="currentColor" stroke-width="20"
						stroke-dasharray={`${segment.length} ${circumference - segment.length}`} stroke-dashoffset={-segment.offset}
						class={`${colorClass(segment.index)} ${onselect && segment.value ? "cursor-pointer" : ""}`}
						onclick={() => segment.value && onselect?.(segment.value)}>
						<title>{segment.label}: {segment.total} santri</title>
					</circle>
				{/each}
			</svg>
		</div>
		<div class="space-y-2.5">
			{#each segments as segment}
				<button type="button" disabled={!onselect || !segment.value} onclick={() => segment.value && onselect?.(segment.value)}
					class="flex w-full items-center justify-between gap-3 rounded-lg px-2 py-1.5 text-left hover:bg-neutral-50 disabled:hover:bg-transparent dark:hover:bg-white/[0.03]">
					<span class="flex min-w-0 items-center gap-2 text-sm text-neutral-700 dark:text-neutral-300">
						<span class={`h-2.5 w-2.5 shrink-0 rounded-full bg-current ${colorClass(segment.index)}`}></span>
						<span class="truncate">{segment.label}</span>
					</span>
					<span class="shrink-0 font-mono text-sm text-neutral-500">{segment.total.toLocaleString("id-ID")}</span>
				</button>
			{/each}
		</div>
	</div>
{:else}
	<div class="flex min-h-48 items-center justify-center text-sm text-neutral-500 dark:text-neutral-400">{emptyText}</div>
{/if}
