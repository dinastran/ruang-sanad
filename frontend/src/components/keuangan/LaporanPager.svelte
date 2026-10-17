<script lang="ts">
	import { ChevronLeft, ChevronRight } from "lucide-svelte";

	interface Props {
		page: number;
		total: number;
		pageSize: number;
		onchange: (page: number) => void;
	}

	let { page, total, pageSize, onchange }: Props = $props();
	let totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));
	let label = $derived(total === 0 ? "Tidak ada data" : `${(page - 1) * pageSize + 1}–${Math.min(page * pageSize, total)} dari ${total}`);
</script>

<div class="flex items-center justify-between gap-3 border-t border-neutral-200 px-4 py-3 text-sm dark:border-neutral-800 print:hidden">
	<p class="text-neutral-500">{label}</p>
	{#if totalPages > 1}
		<div class="flex items-center gap-1">
			<button onclick={() => onchange(page - 1)} disabled={page <= 1} aria-label="Halaman sebelumnya" class="rounded-lg p-1.5 text-neutral-600 hover:bg-neutral-100 disabled:opacity-30 dark:text-neutral-300 dark:hover:bg-neutral-800"><ChevronLeft size="16" /></button>
			<span class="px-2 text-neutral-600 dark:text-neutral-300">{page} / {totalPages}</span>
			<button onclick={() => onchange(page + 1)} disabled={page >= totalPages} aria-label="Halaman berikutnya" class="rounded-lg p-1.5 text-neutral-600 hover:bg-neutral-100 disabled:opacity-30 dark:text-neutral-300 dark:hover:bg-neutral-800"><ChevronRight size="16" /></button>
		</div>
	{/if}
</div>
