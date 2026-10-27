<script lang="ts">
	import { labelTone } from "@lib/leads";
	import { AlertTriangle, PauseCircle } from "lucide-svelte";
	import type { LeadItem } from "./types";

	let { lead }: { lead: LeadItem } = $props();
</script>

{#if lead.needs_review || lead.ai_paused || lead.labels.length > 0}
	<span class="flex flex-wrap gap-1">
		{#if lead.needs_review}
			<span class="inline-flex items-center gap-1 rounded bg-warning/15 px-1.5 py-0.5 text-[11px] font-medium text-amber-800 dark:text-amber-300"><AlertTriangle size={11} /> Perlu dicek</span>
		{/if}
		{#if lead.ai_paused}
			<span class="inline-flex items-center gap-1 rounded bg-info/10 px-1.5 py-0.5 text-[11px] font-medium text-blue-700 dark:text-blue-300"><PauseCircle size={11} /> AI dijeda</span>
		{/if}
		{#each lead.labels as l}
			<span class="rounded border px-1.5 py-0.5 text-[11px] {labelTone(l.warna)}">{l.nama}</span>
		{/each}
	</span>
{/if}
