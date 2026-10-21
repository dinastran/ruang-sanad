<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User, GuruKelas, PertemuanGuru, JadwalDibatalkan } from "@lib/types";
	import { nomorPertemuan } from "@lib/pertemuan";
	import { ArrowLeft, Ban, Calendar, ClipboardPen, Clock, BookOpen, RotateCcw, UserRoundCheck } from "lucide-svelte";

	type Filter = "semua" | "terlaksana" | "dibatalkan";
	type Entri =
		| { kind: "pertemuan"; key: string; tanggal: string; jam: string; data: PertemuanGuru }
		| { kind: "batal"; key: string; tanggal: string; jam: string; data: JadwalDibatalkan };

	interface Props {
		user?: User;
		kelas: GuruKelas;
		pertemuan: PertemuanGuru[];
		dibatalkan?: JadwalDibatalkan[];
	}

	let { user, kelas, pertemuan = [], dibatalkan = [] }: Props = $props();
	let filter = $state<Filter>("semua");

	// Satu linimasa per tanggal: pertemuan selesai dan jadwal yang dibatalkan.
	// Pembatalan tidak memakai nomor pertemuan.
	let entri = $derived.by(() => {
		const list: Entri[] = [
			...pertemuan.map((p): Entri => ({ kind: "pertemuan", key: "p" + p.id, tanggal: p.tanggal, jam: p.jam_mulai, data: p })),
			...dibatalkan.map((b): Entri => ({ kind: "batal", key: "b" + b.id, tanggal: b.tanggal, jam: b.jam_mulai, data: b })),
		];
		return list.sort((a, b) => (a.tanggal !== b.tanggal ? b.tanggal.localeCompare(a.tanggal) : b.jam.localeCompare(a.jam)));
	});
	let tampil = $derived(
		filter === "semua" ? entri : entri.filter((item) => (filter === "terlaksana" ? item.kind === "pertemuan" : item.kind === "batal")),
	);

	const filters: { value: Filter; label: string }[] = [
		{ value: "semua", label: "Semua" },
		{ value: "terlaksana", label: "Terlaksana" },
		{ value: "dibatalkan", label: "Dibatalkan" },
	];

	function count(value: Filter) {
		if (value === "terlaksana") return pertemuan.length;
		if (value === "dibatalkan") return dibatalkan.length;
		return pertemuan.length + dibatalkan.length;
	}

	function formatTanggal(value: string) {
		return new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date(value + "T00:00:00"));
	}

	function pembatal(item: JadwalDibatalkan) {
		if (!item.dibatalkan_oleh) return "";
		const peran = item.dibatalkan_oleh_role === "koordinator_guru" ? "Koordinator" : item.dibatalkan_oleh_role === "guru" ? "Guru" : "Admin";
		return `${item.dibatalkan_oleh} (${peran})`;
	}
</script>

