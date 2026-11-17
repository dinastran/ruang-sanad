<script lang="ts">
	import { Crown, Droplets, Flame, Footprints, Gem, ListChecks, Lock, PenLine, Repeat, Sparkles, Star, Sunrise, Target, Award } from "lucide-svelte";
	import type { Badge } from "@lib/tasbih/types";

	interface Props {
		badges: Badge[];
	}
	let { badges }: Props = $props();
	const IKON: Record<string, typeof Award> = { Crown, Droplets, Flame, Footprints, Gem, ListChecks, PenLine, Repeat, Sparkles, Star, Sunrise, Target };
	const diraih = $derived(badges.filter((b) => b.diraih_at).length);
	const tgl = (iso: string) => new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric" }).format(new Date(iso));
</script>

<p class="mb-3 text-sm text-neutral-500 dark:text-neutral-400"><b class="text-neutral-900 dark:text-white">{diraih}</b> dari {badges.length} pencapaian</p>
<ul class="grid grid-cols-2 gap-2 sm:grid-cols-3">
	{#each badges as b (b.kode)}
		{@const Ikon = IKON[b.ikon] ?? Award}
		<li class="flex items-start gap-3 rounded-xl p-3 {b.diraih_at ? 'bg-secondary-50 dark:bg-secondary-950/40' : 'bg-neutral-50 dark:bg-neutral-800/40'}">
			<span class="grid h-9 w-9 shrink-0 place-items-center rounded-full {b.diraih_at ? 'bg-secondary-500 text-white dark:bg-secondary-400 dark:text-neutral-950' : 'bg-neutral-200 text-neutral-400 dark:bg-neutral-800 dark:text-neutral-500'}">
				{#if b.diraih_at}<Ikon size={16} aria-hidden="true" />{:else}<Lock size={14} aria-hidden="true" />{/if}
			</span>
			<span class="min-w-0">
				<span class="block text-xs font-semibold leading-tight {b.diraih_at ? 'text-neutral-900 dark:text-white' : 'text-neutral-500 dark:text-neutral-400'}">{b.nama}</span>
				<span class="mt-0.5 block text-[11px] leading-snug text-neutral-500 dark:text-neutral-400">{b.diraih_at ? tgl(b.diraih_at) : b.syarat}</span>
			</span>
		</li>
	{/each}
</ul>
