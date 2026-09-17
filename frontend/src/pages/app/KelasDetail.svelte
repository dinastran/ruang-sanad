<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import StatusBadge from "@components/StatusBadge.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import {
		ArrowLeft, Users, BookOpen, UserCheck, MoveRight, Calendar, Clock, GraduationCap, ChevronDown, UserX, Power, Trash2
	} from "lucide-svelte";

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
		pertemuan_terakhir: number;
		tanggal_pertemuan_terakhir: string;
		materi_terakhir: string;
		is_aktif: boolean;
		created_at: string;
	}

	interface SantriResponse {
		id: number;
		id_mahasantri: string;
		nama: string;
		status: string;
		jenis_kelamin: string;
		kelas_id?: number;
		total_hadir: number;
		total_izin: number;
		total_sakit: number;
		total_alpa: number;
		total_telat: number;
		persen_hadir: number;
	}

	interface Guru {
		id: number;
		nama: string;
		jenis_kelamin: string;
	}

	interface Props {
		user?: User;
		kelas: KelasResponse;
		santri: SantriResponse[];
		gurus: Guru[];
		kelas_lain?: KelasResponse[];
		has_pertemuan?: boolean;
		return_to?: string;
		success?: string;
		error?: string;
	}

	let { user, kelas, santri = [], gurus = [], kelas_lain = [], has_pertemuan = false, return_to = "/app/kelas", success, error }: Props = $props();

	let selectedGuruId = $state<number | null>(null);
	let isAssignLoading = $state(false);
	let showGuruDropdown = $state(false);
	let pertemuanTerakhir = $state(kelas.pertemuan_terakhir);
	let isPertemuanLoading = $state(false);

	let pindahModal = $state<{ open: boolean; santriId: number; santriNama: string }>({ open: false, santriId: 0, santriNama: "" });
	let pindahKelasId = $state<number | null>(null);
	let isPindahLoading = $state(false);

	let santriAktif = $derived(santri.filter((s) => s.status === "aktif" || s.status === "lengkap"));
	let santriPerluLengkap = $derived(santri.filter((s) => s.status === "perlu_dilengkapi"));
	let santriTidakLanjut = $derived(santri.filter((s) => s.status === "tidak_lanjut"));
	// Active roster (counts toward occupancy) vs. tidak-lanjut history, shown separately.
	let santriRoster = $derived(santri.filter((s) => s.status !== "tidak_lanjut"));

	let capacityPercent = $derived(kelas.kapasitas > 0 ? Math.round((kelas.jumlah_santri / kelas.kapasitas) * 100) : 0);

	function capacityColor(pct: number): string {
		if (pct >= 90) return "bg-red-500";
		if (pct >= 75) return "bg-amber-500";
		return "bg-brand-500";
	}

	function handleAssignGuru() {
		if (!selectedGuruId) return;
		isAssignLoading = true;
		router.put(`/app/kelas/${kelas.id}/guru?return_to=${encodeURIComponent(return_to)}`, { guru_id: selectedGuruId }, {
			preserveScroll: true,
			onSuccess: () => {
				Toast("Guru berhasil di-assign", "success");
				showGuruDropdown = false;
			},
			onError: () => {
				Toast("Gagal meng-assign guru", "error");
			},
			onFinish: () => {
				isAssignLoading = false;
			},
		});
	}

	function simpanPertemuanTerakhir() {
		if (pertemuanTerakhir < 0) return;
		isPertemuanLoading = true;
		router.put(`/app/kelas/${kelas.id}/pertemuan-terakhir?return_to=${encodeURIComponent(return_to)}`, { pertemuan_terakhir: pertemuanTerakhir }, {
			preserveScroll: true,
			onFinish: () => { isPertemuanLoading = false; },
		});
	}

	function openPindahModal(s: SantriResponse) {
		pindahModal = { open: true, santriId: s.id, santriNama: s.nama };
		pindahKelasId = null;
	}

	let isStatusLoading = $state(false);

	function toggleAktif() {
		isStatusLoading = true;
		const menjadiAktif = !kelas.is_aktif;
		router.put(`/app/kelas/${kelas.id}/status?from_detail=1&return_to=${encodeURIComponent(return_to)}`, { is_aktif: menjadiAktif }, {
			preserveScroll: true,
			onSuccess: () => Toast(menjadiAktif ? "Kelas diaktifkan" : "Kelas dinonaktifkan", "success"),
			onError: () => Toast("Gagal mengubah status kelas", "error"),
			onFinish: () => { isStatusLoading = false; },
		});
	}

	function hapusKelas() {
		if (!confirm(`Hapus kelas "${kelas.nama_kelas}"?\nSantri di kelas ini akan dilepas (kelasnya dikosongkan).`)) return;
		router.delete(`/app/kelas/${kelas.id}?return_to=${encodeURIComponent(return_to)}`, {
			onError: () => Toast("Gagal menghapus kelas", "error"),
		});
	}

	function handlePindahkanSantri() {
		if (!pindahKelasId) return;
		isPindahLoading = true;
		router.post("/app/santri/" + pindahModal.santriId + "/pindah", { kelas_tujuan_id: pindahKelasId }, {
			preserveScroll: true,
			onSuccess: () => {
				Toast("Santri berhasil dipindahkan", "success");
				pindahModal = { open: false, santriId: 0, santriNama: "" };
			},
			onError: () => {
				Toast("Gagal memindahkan santri", "error");
			},
			onFinish: () => {
				isPindahLoading = false;
			},
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
				<a href={return_to} use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Kelas</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{kelas.nama_kelas}</span>
			</div>

			<a href={return_to} use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 transition-colors mb-4">
				<ArrowLeft class="w-4 h-4" />
				Kembali ke daftar kelas
			</a>

			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div class="flex items-center gap-4 min-w-0">
					<div class="w-14 h-14 rounded-2xl bg-brand-400/15 flex items-center justify-center shrink-0">
						<BookOpen class="w-7 h-7 text-brand-600 dark:text-brand-400" />
					</div>
					<div class="min-w-0">
						<div class="flex items-center gap-2.5 flex-wrap mb-1">
							<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight break-words">{kelas.nama_kelas}</h1>
							<GenderBadge gender={kelas.jenis_kelamin} />
							{#if kelas.is_aktif}
								<span class="px-2 py-0.5 rounded-md bg-green-500/10 text-green-700 dark:text-green-400 text-xs font-semibold">Aktif</span>
							{:else}
								<span class="px-2 py-0.5 rounded-md bg-neutral-200 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400 text-xs font-semibold">Nonaktif</span>
							{/if}
						</div>
						<p class="text-neutral-600 dark:text-neutral-400">
						Angkatan Kelas {kelas.angkatan} · {kelas.tipe}{kelas.sub_index > 0 ? ` · Sub ${kelas.sub_index}` : ""} · {kelas.level} · {kelas.frekuensi}
						</p>
					</div>
				</div>

				<div class="flex items-center gap-2 shrink-0">
					<button
						onclick={toggleAktif}
						disabled={isStatusLoading}
						class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold transition-colors disabled:opacity-50 {kelas.is_aktif ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700' : 'bg-green-600 hover:bg-green-700 text-white'}"
					>
						<Power class="w-4 h-4" />
						{kelas.is_aktif ? "Nonaktifkan" : "Aktifkan"}
					</button>
					<button
						onclick={hapusKelas}
						aria-label="Hapus kelas"
						class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold bg-red-500/10 text-red-600 dark:text-red-400 hover:bg-red-500/20 transition-colors"
					>
						<Trash2 class="w-4 h-4" />
						<span class="hidden sm:inline">Hapus</span>
					</button>
				</div>
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

		<div class="grid lg:grid-cols-3 gap-6" in:fly={{ y: 20, duration: 600 }}>
			<div class="lg:col-span-2 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<div class="grid grid-cols-2 gap-6">
					<div>
						<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Level</span>
						<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{kelas.level}</p>
					</div>
					<div>
						<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Frekuensi</span>
						<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{kelas.frekuensi}</p>
					</div>
					<div>
						<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Jadwal</span>
						<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{kelas.jadwal}</p>
					</div>
					<div>
						<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Jenis Kelamin</span>
						<p class="mt-1"><GenderBadge gender={kelas.jenis_kelamin} /></p>
					</div>
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<div class="rounded-xl border border-brand-400/20 bg-brand-400/5 p-4">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
							<div>
								<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Pertemuan terakhir sebelum sistem</h3>
								<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Isi nomor pertemuan riil kelas lama. Pertemuan berikutnya akan menjadi {pertemuanTerakhir + 1}, dan periode tagihan tetap mengikuti nomor riil.</p>
							</div>
							<div class="flex items-center gap-2">
								<input type="number" min="0" bind:value={pertemuanTerakhir} disabled={has_pertemuan} aria-label="Pertemuan terakhir sebelum sistem" class="w-24 rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm font-mono text-neutral-900 outline-none focus:border-brand-400 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white" />
								<button onclick={simpanPertemuanTerakhir} disabled={has_pertemuan || isPertemuanLoading || pertemuanTerakhir < 0} class="rounded-xl bg-brand-600 px-3.5 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 dark:bg-brand-500 dark:hover:bg-brand-400">Simpan</button>
							</div>
						</div>
						<p class="mt-3 text-xs text-amber-700 dark:text-amber-400">{has_pertemuan ? "Pengaturan terkunci karena kelas sudah memiliki pertemuan di aplikasi." : "Pengaturan ini akan terkunci setelah pertemuan pertama dicatat di aplikasi."}</p>
					</div>
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Pertemuan Terakhir</h3>
					{#if kelas.tanggal_pertemuan_terakhir}
						<div class="mt-3 grid gap-3 sm:grid-cols-2">
							<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50">
								<p class="text-xs font-medium uppercase tracking-wider text-neutral-500">Pertemuan</p>
								<p class="mt-1 text-sm font-semibold text-neutral-900 dark:text-white">Ke-{kelas.pertemuan_terakhir} · {kelas.tanggal_pertemuan_terakhir}</p>
							</div>
							<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50">
								<p class="text-xs font-medium uppercase tracking-wider text-neutral-500">Materi</p>
								<p class="mt-1 text-sm font-semibold text-neutral-900 dark:text-white">{kelas.materi_terakhir || "Belum diisi"}</p>
							</div>
						</div>
					{:else}
						<p class="mt-2 text-sm text-neutral-500 dark:text-neutral-400">Belum ada pertemuan selesai yang tercatat di aplikasi.</p>
					{/if}
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<div class="flex items-center justify-between mb-2">
						<span class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kapasitas</span>
						<span class="text-sm font-mono text-neutral-700 dark:text-neutral-300">
							{kelas.jumlah_santri} / {kelas.kapasitas}
							<span class="text-neutral-500 ml-1">({capacityPercent}%)</span>
						</span>
					</div>
					<div class="h-2 rounded-full bg-neutral-200/80 dark:bg-neutral-800 overflow-hidden">
						<div
							class="h-full rounded-full transition-all duration-500 {capacityColor(capacityPercent)}"
							style="width: {Math.min(capacityPercent, 100)}%"
						></div>
					</div>
				</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<GraduationCap class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Guru</h3>
						<p class="text-xs text-neutral-500 dark:text-neutral-400">Pengajar kelas ini</p>
					</div>
				</div>

				{#if kelas.guru_nama}
					<div class="flex items-center gap-3 p-3 rounded-xl bg-brand-400/5 border border-brand-400/15 mb-4">
						<div class="w-9 h-9 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm shrink-0">
							{kelas.guru_nama.charAt(0).toUpperCase()}
						</div>
						<div class="min-w-0">
							<p class="text-sm font-semibold text-neutral-900 dark:text-white truncate">{kelas.guru_nama}</p>
							<p class="text-xs text-neutral-500">Guru</p>
						</div>
					</div>
				{:else}
					<div class="flex items-center gap-3 p-3 rounded-xl bg-amber-500/5 border border-amber-500/15 mb-4">
						<div class="w-9 h-9 rounded-full bg-amber-500/20 flex items-center justify-center text-amber-600 dark:text-amber-400 font-bold text-sm shrink-0">
							?
						</div>
						<p class="text-sm text-amber-700 dark:text-amber-400 font-medium">Belum ada guru</p>
					</div>
				{/if}

				<div class="relative">
					<button
						onclick={() => (showGuruDropdown = !showGuruDropdown)}
						class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 text-sm"
					>
						<UserCheck class="w-4 h-4" />
						Assign Guru
						<ChevronDown class="w-4 h-4" />
					</button>

					{#if showGuruDropdown}
						<div class="absolute top-full left-0 right-0 mt-2 bg-white dark:bg-neutral-925 rounded-xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] overflow-hidden z-10 ring-1 ring-black/10 dark:ring-white/10">
							<div class="max-h-48 overflow-y-auto p-2 space-y-1">
								{#each gurus as guru}
									<button
										onclick={() => { selectedGuruId = guru.id; showGuruDropdown = false; handleAssignGuru(); }}
										class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-left hover:bg-brand-400/10 transition-colors text-neutral-700 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white"
									>
										<div class="w-7 h-7 rounded-full bg-neutral-200 dark:bg-neutral-800 flex items-center justify-center text-xs font-bold text-neutral-600 dark:text-neutral-400 shrink-0">
											{guru.nama.charAt(0).toUpperCase()}
										</div>
										<span>{guru.nama}</span>
									</button>
								{/each}
							</div>
							{#if gurus.length === 0}
								<div class="p-4 text-center text-sm text-neutral-500">Tidak ada guru tersedia</div>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		</div>

		<div class="grid grid-cols-3 gap-4" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Aktif</p>
				<p class="text-3xl font-bold text-green-600 dark:text-green-400 font-mono">{santriAktif.length}</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Perlu Dilengkapi</p>
				<p class="text-3xl font-bold text-amber-600 dark:text-amber-400 font-mono">{santriPerluLengkap.length}</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Tidak Lanjut</p>
				<p class="text-3xl font-bold text-red-600 dark:text-red-400 font-mono">{santriTidakLanjut.length}</p>
			</div>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 150 }}>
			<div class="flex items-center justify-between px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<div class="flex items-center gap-2.5">
					<Users class="w-5 h-5 text-neutral-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Daftar Santri</h3>
					<span class="text-sm font-normal text-neutral-500">({santriRoster.length} aktif)</span>
				</div>
			</div>

			{#if santriRoster.length > 0}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
								<th class="text-left px-6 py-3 w-12">No</th>
								<th class="text-left px-6 py-3">ID Mahasantri</th>
								<th class="text-left px-6 py-3">Nama</th>
								<th class="text-left px-6 py-3">Status</th>
								<th class="text-center px-6 py-3">Hadir</th>
								<th class="text-center px-6 py-3">I/S/A/T</th>
								<th class="text-center px-6 py-3">%</th>
								<th class="text-right px-6 py-3">Aksi</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each santriRoster as s, i}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
									<td class="px-6 py-3.5 text-sm text-neutral-500 dark:text-neutral-400 font-mono">{i + 1}</td>
									<td class="px-6 py-3.5 text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.id_mahasantri}</td>
									<td class="px-6 py-3.5">
										<a href={"/app/santri/" + s.id} use:inertia class="text-sm font-medium text-neutral-900 dark:text-white hover:text-brand-600 dark:hover:text-brand-400 transition-colors">
											{s.nama}
										</a>
									</td>
									<td class="px-6 py-3.5">
										<StatusBadge status={s.status} />
									</td>
									<td class="px-6 py-3.5 text-center text-sm font-mono text-green-600 dark:text-green-400">{s.total_hadir}</td>
									<td class="px-6 py-3.5 text-center text-xs font-mono text-neutral-500 dark:text-neutral-400">{s.total_izin}/{s.total_sakit}/{s.total_alpa}/{s.total_telat}</td>
									<td class="px-6 py-3.5 text-center text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.persen_hadir.toFixed(0)}%</td>
									<td class="px-6 py-3.5 text-right">
										<button
											onclick={() => openPindahModal(s)}
											class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:bg-brand-400/10 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
										>
											<MoveRight class="w-3.5 h-3.5" />
											Pindahkan
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="p-12 text-center">
					<div class="w-12 h-12 rounded-xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-3">
						<Users class="w-6 h-6 text-neutral-500" />
					</div>
					<p class="text-sm text-neutral-500 dark:text-neutral-400">Belum ada santri aktif di kelas ini</p>
				</div>
			{/if}
		</div>

		{#if santriTidakLanjut.length > 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 200 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<UserX class="w-5 h-5 text-red-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Riwayat Tidak Lanjut</h3>
					<span class="text-sm font-normal text-neutral-500">({santriTidakLanjut.length} santri)</span>
				</div>
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
								<th class="text-left px-6 py-3 w-12">No</th>
								<th class="text-left px-6 py-3">ID Mahasantri</th>
								<th class="text-left px-6 py-3">Nama</th>
								<th class="text-left px-6 py-3">Status</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each santriTidakLanjut as s, i}
								<tr class="opacity-70 hover:opacity-100 hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-all">
									<td class="px-6 py-3.5 text-sm text-neutral-500 dark:text-neutral-400 font-mono">{i + 1}</td>
									<td class="px-6 py-3.5 text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.id_mahasantri}</td>
									<td class="px-6 py-3.5">
										<a href={"/app/santri/" + s.id} use:inertia class="text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:text-brand-600 dark:hover:text-brand-400 transition-colors line-through decoration-neutral-400/60">
											{s.nama}
										</a>
									</td>
									<td class="px-6 py-3.5">
										<StatusBadge status={s.status} />
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</div>
</AppLayout>

{#if pindahModal.open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={() => (pindahModal = { open: false, santriId: 0, santriNama: "" })}></button>
		<div class="relative w-full max-w-md bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-2">Pindahkan Santri</h3>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">Pindahkan <strong>{pindahModal.santriNama}</strong> ke kelas lain</p>

			<select
				bind:value={pindahKelasId}
				class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm mb-4"
			>
				<option value={null} disabled>Pilih kelas tujuan</option>
				{#each kelas_lain as kl}
					<option value={kl.id}>Angkatan {kl.angkatan} · {kl.nama_kelas} - {kl.level} ({kl.jadwal})</option>
				{/each}
			</select>

			<div class="flex items-center gap-3">
				<button
					onclick={() => (pindahModal = { open: false, santriId: 0, santriNama: "" })}
					class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
				>
					Batal
				</button>
				<button
					onclick={handlePindahkanSantri}
					disabled={!pindahKelasId || isPindahLoading}
					class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm flex items-center justify-center gap-2"
				>
					{#if isPindahLoading}
						<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Memindahkan...
					{:else}
						Pindahkan
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}
