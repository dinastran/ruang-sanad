<script lang="ts">
	// Pencapaian ringkas: ikon dalam grid, nama di bawahnya. Warna sand khusus pencapaian.
	// Ketuk/arah-fokus satu ikon untuk melihat syarat & tanggal diraih.
	import { Crown, Droplets, Flame, Footprints, Gem, ListChecks, Lock, PenLine, Repeat, Sparkles, Star, Sunrise, Target, Award } from "lucide-svelte";
	import type { Badge } from "@lib/tasbih/types";

	interface Props {
		badges: Badge[];
	}
	let { badges }: Props = $props();
	const IKON: Record<string, typeof Award> = { Crown, Droplets, Flame, Footprints, Gem, ListChecks, PenLine, Repeat, Sparkles, Star, Sunrise, Target };
	const diraih = $derived(badges.filter((b) => b.diraih_at).length);
	const tgl = (iso: string) => new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric" }).format(new Date(iso));
	let pilih = $state<string | null>(null);
	const terpilih = $derived(badges.find((b) => b.kode === pilih) ?? null);
</script>

<p class="mb-3 text-[13px] text-neutral-500 dark:text-brand-100/50"><b class="font-semibold text-neutral-900 dark:text-white">{diraih}</b> dari {badges.length} pencapaian</p>
<ul class="grid grid-cols-5 gap-2">
	{#each badges as b (b.kode)}
		{@const Ikon = IKON[b.ikon] ?? Award}
		<li>
			<button
				type="button"
				onclick={() => (pilih = pilih === b.kode ? null : b.kode)}
				aria-pressed={pilih === b.kode}
				aria-label="{b.nama}{b.diraih_at ? ', diraih' : ', belum diraih'}"
				class="grid aspect-square w-full place-items-center rounded-2xl border transition
					{b.diraih_at
						? 'border-sand-200 bg-sand-50 text-sand-600 dark:border-sand-300/20 dark:bg-sand-300/10 dark:text-sand-300'
						: 'border-neutral-200 bg-neutral-50 text-neutral-300 dark:border-brand-200/10 dark:bg-transparent dark:text-brand-100/25'}
					{pilih === b.kode ? 'ring-2 ring-brand-400' : ''}"
			>
				{#if b.diraih_at}<Ikon size={20} aria-hidden="true" />{:else}<Lock size={15} aria-hidden="true" />{/if}
			</button>
		</li>
	{/each}
</ul>
<p class="mt-3 min-h-[2.5rem] text-[13px] leading-snug text-neutral-500 dark:text-brand-100/50" aria-live="polite">
	{#if terpilih}
		<b class="font-semibold text-neutral-900 dark:text-white">{terpilih.nama}</b> · {terpilih.syarat}{terpilih.diraih_at ? ` · diraih ${tgl(terpilih.diraih_at)}` : ""}
	{:else}
		{@const berikut = badges.find((b) => !b.diraih_at)}
		{#if berikut}Berikutnya: <span class="text-neutral-700 dark:text-brand-100/80">{berikut.nama}</span> · {berikut.syarat}{:else}Semua pencapaian sudah diraih.{/if}
	{/if}
</p>
