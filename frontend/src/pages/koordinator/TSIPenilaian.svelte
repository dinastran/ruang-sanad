<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { TsiPenilaian, TsiKriteriaNilai, User } from "@lib/types";
	import { ArrowLeft, Save, Lock, Unlock, Bot, Hand } from "lucide-svelte";

	interface Props { user?: User; penilaian: TsiPenilaian; success?: string; error?: string; }
	let { user, penilaian, success, error }: Props = $props();

	const KAT_LABEL: Record<string, string> = { kompetensi: "Kompetensi Mengajar", kepuasan: "Kepuasan Murid", kedisiplinan: "Kedisiplinan", kontribusi: "Kontribusi Program" };

	type Editable = { kriteria_id: number; sumber: string; override: boolean; nilai: string | number | null; alasan: string; catatan: string };
	let rows = $state<Record<number, Editable>>(
		Object.fromEntries(penilaian.kriteria.map((k) => [k.kriteria_id, {
			kriteria_id: k.kriteria_id,
			sumber: k.sumber,
			override: k.is_override,
			nilai: k.is_override ? (k.nilai ?? "").toString() : (k.sumber === "auto" ? "" : (k.nilai ?? "").toString()),
			alasan: k.alasan_override,
			catatan: k.catatan,
		}])),
	);
	let saving = $state(false);
	const isFinal = $derived(penilaian.status === "final");

	function byKat(kat: string): TsiKriteriaNilai[] { return penilaian.kriteria.filter((k) => k.kategori === kat); }

	function parseNilai(v: unknown): number | null {
		const s = String(v ?? "").trim();
		if (s === "") return null;
		const n = Number(s);
		return Number.isNaN(n) ? null : n;
	}
	function payload() {
		return penilaian.kriteria.map((k) => {
			const e = rows[k.kriteria_id];
			const isAuto = k.sumber === "auto";
			if (isAuto && !e.override) return { kriteria_id: k.kriteria_id, nilai: null, is_override: false, alasan_override: "", catatan: e.catatan };
			return { kriteria_id: k.kriteria_id, nilai: parseNilai(e.nilai), is_override: isAuto ? e.override : false, alasan_override: e.alasan, catatan: e.catatan };
		});
	}
	function save() {
		saving = true;
		router.post(`/app/koordinator-guru/tsi/${penilaian.guru_id}?bulan=${penilaian.bulan}`, { nilai: payload() }, { preserveScroll: true, onFinish: () => (saving = false) });
	}
	function finalize() { if (confirm("Finalisasi periode ini? Nilai tidak bisa diubah sampai dibuka kembali.")) router.post(`/app/koordinator-guru/tsi/${penilaian.guru_id}/finalize?bulan=${penilaian.bulan}`); }
	function reopen() { router.post(`/app/koordinator-guru/tsi/${penilaian.guru_id}/reopen?bulan=${penilaian.bulan}`); }

	const inputClass = "px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	function fmt(n: number | null): string { return n === null ? "—" : n.toFixed(1); }
</script>

