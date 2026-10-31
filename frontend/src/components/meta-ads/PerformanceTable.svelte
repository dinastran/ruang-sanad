<script lang="ts">
	import { ArrowDown, ArrowUp, Search } from "lucide-svelte";
	import { num, pct, ratio, rp, type Entity, type Level, type SourceOption } from "./data";

	interface Props {
		rows: Entity[];
		level: Level;
		sources: SourceOption[];
		onlevel: (l: Level) => void;
		onfiltercampaign: (id: string) => void;
		onmap: (campaignId: string, sourceId: number) => void;
	}
	let { rows, level, sources, onlevel, onfiltercampaign, onmap }: Props = $props();

	type SortKey = "name" | "spend" | "impressions" | "link" | "ctr" | "chats" | "cpc" | "leads" | "won" | "cps";
	let sortKey = $state<SortKey>("spend");
	let asc = $state(false);
	let query = $state("");
	let showAll = $state(false);
	const PAGE = 25;

	const value = (e: Entity, k: SortKey): number | string => {
		switch (k) {
			case "name": return e.name.toLowerCase();
			case "ctr": return ratio(e.link, e.impressions);
			case "cpc": return ratio(e.spend, e.chats);
			case "cps": return ratio(e.spend, e.won);
			default: return e[k];
		}
	};
	let filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const list = q ? rows.filter((e) => e.name.toLowerCase().includes(q) || e.parent.toLowerCase().includes(q)) : [...rows];
		list.sort((a, b) => {
			const va = value(a, sortKey), vb = value(b, sortKey);
			let o: number;
			if (typeof va === "string") o = va.localeCompare(vb as string, "id");
			else {
				// Empty ratios (NaN) always sort last.
				const na = Number.isFinite(va), nb = Number.isFinite(vb as number);
				if (!na || !nb) return na === nb ? 0 : na ? -1 : 1;
				o = va - (vb as number);
			}
			return asc ? o : -o;
		});
		return list;
	});
	let visible = $derived(showAll ? filtered : filtered.slice(0, PAGE));
	let totalSpend = $derived(rows.reduce((s, e) => s + e.spend, 0));
	let avgCpc = $derived(ratio(totalSpend, rows.reduce((s, e) => s + e.chats, 0)));

	/** Rp/chat against the table average; needs a few chats to mean anything. */
	function badge(e: Entity) {
		const c = ratio(e.spend, e.chats);
		if (!Number.isFinite(c) || !Number.isFinite(avgCpc) || e.chats < 3) return null;
		if (c <= avgCpc * 0.8) return { label: "efisien", cls: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400" };
		if (c >= avgCpc * 1.3) return { label: "mahal", cls: "bg-amber-500/10 text-amber-700 dark:text-amber-400" };
		return null;
	}
	function sortBy(k: SortKey) {
		if (sortKey === k) asc = !asc;
		else { sortKey = k; asc = k === "name" || k === "cpc" || k === "cps"; }
	}

	let columns = $derived<{ key: SortKey; label: string }[]>([
		{ key: "spend", label: "Biaya" }, { key: "impressions", label: "Impresi" }, { key: "link", label: "Klik tautan" },
		{ key: "ctr", label: "CTR" }, { key: "chats", label: "Chat WA" }, { key: "cpc", label: "Rp/chat" },
		...(level === "campaign" ? ([{ key: "leads", label: "Lead" }, { key: "won", label: "Santri" }, { key: "cps", label: "Rp/santri" }] as { key: SortKey; label: string }[]) : []),
	]);
	const levels: { key: Level; label: string }[] = [{ key: "campaign", label: "Kampanye" }, { key: "adset", label: "Ad set" }, { key: "ad", label: "Iklan" }];
	const td = "px-3 py-2.5 text-right tabular-nums text-neutral-700 dark:text-neutral-300";
	const tab = "rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
</script>

<section class="min-w-0 overflow-hidden rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60" aria-labelledby="tabel-title">
	<div class="flex flex-wrap items-end justify-between gap-3 px-5 pt-5 sm:px-6">
		<div>
			<h2 id="tabel-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Rincian performa</h2>
			<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
				Rata-rata {rp(avgCpc)}/chat WA. Label <span class="text-emerald-700 dark:text-emerald-400">efisien</span> ≤80% rata-rata, <span class="text-amber-700 dark:text-amber-400">mahal</span> ≥130% (minimal 3 chat).
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<div class="inline-flex gap-0.5 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900" role="tablist" aria-label="Level tabel">
				{#each levels as l (l.key)}
					<button type="button" role="tab" aria-selected={level === l.key} onclick={() => { onlevel(l.key); showAll = false; }}
						class={tab + (level === l.key ? " bg-white text-neutral-900 shadow-[0_0_0_1px_rgba(15,23,42,.08)] dark:bg-neutral-800 dark:text-white" : " text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white")}>{l.label}</button>
				{/each}
			</div>
			<label class="relative block">
				<span class="sr-only">Cari</span>
				<Search size={14} class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-neutral-400" />
				<input type="search" bind:value={query} placeholder="Cari nama" class="h-9 w-44 rounded-md border border-neutral-200 bg-neutral-50 pl-8 pr-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white" />
			</label>
		</div>
	</div>

	{#if rows.length === 0}
		<p class="px-5 py-10 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Belum ada data iklan untuk periode dan filter ini.</p>
	{:else}
		<div class="mt-4 overflow-x-auto">
			<table class="w-full text-sm" class:min-w-[1080px]={level === "campaign"} class:min-w-[820px]={level !== "campaign"}>
				<thead>
					<tr class="border-y border-neutral-100 bg-neutral-50/60 text-[11px] text-neutral-500 dark:border-white/[0.05] dark:bg-white/[0.015] dark:text-neutral-400">
						<th scope="col" class="py-0 pl-5 pr-3 text-left font-medium sm:pl-6" aria-sort={sortKey === "name" ? (asc ? "ascending" : "descending") : "none"}>
							<button type="button" onclick={() => sortBy("name")} class="inline-flex h-9 items-center gap-1 hover:text-neutral-900 dark:hover:text-white">{levels.find((l) => l.key === level)?.label}{#if sortKey === "name"}{#if asc}<ArrowUp size={12} />{:else}<ArrowDown size={12} />{/if}{/if}</button>
						</th>
						{#each columns as c (c.key)}
							<th scope="col" class="px-3 py-0 text-right font-medium" aria-sort={sortKey === c.key ? (asc ? "ascending" : "descending") : "none"}>
								<button type="button" onclick={() => sortBy(c.key)} class={"inline-flex h-9 items-center gap-1 hover:text-neutral-900 dark:hover:text-white " + (sortKey === c.key ? "text-neutral-900 dark:text-white" : "")}>{c.label}{#if sortKey === c.key}{#if asc}<ArrowUp size={12} />{:else}<ArrowDown size={12} />{/if}{/if}</button>
							</th>
						{/each}
						{#if level === "campaign"}<th scope="col" class="py-0 pl-3 pr-5 text-left font-medium sm:pr-6">Sumber lead</th>{/if}
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
					{#each visible as e (e.id)}
						{@const b = badge(e)}
						<tr class="align-top hover:bg-neutral-50/70 dark:hover:bg-white/[0.02]">
							<th scope="row" class="max-w-[22rem] py-2.5 pl-5 pr-3 text-left font-normal sm:pl-6">
								{#if level === "campaign"}
									<button type="button" onclick={() => onfiltercampaign(e.id)} class="block max-w-full truncate text-left font-medium text-neutral-900 hover:text-brand-700 dark:text-white dark:hover:text-brand-400" title={`Filter ke ${e.name}`}>{e.name}</button>
								{:else}
									<span class="block truncate font-medium text-neutral-900 dark:text-white" title={e.name}>{e.name}</span>
								{/if}
								{#if e.parent}<span class="block truncate text-[11px] text-neutral-400" title={e.parent}>{e.parent}</span>{/if}
							</th>
							<td class={td}>
								<span class="font-semibold text-neutral-900 dark:text-white">{rp(e.spend)}</span>
								<span class="ml-auto mt-1 block h-1 w-20 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]" style="margin-left:auto"><span class="block h-full rounded-full bg-brand-600 dark:bg-brand-500" style:width={`${totalSpend ? (e.spend / totalSpend) * 100 : 0}%`}></span></span>
							</td>
							<td class={td}>{num(e.impressions)}</td>
							<td class={td}>{num(e.link)}</td>
							<td class={td}>{pct(ratio(e.link, e.impressions), 2)}</td>
							<td class={td}>{num(e.chats)}</td>
							<td class={td}>{rp(ratio(e.spend, e.chats))}{#if b}<span class={"ml-1.5 rounded px-1.5 py-px text-[10px] font-semibold " + b.cls}>{b.label}</span>{/if}</td>
							{#if level === "campaign"}
								<td class={td}>{e.sourceId ? num(e.leads) : "—"}</td>
								<td class={td}>{e.sourceId ? num(e.won) : "—"}</td>
								<td class={td}>{e.sourceId && e.sharedWith <= 1 ? rp(ratio(e.spend, e.won)) : "—"}</td>
								<td class="py-2 pl-3 pr-5 sm:pr-6">
									<select class="h-8 w-44 rounded-md border border-neutral-200 bg-neutral-50 px-2 text-xs text-neutral-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white" aria-label={`Sumber lead untuk ${e.name}`} value={String(e.sourceId || 0)} onchange={(ev) => onmap(e.id, Number(ev.currentTarget.value) || 0)}>
										<option value="0">— belum dihubungkan —</option>
										{#each sources as o (o.id)}<option value={String(o.id)}>{o.nama}{o.is_aktif ? "" : " (nonaktif)"}</option>{/each}
									</select>
									{#if e.sharedWith > 1}<span class="mt-1 block text-[11px] text-neutral-400">lead & santri dibagi {e.sharedWith} kampanye</span>{/if}
								</td>
							{/if}
						</tr>
					{:else}
						<tr><td colspan={columns.length + 2} class="px-5 py-8 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Tidak ada yang cocok dengan "{query}". <button type="button" class="font-medium text-brand-700 underline dark:text-brand-400" onclick={() => (query = "")}>Hapus pencarian</button></td></tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if filtered.length > PAGE}
			<div class="border-t border-neutral-100 px-5 py-3 text-center sm:px-6 dark:border-white/[0.05]">
				<button type="button" onclick={() => (showAll = !showAll)} class="text-sm font-medium text-brand-700 hover:underline dark:text-brand-400">{showAll ? "Tampilkan lebih sedikit" : `Tampilkan semua (${filtered.length})`}</button>
			</div>
		{/if}
	{/if}
</section>
