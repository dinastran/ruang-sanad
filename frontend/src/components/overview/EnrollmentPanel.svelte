<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { ArrowDownRight, ArrowUpRight } from "lucide-svelte";
	import OverviewChart from "@components/OverviewChart.svelte";
	import type { Overview } from "@lib/overview";
	import { change, count, focusRing, kicker, monthShort, panel, signed } from "./ui";

	let { growth, leads, leadsHref, currentMonth }: {
		growth: Overview["growth"];
		leads: Overview["leads"];
		leadsHref: string;
		currentMonth: boolean;
	} = $props();

	let stats = $derived(
		leads
			? [
					{ label: "Lead WhatsApp masuk", value: leads.masuk, previous: leads.masuk_sebelumnya },
					{ label: "Closing jadi santri", value: leads.closing, previous: leads.closing_sebelumnya },
				]
			: [],
	);
</script>

<section class={panel + " flex min-w-0 flex-col"} aria-labelledby="pendaftaran-title">
	<div class="flex items-start justify-between gap-3 px-5 pt-5 sm:px-6">
		<div>
			<h2 id="pendaftaran-title" class={kicker}>Pendaftaran santri</h2>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">Santri terdaftar per bulan, enam bulan terakhir.</p>
		</div>
		<a href="/app/analitik-santri" use:inertia class={"shrink-0 rounded text-xs font-medium text-brand-700 hover:underline dark:text-brand-400 " + focusRing}>Analitik</a>
	</div>
	<div class="px-5 pb-4 sm:px-6">
		<OverviewChart
			title="Jumlah pendaftaran santri per bulan"
			labels={growth.map((g) => monthShort(g.month))}
			series={[{ name: "Santri terdaftar", values: growth.map((g) => g.total) }]}
		/>
	</div>

	<div class="mt-auto border-t border-neutral-100 bg-neutral-50/60 dark:border-white/[0.05] dark:bg-white/[0.015]">
		{#if leads}
			<a href={leadsHref} use:inertia class={"grid grid-cols-2 divide-x divide-neutral-100 transition-colors duration-150 hover:bg-neutral-100/60 dark:divide-white/[0.05] dark:hover:bg-white/[0.03] " + focusRing + " focus-visible:ring-inset"} aria-label="Buka dashboard CRM leads bulan ini">
				{#each stats as s (s.label)}
					{@const delta = change(s.value, s.previous)}
					<span class="block min-w-0 px-5 py-3 sm:px-6">
						<span class="block text-xs text-neutral-500 dark:text-neutral-400">{s.label}</span>
						<span class="mt-0.5 flex flex-wrap items-baseline gap-x-2">
							<span class="text-lg font-semibold tabular-nums text-neutral-950 dark:text-white">{count(s.value)}</span>
							{#if delta !== null}
								<span class={"inline-flex items-center gap-0.5 text-xs font-medium tabular-nums " + (delta > 0 ? "text-emerald-700 dark:text-emerald-400" : delta < 0 ? "text-red-700 dark:text-red-400" : "text-neutral-500")}>
									{#if delta > 0}<ArrowUpRight size={12} />{:else if delta < 0}<ArrowDownRight size={12} />{/if}{signed(delta)}
								</span>
							{:else}
								<span class="text-xs text-neutral-500 dark:text-neutral-400">bulan lalu {count(s.previous)}</span>
							{/if}
						</span>
					</span>
				{/each}
			</a>
			<p class="px-5 pb-3 text-[11px] text-neutral-500 sm:px-6 dark:text-neutral-400">
				Dibanding bulan sebelumnya{currentMonth ? " (bulan berjalan belum selesai)" : ""}. Sumber: CRM leads WhatsApp.
			</p>
		{:else}
			<p class="px-5 py-3 text-xs text-neutral-500 sm:px-6 dark:text-neutral-400">Data lead WhatsApp belum tersedia.</p>
		{/if}
	</div>
</section>
