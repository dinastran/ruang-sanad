<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User, GuruKelas, PertemuanGuru } from "@lib/types";
	import { ArrowLeft, Calendar, Clock, BookOpen, RotateCcw, UserRoundCheck } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas: GuruKelas;
		pertemuan: PertemuanGuru[];
	}

	let { user, kelas, pertemuan = [] }: Props = $props();
</script>

<AppLayout {user}>
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<a href={"/app/guru/kelas/" + kelas.id} use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 mb-4">
				<ArrowLeft class="w-4 h-4" /> Kembali ke kelas
			</a>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Riwayat Pertemuan</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">{kelas.nama_kelas} · {pertemuan.length} pertemuan selesai</p>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8">
		{#if pertemuan.length > 0}
			<div class="space-y-3">
				{#each pertemuan as item}
					<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
						<div class="flex gap-4 justify-between flex-wrap">
							<div class="flex gap-3 min-w-0">
								<div class="w-10 h-10 shrink-0 rounded-xl bg-brand-400/10 flex items-center justify-center text-brand-600 dark:text-brand-400"><BookOpen class="w-5 h-5" /></div>
								<div>
									<h2 class="font-semibold text-neutral-900 dark:text-white">Pertemuan ke-{item.pertemuan_ke}</h2>
									<div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm text-neutral-500">
										<span class="inline-flex items-center gap-1"><Calendar class="w-3.5 h-3.5" />{item.tanggal}</span>
										<span class="inline-flex items-center gap-1"><Clock class="w-3.5 h-3.5" />{item.jam_mulai}{item.jam_selesai ? " - " + item.jam_selesai : ""}</span>
									</div>
								</div>
							</div>
							<span class="rounded-full bg-green-500/10 px-2.5 py-1 text-xs font-semibold text-green-700 dark:text-green-400">Selesai</span>
						</div>
						{#if item.materi}<p class="mt-4 text-sm text-neutral-700 dark:text-neutral-300"><span class="font-medium">Materi:</span> {item.materi}</p>{/if}
						{#if item.catatan}<p class="mt-2 text-sm text-neutral-500 dark:text-neutral-400">{item.catatan}</p>{/if}
						{#if item.is_reschedule || item.is_badal}
							<div class="mt-3 flex flex-wrap gap-2 text-xs">
								{#if item.is_reschedule}<span class="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-1 text-blue-700 dark:text-blue-400"><RotateCcw class="w-3.5 h-3.5" /> Dijadwalkan ulang{item.jadwal_semula ? " dari " + item.jadwal_semula : ""}</span>{/if}
								{#if item.is_badal}<span class="inline-flex items-center gap-1 rounded-full bg-secondary-500/10 px-2.5 py-1 text-secondary-700 dark:text-secondary-400"><UserRoundCheck class="w-3.5 h-3.5" /> Dilaksanakan guru badal</span>{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{:else}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center">
				<Calendar class="w-8 h-8 mx-auto text-neutral-400" />
				<h2 class="mt-4 font-semibold text-neutral-900 dark:text-white">Belum ada pertemuan</h2>
				<p class="mt-1 text-sm text-neutral-500">Riwayat akan muncul setelah pertemuan pertama selesai dicatat.</p>
			</div>
		{/if}
	</div>
</AppLayout>
