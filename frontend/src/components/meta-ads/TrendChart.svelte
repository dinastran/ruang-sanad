<script lang="ts">
	import { compact, num, pct, ratio, rp, rpShort, shortDay, type Metrics } from "./data";

	interface Props { days: string[]; cur: Metrics[]; prev: Metrics[] | null; prevDays: string[]; hasMapping: boolean }
	let { days, cur, prev, prevDays, hasMapping }: Props = $props();

	type Key = "spend" | "chats" | "leads" | "cpc" | "ctr";
	const metrics: { key: Key; label: string; fmt: (v: number) => string; axis: (v: number) => string; value: (m: Metrics) => number; crm?: boolean }[] = [
		{ key: "spend", label: "Biaya", fmt: rp, axis: rpShort, value: (m) => m.spend },
		{ key: "chats", label: "Chat WA", fmt: num, axis: compact, value: (m) => m.chats },
		{ key: "leads", label: "Lead CRM", fmt: num, axis: compact, value: (m) => m.leads, crm: true },
		{ key: "cpc", label: "Biaya/chat", fmt: rp, axis: rpShort, value: (m) => ratio(m.spend, m.chats) },
		{ key: "ctr", label: "CTR", fmt: (v) => pct(v, 2), axis: (v) => pct(v, 1), value: (m) => ratio(m.link, m.impressions) },
	];
	let active = $state<Key>("spend");
	let metric = $derived(metrics.find((m) => m.key === active)!);
	let curVals = $derived(cur.map(metric.value));
	let prevVals = $derived(prev ? prev.map(metric.value) : []);
	let max = $derived(Math.max(1e-9, ...[...curVals, ...prevVals].filter(Number.isFinite)) * 1.1);
	let hover = $state<number | null>(null);

	// The SVG is drawn at its real pixel width so text stays 11px at any size.
	let W = $state(640);
	const H = 220, L = 64, R = 12, T = 12, B = 28;
	const x = (i: number, n: number) => L + (n <= 1 ? (W - L - R) / 2 : (i * (W - L - R)) / (n - 1));
	const y = (v: number) => T + (H - T - B) * (1 - v / max);
	function path(vals: number[]) {
		let d = "";
		let pen = false;
		vals.forEach((v, i) => {
			if (!Number.isFinite(v)) { pen = false; return; }
			d += `${pen ? "L" : "M"}${x(i, vals.length).toFixed(1)},${y(v).toFixed(1)}`;
			pen = true;
		});
		return d;
	}
	let curPath = $derived(path(curVals));
	let areaPath = $derived.by(() => {
		if (curVals.length < 2 || curVals.some((v) => !Number.isFinite(v))) return "";
		return `${curPath}L${x(curVals.length - 1, curVals.length)},${H - B}L${x(0, curVals.length)},${H - B}Z`;
	});
	let labelStep = $derived(Math.max(1, Math.ceil(days.length / Math.max(2, Math.floor(W / 80)))));
	let total = $derived.by(() => {
		const sum = (arr: Metrics[]) => arr.reduce((a, m) => ({ spend: a.spend + m.spend, chats: a.chats + m.chats, leads: a.leads + m.leads, link: a.link + m.link, impressions: a.impressions + m.impressions, clicks: 0, won: 0 }), { spend: 0, chats: 0, leads: 0, link: 0, impressions: 0, clicks: 0, won: 0 } as Metrics);
		return { cur: metric.value(sum(cur)), prev: prev ? metric.value(sum(prev)) : NaN };
	});
	function onMove(e: PointerEvent) {
		const svg = e.currentTarget as SVGSVGElement;
		const rect = svg.getBoundingClientRect();
		const px = ((e.clientX - rect.left) / rect.width) * W;
		const n = days.length;
		const i = n <= 1 ? 0 : Math.round(((px - L) / (W - L - R)) * (n - 1));
		hover = Math.min(n - 1, Math.max(0, i));
	}
	const tab = "rounded-md px-2.5 py-1 text-xs font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
</script>

