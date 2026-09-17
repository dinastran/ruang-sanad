<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User } from "@lib/types";
	import { Wallet, TrendingUp, TrendingDown, Landmark, Receipt, ChevronLeft, ChevronRight } from "lucide-svelte";

	interface NominalAngkatan {
		angkatan: string;
		angkatan_kelas: string;
		total_nominal: number;
	}

	interface SantriTidakLanjut {
		id: number;
		id_mahasantri: string;
		nama: string;
		angkatan: string;
		angkatan_kelas: string;
		keterangan_tidak_lanjut: string;
	}

	interface SantriInfaq {
		id: number;
		id_mahasantri: string;
		nama: string;
		angkatan: string;
		nominal: number;
		infaq_terakhir: string;
	}

	interface Props {
		user?: User;
		nominal_angkatan: NominalAngkatan[];
		tidak_lanjut?: SantriTidakLanjut[];
		infaq_rekap?: SantriInfaq[];
		success?: string;
		error?: string;
	}

	let { user, nominal_angkatan = [], tidak_lanjut = [], infaq_rekap = [], success, error }: Props = $props();

	let sudahInfaq = $derived(infaq_rekap.filter((s) => (s.infaq_terakhir || "").trim() !== "").length);

	function rupiah(val: number): string {
		return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(val);
	}

	let totalNominal = $derived(nominal_angkatan.reduce((sum, na) => sum + na.total_nominal, 0));

	let angkatanTertinggi = $derived(
		nominal_angkatan.length > 0
			? nominal_angkatan.reduce((max, na) => na.total_nominal > max.total_nominal ? na : max)
			: null
	);

	const pageSize = 10;
	let nominalPage = $state(1);
	let tidakLanjutPage = $state(1);
	let infaqPage = $state(1);
	let nominalTotalPages = $derived(Math.max(1, Math.ceil(nominal_angkatan.length / pageSize)));
	let tidakLanjutTotalPages = $derived(Math.max(1, Math.ceil(tidak_lanjut.length / pageSize)));
	let infaqTotalPages = $derived(Math.max(1, Math.ceil(infaq_rekap.length / pageSize)));
	let nominalRows = $derived(nominal_angkatan.slice((nominalPage - 1) * pageSize, nominalPage * pageSize));
	let tidakLanjutRows = $derived(tidak_lanjut.slice((tidakLanjutPage - 1) * pageSize, tidakLanjutPage * pageSize));
	let infaqRows = $derived(infaq_rekap.slice((infaqPage - 1) * pageSize, infaqPage * pageSize));

	function rangeLabel(page: number, total: number): string {
		if (total === 0) return "Tidak ada data";
		const from = (page - 1) * pageSize + 1;
		return `${from}-${Math.min(page * pageSize, total)} dari ${total}`;
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
				<span class="text-neutral-700 dark:text-neutral-300">Laporan Keuangan</span>
			</div>
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
						Laporan Keuangan
					</h1>
					<p class="text-neutral-600 dark:text-neutral-400">
					Rekapitulasi nominal per Angkatan Pendaftaran
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

		<div class="grid md:grid-cols-3 gap-5" in:fly={{ y: 20, duration: 600 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-3">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<Wallet class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Nominal</span>
				</div>
				<div class="text-2xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{rupiah(totalNominal)}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-3">
					<div class="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center">
						<TrendingUp class="w-5 h-5 text-blue-600 dark:text-blue-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Jumlah Angkatan Pendaftaran</span>
				</div>
				<div class="text-2xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{nominal_angkatan.length}</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-3">
					<div class="w-10 h-10 rounded-xl bg-green-500/10 flex items-center justify-center">
						<Landmark class="w-5 h-5 text-green-600 dark:text-green-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Angkatan Pendaftaran Tertinggi</span>
				</div>
				<div class="text-2xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{angkatanTertinggi ? angkatanTertinggi.angkatan : "-"}</div>
				{#if angkatanTertinggi}
					<p class="text-xs text-neutral-500 mt-1">{rupiah(angkatanTertinggi.total_nominal)}</p>
				{/if}
			</div>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<Landmark class="w-5 h-5 text-neutral-500" />
				<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Nominal per Angkatan Pendaftaran</h3>
			</div>

			{#if nominal_angkatan.length > 0}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
							<th class="text-left px-6 py-3">Angkatan Pendaftaran</th>
								<th class="text-right px-6 py-3">Total Nominal</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each nominalRows as na}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
									<td class="px-6 py-4 text-sm font-medium text-neutral-900 dark:text-white">Angkatan {na.angkatan}</td>
									<td class="px-6 py-4 text-sm font-mono text-right text-neutral-700 dark:text-neutral-300">{rupiah(na.total_nominal)}</td>
								</tr>
							{/each}
						</tbody>
						<tfoot class="bg-neutral-50 dark:bg-neutral-900/50 border-t-2 border-neutral-200/80 dark:border-white/[0.04]">
							<tr>
								<td class="px-6 py-4 text-sm font-bold text-neutral-900 dark:text-white">Total</td>
								<td class="px-6 py-4 text-sm font-mono font-bold text-right text-brand-600 dark:text-brand-400">{rupiah(totalNominal)}</td>
							</tr>
						</tfoot>
					</table>
				</div>
				{#if nominalTotalPages > 1}
					<div class="flex items-center justify-between gap-3 px-6 py-3 border-t border-neutral-200/80 dark:border-white/[0.04] text-sm text-neutral-500">
						<span>{rangeLabel(nominalPage, nominal_angkatan.length)}</span>
						<div class="flex items-center gap-1"><button onclick={() => nominalPage--} disabled={nominalPage === 1} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-30"><ChevronLeft size="16" /></button><span class="px-2">{nominalPage}/{nominalTotalPages}</span><button onclick={() => nominalPage++} disabled={nominalPage === nominalTotalPages} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-30"><ChevronRight size="16" /></button></div>
					</div>
				{/if}
			{:else}
				<div class="p-12 text-center">
					<div class="w-12 h-12 rounded-xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-3">
						<Wallet class="w-6 h-6 text-neutral-500" />
					</div>
					<p class="text-sm text-neutral-500 dark:text-neutral-400">Belum ada data nominal</p>
				</div>
			{/if}
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 150 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<TrendingDown class="w-5 h-5 text-red-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Santri Tidak Lanjut</h3>
					<span class="text-sm font-normal text-neutral-500">({tidak_lanjut.length} santri)</span>
				</div>
				{#if tidak_lanjut.length > 0}<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
								<th class="text-left px-6 py-3">ID Mahasantri</th>
								<th class="text-left px-6 py-3">Nama</th>
								<th class="text-left px-6 py-3">Angkatan Pendaftaran</th>
								<th class="text-left px-6 py-3">Angkatan Kelas</th>
								<th class="text-left px-6 py-3">Keterangan</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each tidakLanjutRows as tl}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
									<td class="px-6 py-3.5 text-sm font-mono text-neutral-700 dark:text-neutral-300">{tl.id_mahasantri}</td>
									<td class="px-6 py-3.5 text-sm font-medium text-neutral-900 dark:text-white">{tl.nama}</td>
									<td class="px-6 py-3.5 text-sm text-neutral-600">{tl.angkatan}</td>
									<td class="px-6 py-3.5 text-sm text-neutral-600">{tl.angkatan_kelas || '-'}</td>
									<td class="px-6 py-3.5 text-sm text-neutral-500">{tl.keterangan_tidak_lanjut || "-"}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
					{#if tidakLanjutTotalPages > 1}<div class="flex items-center justify-between gap-3 px-6 py-3 border-t border-neutral-200/80 dark:border-white/[0.04] text-sm text-neutral-500"><span>{rangeLabel(tidakLanjutPage, tidak_lanjut.length)}</span><div class="flex items-center gap-1"><button onclick={() => tidakLanjutPage--} disabled={tidakLanjutPage === 1} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-30"><ChevronLeft size="16" /></button><span class="px-2">{tidakLanjutPage}/{tidakLanjutTotalPages}</span><button onclick={() => tidakLanjutPage++} disabled={tidakLanjutPage === tidakLanjutTotalPages} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-30"><ChevronRight size="16" /></button></div></div>{/if}
				{:else}<div class="p-10 text-center text-sm text-neutral-500">Tidak ada santri berstatus tidak lanjut.</div>{/if}
			</div>
		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 200 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<Receipt class="w-5 h-5 text-brand-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Rekap Infaq</h3>
					<span class="ml-auto text-sm font-normal text-neutral-500">{sudahInfaq}/{infaq_rekap.length} sudah tercatat</span>
				</div>
				{#if infaq_rekap.length > 0}
					<div class="overflow-x-auto">
						<table class="w-full">
							<thead>
								<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
									<th class="text-left px-6 py-3">ID Mahasantri</th>
									<th class="text-left px-6 py-3">Nama</th>
									<th class="text-left px-6 py-3">Angkatan Pendaftaran</th>
									<th class="text-left px-6 py-3">Angkatan Kelas</th>
									<th class="text-right px-6 py-3">Nominal</th>
									<th class="text-left px-6 py-3">Infaq Terakhir</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
								{#each infaqRows as s}
									<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
										<td class="px-6 py-3.5 text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.id_mahasantri || '-'}</td>
										<td class="px-6 py-3.5 text-sm font-medium text-neutral-900 dark:text-white">{s.nama}</td>
										<td class="px-6 py-3.5 text-sm text-neutral-600 dark:text-neutral-400">{s.angkatan}</td>
										<td class="px-6 py-3.5 text-sm text-neutral-600 dark:text-neutral-400">{s.angkatan_kelas || '-'}</td>
										<td class="px-6 py-3.5 text-sm font-mono text-right text-neutral-700 dark:text-neutral-300">{rupiah(s.nominal)}</td>
										<td class="px-6 py-3.5 text-sm {s.infaq_terakhir ? 'text-neutral-700 dark:text-neutral-300' : 'text-neutral-400 italic'}">{s.infaq_terakhir || 'belum tercatat'}</td>
									</tr>
								{/each}
							</tbody>
					</table>
					</div>
					{#if infaqTotalPages > 1}<div class="flex items-center justify-between gap-3 px-6 py-3 border-t border-neutral-200/80 dark:border-white/[0.04] text-sm text-neutral-500"><span>{rangeLabel(infaqPage, infaq_rekap.length)}</span><div class="flex items-center gap-1"><button onclick={() => infaqPage--} disabled={infaqPage === 1} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-30"><ChevronLeft size="16" /></button><span class="px-2">{infaqPage}/{infaqTotalPages}</span><button onclick={() => infaqPage++} disabled={infaqPage === infaqTotalPages} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-30"><ChevronRight size="16" /></button></div></div>{/if}
				{:else}
					<div class="p-12 text-center">
						<div class="w-12 h-12 rounded-xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-3">
							<Receipt class="w-6 h-6 text-neutral-500" />
						</div>
						<p class="text-sm text-neutral-500 dark:text-neutral-400">Belum ada data santri</p>
					</div>
				{/if}
			</div>

	</div>
</AppLayout>
