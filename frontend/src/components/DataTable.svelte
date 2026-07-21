<script lang="ts">
	import type { Snippet } from "svelte";

	interface Column {
		key: string;
		label: string;
		class?: string;
	}

	interface Props {
		columns: Column[];
		rows: Record<string, unknown>[];
		loading?: boolean;
		children?: Snippet<[unknown]>;
	}

	let { columns, rows, loading = false }: Props = $props();
</script>

<div class="overflow-x-auto rounded-xl border border-neutral-200/80 dark:border-white/[0.06]">
	<table class="w-full text-sm">
		<thead>
			<tr class="bg-neutral-50 dark:bg-neutral-900/50 border-b border-neutral-200/80 dark:border-white/[0.04]">
				{#each columns as col}
					<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider {col.class || ''}">
						{col.label}
					</th>
				{/each}
			</tr>
		</thead>
		<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
			{#if loading}
				<tr>
					<td colspan={columns.length} class="px-4 py-12 text-center text-neutral-500 dark:text-neutral-400">
						Memuat data...
					</td>
				</tr>
			{:else if rows.length === 0}
				<tr>
					<td colspan={columns.length} class="px-4 py-12 text-center text-neutral-500 dark:text-neutral-400">
						Tidak ada data
					</td>
				</tr>
			{:else}
				{#each rows as row, i}
					<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
			{#each columns as col}
				<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300 {col.class || ''}">
					{typeof row[col.key] === 'object' ? JSON.stringify(row[col.key]) : String(row[col.key] ?? '')}
				</td>
						{/each}
					</tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>