<section class="min-w-0 rounded-xl border border-neutral-200/80 bg-white p-5 sm:p-6 dark:border-white/[0.07] dark:bg-neutral-925/60" aria-labelledby="tren-title">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h2 id="tren-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Tren harian</h2>
			<p class="mt-1 text-sm tabular-nums text-neutral-600 dark:text-neutral-300">
				<strong class="font-semibold text-neutral-950 dark:text-white">{metric.fmt(total.cur)}</strong> periode ini{#if prev} · <span class="text-neutral-500">{metric.fmt(total.prev)} periode lalu</span>{/if}
			</p>
		</div>
		<div class="inline-flex flex-wrap gap-0.5 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900" role="tablist" aria-label="Metrik grafik">
			{#each metrics as m (m.key)}
				{#if !m.crm || hasMapping}
					<button type="button" role="tab" aria-selected={active === m.key} class={tab + (active === m.key ? " bg-white text-neutral-900 shadow-[0_0_0_1px_rgba(15,23,42,.08)] dark:bg-neutral-800 dark:text-white" : " text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white")} onclick={() => (active = m.key)}>{m.label}</button>
				{/if}
			{/each}
		</div>
	</div>

	<div class="relative mt-4" bind:clientWidth={W}>
		<svg width={W} height={H} viewBox={`0 0 ${W} ${H}`} class="block overflow-visible" role="img" aria-label={`${metric.label} per hari`} onpointermove={onMove} onpointerleave={() => (hover = null)}>
			{#each [0, 0.5, 1] as f}
				<line x1={L} x2={W - R} y1={y(max * f / 1.1)} y2={y(max * f / 1.1)} class="stroke-neutral-200 dark:stroke-neutral-800" stroke-dasharray="3 5" />
				<text x={L - 8} y={y(max * f / 1.1) + 4} text-anchor="end" class="fill-neutral-500 text-[11px] tabular-nums">{metric.axis(max * f / 1.1)}</text>
			{/each}
			{#if prev}
				<path d={path(prevVals)} fill="none" class="stroke-neutral-400 dark:stroke-neutral-500" stroke-width="1.5" stroke-dasharray="4 4" />
			{/if}
			{#if areaPath}<path d={areaPath} class="fill-brand-500/10 dark:fill-brand-400/10" />{/if}
			<path d={curPath} fill="none" class="stroke-brand-600 dark:stroke-brand-400" stroke-width="2.25" stroke-linejoin="round" stroke-linecap="round" />
			{#each days as d, i}
				{#if i % labelStep === 0}<text x={x(i, days.length)} y={H - 8} text-anchor="middle" class="fill-neutral-500 text-[11px]">{shortDay(d)}</text>{/if}
			{/each}
			{#if hover !== null}
				<line x1={x(hover, days.length)} x2={x(hover, days.length)} y1={T} y2={H - B} class="stroke-neutral-300 dark:stroke-neutral-700" />
				{#if Number.isFinite(curVals[hover])}<circle cx={x(hover, days.length)} cy={y(curVals[hover])} r="4" class="fill-brand-600 stroke-white dark:fill-brand-400 dark:stroke-neutral-925" stroke-width="2" />{/if}
			{/if}
		</svg>
		{#if hover !== null}
			<div class="pointer-events-none absolute top-0 z-10 min-w-36 -translate-x-1/2 rounded-lg border border-neutral-200 bg-white px-3 py-2 text-xs shadow-[0_4px_12px_rgba(15,23,42,.08)] dark:border-white/10 dark:bg-neutral-900" style={`left:${Math.min(85, Math.max(15, (x(hover, days.length) / W) * 100))}%`}>
				<p class="font-medium text-neutral-900 dark:text-white">{shortDay(days[hover])}</p>
				<p class="mt-0.5 tabular-nums text-neutral-700 dark:text-neutral-300"><i class="mr-1 inline-block h-2 w-2 rounded-sm bg-brand-600 dark:bg-brand-400"></i>{metric.fmt(curVals[hover])}</p>
				{#if prev && prevDays[hover]}<p class="tabular-nums text-neutral-500"><i class="mr-1 inline-block h-0.5 w-2 bg-neutral-400 align-middle"></i>{shortDay(prevDays[hover])}: {metric.fmt(prevVals[hover])}</p>{/if}
			</div>
		{/if}
	</div>

	<div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-neutral-500 dark:text-neutral-400">
		<span class="flex flex-wrap gap-3">
			<span class="inline-flex items-center gap-1.5"><i class="h-0.5 w-4 rounded bg-brand-600 dark:bg-brand-400"></i>Periode ini</span>
			{#if prev}<span class="inline-flex items-center gap-1.5"><i class="h-0 w-4 border-t-[1.5px] border-dashed border-neutral-400"></i>Periode lalu (hari ke-n yang sama)</span>{/if}
		</span>
		<details>
			<summary class="cursor-pointer">Lihat angka</summary>
			<div class="mt-2 max-h-64 overflow-auto">
				<table class="w-full text-left tabular-nums">
					<thead><tr><th class="py-1 pr-4 font-medium">Tanggal</th><th class="py-1 pr-4 text-right font-medium">{metric.label}</th>{#if prev}<th class="py-1 text-right font-medium">Periode lalu</th>{/if}</tr></thead>
					<tbody>
						{#each days as d, i}<tr class="border-t border-neutral-100 dark:border-white/[0.05]"><td class="py-1 pr-4">{shortDay(d)}</td><td class="py-1 pr-4 text-right">{metric.fmt(curVals[i])}</td>{#if prev}<td class="py-1 text-right">{metric.fmt(prevVals[i])}</td>{/if}</tr>{/each}
					</tbody>
				</table>
			</div>
		</details>
	</div>
</section>
