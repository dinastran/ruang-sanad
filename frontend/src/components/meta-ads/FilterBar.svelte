<script lang="ts">
	import { ChevronDown, Search, X } from "lucide-svelte";
	import PeriodPicker from "@components/lead-dashboard/PeriodPicker.svelte";
	import { shortDay, type CampaignInfo, type Filters, type SourceOption } from "./data";

	interface Props {
		from: string; to: string; prevFrom: string; prevTo: string; loading: boolean;
		compare: boolean; filters: Filters; campaigns: CampaignInfo[]; sources: SourceOption[];
		onperiod: (from: string, to: string) => void;
		oncompare: (on: boolean) => void;
		onfilters: (f: Filters) => void;
	}
	let { from, to, prevFrom, prevTo, loading, compare, filters, campaigns, sources, onperiod, oncompare, onfilters }: Props = $props();

	let query = $state("");
	let shown = $derived(campaigns.filter((c) => !query.trim() || c.name.toLowerCase().includes(query.trim().toLowerCase())));
	let campaignLabel = $derived(
		filters.campaigns.length === 0 ? "Semua kampanye"
			: filters.campaigns.length === 1 ? (campaigns.find((c) => c.id === filters.campaigns[0])?.name ?? "1 kampanye")
			: `${filters.campaigns.length} kampanye`,
	);
	function toggle(id: string) {
		const set = new Set(filters.campaigns);
		set.has(id) ? set.delete(id) : set.add(id);
		onfilters({ ...filters, campaigns: [...set] });
	}
	let sourceName = $derived(sources.find((s) => s.id === filters.source)?.nama ?? "");

	const control = "inline-flex h-9 items-center gap-2 rounded-md border border-neutral-200 bg-white px-3 text-sm text-neutral-800 transition-colors duration-150 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-925 dark:text-neutral-200 dark:hover:bg-white/[0.04]";
</script>

<div class="space-y-3">
	<div class="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
		<div class="shrink-0"><PeriodPicker {from} {to} {loading} onapply={onperiod} /></div>
		<div class="flex flex-wrap items-center gap-2 xl:justify-end">
			<label class={control + " cursor-pointer select-none"}>
				<input type="checkbox" class="h-4 w-4 accent-brand-600" checked={compare} onchange={(e) => oncompare(e.currentTarget.checked)} />
				<span>Bandingkan <span class="hidden text-neutral-500 sm:inline">({shortDay(prevFrom)} – {shortDay(prevTo)})</span></span>
			</label>

			<details class="group relative">
				<summary class={control + " list-none [&::-webkit-details-marker]:hidden"} aria-label="Filter kampanye">
					<span class="max-w-[14rem] truncate">{campaignLabel}</span><ChevronDown size={14} class="text-neutral-400 transition-transform group-open:rotate-180" />
				</summary>
				<div class="absolute right-0 z-20 mt-1.5 w-80 max-w-[calc(100vw-2rem)] rounded-lg border border-neutral-200 bg-white p-2 shadow-[0_8px_24px_rgba(15,23,42,.10)] dark:border-white/10 dark:bg-neutral-900">
					<label class="relative block">
						<span class="sr-only">Cari kampanye</span>
						<Search size={14} class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-neutral-400" />
						<input type="search" bind:value={query} placeholder="Cari kampanye" class="h-8 w-full rounded-md border border-neutral-200 bg-neutral-50 pl-8 pr-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white" />
					</label>
					<ul class="mt-2 max-h-64 overflow-y-auto">
						{#each shown as c (c.id)}
							<li>
								<label class="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-neutral-50 dark:hover:bg-white/[0.04]">
									<input type="checkbox" class="mt-0.5 h-4 w-4 shrink-0 accent-brand-600" checked={filters.campaigns.includes(c.id)} onchange={() => toggle(c.id)} />
									<span class="min-w-0"><span class="block truncate text-neutral-900 dark:text-white">{c.name || c.id}</span><span class="text-[11px] text-neutral-400">terakhir aktif {shortDay(c.last_date)}{c.source_id ? "" : " · belum dihubungkan"}</span></span>
								</label>
							</li>
						{:else}
							<li class="px-2 py-3 text-sm text-neutral-500">Tidak ada kampanye.</li>
						{/each}
					</ul>
					{#if filters.campaigns.length}
						<button type="button" class="mt-1 w-full rounded-md px-2 py-1.5 text-left text-xs font-medium text-brand-700 hover:bg-neutral-50 dark:text-brand-400 dark:hover:bg-white/[0.04]" onclick={() => onfilters({ ...filters, campaigns: [] })}>Pilih semua kampanye</button>
					{/if}
				</div>
			</details>

			<select class={control + " w-full pr-8 sm:w-auto"} aria-label="Filter sumber lead" value={String(filters.source)} onchange={(e) => onfilters({ ...filters, source: Number(e.currentTarget.value) || 0 })}>
				<option value="0">Semua sumber lead</option>
				{#each sources as s (s.id)}<option value={String(s.id)}>{s.nama}</option>{/each}
			</select>
		</div>
	</div>

	{#if filters.campaigns.length || filters.source}
		<div class="flex flex-wrap items-center gap-1.5">
			{#each filters.campaigns as id (id)}
				<button type="button" onclick={() => toggle(id)} class="inline-flex max-w-xs items-center gap-1 rounded-md border border-neutral-200 bg-white py-1 pl-2 pr-1.5 text-xs text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:bg-neutral-925 dark:text-neutral-300" aria-label="Hapus filter kampanye">
					<span class="truncate">Kampanye: {campaigns.find((c) => c.id === id)?.name ?? id}</span><X size={12} class="shrink-0 text-neutral-400" />
				</button>
			{/each}
			{#if filters.source}
				<button type="button" onclick={() => onfilters({ ...filters, source: 0 })} class="inline-flex items-center gap-1 rounded-md border border-neutral-200 bg-white py-1 pl-2 pr-1.5 text-xs text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:bg-neutral-925 dark:text-neutral-300" aria-label="Hapus filter sumber">
					Sumber: {sourceName}<X size={12} class="text-neutral-400" />
				</button>
			{/if}
			<button type="button" onclick={() => onfilters({ campaigns: [], source: 0 })} class="ml-1 text-xs font-medium text-brand-700 hover:underline dark:text-brand-400">Reset filter</button>
		</div>
	{/if}
</div>
