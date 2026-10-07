<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { RiwayatMengajarItem, RiwayatMengajarSummary, User } from "@lib/types";
	import {
		AlertCircle,
		BookOpen,
		CalendarDays,
		CheckCircle,
		Clock,
		History,
		RotateCcw,
		Search,
		Users,
	} from "lucide-svelte";

	interface Props {
		user?: User;
		items?: RiwayatMengajarItem[];
		summary?: RiwayatMengajarSummary;
		range?: string;
		start_date?: string;
		end_date?: string;
	}

	let {
		user,
		items = [],
		summary = { total_dimulai: 0, selesai: 0, belum_selesai: 0, kelas_diajar: 0 },
		range = "30",
		start_date = "",
		end_date = "",
	}: Props = $props();

	let searchQuery = $state("");
	let selectedClass = $state("");
	let selectedStatus = $state("");
	let showCustom = $state(range === "custom");
	let isAllGuruView = $derived(user?.role === "super_admin" || user?.role === "admin_kelas");

	let classOptions = $derived(
		Array.from(new Map(items.map((item) => [String(item.kelas_id), item.kelas_nama])).entries())
			.map(([id, nama]) => ({ id, nama }))
			.sort((a, b) => a.nama.localeCompare(b.nama)),
	);

	let baseFilteredItems = $derived(
		items.filter((item) => {
			const query = searchQuery.trim().toLowerCase();
			const matchesSearch =
				query === "" ||
				item.kelas_nama.toLowerCase().includes(query) ||
				item.guru_nama.toLowerCase().includes(query) ||
				item.materi.toLowerCase().includes(query);
			const matchesClass = selectedClass === "" || String(item.kelas_id) === selectedClass;
			return matchesSearch && matchesClass;
		}),
	);

	let filteredItems = $derived(
		baseFilteredItems.filter((item) => {
			if (selectedStatus === "") return true;
			if (selectedStatus === "selesai") return item.status === "selesai";
			if (selectedStatus === "belum") return item.status !== "selesai";
			return true;
		}),
	);

	let visibleSummary = $derived.by(() => {
		if (searchQuery.trim() === "" && selectedClass === "") return summary;
		const kelas = new Set<number>();
		let selesai = 0;
		let belumSelesai = 0;
		for (const item of baseFilteredItems) {
			kelas.add(item.kelas_id);
			if (item.status === "selesai") selesai += 1;
			else belumSelesai += 1;
		}
		return {
			total_dimulai: baseFilteredItems.length,
			selesai,
			belum_selesai: belumSelesai,
			kelas_diajar: kelas.size,
		};
	});

	let hasLocalFilters = $derived(searchQuery.trim() !== "" || selectedClass !== "" || selectedStatus !== "");

	const rangeLabel = $derived(
		range === "7"
			? "7 hari terakhir"
			: range === "14"
				? "14 hari terakhir"
				: range === "custom"
					? `${start_date} sampai ${end_date}`
					: "30 hari terakhir",
	);

	function resetLocalFilters() {
		searchQuery = "";
		selectedClass = "";
		selectedStatus = "";
	}

	function formatTanggal(value: string): string {
		const date = new Date(value + "T00:00:00");
		if (Number.isNaN(date.getTime())) return value;
		return date.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
	}

	function statusLabel(status: string): string {
		if (status === "selesai") return "Selesai";
		if (status === "menyelesaikan") return "Sedang menyelesaikan";
		return "Belum selesai";
	}

	function meetingNumber(item: RiwayatMengajarItem): number {
		return item.pertemuan_level_ke > 0 ? item.pertemuan_level_ke : item.pertemuan_ke;
	}
</script>

<svelte:head><title>Riwayat Mengajar</title></svelte:head>

