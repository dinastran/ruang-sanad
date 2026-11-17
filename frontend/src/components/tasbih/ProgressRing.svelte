<script lang="ts">
	// Ring progres SVG. progress 0..1; animasi lewat stroke-dashoffset.
	interface Props {
		progress: number;
		size?: number;
		stroke?: number;
		tercapai?: boolean;
		children?: import("svelte").Snippet;
	}
	let { progress, size = 260, stroke = 10, tercapai = false, children }: Props = $props();
	const r = $derived((size - stroke) / 2);
	const keliling = $derived(2 * Math.PI * r);
	const offset = $derived(keliling * (1 - Math.max(0, Math.min(1, progress))));
</script>

<div class="relative grid place-items-center" style="width:{size}px;height:{size}px">
	<svg width={size} height={size} viewBox="0 0 {size} {size}" class="-rotate-90" aria-hidden="true">
		<circle cx={size / 2} cy={size / 2} {r} fill="none" stroke-width={stroke} class="stroke-neutral-200/80 dark:stroke-neutral-800" />
		<circle
			cx={size / 2}
			cy={size / 2}
			{r}
			fill="none"
			stroke-width={stroke}
			stroke-linecap="round"
			stroke-dasharray={keliling}
			stroke-dashoffset={offset}
			class="ring-arc {tercapai ? 'stroke-success' : 'stroke-brand-500 dark:stroke-brand-400'}"
		/>
	</svg>
	<div class="absolute inset-0 grid place-items-center">{@render children?.()}</div>
</div>

<style>
	.ring-arc {
		transition:
			stroke-dashoffset 220ms cubic-bezier(0.32, 0.72, 0, 1),
			stroke 300ms ease;
	}
	@media (prefers-reduced-motion: reduce) {
		.ring-arc {
			transition: none;
		}
	}
</style>
