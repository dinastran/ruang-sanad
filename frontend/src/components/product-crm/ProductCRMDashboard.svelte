<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import BarChart from "@components/charts/BarChart.svelte";
	import DonutChart from "@components/charts/DonutChart.svelte";
	import {
		AlertCircle,
		ArrowRight,
		BarChart3,
		Boxes,
		CalendarDays,
		CheckCircle2,
		ChevronLeft,
		ChevronRight,
		Filter,
		MessageCircle,
		PackageSearch,
		RotateCcw,
		ShoppingBag,
		Users,
		X,
	} from "lucide-svelte";

	interface Batch { id: number; nama: string; is_aktif: boolean }
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
	interface Angkatan { id: number; kode: string; keterangan: string }
	interface Point { label: string; value: string; total: number }
	interface TrendPoint { period: string; label: string; total: number }
	interface StockMovement { period: string; label: string; masuk: number; keluar: number }
	interface InventoryHealth { product_id: number; product_name: string; stock: number; minimum_stock: number; status: string }
	interface DashboardFilters {
		date_from: string;
		date_to: string;
		product_id: number;
		batch_id: number;
		category: string;
		angkatan: string;
		status: string;
	}
	interface OwnedPair { product_id: number; batch_id: number }
	interface DrillRow {
		id: number;
		id_mahasantri: string;
		nama: string;
		no_wa: string;
		angkatan: string;
		angkatan_kelas: string;
		level: string;
		status: string;
		owned: OwnedPair[];
	}
	interface Drilldown {
		data: DrillRow[];
		total: number;
		page: number;
		limit: number;
		mode: string;
		period: string;
	}
	interface Dashboard {
		summary: {
			active_products: number;
			santri_total: number;
			santri_with_product: number;
			coverage_rate: number;
			active_ownerships: number;
			total_stock: number;
			low_stock_products: number;
			period_assignments: number;
			previous_assignments: number;
			assignment_change_rate: number;
		};
		distribution_trend: TrendPoint[];
		top_products: Point[];
		category_composition: Point[];
		coverage_by_angkatan: Point[];
		stock_movement: StockMovement[];
		inventory_health: InventoryHealth[];
	}
	interface Offer {
		key: string;
		productID: number;
		batchID: number;
		productName: string;
		batchName: string;
		category: string;
		label: string;
	}
	interface Props {
		dashboard?: Dashboard;
		filters?: DashboardFilters;
		products?: Product[];
		angkatan?: Angkatan[];
		drilldown?: Drilldown;
		drilldownMode?: string;
		drilldownPeriod?: string;
		error?: string;
		drilldownError?: string;
	}

	let {
		dashboard,
		filters,
		products = [],
		angkatan = [],
		drilldown,
		drilldownMode = "",
		drilldownPeriod = "",
		error,
		drilldownError,
	}: Props = $props();

	let draftProductID = $state(0);
	let draftBatchID = $state(0);
	let selectedOffer = $state<Record<number, string>>({});

	$effect(() => {
		draftProductID = Number(filters?.product_id || 0);
		draftBatchID = Number(filters?.batch_id || 0);
	});

	let selectedProduct = $derived(products.find((p) => p.id === Number(filters?.product_id || 0)));
	let draftProduct = $derived(products.find((p) => p.id === Number(draftProductID || 0)));
	let summary = $derived(dashboard?.summary ?? {
		active_products: 0,
		santri_total: 0,
		santri_with_product: 0,
		coverage_rate: 0,
		active_ownerships: 0,
		total_stock: 0,
		low_stock_products: 0,
		period_assignments: 0,
		previous_assignments: 0,
		assignment_change_rate: 0,
	});
	let trend = $derived(dashboard?.distribution_trend ?? []);
	let trendMax = $derived(Math.max(...trend.map((x) => x.total), 1));
	let trendCoords = $derived(trend.map((point, index) => {
		const width = 760, height = 260, left = 44, right = 20, top = 22, bottom = 48;
		const usableW = width - left - right;
		const usableH = height - top - bottom;
		return {
			...point,
			x: trend.length <= 1 ? width / 2 : left + (index / (trend.length - 1)) * usableW,
			y: top + usableH - (point.total / trendMax) * usableH,
		};
	}));
	let trendPolyline = $derived(trendCoords.map((x) => `${x.x},${x.y}`).join(" "));
	let movementMax = $derived(Math.max(...(dashboard?.stock_movement ?? []).flatMap((x) => [x.masuk, x.keluar]), 1));
	let drillTotalPages = $derived(Math.max(1, Math.ceil((drilldown?.total || 0) / (drilldown?.limit || 20))));
	let drillStart = $derived((drilldown?.total || 0) > 0 ? ((drilldown?.page || 1) - 1) * (drilldown?.limit || 20) + 1 : 0);
	let drillEnd = $derived(Math.min((drilldown?.page || 1) * (drilldown?.limit || 20), drilldown?.total || 0));

	function params() {
		return new URLSearchParams(typeof window !== "undefined" ? window.location.search : "");
	}
	function navigate(p: URLSearchParams, preserveScroll = true) {
		p.set("tab", "dashboard");
		router.get(`/app/produk-crm?${p.toString()}`, {}, { preserveScroll });
	}
	function clearDrillParams(p: URLSearchParams) {
		p.delete("drill_page");
		p.delete("drill_period");
	}
	function applyFilters(event: SubmitEvent) {
		event.preventDefault();
		const data = new FormData(event.currentTarget as HTMLFormElement);
		const p = new URLSearchParams();
		p.set("tab", "dashboard");
		for (const [key, raw] of data.entries()) {
			const value = String(raw).trim();
			if (value && value !== "0") p.set(key, value);
		}
		p.set("drilldown", "period");
		navigate(p, false);
	}
	function setFilter(key: string, value: string | number, mode = "period") {
		const p = params();
		clearDrillParams(p);
		p.delete("page");
		if (value === "" || value === 0) p.delete(key);
		else p.set(key, String(value));
		if (key === "product_id") p.delete("dashboard_batch_id");
		p.set("drilldown", mode);
		navigate(p);
	}
	function removeFilter(key: string) {
		const p = params();
		p.delete(key);
		if (key === "product_id") p.delete("dashboard_batch_id");
		clearDrillParams(p);
		p.set("drilldown", "period");
		navigate(p);
	}
	function reset() {
		router.get("/app/produk-crm?tab=dashboard");
	}
	function quickDays(days: number) {
		const to = new Date();
		const from = new Date();
		from.setDate(to.getDate() - days + 1);
		const local = (d: Date) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
		const p = params();
		p.set("date_from", local(from));
		p.set("date_to", local(to));
		p.set("drilldown", "period");
		clearDrillParams(p);
		navigate(p, false);
	}
	function quickMonth() {
		const now = new Date();
		const from = new Date(now.getFullYear(), now.getMonth(), 1);
		const local = (d: Date) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
		const p = params();
		p.set("date_from", local(from));
		p.set("date_to", local(now));
		p.set("drilldown", "period");
		clearDrillParams(p);
		navigate(p, false);
	}
	function showDrilldown(mode: "all" | "has" | "period" | "missing", period = "") {
		const p = params();
		p.set("drilldown", mode);
		p.delete("drill_page");
		if (period) p.set("drill_period", period);
		else p.delete("drill_period");
		navigate(p);
	}
	function selectTrendPeriod(period: string) {
		showDrilldown("period", period);
	}
	function clearDrilldown() {
		const p = params();
		p.delete("drilldown");
		p.delete("drill_page");
		p.delete("drill_period");
		navigate(p);
	}
	function goDrillPage(page: number) {
		if (!drilldown || page < 1 || page > drillTotalPages) return;
		const p = params();
		p.set("drill_page", String(page));
		navigate(p);
	}
	function openStock() {
		router.get("/app/produk-crm?tab=stok");
	}
	function periodLabel(value: string) {
		if (/^\d{4}-W\d{2}$/.test(value)) {
			const [year, week] = value.split("-W");
			return `W${week} ${year}`;
		}
		if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
			return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short" }).format(new Date(value + "T00:00:00"));
		}
		if (/^\d{4}-\d{2}$/.test(value)) {
			const [year, month] = value.split("-");
			return new Intl.DateTimeFormat("id-ID", { month: "short", year: "2-digit" }).format(new Date(Number(year), Number(month) - 1, 1));
		}
		return value;
	}
	function fmt(value: number) { return Number(value || 0).toLocaleString("id-ID"); }
	function pct(value: number) { return `${Number(value || 0).toFixed(1).replace(".0", "")}%`; }
	function healthLabel(status: string) {
		if (status === "empty") return "Habis";
		if (status === "critical") return "Perlu restock";
		return "Aman";
	}
	function healthClass(status: string) {
		if (status === "empty") return "bg-red-500/10 text-red-600 dark:text-red-400";
		if (status === "critical") return "bg-amber-500/10 text-amber-700 dark:text-amber-400";
		return "bg-green-500/10 text-green-700 dark:text-green-400";
	}
	function drillTitle() {
		if (drilldownMode === "missing") return selectedProduct ? `Belum memiliki ${selectedProduct.nama}` : "Belum memiliki produk";
		if (drilldownMode === "has") return selectedProduct ? `Memiliki ${selectedProduct.nama}` : "Memiliki produk";
		if (drilldownMode === "period") return drilldownPeriod ? `Distribusi ${periodLabel(drilldownPeriod)}` : "Mahasantri pada periode terpilih";
		return "Mahasantri sesuai filter";
	}
	function activeBatchLabel() {
		if (!selectedProduct || !filters?.batch_id) return "";
		return selectedProduct.batches.find((b) => b.id === Number(filters.batch_id))?.nama || "";
	}
	function isOwned(row: DrillRow, productID: number, batchID: number) {
		return (row.owned || []).some((item) => item.product_id === productID && Number(item.batch_id || 0) === Number(batchID || 0));
	}
	function offersFor(row: DrillRow): Offer[] {
		const targetMissingProduct = drilldownMode === "missing" ? Number(filters?.product_id || 0) : 0;
		const targetMissingBatch = drilldownMode === "missing" ? Number(filters?.batch_id || 0) : 0;
		const base = products.filter((p) => {
			if (!p.is_aktif) return false;
			if (targetMissingProduct && p.id !== targetMissingProduct) return false;
			if (filters?.category && p.kategori !== filters.category) return false;
			return true;
		});
		const out: Offer[] = [];
		for (const product of base) {
			const activeBatches = (product.batches || []).filter((b) => b.is_aktif && (!targetMissingBatch || b.id === targetMissingBatch));
			if ((product.batches || []).some((b) => b.is_aktif)) {
				for (const batch of activeBatches) {
					if (!isOwned(row, product.id, batch.id)) {
						out.push({
							key: `${product.id}:${batch.id}`,
							productID: product.id,
							batchID: batch.id,
							productName: product.nama,
							batchName: batch.nama,
							category: product.kategori,
							label: `${product.nama} · ${batch.nama}`,
						});
					}
			}
			} else if (!isOwned(row, product.id, 0)) {
				out.push({
					key: `${product.id}:0`,
					productID: product.id,
					batchID: 0,
					productName: product.nama,
					batchName: "",
					category: product.kategori,
					label: product.nama,
				});
			}
		}
		return out;
	}
	function offerFor(row: DrillRow): Offer | undefined {
		const offers = offersFor(row);
		const key = selectedOffer[row.id] || offers[0]?.key;
		return offers.find((offer) => offer.key === key) || offers[0];
	}
	function setOffer(rowID: number, value: string) {
		selectedOffer[rowID] = value;
	}
	function normalizeWA(value: string) {
		let digits = String(value || "").replace(/\D/g, "");
		if (!digits) return "";
		if (digits.startsWith("0")) digits = "62" + digits.slice(1);
		else if (digits.startsWith("8")) digits = "62" + digits;
		return digits;
	}
	function waURL(row: DrillRow, offer?: Offer) {
		const phone = normalizeWA(row.no_wa);
		if (!phone || !offer) return "";
		const kind = offer.category === "program" ? "program" : "buku";
		const name = offer.batchName ? `${offer.productName} (${offer.batchName})` : offer.productName;
		const message = `Assalamu'alaikum Kak ${row.nama}, saya dari Ruang Sanad. Kami ingin menawarkan ${kind} ${name}. Berdasarkan data kami, ${name} belum tercatat pada akun Kakak. Kalau berkenan, saya bisa kirim detailnya di sini.`;
		return `https://wa.me/${phone}?text=${encodeURIComponent(message)}`;
	}
	function ownedLabel(row: DrillRow) {
		const labels: string[] = [];
		for (const owned of row.owned || []) {
			const product = products.find((p) => p.id === owned.product_id);
			if (!product) continue;
			const batch = product.batches?.find((b) => b.id === owned.batch_id);
			labels.push(batch ? `${product.nama} · ${batch.nama}` : product.nama);
		}
		return labels;
	}