<AppLayout {user}>
	<div class="border-b border-neutral-200/80 pt-8 pb-8 dark:border-white/[0.04]">
		<div class="mx-auto max-w-6xl px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400">
				<a href="/app/guru" use:inertia class="transition-colors hover:text-brand-600 dark:hover:text-brand-400">Dashboard</a>
				<span>/</span>
				<span class="text-neutral-700 dark:text-neutral-300">Riwayat Mengajar</span>
			</div>
			<h1 class="mt-4 text-2xl font-bold tracking-tight text-neutral-900 dark:text-white sm:text-3xl">Riwayat Mengajar</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">Lihat aktivitas pertemuan yang pernah dimulai atau diselesaikan.</p>
		</div>
	</div>

	<div class="mx-auto max-w-6xl space-y-6 px-4 py-8 sm:px-6">
		<section class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
			<div class="flex flex-wrap items-center gap-2">
				<a href="/app/guru/riwayat-mengajar?range=7" use:inertia
					class="rounded-xl px-4 py-2 text-sm font-semibold transition-colors {range === '7' ? 'bg-brand-600 text-white' : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700'}"
				>7 Hari</a>
				<a href="/app/guru/riwayat-mengajar?range=14" use:inertia
					class="rounded-xl px-4 py-2 text-sm font-semibold transition-colors {range === '14' ? 'bg-brand-600 text-white' : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700'}"
				>14 Hari</a>
				<a href="/app/guru/riwayat-mengajar?range=30" use:inertia
					class="rounded-xl px-4 py-2 text-sm font-semibold transition-colors {range === '30' ? 'bg-brand-600 text-white' : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700'}"
				>30 Hari</a>
				<button type="button" onclick={() => (showCustom = !showCustom)}
					class="rounded-xl px-4 py-2 text-sm font-semibold transition-colors {range === 'custom' || showCustom ? 'bg-brand-400/15 text-brand-700 dark:text-brand-300' : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700'}"
				>Custom</button>
				<span class="ml-auto text-xs text-neutral-500 dark:text-neutral-400">{rangeLabel}</span>
			</div>

			{#if showCustom}
				<form method="GET" action="/app/guru/riwayat-mengajar" class="mt-4 grid gap-3 border-t border-neutral-200/80 pt-4 dark:border-white/[0.05] sm:grid-cols-[1fr_1fr_auto]">
					<input type="hidden" name="range" value="custom" />
					<label class="space-y-1">
						<span class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Tanggal mulai</span>
						<input type="date" name="start" value={start_date} required
							class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
						/>
					</label>
					<label class="space-y-1">
						<span class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Tanggal akhir</span>
						<input type="date" name="end" value={end_date} required
							class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
						/>
					</label>
					<button type="submit" class="self-end rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700">Terapkan</button>
				</form>
			{/if}
		</section>

		<section class="grid grid-cols-2 gap-3 lg:grid-cols-4">
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="flex items-center gap-2">
					<span class="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-400/10 text-brand-600 dark:text-brand-400"><History class="h-4 w-4" /></span>
					<span class="font-mono text-2xl font-bold text-neutral-900 dark:text-white">{visibleSummary.total_dimulai}</span>
				</div>
				<p class="mt-2 text-xs font-medium text-neutral-600 dark:text-neutral-400">Total Pertemuan Dimulai</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="flex items-center gap-2">
					<span class="flex h-9 w-9 items-center justify-center rounded-xl bg-green-500/10 text-green-600 dark:text-green-400"><CheckCircle class="h-4 w-4" /></span>
					<span class="font-mono text-2xl font-bold text-neutral-900 dark:text-white">{visibleSummary.selesai}</span>
				</div>
				<p class="mt-2 text-xs font-medium text-neutral-600 dark:text-neutral-400">Selesai</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="flex items-center gap-2">
					<span class="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-500/10 text-amber-600 dark:text-amber-400"><AlertCircle class="h-4 w-4" /></span>
					<span class="font-mono text-2xl font-bold text-neutral-900 dark:text-white">{visibleSummary.belum_selesai}</span>
				</div>
				<p class="mt-2 text-xs font-medium text-neutral-600 dark:text-neutral-400">Belum Selesai</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="flex items-center gap-2">
					<span class="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500/10 text-sky-600 dark:text-sky-400"><Users class="h-4 w-4" /></span>
					<span class="font-mono text-2xl font-bold text-neutral-900 dark:text-white">{visibleSummary.kelas_diajar}</span>
				</div>
				<p class="mt-2 text-xs font-medium text-neutral-600 dark:text-neutral-400">Kelas Diajar</p>
			</div>
		</section>

		<section class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-[1.4fr_1fr_1fr]">
				<label class="relative">
					<span class="sr-only">Cari kelas atau materi</span>
					<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
					<input bind:value={searchQuery} type="search" placeholder="Cari kelas atau materi..."
						class="w-full rounded-xl border border-neutral-300 bg-neutral-50 py-2.5 pl-9 pr-3 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
					/>
				</label>
				<select bind:value={selectedClass} aria-label="Filter kelas"
					class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
				>
					<option value="">Semua kelas</option>
					{#each classOptions as option}<option value={option.id}>{option.nama}</option>{/each}
				</select>
				<select bind:value={selectedStatus} aria-label="Filter status"
					class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
				>
					<option value="">Semua status</option>
					<option value="selesai">Selesai</option>
					<option value="belum">Belum selesai</option>
				</select>
			</div>
			<div class="mt-3 flex items-center justify-between gap-3">
				<p class="text-xs text-neutral-500 dark:text-neutral-400">{filteredItems.length} pertemuan ditampilkan</p>
				{#if hasLocalFilters}
					<button type="button" onclick={resetLocalFilters}
						class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
					><RotateCcw class="h-3.5 w-3.5" /> Reset filter</button>
				{/if}
			</div>
		</section>

		<section class="space-y-3">
			{#if filteredItems.length > 0}
				{#each filteredItems as item (item.id)}
					<article class="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
						<div class="flex flex-wrap items-start justify-between gap-4">
							<div class="flex min-w-0 gap-3">
								<span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-brand-400/10 text-brand-600 dark:text-brand-400"><BookOpen class="h-5 w-5" /></span>
								<div class="min-w-0">
									<div class="flex flex-wrap items-center gap-2">
										<h2 class="font-semibold text-neutral-900 dark:text-white">{item.kelas_nama}</h2>
										<span class="rounded-md bg-neutral-100 px-2 py-0.5 text-xs font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">Pertemuan ke-{meetingNumber(item)}</span>
										{#if item.level_nama}<span class="rounded-md bg-brand-400/10 px-2 py-0.5 text-xs font-semibold text-brand-700 dark:text-brand-300">{item.level_nama}</span>{/if}
									</div>
									<div class="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-sm text-neutral-500 dark:text-neutral-400">
										<span class="inline-flex items-center gap-1.5"><CalendarDays class="h-3.5 w-3.5" />{formatTanggal(item.tanggal)}</span>
										<span class="inline-flex items-center gap-1.5"><Clock class="h-3.5 w-3.5" />{item.jam_mulai}{item.jam_selesai ? " - " + item.jam_selesai : ""}</span>
										{#if isAllGuruView && item.guru_nama}<span>{item.guru_nama}</span>{/if}
									</div>
								</div>
							</div>

							<div class="flex flex-wrap items-center gap-2">
								<span class="rounded-full px-2.5 py-1 text-xs font-semibold {item.status === 'selesai' ? 'bg-green-500/10 text-green-700 dark:text-green-400' : 'bg-amber-500/10 text-amber-700 dark:text-amber-400'}">{statusLabel(item.status)}</span>
								{#if item.status === "berlangsung"}
									<a href={"/app/guru/kelas/" + item.kelas_id + "/pertemuan/" + item.id + "/selesai"} use:inertia
										class="rounded-xl bg-brand-600 px-3 py-2 text-xs font-semibold text-white hover:bg-brand-700"
									>Lanjutkan</a>
								{/if}
							</div>
						</div>

						{#if item.materi}
							<p class="mt-4 text-sm text-neutral-700 dark:text-neutral-300"><span class="font-medium">Materi:</span> {item.materi}</p>
						{/if}
						{#if item.catatan}
							<p class="mt-2 text-sm text-neutral-500 dark:text-neutral-400">{item.catatan}</p>
						{/if}
						{#if item.is_reschedule || item.is_badal}
							<div class="mt-3 flex flex-wrap gap-2 text-xs">
								{#if item.is_reschedule}<span class="rounded-full bg-blue-500/10 px-2.5 py-1 text-blue-700 dark:text-blue-400">Reschedule{item.jadwal_semula ? " dari " + item.jadwal_semula : ""}</span>{/if}
								{#if item.is_badal}<span class="rounded-full bg-violet-500/10 px-2.5 py-1 text-violet-700 dark:text-violet-400">Guru badal</span>{/if}
							</div>
						{/if}
					</article>
				{/each}
			{:else}
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-12 text-center dark:border-white/[0.06] dark:bg-neutral-925/50">
					<History class="mx-auto h-8 w-8 text-neutral-400" />
					<h2 class="mt-4 font-semibold text-neutral-900 dark:text-white">Tidak ada riwayat yang sesuai</h2>
					<p class="mt-1 text-sm text-neutral-500">Ubah periode atau reset filter untuk melihat riwayat lainnya.</p>
				</div>
			{/if}
		</section>
	</div>
</AppLayout>
