<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import StatusBadge from "@components/StatusBadge.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import DataTable from "@components/DataTable.svelte";
	import FilterPanel from "@components/FilterPanel.svelte";
	import type { Flash, User } from "@lib/types";
	import { Search, Plus, ChevronLeft, ChevronRight, AlertTriangle } from "lucide-svelte";

	interface MasterItem {
		id: number;
		kode?: string;
		nama?: string;
		keterangan?: string;
	}

	interface SantriItem {
		id: number;
		id_mahasantri: string;
		id_mahasantri_bermasalah: boolean;
		kelas_kode: string;
		nama: string;
		jenis_kelamin: string;
		nominal: number;
		tanggal_daftar: string;
		angkatan: string;
		angkatan_kelas: string;
		usia: number;
		domisili: string;
		no_wa: string;
		level: string;
		tipe: string;
		frekuensi: string;
		is_lengkap: boolean;
		status: string;
	}

	interface Props {
		user?: User;
		santri?: SantriItem[];
		total?: number;
		page?: number;
		limit?: number;
		angkatan?: MasterItem[];
		levels?: MasterItem[];
		jadwals?: MasterItem[];
		flash?: Flash;
		success?: string;
		error?: string;
		id_status?: string;
		total_id_bermasalah?: number;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let santri = $derived(props.santri ?? []);
	let total = $derived(props.total ?? 0);
	let page = $derived(props.page ?? 1);
	let limit = $derived(props.limit ?? 25);
	let angkatan = $derived(props.angkatan ?? []);
	let levels = $derived(props.levels ?? []);
	let jadwals = $derived(props.jadwals ?? []);
	let success = $derived(props.flash?.success ?? props.success);
	let error = $derived(props.flash?.error ?? props.error);
	let idStatus = $derived(props.id_status ?? "");
	let totalIDBermasalah = $derived(props.total_id_bermasalah ?? 0);

	// admin_kelas can view Data Santri but not create/edit — only cs & super_admin can.
	let canEdit = $derived(user?.role === "cs" || user?.role === "super_admin");

	let searchQuery = $state("");
	let totalPages = $derived(Math.max(1, Math.ceil(total / limit)));
	let startRow = $derived((page - 1) * limit + 1);
	let endRow = $derived(Math.min(page * limit, total));

	function doSearch(e: Event) {
		e.preventDefault();
		const params = new URLSearchParams(window.location.search);
		params.delete("page");
		if (searchQuery) params.set("search", searchQuery);
		else params.delete("search");
		router.get(`/app/santri?${params.toString()}`);
	}

	function goToPage(p: number) {
		if (p < 1 || p > totalPages) return;
		const params = new URLSearchParams(window.location.search);
		params.set("page", String(p));
		router.get(`/app/santri?${params.toString()}`);
	}

	let filters = $derived([
		...(canEdit ? [{
			key: "id_status",
			label: "Status ID",
			options: [{ value: "bermasalah", label: "Perlu Diperbaiki" }],
		}] : []),
		{
			key: "angkatan_pendaftaran",
			label: "Angkatan Pendaftaran",
			options: angkatan.map((a) => ({ value: a.kode || String(a.id), label: a.keterangan || a.kode || String(a.id) })),
		},
		{
			key: "angkatan_kelas",
			label: "Angkatan Kelas",
			options: angkatan.map((a) => ({ value: a.kode || String(a.id), label: a.keterangan || a.kode || String(a.id) })),
		},
		{
			key: "level",
			label: "Level",
			options: levels.map((l) => ({ value: l.kode || String(l.id), label: l.nama || l.kode || String(l.id) })),
		},
		{
			key: "gender",
			label: "Jenis Kelamin",
			options: [
				{ value: "L", label: "Laki-laki" },
				{ value: "P", label: "Perempuan" },
			],
		},
		{
			key: "status",
			label: "Status",
			allValue: "all",
			defaultValue: idStatus === "bermasalah" ? "all" : "aktif",
			options: [
				{ value: "aktif", label: "Aktif" },
				{ value: "perlu_dilengkapi", label: "Perlu Dilengkapi" },
				{ value: "cuti", label: "Cuti" },
				{ value: "nonaktif", label: "Nonaktif" },
				{ value: "tidak_lanjut", label: "Tidak Lanjut" },
			],
		},
	]);

	const columns = [
		{ key: "index", label: "No", class: "w-12 text-center" },
		{ key: "id_mahasantri", label: "ID Mahasantri" },
		{ key: "nama", label: "Nama" },
		{ key: "no_wa", label: "No. WhatsApp" },
		{ key: "jenis_kelamin", label: "Jenis Kelamin" },
		{ key: "angkatan", label: "Angkatan Pendaftaran" },
		{ key: "angkatan_kelas", label: "Angkatan Kelas" },
		{ key: "level", label: "Level" },
		{ key: "tipe", label: "Tipe" },
		{ key: "status", label: "Status" },
	];

	let tableRows = $derived(
		santri.map((s, i) => ({
			index: startRow + i,
			id: s.id,
			id_mahasantri: s.id_mahasantri || "-",
				nama: s.nama,
				no_wa: s.no_wa || "-",
			jenis_kelamin: s.jenis_kelamin,
			angkatan: s.angkatan,
			angkatan_kelas: s.angkatan_kelas,
			level: s.level || "-",
			tipe: s.tipe || "-",
			status: s.status,
		}))
	);
</script>

<AppLayout {user} group="santri">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Data Santri</span>
			</div>
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">Data Santri</h1>
					<p class="text-neutral-600 dark:text-neutral-400">Total {total} santri</p>
				</div>
				{#if canEdit}
					<a
						href="/app/santri/new"
						use:inertia
						class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25"
					>
						<Plus class="w-4 h-4" />
						Tambah Santri
					</a>
				{/if}
			</div>
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

		{#if canEdit}
			<a
				href="/app/santri?id_status=bermasalah"
				use:inertia
				aria-current={idStatus === "bermasalah" ? "page" : undefined}
				class="flex flex-col gap-3 rounded-xl border p-4 transition-colors sm:flex-row sm:items-center sm:justify-between {idStatus === 'bermasalah' ? 'border-warning/40 bg-warning/10' : 'border-neutral-200/80 bg-white hover:border-warning/30 dark:border-white/[0.06] dark:bg-neutral-925/50'}"
			>
				<div class="flex items-start gap-3">
					<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning">
						<AlertTriangle class="h-4 w-4" aria-hidden="true" />
					</div>
					<div>
						<p class="text-sm font-semibold text-neutral-900 dark:text-white">ID Perlu Diperbaiki</p>
						<p class="mt-0.5 text-xs text-neutral-600 dark:text-neutral-400">Inventaris lintas status untuk ID kosong, salah format, atau terduplikasi.</p>
					</div>
				</div>
				<span class="self-start rounded-lg bg-warning/10 px-3 py-1.5 font-mono text-sm font-semibold text-warning sm:self-auto">{totalIDBermasalah}</span>
			</a>
		{/if}

		<form onsubmit={doSearch} class="relative" in:fly={{ y: 20, duration: 500 }}>
			<div class="relative">
				<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
					<Search class="w-5 h-5 text-neutral-500" />
				</div>
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Cari nama santri..."
					class="w-full pl-12 pr-4 py-3 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-200/80 dark:border-white/[0.06] text-neutral-900 dark:text-white placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-brand-400/40 transition-all"
				/>
			</div>
		</form>

		<FilterPanel filters={filters} baseUrl="/app/santri" />

		<div class="overflow-x-auto rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50" in:fly={{ y: 20, duration: 600 }}>
			<table class="w-full text-sm">
				<thead>
					<tr class="bg-neutral-50 dark:bg-neutral-900/50 border-b border-neutral-200/80 dark:border-white/[0.04]">
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider w-12 text-center">No</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">ID Mahasantri</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Nama</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">No. WhatsApp</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Jenis Kelamin</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Angkatan Pendaftaran</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Angkatan Kelas</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Level</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Tipe</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Status</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
					{#if santri.length === 0}
						<tr>
							<td colspan="10" class="px-4 py-12 text-center text-neutral-500 dark:text-neutral-400">
								Tidak ada data santri
							</td>
						</tr>
					{:else}
						{#each santri as s, i}
							<tr
								class="transition-colors cursor-pointer {s.id_mahasantri_bermasalah ? 'bg-warning/5 hover:bg-warning/10' : 'hover:bg-neutral-50/50 dark:hover:bg-white/[0.015]'}"
								onclick={() => router.get(`/app/santri/${s.id}`)}
								role="button"
								tabindex="0"
								onkeydown={(e) => e.key === 'Enter' && router.get(`/app/santri/${s.id}`)}
							>
								<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400 text-center">{startRow + i}</td>
								<td class="px-4 py-3 font-mono text-xs text-neutral-600 dark:text-neutral-400">
									<span class="inline-flex items-center gap-1.5" title={s.id_mahasantri_bermasalah ? "ID perlu ditinjau" : undefined}>
										{s.id_mahasantri || '-'}
										{#if s.id_mahasantri_bermasalah}<AlertTriangle class="h-3.5 w-3.5 shrink-0 text-warning" aria-label="ID perlu ditinjau" />{/if}
									</span>
								</td>
								<td class="px-4 py-3 font-medium text-neutral-900 dark:text-white">{s.nama}</td>
								<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">{s.no_wa || '-'}</td>
								<td class="px-4 py-3"><GenderBadge gender={s.jenis_kelamin} /></td>
								<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">{s.angkatan}</td>
								<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">{s.angkatan_kelas || '-'}</td>
								<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">{s.level || '-'}</td>
								<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">{s.tipe || '-'}</td>
								<td class="px-4 py-3"><StatusBadge status={s.status} /></td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>

		{#if totalPages > 1}
			<div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3" in:fly={{ y: 20, duration: 600 }}>
				<p class="text-sm text-neutral-500 dark:text-neutral-400">
					Menampilkan {startRow}-{endRow} dari {total} santri
				</p>
				<div class="flex items-center gap-2 flex-wrap">
					<button
						onclick={() => goToPage(page - 1)}
						disabled={page <= 1}
						class="p-2 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-all"
					>
						<ChevronLeft class="w-4 h-4" />
					</button>

					{#each Array.from({ length: totalPages }, (_, i) => i + 1) as p}
						<button
							onclick={() => goToPage(p)}
							class="w-9 h-9 rounded-lg text-sm font-medium transition-all {p === page ? 'bg-brand-600 text-white dark:bg-brand-500 shadow-lg shadow-brand-600/25' : 'border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'}"
						>
							{p}
						</button>
					{/each}

					<button
						onclick={() => goToPage(page + 1)}
						disabled={page >= totalPages}
						class="p-2 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-all"
					>
						<ChevronRight class="w-4 h-4" />
					</button>
				</div>
			</div>
		{/if}
	</div>
</AppLayout>
