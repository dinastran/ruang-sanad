<script lang="ts">
	import ChartTooltip from "./ChartTooltip.svelte";
	import { angka } from "./format";

	interface Props {
		data: number[]; // 24 jam WIB
	}
	let { data }: Props = $props();
	let lebarBox = $state(480);
	const W = $derived(Math.max(lebarBox, 240));
	const H = 130, PB = 20, PT = 8;
	const max = $derived(Math.max(...data, 1));
	const lebar = $derived((W - 8) / 24);
	const puncak = $derived(data.indexOf(Math.max(...data)));
	let hover = $state<{ i: number; x: number; y: number } | null>(null);
	let box = $state<HTMLDivElement | null>(null);

	function tunjuk(i: number, el: SVGElement) {
		if (!box) return;
		const r = el.getBoundingClientRect(), b = box.getBoundingClientRect();
		hover = { i, x: r.left - b.left + r.width / 2, y: r.top - b.top };
	}
	const jam = (i: number) => `${String(i).padStart(2, "0")}.00`;
</script>

<div bind:this={box} bind:clientWidth={lebarBox} class="relative">
	<svg viewBox="0 0 {W} {H}" width={W} height={H} class="block" role="img" aria-label="Jumlah dzikir per jam (WIB)">
		{#each data as v, i}
			{@const h = ((H - PB - PT) * v) / max}
			<rect x={4 + i * lebar + 2} y={H - PB - Math.max(h, v > 0 ? 2 : 0)} width={lebar - 4} height={Math.max(h, v > 0 ? 2 : 0)} rx="3"
				class="{i === puncak && v > 0 ? 'fill-brand-500 dark:fill-brand-400' : 'fill-brand-200 dark:fill-brand-800'}" />
			<rect x={4 + i * lebar} y={PT} width={lebar} height={H - PB - PT} fill="transparent" role="presentation"
				onpointerenter={(e) => tunjuk(i, e.currentTarget as SVGElement)} onpointerleave={() => (hover = null)} />
		{/each}
		{#each [0, 6, 12, 18, 23] as i}
			<text x={4 + i * lebar + lebar / 2} y={H - 5} text-anchor="middle" class="fill-neutral-500 text-[11px] dark:fill-neutral-400">{jam(i).slice(0, 2)}</text>
		{/each}
	</svg>
	{#if hover}
		<ChartTooltip x={hover.x} y={hover.y} judul="{jam(hover.i)}–{jam((hover.i + 1) % 24)} WIB" nilai="{angka(data[hover.i])} dzikir" />
	{/if}
</div>
