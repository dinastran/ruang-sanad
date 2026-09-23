<script lang="ts">
	interface Point { month: string; total: number }
	interface Props { points?: Point[]; onselect?: (month: string) => void; emptyText?: string }
	let { points = [], onselect, emptyText = "Belum ada data pendaftaran" }: Props = $props();
	const width = 900, height = 260, padX = 42, padTop = 24, padBottom = 42;
	let max = $derived(points.length ? Math.max(...points.map((point) => point.total), 1) : 1);
	let labelEvery = $derived(Math.max(1, Math.ceil(points.length / 8)));
	let coords = $derived(points.map((point, index) => {
		const usableWidth = width - padX * 2, usableHeight = height - padTop - padBottom;
		const x = points.length <= 1 ? width / 2 : padX + (index / (points.length - 1)) * usableWidth;
		const y = padTop + usableHeight - (point.total / max) * usableHeight;
		return { ...point, x, y };
	}));
	let polyline = $derived(coords.map((point) => `${point.x},${point.y}`).join(" "));
	function monthLabel(month: string): string {
		const [year, rawMonth] = month.split("-"), monthIndex = Number(rawMonth) - 1;
		if (!year || monthIndex < 0 || monthIndex > 11) return month;
		return new Intl.DateTimeFormat("id-ID", { month: "short", year: "2-digit" }).format(new Date(Number(year), monthIndex, 1));
	}
</script>

{#if points.length > 0}
	<div class="w-full overflow-x-auto">
		<svg viewBox={`0 0 ${width} ${height}`} class="min-w-[680px] w-full" role="img" aria-label="Grafik pendaftaran santri per bulan">
			<line x1={padX} y1={height - padBottom} x2={width - padX} y2={height - padBottom} class="stroke-neutral-200 dark:stroke-neutral-800" />
			{#each [0, 0.5, 1] as ratio}
				{@const y = padTop + (height - padTop - padBottom) * (1 - ratio)}
				<line x1={padX} y1={y} x2={width - padX} y2={y} class="stroke-neutral-100 dark:stroke-neutral-900" />
				<text x={padX - 8} y={y + 4} text-anchor="end" class="fill-neutral-400 text-[11px]">{Math.round(max * ratio)}</text>
			{/each}
			{#if points.length > 1}<polyline points={polyline} fill="none" class="stroke-brand-500" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />{/if}
			{#each coords as point, index}
				<g role={onselect ? "button" : undefined} tabindex={onselect ? 0 : undefined}
					onclick={() => onselect?.(point.month)}
					onkeydown={(event) => { if (onselect && (event.key === "Enter" || event.key === " ")) onselect(point.month); }}
					class={onselect ? "cursor-pointer group" : ""}>
					<title>{monthLabel(point.month)}: {point.total} santri</title>
					<circle cx={point.x} cy={point.y} r="5" class="fill-white stroke-brand-500 dark:fill-neutral-950" stroke-width="3" />
					{#if index % labelEvery === 0 || index === coords.length - 1}
						<text x={point.x} y={height - 16} text-anchor="middle" class="fill-neutral-500 dark:fill-neutral-400 text-[11px]">{monthLabel(point.month)}</text>
					{/if}
				</g>
			{/each}
		</svg>
	</div>
{:else}
	<div class="flex min-h-56 items-center justify-center text-sm text-neutral-500 dark:text-neutral-400">{emptyText}</div>
{/if}
