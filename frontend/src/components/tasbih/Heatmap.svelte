<script lang="ts">
	// Kalender 1 tahun dalam bentuk butir (signature tasbih): kolom = pekan, baris = hari.
	import ChartTooltip from "./ChartTooltip.svelte";
	import { angka, tglPanjang } from "./format";
	import type { TitikHari } from "@lib/tasbih/types";

	interface Props {
		data: TitikHari[];
		target: number;
	}
	let { data, target }: Props = $props();

	const S = 13; // jarak antar butir
	const LEVEL = ["fill-neutral-200 dark:fill-neutral-800", "fill-brand-200 dark:fill-brand-900", "fill-brand-300 dark:fill-brand-700", "fill-brand-500 dark:fill-brand-500", "fill-brand-700 dark:fill-brand-300"];

	function hariKe(iso: string) {
		const [y, m, d] = iso.split("-").map(Number);
		return (new Date(y, m - 1, d).getDay() + 6) % 7; // Senin = 0
	}

	function level(total: number) {
		if (total <= 0) return 0;
		const r = total / Math.max(target, 1);
		return r < 0.34 ? 1 : r < 0.67 ? 2 : r < 1 ? 3 : 4;
	}

	const sel = $derived.by(() => {
		const off = data.length ? hariKe(data[0].tanggal) : 0;
		return data.map((d, i) => ({ ...d, col: Math.floor((i + off) / 7), row: (i + off) % 7, lv: level(d.total) }));
	});
	const kolom = $derived((sel.at(-1)?.col ?? 0) + 1);
	const bulan = $derived(
		sel.filter((c) => c.tanggal.endsWith("-01")).map((c) => ({ col: c.col, label: new Intl.DateTimeFormat("id-ID", { month: "short" }).format(new Date(c.tanggal + "T00:00:00")) })),
	);
	const W = $derived(24 + kolom * S);
	const H = 16 + 7 * S;
	let hover = $state<{ i: number; x: number; y: number } | null>(null);
	let box = $state<HTMLDivElement | null>(null);
	let scroller = $state<HTMLDivElement | null>(null);

	$effect(() => {
		// gulir ke pekan terbaru saat pertama tampil (side effect DOM)
		if (scroller) scroller.scrollLeft = scroller.scrollWidth;
	});

	function tunjuk(i: number, el: SVGElement) {
		if (!box) return;
		const r = el.getBoundingClientRect(), b = box.getBoundingClientRect();
		hover = { i, x: r.left - b.left + r.width / 2, y: r.top - b.top };
	}
</script>

<div bind:this={box} class="relative">
	<div bind:this={scroller} class="overflow-x-auto pb-1">
		<svg viewBox="0 0 {W} {H}" style="min-width:{Math.min(W * 1.4, 760)}px" class="h-auto w-full" role="img" aria-label="Kalender dzikir satu tahun terakhir">
			{#each bulan as b}
				<text x={24 + b.col * S} y="9" class="fill-neutral-400 text-[8px]">{b.label}</text>
			{/each}
			{#each ["Sen", "", "Rab", "", "Jum", "", ""] as l, r}
				{#if l}<text x="0" y={16 + r * S + 9} class="fill-neutral-400 text-[8px]">{l}</text>{/if}
			{/each}
			{#each sel as c, i (c.tanggal)}
				<circle
					cx={24 + c.col * S + S / 2}
					cy={16 + c.row * S + S / 2}
					r={c.lv === 4 ? 5.5 : 4.8}
					class={LEVEL[c.lv]}
					role="presentation"
					onpointerenter={(e) => tunjuk(i, e.currentTarget as SVGElement)}
					onpointerleave={() => (hover = null)}
				/>
			{/each}
		</svg>
	</div>
	{#if hover}
		{@const c = sel[hover.i]}
		<ChartTooltip x={hover.x} y={hover.y} judul={tglPanjang(c.tanggal)} nilai={c.total ? `${angka(c.total)} dzikir` : "Belum ada dzikir"} />
	{/if}
	<div class="mt-2 flex items-center justify-end gap-1.5 text-[11px] text-neutral-500 dark:text-neutral-400">
		<span>Sedikit</span>
		<svg width="70" height="12" aria-hidden="true">
			{#each LEVEL as l, i}<circle cx={6 + i * 14} cy="6" r="4.5" class={l} />{/each}
		</svg>
		<span>Target tercapai</span>
	</div>
</div>
