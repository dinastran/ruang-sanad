<script lang="ts">
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { TsiPenilaian, TsiKriteriaNilai, User } from "@lib/types";
	import { Award, Bot, Hand } from "lucide-svelte";

	interface Props { user?: User; penilaian?: TsiPenilaian; error?: string; }
	let { user, penilaian, error }: Props = $props();

	const KAT_LABEL: Record<string, string> = { kompetensi: "Kompetensi Mengajar", kepuasan: "Kepuasan Murid", kedisiplinan: "Kedisiplinan", kontribusi: "Kontribusi Program" };
	function byKat(kat: string): TsiKriteriaNilai[] { return penilaian ? penilaian.kriteria.filter((k) => k.kategori === kat) : []; }
	function fmt(n: number | null): string { return n === null ? "—" : n.toFixed(1); }
	function predColor(p: string): string {
		if (p === "Sangat Baik") return "text-green-600 dark:text-green-400";
		if (p === "Baik") return "text-blue-600 dark:text-blue-400";
		if (p === "Cukup") return "text-amber-600 dark:text-amber-400";
		return "text-red-600 dark:text-red-400";
	}
</script>

<AppLayout {user} group="guru-tsi">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Nilai TSI Saya</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">Rincian penilaian kinerja Anda beserta angka mentah tiap indikator otomatis.</p>
		</div>
	</div>

	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		{#if penilaian}
			<div class="rounded-2xl border border-brand-400/20 bg-brand-400/5 p-6 flex items-center justify-between gap-4 flex-wrap" in:fly={{ y: 10, duration: 300 }}>
				<div class="flex items-center gap-3"><div class="w-12 h-12 rounded-xl bg-brand-400/15 flex items-center justify-center"><Award class="w-6 h-6 text-brand-600 dark:text-brand-400" /></div><div><p class="text-sm text-neutral-600 dark:text-neutral-400">Total TSI — {penilaian.bulan}</p><p class="text-3xl font-bold text-neutral-900 dark:text-white font-mono">{penilaian.total.toFixed(1)}</p></div></div>
				<div class="text-right"><p class="text-sm text-neutral-500">Predikat</p><p class="text-lg font-bold {predColor(penilaian.predikat)}">{penilaian.predikat}</p></div>
			</div>

			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
				{#each penilaian.kategori as kat (kat.kategori)}
					<div class="rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4"><p class="text-xs text-neutral-500">{KAT_LABEL[kat.kategori]}</p><p class="mt-1 text-xl font-bold text-neutral-900 dark:text-white font-mono">{kat.skor.toFixed(1)}<span class="text-xs text-neutral-400">/{kat.bobot}</span></p></div>
				{/each}
			</div>

			{#each penilaian.kategori as kat (kat.kategori)}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
					<div class="px-5 py-3 border-b border-neutral-200/80 dark:border-white/[0.04] bg-neutral-50 dark:bg-neutral-900/50"><h2 class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">{KAT_LABEL[kat.kategori]}</h2></div>
					<div class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
						{#each byKat(kat.kategori) as k (k.kriteria_id)}
							<div class="flex items-start gap-2.5 p-4">
								{#if k.sumber === "auto"}<Bot class="w-4 h-4 text-brand-500 mt-0.5 shrink-0" />{:else}<Hand class="w-4 h-4 text-neutral-400 mt-0.5 shrink-0" />{/if}
								<div class="min-w-0 flex-1"><p class="text-sm text-neutral-800 dark:text-neutral-200">{k.nama}</p>{#if k.sumber === "auto" && k.raw_display}<p class="text-xs text-neutral-500 mt-0.5">{k.raw_display}</p>{/if}</div>
								<span class="text-sm font-mono font-semibold text-neutral-900 dark:text-white shrink-0">{fmt(k.nilai)}</span>
							</div>
						{/each}
					</div>
				</div>
			{/each}

			<p class="text-xs text-neutral-500">Indikator kosong (—) tidak dihitung dalam rata-rata. Anda tidak dapat mengubah nilai; hubungi Koordinator Guru untuk klarifikasi.</p>
		{:else}
			<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><Award class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada penilaian untuk bulan ini.</div>
		{/if}
	</div>
</AppLayout>
