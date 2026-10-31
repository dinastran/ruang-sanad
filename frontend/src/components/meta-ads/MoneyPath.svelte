<script lang="ts">
	import { ChevronRight } from "lucide-svelte";
	import { change, compact, num, pct, ratio, rp, rpShort, signedPct, type Metrics } from "./data";

	interface Props {
		cur: Metrics;
		prev: Metrics | null;
		/** Spend of campaigns mapped to a lead source: the base for Rp/lead and Rp/santri. */
		mappedSpend: number;
		prevMappedSpend: number;
		hasMapping: boolean;
	}
	let { cur, prev, mappedSpend, prevMappedSpend, hasMapping }: Props = $props();

	interface Step {
		key: string;
		label: string;
		value: string;
		title: string;
		cost: string; // cost per unit at this step
		costValue: number;
		prevCostValue: number;
		conv: string; // conversion from the previous step
		convLabel: string;
		delta: number; // change of the count
		crm: boolean;
	}

	let steps = $derived.by<Step[]>(() => {
		const p = prev;
		const crmOk = hasMapping;
		return [
			{ key: "spend", label: "Biaya iklan", value: rpShort(cur.spend), title: rp(cur.spend), cost: "dibelanjakan", costValue: NaN, prevCostValue: NaN, conv: "", convLabel: "", delta: p ? change(cur.spend, p.spend) : NaN, crm: false },
			{ key: "impressions", label: "Impresi", value: compact(cur.impressions), title: num(cur.impressions), cost: `${rp(ratio(cur.spend, cur.impressions) * 1000)} / 1.000`, costValue: ratio(cur.spend, cur.impressions) * 1000, prevCostValue: p ? ratio(p.spend, p.impressions) * 1000 : NaN, conv: "", convLabel: "", delta: p ? change(cur.impressions, p.impressions) : NaN, crm: false },
			{ key: "link", label: "Klik tautan", value: num(cur.link), title: num(cur.link), cost: `${rp(ratio(cur.spend, cur.link))} / klik`, costValue: ratio(cur.spend, cur.link), prevCostValue: p ? ratio(p.spend, p.link) : NaN, conv: pct(ratio(cur.link, cur.impressions), 2), convLabel: "CTR", delta: p ? change(cur.link, p.link) : NaN, crm: false },
			{ key: "chats", label: "Chat WA (Meta)", value: num(cur.chats), title: num(cur.chats), cost: `${rp(ratio(cur.spend, cur.chats))} / chat`, costValue: ratio(cur.spend, cur.chats), prevCostValue: p ? ratio(p.spend, p.chats) : NaN, conv: pct(ratio(cur.chats, cur.link)), convLabel: "dari klik", delta: p ? change(cur.chats, p.chats) : NaN, crm: false },
			{ key: "leads", label: "Lead (CRM)", value: crmOk ? num(cur.leads) : "—", title: crmOk ? num(cur.leads) : "Belum ada kampanye terhubung", cost: crmOk ? `${rp(ratio(mappedSpend, cur.leads))} / lead` : "hubungkan kampanye", costValue: crmOk ? ratio(mappedSpend, cur.leads) : NaN, prevCostValue: p && crmOk ? ratio(prevMappedSpend, p.leads) : NaN, conv: crmOk ? pct(ratio(cur.leads, cur.chats)) : "", convLabel: "dari chat", delta: p && crmOk ? change(cur.leads, p.leads) : NaN, crm: true },
			{ key: "won", label: "Jadi santri", value: crmOk ? num(cur.won) : "—", title: crmOk ? num(cur.won) : "Belum ada kampanye terhubung", cost: "", costValue: crmOk ? ratio(mappedSpend, cur.won) : NaN, prevCostValue: p && crmOk ? ratio(prevMappedSpend, p.won) : NaN, conv: crmOk ? pct(ratio(cur.won, cur.leads)) : "", convLabel: "dari lead", delta: p && crmOk ? change(cur.won, p.won) : NaN, crm: true },
		];
	});
	let focal = $derived(steps[5]);
	let focalDelta = $derived(change(focal.costValue, focal.prevCostValue));

	/** Up is good for counts; for costs, down is good. */
	const tone = (v: number, costLike = false) => {
		if (!Number.isFinite(v) || Math.abs(v) < 0.005) return "text-neutral-500 dark:text-neutral-400";
		const good = costLike ? v < 0 : v > 0;
		return good ? "text-emerald-700 dark:text-emerald-400" : "text-amber-700 dark:text-amber-400";
	};
