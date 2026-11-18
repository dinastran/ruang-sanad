<script lang="ts">
	// Jam dzikir 24 jam (WIB): tiap jam satu jari-jari, panjang = jumlah dzikir.
	// Pita amber menandai rentang dzikir pagi (≈04–10) dan petang (≈15–18).
	import { angka } from "./format";

	interface Props {
		data: number[]; // 24 jam WIB
		size?: number;
	}
	let { data, size = 232 }: Props = $props();

	const PAGI = [4, 10]; // [mulai, selesai) jam
	const PETANG = [15, 18];
	const diPagi = (h: number) => h >= PAGI[0] && h < PAGI[1];
	const diPetang = (h: number) => h >= PETANG[0] && h < PETANG[1];

	const c = $derived(size / 2);
	const r0 = $derived(size * 0.24);
	const L = $derived(size * 0.17);
	const max = $derived(Math.max(...data, 1));
	const total = $derived(data.reduce((a, v) => a + v, 0));
	const pagi = $derived(data.reduce((a, v, h) => a + (diPagi(h) ? v : 0), 0));
	const petang = $derived(data.reduce((a, v, h) => a + (diPetang(h) ? v : 0), 0));
	const puncak = $derived(total > 0 ? data.indexOf(Math.max(...data)) : -1);
	let hover = $state<number | null>(null);

	const sudut = (h: number) => (h / 24) * 2 * Math.PI - Math.PI / 2;
	const titik = (h: number, r: number) => ({ x: c + r * Math.cos(sudut(h)), y: c + r * Math.sin(sudut(h)) });
	function busur(h0: number, h1: number, r: number) {
		const a = titik(h0, r), b = titik(h1, r);
		const besar = h1 - h0 > 12 ? 1 : 0;
		return `M ${a.x} ${a.y} A ${r} ${r} 0 ${besar} 1 ${b.x} ${b.y}`;
	}
	const jam = (h: number) => `${String(h).padStart(2, "0")}.00`;
	const pct = (v: number) => (total ? Math.round((v / total) * 100) : 0);
</script>

<div class="flex flex-col items-center">
	<div class="relative" style="width:{size}px;height:{size}px">
		<svg width={size} height={size} viewBox="0 0 {size} {size}" role="img" aria-label="Sebaran dzikir per jam (WIB)">
			<circle cx={c} cy={c} r={r0 - 6} fill="none" stroke-width="3" class="stroke-neutral-100 dark:stroke-neutral-800" />
			<path d={busur(PAGI[0], PAGI[1], r0 - 6)} fill="none" stroke-width="3" stroke-linecap="round" class="stroke-warning" />
			<path d={busur(PETANG[0], PETANG[1], r0 - 6)} fill="none" stroke-width="3" stroke-linecap="round" class="stroke-warning" />
			{#each data as v, h}
				{@const a = titik(h + 0.5, r0)}
				{@const b = titik(h + 0.5, r0 + Math.max(v > 0 ? 3 : 1.5, (L * v) / max))}
				{@const ujung = titik(h + 0.5, r0 + L + 4)}
				<line
					x1={a.x} y1={a.y} x2={b.x} y2={b.y}
					stroke-width={size * 0.028}
					stroke-linecap="round"
					class="jari {v === 0 ? 'stroke-neutral-200 dark:stroke-neutral-800' : h === puncak ? 'stroke-brand-600 dark:stroke-brand-300' : 'stroke-brand-300 dark:stroke-brand-700'} {hover !== null && hover !== h ? 'opacity-50' : ''}"
				/>
				<line
					x1={a.x} y1={a.y} x2={ujung.x} y2={ujung.y}
					stroke="transparent" stroke-width={size * 0.1}
					role="presentation"
					onpointerenter={() => (hover = h)}
					onpointerleave={() => (hover = null)}
				/>
			{/each}
			{#each [0, 6, 12, 18] as h}
				{@const p = titik(h, r0 + L + size * 0.06)}
				<text x={p.x} y={p.y + 4} text-anchor="middle" class="fill-neutral-400 text-[10px] tabular-nums dark:fill-neutral-500">{String(h).padStart(2, "0")}</text>
			{/each}
		</svg>
		<div class="pointer-events-none absolute inset-0 grid place-items-center text-center">
			{#if hover !== null}
				<div>
					<p class="text-[11px] tabular-nums text-neutral-500">{jam(hover)}–{jam((hover + 1) % 24)}</p>
					<p class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(data[hover])}</p>
				</div>
			{:else if puncak >= 0}
				<div>
					<p class="text-[11px] text-neutral-500">Paling sering</p>
					<p class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{jam(puncak).slice(0, 2)}<span class="text-xs font-medium text-neutral-400">.00</span></p>
				</div>
			{:else}
				<p class="text-xs text-neutral-400">Belum ada data</p>
			{/if}
		</div>
	</div>
	<dl class="mt-3 grid w-full grid-cols-2 gap-3 text-center">
		<div class="rounded-xl bg-neutral-50 px-3 py-2 dark:bg-neutral-800/50">
			<dt class="flex items-center justify-center gap-1.5 text-[11px] text-neutral-500"><span class="h-1.5 w-1.5 rounded-full bg-warning"></span>Pagi · 04–10</dt>
			<dd class="mt-0.5 text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{pct(pagi)}%</dd>
		</div>
		<div class="rounded-xl bg-neutral-50 px-3 py-2 dark:bg-neutral-800/50">
			<dt class="flex items-center justify-center gap-1.5 text-[11px] text-neutral-500"><span class="h-1.5 w-1.5 rounded-full bg-warning"></span>Petang · 15–18</dt>
			<dd class="mt-0.5 text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{pct(petang)}%</dd>
		</div>
	</dl>
	<table class="sr-only">
		<caption>Jumlah dzikir per jam WIB</caption>
		<tbody>{#each data as v, h}<tr><td>{jam(h)}</td><td>{v}</td></tr>{/each}</tbody>
	</table>
</div>

<style>
	.jari {
		transition: opacity 150ms cubic-bezier(0.23, 1, 0.32, 1);
	}
</style>