<AppLayout {user}>
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<a href={"/app/guru/kelas/" + kelas.id} use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 mb-4">
				<ArrowLeft class="w-4 h-4" /> Kembali ke kelas
			</a>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Riwayat Pertemuan</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">
				{kelas.nama_kelas} · {pertemuan.length} pertemuan selesai{#if dibatalkan.length > 0} · <span class="text-red-600 dark:text-red-400">{dibatalkan.length} dibatalkan</span>{/if}
			</p>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8">
		{#if entri.length > 0}
			<div class="mb-5 flex flex-wrap gap-2" role="tablist" aria-label="Filter riwayat">
				{#each filters as f}
					<button
						role="tab"
						aria-selected={filter === f.value}
						onclick={() => (filter = f.value)}
						class="inline-flex min-h-10 items-center gap-2 rounded-xl border px-3.5 py-2 text-sm font-medium transition-colors {filter === f.value ? 'border-brand-400/40 bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'border-neutral-200/80 bg-white text-neutral-600 hover:border-neutral-300 dark:border-white/[0.06] dark:bg-neutral-925/50 dark:text-neutral-300'}"
					>
						{f.label}<span class="font-mono text-xs opacity-70">{count(f.value)}</span>
					</button>
				{/each}
			</div>
		{/if}

		{#if tampil.length > 0}
			<div class="space-y-3">
				{#each tampil as entry (entry.key)}
					{#if entry.kind === "pertemuan"}
						{@const item = entry.data}
						<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
							<div class="flex gap-4 justify-between flex-wrap">
								<div class="flex gap-3 min-w-0">
									<div class="w-10 h-10 shrink-0 rounded-xl bg-brand-400/10 flex items-center justify-center text-brand-600 dark:text-brand-400"><BookOpen class="w-5 h-5" /></div>
									<div>
										<h2 class="font-semibold text-neutral-900 dark:text-white">
											Pertemuan ke-{nomorPertemuan(item)}
											{#if item.level_nama}<span class="ml-1.5 rounded-md bg-brand-400/10 px-1.5 py-0.5 align-middle text-xs font-semibold text-brand-700 dark:text-brand-400">{item.level_nama}</span>{/if}
										</h2>
										<div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm text-neutral-500">
											<span class="inline-flex items-center gap-1"><Calendar class="w-3.5 h-3.5" />{formatTanggal(item.tanggal)}</span>
											<span class="inline-flex items-center gap-1"><Clock class="w-3.5 h-3.5" />{item.jam_mulai}{item.jam_selesai ? " - " + item.jam_selesai : ""}</span>
										</div>
									</div>
								</div>
								<span class="h-fit rounded-full bg-green-500/10 px-2.5 py-1 text-xs font-semibold text-green-700 dark:text-green-400">Selesai</span>
							</div>
							{#if item.materi}<p class="mt-4 text-sm text-neutral-700 dark:text-neutral-300"><span class="font-medium">Materi:</span> {item.materi}</p>{/if}
							{#if item.catatan}<p class="mt-2 text-sm text-neutral-500 dark:text-neutral-400">{item.catatan}</p>{/if}
							{#if item.is_reschedule || item.is_badal || item.is_susulan}
								<div class="mt-3 flex flex-wrap gap-2 text-xs">
									{#if item.is_susulan}<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-1 text-amber-700 dark:text-amber-400" title={item.alasan_susulan ? "Alasan: " + item.alasan_susulan : ""}><ClipboardPen class="w-3.5 h-3.5" /> Absen susulan{item.alasan_susulan ? " · " + item.alasan_susulan : ""}</span>{/if}
									{#if item.is_reschedule}<span class="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-1 text-blue-700 dark:text-blue-400"><RotateCcw class="w-3.5 h-3.5" /> Dijadwalkan ulang{item.jadwal_semula ? " dari " + item.jadwal_semula : ""}</span>{/if}
									{#if item.is_badal}<span class="inline-flex items-center gap-1 rounded-full bg-secondary-500/10 px-2.5 py-1 text-secondary-700 dark:text-secondary-400"><UserRoundCheck class="w-3.5 h-3.5" /> Dilaksanakan guru badal</span>{/if}
								</div>
							{/if}
						</div>
					{:else}
						{@const item = entry.data}
						<div class="rounded-2xl border border-dashed border-neutral-300 bg-neutral-50/70 p-5 dark:border-white/[0.08] dark:bg-neutral-900/30">
							<div class="flex gap-4 justify-between flex-wrap">
								<div class="flex gap-3 min-w-0">
									<div class="w-10 h-10 shrink-0 rounded-xl bg-red-500/10 flex items-center justify-center text-red-600 dark:text-red-400"><Ban class="w-5 h-5" /></div>
									<div class="min-w-0">
										<h2 class="font-semibold text-neutral-600 dark:text-neutral-300">Kelas dibatalkan</h2>
										<div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm text-neutral-500">
											<span class="inline-flex items-center gap-1"><Calendar class="w-3.5 h-3.5" />{formatTanggal(item.tanggal)}</span>
											<span class="inline-flex items-center gap-1"><Clock class="w-3.5 h-3.5" />{item.jam_mulai}</span>
										</div>
									</div>
								</div>
								<span class="h-fit rounded-full bg-red-500/10 px-2.5 py-1 text-xs font-semibold text-red-700 dark:text-red-400">Dibatalkan</span>
							</div>
							<p class="mt-4 text-sm text-neutral-700 dark:text-neutral-300"><span class="font-medium">Alasan:</span> {item.alasan_batal || "Tidak dicatat"}</p>
							{#if pembatal(item)}<p class="mt-1 text-xs text-neutral-500">Dibatalkan oleh {pembatal(item)}</p>{/if}
						</div>
					{/if}
				{/each}
			</div>
		{:else if entri.length > 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-10 text-center">
				<p class="text-sm text-neutral-500">Tidak ada riwayat {filter === "dibatalkan" ? "pembatalan" : "pertemuan terlaksana"} untuk kelas ini.</p>
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