</script>

<section class="overflow-hidden rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60" aria-labelledby="jalur-title">
	<div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-5 pt-5 sm:px-6">
		<h2 id="jalur-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Jalur rupiah → santri</h2>
		<p class="text-xs text-neutral-500 dark:text-neutral-400">Biaya per langkah dan konversi dari langkah sebelumnya{prev ? " · ↑↓ dibanding periode lalu" : ""}</p>
	</div>

	<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-[repeat(5,minmax(0,1fr))_minmax(0,1.35fr)]">
		<ol class="contents">
			{#each steps.slice(0, 5) as s, i (s.key)}
				<li class={"relative min-w-0 border-b border-r border-neutral-100 px-4 py-4 even:border-r-0 sm:px-6 sm:even:border-r sm:[&:nth-child(3n)]:border-r-0 lg:border-b-0 lg:py-5 lg:[&:nth-child(3n)]:border-r dark:border-white/[0.05] " + (i === 4 ? "col-span-2 border-r-0 sm:col-span-1" : "")}>
					{#if i > 0}
						<span class="absolute -left-[9px] top-1/2 hidden -translate-y-1/2 rounded-full bg-white p-0.5 text-neutral-300 lg:block dark:bg-neutral-925 dark:text-neutral-600" aria-hidden="true"><ChevronRight size={14} /></span>
					{/if}
					<p class="truncate text-xs font-medium text-neutral-600 dark:text-neutral-300">{s.label}</p>
					<p class="mt-1.5 text-[22px] font-semibold leading-7 tracking-[-0.02em] tabular-nums text-neutral-950 dark:text-white" title={s.title}>{s.value}</p>
					<p class={"mt-1 text-xs tabular-nums " + (s.crm && !hasMapping ? "text-amber-700 dark:text-amber-400" : "text-neutral-500 dark:text-neutral-400")}>{s.cost}</p>
					<p class="mt-0.5 flex flex-wrap items-baseline gap-x-2 text-xs tabular-nums">
						{#if s.conv}<span class="text-neutral-600 dark:text-neutral-300" title={`${s.conv} ${s.convLabel}`}>{s.conv} <span class="text-neutral-400">{s.convLabel}</span></span>{/if}
						{#if Number.isFinite(s.delta)}<span class={"font-medium " + (s.key === "spend" ? "text-neutral-500 dark:text-neutral-400" : tone(s.delta))}>{signedPct(s.delta)}</span>{/if}
					</p>
				</li>
			{/each}
		</ol>

		<!-- Focal: what one santri costs. -->
		<div class="relative col-span-2 bg-brand-500/[0.06] px-5 py-5 sm:col-span-3 sm:px-6 lg:col-span-1 dark:bg-brand-400/[0.07]">
			<span class="absolute -left-[9px] top-1/2 hidden -translate-y-1/2 rounded-full bg-white p-0.5 text-neutral-300 lg:block dark:bg-neutral-925 dark:text-neutral-600" aria-hidden="true"><ChevronRight size={14} /></span>
			<p class="flex items-center justify-between gap-2 text-xs font-medium text-brand-800 dark:text-brand-300">
				<span>Biaya per santri</span>
				{#if focal.conv}<span class="rounded bg-white/70 px-1.5 py-px text-[10px] font-medium tabular-nums text-neutral-600 dark:bg-white/[0.06] dark:text-neutral-300">{focal.conv} <span class="text-neutral-400">{focal.convLabel}</span></span>{/if}
			</p>
			<p class="mt-1.5 text-[40px] font-semibold leading-[44px] tracking-[-0.03em] tabular-nums text-neutral-950 dark:text-white" title={rp(focal.costValue)}>{hasMapping ? rpShort(focal.costValue) : "—"}</p>
			<p class="mt-1.5 flex flex-wrap items-baseline gap-x-2 text-xs tabular-nums text-neutral-600 dark:text-neutral-300">
				{#if hasMapping}
					<span><strong class="font-semibold text-neutral-900 dark:text-white">{focal.value}</strong> santri dari iklan</span>
					{#if Number.isFinite(focalDelta)}<span class={"font-medium " + tone(focalDelta, true)}>{signedPct(focalDelta)} biaya/santri</span>{/if}
				{:else}
					<span class="text-amber-700 dark:text-amber-400">Hubungkan kampanye ke sumber lead di tabel di bawah agar lead &amp; santri terhitung.</span>
				{/if}
			</p>
		</div>
	</div>
</section>
