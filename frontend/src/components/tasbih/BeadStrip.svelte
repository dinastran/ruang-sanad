<script lang="ts">
	// Untaian butir tasbih: setiap tap untaian bergeser satu butir ke kiri.
	// Butir ke-33 (kelipatan) ditandai lebih besar seperti pemisah tasbih asli.
	interface Props {
		count: number;
		pemisah?: number;
	}
	let { count, pemisah = 33 }: Props = $props();
	const JUMLAH = 11;
	const TENGAH = Math.floor(JUMLAH / 2);
	const beads = $derived(
		Array.from({ length: JUMLAH }, (_, i) => {
			const n = count + (i - TENGAH);
			return { n, aktif: i === TENGAH, lewat: i < TENGAH && n > 0, penanda: n > 0 && n % pemisah === 0 };
		}),
	);
</script>

<div class="relative h-10 w-full max-w-[22rem] overflow-hidden" aria-hidden="true">
	<div class="absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-neutral-300 dark:bg-neutral-700"></div>
	{#key count}
		<div class="strip absolute inset-0 flex items-center justify-between px-1">
			{#each beads as b (b.n)}
				<span
					class="rounded-full transition-colors duration-200
						{b.penanda ? 'h-5 w-5 bg-secondary-500 dark:bg-secondary-400' : 'h-3.5 w-3.5'}
						{!b.penanda && b.aktif ? 'bg-brand-500 dark:bg-brand-400 ring-4 ring-brand-500/20' : ''}
						{!b.penanda && !b.aktif && b.lewat ? 'bg-brand-300 dark:bg-brand-700' : ''}
						{!b.penanda && !b.aktif && !b.lewat ? 'bg-neutral-300 dark:bg-neutral-600' : ''}"
				></span>
			{/each}
		</div>
	{/key}
	<div class="pointer-events-none absolute inset-y-0 left-0 w-10 bg-gradient-to-r from-white dark:from-neutral-900"></div>
	<div class="pointer-events-none absolute inset-y-0 right-0 w-10 bg-gradient-to-l from-white dark:from-neutral-900"></div>
</div>

<style>
	.strip {
		animation: geser 200ms cubic-bezier(0.32, 0.72, 0, 1);
	}
	@keyframes geser {
		from {
			transform: translateX(calc(100% / 11));
		}
		to {
			transform: translateX(0);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.strip {
			animation: none;
		}
	}
</style>
