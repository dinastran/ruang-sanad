<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import StatusBadge from "@components/StatusBadge.svelte";
	import StatCard from "@components/StatCard.svelte";
	import type { User } from "@lib/types";
	import {
		Users, BookOpen, Wallet, TrendingUp, AlertCircle, CheckCircle, XCircle, BarChart3,
		UserCheck, UserX, School, ArrowRight, PieChart
	} from "lucide-svelte";

	interface LevelStat {
		level: string;
		total: number;
	}

	interface TipeStat {
		tipe: string;
		total: number;
	}

	interface NominalAngkatan {
		angkatan: string;
		total_nominal: number;
	}

	interface DashboardStats {
		total_santri: number;
		santri_lengkap: number;
		santri_perlu_lengkap: number;
		santri_tidak_lanjut: number;
		total_kelas: number;
		total_nominal: number;
		santri_laki: number;
		santri_perempuan: number;
		per_level: LevelStat[];
		per_tipe: TipeStat[];
		nominal_per_angkatan: NominalAngkatan[];
	}

	interface SantriResponse {
		id: number;
		id_mahasantri: string;
		nama: string;
		status: string;
	}

	interface RoleData {
		perlu_dilengkapi?: SantriResponse[];
	}

	interface Props {
		user?: User;
		stats: DashboardStats;
		role_data: RoleData;
		success?: string;
		error?: string;
	}

	let { user, stats, role_data = {}, success, error }: Props = $props();

	let role = $derived(user?.role || "");

	function rupiah(val: number): string {
		return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(val);
	}

	let maxLevel = $derived(stats?.per_level?.length > 0 ? Math.max(...stats.per_level.map((l) => l.total)) : 1);

	let maxTipe = $derived(stats?.per_tipe?.length > 0 ? Math.max(...stats.per_tipe.map((t) => t.total)) : 1);

	let isSuperAdmin = $derived(role === "super_admin");
	let isAdminKelas = $derived(role === "admin_kelas" || isSuperAdmin);
	let isKeuangan = $derived(role === "keuangan" || isSuperAdmin);
	let isCS = $derived(role === "cs" || isSuperAdmin);

	let perluDilengkapi = $derived(role_data?.perlu_dilengkapi || []);

	function barColor(index: number): string {
		const colors = ["bg-brand-500", "bg-secondary-500", "bg-blue-500", "bg-green-500", "bg-amber-500", "bg-pink-500", "bg-cyan-500", "bg-violet-500"];
		return colors[index % colors.length];
	}

	function barColorLight(index: number): string {
		const colors = ["bg-brand-500/15", "bg-secondary-500/15", "bg-blue-500/15", "bg-green-500/15", "bg-amber-500/15", "bg-pink-500/15", "bg-cyan-500/15", "bg-violet-500/15"];
		return colors[index % colors.length];
	}
</script>

