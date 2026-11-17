<script lang="ts">
	import ChartTooltip from "./ChartTooltip.svelte";
	import { angka, tglPanjang, tglPendek } from "./format";
	import type { TitikHari } from "@lib/tasbih/types";

	interface Props {
		data: TitikHari[];
		target: number;
	}
	let { data, target }: Props = $props();

	let lebarBox = $state(640);
	const W = $derived(Math.max(lebarBox, 280));
	const H = 200, PL = 36, PR = 8, PT = 12, PB = 26;
	const max = $derived(Math.max(target, ...data.map((d) => d.total), 1) * 1.08);
	const lebar = $derived((W - PL - PR) / Math.max(data.length, 1));
	const barW = $derived(Math.max(2, Math.min(28, lebar - 2)));
	const y = (v: number) => PT + (H - PT - PB) * (1 - v / max);
	const labelTiap = $derived(Math.max(1, Math.ceil(data.length / Math.max(3, Math.floor(W / 80)))));
	let hover = $state<{ i: number; x: number; y: number } | null>(null);
	let box = $state<HTMLDivElement | null>(null);

	function tunjuk(i: number, el: SVGElement) {
		if (!box) return;
		const r = el.getBoundingClientRect(), b = box.getBoundingClientRect();
		hover = { i, x: r.left - b.left + r.width / 2, y: r.top - b.top };
	}
</script>

<div bind:this={box} bind:clientWidth={lebarBox} class="relative w-full">
	<svg viewBox="0 0 {W} {H}" width={W} height={H} class="block" role="img" aria-label="Total dzikir per hari selama {data.length} hari terakhir">
		{#each [0, 0.5, 1] as f}
			<line x1={PL} x2={W - PR} y1={y(max * f / 1.08)} y2={y(max * f / 1.08)} class="stroke-neutral-100 dark:stroke-neutral-800" />
			<text x={PL - 6} y={y(max * f / 1.08) + 3} text-anchor="end" class="fill-neutral-400 text-[11px] tabular-nums">{angka(max * f / 1.08)}</text>
		{/each}
		{#each data as d, i (d.tanggal)}
			{@const x = PL + i * lebar + (lebar - barW) / 2}
			{@const h = Math.max(0, H - PB - y(d.total))}
			<rect
				{x}
				y={H - PB - h}
				width={barW}
				height={Math.max(h, d.total > 0 ? 2 : 0)}
				rx={Math.min(4, barW / 2)}
				class="{d.total >= target ? 'fill-brand-500 dark:fill-brand-400' : 'fill-brand-200 dark:fill-brand-800'} transition-opacity {hover && hover.i !== i ? 'opacity-60' : ''}"
			/>
			<!-- area hover lebih besar dari mark -->
			<rect
				x={PL + i * lebar}
				y={PT}
				width={lebar}
				height={H - PT - PB}
				fill="transparent"
				role="presentation"
				onpointerenter={(e) => tunjuk(i, e.currentTarget as SVGElement)}
				onpointerleave={() => (hover = null)}
			/>
			{#if i % labelTiap === 0 || i === data.length - 1}
				<text x={i === data.length - 1 ? W - PR : PL + i * lebar + lebar / 2} y={H - 8} text-anchor={i === data.length - 1 ? "end" : "middle"} class="fill-neutral-500 text-[11px] dark:fill-neutral-400">{tglPendek(d.tanggal)}</text>
			{/if}
		{/each}
		<line x1={PL} x2={W - PR} y1={y(target)} y2={y(target)} stroke-width="1.5" stroke-dasharray="4 4" class="stroke-warning" />
		<text x={PL + 4} y={y(target) - 4} text-anchor="start" class="fill-warning text-[11px] font-medium">target {angka(target)}</text>
	</svg>
	{#if hover}
		{@const d = data[hover.i]}
		<ChartTooltip x={hover.x} y={hover.y} judul={tglPanjang(d.tanggal)} nilai="{angka(d.total)} dzikir" />
	{/if}
	<table class="sr-only">
		<caption>Total dzikir per hari</caption>
		<tbody>{#each data as d}<tr><td>{d.tanggal}</td><td>{d.total}</td></tr>{/each}</tbody>
	</table>
</div>
