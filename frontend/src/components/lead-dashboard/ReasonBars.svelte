<script lang="ts">
	import { fmt, panel, type Count } from "./types";

	interface Props {
		title: string; subtitle: string; empty: string; items: Count[];
		label: (key: string) => string; tone?: "warning" | "neutral"; onselect?: (key: string) => void;
	}
	let { title, subtitle, empty, items, label, tone = "neutral", onselect }: Props = $props();

	let max = $derived(Math.max(1, ...items.map((x) => x.count)));
	let total = $derived(items.reduce((s, x) => s + x.count, 0));
	const fill = $derived(tone === "warning" ? "bg-warning/80" : "bg-neutral-400 dark:bg-neutral-500");
</script>

<section class={panel + " p-5 sm:p-6"}>
	<h2 class="text-base font-semibold text-neutral-950 dark:text-white">{title}</h2>
	<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">{subtitle}</p>
	{#if items.length === 0}
		<p class="py-8 text-center text-sm text-neutral-500 dark:text-neutral-400">{empty}</p>
	{:else}
		<ul class="mt-4 space-y-1">
			{#each items as x}
				{@const share = total ? Math.round(x.count / total * 100) : 0}
				<li>
					<svelte:element this={onselect ? "button" : "div"} type={onselect ? "button" : undefined} role={onselect ? undefined : "group"}
						onclick={onselect ? () => onselect(x.key) : undefined}
						class={"block w-full rounded-md px-2 py-1.5 text-left " + (onselect ? "transition-colors duration-150 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.03]" : "")}>
						<span class="flex items-baseline justify-between gap-3 text-sm">
							<span class="min-w-0 break-words text-neutral-700 first-letter:uppercase dark:text-neutral-300">{label(x.key)}</span>
							<span class="shrink-0 tabular-nums"><span class="font-semibold text-neutral-900 dark:text-white">{fmt(x.count)}</span> <span class="text-xs text-neutral-400 dark:text-neutral-500">{share}%</span></span>
						</span>
						<span class="mt-1.5 block h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]">
							<span class={"block h-full rounded-full " + fill} style={`width:${x.count / max * 100}%`}></span>
						</span>
					</svelte:element>
				</li>
			{/each}
		</ul>
	{/if}
</section>
