<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import StatusBadge from "@components/StatusBadge.svelte";
	import FilterPanel from "@components/FilterPanel.svelte";
	import type { User } from "@lib/types";
	import { Wallet, Search, Save, ChevronLeft, ChevronRight } from "lucide-svelte";

	interface MasterItem { id: number; kode?: string; nama?: string; keterangan?: string; }

	interface SantriItem {
		id: number;
		id_mahasantri: string;
		nama: string;
		angkatan: string;
		nominal: number;
		infaq_terakhir: string;
		keterangan_tidak_lanjut: string;
		status: string;
	}

	interface Props {
		user?: User;
		santri?: SantriItem[];
		total?: number;
		page?: number;
		limit?: number;
		angkatan?: MasterItem[];
		success?: string;
		error?: string;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let santri = $derived(props.santri ?? []);
	let total = $derived(props.total ?? 0);
	let page = $derived(props.page ?? 1);
	let limit = $derived(props.limit ?? 25);
	let angkatan = $derived(props.angkatan ?? []);
	let success = $derived(props.success);
	let error = $derived(props.error);

	let searchQuery = $state("");
	let totalPages = $derived(Math.max(1, Math.ceil(total / limit)));
	let startRow = $derived((page - 1) * limit + 1);
	let endRow = $derived(Math.min(page * limit, total));

	// Per-santri edit state. Seed eagerly at init and re-seed whenever the santri
	// list changes (pagination/filter/after-save reload) — NEVER mutate `forms`
	// during render (that throws Svelte's state_unsafe_mutation).
	type KeuForm = { infaq_terakhir: string; keterangan_tidak_lanjut: string };
	function seedForms(list: SantriItem[]): Record<number, KeuForm> {
		const o: Record<number, KeuForm> = {};
		for (const s of list) {
			o[s.id] = { infaq_terakhir: s.infaq_terakhir || "", keterangan_tidak_lanjut: s.keterangan_tidak_lanjut || "" };
		}
		return o;
	}
	let forms = $state<Record<number, KeuForm>>({});
	let savingId = $state<number | null>(null);

	// Seed before paint and re-seed whenever the santri list changes. The row
	// template has a fallback so the very first render (before this runs) is safe.
	$effect.pre(() => {
		forms = seedForms(santri);
	});

	function save(s: SantriItem) {
		savingId = s.id;
		router.put(`/app/santri/${s.id}/keuangan`, forms[s.id] ?? { infaq_terakhir: s.infaq_terakhir || "", keterangan_tidak_lanjut: s.keterangan_tidak_lanjut || "" }, {
			preserveScroll: true,
			onFinish: () => { savingId = null; },
			onError: () => { savingId = null; },
		});
	}

	function rupiah(val: number): string {
		return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(val || 0);
	}

	function doSearch(e: Event) {
		e.preventDefault();
		router.get(`/app/keuangan?search=${encodeURIComponent(searchQuery)}`);
	}

	function goToPage(p: number) {
		if (p < 1 || p > totalPages) return;
		const params = new URLSearchParams(window.location.search);
		params.set("page", String(p));
		router.get(`/app/keuangan?${params.toString()}`);
	}

	let filters = $derived([
		{
			key: "angkatan",
			label: "Angkatan",
			options: angkatan.map((a) => ({ value: a.kode || String(a.id), label: a.keterangan || a.kode || String(a.id) })),
		},
		{
			key: "status",
			label: "Status",
			options: [
				{ value: "aktif", label: "Aktif" },
				{ value: "tidak_lanjut", label: "Tidak Lanjut" },
			],
		},
	]);

	const inputCls =
		"w-full px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none text-sm";
</script>

<AppLayout {user} group="keuangan">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Keuangan</span>
			</div>
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
					<Wallet class="w-5 h-5 text-brand-600 dark:text-brand-400" />
				</div>
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Keuangan</h1>
					<p class="text-sm text-neutral-600 dark:text-neutral-400">Catat infaq bulanan & keterangan tidak lanjut per santri · {total} santri</p>
				</div>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 20, duration: 300 }}>{success}</div>
		{/if}
		{#if error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 20, duration: 300 }}>{error}</div>
		{/if}

		<form onsubmit={doSearch} class="relative">
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

		<FilterPanel filters={filters} baseUrl="/app/keuangan" />

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="bg-neutral-50 dark:bg-neutral-900/50 border-b border-neutral-200/80 dark:border-white/[0.04] text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">
							<th class="px-4 py-3 text-left">Santri</th>
							<th class="px-4 py-3 text-right">Nominal</th>
							<th class="px-4 py-3 text-left w-44">Infaq Terakhir</th>
							<th class="px-4 py-3 text-left w-56">Keterangan Tidak Lanjut</th>
							<th class="px-4 py-3 text-center w-24">Status</th>
							<th class="px-4 py-3 w-20"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
						{#if santri.length === 0}
							<tr><td colspan="6" class="px-4 py-12 text-center text-neutral-500 dark:text-neutral-400">Tidak ada data santri</td></tr>
						{:else}
							{#each santri as s (s.id)}
								{@const form = forms[s.id] ?? { infaq_terakhir: s.infaq_terakhir || '', keterangan_tidak_lanjut: s.keterangan_tidak_lanjut || '' }}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors align-top">
									<td class="px-4 py-3">
										<a href={`/app/santri/${s.id}`} use:inertia class="font-medium text-neutral-900 dark:text-white hover:text-brand-600 dark:hover:text-brand-400 transition-colors">{s.nama}</a>
										<p class="text-xs text-neutral-500 dark:text-neutral-400 font-mono">{s.id_mahasantri || '-'} · {s.angkatan}</p>
									</td>
									<td class="px-4 py-3 text-right font-mono text-neutral-700 dark:text-neutral-300 whitespace-nowrap">{rupiah(s.nominal)}</td>
									<td class="px-4 py-3">
										<input bind:value={form.infaq_terakhir} placeholder="mis. Juli 2026" class={inputCls} />
									</td>
									<td class="px-4 py-3">
										<input bind:value={form.keterangan_tidak_lanjut} placeholder="Kosongkan bila lanjut" class={inputCls} />
									</td>
									<td class="px-4 py-3 text-center"><StatusBadge status={s.status} /></td>
									<td class="px-4 py-3 text-right">
										<button onclick={() => save(s)} disabled={savingId === s.id}
											class="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 text-white text-xs font-semibold transition-colors disabled:opacity-50">
											<Save class="w-3.5 h-3.5" /> Simpan
										</button>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>

		{#if totalPages > 1}
			<div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
				<p class="text-sm text-neutral-500 dark:text-neutral-400">Menampilkan {startRow}-{endRow} dari {total} santri</p>
				<div class="flex items-center gap-2 flex-wrap">
					<button onclick={() => goToPage(page - 1)} disabled={page <= 1}
						class="p-2 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-all">
						<ChevronLeft class="w-4 h-4" />
					</button>
					{#each Array.from({ length: totalPages }, (_, i) => i + 1) as p}
						<button onclick={() => goToPage(p)}
							class="w-9 h-9 rounded-lg text-sm font-medium transition-all {p === page ? 'bg-brand-600 text-white dark:bg-brand-500 shadow-lg shadow-brand-600/25' : 'border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'}">
							{p}
						</button>
					{/each}
					<button onclick={() => goToPage(page + 1)} disabled={page >= totalPages}
						class="p-2 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-all">
						<ChevronRight class="w-4 h-4" />
					</button>
				</div>
			</div>
		{/if}
	</div>
</AppLayout>