<AppLayout {user} group="koordinator-tsi">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<a href={`/app/koordinator-guru/tsi?bulan=${penilaian.bulan}`} use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 mb-4"><ArrowLeft class="w-4 h-4" /> Rekap TSI</a>
			<div class="flex items-end justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl font-bold text-neutral-900 dark:text-white">{penilaian.guru_nama}</h1>
					<p class="mt-1 text-neutral-600 dark:text-neutral-400">Penilaian TSI — {penilaian.bulan} · <span class="{isFinal ? 'text-neutral-500' : 'text-amber-600 dark:text-amber-400'} font-medium">{isFinal ? "Final" : "Draft"}</span></p>
				</div>
				<div class="text-right">
					<p class="text-3xl font-bold text-neutral-900 dark:text-white font-mono">{penilaian.total.toFixed(1)}</p>
					<p class="text-sm text-neutral-500">{penilaian.predikat}</p>
				</div>
			</div>
		</div>
	</div>

	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		{#if isFinal}
			<div class="rounded-xl bg-neutral-500/10 border border-neutral-500/20 p-4 flex items-center justify-between gap-3">
				<p class="text-sm text-neutral-600 dark:text-neutral-400 flex items-center gap-2"><Lock class="w-4 h-4" /> Periode ini sudah difinalisasi.</p>
				<button onclick={reopen} class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-700 text-sm font-semibold"><Unlock class="w-4 h-4" /> Buka kembali</button>
			</div>
		{/if}

		{#each penilaian.kategori as kat (kat.kategori)}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
				<div class="flex items-center justify-between px-5 py-3.5 border-b border-neutral-200/80 dark:border-white/[0.04] bg-neutral-50 dark:bg-neutral-900/50">
					<h2 class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">{KAT_LABEL[kat.kategori]} <span class="text-neutral-400 font-normal">· bobot {kat.bobot}%</span></h2>
					<p class="text-sm font-mono text-neutral-600 dark:text-neutral-400">Skor: <span class="font-bold text-neutral-900 dark:text-white">{kat.skor.toFixed(1)}</span> <span class="text-xs">({kat.terisi}/{kat.total} terisi)</span></p>
				</div>
				<div class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
					{#each byKat(kat.kategori) as k (k.kriteria_id)}
						<div class="p-4 space-y-2.5">
							<div class="flex items-start gap-2">
								{#if k.sumber === "auto"}<Bot class="w-4 h-4 text-brand-500 mt-0.5 shrink-0" title="Otomatis" />{:else}<Hand class="w-4 h-4 text-neutral-400 mt-0.5 shrink-0" title={k.sumber} />{/if}
								<div class="min-w-0 flex-1">
									<p class="text-sm text-neutral-800 dark:text-neutral-200">{k.nama}</p>
									{#if k.sumber === "auto"}
										<p class="text-xs text-neutral-500 mt-0.5">Otomatis: <span class="font-mono font-semibold text-neutral-700 dark:text-neutral-300">{fmt(k.auto_nilai)}%</span> — {k.raw_display || "belum ada data"}</p>
									{/if}
								</div>
								<div class="text-right shrink-0"><span class="text-sm font-mono font-semibold text-neutral-900 dark:text-white">{fmt(k.nilai)}</span></div>
							</div>

							{#if !isFinal}
								<div class="pl-6 space-y-2">
									{#if k.sumber === "auto"}
										<label class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400"><input type="checkbox" bind:checked={rows[k.kriteria_id].override} class="rounded" /> Override nilai otomatis</label>
										{#if rows[k.kriteria_id].override}
											<div class="flex flex-col sm:flex-row gap-2">
												<input type="number" min="0" max="100" step="0.1" bind:value={rows[k.kriteria_id].nilai} placeholder="Nilai %" class="{inputClass} w-28" />
												<input bind:value={rows[k.kriteria_id].alasan} placeholder="Alasan override (wajib)" class="{inputClass} flex-1" />
											</div>
										{/if}
									{:else}
										<div class="flex flex-col sm:flex-row gap-2">
											<input type="number" min="0" max="100" step="0.1" bind:value={rows[k.kriteria_id].nilai} placeholder="Nilai % (kosongkan bila belum dinilai)" class="{inputClass} w-full sm:w-48" />
											<input bind:value={rows[k.kriteria_id].catatan} placeholder="Catatan (opsional)" class="{inputClass} flex-1" />
										</div>
									{/if}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/each}

		{#if !isFinal}
			<div class="flex justify-end gap-3">
				<button onclick={save} disabled={saving} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Save class="w-4 h-4" /> {saving ? "Menyimpan..." : "Simpan nilai"}</button>
				<button onclick={finalize} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-900 text-sm font-semibold"><Lock class="w-4 h-4" /> Finalisasi</button>
			</div>
		{/if}
	</div>
</AppLayout>
