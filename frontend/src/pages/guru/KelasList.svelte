<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { User, GuruKelas } from "@lib/types";
	import { BookOpen, Users, Calendar, Clock, ArrowRight, Search, RotateCcw } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas?: GuruKelas[];
		santri_search?: Record<string, string[]>;
		success?: string;
		error?: string;
	}

	let { user, kelas = [], santri_search = {}, success, error }: Props = $props();
	let isAllGuruView = $derived(user?.role === "super_admin" || user?.role === "admin_kelas");
	let selectedGuru = $state("");
	let searchQuery = $state("");
	let selectedLevel = $state("");
	let selectedType = $state("");
	let selectedFrequency = $state("");
	let selectedDay = $state("");
	let guruOptions = $derived(
		Array.from(new Map(kelas.filter((k) => k.guru_id).map((k) => [String(k.guru_id), k.guru_nama])).entries())
			.map(([id, nama]) => ({ id, nama }))
			.sort((a, b) => a.nama.localeCompare(b.nama)),
	);
	let levelOptions = $derived(Array.from(new Set(kelas.map((k) => k.level).filter(Boolean))).sort((a, b) => a.localeCompare(b)));
	let typeOptions = $derived(Array.from(new Set(kelas.map((k) => k.tipe).filter(Boolean))).sort((a, b) => a.localeCompare(b)));
	let frequencyOptions = $derived(Array.from(new Set(kelas.map((k) => k.frekuensi).filter(Boolean))).sort((a, b) => a.localeCompare(b)));
	const days = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Ahad"];
	let dayOptions = $derived(days.filter((day) => kelas.some((k) => k.jadwal?.toLowerCase().includes(day.toLowerCase()))));
	let hasActiveFilters = $derived(
		searchQuery.trim() !== "" ||
			selectedGuru !== "" ||
			selectedLevel !== "" ||
			selectedType !== "" ||
			selectedFrequency !== "" ||
			selectedDay !== "",
	);
	let filteredKelas = $derived(
		kelas.filter((k) => {
			const query = searchQuery.trim().toLowerCase();
			const santriTerms = santri_search[String(k.id)] ?? [];
			const matchesSearch =
				query === "" ||
				k.nama_kelas.toLowerCase().includes(query) ||
				santriTerms.some((term) => term.toLowerCase().includes(query));
			return (
				matchesSearch &&
				(selectedGuru === "" || String(k.guru_id ?? "") === selectedGuru) &&
				(selectedLevel === "" || k.level === selectedLevel) &&
				(selectedType === "" || k.tipe === selectedType) &&
				(selectedFrequency === "" || k.frekuensi === selectedFrequency) &&
				(selectedDay === "" || k.jadwal?.toLowerCase().includes(selectedDay.toLowerCase()))
			);
		}),
	);

	function resetFilters() {
		searchQuery = "";
		selectedGuru = "";
		selectedLevel = "";
		selectedType = "";
		selectedFrequency = "";
		selectedDay = "";
	}

	function groupKelas(list: GuruKelas[]) {
		const groups = new Map<string, { guruNama: string; kelas: GuruKelas[] }>();
		for (const item of list) {
			const key = String(item.guru_id ?? "belum-ditugaskan");
			const group = groups.get(key) ?? { guruNama: item.guru_nama || "Belum ditugaskan", kelas: [] };
			group.kelas.push(item);
			groups.set(key, group);
		}
		return Array.from(groups.values()).sort((a, b) => a.guruNama.localeCompare(b.guruNama));
	}

	let kelasPerGuru = $derived(groupKelas(filteredKelas));
</script>

