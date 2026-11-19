<script lang="ts">
	// Untaian tasbih melingkar: butir terisi mengikuti hitungan dalam satu putaran,
	// butir aktif (berikutnya) sedikit membesar. Jumlah butir = target bila ≤ 33,
	// selain itu 33 (satu putaran tasbih) dan hitungan diputar modulo 33.
	import type { Snippet } from "svelte";

	interface Props {
		count: number;
		target: number;
		size?: number;
		tercapai?: boolean;
		children?: Snippet;
	}
	let { count, target, size = 280, tercapai = false, children }: Props = $props();

	const n = $derived(target > 0 && target <= 33 ? target : 33);
	// Saat tepat kelipatan (mis. 33/33) tampilkan untaian penuh, bukan kosong.
	const terisi = $derived(count <= 0 ? 0 : tercapai || count % n === 0 ? n : count % n);
	const c = $derived(size / 2);
	const R = $derived(size / 2 - 14);
	const rButir = $derived(Math.max(3, Math.min(6, (Math.PI * R) / n - 1.5)));
	const butir = $derived(
		Array.from({ length: n }, (_, i) => {
			const a = -Math.PI / 2 + (i * 2 * Math.PI) / n;
			return { x: c + R * Math.cos(a), y: c + R * Math.sin(a), isi: i < terisi, aktif: !tercapai && i === terisi % n };
		}),
	);
</script>

<div class="relative grid place-items-center" style="width:{size}px;height:{size}px">
	<svg width={size} height={size} viewBox="0 0 {size} {size}" aria-hidden="true">
		<circle cx={c} cy={c} r={R} fill="none" stroke-width="1" class="stroke-neutral-200 dark:stroke-brand-200/10" />
		{#each butir as b, i (i)}
			<circle
				cx={b.x}
				cy={b.y}
				r={b.aktif ? rButir + 2.5 : rButir}
				class="butir {b.isi
					? tercapai
						? 'fill-success'
						: 'fill-brand-500 dark:fill-brand-300'
					: b.aktif
						? 'fill-neutral-900 dark:fill-white'
						: 'fill-neutral-200 dark:fill-brand-950'}"
			/>
		{/each}
	</svg>
	<div class="absolute inset-0 grid place-items-center">{@render children?.()}</div>
</div>

<style>
	.butir {
		transition:
			r 160ms cubic-bezier(0.32, 0.72, 0, 1),
			fill 200ms ease;
	}
	@media (prefers-reduced-motion: reduce) {
		.butir {
			transition: none;
		}
	}
</style>
