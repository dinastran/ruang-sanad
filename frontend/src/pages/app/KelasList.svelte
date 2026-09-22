<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { User } from "@lib/types";
	import { BookOpen, Users, Calendar, Clock, UserCheck, ArrowRight, Filter, Search, School, Power, Trash2, RotateCcw } from "lucide-svelte";

	interface KelasResponse {
		id: number;
		kunci_kelas: string;
		angkatan: string;
		tipe: string;
		jenis_kelamin: string;
		level: string;
		frekuensi: string;
		jadwal: string;
		sub_index: number;
		nama_kelas: string;
		guru_id: number | null;
		guru_nama?: string;
		kapasitas: number;
		jumlah_santri: number;
		is_aktif: boolean;
		created_at: string;
	}

	interface AngkatanItem {
		id: number;
		kode: string;
		keterangan?: string;
	}

	interface KelasFilters {
		q?: string;
		guru_id?: string;
		angkatan?: string;
		status?: string;
		gender?: string;
		level?: string;
		jadwal?: string;
	}

	interface Props {
		user?: User;
		kelas?: KelasResponse[];
		angkatan?: AngkatanItem[];
		filters?: KelasFilters;
		success?: string;
		error?: string;
	}

	let p: Props = $props();
	let user = $derived(p.user);
	let kelas = $derived(p.kelas ?? []);
	let angkatan = $derived(p.angkatan ?? []);
	let success = $derived(p.success);
	let error = $derived(p.error);

	let filterAngkatan = $state(p.filters?.angkatan ?? "");
	let filterGuru = $state(p.filters?.guru_id ?? "");
	let filterStatus = $state(p.filters?.status ?? "");
	let filterGender = $state(p.filters?.gender ?? "");
	let filterLevel = $state(p.filters?.level ?? "");
	let filterJadwal = $state(p.filters?.jadwal ?? "");
	let searchQuery = $state(p.filters?.q ?? "");

	let guruOptions = $derived(
		Array.from(new Map(kelas.filter((k) => k.guru_id && k.guru_nama).map((k) => [String(k.guru_id), k.guru_nama!])).entries())
			.map(([id, nama]) => ({ id, nama }))
			.sort((a, b) => a.nama.localeCompare(b.nama)),
	);
	let levelOptions = $derived(Array.from(new Set(kelas.map((k) => k.level).filter(Boolean))).sort((a, b) => a.localeCompare(b)));
	const hariUrutan = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Ahad"];
	function urutkanJadwal(a: string, b: string): number {
		const hariA = hariUrutan.indexOf(a.split(",")[0].trim());
		const hariB = hariUrutan.indexOf(b.split(",")[0].trim());
		return (hariA === -1 ? hariUrutan.length : hariA) - (hariB === -1 ? hariUrutan.length : hariB) || a.localeCompare(b);
	}
	let jadwalOptions = $derived(
		Array.from(new Set(kelas.flatMap((k) => k.jadwal.split(" & ").map((jadwal) => jadwal.trim()).filter(Boolean)))).sort(urutkanJadwal),
	);
	let hasActiveFilters = $derived(Boolean(searchQuery || filterAngkatan || filterGuru || filterStatus || filterGender || filterLevel || filterJadwal));

	function listURL(): string {
		const params = new URLSearchParams();
		if (searchQuery.trim()) params.set("q", searchQuery.trim());
		if (filterGuru) params.set("guru_id", filterGuru);
		if (filterAngkatan) params.set("angkatan", filterAngkatan);
		if (filterStatus) params.set("status", filterStatus);
		if (filterGender) params.set("gender", filterGender);
		if (filterLevel) params.set("level", filterLevel);
		if (filterJadwal) params.set("jadwal", filterJadwal);
		const query = params.toString();
		return query ? `/app/kelas?${query}` : "/app/kelas";
	}

	function syncFilterURL() {
		window.history.replaceState({}, "", listURL());
	}

	function resetFilters() {
		searchQuery = "";
		filterGuru = "";
		filterAngkatan = "";
		filterStatus = "";
		filterGender = "";
		filterLevel = "";
		filterJadwal = "";
		syncFilterURL();
	}

	function kelasDetailURL(id: number): string {
		return `/app/kelas/${id}?return_to=${encodeURIComponent(listURL())}`;
	}

	function groupByAngkatan(items: KelasResponse[]): Record<string, KelasResponse[]> {
		const groups: Record<string, KelasResponse[]> = {};
		for (const k of items) {
			if (!groups[k.angkatan]) groups[k.angkatan] = [];
			groups[k.angkatan].push(k);
		}
		return groups;
	}

	let filteredKelas = $derived(
		kelas.filter((k) => {
			if (filterAngkatan && k.angkatan !== filterAngkatan) return false;
			if (filterGuru === "unassigned" && k.guru_id !== null) return false;
			if (filterGuru && filterGuru !== "unassigned" && String(k.guru_id ?? "") !== filterGuru) return false;
			if (filterStatus === "aktif" && !k.is_aktif) return false;
			if (filterStatus === "nonaktif" && k.is_aktif) return false;
			if (filterGender && k.jenis_kelamin !== filterGender) return false;
			if (filterLevel && k.level !== filterLevel) return false;
			if (filterJadwal && !k.jadwal.split(" & ").some((jadwal) => jadwal.trim() === filterJadwal)) return false;
			const query = searchQuery.trim().toLowerCase();
			if (query) {
				const searchable = [k.nama_kelas, k.tipe, k.level, k.jadwal, k.guru_nama ?? ""].join(" ").toLowerCase();
				if (!searchable.includes(query)) return false;
			}
			return true;
		})
	);

	let grouped = $derived(groupByAngkatan(filteredKelas));
	let angkatanKeys = $derived(Object.keys(grouped).sort().reverse());

	let totalKelas = $derived(kelas.length);
	let totalSantriTerisi = $derived(kelas.reduce((sum, k) => sum + k.jumlah_santri, 0));
	let totalKapasitas = $derived(kelas.reduce((sum, k) => sum + k.kapasitas, 0));

	let capacityPercent = $derived(totalKapasitas > 0 ? Math.round((totalSantriTerisi / totalKapasitas) * 100) : 0);

	function capacityColor(pct: number): string {
		if (pct >= 90) return "bg-red-500";
		if (pct >= 75) return "bg-amber-500";
		return "bg-brand-500";
	}

	function capacityTextColor(pct: number): string {
		if (pct >= 90) return "text-red-600 dark:text-red-400";
		if (pct >= 75) return "text-amber-600 dark:text-amber-400";
		return "text-green-600 dark:text-green-400";
	}

	let busyId = $state<number | null>(null);

	function toggleAktif(k: KelasResponse) {
		busyId = k.id;
		router.put(`/app/kelas/${k.id}/status?return_to=${encodeURIComponent(listURL())}`, { is_aktif: !k.is_aktif }, {
			preserveScroll: true,
			onFinish: () => { busyId = null; },
		});
	}

	function hapusKelas(k: KelasResponse) {
		if (!confirm(`Hapus kelas "${k.nama_kelas}"?\nSantri di kelas ini akan dilepas (kelasnya dikosongkan).`)) return;
		busyId = k.id;
		router.delete(`/app/kelas/${k.id}?return_to=${encodeURIComponent(listURL())}`, {
			preserveScroll: true,
			onFinish: () => { busyId = null; },
		});
	}
