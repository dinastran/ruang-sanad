<script lang="ts">
	interface Item { label: string; value: string; total: number }
	interface Props { items?: Item[]; onselect?: (value: string) => void; emptyText?: string }
	let { items = [], onselect, emptyText = "Belum ada data" }: Props = $props();
	let max = $derived(items.length ? Math.max(...items.map((item) => item.total), 1) : 1);
</script>

{#if items.length > 0}
	<div class="space-y-3">
		{#each items as item}
			{@const selectable = Boolean(onselect && item.value)}
			<button type="button" disabled={!selectable} onclick={() => selectable && onselect?.(item.value)}
				class="group block w-full text-left disabled:cursor-default"
				aria-label={selectable ? `Filter ${item.label}, ${item.total} santri` : `${item.label}, ${item.total} santri`}>
				<div class="mb-1.5 flex items-center justify-between gap-4 text-sm">
					<span class="truncate font-medium text-neutral-700 group-hover:text-brand-600 dark:text-neutral-300 dark:group-hover:text-brand-400">{item.label}</span>
					<span class="shrink-0 font-mono text-neutral-500 dark:text-neutral-400">{item.total.toLocaleString("id-ID")}</span>
				</div>
				<div class="h-2.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
					<div class="h-full rounded-full bg-brand-500 transition-all duration-500 group-hover:bg-brand-600"
						style={`width: ${Math.max((item.total / max) * 100, item.total > 0 ? 2 : 0)}%`}></div>
				</div>
			</button>
		{/each}
	</div>
{:else}
	<div class="flex min-h-40 items-center justify-center text-sm text-neutral-500 dark:text-neutral-400">{emptyText}</div>
{/if}
