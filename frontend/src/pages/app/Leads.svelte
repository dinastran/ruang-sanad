<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import {
		STAGE_LABELS,
		STATUS_LABELS,
		stageTone,
		leadName,
		relativeTime,
		labelTone,
		type LeadLabel,
	} from "@lib/leads";
	import { Search, List, Kanban, AlertTriangle, PauseCircle, ChartColumn, Inbox } from "lucide-svelte";

	interface LeadItem {
		id: number;
		nama: string;
		wa_name: string;
		phone: string;
		stage: string;
		status: string;
		paket_kode: string;
		needs_review: boolean;
		ai_paused: boolean;
		assigned_user_id: number;
		assigned_name: string;
		last_inbound_at: string;
		last_activity_at: string;
		labels: LeadLabel[];
	}

	interface Filters {
		status: string;
		stage: string;
		label: number;
		mine: boolean;
		review: boolean;
		q: string;
		from: string;
		to: string;
	}

	interface Props {
		user?: User;
		leads?: LeadItem[];
		next?: string;
		labels?: LeadLabel[];
		stages?: string[];
		filters?: Filters;
		flash?: Flash;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let labels = $derived(props.labels ?? []);
	let stages = $derived(props.stages ?? []);
	let filters = $derived<Filters>(props.filters ?? { status: "", stage: "", label: 0, mine: false, review: false, q: "", from: "", to: "" });
	let success = $derived(props.flash?.success);
	let error = $derived(props.flash?.error);

	// Items from earlier "Muat lagi" pages; the current page comes from props.
	let earlier = $state<LeadItem[]>([]);
	let loadingMore = $state(false);
	let items = $derived([...earlier, ...(props.leads ?? [])]);

	// svelte-ignore state_referenced_locally
	let search = $state(props.filters?.q ?? "");
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

	function go(changes: Partial<Record<keyof Filters, string | number | boolean>>) {
		const params = new URLSearchParams(window.location.search);
		params.delete("cursor");
		for (const [k, v] of Object.entries(changes)) {
			if (v === "" || v === 0 || v === false || v === undefined) params.delete(k);
			else params.set(k, String(v));
		}
		const qs = params.toString();
		router.get(`/app/leads${qs ? "?" + qs : ""}`);
	}

	function doSearch(e: Event) {
		e.preventDefault();
		go({ q: search.trim() });
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

	let quick = $derived(
		filters.mine ? "mine" : filters.stage === "bukti_masuk" ? "bukti" : filters.review ? "review" : filters.status === "" || filters.status === "open" ? "open" : "",
	);

	// Kanban: drag a card to another stage column (manual stage move).
	let dragId = $state<number | null>(null);
	let dropStage = $state<string | null>(null);
	let kanbanStages = $derived(stages);
	// Terverifikasi is shown (leads waiting for "Jadikan Santri") but only the
	// verify action can move a lead there.
	const canDropOn = (stage: string) => stage !== "terverifikasi";

	function dropOn(stage: string) {
		const id = dragId;
		dragId = null;
		dropStage = null;
		const lead = items.find((l) => l.id === id);
		if (!lead || lead.stage === stage) return;
		router.put(`/app/leads/${lead.id}/stage?back=list`, { stage }, {
			preserveScroll: true,
			onSuccess: (p) => {
				const err = (p.props.flash as Flash | undefined)?.error;
				if (err) Toast(err, "error");
			},
		});
	}

	const fieldClass =
		"w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-200/80 dark:border-white/[0.06] text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-400/40";
</script>

<AppLayout {user}>
	<div class="pt-8 pb-6 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-7xl mx-auto px-4 sm:px-6">
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div class="min-w-0">
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Leads</h1>
					<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Calon santri dari chat CS, dari pertanyaan pertama sampai jadi santri.</p>
				</div>
				<div class="flex items-center gap-2">
					<a href="/app/leads/laporan" use:inertia class="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">
						<ChartColumn class="w-4 h-4" /> Laporan
					</a>
					<div class="hidden lg:inline-flex rounded-lg border border-neutral-200/80 dark:border-white/[0.06] p-0.5" role="group" aria-label="Tampilan">
						<button type="button" onclick={() => setView("list")} aria-pressed={view === "list"} class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm {view === 'list' ? 'bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'text-neutral-600 dark:text-neutral-400'}">
							<List class="w-4 h-4" /> Daftar
						</button>
						<button type="button" onclick={() => setView("kanban")} aria-pressed={view === "kanban"} class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm {view === 'kanban' ? 'bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'text-neutral-600 dark:text-neutral-400'}">
							<Kanban class="w-4 h-4" /> Kanban
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div class="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-4">
		{#if success}
			<div class="rounded-xl border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-success">{success}</div>
		{/if}
		{#if error}
			<div class="rounded-xl border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-error">{error}</div>
		{/if}

		<form onsubmit={doSearch} class="relative">
			<Search class="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-neutral-500 pointer-events-none" />
			<input type="search" bind:value={search} placeholder="Cari nama, nama WA, atau nomor" class="{fieldClass} pl-10" aria-label="Cari lead" />
		</form>

		<div class="flex gap-2 overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0">
			{#each [
				{ key: "open", label: "Semua berjalan", apply: { status: "", stage: "", mine: false, review: false } },
				{ key: "mine", label: "Lead saya", apply: { mine: true, stage: "", review: false } },
				{ key: "bukti", label: "Bukti masuk", apply: { stage: "bukti_masuk", mine: false, review: false } },
				{ key: "review", label: "Perlu dicek", apply: { review: true, stage: "", mine: false } },
			] as chip}
				<button type="button" onclick={() => go(chip.apply)} class="shrink-0 px-3 py-1.5 rounded-full border text-sm font-medium whitespace-nowrap transition-colors {quick === chip.key ? 'border-brand-500 bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'border-neutral-200/80 dark:border-white/[0.08] text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800'}">
					{chip.label}
				</button>
			{/each}
		</div>

		<details class="rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50">
			<summary class="cursor-pointer px-4 py-3 text-sm font-medium text-neutral-700 dark:text-neutral-300">Filter lainnya</summary>
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3 px-4 pb-4">
				<label class="block text-xs font-semibold uppercase tracking-wider text-neutral-500">
					Status
					<select class="{fieldClass} mt-1" value={filters.status || "open"} onchange={(e) => go({ status: (e.currentTarget as HTMLSelectElement).value === "open" ? "" : (e.currentTarget as HTMLSelectElement).value })}>
						{#each ["open", "won", "lost", "do_not_contact", "all"] as s}
							<option value={s}>{s === "all" ? "Semua status" : STATUS_LABELS[s]}</option>
						{/each}
					</select>
				</label>
				<label class="block text-xs font-semibold uppercase tracking-wider text-neutral-500">
					Tahap
					<select class="{fieldClass} mt-1" value={filters.stage} onchange={(e) => go({ stage: (e.currentTarget as HTMLSelectElement).value })}>
						<option value="">Semua tahap</option>
						{#each stages as s}
							<option value={s}>{STAGE_LABELS[s] ?? s}</option>
						{/each}
					</select>
				</label>
				<label class="block text-xs font-semibold uppercase tracking-wider text-neutral-500">
					Label
					<select class="{fieldClass} mt-1" value={String(filters.label || "")} onchange={(e) => go({ label: Number((e.currentTarget as HTMLSelectElement).value) || 0 })}>
						<option value="">Semua label</option>
						{#each labels as l}
							<option value={String(l.id)}>{l.nama}</option>
						{/each}
					</select>
				</label>
				<label class="block text-xs font-semibold uppercase tracking-wider text-neutral-500">
					Masuk dari
					<input type="date" class="{fieldClass} mt-1" value={filters.from} onchange={(e) => go({ from: (e.currentTarget as HTMLInputElement).value })} />
				</label>
				<label class="block text-xs font-semibold uppercase tracking-wider text-neutral-500">
					Sampai
					<input type="date" class="{fieldClass} mt-1" value={filters.to} onchange={(e) => go({ to: (e.currentTarget as HTMLInputElement).value })} />
				</label>
			</div>
		</details>

		{#if items.length === 0}
			<div class="rounded-xl border border-dashed border-neutral-300 dark:border-neutral-700 px-6 py-14 text-center">
				<Inbox class="w-8 h-8 mx-auto text-neutral-400" />
				<p class="mt-3 text-sm font-medium text-neutral-700 dark:text-neutral-300">Belum ada lead untuk filter ini</p>
				<p class="mt-1 text-xs text-neutral-500">Lead baru muncul otomatis saat ada chat masuk ke nomor CS.</p>
			</div>
		{:else if view === "kanban"}
			<!-- Kanban is desktop-only; on small screens the list is shown instead. -->
			<div class="lg:hidden">{@render cardList()}</div>
			<div class="hidden lg:flex gap-3 overflow-x-auto pb-2">
				{#each kanbanStages as stage}
					{@const col = items.filter((l) => l.stage === stage && l.status === "open")}
					<section
						class="w-64 shrink-0 rounded-xl border bg-neutral-50 dark:bg-neutral-900/40 {dropStage === stage ? 'border-brand-500' : 'border-neutral-200/80 dark:border-white/[0.06]'}"
						aria-label={STAGE_LABELS[stage]}
						ondragover={(e) => { if (!canDropOn(stage)) return; e.preventDefault(); dropStage = stage; }}
						ondragleave={() => { if (dropStage === stage) dropStage = null; }}
						ondrop={(e) => { e.preventDefault(); if (canDropOn(stage)) dropOn(stage); }}
					>
						<header class="flex items-center justify-between px-3 py-2.5 border-b border-neutral-200/80 dark:border-white/[0.06]">
							<h2 class="text-xs font-semibold uppercase tracking-wider text-neutral-600 dark:text-neutral-400">{STAGE_LABELS[stage]}</h2>
							<span class="text-xs font-mono text-neutral-500">{col.length}</span>
						</header>
						<div class="p-2 space-y-2 min-h-24">
							{#each col as lead (lead.id)}
								<a
									href="/app/leads/{lead.id}"
									use:inertia
									draggable="true"
									ondragstart={() => (dragId = lead.id)}
									ondragend={() => { dragId = null; dropStage = null; }}
									class="block rounded-lg border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925 p-3 hover:border-brand-400/50 {dragId === lead.id ? 'opacity-50' : ''}"
								>
									<p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{leadName(lead)}</p>
									<p class="mt-0.5 text-xs text-neutral-500">{relativeTime(lead.last_activity_at)}{lead.paket_kode ? ` · ${lead.paket_kode}` : ""}</p>
									{@render flags(lead)}
								</a>
							{/each}
						</div>
					</section>
				{/each}
			</div>
			<p class="hidden lg:block text-xs text-neutral-500">Seret kartu ke kolom lain untuk mengubah tahap. Tahap Terverifikasi hanya lewat tombol Verifikasi pembayaran di halaman lead.</p>
		{:else}
			{@render cardList()}
		{/if}

		{#if props.next}
			<div class="flex justify-center pt-2">
				<button type="button" onclick={loadMore} disabled={loadingMore} class="px-4 py-2 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-50">
					{loadingMore ? "Memuat..." : "Muat lagi"}
				</button>
			</div>
		{/if}
	</div>
</AppLayout>

{#snippet flags(lead: LeadItem)}
	{#if lead.needs_review || lead.ai_paused || lead.labels.length > 0}
		<div class="mt-2 flex flex-wrap gap-1">
			{#if lead.needs_review}
				<span class="inline-flex items-center gap-1 rounded-full bg-warning/10 px-2 py-0.5 text-[11px] font-medium text-warning"><AlertTriangle class="w-3 h-3" /> Perlu dicek</span>
			{/if}
			{#if lead.ai_paused}
				<span class="inline-flex items-center gap-1 rounded-full bg-info/10 px-2 py-0.5 text-[11px] font-medium text-info"><PauseCircle class="w-3 h-3" /> AI dijeda</span>
			{/if}
			{#each lead.labels as l}
				<span class="rounded-full border px-2 py-0.5 text-[11px] {labelTone(l.warna)}">{l.nama}</span>
			{/each}
		</div>
	{/if}
{/snippet}

{#snippet cardList()}
	<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.05] rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50">
		{#each items as lead (lead.id)}
			<li>
				<a href="/app/leads/{lead.id}" use:inertia class="flex items-start gap-3 px-4 py-3.5 hover:bg-neutral-50 dark:hover:bg-white/[0.02]">
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-2">
							<p class="font-medium text-neutral-900 dark:text-white truncate">{leadName(lead)}</p>
							{#if lead.nama && lead.wa_name && lead.nama !== lead.wa_name}
								<span class="hidden sm:inline text-xs text-neutral-500 truncate">({lead.wa_name})</span>
							{/if}
						</div>
						<div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-neutral-500">
							<span class="rounded-full px-2 py-0.5 font-medium {stageTone(lead.stage, lead.status)}">{lead.status === "open" ? STAGE_LABELS[lead.stage] : STATUS_LABELS[lead.status]}</span>
							{#if lead.paket_kode}<span>Paket {lead.paket_kode}</span>{/if}
							{#if lead.assigned_name}<span>· {lead.assigned_name}</span>{/if}
						</div>
						{@render flags(lead)}
					</div>
					<span class="shrink-0 text-xs text-neutral-500 whitespace-nowrap">{relativeTime(lead.last_activity_at)}</span>
				</a>
			</li>
		{/each}
	</ul>
{/snippet}
