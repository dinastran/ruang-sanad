<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { formatDateTime, relativeTime } from "@lib/leads";
	import { Ban, Check, CheckCheck, ChevronDown, CircleAlert, Clock, Hourglass, RotateCcw, SkipForward, Sparkles, UserRound } from "lucide-svelte";

	interface HistoryItem {
		id: number;
		lead_id: number;
		lead_name: string;
		step: string;
		status: string;
		scheduled_at: string;
		draft_text: string;
		created_by_kind: string;
		note: string;
		attempts: number;
		outbox_id: number;
		sent_at: string;
		at: string;
	}
	interface Filter {
		status: string;
		step: string;
		from: string;
		to: string;
	}
	interface History {
		items: HistoryItem[];
		counts: Record<string, number>;
		filter: Filter;
		page: number;
		has_more: boolean;
	}
	interface TimelineEntry {
		type: string;
		actor: string;
		at: string;
		detail: string;
	}

	let { history, canApprove = false, busy = false, onRetry }: { history: History; canApprove?: boolean; busy?: boolean; onRetry?: (outboxID: number) => void } = $props();

	// Status strip order = the order staff check things: outcome first.
	const STATUSES: { key: string; label: string; tone: string }[] = [
		{ key: "sent", label: "Terkirim", tone: "text-success" },
		{ key: "failed", label: "Gagal", tone: "text-error" },
		{ key: "approved", label: "Terjadwal", tone: "text-brand-700 dark:text-brand-300" },
		{ key: "pending_approval", label: "Menunggu", tone: "text-warning" },
		{ key: "skipped", label: "Dilewati", tone: "text-neutral-600 dark:text-neutral-400" },
		{ key: "cancelled", label: "Dibatalkan", tone: "text-neutral-600 dark:text-neutral-400" },
	];
	const STEPS = ["d1", "d3", "d7", "d14", "d30", "manual"];
	const statusMeta = (s: string) => STATUSES.find((x) => x.key === s) ?? { key: s, label: s, tone: "text-neutral-600" };
	let total = $derived(Object.values(history.counts).reduce((a, b) => a + b, 0));

	function go(patch: Partial<Filter & { page: number }>) {
		const f = { ...history.filter, page: 1, ...patch } as Record<string, string | number>;
		const q: Record<string, string | number> = { tab: "riwayat" };
		for (const k of ["status", "step", "from", "to"]) if (f[k]) q[k] = f[k];
		if (Number(f.page) > 1) q.page = f.page;
		router.get("/app/leads/followups", q, { preserveState: true, preserveScroll: true, only: ["history", "tab"] });
	}

	// Timeline per follow-up, fetched once on first expand.
	let open = $state<Record<number, boolean>>({});
	let trails = $state<Record<number, TimelineEntry[] | "loading" | "error">>({});
	async function toggle(it: HistoryItem) {
		open[it.id] = !open[it.id];
		if (!open[it.id] || (trails[it.id] && trails[it.id] !== "error")) return;
		trails[it.id] = "loading";
		try {
			const res = await fetch(`/app/leads/followups/${it.id}/timeline`, { headers: { Accept: "application/json" } });
			if (!res.ok) throw new Error();
			trails[it.id] = (await res.json()).entries as TimelineEntry[];
		} catch {
			trails[it.id] = "error";
		}
	}

	const EVENT_LABELS: Record<string, string> = {
		"followup.proposed": "Diusulkan",
		"followup.approved": "Disetujui",
		"followup.rescheduled": "Jadwal diubah",
		"outbox.failed": "Gagal kirim",
		"outbox.retried": "Dicoba lagi",
		"followup.sent": "Terkirim",
		"followup.skipped": "Dilewati sistem",
		"followup.cancelled": "Dibatalkan",
	};
	const markerTone = (type: string) =>
		type === "followup.sent" ? "bg-success text-white"
		: type === "outbox.failed" ? "bg-error text-white"
		: type === "followup.cancelled" || type === "followup.skipped" ? "bg-neutral-400 text-white dark:bg-neutral-600"
		: "bg-white text-neutral-500 ring-1 ring-neutral-300 dark:bg-neutral-900 dark:text-neutral-400 dark:ring-neutral-700";

	const field = "px-3 py-2 rounded-lg bg-white dark:bg-neutral-900/60 border border-neutral-300 dark:border-neutral-700/80 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-400/30";
	const card = "rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50";
