<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { ArrowUpRight } from "lucide-svelte";
	import { fmt, panel, pct, type Dashboard } from "./types";

	interface Props { d: Dashboard; canApproveFollowup: boolean }
	let { d, canApproveFollowup }: Props = $props();
	let rate = $derived(d.followup_eligible ? d.followup_replied / d.followup_eligible * 100 : 0);
</script>

<section class={panel + " p-5 sm:p-6"} aria-labelledby="followup-title">
	<div class="flex items-start justify-between gap-3">
		<div>
			<h2 id="followup-title" class="text-base font-semibold text-neutral-950 dark:text-white">Efektivitas follow-up</h2>
			<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Follow-up yang terkirim dalam periode.</p>
		</div>
		{#if canApproveFollowup}
			<a href="/app/leads/followups" use:inertia class="inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-brand-700 transition-colors hover:bg-brand-500/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Kelola <ArrowUpRight size={13} /></a>
		{/if}
	</div>

	<div class="mt-5">
		<div class="flex items-baseline justify-between gap-3">
			<span class="text-sm text-neutral-600 dark:text-neutral-300">Dibalas dalam 72 jam</span>
			<span class="text-2xl font-semibold tabular-nums tracking-tight text-neutral-950 dark:text-white">{pct(d.followup_replied, d.followup_eligible)}</span>
		</div>
		<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]" role="meter" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(rate)} aria-label="Rasio balasan follow-up">
			<span class="block h-full rounded-full bg-brand-500" style={`width:${rate}%`}></span>
		</div>
		<p class="mt-1.5 text-[11px] tabular-nums text-neutral-500 dark:text-neutral-400">{fmt(d.followup_replied)} dari {fmt(d.followup_eligible)} follow-up yang jendela 72 jamnya sudah lewat</p>
	</div>

	<dl class="mt-5 grid grid-cols-3 gap-2">
		<div class="rounded-lg bg-neutral-50 px-3 py-2.5 dark:bg-white/[0.03]">
			<dt class="text-[11px] text-neutral-500 dark:text-neutral-400">Terkirim</dt>
			<dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(d.report.followup_sent)}</dd>
		</div>
		<div class="rounded-lg bg-neutral-50 px-3 py-2.5 dark:bg-white/[0.03]">
			<dt class="text-[11px] text-neutral-500 dark:text-neutral-400">Jadi santri</dt>
			<dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(d.report.followup_won)}</dd>
		</div>
		<div class="rounded-lg bg-neutral-50 px-3 py-2.5 dark:bg-white/[0.03]">
			<dt class="text-[11px] text-neutral-500 dark:text-neutral-400">Disetujui, antre</dt>
			<dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(d.followup_approved)}</dd>
		</div>
	</dl>
</section>
