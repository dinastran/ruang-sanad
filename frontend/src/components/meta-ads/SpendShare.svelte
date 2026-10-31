<script lang="ts">
	import { pct, ratio, rp, rpShort, type Entity } from "./data";

	interface Props { campaigns: Entity[]; onselect: (id: string) => void }
	let { campaigns, onselect }: Props = $props();

	// Top 4 campaigns + "lainnya": a stacked bar reads better than a pie.
	const tones = ["bg-brand-700 dark:bg-brand-400", "bg-brand-500 dark:bg-brand-600", "bg-brand-300 dark:bg-brand-700", "bg-brand-200 dark:bg-brand-800", "bg-neutral-300 dark:bg-neutral-600"];
	let total = $derived(campaigns.reduce((s, c) => s + c.spend, 0));
	let parts = $derived.by(() => {
		const sorted = [...campaigns].sort((a, b) => b.spend - a.spend);
		const top = sorted.slice(0, 4).map((c) => ({ id: c.id, name: c.name, spend: c.spend, chats: c.chats, other: false }));
		const rest = sorted.slice(4);
		if (rest.length) top.push({ id: "", name: `${rest.length} kampanye lain`, spend: rest.reduce((s, c) => s + c.spend, 0), chats: rest.reduce((s, c) => s + c.chats, 0), other: true });
		return top;
	});
</script>

<section class="min-w-0 rounded-xl border border-neutral-200/80 bg-white p-5 sm:p-6 dark:border-white/[0.07] dark:bg-neutral-925/60" aria-labelledby="porsi-title">
	<h2 id="porsi-title" class="text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400">Porsi biaya per kampanye</h2>
	{#if total === 0}
		<p class="py-10 text-sm text-neutral-500 dark:text-neutral-400">Belum ada biaya pada periode dan filter ini.</p>
	{:else}
		<div class="mt-4 flex h-3 overflow-hidden rounded-full" role="img" aria-label="Porsi biaya per kampanye">
			{#each parts as p, i}<span class={"h-full " + tones[p.other ? 4 : i]} style:width={`${(p.spend / total) * 100}%`} title={`${p.name}: ${rp(p.spend)}`}></span>{/each}
		</div>
		<ul class="mt-4 space-y-2.5">
			{#each parts as p, i}
				<li>
					<svelte:element this={p.other ? "div" : "button"} type={p.other ? undefined : "button"} role={p.other ? "group" : undefined} onclick={p.other ? undefined : () => onselect(p.id)}
						class={"grid w-full grid-cols-[0.625rem_minmax(0,1fr)_auto] items-baseline gap-x-2.5 rounded text-left text-sm " + (p.other ? "" : "hover:text-brand-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:text-brand-400")}>
						<i class={"h-2.5 w-2.5 self-center rounded-sm " + tones[p.other ? 4 : i]}></i>
						<span class="truncate text-neutral-700 dark:text-neutral-300">{p.name}</span>
						<span class="tabular-nums"><strong class="font-semibold text-neutral-900 dark:text-white">{pct(p.spend / total, 0)}</strong> <span class="text-xs text-neutral-500">{rpShort(p.spend)}</span></span>
						<span></span>
						<span class="col-span-2 text-xs tabular-nums text-neutral-500 dark:text-neutral-400">{rp(ratio(p.spend, p.chats))}/chat WA</span>
					</svelte:element>
				</li>
			{/each}
		</ul>
	{/if}
</section>
