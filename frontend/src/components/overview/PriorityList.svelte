<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { ArrowUpRight, CircleCheck } from "lucide-svelte";
	import type { Overview } from "@lib/overview";
	import { count, focusRing, kicker, panel } from "./ui";

	let { priorities, incomplete }: { priorities: Overview["priorities"]; incomplete: boolean } = $props();
</script>

<section class={panel + " flex min-w-0 flex-col"} aria-labelledby="prioritas-title">
	<div class="flex items-start justify-between gap-3 px-5 pt-5 sm:px-6">
		<div>
			<h2 id="prioritas-title" class={kicker}>Yang perlu ditangani</h2>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">Urut dari operasional kelas, lalu tindak lanjut pendukung.</p>
		</div>
		{#if priorities.length}
			<span class="shrink-0 rounded-md bg-amber-500/10 px-2 py-1 text-xs font-semibold tabular-nums text-amber-700 dark:text-amber-400">{priorities.length} area</span>
		{/if}
	</div>

	{#if priorities.length}
		<ol class="mt-3 divide-y divide-neutral-100 border-t border-neutral-100 dark:divide-white/[0.05] dark:border-white/[0.05]">
			{#each priorities as item, i (item.title)}
				<li>
					<a href={item.href} use:inertia class={"group grid grid-cols-[1.75rem_minmax(0,1fr)_auto] items-start gap-3 px-5 py-3.5 transition-colors duration-150 hover:bg-neutral-50 sm:px-6 dark:hover:bg-white/[0.03] " + focusRing + " focus-visible:ring-inset"}>
						<span class="pt-0.5 font-mono text-[11px] text-neutral-400 dark:text-neutral-500">{String(i + 1).padStart(2, "0")}</span>
						<span class="min-w-0">
							<span class="block text-sm font-semibold text-neutral-900 dark:text-white">{item.title}</span>
							<span class="mt-0.5 block text-xs leading-relaxed text-neutral-500 [text-wrap:pretty] dark:text-neutral-400">{item.detail}</span>
						</span>
						<span class="flex items-center gap-1 pt-px">
							<span class="text-xl font-semibold tabular-nums text-neutral-950 dark:text-white">{count(item.count)}</span>
							<ArrowUpRight size={14} class="text-neutral-400 transition-colors group-hover:text-brand-600 dark:group-hover:text-brand-400" />
						</span>
					</a>
				</li>
			{/each}
		</ol>
	{:else}
		<div class="flex flex-1 flex-col items-start justify-center gap-2 px-5 py-8 sm:px-6">
			<CircleCheck size={20} class={incomplete ? "text-neutral-400" : "text-emerald-600 dark:text-emerald-400"} />
			<p class="text-sm font-semibold text-neutral-900 dark:text-white">{incomplete ? "Daftar belum bisa dinilai lengkap" : "Tidak ada penanda yang perlu ditangani"}</p>
			<p class="max-w-md text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
				{incomplete ? "Sebagian sumber data gagal dimuat. Coba perbarui data." : "Tetap tinjau riayah, pencatatan kelas, dan laporan tim secara berkala."}
			</p>
		</div>
	{/if}
</section>
