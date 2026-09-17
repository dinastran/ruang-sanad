<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruRingkas, TsiRekapRow, User } from "@lib/types";
	import { Award, ArrowRight, TrendingUp } from "lucide-svelte";

	interface Props { user?: User; bulan: string; rekap?: TsiRekapRow[]; guruList?: GuruRingkas[]; success?: string; error?: string; }
	let { user, bulan, rekap = [], guruList = [], success, error }: Props = $props();

	let selectedBulan = $state(bulan);
	let selectedGuru = $state<number | "">("");

	function changeBulan() { router.get("/app/koordinator-guru/tsi", { bulan: selectedBulan }, { preserveState: false }); }
	function openGuru() { if (selectedGuru) router.get(`/app/koordinator-guru/tsi/${selectedGuru}`, { bulan: selectedBulan }); }

	const tetap = $derived(rekap.filter((r) => r.guru_status === "tetap"));
	const partTime = $derived(rekap.filter((r) => r.guru_status !== "tetap"));

	function predColor(p: string): string {
		if (p === "Sangat Baik") return "bg-green-500/10 text-green-700 dark:text-green-400";
		if (p === "Baik") return "bg-blue-500/10 text-blue-700 dark:text-blue-400";
		if (p === "Cukup") return "bg-amber-500/10 text-amber-700 dark:text-amber-400";
		return "bg-red-500/10 text-red-600 dark:text-red-400";
	}
</script>

{#snippet block(title: string, rows: TsiRekapRow[])}
	{#if rows.length}
		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
			<div class="px-5 py-3 border-b border-neutral-200/80 dark:border-white/[0.04] bg-neutral-50 dark:bg-neutral-900/50"><h2 class="text-sm font-semibold text-neutral-700 dark:text-neutral-300">{title}</h2></div>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead><tr class="text-xs text-neutral-500 border-b border-neutral-200/80 dark:border-white/[0.04]">
						<th class="text-left font-semibold px-4 py-2.5">Guru</th>
						<th class="text-right font-semibold px-3 py-2.5">Komp<span class="font-normal">/40</span></th>
						<th class="text-right font-semibold px-3 py-2.5">Kepuasan<span class="font-normal">/30</span></th>
						<th class="text-right font-semibold px-3 py-2.5">Disiplin<span class="font-normal">/20</span></th>
						<th class="text-right font-semibold px-3 py-2.5">Kontrib<span class="font-normal">/10</span></th>
						<th class="text-right font-semibold px-3 py-2.5">Total</th>
						<th class="text-left font-semibold px-3 py-2.5">Predikat</th>
						<th class="px-3 py-2.5"></th>
					</tr></thead>
					<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
						{#each rows as r (r.guru_id)}
							<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015]">
								<td class="px-4 py-2.5 font-medium text-neutral-900 dark:text-white">{r.guru_nama}{#if r.status === "final"}<span class="ml-1.5 text-[10px] px-1.5 py-0.5 rounded bg-neutral-500/10 text-neutral-500">final</span>{/if}</td>
								<td class="px-3 py-2.5 text-right font-mono text-neutral-600 dark:text-neutral-400">{r.kompetensi.toFixed(1)}</td>
								<td class="px-3 py-2.5 text-right font-mono text-neutral-600 dark:text-neutral-400">{r.kepuasan.toFixed(1)}</td>
								<td class="px-3 py-2.5 text-right font-mono text-neutral-600 dark:text-neutral-400">{r.kedisiplinan.toFixed(1)}</td>
								<td class="px-3 py-2.5 text-right font-mono text-neutral-600 dark:text-neutral-400">{r.kontribusi.toFixed(1)}</td>
								<td class="px-3 py-2.5 text-right font-mono font-bold text-neutral-900 dark:text-white">{r.total.toFixed(1)}</td>
								<td class="px-3 py-2.5"><span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium {predColor(r.predikat)}">{r.predikat}</span></td>
								<td class="px-3 py-2.5 text-right"><a href={`/app/koordinator-guru/tsi/${r.guru_id}?bulan=${bulan}`} use:inertia class="inline-flex text-brand-600 hover:text-brand-700"><ArrowRight class="w-4 h-4" /></a></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
{/snippet}

<AppLayout {user} group="koordinator-tsi">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600">Koordinator Guru</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">Penilaian TSI</span></div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Rekap Penilaian TSI</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">Skor 4 kategori (bobot 40/30/20/10) per guru per bulan. Indikator kosong dikecualikan dari rata-rata.</p>
		</div>
	</div>

	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="flex flex-col sm:flex-row gap-3 sm:items-end">
			<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Bulan<input type="month" bind:value={selectedBulan} onchange={changeBulan} class="mt-1.5 block px-3.5 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" /></label>
			<div class="flex-1"></div>
			<div class="flex gap-2 items-end">
				<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Nilai guru<select bind:value={selectedGuru} class="mt-1.5 block w-56 px-3.5 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400"><option value="">Pilih guru...</option>{#each guruList as g (g.id)}<option value={g.id}>{g.nama}</option>{/each}</select></label>
				<button onclick={openGuru} disabled={!selectedGuru} class="px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50">Buka</button>
			</div>
		</div>

		{#if rekap.length === 0}
			<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><Award class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada penilaian untuk {bulan}. Pilih guru di atas untuk memulai.</div>
		{:else}
			{@render block("Guru Tetap", tetap)}
			{@render block("Guru Part Time", partTime)}
		{/if}

		<p class="text-xs text-neutral-500 flex items-center gap-1.5"><TrendingUp class="w-3.5 h-3.5" /> Rekap hanya menampilkan guru yang periodenya sudah dibuat pada bulan ini.</p>
	</div>
</AppLayout>
