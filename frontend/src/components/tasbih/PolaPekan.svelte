<script lang="ts">
	// Rata-rata dzikir per hari dalam pekan (Senin–Ahad) dari deret harian.
	import { angka } from "./format";
	import type { TitikHari } from "@lib/tasbih/types";

	interface Props {
		data: TitikHari[];
	}
	let { data }: Props = $props();
	const NAMA = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Ahd"];

	function hariKe(iso: string) {
		const [y, m, d] = iso.split("-").map(Number);
		return (new Date(y, m - 1, d).getDay() + 6) % 7;
	}
	const rata = $derived.by(() => {
		const sum = Array(7).fill(0), n = Array(7).fill(0);
		for (const d of data) {
			const i = hariKe(d.tanggal);
			sum[i] += d.total;
			n[i]++;
		}
		return sum.map((s, i) => (n[i] ? s / n[i] : 0));
	});
	const max = $derived(Math.max(...rata, 1));
	const terkuat = $derived(rata.some((v) => v > 0) ? rata.indexOf(Math.max(...rata)) : -1);
</script>

<div>
	<div class="flex h-32 items-end gap-2" role="img" aria-label="Rata-rata dzikir per hari dalam pekan">
		{#each rata as v, i}
			<div class="flex h-full flex-1 flex-col items-center justify-end gap-1.5" title="{NAMA[i]}: rata-rata {angka(v)}">
				<span class="text-[10px] tabular-nums {i === terkuat ? 'font-semibold text-neutral-900 dark:text-white' : 'text-neutral-400'}">{v > 0 ? angka(v) : ""}</span>
				<div
					class="w-full max-w-7 rounded-md {v === 0 ? 'bg-neutral-100 dark:bg-neutral-800' : i === terkuat ? 'bg-brand-500 dark:bg-brand-400' : 'bg-brand-200 dark:bg-brand-800'}"
					style="height:{Math.max((v / max) * 100, 4)}%"
				></div>
			</div>
		{/each}
	</div>
	<div class="mt-2 flex gap-2">
		{#each NAMA as n, i}
			<span class="flex-1 text-center text-[11px] {i === 4 ? 'font-semibold text-neutral-700 dark:text-neutral-300' : 'text-neutral-500'}">{n}</span>
		{/each}
	</div>
	<p class="mt-3 text-xs text-neutral-500 dark:text-neutral-400">
		{#if terkuat >= 0}Paling istiqamah di hari <b class="font-semibold text-neutral-800 dark:text-neutral-200">{["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Ahad"][terkuat]}</b>.{:else}Belum ada pola di rentang ini.{/if}
	</p>
</div>
