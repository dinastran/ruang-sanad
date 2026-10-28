<script lang="ts" module>
	export interface ChainLink {
		key: string;
		label: string;
		value: string;
		caption: string;
		/** Empty when the link is healthy. */
		attention: string;
		/** False when the source failed to load: never shown as healthy. */
		available: boolean;
		href: string;
	}
</script>

<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { focusRing, panel } from "./ui";

	let { links }: { links: ChainLink[] } = $props();

	let strained = $derived(links.filter((l) => l.available && l.attention).length);
	let missing = $derived(links.filter((l) => !l.available).length);
	let headline = $derived(
		missing
			? `${missing} mata rantai belum bisa dinilai`
			: strained
				? `${strained} dari ${links.length} mata rantai perlu perhatian`
				: `Kelima mata rantai dalam kondisi baik`,
	);
	const state = (l: ChainLink) => (!l.available ? "unknown" : l.attention ? "attention" : "ok");
	/** A joint is strained when either link it connects needs attention. */
	const jointStrained = (i: number) => state(links[i]) !== "ok" || state(links[i + 1]) !== "ok";
</script>

<section class={panel + " overflow-hidden"} aria-labelledby="rantai-title">
	<div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-5 pt-5 sm:px-6">
		<div>
			<h2 id="rantai-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Rantai lembaga</h2>
			<p class="mt-1 text-lg font-semibold tracking-[-0.01em] text-neutral-950 dark:text-white">{headline}</p>
		</div>
		<p class="text-xs text-neutral-500 dark:text-neutral-400">Santri dibina di kelas, oleh guru, lewat pertemuan, dan ditopang keuangan.</p>
	</div>

	<ol class="chain px-5 pb-5 pt-4 sm:px-6 sm:pb-6">
		{#each links as link, i (link.key)}
			{@const s = state(link)}
			<li class="contents">
				<a
					href={link.href}
					use:inertia
					class={"link group " + focusRing + " " + s}
					aria-label={`${link.label}: ${link.value}. ${link.caption}. ${s === "ok" ? "Kondisi baik" : s === "attention" ? link.attention : "Data belum tersedia"}`}
				>
					<span class="flex items-center justify-between gap-2">
						<span class="flex items-center gap-2 text-xs font-medium text-neutral-600 dark:text-neutral-300">
							<span class="font-mono text-[10px] text-neutral-400 dark:text-neutral-500">{String(i + 1).padStart(2, "0")}</span>{link.label}
						</span>
						<span class={"dot hidden lg:block " + s} aria-hidden="true"></span>
					</span>
					<span class="mt-2 block text-[28px] font-semibold leading-8 tracking-[-0.02em] tabular-nums text-neutral-950 dark:text-white">{link.value}</span>
					<span class="mt-1 block text-xs leading-snug text-neutral-500 dark:text-neutral-400">{link.caption}</span>
					<span class={"mt-1.5 block text-xs font-medium lg:mt-3 leading-snug " + (s === "ok" ? "text-emerald-700 dark:text-emerald-400" : s === "attention" ? "text-amber-700 dark:text-amber-400" : "text-neutral-500")}>
						{s === "ok" ? "Kondisi baik" : s === "attention" ? link.attention : "Data belum tersedia"}
					</span>
				</a>
				{#if i < links.length - 1}
					<span class={"joint " + (jointStrained(i) ? "strained" : "")} aria-hidden="true">
						<svg viewBox="0 0 28 14" fill="none"><rect x="1" y="2" width="15" height="10" rx="5" /><rect x="12" y="2" width="15" height="10" rx="5" /></svg>
					</span>
				{/if}
			</li>
		{/each}
	</ol>
</section>

<style>
	.chain {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 0;
	}
	@media (min-width: 1024px) {
		.chain {
			grid-template-columns: repeat(4, minmax(0, 1fr) 1.75rem) minmax(0, 1fr);
			align-items: stretch;
		}
	}
	.link {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		column-gap: 0.75rem;
		min-width: 0;
		border-radius: 0.75rem;
		border: 1px solid var(--color-neutral-200);
		padding: 0.75rem 0.875rem;
		background: var(--color-neutral-50);
		transition:
			border-color 150ms cubic-bezier(0.23, 1, 0.32, 1),
			background-color 150ms cubic-bezier(0.23, 1, 0.32, 1),
			transform 120ms cubic-bezier(0.23, 1, 0.32, 1);
	}
	.link:hover {
		border-color: var(--color-brand-600);
		background: var(--color-white);
	}
	.link:active {
		transform: scale(0.98);
	}
	.link.attention {
		border-color: color-mix(in oklab, var(--color-warning) 45%, transparent);
		background: color-mix(in oklab, var(--color-warning) 6%, var(--color-white));
	}
	.link.unknown {
		border-style: dashed;
	}
	:global(.dark) .link {
		border-color: rgba(255, 255, 255, 0.08);
		background: rgba(255, 255, 255, 0.02);
	}
	:global(.dark) .link:hover {
		border-color: var(--color-brand-500);
		background: rgba(255, 255, 255, 0.04);
	}
	:global(.dark) .link.attention {
		border-color: color-mix(in oklab, var(--color-warning) 40%, transparent);
		background: color-mix(in oklab, var(--color-warning) 7%, transparent);
	}
	.dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		background: var(--color-success);
		flex-shrink: 0;
	}
	.dot.attention {
		background: var(--color-warning);
		box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-warning) 20%, transparent);
	}
	.dot.unknown {
		background: transparent;
		border: 1.5px solid var(--color-neutral-400);
	}
	.joint {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 1.125rem;
		color: var(--color-neutral-300);
	}
	.joint svg {
		width: 1.5rem;
		height: 0.75rem;
		transform: rotate(90deg);
	}
	.joint rect {
		stroke: currentColor;
		stroke-width: 1.75;
	}
	.joint.strained {
		color: var(--color-warning);
	}
	.joint.strained rect:last-child {
		stroke-dasharray: 3 2.5;
	}
	:global(.dark) .joint {
		color: var(--color-neutral-700);
	}
	:global(.dark) .joint.strained {
		color: var(--color-warning);
	}
	/* Mobile: label and status on the left, figure on the right, one row each. */
	.link > :global(:nth-child(1)),
	.link > :global(:nth-child(3)),
	.link > :global(:nth-child(4)) {
		grid-column: 1;
	}
	.link > :global(:nth-child(2)) {
		grid-column: 2;
		grid-row: 1 / span 3;
		align-self: center;
		margin-top: 0;
	}
	@media (min-width: 1024px) {
		.link {
			display: block;
			padding: 0.875rem 1rem;
		}
		.link > :global(:nth-child(2)) {
			margin-top: 0.5rem;
		}
	}
	@media (min-width: 1024px) {
		.joint {
			height: auto;
		}
		.joint svg {
			width: 1.75rem;
			height: 0.875rem;
			transform: none;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.link {
			transition: none;
		}
		.link:active {
			transform: none;
		}
	}
</style>
