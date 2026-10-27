<script lang="ts">
	import { formatPersen, type RekapGuru } from "@lib/riwayatAbsensi";
	import { UserCheck } from "lucide-svelte";

	interface Props {
		items: RekapGuru[];
		/** Guru yang sedang difilter; diklik lagi untuk melepas. */
		aktif?: string;
		onpilih: (guruID: number) => void;
	}

	let { items, aktif = "", onpilih }: Props = $props();

	const maks = $derived(Math.max(1, ...items.map((g) => g.rekap.alfa)));
</script>

<section class="h-full rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
	<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Guru dengan alfa terbanyak</h2>
	<p class="mt-0.5 text-xs text-neutral-500">Klik nama untuk menyaring riwayat guru tersebut</p>

	{#if items.length === 0}
		<div class="flex h-48 flex-col items-center justify-center gap-2 text-sm text-neutral-500">
			<UserCheck class="h-7 w-7 opacity-40" />
			Tidak ada alfa pada filter ini.
		</div>
	{:else}
		<ol class="mt-4 space-y-2.5">
			{#each items as g (g.guru_id)}
				{@const dipilih = aktif === String(g.guru_id)}
				<li>
					<button
						type="button"
						onclick={() => onpilih(g.guru_id)}
						aria-pressed={dipilih}
						class="group block w-full rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-neutral-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60 dark:hover:bg-white/[0.03] {dipilih ? 'bg-brand-400/10' : ''}"
					>
						<div class="flex items-baseline justify-between gap-3 text-sm">
							<span class="truncate font-medium text-neutral-800 group-hover:text-brand-700 dark:text-neutral-200 dark:group-hover:text-brand-300">{g.guru_nama}</span>
							<span class="shrink-0 tabular-nums text-xs text-neutral-500"><span class="font-semibold text-red-600 dark:text-red-400">{g.rekap.alfa} alfa</span> · hadir {formatPersen(g.rekap.persen === null ? null : Math.round(g.rekap.persen))}</span>
						</div>
						<div class="mt-1.5 h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
							<div class="h-full rounded-full bg-red-500" style="width: {(g.rekap.alfa / maks) * 100}%"></div>
						</div>
					</button>
				</li>
			{/each}
		</ol>
	{/if}
</section>
