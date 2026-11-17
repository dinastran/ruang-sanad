<script lang="ts">
	// Tooltip ringan untuk chart SVG: diposisikan relatif ke kontainer chart.
	interface Props {
		x: number;
		y: number;
		judul: string;
		nilai: string;
	}
	let { x, y, judul, nilai }: Props = $props();
</script>

<div
	class="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full whitespace-nowrap rounded-lg bg-neutral-900 px-2.5 py-1.5 text-xs text-white shadow-lg dark:bg-white dark:text-neutral-900"
	style="left:{x}px;top:{y - 8}px"
	role="tooltip"
>
	<div class="text-[11px] opacity-70">{judul}</div>
	<div class="font-semibold tabular-nums">{nilai}</div>
</div>
