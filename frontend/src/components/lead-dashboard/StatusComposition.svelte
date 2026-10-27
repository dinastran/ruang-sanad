<script lang="ts">
	import { fmt, panel, pct, type Report } from "./types";

	interface Props { r: Report; onstatus: (status: string) => void }
	let { r, onstatus }: Props = $props();

	let items = $derived([
		{ key: "open", label: "Berjalan", count: r.open, bar: "bg-neutral-400 dark:bg-neutral-500" },
		{ key: "won", label: "Jadi santri", count: r.won, bar: "bg-success" },
		{ key: "lost", label: "Lost", count: r.lost, bar: "bg-warning" },
		{ key: "do_not_contact", label: "Jangan hubungi", count: r.do_not_contact, bar: "bg-error" },
	]);
</script>

<section class={panel + " p-5 sm:p-6"} aria-labelledby="status-title">
	<h2 id="status-title" class="text-base font-semibold text-neutral-950 dark:text-white">Status lead periode ini</h2>
	<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Posisi terkini {fmt(r.total_leads)} lead yang masuk dalam periode.</p>
	{#if r.total_leads === 0}
		<p class="py-8 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada lead pada periode ini.</p>
	{:else}
		<div class="mt-4 flex h-2.5 gap-0.5 overflow-hidden rounded-full" role="img" aria-label={items.map((i) => `${i.label} ${pct(i.count, r.total_leads)}`).join(", ")}>
			{#each items as i}
				{#if i.count > 0}<span class={"h-full first:rounded-l-full last:rounded-r-full " + i.bar} style={`width:${i.count / r.total_leads * 100}%`}></span>{/if}
			{/each}
		</div>
		<div class="mt-4 grid grid-cols-2 gap-1">
			{#each items as i}
				<button type="button" onclick={() => onstatus(i.key)} class="flex items-start gap-2 rounded-lg px-2 py-2 text-left transition-colors duration-150 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.03]">
					<span class={"mt-1.5 h-2 w-2 shrink-0 rounded-[2px] " + i.bar} aria-hidden="true"></span>
					<span class="min-w-0">
						<span class="block truncate text-xs text-neutral-600 dark:text-neutral-300">{i.label}</span>
						<span class="block text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(i.count)} <span class="text-xs font-normal text-neutral-500 dark:text-neutral-400">{pct(i.count, r.total_leads)}</span></span>
					</span>
				</button>
			{/each}
		</div>
	{/if}
</section>
