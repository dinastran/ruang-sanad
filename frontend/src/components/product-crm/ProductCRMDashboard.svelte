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
		RotateCcw,
		ShoppingBag,
		Users,
	} from "lucide-svelte";

	interface Batch { id: number; nama: string; is_aktif: boolean }
	interface Product {
		id: number; nama: string; kategori: string; track_stok: boolean; is_aktif: boolean;
		stok: number; minimum_stock: number; batches: Batch[];
	}
	interface Angkatan { id: number; kode: string; keterangan: string }
	interface Point { label: string; value: string; total: number }
	interface TrendPoint { period: string; label: string; total: number }
	interface StockMovement { period: string; label: string; masuk: number; keluar: number }
	interface InventoryHealth { product_id: number; product_name: string; stock: number; minimum_stock: number; status: string }
	interface DashboardFilters {
		date_from: string; date_to: string; product_id: number; batch_id: number;
		category: string; angkatan: string; status: string;
	}
	interface Dashboard {
		summary: {
			active_products: number; santri_total: number; santri_with_product: number; coverage_rate: number;
			active_ownerships: number; total_stock: number; low_stock_products: number;
			period_assignments: number; previous_assignments: number; assignment_change_rate: number;
		};
		distribution_trend: TrendPoint[];
		top_products: Point[];
		category_composition: Point[];
		coverage_by_angkatan: Point[];
		stock_movement: StockMovement[];
		inventory_health: InventoryHealth[];
	}
	interface Props {
		dashboard?: Dashboard;
		filters?: DashboardFilters;
		products?: Product[];
		angkatan?: Angkatan[];
		error?: string;
	}
	let { dashboard, filters, products = [], angkatan = [], error }: Props = $props();

	let selectedProduct = $derived(products.find((p) => p.id === Number(filters?.product_id || 0)));
	let summary = $derived(dashboard?.summary ?? {
		active_products: 0, santri_total: 0, santri_with_product: 0, coverage_rate: 0,
		active_ownerships: 0, total_stock: 0, low_stock_products: 0,
		period_assignments: 0, previous_assignments: 0, assignment_change_rate: 0,
	});
	let trend = $derived(dashboard?.distribution_trend ?? []);
	let trendMax = $derived(Math.max(...trend.map((x) => x.total), 1));
	let trendCoords = $derived(trend.map((point, index) => {
		const width = 900, height = 250, left = 44, right = 24, top = 24, bottom = 44;
		const usableW = width - left - right, usableH = height - top - bottom;
		return {
			...point,
			x: trend.length <= 1 ? width / 2 : left + (index / (trend.length - 1)) * usableW,
			y: top + usableH - (point.total / trendMax) * usableH,
		};
	}));
	let trendPolyline = $derived(trendCoords.map((x) => `${x.x},${x.y}`).join(" "));
	let movementMax = $derived(Math.max(...(dashboard?.stock_movement ?? []).flatMap((x) => [x.masuk, x.keluar]), 1));

	function params() {
		return new URLSearchParams(typeof window !== "undefined" ? window.location.search : "");
	}
	function navigate(p: URLSearchParams) {
		p.set("tab", "dashboard");
		router.get(`/app/produk-crm?${p.toString()}`, {}, { preserveScroll: true });
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
		router.get(`/app/produk-crm?${p.toString()}`, {}, { preserveScroll: true });
	}
	function setFilter(key: string, value: string | number) {
		const p = params();
		p.delete("page");
		if (value === "" || value === 0) p.delete(key); else p.set(key, String(value));
		if (key === "product_id") p.delete("dashboard_batch_id");
		navigate(p);
	}
	function reset() {
		router.get("/app/produk-crm?tab=dashboard");
	}
	function quickDays(days: number) {
		const to = new Date();
		const from = new Date();
		from.setDate(to.getDate() - days + 1);
		const local = (d: Date) => {
			const offset = d.getTimezoneOffset();
			return new Date(d.getTime() - offset * 60000).toISOString().slice(0, 10);
		};
		const p = params();
		p.set("date_from", local(from));
		p.set("date_to", local(to));
		navigate(p);
	}
	function quickMonth() {
		const now = new Date();
		const from = new Date(now.getFullYear(), now.getMonth(), 1);
		const offset = (d: Date) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
		const p = params();
		p.set("date_from", offset(from));
		p.set("date_to", offset(now));
		navigate(p);
	}
	function openCRM(kind: "has" | "missing", productID: number) {
		const p = new URLSearchParams();
		p.set("tab", "crm");
		if (kind === "has") p.set("has_product_id", String(productID)); else p.set("missing_product_id", String(productID));
		if (filters?.angkatan) p.set("angkatan", filters.angkatan);
		if (filters?.status) p.set("status", filters.status);
		router.get(`/app/produk-crm?${p.toString()}`);
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
</script>

{#if error}
	<div class="rounded-2xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-400">{error}</div>
{/if}

<section class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
	<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
		<div>
			<h2 class="font-semibold text-neutral-900 dark:text-white">Filter Intelligence</h2>
			<p class="mt-1 text-xs text-neutral-500">Periode memengaruhi distribusi dan pergerakan. Stok dan coverage menunjukkan kondisi saat ini.</p>
		</div>
		<div class="flex flex-wrap gap-2">
			<button type="button" onclick={() => quickDays(7)} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium dark:border-white/[0.08]">7 hari</button>
			<button type="button" onclick={() => quickDays(30)} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium dark:border-white/[0.08]">30 hari</button>
			<button type="button" onclick={quickMonth} class="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium dark:border-white/[0.08]">Bulan ini</button>
			<button type="button" onclick={reset} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium dark:border-white/[0.08]"><RotateCcw size="13" /> Reset</button>
		</div>
	</div>
	<form onsubmit={applyFilters} class="grid gap-3 md:grid-cols-2 xl:grid-cols-7">
		<label class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Dari tanggal<input name="date_from" type="date" value={filters?.date_from || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white" /></label>
		<label class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Sampai tanggal<input name="date_to" type="date" value={filters?.date_to || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white" /></label>
		<label class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Produk<select name="product_id" value={filters?.product_id || 0} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value={0}>Semua produk</option>{#each products.filter((p) => p.is_aktif) as p}<option value={p.id}>{p.nama}</option>{/each}</select></label>
		<label class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Batch / edisi<select name="dashboard_batch_id" value={filters?.batch_id || 0} disabled={!selectedProduct?.batches?.length} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm disabled:opacity-50 dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value={0}>Semua batch</option>{#each selectedProduct?.batches || [] as b}<option value={b.id}>{b.nama}</option>{/each}</select></label>
		<label class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Kategori<select name="category" value={filters?.category || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option><option value="buku">Buku</option><option value="program">Program</option></select></label>
		<label class="space-y-1.5 text-xs text-neutral-600 dark:text-neutral-400">Angkatan<select name="angkatan" value={filters?.angkatan || ""} class="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option>{#each angkatan as a}<option value={a.kode}>{a.keterangan || a.kode}</option>{/each}</select></label>
		<div class="space-y-1.5"><label class="text-xs text-neutral-600 dark:text-neutral-400">Status santri</label><div class="flex gap-2"><select name="status" value={filters?.status || ""} class="min-w-0 flex-1 rounded-lg border border-neutral-200 bg-white px-3 py-2.5 text-sm dark:border-white/[0.08] dark:bg-neutral-900 dark:text-white"><option value="">Semua</option><option value="aktif">Aktif</option><option value="cuti">Cuti</option><option value="nonaktif">Nonaktif</option><option value="tidak_lanjut">Tidak lanjut</option></select><button type="submit" class="rounded-lg bg-brand-600 px-3.5 py-2.5 text-sm font-semibold text-white">Terapkan</button></div></div>
	</form>
</section>

<section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
	<div class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="flex items-center justify-between"><p class="text-xs font-medium uppercase tracking-wide text-neutral-500">Produk aktif</p><ShoppingBag size="18" class="text-brand-500" /></div><p class="mt-3 text-3xl font-bold text-neutral-900 dark:text-white">{fmt(summary.active_products)}</p><p class="mt-1 text-xs text-neutral-500">sesuai kategori / produk terpilih</p></div>
	<button type="button" onclick={() => selectedProduct ? openCRM("has", selectedProduct.id) : undefined} class="rounded-2xl border border-neutral-200 bg-white p-5 text-left dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="flex items-center justify-between"><p class="text-xs font-medium uppercase tracking-wide text-neutral-500">Mahasantri punya produk</p><Users size="18" class="text-brand-500" /></div><div class="mt-3 flex items-end gap-2"><p class="text-3xl font-bold text-neutral-900 dark:text-white">{fmt(summary.santri_with_product)}</p><p class="pb-1 text-sm font-semibold text-brand-600">{pct(summary.coverage_rate)}</p></div><p class="mt-1 text-xs text-neutral-500">dari {fmt(summary.santri_total)} Mahasantri</p></button>
	<div class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="flex items-center justify-between"><p class="text-xs font-medium uppercase tracking-wide text-neutral-500">Kepemilikan aktif</p><CheckCircle2 size="18" class="text-brand-500" /></div><p class="mt-3 text-3xl font-bold text-neutral-900 dark:text-white">{fmt(summary.active_ownerships)}</p><p class="mt-1 text-xs text-neutral-500">produk / program aktif tercatat</p></div>
	<button type="button" onclick={openStock} class="rounded-2xl border border-neutral-200 bg-white p-5 text-left dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="flex items-center justify-between"><p class="text-xs font-medium uppercase tracking-wide text-neutral-500">Total stok buku</p><Boxes size="18" class="text-brand-500" /></div><p class="mt-3 text-3xl font-bold text-neutral-900 dark:text-white">{fmt(summary.total_stock)}</p><p class="mt-1 text-xs text-neutral-500">snapshot stok saat ini</p></button>
	<button type="button" onclick={openStock} class="rounded-2xl border border-neutral-200 bg-white p-5 text-left dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="flex items-center justify-between"><p class="text-xs font-medium uppercase tracking-wide text-neutral-500">Perlu restock</p><AlertCircle size="18" class={summary.low_stock_products > 0 ? "text-amber-500" : "text-green-500"} /></div><p class="mt-3 text-3xl font-bold text-neutral-900 dark:text-white">{fmt(summary.low_stock_products)}</p><p class="mt-1 text-xs text-neutral-500">stok ≤ minimum atau habis</p></button>
	<div class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="flex items-center justify-between"><p class="text-xs font-medium uppercase tracking-wide text-neutral-500">Distribusi periode</p><CalendarDays size="18" class="text-brand-500" /></div><div class="mt-3 flex items-end gap-2"><p class="text-3xl font-bold text-neutral-900 dark:text-white">{fmt(summary.period_assignments)}</p><span class="pb-1 text-xs font-semibold {summary.assignment_change_rate >= 0 ? 'text-green-600' : 'text-red-500'}">{summary.assignment_change_rate >= 0 ? "+" : ""}{pct(summary.assignment_change_rate)}</span></div><p class="mt-1 text-xs text-neutral-500">periode sebelumnya {fmt(summary.previous_assignments)}</p></div>
</section>

<section class="grid gap-5 xl:grid-cols-[1.45fr_0.85fr]">
	<div class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
		<div class="mb-5"><h3 class="font-semibold text-neutral-900 dark:text-white">Trend Distribusi Produk</h3><p class="mt-1 text-xs text-neutral-500">Jumlah pencatatan produk / program pada periode terpilih.</p></div>
		{#if trend.length}
			<div class="overflow-x-auto">
				<svg viewBox="0 0 900 250" class="min-w-[680px] w-full" role="img" aria-label="Trend distribusi produk">
					{#each [0, 0.5, 1] as ratio}{@const y = 24 + (250 - 24 - 44) * (1 - ratio)}<line x1="44" y1={y} x2="876" y2={y} class="stroke-neutral-100 dark:stroke-neutral-800" /><text x="36" y={y + 4} text-anchor="end" class="fill-neutral-400 text-[11px]">{Math.round(trendMax * ratio)}</text>{/each}
					{#if trendCoords.length > 1}<polyline points={trendPolyline} fill="none" class="stroke-brand-500" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />{/if}
					{#each trendCoords as point, index}<g><title>{periodLabel(point.period)}: {point.total}</title><circle cx={point.x} cy={point.y} r="5" class="fill-white stroke-brand-500 dark:fill-neutral-950" stroke-width="3" />{#if index % Math.max(1, Math.ceil(trendCoords.length / 7)) === 0 || index === trendCoords.length - 1}<text x={point.x} y="232" text-anchor="middle" class="fill-neutral-500 text-[11px]">{periodLabel(point.period)}</text>{/if}</g>{/each}
				</svg>
			</div>
		{:else}<div class="flex min-h-56 items-center justify-center text-sm text-neutral-500">Belum ada distribusi pada periode ini.</div>{/if}
	</div>
	<div class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
		<h3 class="font-semibold text-neutral-900 dark:text-white">Komposisi Distribusi</h3><p class="mb-5 mt-1 text-xs text-neutral-500">Buku dibanding program pada periode terpilih.</p>
		<DonutChart items={dashboard?.category_composition || []} onselect={(value) => setFilter("category", value)} />
	</div>
</section>

<section class="grid gap-5 lg:grid-cols-2">
	<div class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="mb-5 flex items-start justify-between"><div><h3 class="font-semibold text-neutral-900 dark:text-white">Produk Paling Banyak Didistribusikan</h3><p class="mt-1 text-xs text-neutral-500">Klik bar untuk fokus ke satu produk.</p></div><BarChart3 size="18" class="text-brand-500" /></div><BarChart items={dashboard?.top_products || []} onselect={(value) => setFilter("product_id", Number(value))} emptyText="Belum ada distribusi produk." /></div>
	<div class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="mb-5"><h3 class="font-semibold text-neutral-900 dark:text-white">Coverage per Angkatan</h3><p class="mt-1 text-xs text-neutral-500">Persentase Mahasantri yang memiliki produk sesuai filter. Angka bar adalah persen.</p></div><BarChart items={dashboard?.coverage_by_angkatan || []} onselect={(value) => value && setFilter("angkatan", value)} emptyText="Belum ada data angkatan." /></div>
</section>

<section class="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
	<div class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
		<div class="mb-5"><h3 class="font-semibold text-neutral-900 dark:text-white">Pergerakan Stok</h3><p class="mt-1 text-xs text-neutral-500">Stok masuk dibanding stok keluar pada periode terpilih.</p></div>
		<div class="space-y-4">
			{#each dashboard?.stock_movement || [] as item}
				<div class="grid gap-2 sm:grid-cols-[90px_1fr] sm:items-center">
					<p class="text-xs font-medium text-neutral-500">{periodLabel(item.period)}</p>
					<div class="space-y-1.5">
						<div class="flex items-center gap-2"><span class="w-12 text-[11px] text-neutral-500">Masuk</span><div class="h-2 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-green-500" style={`width:${Math.max(item.masuk / movementMax * 100, item.masuk ? 2 : 0)}%`}></div></div><span class="w-10 text-right font-mono text-xs">{fmt(item.masuk)}</span></div>
						<div class="flex items-center gap-2"><span class="w-12 text-[11px] text-neutral-500">Keluar</span><div class="h-2 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-red-500" style={`width:${Math.max(item.keluar / movementMax * 100, item.keluar ? 2 : 0)}%`}></div></div><span class="w-10 text-right font-mono text-xs">{fmt(item.keluar)}</span></div>
					</div>
				</div>
			{:else}<div class="py-10 text-center text-sm text-neutral-500">Belum ada pergerakan stok pada periode ini.</div>{/each}
		</div>
	</div>

	<div class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50">
		<div class="mb-5 flex items-center justify-between"><div><h3 class="font-semibold text-neutral-900 dark:text-white">Inventory Health</h3><p class="mt-1 text-xs text-neutral-500">Produk dengan stok terendah ditampilkan lebih dulu.</p></div><button type="button" onclick={openStock} class="text-xs font-semibold text-brand-600">Kelola stok</button></div>
		<div class="space-y-3">
			{#each dashboard?.inventory_health || [] as item}
				<button type="button" onclick={() => setFilter("product_id", item.product_id)} class="w-full rounded-xl border border-neutral-200 p-3 text-left hover:border-brand-300 dark:border-white/[0.06]">
					<div class="flex items-center justify-between gap-3"><div class="min-w-0"><p class="truncate text-sm font-semibold text-neutral-900 dark:text-white">{item.product_name}</p><p class="mt-1 text-xs text-neutral-500">Minimum {fmt(item.minimum_stock)}</p></div><div class="text-right"><p class="font-mono text-lg font-bold">{fmt(item.stock)}</p><span class="rounded-full px-2 py-0.5 text-[11px] font-semibold {healthClass(item.status)}">{healthLabel(item.status)}</span></div></div>
				</button>
			{:else}<div class="py-10 text-center text-sm text-neutral-500">Belum ada produk yang mengelola stok.</div>{/each}
		</div>
	</div>
</section>

{#if summary.low_stock_products > 0 || selectedProduct}
	<section class="rounded-2xl border border-amber-500/20 bg-amber-500/[0.06] p-5">
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div class="flex items-start gap-3"><AlertCircle size="20" class="mt-0.5 shrink-0 text-amber-600" /><div><h3 class="font-semibold text-neutral-900 dark:text-white">Perlu Perhatian</h3><p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">{#if summary.low_stock_products > 0}{summary.low_stock_products} produk sudah menyentuh minimum stok atau habis.{:else}Stok produk pada filter saat ini masih di atas minimum.{/if}</p>{#if selectedProduct}<p class="mt-1 text-xs text-neutral-500">Produk aktif dipilih: {selectedProduct.nama}. Gunakan drill-down untuk melihat Mahasantri yang sudah atau belum memilikinya.</p>{/if}</div></div>
		<div class="flex flex-wrap gap-2">{#if selectedProduct}<button type="button" onclick={() => openCRM("has", selectedProduct.id)} class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs font-semibold dark:border-neutral-700 dark:bg-neutral-900">Sudah punya <ArrowRight size="13" /></button><button type="button" onclick={() => openCRM("missing", selectedProduct.id)} class="inline-flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white">Belum punya <ArrowRight size="13" /></button>{/if}</div>
		</div>
	</section>
{/if}
