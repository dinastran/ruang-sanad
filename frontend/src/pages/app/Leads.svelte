<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import type { LeadLabel } from "@lib/leads";
	import { Activity, ChartColumn, Inbox, Kanban, LayoutDashboard, List, Megaphone } from "lucide-svelte";
	import { fmt } from "@components/lead-dashboard/types";
	import PipelineStrip from "@components/leads/PipelineStrip.svelte";
	import LeadFilters from "@components/leads/LeadFilters.svelte";
	import LeadTable from "@components/leads/LeadTable.svelte";
	import LeadKanban from "@components/leads/LeadKanban.svelte";
	import type { FilterChange, Filters, LeadItem, SourceOption } from "@components/leads/types";

	interface Props {
		user?: User;
		leads?: LeadItem[];
		next?: string;
		total?: number;
		counts?: Record<string, number>;
		labels?: LeadLabel[];
		sources?: SourceOption[];
		stages?: string[];
		filters?: Filters;
		flash?: Flash;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let labels = $derived(props.labels ?? []);
	let stages = $derived(props.stages ?? []);
	let filters = $derived<Filters>(props.filters ?? { status: "", stage: "", label: 0, mine: false, review: false, q: "", from: "", to: "", date_field: "" });
	let success = $derived(props.flash?.success);
	let error = $derived(props.flash?.error);

	// Items from earlier "Muat lagi" pages; the current page comes from props.
	let earlier = $state<LeadItem[]>([]);
	let loadingMore = $state(false);
	let items = $derived([...earlier, ...(props.leads ?? [])]);
	// Totals for the whole filter (all pages), not only the loaded items.
	let total = $derived(props.total ?? 0);
	let counts = $derived(props.counts ?? {});

	let hasFilters = $derived(
		(filters.status !== "" && filters.status !== "open") ||
			!!(filters.stage || filters.label || filters.mine || filters.review || filters.q || filters.from || filters.to || filters.date_field || filters.reached_stage || filters.lost_reason || filters.sumber),
	);
	let view = $state<"list" | "kanban">(readView());

	function readView(): "list" | "kanban" {
		try {
			return localStorage.getItem("leads-view") === "kanban" ? "kanban" : "list";
		} catch {
			return "list";
		}
	}
	function setView(v: "list" | "kanban") {
		view = v;
		try {
			localStorage.setItem("leads-view", v);
		} catch {
			/* storage unavailable */
		}
	}

	function go(changes: FilterChange) {
		const params = new URLSearchParams(window.location.search);
		params.delete("cursor");
		for (const [k, v] of Object.entries(changes)) {
			if (v === "" || v === 0 || v === false || v === undefined) params.delete(k);
			else params.set(k, String(v));
		}
		const qs = params.toString();
		router.get(`/app/leads${qs ? "?" + qs : ""}`);
	}

	function loadMore() {
		if (!props.next || loadingMore) return;
		const params = new URLSearchParams(window.location.search);
		params.set("cursor", props.next);
		const current = props.leads ?? [];
		loadingMore = true;
		router.get(`/app/leads?${params.toString()}`, {}, {
			preserveState: true,
			preserveScroll: true,
			replace: true,
			only: ["leads", "next"],
			onSuccess: () => {
				earlier = [...earlier, ...current];
				// Keep the cursor out of the address bar so refresh, back and
				// kanban moves return to the first page.
				const clean = new URL(window.location.href);
				clean.searchParams.delete("cursor");
				window.history.replaceState(window.history.state, "", clean.toString());
			},
			onFinish: () => {
				loadingMore = false;
			},
		});
	}

	function moveStage(lead: LeadItem, stage: string) {
		router.put(`/app/leads/${lead.id}/stage?back=list`, { stage }, {
			preserveScroll: true,
			onSuccess: (p) => {
				const err = (p.props.flash as Flash | undefined)?.error;
				if (err) Toast(err, "error");
			},
		});
	}

	const seg = "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
	const segOn = " bg-white text-neutral-900 shadow-[0_0_0_1px_rgba(15,23,42,.08),0_1px_2px_rgba(15,23,42,.08)] dark:bg-neutral-800 dark:text-white dark:shadow-[0_0_0_1px_rgba(255,255,255,.08)]";
	const segIdle = " text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white";
	const ghost = "inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-neutral-300 dark:hover:bg-white/[0.05]";
</script>

<AppLayout {user}>
	<div class="mx-auto max-w-7xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">CRM Leads</p>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Leads</h1>
				<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Calon santri dari chat CS, dari pertanyaan pertama sampai jadi santri.</p>
			</div>
			<div class="flex flex-wrap items-center gap-1">
				<a href="/app/leads/dashboard" use:inertia class={ghost}><LayoutDashboard size={15} /> Dashboard</a>
				<a href="/app/leads/laporan" use:inertia class={ghost}><ChartColumn size={15} /> Laporan</a>
				<a href="/app/leads/sumber" use:inertia class={ghost}><Megaphone size={15} /> Sumber</a>
				{#if user?.role === "super_admin"}<a href="/app/leads/meta-capi" use:inertia class={ghost}><Activity size={15} /> Meta CAPI</a>{/if}
				<div class="ml-1 hidden gap-0.5 rounded-lg bg-neutral-100 p-1 lg:inline-flex dark:bg-neutral-900" role="group" aria-label="Tampilan">
					<button type="button" onclick={() => setView("list")} aria-pressed={view === "list"} class={seg + (view === "list" ? segOn : segIdle)}><List size={15} /> Daftar</button>
					<button type="button" onclick={() => setView("kanban")} aria-pressed={view === "kanban"} class={seg + (view === "kanban" ? segOn : segIdle)}><Kanban size={15} /> Kanban</button>
				</div>
			</div>
		</header>

		{#if success}
			<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{success}</div>
		{/if}
		{#if error}
			<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{error}</div>
		{/if}

		<PipelineStrip {stages} {counts} {total} {filters} onfilter={go} />

		<LeadFilters {filters} {stages} {labels} sources={props.sources ?? []} onfilter={go} />

		<p class="text-sm text-neutral-600 dark:text-neutral-400" aria-live="polite">
			<strong class="font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(total)} lead</strong>
			{#if items.length < total}<span class="tabular-nums"> · {fmt(items.length)} ditampilkan</span>{/if}
			<span class="text-neutral-400 dark:text-neutral-500"> · urut aktivitas terbaru</span>
		</p>

		{#if items.length === 0}
			<div class="rounded-xl border border-dashed border-neutral-300 px-6 py-14 text-center dark:border-white/10">
				<Inbox size={28} class="mx-auto text-neutral-400" />
				<p class="mt-3 text-sm font-medium text-neutral-700 dark:text-neutral-300">Belum ada lead untuk filter ini</p>
				<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Lead baru muncul otomatis saat ada chat masuk ke nomor CS.</p>
				{#if hasFilters}<a href="/app/leads" use:inertia class="mt-4 inline-block text-sm font-medium text-brand-700 hover:underline dark:text-brand-400">Reset filter</a>{/if}
			</div>
		{:else if view === "kanban"}
			<!-- Kanban is desktop-only; on small screens the list is shown instead. -->
			<div class="lg:hidden"><LeadTable {items} /></div>
			<div class="hidden space-y-2 lg:block"><LeadKanban {items} {stages} {counts} onmove={moveStage} /></div>
		{:else}
			<LeadTable {items} />
		{/if}

		{#if props.next}
			<div class="flex justify-center pt-2">
				<button type="button" onclick={loadMore} disabled={loadingMore} class="rounded-md border border-neutral-200 px-4 py-2 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]">
					{loadingMore ? "Memuat…" : `Muat lagi · ${fmt(total - items.length)} tersisa`}
				</button>
			</div>
		{/if}
	</div>
</AppLayout>
