<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { STAGE_LABELS, relativeTime } from "@lib/leads";
	import { ChevronRight, Inbox } from "lucide-svelte";
	import { fmt, panel, type Dashboard } from "./types";

	interface Props { d: Dashboard; canApproveFollowup: boolean; onreview: () => void }
	let { d, canApproveFollowup, onreview }: Props = $props();

	// Reason text comes from the service; map it to a tone so the queue scans by kind.
	function tone(reason: string) {
		if (reason.startsWith("Bukti")) return { dot: "bg-warning", text: "text-amber-700 dark:text-amber-400" };
		if (reason.startsWith("Terverifikasi")) return { dot: "bg-success", text: "text-emerald-700 dark:text-emerald-400" };
		if (reason.startsWith("Data")) return { dot: "bg-info", text: "text-blue-700 dark:text-blue-400" };
		return { dot: "bg-brand-500", text: "text-brand-700 dark:text-brand-400" };
	}
	let counters = $derived([
		{ label: "Perlu ditinjau", hint: "lead periode ini", value: d.review_count, warn: d.review_count > 0, href: "", go: onreview },
		{ label: "Menunggu approval", hint: "follow-up", value: d.followup_pending, warn: d.followup_pending > 0, href: canApproveFollowup ? "/app/leads/followups" : "", go: null },
		{ label: "Gagal kirim", hint: "follow-up", value: d.followup_failed, warn: false, danger: d.followup_failed > 0, href: canApproveFollowup ? "/app/leads/followups" : "", go: null },
	]);
	const row = "flex items-center justify-between gap-3 rounded-lg px-3 py-2.5 text-left transition-colors duration-150";
	const rowLink = " hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.03]";
</script>

<section class={panel + " flex flex-col lg:absolute lg:inset-0"} aria-labelledby="tindakan-title">
	<div class="px-5 pt-5">
		<h2 id="tindakan-title" class="text-base font-semibold text-neutral-950 dark:text-white">Perlu tindakan</h2>
		<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Antrean CS saat ini, tidak terikat filter periode kecuali yang ditandai.</p>
	</div>

	<div class="mt-3 space-y-0.5 px-2">
		{#each counters as c}
			{#snippet body()}
				<span class="min-w-0">
					<span class="block text-sm font-medium text-neutral-800 dark:text-neutral-200">{c.label}</span>
					<span class="block text-[11px] text-neutral-500 dark:text-neutral-400">{c.hint}</span>
				</span>
				<span class={"shrink-0 rounded-md px-2 py-0.5 text-sm font-semibold tabular-nums " + (c.danger ? "bg-error/10 text-red-700 dark:text-red-400" : c.warn ? "bg-warning/15 text-amber-800 dark:text-amber-300" : "text-neutral-400 dark:text-neutral-500")}>{fmt(c.value)}</span>
			{/snippet}
			{#if c.go}
				<button type="button" onclick={c.go} class={row + rowLink + " w-full"}>{@render body()}</button>
			{:else if c.href}
				<a href={c.href} use:inertia class={row + rowLink}>{@render body()}</a>
			{:else}
				<div class={row}>{@render body()}</div>
			{/if}
		{/each}
	</div>

	<div class="mx-5 mt-3 flex items-baseline justify-between border-t border-neutral-100 pt-4 dark:border-white/[0.05]">
		<h3 class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Prioritas lead aktif</h3>
		<span class="text-[11px] tabular-nums text-neutral-400 dark:text-neutral-500">{d.priorities.length} teratas</span>
	</div>
	{#if d.priorities.length === 0}
		<div class="flex flex-1 flex-col items-center justify-center gap-2 px-5 py-10 text-center">
			<Inbox size={20} class="text-neutral-300 dark:text-neutral-600" />
			<p class="text-sm text-neutral-500 dark:text-neutral-400">Tidak ada lead yang menunggu tindakan.</p>
		</div>
	{:else}
		<ul class="mt-1 max-h-[22rem] min-h-0 flex-1 overflow-y-auto px-2 pb-2 lg:max-h-none lg:[mask-image:linear-gradient(to_bottom,black_calc(100%-2rem),transparent)]">
			{#each d.priorities as p}
				{@const t = tone(p.reason)}
				<li>
					<a href={"/app/leads/" + p.id} use:inertia class="group flex items-center gap-3 rounded-lg px-3 py-2 transition-colors duration-150 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.03]">
						<span class={"h-1.5 w-1.5 shrink-0 rounded-full " + t.dot} aria-hidden="true"></span>
						<span class="min-w-0 flex-1">
							<span class="block truncate text-sm font-medium text-neutral-900 dark:text-neutral-100">{p.name || "Tanpa nama"}</span>
							<span class="block truncate text-xs"><span class={t.text}>{p.reason}</span><span class="text-neutral-400 dark:text-neutral-500">{" · "}{STAGE_LABELS[p.stage] || p.stage}</span></span>
						</span>
						<span class="shrink-0 text-[11px] tabular-nums text-neutral-400 dark:text-neutral-500">{relativeTime(p.activity_at)}</span>
						<ChevronRight size={14} class="shrink-0 text-neutral-300 transition-transform duration-150 group-hover:translate-x-0.5 dark:text-neutral-600" />
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</section>
