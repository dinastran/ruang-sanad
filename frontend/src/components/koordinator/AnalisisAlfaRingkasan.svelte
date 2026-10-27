<script lang="ts">
	import type { AnalisisKehadiran, FilterAlfa } from "@lib/types";
	import { AlertTriangle, CheckCircle2 } from "lucide-svelte";

	interface Props {
		analisis: AnalisisKehadiran;
		onpilih: (filter: FilterAlfa) => void;
	}

	let { analisis, onpilih }: Props = $props();

	const adaMasalah = $derived(analisis.merah > 0 || analisis.waspada > 0);
	const persen = $derived(analisis.persen_hadir === null ? "–" : `${Math.round(analisis.persen_hadir)}%`);
</script>

<section class="rounded-2xl border p-4 sm:p-5 {adaMasalah ? 'border-amber-500/30 bg-amber-500/[0.06]' : 'border-success/20 bg-success/10'}" aria-label="Ringkasan kehadiran guru">
	{#if adaMasalah}
		<p class="flex items-center gap-2 text-sm font-semibold text-amber-800 dark:text-amber-300"><AlertTriangle class="h-4 w-4" /> Perlu perhatian</p>
	{:else}
		<p class="flex items-center gap-2 text-sm font-semibold text-emerald-800 dark:text-emerald-300"><CheckCircle2 class="h-4 w-4" /> Tidak ada guru yang alfa berturut-turut</p>
	{/if}
	<div class="mt-3 grid grid-cols-3 gap-2">
		<button type="button" onclick={() => onpilih("merah")} class="rounded-xl bg-white px-3 py-2.5 text-center transition-colors hover:bg-neutral-50 dark:bg-neutral-925/60 dark:hover:bg-neutral-900">
			<p class="text-xl font-bold tabular-nums {analisis.merah > 0 ? 'text-red-600 dark:text-red-400' : 'text-neutral-400 dark:text-neutral-600'}">{analisis.merah}</p>
			<p class="text-xs text-neutral-600 dark:text-neutral-400">Alfa ≥{analisis.ambang_alfa}× beruntun</p>
		</button>
		<button type="button" onclick={() => onpilih("waspada")} class="rounded-xl bg-white px-3 py-2.5 text-center transition-colors hover:bg-neutral-50 dark:bg-neutral-925/60 dark:hover:bg-neutral-900">
			<p class="text-xl font-bold tabular-nums {analisis.waspada > 0 ? 'text-orange-600 dark:text-orange-400' : 'text-neutral-400 dark:text-neutral-600'}">{analisis.waspada}</p>
			<p class="text-xs text-neutral-600 dark:text-neutral-400">Waspada (alfa {analisis.ambang_waspada}×)</p>
		</button>
		<button type="button" onclick={() => onpilih("semua")} class="rounded-xl bg-white px-3 py-2.5 text-center transition-colors hover:bg-neutral-50 dark:bg-neutral-925/60 dark:hover:bg-neutral-900">
			<p class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">{persen}</p>
			<p class="text-xs text-neutral-600 dark:text-neutral-400">Kehadiran {analisis.sesi.length} sesi terakhir</p>
		</button>
	</div>
</section>