<AppLayout {user} group="dashboard">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
						Selamat datang, {user?.name?.split(" ")[0] || "Admin"}
					</h1>
					<p class="text-neutral-600 dark:text-neutral-400">
						Ruang Sanad — Manajemen data santri dan kelas
					</p>
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

		<div class="grid md:grid-cols-4 gap-5" in:fly={{ y: 20, duration: 600 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<Users class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Santri</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.total_santri || 0}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-blue-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center">
						<UserCheck class="w-5 h-5 text-blue-600 dark:text-blue-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Laki-laki</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.santri_laki || 0}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-pink-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-pink-500/10 flex items-center justify-center">
						<UserX class="w-5 h-5 text-pink-600 dark:text-pink-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Perempuan</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.santri_perempuan || 0}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-secondary-500/10 flex items-center justify-center">
						<School class="w-5 h-5 text-secondary-600 dark:text-secondary-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Kelas</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.total_kelas || 0}</div>
			</div>
		</div>

		<div class="grid md:grid-cols-4 gap-5" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-green-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-green-500/10 flex items-center justify-center">
						<CheckCircle class="w-5 h-5 text-green-600 dark:text-green-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Santri Lengkap</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.santri_lengkap || 0}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-amber-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center">
						<AlertCircle class="w-5 h-5 text-amber-600 dark:text-amber-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Perlu Dilengkapi</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.santri_perlu_lengkap || 0}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-red-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-red-500/10 flex items-center justify-center">
						<XCircle class="w-5 h-5 text-red-600 dark:text-red-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Tidak Lanjut</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{stats?.santri_tidak_lanjut || 0}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<Wallet class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Nominal</span>
				</div>
				<div class="text-2xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{rupiah(stats?.total_nominal || 0)}</div>
			</div>
		</div>

		{#if isAdminKelas && perluDilengkapi.length > 0}
			<div class="rounded-2xl border border-amber-500/20 bg-amber-500/5 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 150 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-amber-500/15">
					<AlertCircle class="w-5 h-5 text-amber-600 dark:text-amber-400" />
					<h3 class="text-base font-semibold text-amber-800 dark:text-amber-300">Santri Perlu Dilengkapi</h3>
					<span class="text-sm font-normal text-amber-600 dark:text-amber-400">({perluDilengkapi.length} santri)</span>
				</div>
				<div class="divide-y divide-amber-500/10">
					{#each perluDilengkapi as s}
						<a
							href="/app/perlu-dilengkapi"
							use:inertia
							class="flex items-center justify-between px-6 py-3.5 hover:bg-amber-500/5 transition-colors"
						>
							<div class="flex items-center gap-3 min-w-0">
								<div class="w-8 h-8 rounded-full bg-amber-500/15 flex items-center justify-center text-amber-700 dark:text-amber-400 text-xs font-bold shrink-0">
									{s.nama.charAt(0).toUpperCase()}
								</div>
								<div class="min-w-0">
									<p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{s.nama}</p>
									<p class="text-xs text-neutral-500 dark:text-neutral-400 font-mono">{s.id_mahasantri}</p>
								</div>
							</div>
							<ArrowRight class="w-4 h-4 text-amber-500 shrink-0" />
						</a>
					{/each}
				</div>
			</div>
		{/if}

		{#if (isSuperAdmin || isKeuangan) && stats?.nominal_per_angkatan?.length > 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 150 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<Wallet class="w-5 h-5 text-neutral-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Nominal per Angkatan Pendaftaran</h3>
				</div>
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
							<th class="text-left px-6 py-3">Angkatan Pendaftaran</th>
								<th class="text-right px-6 py-3">Total Nominal</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each stats.nominal_per_angkatan as na}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
								<td class="px-6 py-3.5 text-sm font-medium text-neutral-900 dark:text-white">Angkatan Pendaftaran {na.angkatan}</td>
									<td class="px-6 py-3.5 text-sm font-mono text-right text-neutral-700 dark:text-neutral-300">{rupiah(na.total_nominal)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}

		<div class="grid lg:grid-cols-2 gap-6" in:fly={{ y: 20, duration: 600, delay: 200 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<div class="flex items-center gap-3 mb-5">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<BarChart3 class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<div>
						<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Distribusi Level</h3>
						<p class="text-xs text-neutral-500 dark:text-neutral-400">Jumlah santri per level</p>
					</div>
				</div>

				{#if stats?.per_level?.length > 0}
					<div class="space-y-3">
						{#each stats.per_level as level, i}
							<div>
								<div class="flex items-center justify-between text-sm mb-1">
									<span class="font-medium text-neutral-700 dark:text-neutral-300">{level.level || "Tanpa Level"}</span>
									<span class="text-sm font-mono text-neutral-500">{level.total}</span>
								</div>
								<div class="h-2.5 rounded-full bg-neutral-200/80 dark:bg-neutral-800 overflow-hidden">
									<div
										class="h-full rounded-full transition-all duration-700 {barColor(i)}"
										style="width: {(level.total / maxLevel) * 100}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="text-center py-8 text-sm text-neutral-500">Belum ada data level</div>
				{/if}
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<div class="flex items-center gap-3 mb-5">
					<div class="w-10 h-10 rounded-xl bg-secondary-500/10 flex items-center justify-center">
						<PieChart class="w-5 h-5 text-secondary-600 dark:text-secondary-400" />
					</div>
					<div>
						<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Distribusi Tipe</h3>
						<p class="text-xs text-neutral-500 dark:text-neutral-400">Jumlah santri per tipe</p>
					</div>
				</div>

				{#if stats?.per_tipe?.length > 0}
					<div class="space-y-3">
						{#each stats.per_tipe as tipe, i}
							<div>
								<div class="flex items-center justify-between text-sm mb-1">
									<span class="font-medium text-neutral-700 dark:text-neutral-300">{tipe.tipe || "Tanpa Tipe"}</span>
									<span class="text-sm font-mono text-neutral-500">{tipe.total}</span>
								</div>
								<div class="h-2.5 rounded-full bg-neutral-200/80 dark:bg-neutral-800 overflow-hidden">
									<div
										class="h-full rounded-full transition-all duration-700 {barColor(i + 3)}"
										style="width: {(tipe.total / maxTipe) * 100}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="text-center py-8 text-sm text-neutral-500">Belum ada data tipe</div>
				{/if}
			</div>
		</div>
	</div>
</AppLayout>
