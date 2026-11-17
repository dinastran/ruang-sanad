<script lang="ts">
	import { fly } from "svelte/transition";
	import { Award } from "lucide-svelte";
	import type { Badge } from "@lib/tasbih/types";

	interface Props {
		badges: Badge[];
		ondismiss: (kode: string) => void;
	}
	let { badges, ondismiss }: Props = $props();

	function jadwalTutup(node: HTMLElement, kode: string) {
		const t = setTimeout(() => ondismiss(kode), 5000);
		return { destroy: () => clearTimeout(t) };
	}
</script>

<div class="pointer-events-none fixed inset-x-0 top-20 z-[60] flex flex-col items-center gap-2 px-4" aria-live="polite">
	{#each badges as b (b.kode)}
		<button
			type="button"
			use:jadwalTutup={b.kode}
			in:fly={{ y: -16, duration: 260 }}
			out:fly={{ y: -8, duration: 160 }}
			onclick={() => ondismiss(b.kode)}
			class="pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-2xl border border-secondary-200 bg-white px-4 py-3 text-left shadow-lg dark:border-secondary-800 dark:bg-neutral-900"
		>
			<span class="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-100 text-secondary-600 dark:bg-secondary-950 dark:text-secondary-300">
				<Award size={20} aria-hidden="true" />
			</span>
			<span class="min-w-0">
				<span class="block text-xs font-medium uppercase tracking-wide text-secondary-600 dark:text-secondary-300">Pencapaian baru</span>
				<span class="block truncate text-sm font-semibold text-neutral-900 dark:text-white">{b.nama}</span>
				<span class="block truncate text-xs text-neutral-500 dark:text-neutral-400">{b.syarat}</span>
			</span>
		</button>
	{/each}
</div>
