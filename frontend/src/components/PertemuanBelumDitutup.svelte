<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import BatalPertemuanDialog from "@components/BatalPertemuanDialog.svelte";
	import type { PertemuanBelumDitutup } from "@lib/types";
	import { AlertTriangle, CheckSquare, XCircle } from "lucide-svelte";

	// Daftar pertemuan yang masih berstatus berlangsung, dengan aksi
	// Selesaikan (isi absensi) dan Batalkan (salah klik Mulai).
	interface Props { items?: PertemuanBelumDitutup[]; showGuru?: boolean; }
	let { items = [], showGuru = false }: Props = $props();

	let target = $state<PertemuanBelumDitutup | null>(null);

	function durasi(jam: number): string {
		if (jam < 1) return "kurang dari 1 jam";
		if (jam < 24) return `${jam} jam`;
		return `${Math.floor(jam / 24)} hari ${jam % 24} jam`;
	}
</script>

{#if items.length > 0}
	<section class="rounded-2xl border border-amber-500/30 bg-amber-500/[0.04] p-4 sm:p-5">
		<div class="flex items-start gap-3">
			<AlertTriangle class="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
			<div>
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Pertemuan belum ditutup ({items.length})</h2>
				<p class="mt-0.5 text-xs leading-5 text-neutral-600 dark:text-neutral-400">Kelas tidak bisa memulai pertemuan baru sebelum pertemuan ini ditutup. Pilih <strong>Selesaikan</strong> bila kelas benar berlangsung (isi absensi), atau <strong>Batalkan</strong> bila salah klik Mulai.</p>
			</div>
		</div>
		<ul class="mt-3 divide-y divide-amber-500/15">
			{#each items as item (item.id)}
				<li class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<p class="text-sm font-semibold text-neutral-900 dark:text-white">{item.kelas_nama}</p>
							{#if item.lupa}<span class="rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-semibold text-red-600 dark:text-red-400">Lupa ditutup</span>{/if}
						</div>
						<p class="mt-0.5 text-xs text-neutral-600 dark:text-neutral-400">
							Pertemuan ke-{item.pertemuan_ke} · mulai {item.tanggal} pukul {item.jam_mulai} · berjalan {durasi(item.durasi_jam)}
							{#if showGuru} · {item.guru_pengganti_nama ? `${item.guru_pengganti_nama} (badal)` : item.guru_nama || "Belum ada guru"}{/if}
						</p>
					</div>
					<div class="flex shrink-0 gap-2">
						<a href={`/app/guru/kelas/${item.kelas_id}/pertemuan/${item.id}/selesai`} use:inertia class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-xl bg-amber-600 px-3.5 py-2 text-sm font-semibold text-white hover:bg-amber-700 sm:flex-none"><CheckSquare class="h-4 w-4" /> Selesaikan</a>
						<button onclick={() => (target = item)} class="inline-flex flex-1 items-center justify-center gap-1.5 rounded-xl border border-neutral-300 px-3.5 py-2 text-sm font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800 sm:flex-none"><XCircle class="h-4 w-4" /> Batalkan</button>
					</div>
				</li>
			{/each}
		</ul>
	</section>
{/if}

{#if target}
	<BatalPertemuanDialog kelasId={target.kelas_id} pertemuanId={target.id} label={`${target.kelas_nama} · mulai ${target.tanggal} pukul ${target.jam_mulai}`} onclose={() => (target = null)} />
{/if}