</script>

{#if error}
	<div class="rounded-2xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-400">{error}</div>
{/if}

<section class="overflow-hidden rounded-2xl border border-neutral-200 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
	<div class="border-b border-neutral-200 p-4 dark:border-white/[0.05] sm:p-5">
		<div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
			<div class="flex items-start gap-3">
				<div class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-brand-500/10 text-brand-600 dark:text-brand-400"><Filter size="17" /></div>
				<div>
					<h2 class="font-semibold text-neutral-900 dark:text-white">Filter Intelligence</h2>
					<p class="mt-1 max-w-2xl text-xs leading-5 text-neutral-500">Filter diterapkan ke card, grafik, dan data Mahasantri. Klik card atau grafik untuk membuka drill-down Mahasantri yang terkait.</p>
				</div>
			</div>
			<div class="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
				<button type="button" onclick={() => quickDays(7)} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium hover:border-brand-300 hover:text-brand-600 dark:border-white/[0.08]">7 hari</button>
				<button type="button" onclick={() => quickDays(30)} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium hover:border-brand-300 hover:text-brand-600 dark:border-white/[0.08]">30 hari</button>
				<button type="button" onclick={quickMonth} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium hover:border-brand-300 hover:text-brand-600 dark:border-white/[0.08]">Bulan ini</button>
				<button type="button" onclick={reset} class="inline-flex items-center justify-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium hover:border-brand-300 hover:text-brand-600 dark:border-white/[0.08]"><RotateCcw size="13" /> Reset</button>
			</div>
		</div>
	</div>

	<form onsubmit={applyFilters} class="space-y-4 p-4 sm:p-5">
		<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Dari tanggal
				<input name="date_from" type="date" value={filters?.date_from || ""} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white" />
			</label>
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Sampai tanggal
				<input name="date_to" type="date" value={filters?.date_to || ""} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white" />
			</label>
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Kategori
				<select name="category" value={filters?.category || ""} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua kategori</option><option value="buku">Buku</option><option value="program">Program</option></select>
			</label>
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Status Mahasantri
				<select name="status" value={filters?.status || ""} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua status</option><option value="aktif">Aktif</option><option value="cuti">Cuti</option><option value="nonaktif">Nonaktif</option><option value="tidak_lanjut">Tidak lanjut</option></select>
			</label>
		</div>
		<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Produk / program
				<select name="product_id" bind:value={draftProductID} onchange={() => (draftBatchID = 0)} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value={0}>Semua produk</option>{#each products.filter((p) => p.is_aktif) as p}<option value={p.id}>{p.nama}</option>{/each}</select>
			</label>
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Batch / edisi
				<select name="dashboard_batch_id" bind:value={draftBatchID} disabled={!draftProduct?.batches?.length} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value={0}>Semua batch / edisi</option>{#each draftProduct?.batches || [] as b}<option value={b.id}>{b.nama}</option>{/each}</select>
			</label>
			<label class="space-y-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400">Angkatan
				<select name="angkatan" value={filters?.angkatan || ""} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm font-normal text-neutral-900 outline-none focus:border-brand-400 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua angkatan</option>{#each angkatan as a}<option value={a.kode}>{a.keterangan || a.kode}</option>{/each}</select>
			</label>
		</div>
		<div class="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
			<div class="flex flex-wrap gap-2">
				{#if filters?.product_id}<button type="button" onclick={() => removeFilter("product_id")} class="inline-flex items-center gap-1 rounded-full bg-brand-500/10 px-2.5 py-1.5 text-xs font-medium text-brand-700 dark:text-brand-300">{selectedProduct?.nama || "Produk"} <X size="12" /></button>{/if}
				{#if filters?.batch_id}<span class="rounded-full bg-neutral-100 px-2.5 py-1.5 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{activeBatchLabel()}</span>{/if}
				{#if filters?.category}<button type="button" onclick={() => removeFilter("category")} class="inline-flex items-center gap-1 rounded-full bg-neutral-100 px-2.5 py-1.5 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{filters.category === "buku" ? "Buku" : "Program"} <X size="12" /></button>{/if}
				{#if filters?.angkatan}<button type="button" onclick={() => removeFilter("angkatan")} class="inline-flex items-center gap-1 rounded-full bg-neutral-100 px-2.5 py-1.5 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{filters.angkatan} <X size="12" /></button>{/if}
				{#if filters?.status}<button type="button" onclick={() => removeFilter("status")} class="inline-flex items-center gap-1 rounded-full bg-neutral-100 px-2.5 py-1.5 text-xs capitalize text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{filters.status.replaceAll("_", " ")} <X size="12" /></button>{/if}
			</div>
			<button type="submit" class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 sm:w-auto"><Filter size="15" /> Terapkan Filter & Tampilkan Mahasantri</button>
		</div>
	</form>
</section>

<section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
	<button type="button" onclick={() => showDrilldown("has")} class="group rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-5"><div class="flex items-center justify-between gap-2"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Produk aktif</p><ShoppingBag size="17" class="text-brand-500" /></div><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{fmt(summary.active_products)}</p><p class="mt-1 text-xs text-neutral-500">Klik lihat Mahasantri terkait</p></button>
	<button type="button" onclick={() => showDrilldown("has")} class="group rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-5"><div class="flex items-center justify-between gap-2"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Mahasantri punya produk</p><Users size="17" class="text-brand-500" /></div><div class="mt-3 flex items-end gap-2"><p class="text-2xl font-bold text-neutral-900 dark:text-white">{fmt(summary.santri_with_product)}</p><span class="pb-0.5 text-xs font-semibold text-brand-600">{pct(summary.coverage_rate)}</span></div><p class="mt-1 text-xs text-neutral-500">dari {fmt(summary.santri_total)} Mahasantri</p></button>
	<button type="button" onclick={() => showDrilldown("has")} class="group rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-5"><div class="flex items-center justify-between gap-2"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Kepemilikan aktif</p><CheckCircle2 size="17" class="text-brand-500" /></div><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{fmt(summary.active_ownerships)}</p><p class="mt-1 text-xs text-neutral-500">Produk / program aktif tercatat</p></button>
	<button type="button" onclick={() => showDrilldown("has")} class="group rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-5"><div class="flex items-center justify-between gap-2"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Total stok buku</p><Boxes size="17" class="text-brand-500" /></div><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{fmt(summary.total_stock)}</p><p class="mt-1 text-xs text-neutral-500">Snapshot stok saat ini</p></button>
	<button type="button" onclick={() => showDrilldown("has")} class="group rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-5"><div class="flex items-center justify-between gap-2"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Perlu restock</p><AlertCircle size="17" class={summary.low_stock_products > 0 ? "text-amber-500" : "text-green-500"} /></div><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{fmt(summary.low_stock_products)}</p><p class="mt-1 text-xs text-neutral-500">Stok ≤ minimum atau habis</p></button>
	<button type="button" onclick={() => showDrilldown("period")} class="group rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-5"><div class="flex items-center justify-between gap-2"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Distribusi periode</p><CalendarDays size="17" class="text-brand-500" /></div><div class="mt-3 flex items-end gap-2"><p class="text-2xl font-bold text-neutral-900 dark:text-white">{fmt(summary.period_assignments)}</p><span class="pb-0.5 text-xs font-semibold {summary.assignment_change_rate >= 0 ? 'text-green-600' : 'text-red-500'}">{summary.assignment_change_rate >= 0 ? "+" : ""}{pct(summary.assignment_change_rate)}</span></div><p class="mt-1 text-xs text-neutral-500">Periode sebelumnya {fmt(summary.previous_assignments)}</p></button>
</section>

<section class="grid gap-4 xl:grid-cols-[1.45fr_0.85fr]">
	<div class="min-w-0 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-6">
		<div class="mb-4 flex items-start justify-between gap-3"><div><h3 class="font-semibold text-neutral-900 dark:text-white">Trend Distribusi Produk</h3><p class="mt-1 text-xs leading-5 text-neutral-500">Klik titik grafik untuk melihat Mahasantri pada periode tersebut.</p></div><CalendarDays size="17" class="shrink-0 text-brand-500" /></div>
		{#if trend.length}
			<div class="w-full overflow-hidden">
				<svg viewBox="0 0 760 260" class="h-auto w-full min-w-0" role="img" aria-label="Trend distribusi produk">
					{#each [0, 0.5, 1] as ratio}{@const y = 22 + (260 - 22 - 48) * (1 - ratio)}<line x1="44" y1={y} x2="740" y2={y} class="stroke-neutral-100 dark:stroke-neutral-800" /><text x="36" y={y + 4} text-anchor="end" class="fill-neutral-400 text-[10px]">{Math.round(trendMax * ratio)}</text>{/each}
					{#if trendCoords.length > 1}<polyline points={trendPolyline} fill="none" class="stroke-brand-500" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />{/if}
					{#each trendCoords as point, index}
						<g role="button" tabindex="0" class="cursor-pointer" onclick={() => selectTrendPeriod(point.period)} onkeydown={(event) => { if (event.key === "Enter" || event.key === " ") selectTrendPeriod(point.period); }}>
							<title>{periodLabel(point.period)}: {point.total}</title>
							<circle cx={point.x} cy={point.y} r="6" class="fill-white stroke-brand-500 dark:fill-neutral-950" stroke-width="3" />
							{#if index % Math.max(1, Math.ceil(trendCoords.length / 6)) === 0 || index === trendCoords.length - 1}<text x={point.x} y="240" text-anchor="middle" class="fill-neutral-500 text-[10px]">{periodLabel(point.period)}</text>{/if}
						</g>
					{/each}
				</svg>
			</div>
		{:else}<div class="flex min-h-48 items-center justify-center text-sm text-neutral-500">Belum ada distribusi pada periode ini.</div>{/if}
	</div>

	<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-6">
		<div class="mb-4"><h3 class="font-semibold text-neutral-900 dark:text-white">Komposisi Distribusi</h3><p class="mt-1 text-xs leading-5 text-neutral-500">Klik segmen untuk memfilter kategori dan melihat Mahasantri terkait.</p></div>
		<DonutChart items={dashboard?.category_composition || []} onselect={(value) => setFilter("category", value, "period")} />
	</div>
</section>

<section class="grid gap-4 lg:grid-cols-2">
	<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-6"><div class="mb-4 flex items-start justify-between gap-3"><div><h3 class="font-semibold text-neutral-900 dark:text-white">Produk Paling Banyak Didistribusikan</h3><p class="mt-1 text-xs leading-5 text-neutral-500">Klik bar untuk fokus ke produk dan melihat penerimanya.</p></div><BarChart3 size="17" class="shrink-0 text-brand-500" /></div><BarChart items={dashboard?.top_products || []} onselect={(value) => setFilter("product_id", Number(value), "period")} emptyText="Belum ada distribusi produk." /></div>
	<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-6"><div class="mb-4"><h3 class="font-semibold text-neutral-900 dark:text-white">Coverage per Angkatan</h3><p class="mt-1 text-xs leading-5 text-neutral-500">Klik angkatan untuk melihat Mahasantri yang memiliki produk sesuai filter.</p></div><BarChart items={dashboard?.coverage_by_angkatan || []} onselect={(value) => value && setFilter("angkatan", value, "has")} emptyText="Belum ada data angkatan." /></div>
</section>

<section class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
	<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-6">
		<div class="mb-4"><h3 class="font-semibold text-neutral-900 dark:text-white">Pergerakan Stok</h3><p class="mt-1 text-xs leading-5 text-neutral-500">Stok masuk dibanding stok keluar. Klik periode untuk melihat distribusi Mahasantri pada periode yang sama.</p></div>
		<div class="space-y-3">
			{#each dashboard?.stock_movement || [] as item}
				<button type="button" onclick={() => selectTrendPeriod(item.period)} class="grid w-full gap-2 rounded-xl p-2 text-left hover:bg-neutral-50 dark:hover:bg-white/[0.03] sm:grid-cols-[88px_1fr] sm:items-center">
					<p class="text-xs font-medium text-neutral-500">{periodLabel(item.period)}</p>
					<div class="space-y-1.5">
						<div class="flex items-center gap-2"><span class="w-11 text-[11px] text-neutral-500">Masuk</span><div class="h-2 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-green-500" style={`width:${Math.max(item.masuk / movementMax * 100, item.masuk ? 2 : 0)}%`}></div></div><span class="w-9 text-right font-mono text-xs">{fmt(item.masuk)}</span></div>
						<div class="flex items-center gap-2"><span class="w-11 text-[11px] text-neutral-500">Keluar</span><div class="h-2 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-red-500" style={`width:${Math.max(item.keluar / movementMax * 100, item.keluar ? 2 : 0)}%`}></div></div><span class="w-9 text-right font-mono text-xs">{fmt(item.keluar)}</span></div>
					</div>
				</button>
			{:else}<div class="py-10 text-center text-sm text-neutral-500">Belum ada pergerakan stok pada periode ini.</div>{/each}
		</div>
	</div>

	<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50 sm:p-6">
		<div class="mb-4 flex items-center justify-between gap-3"><div><h3 class="font-semibold text-neutral-900 dark:text-white">Inventory Health</h3><p class="mt-1 text-xs leading-5 text-neutral-500">Produk dengan stok terendah ditampilkan lebih dulu.</p></div><button type="button" onclick={openStock} class="shrink-0 text-xs font-semibold text-brand-600">Kelola stok</button></div>
		<div class="space-y-2.5">
			{#each dashboard?.inventory_health || [] as item}
				<button type="button" onclick={() => setFilter("product_id", item.product_id, "has")} class="w-full rounded-xl border border-neutral-200 p-3 text-left transition hover:border-brand-300 dark:border-white/[0.06]">
					<div class="flex items-center justify-between gap-3"><div class="min-w-0"><p class="truncate text-sm font-semibold text-neutral-900 dark:text-white">{item.product_name}</p><p class="mt-1 text-xs text-neutral-500">Minimum {fmt(item.minimum_stock)}</p></div><div class="text-right"><p class="font-mono text-lg font-bold">{fmt(item.stock)}</p><span class="rounded-full px-2 py-0.5 text-[11px] font-semibold {healthClass(item.status)}">{healthLabel(item.status)}</span></div></div>
				</button>
			{:else}<div class="py-10 text-center text-sm text-neutral-500">Belum ada produk yang mengelola stok.</div>{/each}
		</div>
	</div>
</section>

{#if selectedProduct || summary.low_stock_products > 0}
	<section class="rounded-2xl border border-amber-500/20 bg-amber-500/[0.06] p-4 sm:p-5">
		<div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
			<div class="flex items-start gap-3"><AlertCircle size="19" class="mt-0.5 shrink-0 text-amber-600" /><div><h3 class="font-semibold text-neutral-900 dark:text-white">Perlu Perhatian</h3><p class="mt-1 text-sm leading-6 text-neutral-600 dark:text-neutral-300">{#if summary.low_stock_products > 0}{summary.low_stock_products} produk sudah menyentuh minimum stok atau habis.{:else}Stok produk pada filter saat ini masih di atas minimum.{/if}</p>{#if selectedProduct}<p class="mt-1 text-xs text-neutral-500">Produk dipilih: {selectedProduct.nama}. Anda bisa membandingkan Mahasantri yang sudah dan belum memilikinya.</p>{/if}</div></div>
		{#if selectedProduct}<div class="grid grid-cols-2 gap-2 sm:flex"><button type="button" onclick={() => showDrilldown("has")} class="inline-flex items-center justify-center gap-1 rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs font-semibold dark:border-neutral-700 dark:bg-neutral-900">Sudah punya <ArrowRight size="13" /></button><button type="button" onclick={() => showDrilldown("missing")} class="inline-flex items-center justify-center gap-1 rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white">Belum punya <ArrowRight size="13" /></button></div>{/if}
		</div>
	</section>
{/if}

{#if drilldown || drilldownError}
	<section id="crm-drilldown" class="overflow-hidden rounded-2xl border border-neutral-200 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
		<div class="border-b border-neutral-200 p-4 dark:border-white/[0.05] sm:p-5">
			<div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
				<div>
					<div class="flex items-center gap-2"><PackageSearch size="18" class="text-brand-500" /><h2 class="font-semibold text-neutral-900 dark:text-white">Data Mahasantri</h2></div>
					<p class="mt-1 text-sm font-medium text-neutral-700 dark:text-neutral-300">{drillTitle()}</p>
					<p class="mt-1 text-xs text-neutral-500">{fmt(drilldown?.total || 0)} Mahasantri ditemukan. Pilih produk/program yang belum dimiliki untuk follow up WhatsApp.</p>
				</div>
				<div class="flex flex-wrap gap-2">
					{#if filters?.product_id && selectedProduct}<span class="rounded-full bg-brand-500/10 px-2.5 py-1.5 text-xs font-medium text-brand-700 dark:text-brand-300">{selectedProduct.nama}{activeBatchLabel() ? ` · ${activeBatchLabel()}` : ""}</span>{/if}
					{#if filters?.angkatan}<span class="rounded-full bg-neutral-100 px-2.5 py-1.5 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">Angkatan {filters.angkatan}</span>{/if}
					{#if drilldownPeriod}<span class="rounded-full bg-neutral-100 px-2.5 py-1.5 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{periodLabel(drilldownPeriod)}</span>{/if}
					<button type="button" onclick={clearDrilldown} class="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs font-medium text-neutral-600 dark:border-white/[0.08] dark:text-neutral-300"><X size="12" /> Sembunyikan</button>
				</div>
			</div>
		</div>

		{#if drilldownError}
			<div class="m-4 rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-400 sm:m-5">{drilldownError}</div>
		{/if}

		{#if drilldown}
			<div class="space-y-3 p-3 lg:hidden">
				{#each drilldown.data || [] as row}
					{@const offers = offersFor(row)}
					{@const offer = offerFor(row)}
					{@const owned = ownedLabel(row)}
					<article class="rounded-xl border border-neutral-200 p-4 dark:border-white/[0.06]">
						<div class="flex items-start justify-between gap-3"><div class="min-w-0"><a href={`/app/santri/${row.id}`} class="block truncate font-semibold text-brand-600 hover:underline">{row.nama}</a><p class="mt-1 font-mono text-[11px] text-neutral-500">{row.id_mahasantri || "-"}</p></div><span class="shrink-0 rounded-full bg-neutral-100 px-2 py-1 text-[11px] capitalize text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{(row.status || "-").replaceAll("_", " ")}</span></div>
						<div class="mt-3 grid grid-cols-2 gap-2 text-xs"><div class="rounded-lg bg-neutral-50 p-2.5 dark:bg-neutral-900/60"><p class="text-neutral-500">Angkatan</p><p class="mt-1 font-medium text-neutral-800 dark:text-neutral-200">{row.angkatan_kelas || row.angkatan || "-"}</p></div><div class="rounded-lg bg-neutral-50 p-2.5 dark:bg-neutral-900/60"><p class="text-neutral-500">Level</p><p class="mt-1 font-medium text-neutral-800 dark:text-neutral-200">{row.level || "-"}</p></div></div>
						<div class="mt-3"><p class="text-[11px] font-semibold uppercase tracking-wide text-neutral-500">Sudah dimiliki</p><div class="mt-1.5 flex flex-wrap gap-1.5">{#each owned.slice(0, 4) as label}<span class="rounded-full bg-green-500/10 px-2 py-1 text-[11px] text-green-700 dark:text-green-300">{label}</span>{:else}<span class="text-xs text-neutral-400">Belum ada produk</span>{/each}{#if owned.length > 4}<span class="rounded-full bg-neutral-100 px-2 py-1 text-[11px] text-neutral-500 dark:bg-neutral-800">+{owned.length - 4}</span>{/if}</div></div>
						<div class="mt-4 space-y-2">
							{#if offers.length}
								<select value={offer?.key || ""} onchange={(event) => setOffer(row.id, event.currentTarget.value)} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white">{#each offers as option}<option value={option.key}>{option.label}</option>{/each}</select>
								{#if row.no_wa && waURL(row, offer)}<a href={waURL(row, offer)} target="_blank" rel="noreferrer" class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-green-600 px-3 py-2.5 text-sm font-semibold text-white hover:bg-green-700"><MessageCircle size="16" /> Follow Up WhatsApp</a>{:else}<button type="button" disabled class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-neutral-200 px-3 py-2.5 text-sm font-semibold text-neutral-500 dark:bg-neutral-800"><MessageCircle size="16" /> Nomor WA belum tersedia</button>{/if}
							{:else}<div class="rounded-xl bg-green-500/10 px-3 py-2.5 text-center text-xs font-medium text-green-700 dark:text-green-300">Semua produk/program aktif pada filter ini sudah dimiliki.</div>{/if}
						</div>
					</article>
				{:else}<div class="py-10 text-center text-sm text-neutral-500">Tidak ada Mahasantri pada segmen ini.</div>{/each}
			</div>

			<div class="hidden overflow-x-auto lg:block">
				<table class="w-full min-w-[1120px] text-sm">
					<thead class="bg-neutral-50 text-left text-[11px] font-semibold uppercase tracking-wide text-neutral-500 dark:bg-neutral-900/60"><tr><th class="px-4 py-3">Mahasantri</th><th class="px-4 py-3">Kelas</th><th class="px-4 py-3">Sudah Dimiliki</th><th class="px-4 py-3">Tawarkan Produk / Program</th><th class="px-4 py-3 text-right">Follow Up</th></tr></thead>
					<tbody class="divide-y divide-neutral-200/70 dark:divide-white/[0.04]">
						{#each drilldown.data || [] as row}
							{@const offers = offersFor(row)}
							{@const offer = offerFor(row)}
							{@const owned = ownedLabel(row)}
							<tr class="align-top hover:bg-neutral-50/60 dark:hover:bg-white/[0.02]">
								<td class="px-4 py-4"><a href={`/app/santri/${row.id}`} class="font-semibold text-brand-600 hover:underline">{row.nama}</a><p class="mt-1 font-mono text-xs text-neutral-500">{row.id_mahasantri || "-"}</p><span class="mt-2 inline-block rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] capitalize text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">{(row.status || "-").replaceAll("_", " ")}</span></td>
								<td class="px-4 py-4 text-neutral-600 dark:text-neutral-300"><p>{row.angkatan_kelas || row.angkatan || "-"}</p><p class="mt-1 text-xs text-neutral-500">{row.level || "-"}</p></td>
								<td class="max-w-[320px] px-4 py-4"><div class="flex flex-wrap gap-1.5">{#each owned.slice(0, 5) as label}<span class="rounded-full bg-green-500/10 px-2 py-1 text-[11px] text-green-700 dark:text-green-300">{label}</span>{:else}<span class="text-xs text-neutral-400">Belum ada produk</span>{/each}{#if owned.length > 5}<span class="rounded-full bg-neutral-100 px-2 py-1 text-[11px] text-neutral-500 dark:bg-neutral-800">+{owned.length - 5}</span>{/if}</div></td>
								<td class="w-[300px] px-4 py-4">{#if offers.length}<select value={offer?.key || ""} onchange={(event) => setOffer(row.id, event.currentTarget.value)} class="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white">{#each offers as option}<option value={option.key}>{option.label}</option>{/each}</select>{:else}<span class="text-xs text-green-600">Semua produk aktif sudah dimiliki</span>{/if}</td>
								<td class="px-4 py-4 text-right">{#if offers.length && row.no_wa && waURL(row, offer)}<a href={waURL(row, offer)} target="_blank" rel="noreferrer" class="inline-flex items-center gap-1.5 rounded-lg bg-green-600 px-3 py-2 text-xs font-semibold text-white hover:bg-green-700"><MessageCircle size="14" /> Follow Up WA</a>{:else if offers.length}<span class="text-xs text-neutral-400">No. WA kosong</span>{else}<CheckCircle2 size="18" class="ml-auto text-green-500" />{/if}</td>
							</tr>
						{:else}<tr><td colspan="5" class="px-4 py-12 text-center text-sm text-neutral-500">Tidak ada Mahasantri pada segmen ini.</td></tr>{/each}
					</tbody>
				</table>
			</div>

			<div class="flex flex-col gap-3 border-t border-neutral-200 px-4 py-3 dark:border-white/[0.05] sm:flex-row sm:items-center sm:justify-between">
				<p class="text-xs text-neutral-500">Menampilkan {drillStart}-{drillEnd} dari {fmt(drilldown.total)} Mahasantri</p>
				<div class="flex items-center justify-between gap-2 sm:justify-end"><button type="button" onclick={() => goDrillPage((drilldown.page || 1) - 1)} disabled={(drilldown.page || 1) <= 1} class="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium disabled:opacity-40 dark:border-white/[0.08]"><ChevronLeft size="14" /> Sebelumnya</button><span class="px-2 text-xs text-neutral-500">{drilldown.page || 1} / {drillTotalPages}</span><button type="button" onclick={() => goDrillPage((drilldown.page || 1) + 1)} disabled={(drilldown.page || 1) >= drillTotalPages} class="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium disabled:opacity-40 dark:border-white/[0.08]">Berikutnya <ChevronRight size="14" /></button></div>
			</div>
		{/if}
	</section>
{/if}
