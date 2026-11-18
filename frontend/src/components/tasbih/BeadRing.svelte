<script lang="ts">
	// Progres harian sebagai untaian tasbih 33 butir melingkar, dengan butir
	// "imam" (lebih besar) di puncak seperti tasbih asli. Butir menyala sesuai
	// progres; saat target tercapai seluruh untaian berubah warna sukses.
	import { onMount } from "svelte";

	interface Props {
		progress: number; // 0..n (boleh > 1)
		size?: number;
		children?: import("svelte").Snippet;
	}
	let { progress, size = 200, children }: Props = $props();

	const N = 33;
	let siap = $state(false);
	onMount(() => {
		// nyalakan butir setelah mount agar transisi bertahap terlihat
		const id = requestAnimationFrame(() => (siap = true));
		return () => cancelAnimationFrame(id);
	});

	const tercapai = $derived(progress >= 1);
	const nyala = $derived(siap ? (tercapai ? N : Math.floor(Math.max(0, progress) * N)) : 0);
	const c = $derived(size / 2);
	const rButir = $derived(size * 0.03);
	const R = $derived(c - size * 0.06);
	const butir = $derived(
		Array.from({ length: N }, (_, i) => {
			const a = ((i + 1) / (N + 1)) * 2 * Math.PI - Math.PI / 2;
			return { x: c + R * Math.cos(a), y: c + R * Math.sin(a) };
		}),
	);
</script>

<div class="relative grid shrink-0 place-items-center" style="width:{size}px;height:{size}px">
	<svg width={size} height={size} viewBox="0 0 {size} {size}" aria-hidden="true">
		<circle cx={c} cy={c} r={R} fill="none" stroke-width="1" class="stroke-neutral-200 dark:stroke-neutral-800" />
		{#each butir as b, i}
			<circle
				cx={b.x}
				cy={b.y}
				r={rButir}
				style="transition-delay:{i * 14}ms"
				class="butir {i < nyala ? (tercapai ? 'fill-success' : 'fill-brand-500 dark:fill-brand-400') : 'fill-neutral-200 dark:fill-neutral-800'}"
			/>
		{/each}
		<!-- imam -->
		<circle cx={c} cy={c - R} r={rButir * 1.7} class="butir {tercapai ? 'fill-success' : 'fill-neutral-300 dark:fill-neutral-700'}" style="transition-delay:{N * 14}ms" />
	</svg>
	<div class="absolute inset-0 grid place-items-center">{@render children?.()}</div>
</div>

<style>
	.butir {
		transition: fill 260ms cubic-bezier(0.23, 1, 0.32, 1);
	}
	@media (prefers-reduced-motion: reduce) {
		.butir {
			transition: none;
		}
	}
</style>
