<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { fmt, panel, pct, type SourceStat } from "./types";

	interface Props { sources: SourceStat[]; onselect: (sumber: string) => void }
	let { sources, onselect }: Props = $props();

	let max = $derived(Math.max(1, ...sources.map((s) => s.leads)));
	let total = $derived(sources.reduce((sum, s) => sum + s.leads, 0));
	let untracked = $derived(sources.find((s) => s.id === 0)?.leads ?? 0);
</script>

<section class={panel + " p-5 sm:p-6"} aria-labelledby="sumber-title">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h2 id="sumber-title" class="text-base font-semibold text-neutral-950 dark:text-white">Sumber lead</h2>
			<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">LP atau iklan asal lead, dikenali dari teks pesan pertama. Lead masuk periode ini dan status terkininya.</p>
		</div>
		<a href="/app/leads/sumber" use:inertia class="rounded text-xs font-medium text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">Atur teks pemicu</a>
	</div>
	{#if total === 0}
		<p class="py-8 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada lead pada periode ini.</p>
	{:else}
		<div class="mt-4 overflow-x-auto">
			<table class="w-full min-w-[520px] text-sm">
				<thead>
					<tr class="text-left text-[11px] font-medium text-neutral-500 dark:text-neutral-400">
						<th scope="col" class="pb-2 pr-3 font-medium">Sumber</th>
						<th scope="col" class="pb-2 px-3 text-right font-medium">Lead</th>
						<th scope="col" class="pb-2 px-3 text-right font-medium">Jadi santri</th>
						<th scope="col" class="pb-2 pl-3 text-right font-medium">Konversi</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
					{#each sources as s (s.id)}
						<tr>
							<th scope="row" class="py-2 pr-3 text-left font-normal">
								<button type="button" onclick={() => onselect(s.id ? String(s.id) : "none")} class="block w-full rounded text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">
									<span class={"block truncate " + (s.id ? "font-medium text-neutral-900 hover:text-brand-700 dark:text-white dark:hover:text-brand-400" : "text-neutral-500 dark:text-neutral-400")}>{s.nama}</span>
									<span class="mt-1 block h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]">
										<span class={"block h-full rounded-full " + (s.id ? "bg-brand-600 dark:bg-brand-500" : "bg-neutral-300 dark:bg-neutral-600")} style={`width:${s.leads / max * 100}%`}></span>
									</span>
								</button>
							</th>
							<td class="px-3 py-2 text-right tabular-nums"><span class="font-semibold text-neutral-900 dark:text-white">{fmt(s.leads)}</span> <span class="text-xs text-neutral-400 dark:text-neutral-500">{pct(s.leads, total)}</span></td>
							<td class="px-3 py-2 text-right font-semibold tabular-nums text-neutral-900 dark:text-white">{fmt(s.won)}</td>
							<td class="pl-3 py-2 text-right tabular-nums text-neutral-700 dark:text-neutral-300">{pct(s.won, s.leads)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if untracked > 0}
			<p class="mt-3 text-xs text-neutral-500 dark:text-neutral-400">{fmt(untracked)} lead tanpa sumber: pesan pertamanya tidak cocok dengan teks pemicu mana pun. Tambahkan teks pemicu di halaman sumber, lalu cocokkan ulang.</p>
		{/if}
	{/if}
</section>
