<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { STAGE_LABELS, STATUS_LABELS, LOST_REASON_LABELS, type LeadLabel } from "@lib/leads";
	import { Search, SlidersHorizontal, X } from "lucide-svelte";
	import type { FilterChange, Filters, SourceOption } from "./types";

	interface Props { filters: Filters; stages: string[]; labels: LeadLabel[]; sources: SourceOption[]; onfilter: (c: FilterChange) => void }
	let { filters, stages, labels, sources, onfilter }: Props = $props();

	// svelte-ignore state_referenced_locally
	let search = $state(filters.q ?? "");
	let moreOpen = $state(false);

	const views = [
		{ key: "open", label: "Semua berjalan", apply: { status: "", stage: "", mine: false, review: false } },
		{ key: "mine", label: "Lead saya", apply: { mine: true, stage: "", review: false } },
		{ key: "bukti", label: "Bukti masuk", apply: { stage: "bukti_masuk", mine: false, review: false } },
		{ key: "review", label: "Perlu dicek", apply: { review: true, stage: "", mine: false } },
	];
	let view = $derived(
		filters.mine ? "mine" : filters.stage === "bukti_masuk" ? "bukti" : filters.review ? "review" : filters.status === "" || filters.status === "open" ? "open" : "",
	);

	const fmtDate = (d: string) => new Date(d + "T12:00:00+07:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "Asia/Jakarta" });
	// Every active filter becomes a removable chip so the current scope is always visible.
	let chips = $derived.by(() => {
		const out: { label: string; clear: FilterChange }[] = [];
		if (filters.status && filters.status !== "open") out.push({ label: filters.status === "all" ? "Semua status" : `Status: ${STATUS_LABELS[filters.status] ?? filters.status}`, clear: { status: "" } });
		if (filters.stage) out.push({ label: `Tahap: ${STAGE_LABELS[filters.stage] ?? filters.stage}`, clear: { stage: "" } });
		if (filters.reached_stage) out.push({ label: `Mencapai: ${STAGE_LABELS[filters.reached_stage] ?? filters.reached_stage}`, clear: { reached_stage: "" } });
		if (filters.label) out.push({ label: `Label: ${labels.find((l) => l.id === filters.label)?.nama ?? filters.label}`, clear: { label: 0 } });
		if (filters.mine) out.push({ label: "Lead saya", clear: { mine: false } });
		if (filters.review) out.push({ label: "Perlu dicek", clear: { review: false } });
		if (filters.sumber) out.push({ label: `Sumber: ${filters.sumber === "none" ? "Tanpa sumber" : (sources.find((s) => String(s.id) === filters.sumber)?.nama ?? filters.sumber)}`, clear: { sumber: "" } });
		if (filters.lost_reason) out.push({ label: `Alasan lost: ${LOST_REASON_LABELS[filters.lost_reason] ?? filters.lost_reason.replaceAll("_", " ")}`, clear: { lost_reason: "" } });
		if (filters.q) out.push({ label: `“${filters.q}”`, clear: { q: "" } });
		if (filters.from || filters.to) {
			const what = filters.date_field === "converted" ? "Closing" : "Masuk";
			const range = filters.from && filters.to ? (filters.from === filters.to ? fmtDate(filters.from) : `${fmtDate(filters.from)} – ${fmtDate(filters.to)}`) : filters.from ? `sejak ${fmtDate(filters.from)}` : `sampai ${fmtDate(filters.to)}`;
			out.push({ label: `${what}: ${range}`, clear: { from: "", to: "", date_field: "" } });
		}
		return out;
	});

	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 transition-colors focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
	const seg = "shrink-0 whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
	const segOn = " bg-white text-neutral-900 shadow-[0_0_0_1px_rgba(15,23,42,.08),0_1px_2px_rgba(15,23,42,.08)] dark:bg-neutral-800 dark:text-white dark:shadow-[0_0_0_1px_rgba(255,255,255,.08)]";
	const segIdle = " text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white";
</script>

<div class="space-y-3">
	<div class="flex flex-col gap-2 lg:flex-row lg:items-center">
		<div class="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
			<div role="group" aria-label="Antrean" class="inline-flex gap-0.5 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900">
				{#each views as v}
					<button type="button" onclick={() => onfilter(v.apply)} aria-pressed={view === v.key} class={seg + (view === v.key ? segOn : segIdle)}>{v.label}</button>
				{/each}
			</div>
		</div>
		<form class="relative min-w-0 flex-1" onsubmit={(e) => { e.preventDefault(); onfilter({ q: search.trim() }); }}>
			<Search size={16} class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
			<input type="search" bind:value={search} placeholder="Cari nama, nama WA, atau nomor" aria-label="Cari lead" class={field + " h-10 pl-9 lg:h-[42px]"} />
		</form>
		<button type="button" onclick={() => (moreOpen = !moreOpen)} aria-expanded={moreOpen} aria-controls="lead-filter-more"
			class={"inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-md border px-3 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 lg:h-[42px] " +
				(moreOpen ? "border-brand-500/50 bg-brand-500/10 text-brand-700 dark:text-brand-300" : "border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.03]")}>
			<SlidersHorizontal size={15} /> Filter
		</button>
	</div>

	{#if moreOpen}
		<div id="lead-filter-more" class="grid grid-cols-2 gap-3 rounded-lg border border-neutral-200/80 bg-white p-3 sm:grid-cols-3 lg:grid-cols-6 dark:border-white/[0.07] dark:bg-neutral-925/60">
			<label class={lbl}>Status
				<select class={field + " mt-1"} value={filters.status || "open"} onchange={(e) => onfilter({ status: e.currentTarget.value === "open" ? "" : e.currentTarget.value })}>
					{#each ["open", "won", "lost", "do_not_contact", "all"] as s}<option value={s}>{s === "all" ? "Semua status" : STATUS_LABELS[s]}</option>{/each}
				</select>
			</label>
			<label class={lbl}>Tahap
				<select class={field + " mt-1"} value={filters.stage} onchange={(e) => onfilter({ stage: e.currentTarget.value })}>
					<option value="">Semua tahap</option>
					{#each stages as s}<option value={s}>{STAGE_LABELS[s] ?? s}</option>{/each}
				</select>
			</label>
			<label class={lbl}>Label
				<select class={field + " mt-1"} value={String(filters.label || "")} onchange={(e) => onfilter({ label: Number(e.currentTarget.value) || 0 })}>
					<option value="">Semua label</option>
					{#each labels as l}<option value={String(l.id)}>{l.nama}</option>{/each}
				</select>
			</label>
			<label class={lbl}>Sumber
				<select class={field + " mt-1"} value={filters.sumber ?? ""} onchange={(e) => onfilter({ sumber: e.currentTarget.value })}>
					<option value="">Semua sumber</option>
					{#each sources as s}<option value={String(s.id)}>{s.nama}{s.is_aktif ? "" : " (nonaktif)"}</option>{/each}
					<option value="none">Tanpa sumber</option>
				</select>
			</label>
			<label class={lbl}>{filters.date_field === "converted" ? "Closing dari" : "Masuk dari"}
				<input type="date" class={field + " mt-1"} value={filters.from} onchange={(e) => onfilter({ from: e.currentTarget.value })} />
			</label>
			<label class={lbl}>Sampai
				<input type="date" class={field + " mt-1"} value={filters.to} onchange={(e) => onfilter({ to: e.currentTarget.value })} />
			</label>
		</div>
	{/if}

	{#if chips.length}
		<div class="flex flex-wrap items-center gap-1.5">
			{#each chips as c}
				<button type="button" onclick={() => onfilter(c.clear)} aria-label={`Hapus filter ${c.label}`}
					class="inline-flex items-center gap-1 rounded-md border border-neutral-200 bg-white py-1 pl-2 pr-1.5 text-xs text-neutral-700 transition-colors hover:border-neutral-300 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-925 dark:text-neutral-300 dark:hover:bg-white/[0.04]">
					{c.label}<X size={12} class="text-neutral-400" />
				</button>
			{/each}
			<a href="/app/leads" use:inertia class="ml-1 rounded px-1 text-xs font-medium text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Reset semua</a>
		</div>
	{/if}
	{#if filters.reached_stage}
		<p class="text-xs text-neutral-500 dark:text-neutral-400">Filter funnel: lead yang diperkirakan sudah mencapai tahap {STAGE_LABELS[filters.reached_stage] ?? filters.reached_stage}, termasuk yang sudah closing.</p>
	{/if}
	{#if filters.date_field === "converted"}
		<p class="text-xs text-neutral-500 dark:text-neutral-400">Tanggal memakai tanggal closing (jadi santri), bukan tanggal lead pertama masuk.</p>
	{/if}
</div>