</script>

<AppLayout {user} group="dashboard">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Daftar Kelas</span>
			</div>
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
						Daftar Kelas
					</h1>
					<p class="text-neutral-600 dark:text-neutral-400">
						Kelola kelas dan lihat daftar santri per kelas
					</p>
				</div>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<div class="w-8 h-8 rounded-full bg-green-500/20 flex items-center justify-center shrink-0">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
				<p class="text-sm font-medium">{success}</p>
			</div>
		{/if}

		{#if error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<div class="w-8 h-8 rounded-full bg-red-500/20 flex items-center justify-center shrink-0">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
				<p class="text-sm font-medium">{error}</p>
			</div>
		{/if}

		<div class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4" in:fly={{ y: 20, duration: 600 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 sm:p-5">
				<div class="flex items-center gap-2.5 sm:gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<School class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<span class="text-xs sm:text-sm font-medium text-neutral-500 dark:text-neutral-400 leading-tight">Total Kelas</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{totalKelas}</div>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 sm:p-5">
				<div class="flex items-center gap-2.5 sm:gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center">
						<Users class="w-5 h-5 text-blue-600 dark:text-blue-400" />
					</div>
					<span class="text-xs sm:text-sm font-medium text-neutral-500 dark:text-neutral-400 leading-tight">Santri Terisi</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{totalSantriTerisi}</div>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 sm:p-5">
				<div class="flex items-center gap-2.5 sm:gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center">
						<Filter class="w-5 h-5 text-amber-600 dark:text-amber-400" />
					</div>
					<span class="text-xs sm:text-sm font-medium text-neutral-500 dark:text-neutral-400 leading-tight">Total Kapasitas</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{totalKapasitas}</div>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 sm:p-5">
				<div class="flex items-center gap-2.5 sm:gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-green-500/10 flex items-center justify-center">
						<Users class="w-5 h-5 text-green-600 dark:text-green-400" />
					</div>
					<span class="text-xs sm:text-sm font-medium text-neutral-500 dark:text-neutral-400 leading-tight">Kapasitas Terpakai</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{capacityPercent}%</div>
			</div>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 space-y-3" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			<div class="flex flex-col sm:flex-row sm:items-center gap-3">
			<div class="relative flex-1 sm:max-w-sm">
				<label for="kelas-search" class="sr-only">Cari kelas</label>
				<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
					<Search class="w-4 h-4 text-neutral-500" />
				</div>
				<input
					id="kelas-search"
					type="text"
					bind:value={searchQuery}
					oninput={syncFilterURL}
					placeholder="Cari kelas, guru, jadwal..."
					class="w-full pl-10 pr-4 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none text-sm"
				/>
			</div>
			<select
				aria-label="Angkatan kelas"
				bind:value={filterAngkatan}
				onchange={syncFilterURL}
				class="w-full sm:w-auto px-4 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm appearance-none"
			>
				<option value="">Semua Angkatan Kelas</option>
				{#each angkatan as ang}
					<option value={ang.kode}>{ang.keterangan || ang.kode}</option>
				{/each}
			</select>
			</div>
			<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
				<select aria-label="Guru" bind:value={filterGuru} onchange={syncFilterURL} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
					<option value="">Semua Guru</option>
					<option value="unassigned">Belum Ada Guru</option>
					{#each guruOptions as guru}<option value={guru.id}>{guru.nama}</option>{/each}
				</select>
				<select aria-label="Status kelas" bind:value={filterStatus} onchange={syncFilterURL} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
					<option value="">Semua Status</option><option value="aktif">Aktif</option><option value="nonaktif">Nonaktif</option>
				</select>
				<select aria-label="Jenis kelamin kelas" bind:value={filterGender} onchange={syncFilterURL} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
					<option value="">Semua Gender</option><option value="L">Laki-laki</option><option value="P">Perempuan</option>
				</select>
				<select aria-label="Level kelas" bind:value={filterLevel} onchange={syncFilterURL} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
					<option value="">Semua Level</option>{#each levelOptions as level}<option value={level}>{level}</option>{/each}
				</select>
				<select aria-label="Jadwal kelas" bind:value={filterJadwal} onchange={syncFilterURL} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
					<option value="">Semua Jadwal</option>{#each jadwalOptions as jadwal}<option value={jadwal}>{jadwal}</option>{/each}
				</select>
				{#if hasActiveFilters}
					<button onclick={resetFilters} class="inline-flex items-center justify-center gap-2 rounded-xl border border-neutral-300 dark:border-neutral-700/80 px-3 py-2.5 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">
						<RotateCcw class="w-4 h-4" /> Reset
					</button>
				{/if}
			</div>
			<p class="text-xs text-neutral-500 dark:text-neutral-400">Menampilkan {filteredKelas.length} dari {totalKelas} kelas{hasActiveFilters ? " sesuai filter" : ""}.</p>
		</div>

		{#each angkatanKeys as angkatanKey}
			<section in:fly={{ y: 20, duration: 500 }}>
				<div class="flex items-center gap-3 mb-4 mt-2">
					<div class="flex items-center justify-center w-8 h-8 rounded-lg bg-brand-400/15 text-brand-600 dark:text-brand-400 text-sm font-bold">
						{angkatanKey}
					</div>
					<h2 class="text-xl font-bold text-neutral-900 dark:text-white">Angkatan Kelas {angkatanKey}</h2>
					<span class="text-sm text-neutral-500 dark:text-neutral-400 font-mono">({grouped[angkatanKey].length} kelas)</span>
				</div>

				<div class="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
					{#each grouped[angkatanKey] as k (k.id)}
						<div
							class="group relative rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 sm:p-5 transition-all hover:border-brand-400/30 hover:shadow-lg hover:shadow-brand-400/5 {k.is_aktif ? '' : 'opacity-60'}"
						>
							<div class="flex items-start justify-between gap-2 mb-3">
								<div class="flex items-start gap-2.5 min-w-0">
									<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
										<BookOpen class="w-5 h-5 text-brand-600 dark:text-brand-400" />
									</div>
									<div class="min-w-0">
										<h3 class="font-semibold text-neutral-900 dark:text-white leading-snug break-words">
											<a href={kelasDetailURL(k.id)} use:inertia aria-label={`Buka detail ${k.nama_kelas}`} class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors after:absolute after:inset-0 after:content-['']">{k.nama_kelas}</a>
										</h3>
										<p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">
											{k.tipe}{k.sub_index ? ` - ${k.sub_index}` : ""}
										</p>
									</div>
								</div>
								<div class="shrink-0 flex flex-col items-end gap-1">
									<GenderBadge gender={k.jenis_kelamin} />
									{#if !k.is_aktif}
										<span class="px-2 py-0.5 rounded-md bg-neutral-200 dark:bg-neutral-800 text-[10px] font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wide">Nonaktif</span>
									{/if}
								</div>
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
								<div class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400">
									<UserCheck class="w-3.5 h-3.5 shrink-0" />
									<span class="truncate">{k.guru_nama || "Belum ada guru"}</span>
								</div>
							</div>

							<div class="flex items-center justify-between gap-3 mb-1.5">
								<div class="flex items-center gap-1.5 text-xs font-medium">
									<Users class="w-3.5 h-3.5 text-neutral-500" />
									<span class={capacityTextColor(k.kapasitas > 0 ? (k.jumlah_santri / k.kapasitas) * 100 : 0)}>
										{k.jumlah_santri}/{k.kapasitas}
									</span>
								</div>
								<ArrowRight class="w-4 h-4 text-neutral-400 group-hover:text-brand-500 transition-colors" />
							</div>

							<div class="h-1.5 rounded-full bg-neutral-200/80 dark:bg-neutral-800 overflow-hidden">
								<div
									class="h-full rounded-full transition-all duration-500 {capacityColor(k.kapasitas > 0 ? (k.jumlah_santri / k.kapasitas) * 100 : 0)}"
									style="width: {k.kapasitas > 0 ? Math.min((k.jumlah_santri / k.kapasitas) * 100, 100) : 0}%"
								></div>
							</div>

							<div class="relative z-10 flex items-center gap-2 mt-3 pt-3 border-t border-neutral-200/70 dark:border-white/[0.04]">
								<button
									onclick={() => toggleAktif(k)}
									disabled={busyId === k.id}
									class="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-semibold transition-colors disabled:opacity-50 {k.is_aktif ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700' : 'bg-green-600 hover:bg-green-700 text-white'}"
								>
									<Power class="w-3.5 h-3.5" />
									{k.is_aktif ? "Nonaktifkan" : "Aktifkan"}
								</button>
								<button
									onclick={() => hapusKelas(k)}
									disabled={busyId === k.id}
									aria-label="Hapus kelas"
									class="inline-flex items-center justify-center p-2 rounded-lg text-neutral-400 hover:text-red-500 hover:bg-red-500/10 transition-colors disabled:opacity-50"
								>
									<Trash2 class="w-4 h-4" />
								</button>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/each}

		{#if angkatanKeys.length === 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500, delay: 150 }}>
				<div class="w-16 h-16 rounded-2xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-4">
					<BookOpen class="w-8 h-8 text-neutral-500" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">Tidak ada kelas</h3>
				<p class="text-neutral-500 dark:text-neutral-400 max-w-md mx-auto text-sm">
					{hasActiveFilters ? "Tidak ada kelas yang sesuai dengan filter yang dipilih" : "Belum ada kelas yang terdaftar"}
				</p>
				{#if hasActiveFilters}<button onclick={resetFilters} class="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-brand-600 hover:text-brand-700 dark:text-brand-400"><RotateCcw class="w-4 h-4" /> Reset filter</button>{/if}
			</div>
		{/if}
	</div>
</AppLayout>