<AppLayout {user} group="guru-kelas">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{isAllGuruView ? "Kelas Guru" : "Kelas Saya"}</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
				{isAllGuruView ? "Kelas Guru" : "Kelas Saya"}
			</h1>
			<p class="text-neutral-600 dark:text-neutral-400">{isAllGuruView ? "Daftar seluruh kelas, dikelompokkan berdasarkan guru" : "Daftar kelas yang Anda ajar"}</p>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{success}</p>
			</div>
		{/if}

		{#if error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{error}</p>
			</div>
		{/if}

		{#if kelas.length > 0}
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50" in:fly={{ y: 20, duration: 500 }}>
				<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
					<label class="relative sm:col-span-2 lg:col-span-1">
						<span class="sr-only">Cari mahasantri atau kelas</span>
						<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
						<input
							bind:value={searchQuery}
							type="search"
							placeholder="Cari nama mahasantri..."
							class="w-full rounded-xl border border-neutral-300 bg-neutral-50 py-2.5 pl-9 pr-3 text-sm text-neutral-900 outline-none transition focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
						/>
					</label>

					<select bind:value={selectedLevel} aria-label="Filter level"
						class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
					>
						<option value="">Semua level</option>
						{#each levelOptions as level}<option value={level}>{level}</option>{/each}
					</select>

					<select bind:value={selectedType} aria-label="Filter tipe kelas"
						class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
					>
						<option value="">Semua tipe</option>
						{#each typeOptions as type}<option value={type}>{type}</option>{/each}
					</select>

					<select bind:value={selectedFrequency} aria-label="Filter frekuensi"
						class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
					>
						<option value="">Semua frekuensi</option>
						{#each frequencyOptions as frequency}<option value={frequency}>{frequency}</option>{/each}
					</select>

					<select bind:value={selectedDay} aria-label="Filter hari"
						class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
					>
						<option value="">Semua hari</option>
						{#each dayOptions as day}<option value={day}>{day}</option>{/each}
					</select>

					{#if isAllGuruView}
						<select bind:value={selectedGuru} aria-label="Filter guru"
							class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white"
						>
							<option value="">Semua guru</option>
							{#each guruOptions as guru}<option value={guru.id}>{guru.nama}</option>{/each}
						</select>
					{/if}
				</div>
				<div class="mt-3 flex items-center justify-between gap-3">
					<p class="text-xs text-neutral-500 dark:text-neutral-400">{filteredKelas.length} dari {kelas.length} kelas ditampilkan</p>
					{#if hasActiveFilters}
						<button onclick={resetFilters}
							class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
						>
							<RotateCcw class="h-3.5 w-3.5" /> Reset filter
						</button>
					{/if}
				</div>
			</div>

			{#if kelasPerGuru.length > 0}
				<div class="space-y-7" in:fly={{ y: 20, duration: 600 }}>
					{#each kelasPerGuru as group}
						<section>
							{#if isAllGuruView}
								<div class="flex items-center gap-2 mb-3">
									<Users class="w-4 h-4 text-brand-600 dark:text-brand-400" />
									<h2 class="text-base font-semibold text-neutral-900 dark:text-white">{group.guruNama}</h2>
									<span class="text-xs text-neutral-500 dark:text-neutral-400">{group.kelas.length} kelas</span>
								</div>
							{/if}
							<div class="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
								{#each group.kelas as k}
					<a href={"/app/guru/kelas/" + k.id} use:inertia
						class="group relative rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30 hover:shadow-lg hover:shadow-brand-400/5"
					>
						<div class="flex items-start justify-between gap-2 mb-3">
							<div class="flex items-start gap-2.5 min-w-0">
								<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
									<BookOpen class="w-5 h-5 text-brand-600 dark:text-brand-400" />
								</div>
								<div class="min-w-0">
									<h3 class="font-semibold text-neutral-900 dark:text-white leading-snug break-words">{k.nama_kelas}</h3>
									<p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">{k.tipe}</p>
								</div>
							</div>
							<GenderBadge gender={k.jenis_kelamin} />
						</div>

						<div class="space-y-1.5 mb-3">
							<div class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400">
								<Clock class="w-3.5 h-3.5 shrink-0" />
								<span class="font-medium">{k.level}</span>
								<span class="text-neutral-500">·</span>
								<span>{k.frekuensi}</span>
							</div>
							<div class="flex items-start gap-2 text-xs text-neutral-600 dark:text-neutral-400">
								<Calendar class="w-3.5 h-3.5 shrink-0 mt-0.5" />
								<span class="min-w-0 break-words">{k.jadwal}</span>
							</div>
						</div>

						<div class="flex items-center justify-between pt-3 border-t border-neutral-200/70 dark:border-white/[0.04]">
							<div class="flex items-center gap-1.5 text-xs font-medium">
								<Users class="w-3.5 h-3.5 text-neutral-500" />
								<span class="text-neutral-700 dark:text-neutral-300">{k.jumlah_santri}/{k.kapasitas}</span>
							</div>
							<div class="flex items-center gap-2">
								{#if k.tanggal_terakhir}
									<span class="text-[10px] text-neutral-500 dark:text-neutral-400 bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 rounded-md">
										{k.tanggal_terakhir}
									</span>
								{/if}
								<ArrowRight class="w-4 h-4 text-neutral-400 group-hover:text-brand-500 transition-colors shrink-0" />
							</div>
						</div>
					</a>
								{/each}
							</div>
						</section>
					{/each}
				</div>
			{:else}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-10 text-center">
					<Search class="mx-auto mb-3 h-6 w-6 text-neutral-400" />
					<p class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tidak ada kelas yang sesuai filter</p>
					<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Ubah pencarian atau reset filter untuk menampilkan semua kelas.</p>
					<button onclick={resetFilters} class="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-neutral-100 px-3 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700">
						<RotateCcw class="h-3.5 w-3.5" /> Reset filter
					</button>
				</div>
			{/if}
		{:else}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500, delay: 100 }}>
				<div class="w-16 h-16 rounded-2xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-4">
					<BookOpen class="w-8 h-8 text-neutral-500" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">Belum ada kelas</h3>
				<p class="text-neutral-500 dark:text-neutral-400 max-w-md mx-auto text-sm">
					Anda belum memiliki kelas yang diajar. Hubungi admin untuk mendapatkan penugasan kelas.
				</p>
			</div>
		{/if}
	</div>
</AppLayout>
