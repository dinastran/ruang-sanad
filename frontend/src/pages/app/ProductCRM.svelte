<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import ProductCRMDashboard from "@components/product-crm/ProductCRMDashboard.svelte";
	import type { User } from "@lib/types";
	import {
		Archive,
		BarChart3,
		Boxes,
		CalendarDays,
		CheckCircle2,
		ClipboardCheck,
		History,
		PackagePlus,
		Pencil,
		Plus,
		Search,
		ShoppingBag,
		X,
	} from "lucide-svelte";

	interface Batch {
		id: number;
		produk_id: number;
		nama: string;
		tanggal_mulai: string;
		tanggal_selesai: string;
		is_aktif: boolean;
	}
	interface Product {
		id: number;
		nama: string;
		kategori: string;
		track_stok: boolean;
		is_aktif: boolean;
		stok: number;
		minimum_stock: number;
		batches: Batch[];
	}
	interface OwnedProduct {
		id: number;
		produk_id: number;
		produk_batch_id: number;
		produk_nama: string;
		batch_nama: string;
		kategori: string;
		tanggal: string;
		catatan: string;
	}
	interface CRMRow {
		id: number;
		id_mahasantri: string;
		nama: string;
		angkatan: string;
		angkatan_kelas: string;
		level: string;
		status: string;
		produk: OwnedProduct[];
	}
	interface Mutation {
		id: number;
		produk_nama: string;
		batch_nama: string;
		tipe: string;
		qty: number;
		catatan: string;
		dicatat_oleh: string;
		created_at: string;
	}
	interface Angkatan { id: number; kode: string; keterangan: string; }
	interface Filters {
		search?: string;
		angkatan?: string;
		status?: string;
		has_product_id?: number;
		missing_product_id?: number;
		batch_id?: number;
	}
	interface Props {
		user?: User;
		products?: Product[];
		crm?: CRMRow[];
		total?: number;
		page?: number;
		limit?: number;
		mutations?: Mutation[];
		angkatan?: Angkatan[];
		tab?: string;
		filters?: Filters;
		success?: string;
		error?: string;
		dashboard?: any;
		dashboard_filters?: any;
		dashboard_error?: string;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let products = $derived(props.products ?? []);
	let crm = $derived(props.crm ?? []);
	let total = $derived(props.total ?? 0);
	let page = $derived(props.page ?? 1);
	let limit = $derived(props.limit ?? 25);
	let mutations = $derived(props.mutations ?? []);
	let angkatan = $derived(props.angkatan ?? []);
	let success = $derived(props.success);
	let error = $derived(props.error);
	let dashboard = $derived(props.dashboard);
	let dashboardFilters = $derived(props.dashboard_filters);
	let dashboardError = $derived(props.dashboard_error);

	let activeTab = $state(props.tab ?? "dashboard");
	let loading = $state<string | null>(null);

	function todayLocal() {
		const d = new Date();
		const offset = d.getTimezoneOffset();
		return new Date(d.getTime() - offset * 60000).toISOString().slice(0, 10);
	}

	let productForm = $state({ nama: "", kategori: "buku", track_stok: true, is_aktif: true, minimum_stock: 5 });
	let editingProductID = $state<number | null>(null);
	let batchForm = $state({ produk_id: 0, nama: "", tanggal_mulai: "", tanggal_selesai: "", is_aktif: true });
	let editingBatchID = $state<number | null>(null);
	let stockForm = $state({ produk_id: 0, produk_batch_id: 0, tipe: "stok_masuk", qty: 1, catatan: "" });
	let opnameForm = $state({ produk_id: 0, produk_batch_id: 0, stok_fisik: 0, catatan: "" });
	let filterForm = $state({
		search: props.filters?.search ?? "",
		angkatan: props.filters?.angkatan ?? "",
		status: props.filters?.status ?? "",
		has_product_id: props.filters?.has_product_id ?? 0,
		missing_product_id: props.filters?.missing_product_id ?? 0,
		batch_id: props.filters?.batch_id ?? 0,
	});
	let assignDialog = $state<HTMLDialogElement>();
	let assignSantri = $state<CRMRow | null>(null);
	let assignForm = $state({ produk_id: 0, produk_batch_id: 0, tanggal: todayLocal(), catatan: "" });

	let stockProduct = $derived(products.find((p) => p.id === Number(stockForm.produk_id)));
	let opnameProduct = $derived(products.find((p) => p.id === Number(opnameForm.produk_id)));
	let assignProduct = $derived(products.find((p) => p.id === Number(assignForm.produk_id)));
	let filterProduct = $derived(products.find((p) => p.id === Number(filterForm.has_product_id)));
	let totalPages = $derived(Math.max(1, Math.ceil(total / limit)));

	const inputCls = "w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3.5 py-2.5 text-sm text-neutral-900 outline-none transition focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-900 dark:text-white";

	function setTab(tab: string) {
		activeTab = tab;
		const url = new URL(window.location.href);
		url.searchParams.set("tab", tab);
		history.replaceState({}, "", url);
	}

	function resetProductForm() {
		productForm = { nama: "", kategori: "buku", track_stok: true, is_aktif: true, minimum_stock: 5 };
		editingProductID = null;
	}

	function editProduct(p: Product) {
		productForm = { nama: p.nama, kategori: p.kategori, track_stok: p.track_stok, is_aktif: p.is_aktif, minimum_stock: p.minimum_stock ?? 5 };
		editingProductID = p.id;
		setTab("produk");
	}

	function submitProduct(e: Event) {
		e.preventDefault();
		loading = "product";
		const done = () => { loading = null; };
		if (editingProductID) {
			router.put(`/app/produk-crm/produk/${editingProductID}`, productForm, { preserveScroll: true, onSuccess: resetProductForm, onFinish: done });
		} else {
			router.post("/app/produk-crm/produk", productForm, { preserveScroll: true, onSuccess: resetProductForm, onFinish: done });
		}
	}

	function editBatch(productID: number, b: Batch) {
		batchForm = { produk_id: productID, nama: b.nama, tanggal_mulai: b.tanggal_mulai, tanggal_selesai: b.tanggal_selesai, is_aktif: b.is_aktif };
		editingBatchID = b.id;
	}

	function resetBatchForm() {
		batchForm = { produk_id: 0, nama: "", tanggal_mulai: "", tanggal_selesai: "", is_aktif: true };
		editingBatchID = null;
	}

	function submitBatch(e: Event) {
		e.preventDefault();
		if (!batchForm.produk_id) return;
		loading = "batch";
		const body = { nama: batchForm.nama, tanggal_mulai: batchForm.tanggal_mulai, tanggal_selesai: batchForm.tanggal_selesai, is_aktif: batchForm.is_aktif };
		const done = () => { loading = null; };
		if (editingBatchID) {
			router.put(`/app/produk-crm/batch/${editingBatchID}`, body, { preserveScroll: true, onSuccess: resetBatchForm, onFinish: done });
		} else {
			router.post(`/app/produk-crm/produk/${batchForm.produk_id}/batch`, body, { preserveScroll: true, onSuccess: resetBatchForm, onFinish: done });
		}
	}

	function addStock(e: Event) {
		e.preventDefault();
		loading = "stock";
		router.post("/app/produk-crm/stok", stockForm, {
			preserveScroll: true,
			onSuccess: () => stockForm = { ...stockForm, qty: 1, catatan: "" },
			onFinish: () => { loading = null; },
		});
	}

	function saveOpname(e: Event) {
		e.preventDefault();
		loading = "opname";
		router.post("/app/produk-crm/opname", opnameForm, {
			preserveScroll: true,
			onSuccess: () => opnameForm = { ...opnameForm, catatan: "" },
			onFinish: () => { loading = null; },
		});
	}

	function applyFilters(pageNumber = 1) {
		router.get("/app/produk-crm", {
			tab: "crm",
			...filterForm,
			page: pageNumber,
		}, { preserveState: true, preserveScroll: true });
	}

	function openAssign(row: CRMRow) {
		assignSantri = row;
		assignForm = { produk_id: 0, produk_batch_id: 0, tanggal: todayLocal(), catatan: "" };
		assignDialog?.showModal();
	}

	function submitAssign(e: Event) {
		e.preventDefault();
		if (!assignSantri) return;
		loading = "assign";
		router.post(`/app/produk-crm/mahasantri/${assignSantri.id}/produk`, assignForm, {
			preserveScroll: true,
			onSuccess: () => assignDialog?.close(),
			onFinish: () => { loading = null; },
		});
	}

	function cancelAssignment(item: OwnedProduct) {
		if (!confirm(`Batalkan ${item.produk_nama}${item.batch_nama ? " - " + item.batch_nama : ""}? Stok buku akan dikembalikan bila relevan.`)) return;
		router.post(`/app/produk-crm/mahasantri-produk/${item.id}/batal`, {}, { preserveScroll: true });
	}
</script>

<AppLayout {user}>
	<div class="border-b border-neutral-200/80 px-4 pb-6 pt-6 dark:border-white/[0.04] sm:px-6 sm:pt-8">
		<div class="mx-auto flex max-w-7xl items-start gap-3">
			<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-400/10 text-brand-600 dark:text-brand-400"><Boxes size="22" /></div>
			<div>
				<h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white sm:text-3xl">Produk & CRM Mahasantri</h1>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Kelola produk, batch/edisi, stok buku, dan riwayat kepemilikan produk Mahasantri.</p>
			</div>
		</div>
	</div>

	<div class="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6">
		{#if success}<div class="rounded-xl border border-green-500/20 bg-green-500/10 px-4 py-3 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm font-medium text-red-600 dark:text-red-400">{error}</div>{/if}

		<div class="flex gap-2 overflow-x-auto rounded-2xl border border-neutral-200 bg-white p-2 dark:border-white/[0.06] dark:bg-neutral-925/50">
			{#each [
				{ id: "dashboard", label: "Dashboard", icon: BarChart3 },
				{ id: "produk", label: "Master Produk", icon: ShoppingBag },
				{ id: "stok", label: "Stok & Opname", icon: Boxes },
				{ id: "crm", label: "CRM Mahasantri", icon: ClipboardCheck },
				{ id: "riwayat", label: "Riwayat", icon: History },
			] as tab}
				{@const Icon = tab.icon}
				<button onclick={() => setTab(tab.id)} class="inline-flex shrink-0 items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition {activeTab === tab.id ? 'bg-brand-600 text-white' : 'text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800'}">
					<Icon size="16" /> {tab.label}
				</button>
			{/each}
		</div>

		{#if activeTab === "dashboard"}
			<ProductCRMDashboard {dashboard} filters={dashboardFilters} {products} {angkatan} error={dashboardError} />
		{:else if activeTab === "produk"}
			<div class="grid gap-5 lg:grid-cols-[360px_1fr]">
				<div class="space-y-5">
					<form onsubmit={submitProduct} class="space-y-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
						<div class="flex items-center justify-between"><h2 class="font-semibold text-neutral-900 dark:text-white">{editingProductID ? "Edit Produk" : "Tambah Produk"}</h2>{#if editingProductID}<button type="button" onclick={resetProductForm} class="text-xs text-neutral-500">Batal edit</button>{/if}</div>
						<input bind:value={productForm.nama} required placeholder="Nama produk" class={inputCls} />
						<select bind:value={productForm.kategori} onchange={() => { if (productForm.kategori === "program") productForm.track_stok = false; }} class={inputCls}>
							<option value="buku">Buku</option><option value="program">Program</option>
						</select>
						<label class="flex items-center gap-2 text-sm text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={productForm.track_stok} disabled={productForm.kategori === "program"} /> Kelola stok fisik</label>
						{#if productForm.track_stok && productForm.kategori === "buku"}
							<label class="block space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Minimum stok untuk alert<input type="number" min="0" bind:value={productForm.minimum_stock} class={inputCls} placeholder="Mis. 5" /></label>
						{/if}
						{#if editingProductID}<label class="flex items-center gap-2 text-sm text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={productForm.is_aktif} /> Produk aktif</label>{/if}
						<button disabled={loading === "product"} class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"><Plus size="16" /> {editingProductID ? "Simpan Perubahan" : "Tambah Produk"}</button>
					</form>

					<form onsubmit={submitBatch} class="space-y-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
						<div class="flex items-center justify-between"><h2 class="font-semibold text-neutral-900 dark:text-white">{editingBatchID ? "Edit Batch/Edisi" : "Tambah Batch/Edisi"}</h2>{#if editingBatchID}<button type="button" onclick={resetBatchForm} class="text-xs text-neutral-500">Batal edit</button>{/if}</div>
						<select bind:value={batchForm.produk_id} disabled={editingBatchID !== null} required class={inputCls}><option value={0}>Pilih produk</option>{#each products as p}<option value={p.id}>{p.nama}</option>{/each}</select>
						<input bind:value={batchForm.nama} required placeholder="Mis. Batch 1 / Edisi 2026" class={inputCls} />
						<div class="grid grid-cols-2 gap-2"><input type="date" bind:value={batchForm.tanggal_mulai} class={inputCls} /><input type="date" bind:value={batchForm.tanggal_selesai} class={inputCls} /></div>
						{#if editingBatchID}<label class="flex items-center gap-2 text-sm text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={batchForm.is_aktif} /> Batch aktif</label>{/if}
						<button disabled={loading === "batch"} class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-secondary-500 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"><CalendarDays size="16" /> Simpan Batch/Edisi</button>
					</form>
				</div>

				<div class="space-y-3">
					{#each products as p (p.id)}
						<section class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
							<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
								<div>
									<div class="flex flex-wrap items-center gap-2">
										<h3 class="font-semibold text-neutral-900 dark:text-white">{p.nama}</h3>
										<span class="rounded-full px-2 py-0.5 text-xs font-semibold {p.kategori === 'buku' ? 'bg-amber-500/10 text-amber-600' : 'bg-blue-500/10 text-blue-600'}">{p.kategori === "buku" ? "Buku" : "Program"}</span>
										<span class="rounded-full px-2 py-0.5 text-xs {p.is_aktif ? 'bg-green-500/10 text-green-600' : 'bg-neutral-500/10 text-neutral-500'}">{p.is_aktif ? "Aktif" : "Nonaktif"}</span>
									</div>
									<p class="mt-1 text-sm text-neutral-500">{p.track_stok ? `Stok total: ${p.stok} · minimum ${p.minimum_stock ?? 0}` : "Tanpa stok fisik"}</p>
								</div>
								<button onclick={() => editProduct(p)} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-3 py-2 text-xs font-semibold text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"><Pencil size="14" /> Edit</button>
							</div>
							{#if p.batches.length}
								<div class="mt-4 flex flex-wrap gap-2">
									{#each p.batches as b}
										<button onclick={() => editBatch(p.id, b)} class="rounded-xl border border-neutral-200 px-3 py-2 text-left text-xs dark:border-neutral-700">
											<span class="font-semibold text-neutral-800 dark:text-neutral-200">{b.nama}</span>
											<span class="ml-1 text-neutral-500">{b.is_aktif ? "aktif" : "nonaktif"}</span>
										</button>
									{/each}
								</div>
							{:else}
								<p class="mt-4 text-xs text-neutral-500">Belum ada batch/edisi.</p>
							{/if}
						</section>
					{:else}
						<div class="rounded-2xl border border-dashed border-neutral-300 p-10 text-center text-sm text-neutral-500 dark:border-neutral-700">Belum ada produk.</div>
					{/each}
				</div>
			</div>
		{:else if activeTab === "stok"}
			<div class="grid gap-5 lg:grid-cols-2">
				<form onsubmit={addStock} class="space-y-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
					<div><h2 class="font-semibold text-neutral-900 dark:text-white">Stok Masuk</h2><p class="mt-1 text-xs text-neutral-500">Gunakan untuk stok awal atau restock. Pembelian Mahasantri otomatis mengurangi stok.</p></div>
					<select bind:value={stockForm.produk_id} onchange={() => stockForm.produk_batch_id = 0} required class={inputCls}><option value={0}>Pilih buku</option>{#each products.filter((p) => p.track_stok) as p}<option value={p.id}>{p.nama} · stok {p.stok}</option>{/each}</select>
					{#if stockProduct?.batches?.length}<select bind:value={stockForm.produk_batch_id} required class={inputCls}><option value={0}>Pilih batch/edisi</option>{#each stockProduct.batches.filter((b) => b.is_aktif) as b}<option value={b.id}>{b.nama}</option>{/each}</select>{/if}
					<select bind:value={stockForm.tipe} class={inputCls}><option value="stok_masuk">Stok masuk</option><option value="stok_awal">Stok awal</option></select>
					<input type="number" min="1" bind:value={stockForm.qty} required class={inputCls} placeholder="Jumlah" />
					<textarea bind:value={stockForm.catatan} class={inputCls} rows="2" placeholder="Catatan"></textarea>
					<button disabled={loading === "stock"} class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white"><PackagePlus size="16" /> Tambah Stok</button>
				</form>

				<form onsubmit={saveOpname} class="space-y-4 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
					<div><h2 class="font-semibold text-neutral-900 dark:text-white">Stok Opname</h2><p class="mt-1 text-xs text-neutral-500">Masukkan jumlah fisik. Sistem membuat penyesuaian otomatis dan menyimpan audit trail.</p></div>
					<select bind:value={opnameForm.produk_id} onchange={() => opnameForm.produk_batch_id = 0} required class={inputCls}><option value={0}>Pilih buku</option>{#each products.filter((p) => p.track_stok) as p}<option value={p.id}>{p.nama} · stok total {p.stok}</option>{/each}</select>
					{#if opnameProduct?.batches?.length}<select bind:value={opnameForm.produk_batch_id} required class={inputCls}><option value={0}>Pilih batch/edisi</option>{#each opnameProduct.batches.filter((b) => b.is_aktif) as b}<option value={b.id}>{b.nama}</option>{/each}</select>{/if}
					<input type="number" min="0" bind:value={opnameForm.stok_fisik} required class={inputCls} placeholder="Stok fisik" />
					<textarea bind:value={opnameForm.catatan} class={inputCls} rows="2" placeholder="Catatan opname"></textarea>
					<button disabled={loading === "opname"} class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-secondary-500 px-4 py-2.5 text-sm font-semibold text-white"><Archive size="16" /> Simpan Opname</button>
				</form>
			</div>

			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				{#each products.filter((p) => p.track_stok) as p}
					<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50"><p class="text-xs text-neutral-500">{p.nama}</p><p class="mt-2 text-2xl font-bold text-neutral-900 dark:text-white">{p.stok}</p><p class="text-xs text-neutral-500">stok tercatat</p></div>
				{/each}
			</div>
		{:else if activeTab === "crm"}
			<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="grid gap-2 md:grid-cols-3 lg:grid-cols-6">
					<div class="relative lg:col-span-2"><Search size="16" class="absolute left-3 top-3 text-neutral-400" /><input bind:value={filterForm.search} class="{inputCls} pl-9" placeholder="Nama / ID Mahasantri" /></div>
					<select bind:value={filterForm.angkatan} class={inputCls}><option value="">Semua angkatan</option>{#each angkatan as a}<option value={a.kode}>{a.keterangan || a.kode}</option>{/each}</select>
					<select bind:value={filterForm.status} class={inputCls}><option value="">Semua status</option><option value="aktif">Aktif</option><option value="cuti">Cuti</option><option value="nonaktif">Nonaktif</option><option value="tidak_lanjut">Tidak lanjut</option></select>
					<select bind:value={filterForm.has_product_id} onchange={() => filterForm.batch_id = 0} class={inputCls}><option value={0}>Sudah punya...</option>{#each products as p}<option value={p.id}>{p.nama}</option>{/each}</select>
					<select bind:value={filterForm.missing_product_id} class={inputCls}><option value={0}>Belum punya...</option>{#each products as p}<option value={p.id}>{p.nama}</option>{/each}</select>
				</div>
				{#if filterProduct?.batches?.length}<div class="mt-2 max-w-sm"><select bind:value={filterForm.batch_id} class={inputCls}><option value={0}>Semua batch/edisi</option>{#each filterProduct.batches as b}<option value={b.id}>{b.nama}</option>{/each}</select></div>{/if}
				<div class="mt-3 flex justify-end"><button onclick={() => applyFilters()} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white">Terapkan Filter</button></div>
			</div>

			<div class="overflow-hidden rounded-2xl border border-neutral-200 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="overflow-x-auto">
					<table class="min-w-full text-sm">
						<thead class="bg-neutral-50 text-left text-xs uppercase tracking-wide text-neutral-500 dark:bg-neutral-900/70"><tr><th class="px-4 py-3">Mahasantri</th><th class="px-4 py-3">Kelas</th><th class="px-4 py-3">Produk / Program</th><th class="px-4 py-3 text-right">Aksi</th></tr></thead>
						<tbody class="divide-y divide-neutral-200 dark:divide-white/[0.05]">
							{#each crm as row (row.id)}
								<tr>
									<td class="px-4 py-4"><a href={`/app/santri/${row.id}`} class="font-semibold text-brand-600 hover:underline">{row.nama}</a><p class="mt-1 font-mono text-xs text-neutral-500">{row.id_mahasantri || "-"}</p></td>
									<td class="px-4 py-4 text-neutral-600 dark:text-neutral-300"><p>{row.angkatan_kelas || row.angkatan || "-"}</p><p class="text-xs text-neutral-500">{row.level || "-"} · {row.status}</p></td>
									<td class="px-4 py-4">
										<div class="flex max-w-xl flex-wrap gap-2">
											{#each row.produk as item}
												<span class="inline-flex items-center gap-1.5 rounded-full bg-brand-500/10 px-2.5 py-1 text-xs font-medium text-brand-700 dark:text-brand-300">
													<CheckCircle2 size="13" /> {item.produk_nama}{#if item.batch_nama} · {item.batch_nama}{/if}
													<button onclick={() => cancelAssignment(item)} class="ml-1 rounded-full p-0.5 hover:bg-red-500/15 hover:text-red-500" aria-label="Batalkan"><X size="12" /></button>
												</span>
											{:else}<span class="text-xs text-neutral-400">Belum ada produk</span>{/each}
										</div>
									</td>
									<td class="px-4 py-4 text-right"><button onclick={() => openAssign(row)} class="inline-flex items-center gap-1.5 rounded-lg border border-brand-500/30 px-3 py-2 text-xs font-semibold text-brand-600"><Plus size="14" /> Catat Produk</button></td>
								</tr>
							{:else}<tr><td colspan="4" class="px-4 py-10 text-center text-neutral-500">Tidak ada Mahasantri sesuai filter.</td></tr>{/each}
						</tbody>
					</table>
				</div>
				<div class="flex items-center justify-between border-t border-neutral-200 px-4 py-3 text-sm dark:border-white/[0.05]"><span class="text-neutral-500">{total} Mahasantri</span><div class="flex gap-2"><button disabled={page <= 1} onclick={() => applyFilters(page - 1)} class="rounded-lg border px-3 py-1.5 disabled:opacity-40 dark:border-neutral-700">Sebelumnya</button><span class="px-2 py-1.5">{page} / {totalPages}</span><button disabled={page >= totalPages} onclick={() => applyFilters(page + 1)} class="rounded-lg border px-3 py-1.5 disabled:opacity-40 dark:border-neutral-700">Berikutnya</button></div></div>
			</div>
		{:else}
			<div class="overflow-hidden rounded-2xl border border-neutral-200 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
				<div class="border-b border-neutral-200 px-5 py-4 dark:border-white/[0.05]"><h2 class="font-semibold text-neutral-900 dark:text-white">100 Mutasi Stok Terakhir</h2></div>
				<div class="overflow-x-auto"><table class="min-w-full text-sm"><thead class="bg-neutral-50 text-left text-xs uppercase text-neutral-500 dark:bg-neutral-900/70"><tr><th class="px-4 py-3">Waktu</th><th class="px-4 py-3">Produk</th><th class="px-4 py-3">Tipe</th><th class="px-4 py-3">Qty</th><th class="px-4 py-3">Catatan</th><th class="px-4 py-3">Admin</th></tr></thead><tbody class="divide-y divide-neutral-200 dark:divide-white/[0.05]">{#each mutations as m}<tr><td class="px-4 py-3 text-neutral-500">{m.created_at}</td><td class="px-4 py-3 font-medium">{m.produk_nama}{#if m.batch_nama}<span class="text-neutral-500"> · {m.batch_nama}</span>{/if}</td><td class="px-4 py-3">{m.tipe.replaceAll("_", " ")}</td><td class="px-4 py-3 font-mono font-bold {m.qty > 0 ? 'text-green-600' : 'text-red-500'}">{m.qty > 0 ? "+" : ""}{m.qty}</td><td class="px-4 py-3 text-neutral-500">{m.catatan || "-"}</td><td class="px-4 py-3 text-neutral-500">{m.dicatat_oleh || "-"}</td></tr>{:else}<tr><td colspan="6" class="px-4 py-10 text-center text-neutral-500">Belum ada mutasi stok.</td></tr>{/each}</tbody></table></div>
			</div>
		{/if}
	</div>

	<dialog bind:this={assignDialog} class="m-auto w-[calc(100%-2rem)] max-w-lg rounded-2xl border border-neutral-200 bg-white p-0 shadow-xl backdrop:bg-neutral-950/70 dark:border-white/[0.06] dark:bg-neutral-925">
		<form onsubmit={submitAssign} class="space-y-4 p-6">
			<div><h2 class="text-lg font-semibold text-neutral-900 dark:text-white">Catat Produk Mahasantri</h2><p class="mt-1 text-sm text-neutral-500">{assignSantri?.nama} · {assignSantri?.id_mahasantri}</p></div>
			<select bind:value={assignForm.produk_id} onchange={() => assignForm.produk_batch_id = 0} required class={inputCls}><option value={0}>Pilih produk / program</option>{#each products.filter((p) => p.is_aktif) as p}<option value={p.id}>{p.nama}{p.track_stok ? ` · stok ${p.stok}` : ""}</option>{/each}</select>
			{#if assignProduct?.batches?.filter((b) => b.is_aktif).length}<select bind:value={assignForm.produk_batch_id} required class={inputCls}><option value={0}>Pilih batch/edisi</option>{#each assignProduct.batches.filter((b) => b.is_aktif) as b}<option value={b.id}>{b.nama}</option>{/each}</select>{/if}
			<input type="date" bind:value={assignForm.tanggal} required class={inputCls} />
			<textarea bind:value={assignForm.catatan} rows="2" placeholder="Catatan opsional" class={inputCls}></textarea>
			<div class="flex justify-end gap-2"><button type="button" onclick={() => assignDialog?.close()} class="rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-semibold dark:border-neutral-700">Batal</button><button disabled={loading === "assign"} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50">Simpan</button></div>
		</form>
	</dialog>
</AppLayout>
