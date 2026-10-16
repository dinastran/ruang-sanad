<script lang="ts">
	import type { KeuanganBulan } from "@lib/types";

	interface Props { items?: KeuanganBulan[]; selected?: string; onselect?: (bulan: string) => void }
	let { items = [], selected = "", onselect }: Props = $props();

	const width = 720, height = 260, padX = 56, padTop = 20, padBottom = 36;
	const plotH = height - padTop - padBottom;

	let max = $derived(Math.max(1, ...items.flatMap((b) => [b.kas_masuk, b.kas_keluar, b.kas_bersih])));
	let min = $derived(Math.min(0, ...items.map((b) => b.kas_bersih)));
	let slot = $derived(items.length ? (width - padX * 2) / items.length : 0);
	let barW = $derived(Math.min(28, slot / 3.2));

	const y = (v: number) => padTop + ((max - v) / (max - min || 1)) * plotH;
	const cx = (i: number) => padX + slot * i + slot / 2;

	let netPath = $derived(items.map((b, i) => `${i ? "L" : "M"}${cx(i)},${y(b.kas_bersih)}`).join(" "));
	let ticks = $derived([max, max / 2, 0, ...(min < 0 ? [min] : [])]);

	function compact(n: number): string {
		const abs = Math.abs(n);
		if (abs >= 1_000_000) return `${(n / 1_000_000).toLocaleString("id-ID", { maximumFractionDigits: 1 })} jt`;
		if (abs >= 1_000) return `${Math.round(n / 1_000).toLocaleString("id-ID")} rb`;
		return n.toLocaleString("id-ID");
	}
</script>

<div>
	<svg viewBox={`0 0 ${width} ${height}`} class="w-full h-auto" role="img" aria-label="Grafik kas masuk, kas keluar, dan kas bersih 6 bulan">
		{#each ticks as t}
			<line x1={padX} x2={width - padX} y1={y(t)} y2={y(t)} class={t === 0 ? "stroke-neutral-300 dark:stroke-neutral-700" : "stroke-neutral-100 dark:stroke-neutral-800"} stroke-width="1" />
			<text x={padX - 8} y={y(t) + 4} text-anchor="end" class="fill-neutral-400 text-[11px]">{compact(t)}</text>
		{/each}
		{#each items as b, i}
			<g class={onselect ? "cursor-pointer" : ""} onclick={() => onselect?.(b.bulan)} role="presentation">
				{#if b.bulan === selected}<rect x={padX + slot * i + 2} y={padTop - 8} width={slot - 4} height={plotH + 16} rx="10" class="fill-brand-500/5" />{/if}
				<rect x={cx(i) - barW - 2} y={y(Math.max(b.kas_masuk, 0))} width={barW} height={Math.max(y(0) - y(b.kas_masuk), 0)} rx="4" class="fill-success/80"><title>{b.label} · Kas masuk Rp{b.kas_masuk.toLocaleString("id-ID")}</title></rect>
				<rect x={cx(i) + 2} y={y(Math.max(b.kas_keluar, 0))} width={barW} height={Math.max(y(0) - y(b.kas_keluar), 0)} rx="4" class="fill-error/70"><title>{b.label} · Kas keluar Rp{b.kas_keluar.toLocaleString("id-ID")}</title></rect>
				<text x={cx(i)} y={height - 12} text-anchor="middle" class="text-[11px] {b.bulan === selected ? 'fill-brand-600 dark:fill-brand-400 font-semibold' : 'fill-neutral-500'}">{b.label}</text>
			</g>
		{/each}
		{#if items.length > 1}<path d={netPath} fill="none" class="stroke-brand-600 dark:stroke-brand-400" stroke-width="2.5" stroke-linejoin="round" />{/if}
		{#each items as b, i}
			<circle cx={cx(i)} cy={y(b.kas_bersih)} r="4" class="fill-white stroke-brand-600 dark:fill-neutral-925 dark:stroke-brand-400" stroke-width="2"><title>{b.label} · Kas bersih Rp{b.kas_bersih.toLocaleString("id-ID")}</title></circle>
		{/each}
	</svg>
	<div class="mt-2 flex flex-wrap gap-4 text-xs text-neutral-600 dark:text-neutral-400">
		<span class="inline-flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm bg-success/80"></span>Kas masuk</span>
		<span class="inline-flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm bg-error/70"></span>Kas keluar (ujroh dibayar)</span>
		<span class="inline-flex items-center gap-1.5"><span class="h-0.5 w-4 rounded bg-brand-600 dark:bg-brand-400"></span>Kas bersih</span>
	</div>
</div>
