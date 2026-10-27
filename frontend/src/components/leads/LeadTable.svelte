<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { STAGE_LABELS, STATUS_LABELS, leadName, relativeTime, stageTone } from "@lib/leads";
	import { ChevronRight } from "lucide-svelte";
	import { panel } from "@components/lead-dashboard/types";
	import LeadFlags from "./LeadFlags.svelte";
	import type { LeadItem } from "./types";

	let { items }: { items: LeadItem[] } = $props();

	const stageText = (l: LeadItem) => (l.status === "open" ? STAGE_LABELS[l.stage] : STATUS_LABELS[l.status]) ?? l.stage;
	function open(e: MouseEvent, id: number) {
		if ((e.target as HTMLElement).closest("a")) return;
		router.get(`/app/leads/${id}`);
	}
</script>

<!-- Desktop: dense table. The whole row opens the lead; the name stays a real link for keyboard and new-tab use. -->
<div class={panel + " hidden overflow-hidden lg:block"}>
	<table class="w-full table-fixed text-sm">
		<colgroup><col class="w-[30%]" /><col class="w-[17%]" /><col class="w-[9%]" /><col class="w-[14%]" /><col /><col class="w-[9rem]" /></colgroup>
		<thead class="border-b border-neutral-200/80 bg-neutral-50/70 text-left dark:border-white/[0.06] dark:bg-white/[0.02]">
			<tr class="text-[11px] font-semibold uppercase tracking-[0.06em] text-neutral-500 dark:text-neutral-400">
				<th scope="col" class="px-4 py-2.5 font-semibold">Lead</th>
				<th scope="col" class="px-3 py-2.5 font-semibold">Tahap</th>
				<th scope="col" class="px-3 py-2.5 font-semibold">Paket</th>
				<th scope="col" class="px-3 py-2.5 font-semibold">CS</th>
				<th scope="col" class="px-3 py-2.5 font-semibold">Tanda</th>
				<th scope="col" class="px-4 py-2.5 text-right font-semibold">Aktivitas</th>
			</tr>
		</thead>
		<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.04]">
			{#each items as lead (lead.id)}
				<tr class="group cursor-pointer transition-colors duration-100 hover:bg-neutral-50 dark:hover:bg-white/[0.025]" onclick={(e) => open(e, lead.id)}>
					<td class="px-4 py-2.5">
						<a href="/app/leads/{lead.id}" use:inertia class="block truncate font-medium text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 group-hover:text-brand-700 dark:text-white dark:group-hover:text-brand-400">{leadName(lead)}</a>
						{#if lead.nama && lead.wa_name && lead.nama !== lead.wa_name}
							<span class="block truncate text-xs text-neutral-500 dark:text-neutral-400">WA: {lead.wa_name}</span>
						{/if}
					</td>
					<td class="px-3 py-2.5"><span class="inline-block max-w-full truncate rounded px-1.5 py-0.5 text-xs font-medium {stageTone(lead.stage, lead.status)}">{stageText(lead)}</span></td>
					<td class="truncate px-3 py-2.5 text-xs text-neutral-600 dark:text-neutral-300">{#if lead.paket_kode}<span class="font-mono">{lead.paket_kode}</span>{:else}<span class="text-neutral-300 dark:text-neutral-600">—</span>{/if}</td>
					<td class="truncate px-3 py-2.5 text-xs text-neutral-600 dark:text-neutral-300">{#if lead.assigned_name}{lead.assigned_name}{:else}<span class="text-neutral-300 dark:text-neutral-600">—</span>{/if}</td>
					<td class="px-3 py-2"><LeadFlags {lead} /></td>
					<td class="px-4 py-2.5 text-right">
						<span class="inline-flex items-center gap-1.5 whitespace-nowrap text-xs tabular-nums text-neutral-500 dark:text-neutral-400">
							{relativeTime(lead.last_activity_at)}
							<ChevronRight size={14} class="text-neutral-300 transition-transform duration-150 group-hover:translate-x-0.5 dark:text-neutral-600" />
						</span>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<!-- Mobile/tablet: cards. -->
<ul class={panel + " divide-y divide-neutral-100 lg:hidden dark:divide-white/[0.04]"}>
	{#each items as lead (lead.id)}
		<li>
			<a href="/app/leads/{lead.id}" use:inertia class="flex items-start gap-3 px-4 py-3 transition-colors hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand-500 dark:hover:bg-white/[0.025]">
				<span class="min-w-0 flex-1">
					<span class="block truncate font-medium text-neutral-900 dark:text-white">{leadName(lead)}</span>
					<span class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-neutral-500 dark:text-neutral-400">
						<span class="rounded px-1.5 py-0.5 font-medium {stageTone(lead.stage, lead.status)}">{stageText(lead)}</span>
						{#if lead.paket_kode}<span class="font-mono">{lead.paket_kode}</span>{/if}
						{#if lead.assigned_name}<span>· {lead.assigned_name}</span>{/if}
					</span>
					<span class="mt-1.5 block empty:hidden"><LeadFlags {lead} /></span>
				</span>
				<span class="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap text-xs tabular-nums text-neutral-500 dark:text-neutral-400">
{relativeTime(lead.last_activity_at)}
				</span>
			</a>
		</li>
	{/each}
</ul>
