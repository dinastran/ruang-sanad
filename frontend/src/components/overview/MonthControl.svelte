<script lang="ts">
	import { ChevronLeft, ChevronRight, RotateCw } from "lucide-svelte";
	import { focusRing } from "./ui";

	let { bulan, maxMonth, loading, updated, onvisit }: {
		bulan: string;
		maxMonth: string;
		loading: boolean;
		updated: string;
		onvisit: (month: string) => void;
	} = $props();

	function shift(offset: number) {
		const [year, month] = bulan.split("-").map(Number);
		onvisit(new Date(Date.UTC(year, month - 1 + offset, 1)).toISOString().slice(0, 7));
	}
	const button = "inline-flex h-9 w-9 items-center justify-center rounded-md text-neutral-600 transition-colors duration-150 hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-40 dark:text-neutral-300 dark:hover:bg-white/[0.06] dark:hover:text-white " + focusRing;
</script>

<div class="flex flex-col items-start gap-1.5 sm:items-end">
	<div class="flex items-center gap-2">
		<div class="flex items-center rounded-lg border border-neutral-200 bg-white p-0.5 dark:border-white/[0.08] dark:bg-neutral-925">
			<button type="button" class={button} aria-label="Bulan sebelumnya" disabled={loading || !bulan || bulan <= "2000-01"} onclick={() => shift(-1)}><ChevronLeft size={16} /></button>
			<label for="overview-month" class="sr-only">Periode aktivitas</label>
			<input
				id="overview-month"
				type="month"
				value={bulan}
				min="2000-01"
				max={maxMonth}
				disabled={loading}
				class="h-9 w-[9.5rem] rounded-md bg-transparent px-2 text-center text-sm font-medium tabular-nums text-neutral-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-white dark:[color-scheme:dark]"
				onchange={(e) => {
					if (e.currentTarget.validity.valid && e.currentTarget.value) onvisit(e.currentTarget.value);
				}}
			/>
			<button type="button" class={button} aria-label="Bulan berikutnya" disabled={loading || !bulan || bulan >= maxMonth} onclick={() => shift(1)}><ChevronRight size={16} /></button>
		</div>
		<button type="button" class={button + " border border-neutral-200 bg-white dark:border-white/[0.08] dark:bg-neutral-925"} aria-label="Perbarui data" title="Perbarui data" disabled={loading} onclick={() => onvisit(bulan)}>
			<RotateCw size={15} class={loading ? "animate-spin motion-reduce:animate-none" : ""} />
		</button>
	</div>
	{#if updated}<p class="text-[11px] tabular-nums text-neutral-500 dark:text-neutral-400">{loading ? "Memperbarui…" : `Diperbarui ${updated}`}</p>{/if}
</div>
