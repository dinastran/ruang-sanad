<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User } from "@lib/types";
	import { STAGE_LABELS, LOST_REASON_LABELS } from "@lib/leads";
	import { ArrowLeft, Download } from "lucide-svelte";

	interface CountByKey {
		key: string;
		count: number;
	}
	interface Report {
		from: string;
		to: string;
		total_leads: number;
		won: number;
		lost: number;
		do_not_contact: number;
		open: number;
		funnel: { stage: string; reached: number; conversion: number }[];
		median_hours_to_won: number;
		lost_reasons: CountByKey[];
		objections: CountByKey[];
		weekly: { week: string; leads: number; won: number }[];
		followup_sent: number;
		followup_replied: number;
		followup_won: number;
	}
	interface Props {
		user?: User;
		report: Report;
	}

	let props: Props = $props();
	let r = $derived(props.report);
	// svelte-ignore state_referenced_locally
	let from = $state(props.report.from);
	// svelte-ignore state_referenced_locally
	let to = $state(props.report.to);

	let maxReached = $derived(Math.max(1, ...r.funnel.map((f) => f.reached)));
	let winRate = $derived(r.total_leads ? Math.round((r.won / r.total_leads) * 100) : 0);
	let exportUrl = $derived(`/app/leads/laporan/export?from=${encodeURIComponent(r.from)}&to=${encodeURIComponent(r.to)}`);

	function apply(e: Event) {
		e.preventDefault();
		router.get(`/app/leads/laporan?from=${from}&to=${to}`);
	}

	function pct(n: number): string {
		return `${Math.round(n * 100)}%`;
	}
	function duration(hours: number): string {
		if (!hours) return "-";
		if (hours < 48) return `${Math.round(hours)} jam`;
		return `${(hours / 24).toFixed(1)} hari`;
	}
	const field = "px-3 py-2 rounded-lg bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 text-sm text-neutral-900 dark:text-white";
	const card = "rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4";
</script>

<AppLayout user={props.user}>
	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-6 space-y-5">
		<div>
			<a href="/app/leads" use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400"><ArrowLeft class="w-4 h-4" /> Leads</a>
			<h1 class="mt-2 text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Laporan leads</h1>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Lead yang masuk {r.from} s.d. {r.to} (WIB).</p>
		</div>

		<form onsubmit={apply} class="flex flex-wrap items-end gap-3">
			<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Dari<input type="date" class="{field} mt-1 block" bind:value={from} /></label>
			<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Sampai<input type="date" class="{field} mt-1 block" bind:value={to} /></label>
			<button type="submit" class="px-4 py-2 rounded-lg bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold dark:bg-brand-500">Terapkan</button>
			<a href={exportUrl} class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg border border-neutral-300 dark:border-neutral-700 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800"><Download class="w-4 h-4" /> Ekspor CSV</a>
		</form>

		<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
			<div class={card}><p class="text-xs text-neutral-500">Lead masuk</p><p class="mt-1 text-2xl font-bold text-neutral-900 dark:text-white">{r.total_leads}</p></div>
			<div class={card}><p class="text-xs text-neutral-500">Jadi santri</p><p class="mt-1 text-2xl font-bold text-success">{r.won} <span class="text-sm font-medium text-neutral-500">({winRate}%)</span></p></div>
			<div class={card}><p class="text-xs text-neutral-500">Masih berjalan</p><p class="mt-1 text-2xl font-bold text-neutral-900 dark:text-white">{r.open}</p></div>
			<div class={card}><p class="text-xs text-neutral-500">Median sampai jadi santri</p><p class="mt-1 text-2xl font-bold text-neutral-900 dark:text-white">{duration(r.median_hours_to_won)}</p></div>
		</div>

		<section class={card}>
			<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Funnel</h2>
			<p class="text-xs text-neutral-500">Jumlah lead yang pernah mencapai tiap tahap, dan konversi dari tahap sebelumnya.</p>
			<ul class="mt-4 space-y-2.5">
				{#each r.funnel as f, i}
					<li class="grid grid-cols-[7.5rem_1fr_auto] sm:grid-cols-[10rem_1fr_6rem] items-center gap-3 text-sm">
						<span class="text-neutral-700 dark:text-neutral-300 truncate">{STAGE_LABELS[f.stage] ?? f.stage}</span>
						<div class="h-5 rounded bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
							<div class="h-full rounded {f.stage === 'won' ? 'bg-success' : 'bg-brand-500'}" style="width: {(f.reached / maxReached) * 100}%"></div>
						</div>
						<span class="text-right font-mono text-xs text-neutral-700 dark:text-neutral-300">{f.reached}{#if i > 0}<span class="text-neutral-500"> · {pct(f.conversion)}</span>{/if}</span>
					</li>
				{/each}
			</ul>
		</section>

		<div class="grid gap-4 lg:grid-cols-2">
			<section class={card}>
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Alasan hilang</h2>
				{#if r.lost_reasons.length === 0}<p class="mt-2 text-sm text-neutral-500">Belum ada lead hilang.</p>{/if}
				<ul class="mt-2 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
					{#each r.lost_reasons as x}<li class="flex justify-between py-1.5 text-sm"><span class="text-neutral-700 dark:text-neutral-300">{LOST_REASON_LABELS[x.key] ?? (x.key || "Tanpa alasan")}</span><span class="font-mono">{x.count}</span></li>{/each}
				</ul>
				<p class="mt-3 text-xs text-neutral-500">Jangan hubungi: {r.do_not_contact}</p>
			</section>
			<section class={card}>
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Keberatan utama (dugaan AI)</h2>
				{#if r.objections.length === 0}<p class="mt-2 text-sm text-neutral-500">Belum ada analisis AI.</p>{/if}
				<ul class="mt-2 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
					{#each r.objections as x}<li class="flex justify-between py-1.5 text-sm"><span class="text-neutral-700 dark:text-neutral-300">{x.key.replaceAll("_", " ")}</span><span class="font-mono">{x.count}</span></li>{/each}
				</ul>
			</section>
		</div>

		<div class="grid gap-4 lg:grid-cols-2">
			<section class={card}>
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Per minggu</h2>
				<div class="mt-2 overflow-x-auto">
					<table class="w-full text-sm">
						<thead><tr class="text-left text-xs text-neutral-500"><th class="py-1.5 font-medium">Minggu</th><th class="py-1.5 font-medium text-right">Lead</th><th class="py-1.5 font-medium text-right">Jadi santri</th><th class="py-1.5 font-medium text-right">Konversi</th></tr></thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
							{#each r.weekly as w}<tr><td class="py-1.5 font-mono text-xs">{w.week}</td><td class="py-1.5 text-right">{w.leads}</td><td class="py-1.5 text-right">{w.won}</td><td class="py-1.5 text-right">{w.leads ? Math.round((w.won / w.leads) * 100) : 0}%</td></tr>{/each}
						</tbody>
					</table>
				</div>
			</section>
			<section class={card}>
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Follow-up</h2>
				<dl class="mt-2 grid grid-cols-3 gap-3 text-sm">
					<div><dt class="text-xs text-neutral-500">Terkirim</dt><dd class="text-xl font-bold">{r.followup_sent}</dd></div>
					<div><dt class="text-xs text-neutral-500">Dibalas ≤ 72 jam</dt><dd class="text-xl font-bold">{r.followup_replied}</dd></div>
					<div><dt class="text-xs text-neutral-500">Lanjut jadi santri</dt><dd class="text-xl font-bold">{r.followup_won}</dd></div>
				</dl>
			</section>
		</div>
	</div>
</AppLayout>