</script>

<!-- Status strip: counts double as the status filter (the focal point). -->
<section class="{card} p-1.5" aria-label="Ringkasan status">
	<div class="grid grid-cols-4 sm:grid-cols-7 gap-1">
		<button type="button" onclick={() => go({ status: "" })}
			class="rounded-lg px-3 py-2 text-left transition-colors {history.filter.status === '' ? 'bg-neutral-100 dark:bg-neutral-800' : 'hover:bg-neutral-50 dark:hover:bg-neutral-800/50'}">
			<span class="block text-[11px] font-medium uppercase tracking-wide text-neutral-500">Semua</span>
			<span class="block text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{total}</span>
		</button>
		{#each STATUSES as s (s.key)}
			<button type="button" onclick={() => go({ status: history.filter.status === s.key ? "" : s.key })} aria-pressed={history.filter.status === s.key}
				class="rounded-lg px-3 py-2 text-left transition-colors {history.filter.status === s.key ? 'bg-neutral-100 dark:bg-neutral-800' : 'hover:bg-neutral-50 dark:hover:bg-neutral-800/50'}">
				<span class="block text-[11px] font-medium uppercase tracking-wide text-neutral-500">{s.label}</span>
				<span class="block text-xl font-semibold tabular-nums {(history.counts[s.key] ?? 0) > 0 ? s.tone : 'text-neutral-300 dark:text-neutral-700'}">{history.counts[s.key] ?? 0}</span>
			</button>
		{/each}
	</div>
</section>

<div class="flex flex-wrap items-end gap-2">
	<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Step
		<select class="{field} mt-1 block" value={history.filter.step} onchange={(e) => go({ step: (e.currentTarget as HTMLSelectElement).value })}>
			<option value="">Semua step</option>
			{#each STEPS as st}<option value={st}>{st.toUpperCase()}</option>{/each}
		</select>
	</label>
	<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Dari
		<input type="date" class="{field} mt-1 block" value={history.filter.from} onchange={(e) => go({ from: (e.currentTarget as HTMLInputElement).value })} />
	</label>
	<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Sampai
		<input type="date" class="{field} mt-1 block" value={history.filter.to} onchange={(e) => go({ to: (e.currentTarget as HTMLInputElement).value })} />
	</label>
	{#if history.filter.status || history.filter.step || history.filter.from || history.filter.to}
		<button type="button" class="px-3 py-2 text-sm font-medium text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white" onclick={() => go({ status: "", step: "", from: "", to: "" })}>Reset filter</button>
	{/if}
</div>

<section class={card}>
	{#if history.items.length === 0}
		<p class="p-6 text-center text-sm text-neutral-500">Tidak ada follow-up untuk filter ini.</p>
	{:else}
		<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
			{#each history.items as it (it.id)}
				{@const meta = statusMeta(it.status)}
				<li>
					<button type="button" class="w-full px-4 py-3 text-left hover:bg-neutral-50 dark:hover:bg-white/[0.02] transition-colors" aria-expanded={!!open[it.id]} onclick={() => toggle(it)}>
						<div class="flex items-start gap-3">
							<span class="mt-0.5 shrink-0 {meta.tone}" aria-hidden="true">
								{#if it.status === "sent"}<CheckCheck class="w-4 h-4" />
								{:else if it.status === "failed"}<CircleAlert class="w-4 h-4" />
								{:else if it.status === "approved"}<Clock class="w-4 h-4" />
								{:else if it.status === "pending_approval"}<Hourglass class="w-4 h-4" />
								{:else if it.status === "skipped"}<SkipForward class="w-4 h-4" />
								{:else}<Ban class="w-4 h-4" />{/if}
							</span>
							<span class="min-w-0 flex-1">
								<span class="flex flex-wrap items-baseline gap-x-2">
									<span class="font-medium text-sm text-neutral-900 dark:text-white">{it.lead_name}</span>
									<span class="text-xs font-semibold uppercase text-neutral-500">{it.step}</span>
									<span class="text-xs font-medium {meta.tone}">{meta.label}</span>
								</span>
								<span class="mt-0.5 block text-xs text-neutral-500 tabular-nums">
									{it.status === "sent" ? `Terkirim ${formatDateTime(it.sent_at)}` : it.status === "approved" || it.status === "pending_approval" ? `Jadwal ${formatDateTime(it.scheduled_at)}` : relativeTime(it.at)}
									{#if it.attempts > 0 && it.status !== "sent"} · {it.attempts}× percobaan{/if}
								</span>
								{#if it.note && it.status !== "sent"}
									<span class="mt-1 block text-xs {it.status === 'failed' ? 'text-error' : 'text-neutral-600 dark:text-neutral-400'}">{it.note}</span>
								{/if}
							</span>
							<ChevronDown class="w-4 h-4 mt-0.5 shrink-0 text-neutral-400 transition-transform duration-150 {open[it.id] ? 'rotate-180' : ''}" />
						</div>
					</button>

					{#if open[it.id]}
						<div class="px-4 pb-4 sm:pl-11 space-y-3">
							<p class="rounded-lg bg-neutral-50 dark:bg-neutral-900/60 p-3 text-sm text-neutral-700 dark:text-neutral-300 whitespace-pre-wrap break-words">{it.draft_text}</p>

							{#if trails[it.id] === "loading"}
								<p class="text-xs text-neutral-500">Memuat jejak…</p>
							{:else if trails[it.id] === "error"}
								<p class="text-xs text-error">Jejak gagal dimuat. Tutup dan buka lagi untuk mencoba.</p>
							{:else if Array.isArray(trails[it.id])}
								{@const trail = trails[it.id] as TimelineEntry[]}
								{#if trail.length === 0}
									<p class="text-xs text-neutral-500">Belum ada jejak tercatat.</p>
								{:else}
									<ol class="relative space-y-3 before:absolute before:left-[9px] before:top-2 before:bottom-2 before:w-px before:bg-neutral-200 dark:before:bg-neutral-800">
										{#each trail as e, i (i)}
											<li class="relative flex gap-3">
												<span class="relative z-10 mt-0.5 grid h-[19px] w-[19px] shrink-0 place-items-center rounded-full {markerTone(e.type)}">
													{#if e.type === "followup.sent"}<CheckCheck class="w-3 h-3" />
													{:else if e.type === "outbox.failed"}<CircleAlert class="w-3 h-3" />
													{:else if e.type === "outbox.retried"}<RotateCcw class="w-3 h-3" />
													{:else if e.type === "followup.proposed" && e.actor === "AI"}<Sparkles class="w-3 h-3" />
													{:else if e.type === "followup.proposed"}<UserRound class="w-3 h-3" />
													{:else}<Check class="w-3 h-3" />{/if}
												</span>
												<div class="min-w-0 text-xs">
													<p><span class="font-semibold text-neutral-900 dark:text-white">{EVENT_LABELS[e.type] ?? e.type}</span> <span class="text-neutral-500">oleh {e.actor} · <span class="tabular-nums">{formatDateTime(e.at)}</span></span></p>
													{#if e.detail}<p class="mt-0.5 break-words {e.type === 'outbox.failed' ? 'text-error' : 'text-neutral-600 dark:text-neutral-400'}">{e.detail}</p>{/if}
												</div>
											</li>
										{/each}
									</ol>
								{/if}
							{/if}

							<div class="flex flex-wrap gap-2">
								<a href={`/app/leads/${it.lead_id}`} use:inertia class="text-xs font-medium text-brand-600 hover:text-brand-700 dark:text-brand-400">Buka lead →</a>
								{#if canApprove && it.status === "failed" && it.outbox_id && onRetry}
									<button type="button" class="inline-flex items-center gap-1 text-xs font-medium text-neutral-700 hover:text-neutral-900 dark:text-neutral-300 dark:hover:text-white disabled:opacity-50" disabled={busy} onclick={() => onRetry(it.outbox_id)}><RotateCcw class="w-3.5 h-3.5" /> Coba lagi</button>
								{/if}
							</div>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>

{#if history.page > 1 || history.has_more}
	<div class="flex items-center justify-between text-sm">
		<button type="button" class="px-3 py-2 rounded-lg font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-40" disabled={history.page <= 1} onclick={() => go({ page: history.page - 1 })}>← Sebelumnya</button>
		<span class="text-xs text-neutral-500 tabular-nums">Halaman {history.page}</span>
		<button type="button" class="px-3 py-2 rounded-lg font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-40" disabled={!history.has_more} onclick={() => go({ page: history.page + 1 })}>Berikutnya →</button>
	</div>
{/if}
