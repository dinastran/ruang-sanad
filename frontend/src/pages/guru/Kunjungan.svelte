<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Kunjungan, User } from "@lib/types";
	import { ChevronRight, ClipboardCheck } from "lucide-svelte";

	interface Props { user?: User; kunjungan?: Kunjungan[]; success?: string; error?: string; }
	let { user, kunjungan = [], error }: Props = $props();

	const fmt = (n: number) => n.toFixed(2).replace(".", ",");
	const BADGE: Record<string, { label: string; cls: string }> = {
		terkirim: { label: "Baru", cls: "bg-brand-600 text-white" },
		dibaca: { label: "Belum ditanggapi", cls: "bg-amber-500/10 text-amber-700 dark:text-amber-400" },
		ditanggapi: { label: "Sudah ditanggapi", cls: "bg-green-500/10 text-green-700 dark:text-green-400" },
	};
</script>

<AppLayout {user} group="guru-kunjungan">
	<div class="max-w-3xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		<header>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Hasil Kunjungan</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">Penilaian dan catatan dari Koordinator Guru setelah kunjungan kelas.</p>
		</header>
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="space-y-3">
			{#each kunjungan as k (k.id)}
				<a href={`/app/guru/kunjungan/${k.id}`} use:inertia class="flex items-center justify-between gap-4 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 hover:border-brand-400/60 transition-colors">
					<div class="min-w-0">
						<div class="flex items-center gap-2 flex-wrap">
							<p class="font-semibold text-neutral-900 dark:text-white">{k.kelas_nama || "Kunjungan kelas"}</p>
							{#if BADGE[k.status_kirim]}<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium {BADGE[k.status_kirim].cls}">{BADGE[k.status_kirim].label}</span>{/if}
						</div>
						<p class="mt-1 text-sm text-neutral-500">{k.tanggal}{k.tindak_lanjut.length > 0 ? ` · ${k.tindak_lanjut.length} tindak lanjut` : ""}</p>
					</div>
					<div class="flex items-center gap-3 shrink-0">
						<div class="text-right"><p class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">{fmt(k.nilai_rata_rata)}</p><p class="text-xs text-neutral-500">{k.predikat}</p></div>
						<ChevronRight class="w-4 h-4 text-neutral-400" />
					</div>
				</a>
			{:else}
				<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><ClipboardCheck class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada hasil kunjungan.</div>
			{/each}
		</div>
	</div>
</AppLayout>
